package validate

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/wricardo/drbrain-mazes/game/engine"
)

const (
	mapsDir = "../maps"
	solDir  = "../solutions"
)

// towerCfg: paired stairs, a portal pair across levels and the finish on
// level 1. RIGHT, RIGHT, RIGHT, UP, DOWN wins: par 5.
var towerCfg = engine.MapConfig{
	ID: "tower", Name: "Tower", MaxMoves: 6,
	Levels: [][]string{
		{"SoU.", "...U", "...1"},
		{"..D1", "...D", "...F"},
	},
}

var (
	right = engine.DirRight
	up    = engine.DirUp
	down  = engine.DirDown
	left  = engine.DirLeft
)

// requireSolutions skips tests that need the reference solutions, which are
// kept out of the public repo (solutions/ is gitignored).
func requireSolutions(t *testing.T) {
	t.Helper()
	if _, err := os.Stat(solDir); err != nil {
		t.Skipf("reference solutions not present in %s", solDir)
	}
}

func mustMap(t *testing.T, cfg engine.MapConfig) *engine.Map {
	t.Helper()
	m, err := engine.NewMap(cfg)
	if err != nil {
		t.Fatal(err)
	}
	return m
}

func loadMap(t *testing.T, id string) *engine.Map {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(mapsDir, id+".json"))
	if err != nil {
		t.Fatal(err)
	}
	var cfg engine.MapConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatal(err)
	}
	return mustMap(t, cfg)
}

func TestEveryMapSolutionWinsWithGoldenMoves(t *testing.T) {
	requireSolutions(t)
	golden := map[string]int{
		"first_steps": 9, "ladder": 17, "portal_twins": 11, "pipe_dream": 9, "two_floors": 18, "lost_garden": 90,
		"wormholes": 78, "four_floors": 72, "portal_climb": 80, "tower_maze": 62, "miners_shaft": 71,
		"deep_tower": 94, "grand_tower": 114,
	}
	files, _ := filepath.Glob(filepath.Join(mapsDir, "*.json"))
	if len(files) != len(golden) {
		t.Fatalf("found %d maps, golden table has %d: add the new map and its solution move count", len(files), len(golden))
	}
	for id, moves := range golden {
		t.Run(id, func(t *testing.T) {
			m := loadMap(t, id)
			if res := Check(m.Config()); !res.Valid {
				t.Fatalf("Check: %v", res.Issues)
			}
			dirs, err := LoadSolution(solDir, id)
			if err != nil {
				t.Fatal(err)
			}
			if err := CheckSolution(m, dirs); err != nil {
				t.Fatal(err)
			}
			if _, st := engine.Simulate(m, dirs, false); st.Moves != moves || st.Moves < m.Par {
				t.Fatalf("moves=%d want %d (par %d)", st.Moves, moves, m.Par)
			}
		})
	}
}

func TestCheckValidMap(t *testing.T) {
	res := Check(towerCfg)
	if !res.Valid || len(res.Issues) != 0 || !res.FinishReachable || res.ParMoves != 5 {
		t.Fatalf("tower: %+v", res)
	}
	if res.Issues == nil {
		t.Fatal("Issues must be an empty list, not nil (it is serialized to clients)")
	}
}

func TestCheckStructuralIssues(t *testing.T) {
	res := Check(engine.MapConfig{ID: "x", Name: "x", Levels: [][]string{{"...", "...", "..."}}})
	if res.Valid || len(res.Issues) == 0 || res.FinishReachable || res.ParMoves != -1 {
		t.Fatalf("want invalid with issues, got %+v", res)
	}
	// Dimension errors abort early and still report.
	res = Check(engine.MapConfig{ID: "BAD ID", Name: "", Levels: nil})
	if res.Valid || len(res.Issues) < 2 {
		t.Fatalf("want several issues, got %+v", res)
	}
}

