package service_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/wricardo/drbrain-mazes/game/engine"
	"github.com/wricardo/drbrain-mazes/game/service"
	"github.com/wricardo/drbrain-mazes/game/session"
	"github.com/wricardo/drbrain-mazes/transport/websocket"
)

// boot builds a service on top of file persistence in dir, loading what is there.
func boot(t *testing.T, dir string, maps service.MapStore, bc service.Broadcaster) (service.GameService, *session.Manager) {
	t.Helper()
	fp, err := session.NewFilePersistence(dir)
	if err != nil {
		t.Fatal(err)
	}
	mgr := session.NewManager(fp, 100)
	if err := mgr.LoadAll(); err != nil {
		t.Fatal(err)
	}
	svc := service.New(maps, mgr, bc, service.Options{})
	t.Cleanup(svc.Shutdown)
	return svc, mgr
}

func mustGet(t *testing.T, svc service.GameService, id string) *service.Session {
	t.Helper()
	s, err := svc.GetSession(id)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestRestartRestoresSession(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "sessions")
	maps := newFakeMaps(t, towerCfg)

	svc1, _ := boot(t, dir, maps, nil)
	s, err := svc1.CreateSession("tower", "restart me")
	if err != nil {
		t.Fatal(err)
	}
	// Stairs up, portal to level 0, then a blocked LEFT (bumps seq, not persisted
	// until Shutdown).
	for _, d := range []engine.Direction{right, right, right, left} {
		if _, err := svc1.Move(s.ID, d); err != nil {
			t.Fatal(err)
		}
	}
	svc1.Shutdown()
	want := mustGet(t, svc1, s.ID)
	if want.State.Pos != (engine.Position{Level: 0, X: 3, Y: 2}) || want.State.Moves != 3 || !want.LastEvent.Blocked {
		t.Fatalf("pre-restart state: %+v", want.State)
	}

	raw, err := os.ReadFile(filepath.Join(dir, s.ID+".json"))
	if err != nil {
		t.Fatal(err)
	}
	var onDisk struct {
		SchemaVersion int `json:"schema_version"`
	}
	if err := json.Unmarshal(raw, &onDisk); err != nil || onDisk.SchemaVersion != session.SchemaVersion {
		t.Fatalf("schema_version = %d (%v)", onDisk.SchemaVersion, err)
	}

	// "Restart": brand-new manager and service over the same directory.
	svc2, _ := boot(t, dir, maps, nil)
	got, err := svc2.GetSession(s.ID)
	if err != nil {
		t.Fatalf("session lost across restart: %v", err)
	}
	if got.DisplayName != "restart me" || got.MapID != "tower" || got.Seq != want.Seq || got.Attempts != want.Attempts {
		t.Fatalf("metadata differs: %+v", got)
	}
	if !reflect.DeepEqual(got.State, want.State) || !reflect.DeepEqual(got.Map.Config(), want.Map.Config()) {
		t.Fatalf("state/map not restored intact:\n got %+v\nwant %+v", got.State, want.State)
	}
	if got.LastEvent != nil {
		t.Fatal("LastEvent is transient and must not be restored")
	}

	// Play continues from the restored position instead of starting over.
	if _, err := svc2.Move(s.ID, up); err != nil { // stairs: land on the D above
		t.Fatal(err)
	}
	next, err := svc2.Move(s.ID, down)
	if err != nil {
		t.Fatal(err)
	}
	if next.State.Status != engine.StatusWon || next.State.Moves != 5 || next.Attempts != 1 || *next.BestMoves != 5 {
		t.Fatalf("resumed move: %+v attempts %d", next.State, next.Attempts)
	}
}

func TestRestartKeepsOutcomeAndCounters(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "sessions")
	maps := newFakeMaps(t, corridorCfg)

	svc1, _ := boot(t, dir, maps, nil)
	s, _ := svc1.CreateSession("corridor", "")
	want, err := svc1.BulkMove(s.ID, repeat(right, 4))
	if err != nil {
		t.Fatal(err)
	}
	// The terminal outcome reaches the disk without Shutdown: every
	// state-changing move persists.
	fp, err := session.NewFilePersistence(dir)
	if err != nil {
		t.Fatal(err)
	}
	onDisk, err := fp.Load(s.ID)
	if err != nil || onDisk.State.Status != engine.StatusWon {
		t.Fatalf("outcome not on disk: %v %+v", err, onDisk)
	}

	svc2, _ := boot(t, dir, maps, nil)
	got, err := svc2.GetSession(s.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Attempts != 1 || got.BestMoves == nil || *got.BestMoves != 4 || !reflect.DeepEqual(got.State, want.State) {
		t.Fatalf("restored: %+v attempts %d best %v", got.State, got.Attempts, got.BestMoves)
	}
	_, err = svc2.Move(s.ID, left)
	wantCode(t, err, service.CodeSessionTerminal)
	if r, err := svc2.Reset(s.ID); err != nil || r.Attempts != 1 || *r.BestMoves != 4 || r.State.Status != engine.StatusReady {
		t.Fatalf("Reset after restart: %v %+v", err, r)
	}
}

