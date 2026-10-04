package service_test

import (
	"reflect"
	"regexp"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/wricardo/drbrain-mazes/game/engine"
	"github.com/wricardo/drbrain-mazes/game/service"
	"github.com/wricardo/drbrain-mazes/game/session"
)

func TestCreateSession(t *testing.T) {
	e := newEnv(t, envOpts{})

	s, err := e.svc.CreateSession("corridor", "  Ada  ")
	if err != nil {
		t.Fatal(err)
	}
	if !regexp.MustCompile(`^[0-9a-f]{16}$`).MatchString(s.ID) {
		t.Fatalf("id %q is not 8 random bytes hex", s.ID)
	}
	if s.DisplayName != "Ada" || s.MapID != "corridor" {
		t.Fatalf("name/map = %q/%q", s.DisplayName, s.MapID)
	}
	if s.Seq != 1 || s.Attempts != 0 || s.BestMoves != nil || s.LastEvent != nil {
		t.Fatalf("unexpected initial session: %+v", s)
	}
	if s.State.Status != engine.StatusReady || s.State.Moves != 0 || s.State.Pos != s.Map.Start {
		t.Fatalf("state = %+v, want fresh READY at the start", s.State)
	}
	if !reflect.DeepEqual(s.State.Path, []engine.Position{s.Map.Start}) {
		t.Fatalf("path = %v, want just the start", s.State.Path)
	}
	if e.pers.saveCount(s.ID) != 1 {
		t.Fatalf("saves = %d, want 1 on create", e.pers.saveCount(s.ID))
	}

	// A second session gets a different id.
	if s2 := e.create(t, "corridor"); s2.ID == s.ID {
		t.Fatal("duplicate session id")
	}

	_, err = e.svc.CreateSession("nope", "")
	wantCode(t, err, service.CodeNotFound)
	_, err = e.svc.CreateSession("corridor", strings.Repeat("x", 41))
	wantCode(t, err, service.CodeInvalidArgument)
	if _, err = e.svc.CreateSession("corridor", strings.Repeat("é", 40)); err != nil {
		t.Fatalf("40 multibyte characters must fit: %v", err)
	}
}

func TestCustomDisplayNameLimit(t *testing.T) {
	e := newEnv(t, envOpts{svcOpts: service.Options{MaxDisplayName: 5}})
	if _, err := e.svc.CreateSession("corridor", "abcdef"); err == nil {
		t.Fatal("6 characters accepted with limit 5")
	} else {
		wantCode(t, err, service.CodeInvalidArgument)
	}
	if _, err := e.svc.CreateSession("corridor", " abcde "); err != nil {
		t.Fatalf("name is limited after trimming: %v", err)
	}
}

func TestRenameSession(t *testing.T) {
	e := newEnv(t, envOpts{})
	s := e.create(t, "corridor")

	r, err := e.svc.RenameSession(s.ID, "  Grace ")
	if err != nil {
		t.Fatal(err)
	}
	if r.DisplayName != "Grace" || r.Seq != s.Seq+1 {
		t.Fatalf("rename: name %q seq %d (was %d)", r.DisplayName, r.Seq, s.Seq)
	}
	if e.pers.saveCount(s.ID) != 2 {
		t.Fatalf("saves = %d, want create+rename", e.pers.saveCount(s.ID))
	}
	last := e.rec.forSession(s.ID)
	if got := last[len(last)-1]; got.DisplayName != "Grace" {
		t.Fatalf("rename not broadcast: %q", got.DisplayName)
	}
	_, err = e.svc.RenameSession(s.ID, strings.Repeat("x", 41))
	wantCode(t, err, service.CodeInvalidArgument)
	_, err = e.svc.RenameSession("0000000000000000", "x")
	wantCode(t, err, service.CodeNotFound)
	if got := e.get(t, s.ID); got.DisplayName != "Grace" {
		t.Fatalf("failed rename changed name to %q", got.DisplayName)
	}
}

func TestSnapshotsAreIsolatedFromLiveSession(t *testing.T) {
	e := newEnv(t, envOpts{})
	s := e.create(t, "corridor")
	e.move(t, s.ID, right)
	e.bulk(t, s.ID, repeat(right, 3)...) // WON: BestMoves is set

	snap := e.get(t, s.ID)
	snap.State.Path[0] = engine.Position{X: 99, Y: 99}
	*snap.BestMoves = 99
	snap.LastEvent.Move = 99

	again := e.get(t, s.ID)
	if again.State.Path[0] == (engine.Position{X: 99, Y: 99}) || *again.BestMoves == 99 || again.LastEvent.Move == 99 {
		t.Fatal("mutating a returned snapshot changed the live session")
	}
}

