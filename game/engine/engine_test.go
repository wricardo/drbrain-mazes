package engine

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func mustMap(t *testing.T, cfg MapConfig) *Map {
	t.Helper()
	m, err := NewMap(cfg)
	if err != nil {
		t.Fatal(err)
	}
	return m
}

// tower is a two-level map: S steps right onto the stairs up (U) and lands on
// the paired D at level 1 (1,0), then walks right to F.
var tower = MapConfig{ID: "tower", Name: "t", MaxMoves: 20, Levels: [][]string{
	{
		"SUo",
		"...",
		"...",
	},
	{
		"oDF",
		"...",
		"...",
	},
}}

func dirs(s string) []Direction {
	out := make([]Direction, 0, len(s))
	for _, c := range s {
		out = append(out, Direction(strings.IndexRune("URDL", c)))
	}
	return out
}

func TestStairsUpAndWin(t *testing.T) {
	m := mustMap(t, tower)
	evs, st := Simulate(m, dirs("RRR"), true)
	// S -> U (lifts onto the paired D at level 1 (1,0)) -> F: won on the 2nd move.
	if st.Status != StatusWon || st.Moves != 2 {
		t.Fatalf("status=%s moves=%d, want WON in 2", st.Status, st.Moves)
	}
	if len(evs) != 2 {
		t.Fatalf("directions after WON must be ignored, got %d events", len(evs))
	}
	up := evs[0]
	if up.Via != ViaStairsUp || up.Stepped != (Position{0, 1, 0}) || up.To != (Position{1, 1, 0}) {
		t.Fatalf("unexpected stairs event %+v", up)
	}
	if evs[1].Via != ViaWalk {
		t.Fatalf("arriving on the paired D must not trigger it: %+v", evs[1])
	}
	wantPath := []Position{{0, 0, 0}, {0, 1, 0}, {1, 1, 0}, {1, 2, 0}}
	if len(st.Path) != len(wantPath) {
		t.Fatalf("path %v, want %v", st.Path, wantPath)
	}
	for i := range wantPath {
		if st.Path[i] != wantPath[i] {
			t.Fatalf("path %v, want %v", st.Path, wantPath)
		}
	}
}

// Whoever can climb a staircase can climb back from the same x,y: the tile
// you land on is the paired stairs, and stepping off and on again uses it.
func TestStairsAreTwoWay(t *testing.T) {
	m := mustMap(t, tower)
	s := NewState(m)
	if ev, _ := s.Move(m, DirRight); ev.Via != ViaStairsUp || s.Pos != (Position{1, 1, 0}) {
		t.Fatalf("up: %+v pos=%v", ev, s.Pos)
	}
	if ev, _ := s.Move(m, DirLeft); ev.Via != ViaWalk || s.Pos != (Position{1, 0, 0}) {
		t.Fatalf("step off: %+v pos=%v", ev, s.Pos)
	}
	ev, _ := s.Move(m, DirRight) // back onto the D at level 1 (1,0)
	if ev.Via != ViaStairsDown || ev.Stepped != (Position{1, 1, 0}) || ev.To != (Position{0, 1, 0}) {
		t.Fatalf("down: %+v", ev)
	}
	if ev, _ = s.Move(m, DirRight); ev.Via != ViaWalk || s.Pos != (Position{0, 2, 0}) {
		t.Fatalf("arriving on the paired U must not trigger it: %+v pos=%v", ev, s.Pos)
	}
}

func TestViewHidesOtherLevels(t *testing.T) {
	m := mustMap(t, MapConfig{ID: "v", Name: "v", Levels: [][]string{
		{"S12", "21.", "..."},
		{"ooo", "...", "..F"},
	}})
	v := m.View(0)
	if v.Level != 0 || v.LevelCount != 2 || len(v.Rows) != 3 || v.Rows[0] != "S12" {
		t.Fatalf("view: %+v", v)
	}
	byGlyph := map[string]PortalView{}
	for _, p := range v.Portals {
		byGlyph[p.Glyph] = p
	}
	if len(byGlyph) != 2 {
		t.Fatalf("level 0 holds both portals 1 and 2: %+v", v.Portals)
	}
	for _, g := range []string{"1", "2"} {
		p := byGlyph[g]
		if p.A.Level != 0 || p.B == nil || p.B.Level != 0 {
			t.Fatalf("portal %s: both ends must be on level 0: %+v", g, p)
		}
	}
	if got := m.View(5); len(got.Rows) != 0 || got.Level != 5 {
		t.Fatalf("out of range view: %+v", got)
	}
	if got := m.View(1); len(got.Portals) != 0 {
		t.Fatalf("level 1 has no portals: %+v", got.Portals)
	}
	v.Rows[0] = "xxx"
	if m.Levels[0][0] != "S12" {
		t.Fatal("View must copy rows")
	}
}