func TestCheckStairsNeedPairs(t *testing.T) {
	cases := map[string][][]string{
		"U landing on void":       {{"SoU", "...", "..."}, {"...", "..F", "..."}},
		"U landing on a stone":    {{"SoU", "...", "..."}, {"..o", "..F", "..."}},
		"U landing on the F":      {{"SoU", "...", "..."}, {"..F", "...", "..."}},
		"D without a U below":     {{"So.", "...", "..."}, {"..D", "..F", "..."}},
		"U leading off the tower": {{"SoU", "..F", "..."}},
		"D leading off the tower": {{"SoD", "..F", "..."}},
	}
	for name, levels := range cases {
		cfg := towerCfg
		cfg.Levels = levels
		res := Check(cfg)
		if res.Valid {
			t.Errorf("%s accepted: %+v", name, res)
		}
	}
	cfg := towerCfg
	cfg.Levels = [][]string{{"SUo", "...", "..."}, {"oDF", "...", "..."}}
	if res := Check(cfg); !res.Valid || res.ParMoves != 2 {
		t.Fatalf("paired stairs: %+v", res)
	}
}

func TestCheckUnreachableFinish(t *testing.T) {
	res := Check(engine.MapConfig{
		ID: "walled", Name: "w",
		Levels: [][]string{{
			"Soo.F",
			"ooo..",
			"ooo..",
		}},
	})
	if res.Valid || res.FinishReachable || res.ParMoves != -1 || len(res.Issues) != 1 || !strings.Contains(res.Issues[0], "unreachable") {
		t.Fatalf("want unreachable finish, got %+v", res)
	}
}

// A finish only reachable through a portal or stairs is reachable.
func TestCheckReachableOnlyViaPortal(t *testing.T) {
	res := Check(engine.MapConfig{
		ID: "hop", Name: "h",
		Levels: [][]string{{
			"So1..",
			"...F.",
			"..1oo",
		}},
	})
	// S -> o -> hole 1 teleports to (2,2), then RIGHT and UP reach F.
	if !res.Valid || res.ParMoves != 4 {
		t.Fatalf("portal route: %+v", res)
	}
}

func TestCheckSolution(t *testing.T) {
	m := mustMap(t, towerCfg)
	win := []engine.Direction{right, right, right, up, down}

	if err := CheckSolution(m, win); err != nil {
		t.Fatalf("winning solution rejected: %v", err)
	}
	// Blocked attempts are free and directions after the win are ignored.
	if err := CheckSolution(m, []engine.Direction{up, down, right, right, right, up, down, left, left}); err != nil {
		t.Fatalf("solution with blocked and trailing moves rejected: %v", err)
	}
	for name, dirs := range map[string][]engine.Direction{
		"empty":      nil,
		"incomplete": win[:4],
		"wrong way":  {down, down, right},
	} {
		if err := CheckSolution(m, dirs); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
	// Winning only after the move limit is a loss.
	slow := []engine.Direction{right, left, right, left, right, left, right, right, right, up, down}
	err := CheckSolution(m, slow)
	if err == nil || !strings.Contains(err.Error(), "LOST") {
		t.Fatalf("over the move limit: %v", err)
	}
}

func writeSolution(t *testing.T, dir, id, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, id+".json"), []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestLoadSolution(t *testing.T) {
	dir := t.TempDir()
	writeSolution(t, dir, "ok", `{"moves":["UP","RIGHT","DOWN","LEFT"]}`)
	got, err := LoadSolution(dir, "ok")
	if err != nil {
		t.Fatal(err)
	}
	if want := []engine.Direction{up, right, down, left}; !reflect.DeepEqual(got, want) {
		t.Fatalf("moves = %v, want %v", got, want)
	}

	writeSolution(t, dir, "empty", `{"moves":[]}`)
	if got, err := LoadSolution(dir, "empty"); err != nil || len(got) != 0 {
		t.Fatalf("empty list: %v %v", got, err)
	}

	bad := map[string]string{
		"unknown_field": `{"moves":["UP"],"note":"x"}`,
		"trailing":      `{"moves":["UP"]} {"moves":[]}`,
		"lowercase":     `{"moves":["up"]}`,
		"bogus_dir":     `{"moves":["NORTH"]}`,
		"not_json":      `moves: UP`,
		"wrong_type":    `{"moves":[1,2]}`,
	}
	for id, content := range bad {
		writeSolution(t, dir, id, content)
		if _, err := LoadSolution(dir, id); err == nil {
			t.Errorf("%s: accepted %s", id, content)
		}
	}
	if _, err := LoadSolution(dir, "missing"); err == nil {
		t.Error("missing file accepted")
	}
	for _, id := range []string{"../ok", "a/b", "", "UPPER", "ok.json"} {
		if _, err := LoadSolution(dir, id); err == nil || !strings.Contains(err.Error(), "invalid map id") {
			t.Errorf("id %q: %v", id, err)
		}
	}
}