func TestMapSnapshotIsolation(t *testing.T) {
	e := newEnv(t, envOpts{})
	s := e.create(t, "corridor")
	storeMap, _ := e.maps.Get("corridor")
	if s.Map == storeMap {
		t.Fatal("session shares the map store's *Map instead of a snapshot")
	}

	// Replace the map under the same id with a layout where the route is
	// different, then delete it entirely.
	edited := corridorCfg
	edited.Levels = [][]string{{".....", "S....", "...F."}}
	if _, err := e.svc.SaveMap(edited); err != nil {
		t.Fatal(err)
	}
	if err := e.svc.DeleteMap("corridor"); err != nil {
		t.Fatal(err)
	}
	if _, err := e.svc.GetMap("corridor"); err == nil {
		t.Fatal("map still present")
	}

	got := e.get(t, s.ID)
	if !reflect.DeepEqual(got.Map.Levels, corridorCfg.Levels) {
		t.Fatalf("snapshot layout changed: %v", got.Map.Levels)
	}
	done := e.bulk(t, s.ID, repeat(right, 4)...)
	if done.State.Status != engine.StatusWon || done.State.Moves != 4 {
		t.Fatalf("after map edit/delete: %s in %d moves, want WON in 4", done.State.Status, done.State.Moves)
	}
	_, err := e.svc.CreateSession("corridor", "")
	wantCode(t, err, service.CodeNotFound)
}

func TestMoveToWon(t *testing.T) {
	e := newEnv(t, envOpts{})
	s := e.create(t, "corridor")

	prev := s
	for i := 1; i <= 4; i++ {
		got := e.move(t, s.ID, right)
		wantStatus := engine.StatusPlaying
		if i == 4 {
			wantStatus = engine.StatusWon
		}
		if got.State.Moves != i || got.State.Status != wantStatus || got.Seq != prev.Seq+1 {
			t.Fatalf("move %d: moves=%d status=%s seq=%d (prev %d)", i, got.State.Moves, got.State.Status, got.Seq, prev.Seq)
		}
		if ev := got.LastEvent; ev == nil || ev.Move != i || ev.Direction != right || ev.Blocked || ev.Via != engine.ViaWalk || ev.Status != wantStatus {
			t.Fatalf("move %d: LastEvent %+v", i, got.LastEvent)
		}
		if got.State.Pos != (engine.Position{X: i, Y: 1}) || len(got.State.Path) != i+1 {
			t.Fatalf("move %d: pos %v path %v", i, got.State.Pos, got.State.Path)
		}
		if n := e.pers.saveCount(s.ID); n != 1+i {
			t.Fatalf("move %d: saves = %d, want %d (every state-changing move persists)", i, n, 1+i)
		}
		prev = got
	}
	if prev.Attempts != 1 || prev.BestMoves == nil || *prev.BestMoves != 4 {
		t.Fatalf("attempts/best = %d/%v, want 1/4", prev.Attempts, prev.BestMoves)
	}
	if saved := e.pers.latest(s.ID); saved.State.Status != engine.StatusWon || saved.Attempts != 1 {
		t.Fatalf("persisted state is stale: %s attempts %d", saved.State.Status, saved.Attempts)
	}
}

// A blocked move is not an error and not counted, but it is an event: Seq
// bumps and it is broadcast so UIs can flash a bump. Nothing durable changed,
// so it is not persisted.
func TestBlockedMoveIsVisibleButFree(t *testing.T) {
	e := newEnv(t, envOpts{})
	s := e.create(t, "corridor")
	e.move(t, s.ID, right)
	before := e.get(t, s.ID)
	saves, casts := e.pers.saveCount(s.ID), len(e.rec.forSession(s.ID))

	for _, d := range []engine.Direction{up, down} {
		got, err := e.svc.Move(s.ID, d)
		if err != nil {
			t.Fatalf("blocked %s returned an error: %v", d, err)
		}
		ev := got.LastEvent
		if ev == nil || !ev.Blocked || ev.Via != engine.ViaNone || ev.Direction != d || ev.From != before.State.Pos || ev.To != before.State.Pos || ev.Move != 1 {
			t.Fatalf("blocked %s: LastEvent %+v", d, ev)
		}
		if got.State.Moves != 1 || got.State.Pos != before.State.Pos || got.State.Status != engine.StatusPlaying || len(got.State.Path) != 2 {
			t.Fatalf("blocked %s changed the game: %+v", d, got.State)
		}
		before.Seq++
		if got.Seq != before.Seq {
			t.Fatalf("blocked %s: seq %d, want %d", d, got.Seq, before.Seq)
		}
	}
	if e.pers.saveCount(s.ID) != saves {
		t.Fatal("blocked moves were persisted")
	}
	if n := len(e.rec.forSession(s.ID)); n != casts+2 {
		t.Fatalf("broadcasts = %d, want %d (blocked moves are broadcast)", n, casts+2)
	}

	// Blocked at the very start keeps the session READY and counts nothing.
	s2 := e.create(t, "corridor")
	got := e.move(t, s2.ID, left)
	if got.State.Status != engine.StatusReady || got.State.Moves != 0 || !got.LastEvent.Blocked {
		t.Fatalf("blocked first move: %+v / %+v", got.State, got.LastEvent)
	}
}