func TestDeleteSessionRemovesFile(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "sessions")
	maps := newFakeMaps(t, corridorCfg)
	svc, _ := boot(t, dir, maps, nil)
	s, _ := svc.CreateSession("corridor", "")
	if _, err := svc.Move(s.ID, right); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, s.ID+".json")
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("session file missing: %v", err)
	}
	if err := svc.DeleteSession(s.ID); err != nil {
		t.Fatal(err)
	}
	svc.Shutdown() // persists everything still registered: must not bring the file back
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("deleted session file exists: %v", err)
	}
	svc2, _ := boot(t, dir, maps, nil)
	if _, err := svc2.GetSession(s.ID); err == nil {
		t.Fatal("deleted session came back after restart")
	}
}

func TestMaxSessionsFromManager(t *testing.T) {
	maps := newFakeMaps(t, corridorCfg)
	mgr := session.NewManager(nil, 2)
	svc := service.New(maps, mgr, nil, service.Options{})
	t.Cleanup(svc.Shutdown)
	for range 2 {
		if _, err := svc.CreateSession("corridor", ""); err != nil {
			t.Fatal(err)
		}
	}
	_, err := svc.CreateSession("corridor", "")
	wantCode(t, err, service.CodeLimit)
}

// TestSlowSubscriberStillGetsTerminalUpdate plays a 19-move win (more updates
// than the 8-slot subscriber buffer) with a subscriber that reads nothing
// until the game has finished: the final WON snapshot must still be delivered.
func TestSlowSubscriberStillGetsTerminalUpdate(t *testing.T) {
	hub := websocket.NewHub()
	e := newEnv(t, envOpts{bc: hub})
	s := e.create(t, "long")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	updates := hub.SubscribeSession(ctx, s.ID)

	var done *service.Session
	for range 19 {
		done = e.move(t, s.ID, right)
	}
	if done.State.Status != engine.StatusWon || done.State.Moves != 19 {
		t.Fatalf("game ended %s in %d moves", done.State.Status, done.State.Moves)
	}

	var got []*service.Session
	for len(updates) > 0 {
		got = append(got, <-updates)
	}
	if len(got) == 0 || len(got) > 8 {
		t.Fatalf("buffered %d updates, want 1..8", len(got))
	}
	last := got[len(got)-1]
	if last.State.Status != engine.StatusWon || last.Seq != done.Seq {
		t.Fatalf("last buffered update is %s seq %d, want WON seq %d", last.State.Status, last.Seq, done.Seq)
	}
	for i := 1; i < len(got); i++ {
		if got[i].Seq <= got[i-1].Seq {
			t.Fatalf("seq not strictly increasing: %d then %d", got[i-1].Seq, got[i].Seq)
		}
	}
}

// TestSubscriberSeesMovesAndReset: spectators must see every mutation,
// including blocked moves and resets.
func TestSubscriberSeesMovesAndReset(t *testing.T) {
	hub := websocket.NewHub()
	e := newEnv(t, envOpts{bc: hub})
	s := e.create(t, "corridor")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	updates := hub.SubscribeSession(ctx, s.ID)

	next := func() *service.Session {
		t.Helper()
		select {
		case u := <-updates:
			return u
		case <-ctx.Done():
			t.Fatal("timed out waiting for an update")
			return nil
		}
	}

	e.move(t, s.ID, up)
	if u := next(); u.LastEvent == nil || !u.LastEvent.Blocked || u.State.Moves != 0 {
		t.Fatalf("blocked move update: %+v / %+v", u.State, u.LastEvent)
	}
	e.move(t, s.ID, right)
	if u := next(); u.State.Moves != 1 || u.LastEvent.Via != engine.ViaWalk {
		t.Fatalf("move update: %+v", u.State)
	}
	e.bulk(t, s.ID, repeat(right, 3)...)
	if u := next(); u.State.Status != engine.StatusWon {
		t.Fatalf("bulk update: %s", u.State.Status)
	}
	if _, err := e.svc.Reset(s.ID); err != nil {
		t.Fatal(err)
	}
	if u := next(); u.State.Moves != 0 || u.State.Status != engine.StatusReady || u.LastEvent != nil {
		t.Fatalf("Reset update: %+v", u.State)
	}
}
