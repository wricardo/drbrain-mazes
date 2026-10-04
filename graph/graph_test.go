package graph

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"reflect"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/vektah/gqlparser/v2/gqlerror"
	"github.com/wricardo/drbrain-mazes/game/engine"
	"github.com/wricardo/drbrain-mazes/game/service"
	"github.com/wricardo/drbrain-mazes/graph/model"
	"github.com/wricardo/drbrain-mazes/transport/websocket"
)

// fakeService overrides only what the tests exercise; any other call panics
// on the nil embedded interface, which flags an unexpected dependency.
type fakeService struct {
	service.GameService
	maps     map[string]*engine.Map
	sessions map[string]*service.Session
	saved    []engine.MapConfig
	dirs     []engine.Direction
	err      error
}

func (f *fakeService) GetMap(id string) (*engine.Map, error) {
	if f.err != nil {
		return nil, f.err
	}
	if m, ok := f.maps[id]; ok {
		return m, nil
	}
	return nil, service.Errorf(service.CodeNotFound, "map %q not found", id)
}

func (f *fakeService) SaveMap(cfg engine.MapConfig) (*engine.Map, error) {
	m, err := engine.NewMap(cfg)
	if err != nil {
		return nil, err
	}
	f.saved = append(f.saved, cfg)
	return m, nil
}

func (f *fakeService) CreateMap(cfg engine.MapConfig) (*engine.Map, error) {
	if f.err != nil {
		return nil, f.err
	}
	if _, ok := f.maps[cfg.ID]; ok {
		return nil, service.Errorf(service.CodeInvalidArgument, "map %q already exists", cfg.ID)
	}
	return f.SaveMap(cfg)
}

func (f *fakeService) DeleteMap(id string) error { return nil }

func (f *fakeService) GetSession(id string) (*service.Session, error) {
	if s, ok := f.sessions[id]; ok {
		return s.Clone(), nil
	}
	return nil, service.Errorf(service.CodeNotFound, "session %q not found", id)
}

// Move records the direction and returns the configured error.
func (f *fakeService) Move(id string, d engine.Direction) (*service.Session, error) {
	f.dirs = []engine.Direction{d}
	if f.err != nil {
		return nil, f.err
	}
	return f.GetSession(id)
}

func (f *fakeService) BulkMove(id string, dirs []engine.Direction) (*service.Session, error) {
	f.dirs = dirs
	if f.err != nil {
		return nil, f.err
	}
	return f.GetSession(id)
}

func (f *fakeService) Simulate(mapID string, dirs []engine.Direction, includeEvents bool) (*service.SimulationResult, error) {
	m, err := f.GetMap(mapID)
	if err != nil {
		return nil, err
	}
	evs, st := engine.Simulate(m, dirs, includeEvents)
	return &service.SimulationResult{Status: st.Status, LossReason: st.LossReason, Moves: st.Moves, FinalState: st, Map: m, Events: evs}, nil
}

// Fog markers: rows that exist on exactly one level, so a leak of that level
// is detectable by substring.
const (
	level0Row = "SoU."
	level1Top = "..D."
	level1Row = "..U."
	level1Mrk = "1.2o"
	level2Row = ".FD2"
)

// testMapConfig is a 3-level tower with paired stairs. RIGHT, RIGHT takes the
// stairs up and lands on the D at level 1 (2,0); DOWN steps on the U at (2,1),
// landing on the D at level 2 (2,1); LEFT steps onto F: par 4.
// Portal 1 joins level 0 (3,1) with level 1 (0,2); portal 2 joins level 1
// (2,2) with level 2 (3,1).
func testMapConfig(id string) engine.MapConfig {
	return engine.MapConfig{
		ID: id, Name: id,
		Levels: [][]string{
			{level0Row, "...1", "...."},
			{level1Top, level1Row, level1Mrk},
			{"....", level2Row, "...."},
		},
	}
}

func mustMap(t *testing.T, id string) *engine.Map {
	t.Helper()
	m, err := engine.NewMap(testMapConfig(id))
	if err != nil {
		t.Fatal(err)
	}
	return m
}

func codeOf(t *testing.T, err error) string {
	t.Helper()
	var ge *gqlerror.Error
	if !errors.As(err, &ge) {
		t.Fatalf("want *gqlerror.Error, got %T (%v)", err, err)
	}
	code, _ := ge.Extensions["code"].(string)
	return code
}

