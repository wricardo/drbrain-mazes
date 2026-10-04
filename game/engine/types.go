package engine

import (
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"
)

// Direction is a move direction on a level: 0=up, 1=right, 2=down, 3=left.
type Direction int

const (
	DirUp Direction = iota
	DirRight
	DirDown
	DirLeft
)

var dirNames = [4]string{"UP", "RIGHT", "DOWN", "LEFT"}

var dirDelta = [4][2]int{{0, -1}, {1, 0}, {0, 1}, {-1, 0}}

// String returns the canonical name (UP, RIGHT, DOWN, LEFT).
func (d Direction) String() string {
	if d < 0 || int(d) >= len(dirNames) {
		return fmt.Sprintf("Direction(%d)", int(d))
	}
	return dirNames[d]
}

// ParseDirection parses a canonical direction name.
func ParseDirection(s string) (Direction, bool) {
	for i, n := range dirNames {
		if s == n {
			return Direction(i), true
		}
	}
	return 0, false
}

// Position is a coordinate in the tower.
type Position struct {
	Level int `json:"level"`
	X     int `json:"x"`
	Y     int `json:"y"`
}

// Tile is a layout glyph.
type Tile byte

// Layout glyphs.
const (
	TileVoid   Tile = '.' // not walkable
	TileStone  Tile = 'o' // plain walkable floor
	TileStart  Tile = 'S' // walkable; exactly one per map
	TileFinish Tile = 'F' // walkable; exactly one per map; reaching it wins
	TileUp     Tile = 'U' // stepping on it lifts you onto the paired D at the same x,y one level up
	TileDown   Tile = 'D' // stepping on it drops you onto the paired U at the same x,y one level down
	// '1'..'9' are portal holes: each digit appears exactly twice per map and
	// stepping on one teleports you to the other.
)

// IsPortal reports whether t is a portal hole glyph.
func (t Tile) IsPortal() bool { return t >= '1' && t <= '9' }

// Walkable reports whether a move onto t is allowed.
func (t Tile) Walkable() bool { return t != TileVoid }

// valid reports whether t is a known glyph.
func (t Tile) valid() bool {
	switch t {
	case TileVoid, TileStone, TileStart, TileFinish, TileUp, TileDown:
		return true
	}
	return t.IsPortal()
}

// Status is the game lifecycle status.
type Status string

const (
	StatusReady   Status = "READY"   // no successful move yet
	StatusPlaying Status = "PLAYING" // at least one move made
	StatusWon     Status = "WON"
	StatusLost    Status = "LOST"
)

// Terminal reports whether the status is WON or LOST.
func (s Status) Terminal() bool { return s == StatusWon || s == StatusLost }

// LossReason explains a LOST status.
type LossReason string

const (
	LossNone      LossReason = ""
	LossMoveLimit LossReason = "MOVE_LIMIT"
)

// Via describes how a move changed the position.
type Via string

const (
	ViaNone       Via = "NONE" // blocked: nothing happened
	ViaWalk       Via = "WALK"
	ViaStairsUp   Via = "STAIRS_UP"
	ViaStairsDown Via = "STAIRS_DOWN"
	ViaPortal     Via = "PORTAL"
)

// Limits and defaults.
const (
	MinSide         = 3
	MaxSide         = 20
	MaxLevels       = 12
	MaxMaxMoves     = 10000
	DefaultMaxMoves = 500
)

var idRe = regexp.MustCompile(`^[a-z0-9_-]{1,64}$`)

// ValidID reports whether id is a legal map id (also a safe filename).
func ValidID(id string) bool { return idRe.MatchString(id) }

// MapConfig is the JSON representation of a map (maps/*.json).
type MapConfig struct {
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	Description string     `json:"description,omitempty"`
	Difficulty  string     `json:"difficulty,omitempty"`
	Levels      [][]string `json:"levels"` // index 0 = bottom level; each level is rows of glyphs
	MaxMoves    int        `json:"max_moves,omitempty"`
}

// Portal is a pair of linked holes (same glyph digit).
type Portal struct {
	Glyph string   `json:"glyph"`
	A     Position `json:"a"`
	B     Position `json:"b"`
}

