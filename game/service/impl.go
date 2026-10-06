package service

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/wricardo/drbrain-mazes/game/engine"
)

const (
	// DefaultMaxDisplayName is the display-name limit in characters.
	DefaultMaxDisplayName = 40

	// DefaultListLimit and MaxListLimit bound ListSessions.
	DefaultListLimit = 50
	MaxListLimit     = 500

	// MaxDirections bounds the directions accepted by one BulkMove
	// call. Blocked attempts are free, so without a cap a request
	// could ask for an unbounded amount of work and events.
	MaxDirections = 20000
)

// Options tunes New. The zero value selects the defaults.
type Options struct {
	// SessionTTL: on New, sessions whose LastActionAt is older than this are
	// deleted. 0 (default) disables sweeping.
	SessionTTL time.Duration
	// MaxDisplayName is the maximum display-name length in characters
	// (default DefaultMaxDisplayName).
	MaxDisplayName int
}

// gameService implements GameService.
//
// Concurrency model: every Session is guarded by its own lock. The mutate
// pattern is lock -> mutate -> touch (Seq++) -> Clone -> unlock -> persist ->
// broadcast: nothing persists or broadcasts while holding a session lock.
type gameService struct {
	maps     MapStore
	sessions SessionStore
	bc       Broadcaster // may be nil
	opts     Options
}

// New creates the game service. bc may be nil (no subscribers). If
// opts.SessionTTL > 0 expired sessions already present in sessions (e.g. just
// loaded by session.Manager.LoadAll) are swept before New returns.
func New(maps MapStore, sessions SessionStore, bc Broadcaster, opts Options) GameService {
	if opts.MaxDisplayName <= 0 {
		opts.MaxDisplayName = DefaultMaxDisplayName
	}
	svc := &gameService{maps: maps, sessions: sessions, bc: bc, opts: opts}
	if opts.SessionTTL > 0 {
		svc.sweepExpired(time.Now().Add(-opts.SessionTTL))
	}
	return svc
}

// sweepExpired deletes sessions idle since before cutoff.
func (svc *gameService) sweepExpired(cutoff time.Time) {
	for _, s := range svc.sessions.List() {
		s.Lock()
		expired := s.LastActionAt.Before(cutoff)
		id := s.ID
		s.Unlock()
		if !expired {
			continue
		}
		if err := svc.DeleteSession(id); err != nil {
			log.Printf("service: sweep expired session %s: %v", id, err)
		}
	}
}

// ---- helpers ---------------------------------------------------------------

func notFound(id string) *Error { return Errorf(CodeNotFound, "session %q not found", id) }

func (svc *gameService) get(id string) (*Session, error) {
	s, ok := svc.sessions.Get(id)
	if !ok {
		return nil, notFound(id)
	}
	return s, nil
}

// persist writes s durably; failures are logged, the in-memory state stays
// authoritative. s must not be locked by the caller.
func (svc *gameService) persist(s *Session) {
	if err := svc.sessions.Persist(s); err != nil {
		log.Printf("service: persist session %s: %v", s.ID, err)
	}
}

func (svc *gameService) broadcast(snap *Session) {
	if svc.bc != nil {
		svc.bc.Broadcast(snap)
	}
}

// touch records a state change: Seq++ and LastActionAt. Lock must be held.
func touch(s *Session) {
	s.Seq++
	s.LastActionAt = time.Now()
}

// recordOutcome counts a finished game. Lock must be held and the state
// terminal.
func recordOutcome(s *Session) {
	s.Attempts++
	if s.State.Status == engine.StatusWon && (s.BestMoves == nil || s.State.Moves < *s.BestMoves) {
		b := s.State.Moves
		s.BestMoves = &b
	}
}

// applyMove plays one move on s. A blocked attempt leaves the game state
// untouched but is still recorded in LastEvent. It reports the event and
// whether the game just ended. Lock must be held.
func applyMove(s *Session, d engine.Direction) (ev engine.MoveEvent, ended bool, err error) {
	ev, err = s.State.Move(s.Map, d)
	if err != nil {
		return ev, false, Errorf(CodeSessionTerminal, "game is already %s; reset the session first", s.State.Status)
	}
	s.LastEvent = &ev
	if s.State.Status.Terminal() {
		recordOutcome(s)
		return ev, true, nil
	}
	return ev, false, nil
}