func ctxWithKey(key string) context.Context {
	r, _ := http.NewRequest(http.MethodPost, "/graphql", nil)
	if key != "" {
		r.Header.Set("X-Admin-Key", key)
	}
	return context.WithValue(context.Background(), HTTPRequestKey{}, r)
}

func TestAdminGate(t *testing.T) {
	input := model.MapInput{ID: "new_map", Name: "n", Levels: testMapConfig("x").Levels}
	cases := []struct {
		name      string
		adminKey  string
		allowOpen string
		ctx       context.Context
		wantCode  string
	}{
		{"no key configured, not opted out", "", "", ctxWithKey(""), service.CodeForbidden},
		{"no key configured, opted out", "", "true", ctxWithKey(""), ""},
		{"key configured, header missing", "secret", "", ctxWithKey(""), service.CodeForbidden},
		{"key configured, wrong header", "secret", "", ctxWithKey("nope"), service.CodeForbidden},
		{"key configured, correct header", "secret", "", ctxWithKey("secret"), ""},
		{"key configured, no request in ctx", "secret", "", context.Background(), service.CodeForbidden},
		{"opt-out ignored when key configured", "secret", "true", ctxWithKey("nope"), service.CodeForbidden},
		{"key is prefix of header", "secret", "", ctxWithKey("secret-and-more"), service.CodeForbidden},
		{"header is prefix of key", "secret", "", ctxWithKey("secre"), service.CodeForbidden},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("ADMIN_API_KEY", tc.adminKey)
			t.Setenv("ALLOW_UNAUTHENTICATED_ADMIN", tc.allowOpen)
			fs := &fakeService{maps: map[string]*engine.Map{"existing": mustMap(t, "existing")}}
			r := &Resolver{Service: fs}
			m := &mutationResolver{r}

			check := func(op string, err error) {
				t.Helper()
				if tc.wantCode == "" {
					if err != nil {
						t.Fatalf("%s: unexpected error %v", op, err)
					}
					return
				}
				if err == nil {
					t.Fatalf("%s: expected %s error, got none", op, tc.wantCode)
				}
				if got := codeOf(t, err); got != tc.wantCode {
					t.Fatalf("%s: code = %q, want %q", op, got, tc.wantCode)
				}
			}
			_, err := m.CreateMap(tc.ctx, input)
			check("createMap", err)
			upd := input
			upd.ID = "existing"
			_, err = m.UpdateMap(tc.ctx, upd)
			check("updateMap", err)
			_, err = m.DeleteMap(tc.ctx, "existing")
			check("deleteMap", err)
			_, err = m.ValidateMap(tc.ctx, input)
			check("validateMap", err)
			if tc.wantCode != "" && len(fs.saved) != 0 {
				t.Fatalf("map saved despite gate: %v", fs.saved)
			}
		})
	}
}

func TestCreateUpdateMapSemantics(t *testing.T) {
	t.Setenv("ALLOW_UNAUTHENTICATED_ADMIN", "true")
	t.Setenv("ADMIN_API_KEY", "")
	fs := &fakeService{maps: map[string]*engine.Map{"existing": mustMap(t, "existing")}}
	m := &mutationResolver{&Resolver{Service: fs}}
	ctx := context.Background()
	levels := testMapConfig("x").Levels

	if _, err := m.CreateMap(ctx, model.MapInput{ID: "existing", Name: "n", Levels: levels}); codeOf(t, err) != service.CodeInvalidArgument {
		t.Fatalf("create existing: err = %v", err)
	}
	if _, err := m.UpdateMap(ctx, model.MapInput{ID: "missing", Name: "n", Levels: levels}); codeOf(t, err) != service.CodeNotFound {
		t.Fatalf("update missing: err = %v", err)
	}
	// Invalid levels from the store surfaces as INVALID_ARGUMENT.
	if _, err := m.CreateMap(ctx, model.MapInput{ID: "bad", Name: "n", Levels: [][]string{{"."}}}); codeOf(t, err) != service.CodeInvalidArgument {
		t.Fatalf("create invalid: err = %v", err)
	}
	got, err := m.CreateMap(ctx, model.MapInput{ID: "fresh", Name: "n", Levels: levels})
	if err != nil || got.ID != "fresh" || got.Start.X != 0 || got.Start.Y != 0 || got.Start.Level != 0 || *got.Finish != (model.Position{Level: 2, X: 1, Y: 1}) {
		t.Fatalf("create fresh = %+v, %v", got, err)
	}
}