// Map is a parsed, validated map.
type Map struct {
	ID            string
	Name          string
	Description   string
	Difficulty    string
	Levels        [][]string
	Width, Height int
	Start, Finish Position
	Portals       []Portal
	MaxMoves      int
	// Par is the length of the shortest winning move sequence, or -1 if the
	// finish cannot be reached from the start.
	Par int

	tiles   []Tile                // level*Width*Height + y*Width + x
	partner map[Position]Position // portal hole -> its twin
}

// LevelCount returns the number of levels.
func (m *Map) LevelCount() int { return len(m.Levels) }

// Config returns the JSON form of the map (all fields explicit).
func (m *Map) Config() MapConfig {
	lv := make([][]string, len(m.Levels))
	for i, rows := range m.Levels {
		lv[i] = append([]string(nil), rows...)
	}
	return MapConfig{
		ID: m.ID, Name: m.Name, Description: m.Description, Difficulty: m.Difficulty,
		Levels: lv, MaxMoves: m.MaxMoves,
	}
}

// Clone deep-copies the map.
func (m *Map) Clone() *Map {
	m2, err := NewMap(m.Config())
	if err != nil {
		// A *Map can only exist if NewMap succeeded; cannot fail.
		panic(err)
	}
	return m2
}

// InBounds reports whether p lies inside the tower.
func (m *Map) InBounds(p Position) bool {
	return p.Level >= 0 && p.Level < len(m.Levels) && p.X >= 0 && p.X < m.Width && p.Y >= 0 && p.Y < m.Height
}

// TileAt returns the glyph at p; out of bounds is TileVoid.
func (m *Map) TileAt(p Position) Tile {
	if !m.InBounds(p) {
		return TileVoid
	}
	return m.tiles[(p.Level*m.Height+p.Y)*m.Width+p.X]
}

// Partner returns the twin of portal hole p.
func (m *Map) Partner(p Position) (Position, bool) {
	q, ok := m.partner[p]
	return q, ok
}

