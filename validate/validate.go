// Package validate checks maps (structure, finish reachability) and their
// reference solutions.
package validate

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/wricardo/drbrain-mazes/game/engine"
)

// Result is the structured outcome of Check.
type Result struct {
	Valid  bool     `json:"valid"`
	Issues []string `json:"issues"`
	// FinishReachable is false when the map is structurally invalid or the
	// finish cannot be reached from the start.
	FinishReachable bool `json:"finishReachable"`
	// ParMoves is the shortest winning route length, or -1 when the finish is
	// not reachable (same convention as engine.Map.Par).
	ParMoves int `json:"parMoves"`
	// MinLevelChanges is the fewest stairs any winning route takes; -1 when
	// the finish is not reachable.
	MinLevelChanges int `json:"minLevelChanges"`
	// ForcedDescents is the fewest down stairs any winning route takes; -1
	// when the finish is not reachable.
	ForcedDescents int `json:"forcedDescents"`
}

// Check validates cfg structurally (engine.NewMap) and verifies the finish
// can be reached from the start.
func Check(cfg engine.MapConfig) Result {
	res := Result{Issues: []string{}, ParMoves: -1, MinLevelChanges: -1, ForcedDescents: -1}
	m, err := engine.NewMap(cfg)
	if err != nil {
		var ve *engine.ValidationError
		if errors.As(err, &ve) {
			res.Issues = append(res.Issues, ve.Issues...)
		} else {
			res.Issues = append(res.Issues, err.Error())
		}
		return res
	}
	res.ParMoves = m.Par
	res.MinLevelChanges = m.MinLevelChanges()
	res.ForcedDescents = m.ForcedDescents
	res.FinishReachable = m.Par >= 0
	if !res.FinishReachable {
		res.Issues = append(res.Issues, fmt.Sprintf("finish at level %d (%d,%d) is unreachable from the start", m.Finish.Level, m.Finish.X, m.Finish.Y))
	}
	res.Valid = len(res.Issues) == 0
	return res
}

// CheckSolution verifies that playing dirs on m from the start ends WON
// (within the map's move limit). Directions after the win are ignored.
func CheckSolution(m *engine.Map, dirs []engine.Direction) error {
	_, st := engine.Simulate(m, dirs, false)
	if st.Status != engine.StatusWon {
		return fmt.Errorf("solution does not win: status=%s loss=%s moves=%d", st.Status, st.LossReason, st.Moves)
	}
	return nil
}

// solutionFile is the JSON form of solutions/<id>.json.
type solutionFile struct {
	Moves []string `json:"moves"`
}

// LoadSolution reads <dir>/<mapID>.json ({"moves":["UP","RIGHT",...]}).
func LoadSolution(dir, mapID string) ([]engine.Direction, error) {
	if !engine.ValidID(mapID) {
		return nil, fmt.Errorf("invalid map id %q", mapID)
	}
	path := filepath.Join(dir, mapID+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("solution for map %q: %w", mapID, err)
	}
	var f solutionFile
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&f); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	if _, err := dec.Token(); err != io.EOF {
		return nil, fmt.Errorf("%s: unexpected data after JSON value", path)
	}
	dirs := make([]engine.Direction, len(f.Moves))
	for i, name := range f.Moves {
		d, ok := engine.ParseDirection(name)
		if !ok {
			return nil, fmt.Errorf("%s: move %d: unknown direction %q", path, i, name)
		}
		dirs[i] = d
	}
	return dirs, nil
}