func TestPublicMapConversionHidesLayout(t *testing.T) {
	m := mustMap(t, "m")
	got := toMap(m)
	if got.LevelCount != 3 || got.Width != 4 || got.Height != 3 || !reflect.DeepEqual(got.Preview, []string{level0Row, "...1", "...."}) {
		t.Fatalf("catalog entry: %+v", got)
	}
	if got.MaxMoves != engine.DefaultMaxMoves || got.ParMoves == nil || *got.ParMoves != 4 {
		t.Fatalf("maxMoves/parMoves = %d/%v, want default/4", got.MaxMoves, got.ParMoves)
	}
	raw, _ := json.Marshal(got)
	for _, leak := range []string{level1Row, level1Mrk, level2Row, "finish", "levels"} {
		if strings.Contains(string(raw), leak) {
			t.Fatalf("public map leaks %q: %s", leak, raw)
		}
	}
	got.Preview[0] = "xxxx"
	if m.Levels[0][0] != level0Row {
		t.Fatal("preview aliases the engine map")
	}

	// An unreachable finish hides par (null), not -1.
	cfg := testMapConfig("walled")
	cfg.Levels = [][]string{{"S..", ".o.", "..F"}}
	unreachable, err := engine.NewMap(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if toMap(unreachable).ParMoves != nil || toAdminMap(unreachable).ParMoves != nil {
		t.Fatal("parMoves must be null when the finish is unreachable")
	}
}

func TestAdminMapConversionIsFull(t *testing.T) {
	m := mustMap(t, "m")
	got := toAdminMap(m)
	if got.LevelCount != 3 || len(got.Levels) != 3 || got.Levels[2][1] != level2Row || got.Width != 4 || got.Height != 3 {
		t.Fatalf("levels/dimensions: %+v", got)
	}
	if *got.Start != (model.Position{}) || *got.Finish != (model.Position{Level: 2, X: 1, Y: 1}) || *got.ParMoves != 4 {
		t.Fatalf("start/finish/par: %v %v %v", got.Start, got.Finish, got.ParMoves)
	}
	if len(got.Portals) != 2 || got.Portals[0].Glyph != "1" || *got.Portals[0].A != (model.Position{Level: 0, X: 3, Y: 1}) || *got.Portals[0].B != (model.Position{Level: 1, X: 0, Y: 2}) {
		t.Fatalf("portals = %+v", got.Portals)
	}
	got.Levels[0][0] = "xxxx"
	if m.Levels[0][0] != level0Row {
		t.Fatal("converted levels alias the engine map")
	}
}

func TestAdminMapQueryGateAndLookup(t *testing.T) {
	fs := &fakeService{maps: map[string]*engine.Map{"m": mustMap(t, "m")}}
	q := &queryResolver{&Resolver{Service: fs}}

	t.Setenv("ADMIN_API_KEY", "secret")
	t.Setenv("ALLOW_UNAUTHENTICATED_ADMIN", "")
	for name, ctx := range map[string]context.Context{
		"no header":    ctxWithKey(""),
		"wrong key":    ctxWithKey("nope"),
		"no request":   context.Background(),
		"key prefix":   ctxWithKey("secre"),
		"key and tail": ctxWithKey("secret-and-more"),
	} {
		got, err := q.AdminMap(ctx, "m")
		if got != nil || codeOf(t, err) != service.CodeForbidden {
			t.Fatalf("%s: got %v err %v, want FORBIDDEN and no map", name, got, err)
		}
	}
	// The gate runs before the lookup: an unknown id must not reveal existence either.
	if _, err := q.AdminMap(ctxWithKey("nope"), "missing"); codeOf(t, err) != service.CodeForbidden {
		t.Fatalf("unauthenticated lookup of a missing id: %v", err)
	}

	got, err := q.AdminMap(ctxWithKey("secret"), "m")
	if err != nil || got == nil || len(got.Levels) != 3 {
		t.Fatalf("authorised: %+v %v", got, err)
	}
	if got, err := q.AdminMap(ctxWithKey("secret"), "missing"); got != nil || err != nil {
		t.Fatalf("missing map must be null for an admin: %v %v", got, err)
	}

	t.Setenv("ADMIN_API_KEY", "")
	t.Setenv("ALLOW_UNAUTHENTICATED_ADMIN", "true")
	if got, err := q.AdminMap(context.Background(), "m"); err != nil || got == nil {
		t.Fatalf("opt-out: %v %v", got, err)
	}
}

func TestMapInputConversion(t *testing.T) {
	desc, diff, max := "d", "hard", 77
	cfg := fromMapInput(model.MapInput{ID: "i", Name: "n", Description: &desc, Difficulty: &diff, MaxMoves: &max, Levels: [][]string{{"abc"}}})
	if cfg.Description != "d" || cfg.Difficulty != "hard" || cfg.MaxMoves != 77 || cfg.Levels[0][0] != "abc" {
		t.Fatalf("cfg = %+v", cfg)
	}
	if def := fromMapInput(model.MapInput{ID: "i", Name: "n"}); def.MaxMoves != 0 || def.Description != "" || def.Difficulty != "" {
		t.Fatalf("omitted optionals must stay zero (engine defaults apply): %+v", def)
	}
}

func TestDirectionConversion(t *testing.T) {
	for _, d := range []engine.Direction{engine.DirUp, engine.DirRight, engine.DirDown, engine.DirLeft} {
		if got := fromDirection(toDirection(d)); got != d {
			t.Errorf("round trip %s = %s", d, got)
		}
		if string(toDirection(d)) != d.String() {
			t.Errorf("enum name %q != %q", toDirection(d), d.String())
		}
	}
	if got := fromDirection("SIDEWAYS"); got >= 0 {
		t.Errorf("unknown direction must map to an invalid value, got %d", got)
	}
}

func TestMoveResolversConvertAndPropagate(t *testing.T) {
	fs := &fakeService{sessions: map[string]*service.Session{"s": liveSession(t, 2)}}
	m := &mutationResolver{&Resolver{Service: fs}}
	ctx := context.Background()

	got, err := m.Move(ctx, "s", model.DirectionLeft)
	if err != nil || got.ID != "s1" || !reflect.DeepEqual(fs.dirs, []engine.Direction{engine.DirLeft}) {
		t.Fatalf("Move: %+v %v dirs %v", got, err, fs.dirs)
	}
	if _, err := m.BulkMove(ctx, "s", []model.Direction{model.DirectionUp, model.DirectionDown}); err != nil || !reflect.DeepEqual(fs.dirs, []engine.Direction{engine.DirUp, engine.DirDown}) {
		t.Fatalf("BulkMove: %v dirs %v", err, fs.dirs)
	}

	fs.err = service.Errorf(service.CodeSessionTerminal, "game is already WON")
	if _, err := m.Move(ctx, "s", model.DirectionUp); codeOf(t, err) != service.CodeSessionTerminal {
		t.Fatalf("Move err = %v", err)
	}
	if _, err := m.BulkMove(ctx, "s", nil); codeOf(t, err) != service.CodeSessionTerminal {
		t.Fatalf("BulkMove err = %v", err)
	}
}

func TestSimulateResolver(t *testing.T) {
	fs := &fakeService{maps: map[string]*engine.Map{"m": mustMap(t, "m")}}
	q := &queryResolver{&Resolver{Service: fs}}
	ctx := context.Background()
	yes := true

	win := []model.Direction{model.DirectionRight, model.DirectionRight, model.DirectionDown, model.DirectionLeft}
	res, err := q.Simulate(ctx, "m", win, &yes)
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != model.StatusWon || res.Moves != 4 || res.LossReason != nil || res.FinalState.Moves != 4 || len(res.Events) != 4 {
		t.Fatalf("simulate: %+v", res)
	}
	if ev := res.Events[1]; ev.Via != model.ViaStairsUp || ev.Direction != model.DirectionRight || ev.To.Level != 1 || ev.Stepped.Level != 0 || ev.Blocked {
		t.Fatalf("stairs event: %+v", ev)
	}
	if ev := res.Events[2]; ev.Via != model.ViaStairsUp || ev.Status != model.StatusPlaying || *ev.To != (model.Position{Level: 2, X: 2, Y: 1}) || ev.Stepped.Level != 1 {
		t.Fatalf("second stairs event: %+v", ev)
	}
	if ev := res.Events[3]; ev.Via != model.ViaWalk || ev.Status != model.StatusWon || ev.To.Level != 2 {
		t.Fatalf("finish event: %+v", ev)
	}

	// Events are only produced on request; blocked moves are reported as such.
	res, err = q.Simulate(ctx, "m", []model.Direction{model.DirectionUp}, nil)
	if err != nil || res.Events != nil || res.Status != model.StatusReady || res.Moves != 0 {
		t.Fatalf("no events: %+v %v", res, err)
	}
	res, _ = q.Simulate(ctx, "m", []model.Direction{model.DirectionUp}, &yes)
	if len(res.Events) != 1 || !res.Events[0].Blocked || res.Events[0].Via != model.ViaNone {
		t.Fatalf("blocked event: %+v", res.Events)
	}

	if _, err := q.Simulate(ctx, "nope", win, nil); codeOf(t, err) != service.CodeNotFound {
		t.Fatalf("unknown map: %v", err)
	}
}

func TestValidateMapResult(t *testing.T) {
	t.Setenv("ALLOW_UNAUTHENTICATED_ADMIN", "true")
	t.Setenv("ADMIN_API_KEY", "")
	m := &mutationResolver{&Resolver{Service: &fakeService{}}}
	ctx := context.Background()

	ok, err := m.ValidateMap(ctx, model.MapInput{ID: "ok", Name: "n", Levels: testMapConfig("x").Levels})
	if err != nil || !ok.Valid || len(ok.Issues) != 0 || !ok.FinishReachable || ok.ParMoves == nil || *ok.ParMoves != 4 {
		t.Fatalf("valid map: %+v %v", ok, err)
	}
	walled, err := m.ValidateMap(ctx, model.MapInput{ID: "w", Name: "n", Levels: [][]string{{"S..", ".o.", "..F"}}})
	if err != nil || walled.Valid || walled.FinishReachable || walled.ParMoves != nil || len(walled.Issues) != 1 {
		t.Fatalf("unreachable finish: %+v %v", walled, err)
	}
	bad, err := m.ValidateMap(ctx, model.MapInput{ID: "BAD", Name: "n", Levels: [][]string{{"."}}})
	if err != nil || bad.Valid || bad.FinishReachable || len(bad.Issues) == 0 {
		t.Fatalf("invalid map: %+v %v", bad, err)
	}
}

func TestToGQLErrorMapping(t *testing.T) {
	cases := []struct {
		name string
		err  error
		code string
	}{
		{"coded", service.Errorf(service.CodeSessionTerminal, "won"), service.CodeSessionTerminal},
		{"wrapped coded", errors.Join(errors.New("ctx"), service.Errorf(service.CodeNotFound, "x")), service.CodeNotFound},
		{"map validation", &engine.ValidationError{Issues: []string{"bad"}}, service.CodeInvalidArgument},
		{"unknown", errors.New("disk on fire"), CodeInternal},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := codeOf(t, toGQLError(tc.err)); got != tc.code {
				t.Fatalf("code = %q, want %q", got, tc.code)
			}
		})
	}
	if err := toGQLError(errors.New("secret detail")); err.Error() == "secret detail" {
		t.Fatal("internal error message leaked to client")
	}
	if toGQLError(nil) != nil {
		t.Fatal("nil must map to nil")
	}
}

