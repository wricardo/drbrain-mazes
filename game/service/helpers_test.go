package service_test

import (
	"errors"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/wricardo/drbrain-mazes/game/engine"
	"github.com/wricardo/drbrain-mazes/game/service"
	"github.com/wricardo/drbrain-mazes/game/session"
)

// ---- maps ------------------------------------------------------------------

var (
	// corridorCfg: one level, S at (0,1), F at (4,1): 4 RIGHT moves win; UP/DOWN
	// from the start are blocked by void.
	corridorCfg = engine.MapConfig{
		ID: "corridor", Name: "Corridor", Levels: [][]string{{".....", "SoooF", "....."}}, MaxMoves: 10,
	}
	// limitCfg is corridor with a move limit (3) shorter than the route (4).
	limitCfg = engine.MapConfig{
		ID: "limit", Name: "Limit", Levels: [][]string{{".....", "SoooF", "....."}}, MaxMoves: 3,
	}
	// bypassCfg has a fast route (DOWN DOWN RIGHT RIGHT = 4 moves) and a slow
	// one (RIGHT RIGHT RIGHT DOWN DOWN LEFT = 6 moves, exactly max_moves).
	bypassCfg = engine.MapConfig{
		ID: "bypass", Name: "Bypass", Levels: [][]string{{"Sooo", "o..o", "ooFo"}}, MaxMoves: 6,
	}
	// towerCfg: two levels with paired stairs and a portal pair across levels.
	// RIGHT, RIGHT (stairs up, landing on the D at level 1 (2,0)), RIGHT (portal
	// 1 -> level 0 (3,2)), UP (stairs up, landing on the D at level 1 (3,1)),
	// DOWN steps onto F: par 5.
	towerCfg = engine.MapConfig{
		ID: "tower", Name: "Tower", MaxMoves: 50,
		Levels: [][]string{
			{"SoU.", "...U", "...1"},
			{"..D1", "...D", "...F"},
		},
	}
	// longCfg: 20x3, 19 RIGHT moves win (more updates than a subscriber buffer holds).
	longCfg = engine.MapConfig{
		ID: "long", Name: "Long", MaxMoves: 100,
		Levels: [][]string{{strings.Repeat(".", 20), "S" + strings.Repeat("o", 18) + "F", strings.Repeat(".", 20)}},
	}
)

var (
	right = engine.DirRight
	up    = engine.DirUp
	down  = engine.DirDown
	left  = engine.DirLeft
)

func mustMap(t testing.TB, cfg engine.MapConfig) *engine.Map {
	t.Helper()
	m, err := engine.NewMap(cfg)
	if err != nil {
		t.Fatalf("NewMap(%s): %v", cfg.ID, err)
	}
	return m
}

// fakeMaps is an in-memory MapStore built from engine.NewMap.
type fakeMaps struct {
	mu sync.Mutex
	m  map[string]*engine.Map
}

func newFakeMaps(t testing.TB, cfgs ...engine.MapConfig) *fakeMaps {
	fm := &fakeMaps{m: make(map[string]*engine.Map)}
	for _, c := range cfgs {
		fm.m[c.ID] = mustMap(t, c)
	}
	return fm
}