func TestViewShowsTwinOnSameLevel(t *testing.T) {
	m := mustMap(t, MapConfig{ID: "v2", Name: "v2", Levels: [][]string{{"S1F", "...", "1.."}}})
	v := m.View(0)
	if len(v.Portals) != 1 || v.Portals[0].B == nil || *v.Portals[0].B != (Position{0, 0, 2}) {
		t.Fatalf("portals: %+v", v.Portals)
	}
}

func TestBlockedIsFreeAndKeepsStatus(t *testing.T) {
	m := mustMap(t, tower)
	s := NewState(m)
	for _, d := range []Direction{DirUp, DirLeft, DirDown} {
		ev, err := s.Move(m, d)
		if err != nil {
			t.Fatal(err)
		}
		if !ev.Blocked || ev.Via != ViaNone || ev.To != m.Start {
			t.Fatalf("%v: want blocked, got %+v", d, ev)
		}
	}
	if s.Moves != 0 || s.Status != StatusReady || len(s.Path) != 1 {
		t.Fatalf("blocked moves must not change state: %+v", s)
	}
	if _, err := s.Move(m, DirRight); err != nil || s.Moves != 1 || s.Status != StatusPlaying {
		t.Fatalf("after a real move: %+v err=%v", s, err)
	}
}

func TestTerminalRejectsMoves(t *testing.T) {
	m := mustMap(t, tower)
	_, st := Simulate(m, dirs("RR"), false)
	if _, err := st.Move(m, DirRight); !errors.Is(err, ErrTerminal) {
		t.Fatalf("err = %v, want ErrTerminal", err)
	}
}

func TestMoveLimitLoses(t *testing.T) {
	cfg := MapConfig{ID: "lim", Name: "l", MaxMoves: 3, Levels: [][]string{{"Soo", "ooF", "..."}}}
	m := mustMap(t, cfg)
	_, st := Simulate(m, dirs("RLRLRL"), false)
	if st.Status != StatusLost || st.LossReason != LossMoveLimit || st.Moves != 3 {
		t.Fatalf("status=%s reason=%s moves=%d", st.Status, st.LossReason, st.Moves)
	}
}

func TestWinOnLastAllowedMoveBeatsLimit(t *testing.T) {
	cfg := MapConfig{ID: "edge", Name: "e", MaxMoves: 2, Levels: [][]string{{"SoF", "...", "..."}}}
	m := mustMap(t, cfg)
	_, st := Simulate(m, dirs("RR"), false)
	if st.Status != StatusWon {
		t.Fatalf("status=%s, want WON", st.Status)
	}
}

func TestPortalTeleportsWithoutRetrigger(t *testing.T) {
	cfg := MapConfig{ID: "pipe", Name: "p", Levels: [][]string{{
		"S1.",
		"...",
		"1oF",
	}}}
	m := mustMap(t, cfg)
	if len(m.Portals) != 1 || m.Portals[0].Glyph != "1" {
		t.Fatalf("portals: %+v", m.Portals)
	}
	s := NewState(m)
	ev, _ := s.Move(m, DirRight) // onto hole 1 at (1,0) -> twin at (0,2)
	if ev.Via != ViaPortal || ev.Stepped != (Position{0, 1, 0}) || ev.To != (Position{0, 0, 2}) {
		t.Fatalf("portal event %+v", ev)
	}
	if s.Pos != (Position{0, 0, 2}) {
		t.Fatalf("pos %v", s.Pos)
	}
	// Walking off the arrival hole must not teleport again; walking back onto it must.
	if ev, _ = s.Move(m, DirRight); ev.Via != ViaWalk {
		t.Fatalf("leaving arrival hole: %+v", ev)
	}
	if ev, _ = s.Move(m, DirLeft); ev.Via != ViaPortal || ev.To != (Position{0, 1, 0}) {
		t.Fatalf("re-entering hole: %+v", ev)
	}
}

func TestPortalAcrossLevelsRejected(t *testing.T) {
	cfg := MapConfig{ID: "x", Name: "x", Levels: [][]string{
		{"S2.", "...", "..."},
		{"2oF", "...", "..."},
	}}
	_, err := NewMap(cfg)
	var ve *ValidationError
	if !errors.As(err, &ve) || !strings.Contains(err.Error(), "same level") {
		t.Fatalf("err = %v, want a same-level ValidationError", err)
	}
}