func TestQueryNotFoundIsNull(t *testing.T) {
	q := &queryResolver{&Resolver{Service: &fakeService{}}}
	if s, err := q.Session(context.Background(), "nope"); s != nil || err != nil {
		t.Fatalf("session = %v, %v; want nil,nil", s, err)
	}
	if m, err := q.Map(context.Background(), "nope"); m != nil || err != nil {
		t.Fatalf("map = %v, %v; want nil,nil", m, err)
	}
}

func TestSessionConversion(t *testing.T) {
	m := mustMap(t, "m")
	st := engine.NewState(m)
	ev, err := st.Move(m, engine.DirRight)
	if err != nil {
		t.Fatal(err)
	}
	s := &service.Session{ID: "s", MapID: "m", Map: m, State: st, Seq: 3, LastEvent: &ev}
	got := toSession(s)
	if got.Seq != 3 || got.State.Moves != 1 || got.State.Status != model.StatusPlaying || got.State.LossReason != nil {
		t.Fatalf("state: %+v", got.State)
	}
	if want := []*model.Position{{Level: 0, X: 0, Y: 0}, {Level: 0, X: 1, Y: 0}}; !reflect.DeepEqual(got.State.Path, want) {
		t.Fatalf("path = %v", got.State.Path)
	}
	if got.BestMoves != nil {
		t.Fatal("bestMoves must be null before the first win")
	}
	if got.LastEvent == nil || got.LastEvent.Via != model.ViaWalk || got.LastEvent.Direction != model.DirectionRight || got.LastEvent.Blocked {
		t.Fatalf("lastEvent = %+v", got.LastEvent)
	}
	if upd := toSessionUpdate(s); upd.Seq != 3 || upd.Event == nil || upd.Event.Move != 1 || upd.Session.ID != "s" {
		t.Fatalf("update = %+v", upd)
	}

	best := 4
	s.BestMoves, s.LastEvent = &best, nil
	s.State.Status, s.State.LossReason = engine.StatusLost, engine.LossMoveLimit
	got = toSession(s)
	if got.BestMoves == nil || *got.BestMoves != 4 || got.LastEvent != nil {
		t.Fatalf("bestMoves/lastEvent: %v / %v", got.BestMoves, got.LastEvent)
	}
	if got.State.LossReason == nil || *got.State.LossReason != model.LossReasonMoveLimit {
		t.Fatalf("lossReason = %v", got.State.LossReason)
	}
	if toSessionUpdate(s).Event != nil {
		t.Fatal("update event must be null without a last event")
	}
}