func TestStairsAndPortalMoves(t *testing.T) {
	e := newEnv(t, envOpts{})
	s := e.create(t, "tower")

	e.move(t, s.ID, right)
	got := e.move(t, s.ID, right) // stairs up
	ev := got.LastEvent
	if ev.Via != engine.ViaStairsUp || ev.Stepped != (engine.Position{Level: 0, X: 2, Y: 0}) || ev.To != (engine.Position{Level: 1, X: 2, Y: 0}) {
		t.Fatalf("stairs event: %+v", ev)
	}
	if got.State.Pos != ev.To || got.State.Moves != 2 {
		t.Fatalf("after stairs: pos %v moves %d", got.State.Pos, got.State.Moves)
	}
	got = e.move(t, s.ID, right) // portal 1 -> level 0 (3,2)
	if got.LastEvent.Via != engine.ViaPortal || got.State.Pos != (engine.Position{Level: 0, X: 3, Y: 2}) {
		t.Fatalf("portal: %+v pos %v", got.LastEvent, got.State.Pos)
	}
	got = e.move(t, s.ID, up) // stairs at level 0 (3,1): land on the D above, not yet won
	if got.LastEvent.Via != engine.ViaStairsUp || got.State.Pos != pos(1, 3, 1) || got.State.Status != engine.StatusPlaying {
		t.Fatalf("second stairs: %+v pos %v", got.LastEvent, got.State.Pos)
	}
	got = e.move(t, s.ID, down)
	if got.State.Status != engine.StatusWon || got.State.Moves != 5 || got.State.Moves != got.Map.Par {
		t.Fatalf("finish: %s in %d moves (par %d)", got.State.Status, got.State.Moves, got.Map.Par)
	}
	want := []engine.Position{
		pos(0, 0, 0), pos(0, 1, 0), pos(0, 2, 0), pos(1, 2, 0), pos(1, 3, 0), pos(0, 3, 2), pos(0, 3, 1), pos(1, 3, 1), pos(1, 3, 2),
	}
	if !reflect.DeepEqual(got.State.Path, want) {
		t.Fatalf("path = %v, want %v", got.State.Path, want)
	}
}

func TestInvalidDirectionRejected(t *testing.T) {
	e := newEnv(t, envOpts{})
	s := e.create(t, "corridor")
	for _, d := range []engine.Direction{-1, 4} {
		_, err := e.svc.Move(s.ID, d)
		wantCode(t, err, service.CodeInvalidArgument)
		_, err = e.svc.BulkMove(s.ID, []engine.Direction{right, d}, false)
		wantCode(t, err, service.CodeInvalidArgument)
		_, err = e.svc.Simulate("corridor", []engine.Direction{d}, false)
		wantCode(t, err, service.CodeInvalidArgument)
	}
	if got := e.get(t, s.ID); got.Seq != s.Seq || got.State.Moves != 0 {
		t.Fatal("a rejected direction changed the session")
	}
}

func TestMoveNotFound(t *testing.T) {
	e := newEnv(t, envOpts{})
	_, err := e.svc.Move("0000000000000000", right)
	wantCode(t, err, service.CodeNotFound)
	_, err = e.svc.BulkMove("0000000000000000", []engine.Direction{right}, false)
	wantCode(t, err, service.CodeNotFound)
	_, err = e.svc.Reset("0000000000000000")
	wantCode(t, err, service.CodeNotFound)
}