func (f *fakeMaps) List() []*engine.Map {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]*engine.Map, 0, len(f.m))
	for _, m := range f.m {
		out = append(out, m)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

func (f *fakeMaps) Get(id string) (*engine.Map, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	m, ok := f.m[id]
	if !ok {
		return nil, service.Errorf(service.CodeNotFound, "map %q not found", id)
	}
	return m, nil
}

func (f *fakeMaps) Save(cfg engine.MapConfig) (*engine.Map, error) {
	m, err := engine.NewMap(cfg)
	if err != nil {
		return nil, service.Errorf(service.CodeInvalidArgument, "invalid map: %v", err)
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	f.m[m.ID] = m
	return m, nil
}

func (f *fakeMaps) Create(cfg engine.MapConfig) (*engine.Map, error) {
	m, err := engine.NewMap(cfg)
	if err != nil {
		return nil, service.Errorf(service.CodeInvalidArgument, "invalid map: %v", err)
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, ok := f.m[m.ID]; ok {
		return nil, service.Errorf(service.CodeInvalidArgument, "map %q already exists", m.ID)
	}
	f.m[m.ID] = m
	return m, nil
}

func (f *fakeMaps) Delete(id string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, ok := f.m[id]; !ok {
		return service.Errorf(service.CodeNotFound, "map %q not found", id)
	}
	delete(f.m, id)
	return nil
}

// ---- persistence spy -------------------------------------------------------

// spyPersistence is an in-memory SessionPersistence that counts saves.
type spyPersistence struct {
	mu    sync.Mutex
	saved map[string]*service.Session
	saves map[string]int
}

func newSpyPersistence() *spyPersistence {
	return &spyPersistence{saved: make(map[string]*service.Session), saves: make(map[string]int)}
}

func (p *spyPersistence) Save(s *service.Session) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.saved[s.ID] = s
	p.saves[s.ID]++
	return nil
}

func (p *spyPersistence) Load(id string) (*service.Session, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	s, ok := p.saved[id]
	if !ok {
		return nil, session.ErrNotFound
	}
	return s.Clone(), nil
}

func (p *spyPersistence) Delete(id string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if _, ok := p.saved[id]; !ok {
		return session.ErrNotFound
	}
	delete(p.saved, id)
	return nil
}

func (p *spyPersistence) ListAll() ([]string, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	ids := make([]string, 0, len(p.saved))
	for id := range p.saved {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids, nil
}

func (p *spyPersistence) saveCount(id string) int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.saves[id]
}

// latest returns the most recently saved snapshot for id (nil if none).
func (p *spyPersistence) latest(id string) *service.Session {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.saved[id]
}

// ---- broadcaster recorder --------------------------------------------------

type recorder struct {
	mu    sync.Mutex
	snaps []*service.Session
}

func (r *recorder) Broadcast(s *service.Session) {
	r.mu.Lock()
	r.snaps = append(r.snaps, s)
	r.mu.Unlock()
}

func (r *recorder) forSession(id string) []*service.Session {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []*service.Session
	for _, s := range r.snaps {
		if s.ID == id {
			out = append(out, s)
		}
	}
	return out
}

func (r *recorder) count() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.snaps)
}

// ---- environment -----------------------------------------------------------

type env struct {
	svc  service.GameService
	mgr  *session.Manager
	pers *spyPersistence
	maps *fakeMaps
	rec  *recorder
}

type envOpts struct {
	maxSessions int
	svcOpts     service.Options
	bc          service.Broadcaster // nil = recorder
}

// newEnv wires a service over fake maps, a spy persistence and a recorder.
// The service is shut down when the test ends.
func newEnv(t *testing.T, o envOpts, cfgs ...engine.MapConfig) *env {
	t.Helper()
	if len(cfgs) == 0 {
		cfgs = []engine.MapConfig{corridorCfg, limitCfg, bypassCfg, towerCfg, longCfg}
	}
	e := &env{maps: newFakeMaps(t, cfgs...), pers: newSpyPersistence(), rec: &recorder{}}
	e.mgr = session.NewManager(e.pers, o.maxSessions)
	var bc service.Broadcaster = e.rec
	if o.bc != nil {
		bc = o.bc
	}
	e.svc = service.New(e.maps, e.mgr, bc, o.svcOpts)
	t.Cleanup(e.svc.Shutdown)
	return e
}

// ---- helpers ---------------------------------------------------------------

func wantCode(t testing.TB, err error, code string) {
	t.Helper()
	var se *service.Error
	if !errors.As(err, &se) {
		t.Fatalf("error = %v (%T), want *service.Error with code %s", err, err, code)
	}
	if se.Code != code {
		t.Fatalf("error code = %s (%v), want %s", se.Code, err, code)
	}
}

func waitFor(t testing.TB, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(15 * time.Second)
	for !cond() {
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for %s", what)
		}
		time.Sleep(2 * time.Millisecond)
	}
}

func (e *env) get(t testing.TB, id string) *service.Session {
	t.Helper()
	s, err := e.svc.GetSession(id)
	if err != nil {
		t.Fatalf("GetSession(%s): %v", id, err)
	}
	return s
}

func (e *env) create(t testing.TB, mapID string) *service.Session {
	t.Helper()
	s, err := e.svc.CreateSession(mapID, "")
	if err != nil {
		t.Fatalf("CreateSession(%s): %v", mapID, err)
	}
	return s
}

func (e *env) move(t testing.TB, id string, d engine.Direction) *service.Session {
	t.Helper()
	s, err := e.svc.Move(id, d)
	if err != nil {
		t.Fatalf("Move(%s): %v", d, err)
	}
	return s
}

func (e *env) bulk(t testing.TB, id string, dirs ...engine.Direction) *service.Session {
	t.Helper()
	s, err := e.svc.BulkMove(id, dirs, false)
	if err != nil {
		t.Fatalf("BulkMove: %v", err)
	}
	return s
}

func pos(level, x, y int) engine.Position { return engine.Position{Level: level, X: x, Y: y} }

// repeat returns n copies of d.
func repeat(d engine.Direction, n int) []engine.Direction {
	out := make([]engine.Direction, n)
	for i := range out {
		out[i] = d
	}
	return out
}
