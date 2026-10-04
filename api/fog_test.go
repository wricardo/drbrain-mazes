package api

import (
	"encoding/json"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/wricardo/drbrain-mazes/game/engine"
	"github.com/wricardo/drbrain-mazes/game/service"
	"github.com/wricardo/drbrain-mazes/game/session"
	"github.com/wricardo/drbrain-mazes/transport/websocket"
)

// Rows that exist on exactly one level of fogCfg. Level 0 rows are public
// (Map.preview); every row below must never reach a player standing on level 0.
const (
	hiddenLevel1Top = "..D."
	hiddenLevel1Row = "..U."
	hiddenLevel1Mrk = "1.2o"
	hiddenFinishRow = ".FD2"
)

// fogCfg: 3 levels. Level 0 (3,1) holds portal 1 whose twin is on level 1;
// the finish is on the top level.
var fogCfg = engine.MapConfig{
	ID: "fog", Name: "Fog",
	Levels: [][]string{
		{"SoU.", "...1", "...."},
		{hiddenLevel1Top, hiddenLevel1Row, hiddenLevel1Mrk},
		{"....", hiddenFinishRow, "...."},
	},
}

// mapStore is a minimal service.MapStore over fixed maps.
type mapStore struct{ maps map[string]*engine.Map }

func (s mapStore) List() []*engine.Map {
	out := make([]*engine.Map, 0, len(s.maps))
	for _, m := range s.maps {
		out = append(out, m.Clone())
	}
	return out
}

func (s mapStore) Get(id string) (*engine.Map, error) {
	if m, ok := s.maps[id]; ok {
		return m.Clone(), nil
	}
	return nil, service.Errorf(service.CodeNotFound, "map %q not found", id)
}

func (s mapStore) Save(engine.MapConfig) (*engine.Map, error)   { panic("unused") }
func (s mapStore) Create(engine.MapConfig) (*engine.Map, error) { panic("unused") }
func (s mapStore) Delete(string) error                          { panic("unused") }

// newFogServer wires the real service, session manager and hub.
func newFogServer(t *testing.T) *httptest.Server {
	t.Helper()
	m, err := engine.NewMap(fogCfg)
	if err != nil {
		t.Fatal(err)
	}
	hub := websocket.NewHub()
	svc := service.New(mapStore{maps: map[string]*engine.Map{"fog": m}}, session.NewManager(nil, 10), hub, service.Options{})
	t.Cleanup(svc.Shutdown)
	srv, err := NewServer(svc, hub, Options{Introspection: true, LLMSTemplate: "x", UIDirs: []string{filepath.Join(t.TempDir(), "missing")}})
	if err != nil {
		t.Fatal(err)
	}
	ts := httptest.NewServer(srv)
	t.Cleanup(ts.Close)
	return ts
}

// Everything a player-facing selection can ask for about a session.
const fogSession = `id map { id levelCount preview parMoves } ` +
	`view { level levelCount rows portals { glyph a { level x y } b { level x y } } } ` +
	`state { pos { level x y } moves status path { level x y } } lastEvent { from { level x y } stepped { level x y } to { level x y } } seq`

func assertNoHiddenRows(t *testing.T, what string, raw []byte) {
	t.Helper()
	for _, leak := range []string{hiddenLevel1Top, hiddenLevel1Row, hiddenLevel1Mrk, hiddenFinishRow} {
		if strings.Contains(string(raw), leak) {
			t.Fatalf("%s leaks hidden row %q: %s", what, leak, raw)
		}
	}
}

func dataOf(t *testing.T, out map[string]any, key string) any {
	t.Helper()
	if out["errors"] != nil {
		t.Fatalf("%s: errors %v", key, out["errors"])
	}
	return out["data"].(map[string]any)[key]
}