func TestTerminalSessionRejectsMovesUntilReset(t *testing.T) {
	e := newEnv(t, envOpts{})
	s := e.create(t, "corridor")
	won := e.bulk(t, s.ID, repeat(right, 4)...)

	_, err := e.svc.Move(s.ID, left)
	wantCode(t, err, service.CodeSessionTerminal)
	_, err = e.svc.BulkMove(s.ID, []engine.Direction{left}, false)
	wantCode(t, err, service.CodeSessionTerminal)
	_, err = e.svc.BulkMove(s.ID, nil, false)
	wantCode(t, err, service.CodeSessionTerminal)
	if got := e.get(t, s.ID); got.Seq != won.Seq || got.State.Moves != 4 || got.Attempts != 1 {
		t.Fatalf("rejected moves changed the session: seq %d moves %d attempts %d", got.Seq, got.State.Moves, got.Attempts)
	}

	r, err := e.svc.Reset(s.ID)
	if err != nil {
		t.Fatal(err)
	}
	if r.State.Status != engine.StatusReady || r.State.Moves != 0 || r.State.Pos != r.Map.Start || len(r.State.Path) != 1 ||
		r.LastEvent != nil || r.Attempts != 1 || r.BestMoves == nil || *r.BestMoves != 4 || r.Seq != won.Seq+1 {
		t.Fatalf("Reset: %+v event %v attempts %d best %v seq %d", r.State, r.LastEvent, r.Attempts, r.BestMoves, r.Seq)
	}
	if again := e.bulk(t, s.ID, repeat(right, 4)...); again.Attempts != 2 || again.State.Status != engine.StatusWon {
		t.Fatalf("replay after reset: attempts %d status %s", again.Attempts, again.State.Status)
	}
}

func TestAttemptsAndBestMoves(t *testing.T) {
	e := newEnv(t, envOpts{})
	s := e.create(t, "bypass")
	fast := []engine.Direction{down, down, right, right}
	slow := []engine.Direction{right, right, right, down, down, left}

	play := func(dirs ...engine.Direction) *service.Session {
		t.Helper()
		if _, err := e.svc.Reset(s.ID); err != nil {
			t.Fatal(err)
		}
		return e.bulk(t, s.ID, dirs...)
	}
	best := func(g *service.Session) int {
		t.Helper()
		if g.BestMoves == nil {
			t.Fatal("BestMoves is nil after a win")
		}
		return *g.BestMoves
	}

	// Winning on the very last allowed move still wins.
	if got := play(slow...); got.State.Status != engine.StatusWon || got.State.Moves != 6 || got.Attempts != 1 || best(got) != 6 {
		t.Fatalf("slow win: %s moves %d attempts %d", got.State.Status, got.State.Moves, got.Attempts)
	}
	// Reset never counts as an attempt.
	if got, _ := e.svc.Reset(s.ID); got.Attempts != 1 || best(got) != 6 {
		t.Fatalf("Reset changed counters: %d/%v", got.Attempts, got.BestMoves)
	}
	if got := play(fast...); got.Attempts != 2 || best(got) != 4 {
		t.Fatalf("faster win: attempts %d best %v, want 2/4", got.Attempts, got.BestMoves)
	}
	if got := play(slow...); got.Attempts != 3 || best(got) != 4 {
		t.Fatalf("slower win must not raise best: attempts %d best %v", got.Attempts, got.BestMoves)
	}
	// Six counted moves that never reach F end in LOST and do not touch best.
	got := play(right, left, right, left, right, left)
	if got.State.Status != engine.StatusLost || got.State.LossReason != engine.LossMoveLimit || got.Attempts != 4 || best(got) != 4 {
		t.Fatalf("loss: %s/%s attempts %d best %v", got.State.Status, got.State.LossReason, got.Attempts, got.BestMoves)
	}
}

func TestLostByMoveLimit(t *testing.T) {
	e := newEnv(t, envOpts{})
	s := e.create(t, "limit")
	got := e.bulk(t, s.ID, repeat(right, 4)...) // the 4th move is never played
	if got.State.Status != engine.StatusLost || got.State.LossReason != engine.LossMoveLimit || got.State.Moves != 3 {
		t.Fatalf("limit: %s/%s moves %d", got.State.Status, got.State.LossReason, got.State.Moves)
	}
	if got.Attempts != 1 || got.BestMoves != nil {
		t.Fatalf("a loss counts an attempt but sets no best: %d/%v", got.Attempts, got.BestMoves)
	}
	if ev := got.LastEvent; ev.Status != engine.StatusLost || ev.LossReason != engine.LossMoveLimit || ev.Move != 3 {
		t.Fatalf("LastEvent %+v", ev)
	}
	_, err := e.svc.Move(s.ID, right)
	wantCode(t, err, service.CodeSessionTerminal)
}