func TestNewMapIssues(t *testing.T) {
	tests := []struct {
		name string
		mut  func(*MapConfig)
		want string
	}{
		{"bad id", func(c *MapConfig) { c.ID = "Bad Id" }, "id"},
		{"no name", func(c *MapConfig) { c.Name = " " }, "name is required"},
		{"no levels", func(c *MapConfig) { c.Levels = nil }, "levels must have"},
		{"too many levels", func(c *MapConfig) {
			c.Levels = make([][]string, MaxLevels+1)
			for i := range c.Levels {
				c.Levels[i] = tower.Levels[0]
			}
		}, "levels must have"},
		{"max moves", func(c *MapConfig) { c.MaxMoves = MaxMaxMoves + 1 }, "max_moves"},
		{"ragged row", func(c *MapConfig) { c.Levels = [][]string{{"SoF", "..", "..."}} }, "width"},
		{"level size mismatch", func(c *MapConfig) { c.Levels = [][]string{{"SoF", "...", "..."}, {"ooo", "..."}} }, "rows"},
		{"unknown glyph", func(c *MapConfig) { c.Levels = [][]string{{"SoF", ".x.", "..."}} }, "unknown glyph"},
		{"no start", func(c *MapConfig) { c.Levels = [][]string{{"ooF", "...", "..."}} }, "exactly one start"},
		{"two finishes", func(c *MapConfig) { c.Levels = [][]string{{"SFF", "...", "..."}} }, "exactly one finish"},
		{"lonely portal", func(c *MapConfig) { c.Levels = [][]string{{"S1F", "...", "..."}} }, "exactly twice"},
		{"triple portal", func(c *MapConfig) { c.Levels = [][]string{{"S1F", "1.1", "..."}} }, "exactly twice"},
		{"portal across levels", func(c *MapConfig) {
			c.Levels = [][]string{{"S1F", "...", "..."}, {"o1o", "...", "..."}}
		}, "same level"},
		{"stairs off tower top", func(c *MapConfig) { c.Levels = [][]string{{"SUF", "...", "..."}} }, "off the tower"},
		{"up stairs land on void", func(c *MapConfig) {
			c.Levels = [][]string{{"SUo", "...", "..."}, {"o.F", "...", "..."}}
		}, "matching D"},
		{"up stairs land on plain floor", func(c *MapConfig) {
			c.Levels = [][]string{{"SUo", "...", "..."}, {"ooF", "...", "..."}}
		}, "matching D"},
		{"up stairs land on up stairs", func(c *MapConfig) {
			c.Levels = [][]string{{"SUo", "...", "..."}, {"oUo", "...", "..."}, {"ooF", "...", "..."}}
		}, "matching D"},
		{"down stairs without an up partner", func(c *MapConfig) {
			c.Levels = [][]string{{"Soo", "...", "..."}, {"oDF", "...", "..."}}
		}, "matching U"},
		{"start above level 0", func(c *MapConfig) {
			c.Levels = [][]string{{"ooo", "...", "..."}, {"SoF", "...", "..."}}
		}, "start (S) must be on level 0"},
		{"finish below top", func(c *MapConfig) {
			c.Levels = [][]string{{"SoF", "...", "..."}, {"ooo", "...", "..."}}
		}, "finish (F) must be on the top level"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cfg := tower
			cfg.Levels = append([][]string(nil), tower.Levels...)
			tc.mut(&cfg)
			_, err := NewMap(cfg)
			var ve *ValidationError
			if !errors.As(err, &ve) {
				t.Fatalf("err = %v, want *ValidationError", err)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("err %q does not mention %q", err, tc.want)
			}
		})
	}
}

func TestDefaultsAndConfigRoundTrip(t *testing.T) {
	cfg := MapConfig{ID: "rt", Name: "rt", Levels: [][]string{{"SoF", "...", "..."}}}
	m := mustMap(t, cfg)
	if m.MaxMoves != DefaultMaxMoves || m.Difficulty != "easy" {
		t.Fatalf("defaults: max=%d diff=%q", m.MaxMoves, m.Difficulty)
	}
	back := m.Config()
	m2 := mustMap(t, back)
	if m2.Par != m.Par || m2.MaxMoves != m.MaxMoves || m2.Finish != m.Finish {
		t.Fatalf("round trip changed the map: %+v vs %+v", m, m2)
	}
	c := m.Clone()
	c.Levels[0][0] = "xxx"
	if m.Levels[0][0] != "SoF" {
		t.Fatal("Clone must not share level rows with the original")
	}
}