func (svc *gameService) cleanName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if n := utf8.RuneCountInString(name); n > svc.opts.MaxDisplayName {
		return "", Errorf(CodeInvalidArgument, "display name too long (%d characters, max %d)", n, svc.opts.MaxDisplayName)
	}
	return name, nil
}

func newSessionID() (string, error) {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", fmt.Errorf("generate session id: %w", err)
	}
	return hex.EncodeToString(b[:]), nil
}

// ---- sessions --------------------------------------------------------------

// CreateSession starts a session on a private snapshot of map mapID.
func (svc *gameService) CreateSession(mapID, displayName string) (*Session, error) {
	name, err := svc.cleanName(displayName)
	if err != nil {
		return nil, err
	}
	m, err := svc.maps.Get(mapID)
	if err != nil {
		return nil, err
	}
	id, err := newSessionID()
	if err != nil {
		return nil, err
	}
	snapMap := m.Clone()
	now := time.Now()
	s := &Session{
		ID: id, DisplayName: name, MapID: snapMap.ID, Map: snapMap,
		State: engine.NewState(snapMap),
		Seq:   1, CreatedAt: now, LastActionAt: now,
	}
	if err := svc.sessions.Put(s); err != nil {
		return nil, err
	}
	svc.persist(s)
	s.Lock()
	snap := s.Clone()
	s.Unlock()
	svc.broadcast(snap)
	return snap, nil
}

// GetSession returns a snapshot of the session.
func (svc *gameService) GetSession(id string) (*Session, error) {
	s, err := svc.get(id)
	if err != nil {
		return nil, err
	}
	s.Lock()
	defer s.Unlock()
	return s.Clone(), nil
}

// ListSessions returns snapshots of the sessions matching f.
func (svc *gameService) ListSessions(f SessionFilter) ([]*Session, error) {
	limit := f.Limit
	switch {
	case limit < 0:
		return nil, Errorf(CodeInvalidArgument, "limit must be >= 0")
	case limit == 0:
		limit = DefaultListLimit
	case limit > MaxListLimit:
		limit = MaxListLimit
	}
	byCreated := false
	switch strings.ToLower(f.Sort) {
	case "", "recent":
	case "created":
		byCreated = true
	default:
		return nil, Errorf(CodeInvalidArgument, "unknown sort %q (want recent or created)", f.Sort)
	}

	type entry struct {
		s   *Session
		id  string
		key time.Time
	}
	var entries []entry
	for _, s := range svc.sessions.List() {
		s.Lock()
		if f.MapID == "" || s.MapID == f.MapID {
			key := s.LastActionAt
			if byCreated {
				key = s.CreatedAt
			}
			entries = append(entries, entry{s: s, id: s.ID, key: key})
		}
		s.Unlock()
	}
	sort.Slice(entries, func(i, j int) bool {
		if !entries[i].key.Equal(entries[j].key) {
			return entries[i].key.After(entries[j].key)
		}
		return entries[i].id < entries[j].id
	})
	if len(entries) > limit {
		entries = entries[:limit]
	}
	out := make([]*Session, len(entries))
	for i, e := range entries {
		e.s.Lock()
		out[i] = e.s.Clone()
		e.s.Unlock()
	}
	return out, nil
}

// DeleteSession removes the session from memory and storage.
func (svc *gameService) DeleteSession(id string) error {
	return svc.sessions.Delete(id)
}

// RenameSession sets the display name.
func (svc *gameService) RenameSession(id, name string) (*Session, error) {
	name, err := svc.cleanName(name)
	if err != nil {
		return nil, err
	}
	s, err := svc.get(id)
	if err != nil {
		return nil, err
	}
	s.Lock()
	s.DisplayName = name
	touch(s)
	snap := s.Clone()
	s.Unlock()
	svc.persist(s)
	svc.broadcast(snap)
	return snap, nil
}