func TestBulkMove(t *testing.T) {
	e := newEnv(t, envOpts{})
	s := e.create(t, "corridor")
	saves, casts := e.pers.saveCount(s.ID), len(e.rec.forSession(s.ID))

	// Blocked moves are processed and ignored; one Seq bump, one broadcast, one
	// persist for the whole batch; LastEvent is the last attempt made.
	got := e.bulk(t, s.ID, up, right, down, right, right, right, right, left)
	if got.State.Status != engine.StatusWon || got.State.Moves != 4 || got.State.Pos != (engine.Position{X: 4, Y: 1}) {
		t.Fatalf("bulk: %s moves %d pos %v", got.State.Status, got.State.Moves, got.State.Pos)
	}
	if got.Seq != s.Seq+1 || len(e.rec.forSession(s.ID)) != casts+1 || e.pers.saveCount(s.ID) != saves+1 {
		t.Fatalf("seq %d, broadcasts %d, saves %d: want one bump/broadcast/persist", got.Seq, len(e.rec.forSession(s.ID))-casts, e.pers.saveCount(s.ID)-saves)
	}
	if got.Attempts != 1 || *got.BestMoves != 4 {
		t.Fatalf("attempts/best = %d/%d", got.Attempts, *got.BestMoves)
	}
	if ev := got.LastEvent; ev.Blocked || ev.Status != engine.StatusWon || ev.Move != 4 {
		t.Fatalf("LastEvent %+v, want the winning move", ev)
	}

	// The two trailing directions after the win were not played.
	if want := []engine.Position{pos(0, 0, 1), pos(0, 1, 1), pos(0, 2, 1), pos(0, 3, 1), pos(0, 4, 1)}; !reflect.DeepEqual(got.State.Path, want) {
		t.Fatalf("path = %v, want %v", got.State.Path, want)
	}
}

func TestBulkMoveEdgeCases(t *testing.T) {
	e := newEnv(t, envOpts{})
	s := e.create(t, "corridor")

	// Empty batch is a no-op: no bump, no broadcast.
	casts := len(e.rec.forSession(s.ID))
	got, err := e.svc.BulkMove(s.ID, nil, false)
	if err != nil || got.Seq != s.Seq || len(e.rec.forSession(s.ID)) != casts {
		t.Fatalf("empty bulk: err %v seq %d", err, got.Seq)
	}

	// A batch of blocked moves only is visible (seq bump, broadcast) but not
	// persisted.
	saves := e.pers.saveCount(s.ID)
	got = e.bulk(t, s.ID, up, down, left)
	if got.Seq != s.Seq+1 || got.State.Moves != 0 || got.State.Status != engine.StatusReady || !got.LastEvent.Blocked || got.LastEvent.Direction != left {
		t.Fatalf("blocked-only bulk: seq %d %+v %+v", got.Seq, got.State, got.LastEvent)
	}
	if e.pers.saveCount(s.ID) != saves || len(e.rec.forSession(s.ID)) != casts+1 {
		t.Fatal("blocked-only bulk must broadcast but not persist")
	}

	// Size cap.
	_, err = e.svc.BulkMove(s.ID, repeat(up, service.MaxDirections+1), false)
	wantCode(t, err, service.CodeInvalidArgument)
	if _, err = e.svc.BulkMove(s.ID, repeat(up, service.MaxDirections), false); err != nil {
		t.Fatalf("exactly the cap must be accepted: %v", err)
	}
}

// A wrong plan must not keep running from the wrong tile: with stopOnBlocked the
// batch ends at the first blocked attempt (which is reported in LastEvent).
func TestBulkMoveStopOnBlocked(t *testing.T) {
	e := newEnv(t, envOpts{})
	s := e.create(t, "corridor")

	got, err := e.svc.BulkMove(s.ID, []engine.Direction{right, up, right, right}, true)
	if err != nil {
		t.Fatal(err)
	}
	if got.State.Pos != (engine.Position{X: 1, Y: 1}) || got.State.Moves != 1 {
		t.Fatalf("pos %v moves %d, want (1,1) after 1 counted move", got.State.Pos, got.State.Moves)
	}
	if ev := got.LastEvent; !ev.Blocked || ev.Direction != up {
		t.Fatalf("LastEvent %+v, want the blocked UP attempt", ev)
	}

	// With stopOnBlocked false the same plan skips the bump and carries on.
	got, err = e.svc.BulkMove(s.ID, []engine.Direction{up, right}, false)
	if err != nil {
		t.Fatal(err)
	}
	if got.State.Pos != (engine.Position{X: 2, Y: 1}) || got.State.Moves != 2 {
		t.Fatalf("pos %v moves %d, want (2,1) after 2 counted moves", got.State.Pos, got.State.Moves)
	}
}