func TestShortestAndUnreachable(t *testing.T) {
	m := mustMap(t, tower)
	path, ok := m.Shortest()
	if !ok || m.Par != 2 || len(path) != 2 {
		t.Fatalf("par=%d path=%v ok=%v", m.Par, path, ok)
	}
	cfg := MapConfig{ID: "u", Name: "u", Levels: [][]string{{"So.", "...", "..F"}}}
	m = mustMap(t, cfg)
	if _, ok := m.Shortest(); ok || m.Par != -1 {
		t.Fatalf("par=%d ok=%v, want unreachable", m.Par, ok)
	}
	if got := len(m.Reachable()); got != 2 {
		t.Fatalf("reachable = %d, want 2", got)
	}
}

func TestForcedDescentsAndMinLevelChanges(t *testing.T) {
	// Up to level 1 at x1 is a pocket (x1..x3); the way on is down the D at
	// x3, along level 0 to the U at x5, up to level 1 x5, then up again at x6.
	m := mustMap(t, MapConfig{ID: "dip", Name: "d", Levels: [][]string{
		{"SU.UoU.", ".......", "......."},
		{".DoD.DU", ".......", "......."},
		{".....FD", ".......", "......."},
	}})
	if m.Par != 7 || m.ForcedDescents != 1 || m.MinLevelChanges() != 4 {
		t.Fatalf("par=%d descents=%d changes=%d, want 7/1/4", m.Par, m.ForcedDescents, m.MinLevelChanges())
	}
	if m := mustMap(t, tower); m.ForcedDescents != 0 || m.MinLevelChanges() != m.LevelCount()-1 {
		t.Fatalf("straight climb: descents=%d changes=%d", m.ForcedDescents, m.MinLevelChanges())
	}
	m = mustMap(t, MapConfig{ID: "u", Name: "u", Levels: [][]string{{"So.", "...", "..F"}}})
	if m.ForcedDescents != -1 || m.MinLevelChanges() != -1 {
		t.Fatalf("unreachable: descents=%d changes=%d", m.ForcedDescents, m.MinLevelChanges())
	}
}

func TestShortestPathReplays(t *testing.T) {
	// The BFS route must be a legal winning move list (stairs + portals included).
	m := mustMap(t, MapConfig{ID: "r", Name: "r", Levels: [][]string{
		{"S1oU", "....", "..oo", "o..1"},
		{"oooD", "oooo", "ooFo", "oooo"},
	}})
	path, ok := m.Shortest()
	if !ok {
		t.Fatal("unreachable")
	}
	_, st := Simulate(m, path, false)
	if st.Status != StatusWon || st.Moves != len(path) {
		t.Fatalf("replay: status=%s moves=%d path=%d", st.Status, st.Moves, len(path))
	}
}

// Golden shortest-route lengths of the shipped maps (change deliberately only).
func TestShippedMapsGolden(t *testing.T) {
	want := map[string]int{
		"first_steps":  9,
		"ladder":       17,
		"portal_twins": 11,
		"pipe_dream":   9,
		"two_floors":   18,
		"lost_garden":  90,
		"wormholes":    78,
		"four_floors":  72,
		"portal_climb": 80,
		"tower_maze":   62,
		"miners_shaft": 71,
		"deep_tower":   93,
		"grand_tower":  112,
		"switchback":   34,
		"false_summit": 66,
		"portal_locks": 46,
	}
	// Fewest down stairs a winning route needs; maps not listed climb straight (0).
	wantDescents := map[string]int{
		"switchback":   2,
		"false_summit": 1,
		"portal_locks": 1,
	}
	files, err := filepath.Glob("../../maps/*.json")
	if err != nil || len(files) != len(want) {
		t.Fatalf("maps: %v (err %v), want %d files", files, err, len(want))
	}
	for _, f := range files {
		id := strings.TrimSuffix(filepath.Base(f), ".json")
		data, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		var cfg MapConfig
		if err := json.Unmarshal(data, &cfg); err != nil {
			t.Fatal(err)
		}
		m := mustMap(t, cfg)
		if m.ID != id {
			t.Errorf("%s: id %q does not match file name", f, m.ID)
		}
		if m.Par != want[id] {
			t.Errorf("%s: par = %d, want %d", id, m.Par, want[id])
		}
		if m.ForcedDescents != wantDescents[id] {
			t.Errorf("%s: forced descents = %d, want %d", id, m.ForcedDescents, wantDescents[id])
		}
		if m.MaxMoves < m.Par {
			t.Errorf("%s: max_moves %d below par %d", id, m.MaxMoves, m.Par)
		}
	}
}

func TestDirectionParse(t *testing.T) {
	for i, n := range []string{"UP", "RIGHT", "DOWN", "LEFT"} {
		d, ok := ParseDirection(n)
		if !ok || d != Direction(i) || d.String() != n {
			t.Fatalf("%s -> %v ok=%v", n, d, ok)
		}
	}
	if _, ok := ParseDirection("up"); ok {
		t.Fatal("lowercase must not parse")
	}
}