// Move plays one move. A blocked attempt (void or edge) is not an error and
// is not counted, but it still bumps Seq, sets LastEvent (Blocked) and is
// broadcast, so subscribers can show a bump animation: the game state is
// unchanged, the event is visible. Only state-changing moves are persisted.
func (svc *gameService) Move(id string, d engine.Direction) (*Session, error) {
	if !validDirection(d) {
		return nil, Errorf(CodeInvalidArgument, "invalid direction %d", int(d))
	}
	s, err := svc.get(id)
	if err != nil {
		return nil, err
	}
	s.Lock()
	ev, _, err := applyMove(s, d)
	if err != nil {
		s.Unlock()
		return nil, err
	}
	touch(s)
	snap := s.Clone()
	s.Unlock()
	if !ev.Blocked {
		svc.persist(s)
	}
	svc.broadcast(snap)
	return snap, nil
}

// BulkMove plays dirs in order under one lock and one broadcast. It stops as
// soon as the game ends and, when stopOnBlocked is set, at the first blocked
// attempt (a wrong plan then cannot run on from the wrong tile); otherwise
// blocked attempts are processed and ignored. LastEvent is the last attempt
// made. An empty list changes nothing.
func (svc *gameService) BulkMove(id string, dirs []engine.Direction, stopOnBlocked bool) (*Session, error) {
	if len(dirs) > MaxDirections {
		return nil, Errorf(CodeInvalidArgument, "too many directions (%d, max %d)", len(dirs), MaxDirections)
	}
	for _, d := range dirs {
		if !validDirection(d) {
			return nil, Errorf(CodeInvalidArgument, "invalid direction %d", int(d))
		}
	}
	s, err := svc.get(id)
	if err != nil {
		return nil, err
	}
	s.Lock()
	if s.State.Status.Terminal() {
		status := s.State.Status
		s.Unlock()
		return nil, Errorf(CodeSessionTerminal, "game is already %s; reset the session first", status)
	}
	if len(dirs) == 0 {
		snap := s.Clone()
		s.Unlock()
		return snap, nil
	}
	moved := false
	for _, d := range dirs {
		ev, ended, err := applyMove(s, d)
		if err != nil { // unreachable: terminal was checked and we stop at the end
			s.Unlock()
			return nil, err
		}
		moved = moved || !ev.Blocked
		if ended || (ev.Blocked && stopOnBlocked) {
			break
		}
	}
	touch(s)
	snap := s.Clone()
	s.Unlock()
	if moved {
		svc.persist(s)
	}
	svc.broadcast(snap)
	return snap, nil
}

func validDirection(d engine.Direction) bool { return d >= engine.DirUp && d <= engine.DirLeft }

// Reset puts the game back at the start, keeping Attempts and BestMoves.
func (svc *gameService) Reset(id string) (*Session, error) {
	s, err := svc.get(id)
	if err != nil {
		return nil, err
	}
	s.Lock()
	s.State = engine.NewState(s.Map)
	s.LastEvent = nil
	touch(s)
	snap := s.Clone()
	s.Unlock()
	svc.persist(s)
	svc.broadcast(snap)
	return snap, nil
}

// ---- maps ------------------------------------------------------------------

// ListMaps lists all maps.
func (svc *gameService) ListMaps() []*engine.Map { return svc.maps.List() }

// GetMap returns one map.
func (svc *gameService) GetMap(id string) (*engine.Map, error) { return svc.maps.Get(id) }

// SaveMap validates and stores a map. Existing sessions keep their snapshot.
func (svc *gameService) SaveMap(cfg engine.MapConfig) (*engine.Map, error) {
	return svc.maps.Save(cfg)
}

// CreateMap validates and stores a new map; it fails if the id already exists.
func (svc *gameService) CreateMap(cfg engine.MapConfig) (*engine.Map, error) {
	return svc.maps.Create(cfg)
}

// DeleteMap removes a map. Existing sessions keep their snapshot.
func (svc *gameService) DeleteMap(id string) error { return svc.maps.Delete(id) }

// ---- shutdown --------------------------------------------------------------

// Shutdown persists every session. It is idempotent.
func (svc *gameService) Shutdown() {
	for _, s := range svc.sessions.List() {
		svc.persist(s)
	}
}