// A session on level 0 of a multi-level map exposes only level 0 through
// every player-facing path: queries, mutations and the subscription.
func TestSessionsNeverLeakOtherLevels(t *testing.T) {
	ts := newFogServer(t)

	// Subscribe first so every mutation below also flows through the hub.
	c := dialWS(t, ts)
	defer c.Close()
	_ = c.SetReadDeadline(time.Now().Add(10 * time.Second))
	send := func(v any) {
		t.Helper()
		if err := c.WriteJSON(v); err != nil {
			t.Fatal(err)
		}
	}
	send(map[string]any{"type": "connection_init"})

	_, out := gql(t, ts.URL, `mutation { createSession(mapID: "fog") { `+fogSession+` } }`, nil)
	created := dataOf(t, out, "createSession").(map[string]any)
	sid := created["id"].(string)
	raw, _ := json.Marshal(out)
	assertNoHiddenRows(t, "createSession", raw)
	view := created["view"].(map[string]any)
	if view["level"] != float64(0) || view["levelCount"] != float64(3) || len(view["rows"].([]any)) != 3 || view["rows"].([]any)[0] != "SoU." {
		t.Fatalf("view at start = %v", view)
	}
	// Portal 1 is listed with its level-0 end only; the twin on level 1 is hidden.
	portals := view["portals"].([]any)
	if len(portals) != 1 || portals[0].(map[string]any)["b"] != nil {
		t.Fatalf("portals = %v, want one hole with a hidden twin", portals)
	}

	var mu sync.Mutex
	var frames []string
	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			_, msg, err := c.ReadMessage()
			if err != nil {
				return
			}
			var m map[string]any
			_ = json.Unmarshal(msg, &m)
			switch m["type"] {
			case "connection_ack":
				_ = c.WriteJSON(map[string]any{"id": "1", "type": "subscribe", "payload": map[string]any{
					"query": `subscription { sessionUpdated(sessionID: "` + sid + `") { seq event { from { level x y } to { level x y } } session { ` + fogSession + ` } } }`,
				}})
			case "next":
				mu.Lock()
				frames = append(frames, string(msg))
				mu.Unlock()
			}
		}
	}()
	waitFrames := func(n int) {
		t.Helper()
		deadline := time.Now().Add(5 * time.Second)
		for time.Now().Before(deadline) {
			mu.Lock()
			got := len(frames)
			mu.Unlock()
			if got >= n {
				return
			}
			time.Sleep(10 * time.Millisecond)
		}
		t.Fatalf("subscription delivered fewer than %d frames", n)
	}
	waitFrames(1) // initial snapshot

	q := func(query string) []byte {
		t.Helper()
		_, o := gql(t, ts.URL, query, nil)
		if o["errors"] != nil {
			t.Fatalf("%s: %v", query, o["errors"])
		}
		b, _ := json.Marshal(o)
		assertNoHiddenRows(t, query, b)
		return b
	}
	q(`query { session(id: "` + sid + `") { ` + fogSession + ` } }`)
	q(`query { sessions { ` + fogSession + ` } }`)
	q(`query { maps { id levelCount preview parMoves } map(id: "fog") { id levelCount preview } }`)
	q(`mutation { move(sessionID: "` + sid + `", direction: UP) { ` + fogSession + ` } }`)    // blocked
	q(`mutation { move(sessionID: "` + sid + `", direction: RIGHT) { ` + fogSession + ` } }`) // walk
	q(`mutation { renameSession(id: "` + sid + `", displayName: "x") { ` + fogSession + ` } }`)
	q(`mutation { reset(sessionID: "` + sid + `") { ` + fogSession + ` } }`)
	q(`mutation { bulkMove(sessionID: "` + sid + `", directions: [RIGHT]) { ` + fogSession + ` } }`)
	waitFrames(6) // initial + blocked + walk + rename + reset + bulk
	_ = c.Close()
	<-done
	mu.Lock()
	for i, f := range frames {
		assertNoHiddenRows(t, "subscription frame "+string(rune('0'+i)), []byte(f))
	}
	mu.Unlock()

	// Standing on level 1 (after the stairs) the view switches to level 1's
	// rows, and then level 0's rows are only visible through the public preview.
	_, o := gql(t, ts.URL, `mutation { bulkMove(sessionID: "`+sid+`", directions: [RIGHT]) { view { level rows } } }`, nil)
	view = dataOf(t, o, "bulkMove").(map[string]any)["view"].(map[string]any)
	if view["level"] != float64(1) || view["rows"].([]any)[1] != hiddenLevel1Row {
		t.Fatalf("view after the stairs = %v", view)
	}
	raw, _ = json.Marshal(o)
	if strings.Contains(string(raw), hiddenFinishRow) {
		t.Fatalf("level 1 view leaks the finish level: %s", raw)
	}
}

// The schema offers players no way to ask for the whole tower.
func TestPlayerTypesHaveNoFullLayoutFields(t *testing.T) {
	ts := newFogServer(t)
	for _, q := range []string{
		`{ maps { levels } }`,
		`{ maps { start { x } } }`,
		`{ map(id: "fog") { finish { level } } }`,
		`{ map(id: "fog") { portals { glyph } } }`,
		`{ sessions { map { levels } } }`,
		`{ sessions { map { finish { level } } } }`,
	} {
		status, out := gql(t, ts.URL, q, nil)
		if status < 400 || firstErrCode(out) != "GRAPHQL_VALIDATION_FAILED" {
			t.Errorf("%s: status %d out %v, want a validation error", q, status, out)
		}
	}
}

func TestAdminMapNeedsKey(t *testing.T) {
	t.Setenv("ADMIN_API_KEY", "secret")
	t.Setenv("ALLOW_UNAUTHENTICATED_ADMIN", "")
	ts := newFogServer(t)
	const q = `{ adminMap(id: "fog") { id levelCount levels start { level x y } finish { level x y } portals { glyph a { level x y } b { level x y } } } }`

	for name, headers := range map[string]map[string]string{
		"no key":    nil,
		"wrong key": {"X-Admin-Key": "nope"},
	} {
		_, out := gql(t, ts.URL, q, headers)
		raw, _ := json.Marshal(out)
		if got := firstErrCode(out); got != service.CodeForbidden {
			t.Fatalf("%s: code %q out %s", name, got, raw)
		}
		assertNoHiddenRows(t, name, raw)
	}
	_, out := gql(t, ts.URL, q, map[string]string{"X-Admin-Key": "secret"})
	m := dataOf(t, out, "adminMap").(map[string]any)
	levels := m["levels"].([]any)
	if len(levels) != 3 || levels[2].([]any)[1] != hiddenFinishRow || m["finish"].(map[string]any)["level"] != float64(2) {
		t.Fatalf("admin map = %v", m)
	}
	if p := m["portals"].([]any)[0].(map[string]any); p["b"] == nil {
		t.Fatalf("admin portals must include both ends: %v", p)
	}
}