func liveSession(t *testing.T, seq uint64) *service.Session {
	t.Helper()
	m := mustMap(t, "m")
	return &service.Session{ID: "s1", MapID: "m", Map: m, State: engine.NewState(m), Seq: seq}
}

func TestForwardInitialThenUpdatesAndSeqFilter(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	updates := make(chan *service.Session, 4)
	out := forwardSessionUpdates(ctx, liveSession(t, 5), updates, 4)

	recv := func() *model.SessionUpdate {
		t.Helper()
		select {
		case u, ok := <-out:
			if !ok {
				t.Fatal("channel closed early")
			}
			return u
		case <-time.After(2 * time.Second):
			t.Fatal("timeout")
			return nil
		}
	}
	if u := recv(); u.Seq != 5 {
		t.Fatalf("initial seq = %d, want 5", u.Seq)
	}
	updates <- liveSession(t, 5) // duplicate of initial: dropped
	updates <- liveSession(t, 4) // stale: dropped
	updates <- liveSession(t, 6)
	if u := recv(); u.Seq != 6 || u.Session.Seq != 6 {
		t.Fatalf("next seq = %d, want 6", u.Seq)
	}
}

func goroutinesSettle(t *testing.T, base int) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if runtime.NumGoroutine() <= base {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("goroutine leak: %d running, baseline %d", runtime.NumGoroutine(), base)
}