// NewMap validates cfg (applying defaults) and builds a Map. All structural
// problems are returned together as a *ValidationError. Reachability of the
// finish is not an error here (see Map.Par and the validate package).
func NewMap(cfg MapConfig) (*Map, error) {
	var issues []string
	add := func(f string, a ...any) { issues = append(issues, fmt.Sprintf(f, a...)) }

	if !ValidID(cfg.ID) {
		add("id %q must match %s", cfg.ID, idRe.String())
	}
	if strings.TrimSpace(cfg.Name) == "" {
		add("name is required")
	}
	m := &Map{
		ID: cfg.ID, Name: cfg.Name, Description: cfg.Description, Difficulty: cfg.Difficulty,
		MaxMoves: cfg.MaxMoves, Par: -1,
	}
	if m.Difficulty == "" {
		m.Difficulty = "easy"
	}
	if m.MaxMoves == 0 {
		m.MaxMoves = DefaultMaxMoves
	}
	if m.MaxMoves < 1 || m.MaxMoves > MaxMaxMoves {
		add("max_moves must be 1..%d", MaxMaxMoves)
	}

	nl := len(cfg.Levels)
	if nl < 1 || nl > MaxLevels {
		add("levels must have 1..%d entries", MaxLevels)
	}
	if nl > 0 {
		m.Height = len(cfg.Levels[0])
		if m.Height > 0 {
			m.Width = utf8.RuneCountInString(cfg.Levels[0][0])
		}
	}
	if m.Height < MinSide || m.Height > MaxSide {
		add("each level must have %d..%d rows", MinSide, MaxSide)
	}
	if m.Width < MinSide || m.Width > MaxSide {
		add("level rows must be %d..%d wide", MinSide, MaxSide)
	}
	dimsOK := nl >= 1 && nl <= MaxLevels && m.Height >= MinSide && m.Height <= MaxSide && m.Width >= MinSide && m.Width <= MaxSide
	if !dimsOK {
		return nil, &ValidationError{Issues: issues}
	}

	m.Levels = make([][]string, nl)
	m.tiles = make([]Tile, nl*m.Height*m.Width)
	m.partner = map[Position]Position{}
	holes := map[Tile][]Position{}
	starts, finishes := 0, 0
	for l, rows := range cfg.Levels {
		m.Levels[l] = append([]string(nil), rows...)
		if len(rows) != m.Height {
			add("level %d has %d rows, want %d", l, len(rows), m.Height)
			continue
		}
		for y, row := range rows {
			if w := utf8.RuneCountInString(row); w != m.Width {
				add("level %d row %d has width %d, want %d", l, y, w, m.Width)
				continue
			}
			for x, c := range []rune(row) {
				p := Position{l, x, y}
				if c > 127 || !Tile(c).valid() {
					add("level %d (%d,%d): unknown glyph %q", l, x, y, c)
					continue
				}
				t := Tile(c)
				m.tiles[(l*m.Height+y)*m.Width+x] = t
				switch {
				case t == TileStart:
					starts++
					m.Start = p
				case t == TileFinish:
					finishes++
					m.Finish = p
				case t.IsPortal():
					holes[t] = append(holes[t], p)
				}
			}
		}
	}
	if starts != 1 {
		add("layout must contain exactly one start (S), found %d", starts)
	}
	if finishes != 1 {
		add("layout must contain exactly one finish (F), found %d", finishes)
	}
	if starts == 1 && m.Start.Level != 0 {
		add("start (S) must be on level 0 (the bottom), found level %d", m.Start.Level)
	}
	if finishes == 1 && m.Finish.Level != nl-1 {
		add("finish (F) must be on the top level %d, found level %d", nl-1, m.Finish.Level)
	}
	for g := Tile('1'); g <= '9'; g++ {
		ps := holes[g]
		switch len(ps) {
		case 0:
		case 2:
			m.Portals = append(m.Portals, Portal{Glyph: string(rune(g)), A: ps[0], B: ps[1]})
			m.partner[ps[0]] = ps[1]
			m.partner[ps[1]] = ps[0]
		default:
			add("portal %q must appear exactly twice, found %d", rune(g), len(ps))
		}
	}
	if len(issues) > 0 {
		return nil, &ValidationError{Issues: issues}
	}
	// Stairs come in pairs: U at (l,x,y) needs D at (l+1,x,y) and D at (l,x,y)
	// needs U at (l-1,x,y), so whoever can climb a staircase can climb back.
	for l := 0; l < nl; l++ {
		for y := 0; y < m.Height; y++ {
			for x := 0; x < m.Width; x++ {
				p := Position{l, x, y}
				var to Position
				var want Tile
				switch m.TileAt(p) {
				case TileUp:
					to, want = Position{l + 1, x, y}, TileDown
				case TileDown:
					to, want = Position{l - 1, x, y}, TileUp
				default:
					continue
				}
				if !m.InBounds(to) {
					add("level %d (%d,%d): stairs %c lead off the tower", l, x, y, rune(m.TileAt(p)))
				} else if lt := m.TileAt(to); lt != want {
					add("level %d (%d,%d): stairs %c need a matching %c at level %d (%d,%d), found %q",
						l, x, y, rune(m.TileAt(p)), rune(want), to.Level, x, y, rune(lt))
				}
			}
		}
	}
	if len(issues) > 0 {
		return nil, &ValidationError{Issues: issues}
	}
	if path, ok := m.Shortest(); ok {
		m.Par = len(path)
	}
	return m, nil
}

// ValidationError aggregates structural problems.
type ValidationError struct{ Issues []string }

func (e *ValidationError) Error() string { return strings.Join(e.Issues, "; ") }

// State is the mutable state of one game.
type State struct {
	Pos        Position   `json:"pos"`
	Moves      int        `json:"moves"` // counted moves (blocked attempts are free)
	Status     Status     `json:"status"`
	LossReason LossReason `json:"loss_reason,omitempty"`
	// Path lists every tile entered, in order, starting with the start tile.
	// A stairs/portal move adds both the tile stepped on and the landing tile.
	Path []Position `json:"path"`
}

// Clone deep-copies the state.
func (s State) Clone() State {
	s.Path = append([]Position(nil), s.Path...)
	return s
}

// MoveEvent describes one attempted move.
type MoveEvent struct {
	Move       int        `json:"move"` // counted-move index after this attempt (unchanged when blocked)
	Direction  Direction  `json:"direction"`
	From       Position   `json:"from"`
	Stepped    Position   `json:"stepped"` // tile entered by the step; equals From when blocked
	To         Position   `json:"to"`      // final position after stairs/portal; equals From when blocked
	Blocked    bool       `json:"blocked"`
	Via        Via        `json:"via"`
	Status     Status     `json:"status"`
	LossReason LossReason `json:"loss_reason,omitempty"`
}