func TestBroadcastSeqMonotonicAndComplete(t *testing.T) {
	e := newEnv(t, envOpts{})
	s := e.create(t, "corridor")
	e.move(t, s.ID, up) // blocked
	e.move(t, s.ID, right)
	e.bulk(t, s.ID, right, right, right)
	if _, err := e.svc.Reset(s.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := e.svc.RenameSession(s.ID, "x"); err != nil {
		t.Fatal(err)
	}

	snaps := e.rec.forSession(s.ID)
	if len(snaps) != 6 {
		t.Fatalf("%d broadcasts, want 6 (create, blocked, move, bulk, reset, rename)", len(snaps))
	}
	for i, sn := range snaps {
		if sn.Seq != uint64(i+1) {
			t.Fatalf("broadcast %d has seq %d, want %d (every change +1, in order)", i, sn.Seq, i+1)
		}
	}
	if !snaps[1].LastEvent.Blocked || snaps[3].State.Status != engine.StatusWon || snaps[4].State.Status != engine.StatusReady {
		t.Fatal("broadcast snapshots do not carry the expected events/state")
	}
}

func TestDeleteSession(t *testing.T) {
	e := newEnv(t, envOpts{})
	s := e.create(t, "corridor")
	e.move(t, s.ID, right)

	if err := e.svc.DeleteSession(s.ID); err != nil {
		t.Fatal(err)
	}
	_, err := e.svc.GetSession(s.ID)
	wantCode(t, err, service.CodeNotFound)
	wantCode(t, e.svc.DeleteSession(s.ID), service.CodeNotFound)
	_, err = e.svc.Move(s.ID, right)
	wantCode(t, err, service.CodeNotFound)
	if list, _ := e.svc.ListSessions(service.SessionFilter{}); len(list) != 0 {
		t.Fatalf("deleted session still listed: %d", len(list))
	}
}

func TestShutdownPersistsEverythingAndIsIdempotent(t *testing.T) {
	e := newEnv(t, envOpts{})
	var ids []string
	for range 3 {
		s := e.create(t, "corridor")
		e.move(t, s.ID, right)
		e.move(t, s.ID, up) // blocked: bumps seq without persisting
		ids = append(ids, s.ID)
	}
	e.svc.Shutdown()
	for _, id := range ids {
		got := e.get(t, id)
		saved := e.pers.latest(id)
		if saved == nil || saved.Seq != got.Seq || !reflect.DeepEqual(saved.State, got.State) {
			t.Fatalf("%s: persisted state is not the final state: %+v vs %+v", id, saved, got)
		}
	}
	e.svc.Shutdown()
	if got := e.get(t, ids[0]); got.State.Moves != 1 {
		t.Fatalf("Shutdown changed the game: moves %d", got.State.Moves)
	}
}

func TestSimulate(t *testing.T) {
	e := newEnv(t, envOpts{})

	res, err := e.svc.Simulate("corridor", repeat(right, 4), true)
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != engine.StatusWon || res.Moves != 4 || len(res.Events) != 4 || res.Map == nil || res.FinalState.Moves != 4 {
		t.Fatalf("simulate: %s moves %d events %d", res.Status, res.Moves, len(res.Events))
	}
	res, err = e.svc.Simulate("corridor", repeat(right, 4), false)
	if err != nil || res.Status != engine.StatusWon || res.Moves != 4 || res.Events != nil {
		t.Fatalf("no-events simulate: %v %+v", err, res)
	}

	// Blocked attempts are events; moves after the end are ignored.
	res, err = e.svc.Simulate("corridor", []engine.Direction{up, right, right, right, right, right, left}, true)
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != engine.StatusWon || res.Moves != 4 || len(res.Events) != 5 || !res.Events[0].Blocked || res.Events[4].Status != engine.StatusWon {
		t.Fatalf("blocked/trailing: %s moves %d events %+v", res.Status, res.Moves, res.Events)
	}

	res, err = e.svc.Simulate("limit", repeat(right, 4), false)
	if err != nil || res.Status != engine.StatusLost || res.LossReason != engine.LossMoveLimit || res.Moves != 3 {
		t.Fatalf("limit: %v %+v", err, res)
	}

	// No moves: READY, and asking for events yields an empty (non-nil) list.
	res, err = e.svc.Simulate("corridor", nil, true)
	if err != nil || res.Status != engine.StatusReady || res.Moves != 0 || res.Events == nil || len(res.Events) != 0 {
		t.Fatalf("empty: %v %+v", err, res)
	}

	_, err = e.svc.Simulate("nope", repeat(right, 4), false)
	wantCode(t, err, service.CodeNotFound)
	_, err = e.svc.Simulate("corridor", repeat(up, service.MaxDirections+1), false)
	wantCode(t, err, service.CodeInvalidArgument)
	if e.mgr.Count() != 0 {
		t.Fatal("Simulate created a session")
	}
}

func TestMapCRUD(t *testing.T) {
	e := newEnv(t, envOpts{})
	if got := e.svc.ListMaps(); len(got) != 5 || got[0].ID != "bypass" {
		t.Fatalf("ListMaps: %d maps, first %q", len(got), got[0].ID)
	}
	extra := engine.MapConfig{ID: "extra", Name: "Extra", Levels: [][]string{{"S.F", "...", "..."}}}
	if _, err := e.svc.CreateMap(extra); err != nil {
		t.Fatal(err)
	}
	_, err := e.svc.CreateMap(extra)
	wantCode(t, err, service.CodeInvalidArgument)
	if _, err := e.svc.SaveMap(extra); err != nil {
		t.Fatalf("SaveMap overwrites: %v", err)
	}
	if m, err := e.svc.GetMap("extra"); err != nil || m.Par != -1 {
		t.Fatalf("GetMap: %v %+v", err, m)
	}
	bad := engine.MapConfig{ID: "bad", Name: "Bad", Levels: [][]string{{"..."}}}
	_, err = e.svc.CreateMap(bad)
	wantCode(t, err, service.CodeInvalidArgument)
	if err := e.svc.DeleteMap("extra"); err != nil {
		t.Fatal(err)
	}
	wantCode(t, e.svc.DeleteMap("extra"), service.CodeNotFound)
}

func TestListSessions(t *testing.T) {
	e := newEnv(t, envOpts{})
	a := e.create(t, "corridor")
	time.Sleep(3 * time.Millisecond)
	b := e.create(t, "long")
	time.Sleep(3 * time.Millisecond)
	c := e.create(t, "corridor")
	time.Sleep(3 * time.Millisecond)
	// Touch a so it is the most recently active.
	if _, err := e.svc.RenameSession(a.ID, "a"); err != nil {
		t.Fatal(err)
	}

	ids := func(f service.SessionFilter) []string {
		t.Helper()
		l, err := e.svc.ListSessions(f)
		if err != nil {
			t.Fatal(err)
		}
		var out []string
		for _, s := range l {
			out = append(out, s.ID)
		}
		return out
	}
	if got, want := ids(service.SessionFilter{}), []string{a.ID, c.ID, b.ID}; !reflect.DeepEqual(got, want) {
		t.Fatalf("recent = %v, want %v", got, want)
	}
	if got, want := ids(service.SessionFilter{Sort: "created"}), []string{c.ID, b.ID, a.ID}; !reflect.DeepEqual(got, want) {
		t.Fatalf("created = %v, want %v", got, want)
	}
	if got, want := ids(service.SessionFilter{Sort: "CREATED", Limit: 2}), []string{c.ID, b.ID}; !reflect.DeepEqual(got, want) {
		t.Fatalf("limit 2 = %v, want %v", got, want)
	}
	if got, want := ids(service.SessionFilter{MapID: "corridor"}), []string{a.ID, c.ID}; !reflect.DeepEqual(got, want) {
		t.Fatalf("map filter = %v, want %v", got, want)
	}
	if got := ids(service.SessionFilter{MapID: "unknown"}); len(got) != 0 {
		t.Fatalf("unknown map filter = %v", got)
	}
	if got := ids(service.SessionFilter{Limit: 100000}); len(got) != 3 {
		t.Fatalf("limit is clamped, not rejected: %v", got)
	}
	_, err := e.svc.ListSessions(service.SessionFilter{Limit: -1})
	wantCode(t, err, service.CodeInvalidArgument)
	_, err = e.svc.ListSessions(service.SessionFilter{Sort: "bogus"})
	wantCode(t, err, service.CodeInvalidArgument)
}

func TestListSessionsDefaultLimit(t *testing.T) {
	e := newEnv(t, envOpts{maxSessions: 600})
	for range service.MaxListLimit + 1 {
		e.create(t, "corridor")
	}
	l, _ := e.svc.ListSessions(service.SessionFilter{})
	if len(l) != service.DefaultListLimit {
		t.Fatalf("default limit: %d, want %d", len(l), service.DefaultListLimit)
	}
	l, _ = e.svc.ListSessions(service.SessionFilter{Limit: 100000})
	if len(l) != service.MaxListLimit {
		t.Fatalf("max limit: %d, want %d", len(l), service.MaxListLimit)
	}
}

func TestMaxSessions(t *testing.T) {
	e := newEnv(t, envOpts{maxSessions: 2})
	a := e.create(t, "corridor")
	e.create(t, "corridor")
	_, err := e.svc.CreateSession("corridor", "")
	wantCode(t, err, service.CodeLimit)
	if err := e.svc.DeleteSession(a.ID); err != nil {
		t.Fatal(err)
	}
	e.create(t, "corridor")
}

func TestSessionTTLSweptOnNew(t *testing.T) {
	maps := newFakeMaps(t, corridorCfg)
	mgr := session.NewManager(nil, 10)
	m := mustMap(t, corridorCfg)
	put := func(id string, idle time.Duration) {
		now := time.Now()
		s := &service.Session{
			ID: id, MapID: m.ID, Map: m, State: engine.NewState(m),
			CreatedAt: now.Add(-idle), LastActionAt: now.Add(-idle),
		}
		if err := mgr.Put(s); err != nil {
			t.Fatal(err)
		}
	}
	put("0000000000000001", 48*time.Hour)
	put("0000000000000002", time.Hour)

	svc := service.New(maps, mgr, nil, service.Options{SessionTTL: 24 * time.Hour})
	t.Cleanup(svc.Shutdown)
	if _, ok := mgr.Get("0000000000000001"); ok {
		t.Fatal("expired session survived")
	}
	if _, ok := mgr.Get("0000000000000002"); !ok {
		t.Fatal("fresh session was swept")
	}
}

// TestConcurrentMutations hammers one session from many goroutines. It must
// be race-free, return only coded errors, and every Seq bump must be
// broadcast exactly once, in a gap-free sequence.
func TestConcurrentMutations(t *testing.T) {
	e := newEnv(t, envOpts{})
	s := e.create(t, "bypass")

	var ok atomic.Int64 // successful mutations (each bumps Seq once)
	isTerminal := func(err error) bool {
		se, isSE := err.(*service.Error)
		return isSE && se.Code == service.CodeSessionTerminal
	}

	var wg sync.WaitGroup
	for g := range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range 60 {
				var err error
				mutates := true
				switch (i + g) % 7 {
				case 0, 1:
					_, err = e.svc.Move(s.ID, engine.Direction((i+g)%4))
				case 2:
					_, err = e.svc.BulkMove(s.ID, []engine.Direction{down, down, right, right}, false)
				case 3:
					_, err = e.svc.Reset(s.ID)
				case 4:
					_, err = e.svc.RenameSession(s.ID, "n")
				case 5:
					mutates = false
					_, err = e.svc.GetSession(s.ID)
				case 6:
					mutates = false
					_, err = e.svc.ListSessions(service.SessionFilter{})
				}
				switch {
				case err == nil && mutates:
					ok.Add(1)
				case err == nil || isTerminal(err):
				default:
					t.Errorf("unexpected error: %v", err)
				}
			}
		}()
	}
	wg.Wait()

	final := e.get(t, s.ID)
	if want := uint64(1 + ok.Load()); final.Seq != want {
		t.Fatalf("final seq %d, want 1 + %d successful mutations = %d", final.Seq, ok.Load(), want)
	}
	snaps := e.rec.forSession(s.ID)
	seqs := make([]int, len(snaps))
	for i, sn := range snaps {
		seqs[i] = int(sn.Seq)
	}
	sort.Ints(seqs)
	for i, q := range seqs {
		if q != i+1 {
			t.Fatalf("broadcast seqs are not 1..%d exactly once: got %v at %d", len(seqs), q, i)
		}
	}
	if len(seqs) != int(final.Seq) {
		t.Fatalf("%d broadcasts for final seq %d", len(seqs), final.Seq)
	}
}