func TestForwardEndsOnCancelWithoutReader(t *testing.T) {
	base := runtime.NumGoroutine()
	ctx, cancel := context.WithCancel(context.Background())
	updates := make(chan *service.Session)
	// Buffer 1 and nobody reading: the initial send fits, the next blocks.
	out := forwardSessionUpdates(ctx, liveSession(t, 1), updates, 1)
	go func() { updates <- liveSession(t, 2) }()
	time.Sleep(50 * time.Millisecond)
	cancel()
	for range out { // drains until close; close proves the goroutine exited
	}
	goroutinesSettle(t, base)
}

func TestSubscriptionResolverEmitsSnapshotAndStopsOnCancel(t *testing.T) {
	hub := websocket.NewHub()
	fs := &fakeService{sessions: map[string]*service.Session{"s1": liveSession(t, 2)}}
	sub := &subscriptionResolver{&Resolver{Service: fs, Hub: hub}}

	base := runtime.NumGoroutine()
	ctx, cancel := context.WithCancel(context.Background())
	ch, err := sub.SessionUpdated(ctx, "s1")
	if err != nil {
		t.Fatal(err)
	}
	select {
	case u := <-ch:
		if u.Seq != 2 || u.Session.ID != "s1" {
			t.Fatalf("initial = %+v", u)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("no initial snapshot")
	}
	next := liveSession(t, 3)
	hub.Broadcast(next)
	select {
	case u := <-ch:
		if u.Seq != 3 {
			t.Fatalf("update seq = %d", u.Seq)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("no hub update forwarded")
	}
	cancel()
	for range ch {
	}
	goroutinesSettle(t, base)

	if _, err := sub.SessionUpdated(context.Background(), "missing"); codeOf(t, err) != service.CodeNotFound {
		t.Fatalf("missing session err = %v", err)
	}
}

// playTo returns a session of testMapConfig after dirs.
func playTo(t *testing.T, dirs ...engine.Direction) *service.Session {
	t.Helper()
	m := mustMap(t, "m")
	st := engine.NewState(m)
	var last *engine.MoveEvent
	for _, d := range dirs {
		ev, err := st.Move(m, d)
		if err != nil {
			t.Fatal(err)
		}
		last = &ev
	}
	return &service.Session{ID: "s1", MapID: "m", Map: m, State: st, Seq: 7, LastEvent: last}
}

// The fog contract: a session exposes the rows of the level the player stands
// on and nothing else about the tower, wherever the conversion is used.
func TestSessionViewIsFogged(t *testing.T) {
	rows := map[int][]string{
		0: {level0Row, "...1", "...."},
		1: {level1Top, level1Row, level1Mrk},
		2: {"....", level2Row, "...."},
	}
	cases := []struct {
		name    string
		dirs    []engine.Direction
		level   int
		portals map[string]model.Position // glyph -> a; twins are always hidden here
	}{
		{"start on level 0", nil, 0, map[string]model.Position{"1": {Level: 0, X: 3, Y: 1}}},
		{"after stairs on level 1", []engine.Direction{engine.DirRight, engine.DirRight}, 1,
			map[string]model.Position{"1": {Level: 1, X: 0, Y: 2}, "2": {Level: 1, X: 2, Y: 2}}},
		{"after second stairs on level 2", []engine.Direction{engine.DirRight, engine.DirRight, engine.DirDown}, 2,
			map[string]model.Position{"2": {Level: 2, X: 3, Y: 1}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := playTo(t, tc.dirs...)
			for name, got := range map[string]*model.Session{"session": toSession(s), "update": toSessionUpdate(s).Session} {
				v := got.View
				if v.Level != tc.level || v.LevelCount != 3 || !reflect.DeepEqual(v.Rows, rows[tc.level]) {
					t.Fatalf("%s view = %+v, want level %d rows %v", name, v, tc.level, rows[tc.level])
				}
				if len(v.Portals) != len(tc.portals) {
					t.Fatalf("%s portals = %+v, want %v", name, v.Portals, tc.portals)
				}
				for _, p := range v.Portals {
					if want, ok := tc.portals[p.Glyph]; !ok || *p.A != want || p.B != nil {
						t.Fatalf("%s portal %s = a %v b %v, want a %v and a hidden twin", name, p.Glyph, p.A, p.B, want)
					}
				}
				// Nothing else of the tower is serialized: only the viewed level and
				// the public level-0 preview may appear; no other level's rows
				// (the finish is on the top level).
				raw, _ := json.Marshal(got)
				for l, other := range rows {
					if l == tc.level || l == 0 {
						continue
					}
					for _, row := range other {
						if !contains(rows[tc.level], row) && !contains(rows[0], row) && strings.Contains(string(raw), `"`+row+`"`) {
							t.Fatalf("%s leaks level %d row %q: %s", name, l, row, raw)
						}
					}
				}
				if got.Map.LevelCount != 3 || !reflect.DeepEqual(got.Map.Preview, rows[0]) {
					t.Fatalf("%s catalog entry = %+v", name, got.Map)
				}
			}
		})
	}
}

func contains(ss []string, s string) bool {
	for _, v := range ss {
		if v == s {
			return true
		}
	}
	return false
}

// A portal whose twin is on the same level reveals the twin; otherwise the
// twin's location stays hidden.
func TestLevelViewRevealsSameLevelTwin(t *testing.T) {
	m, err := engine.NewMap(engine.MapConfig{
		ID: "twins", Name: "t",
		Levels: [][]string{
			{"S1o", ".1.", "..U"},
			{"F..", "...", "..D"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	v := toLevelView(m, 0)
	if len(v.Portals) != 1 || v.Portals[0].B == nil || *v.Portals[0].A != (model.Position{Level: 0, X: 1, Y: 0}) || *v.Portals[0].B != (model.Position{Level: 0, X: 1, Y: 1}) {
		t.Fatalf("same-level portal = %+v", v.Portals)
	}
	if up := toLevelView(m, 1); len(up.Portals) != 0 || up.Level != 1 {
		t.Fatalf("level 1 view = %+v", up)
	}
}
