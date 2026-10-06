package engine

import "errors"

// ErrTerminal is returned by Move when the game already finished.
var ErrTerminal = errors.New("engine: game is in a terminal state")

// NewState returns the initial state for m.
func NewState(m *Map) State {
	return State{Pos: m.Start, Status: StatusReady, Path: []Position{m.Start}}
}

// resolve applies the tile effect of stepping onto q: stairs lift or drop,
// portals teleport, everything else stays. Effects never chain: you land on
// the paired stairs tile (NewMap guarantees U pairs with D one level up) or on
// the twin portal hole, and arriving does not trigger it. To use it you step
// off and back on.
func (m *Map) resolve(q Position) (Position, Via) {
	switch t := m.TileAt(q); {
	case t == TileUp:
		return Position{q.Level + 1, q.X, q.Y}, ViaStairsUp
	case t == TileDown:
		return Position{q.Level - 1, q.X, q.Y}, ViaStairsDown
	case t.IsPortal():
		return m.partner[q], ViaPortal
	}
	return q, ViaWalk
}

// target returns the tile a move in direction d from p enters, and whether it
// is walkable.
func (m *Map) target(p Position, d Direction) (Position, bool) {
	q := Position{p.Level, p.X + dirDelta[d][0], p.Y + dirDelta[d][1]}
	return q, m.TileAt(q).Walkable()
}

// Move attempts one move. A blocked attempt (void or edge) changes nothing and
// is not counted. It returns ErrTerminal if the game already finished. The
// caller owns s.
func (s *State) Move(m *Map, d Direction) (MoveEvent, error) {
	if s.Status.Terminal() {
		return MoveEvent{}, ErrTerminal
	}
	ev := MoveEvent{Move: s.Moves, Direction: d, From: s.Pos, Stepped: s.Pos, To: s.Pos, Via: ViaNone, Status: s.Status}
	if d < 0 || int(d) >= len(dirDelta) {
		ev.Blocked = true
		return ev, nil
	}
	q, ok := m.target(s.Pos, d)
	if !ok {
		ev.Blocked = true
		return ev, nil
	}
	to, via := m.resolve(q)
	s.Moves++
	s.Pos = to
	s.Path = append(s.Path, q)
	if to != q {
		s.Path = append(s.Path, to)
	}
	s.Status = StatusPlaying
	switch {
	case m.TileAt(to) == TileFinish:
		s.Status = StatusWon
	case s.Moves >= m.MaxMoves:
		s.Status = StatusLost
		s.LossReason = LossMoveLimit
	}
	ev.Move, ev.Stepped, ev.To, ev.Via = s.Moves, q, to, via
	ev.Status, ev.LossReason = s.Status, s.LossReason
	return ev, nil
}

// Simulate applies dirs from a fresh state until the game ends or dirs run
// out. Directions after a terminal state are ignored. With withEvents false no
// events are allocated.
func Simulate(m *Map, dirs []Direction, withEvents bool) ([]MoveEvent, State) {
	s := NewState(m)
	var evs []MoveEvent
	for _, d := range dirs {
		ev, err := s.Move(m, d)
		if err != nil {
			break
		}
		if withEvents {
			evs = append(evs, ev)
		}
	}
	return evs, s
}

// Shortest returns a shortest sequence of directions from the start to the
// finish (BFS over positions; every non-blocked move costs 1). ok is false if
// the finish is unreachable. Move limits are ignored.
func (m *Map) Shortest() (path []Direction, ok bool) {
	n := len(m.tiles)
	idx := func(p Position) int { return (p.Level*m.Height+p.Y)*m.Width + p.X }
	prev := make([]int32, n)
	via := make([]int8, n)
	for i := range prev {
		prev[i] = -2
	}
	prev[idx(m.Start)] = -1
	queue := []Position{m.Start}
	for len(queue) > 0 {
		p := queue[0]
		queue = queue[1:]
		if p == m.Finish {
			for i := idx(p); prev[i] != -1; i = int(prev[i]) {
				path = append(path, Direction(via[i]))
			}
			for l, r := 0, len(path)-1; l < r; l, r = l+1, r-1 {
				path[l], path[r] = path[r], path[l]
			}
			return path, true
		}
		for d := Direction(0); d < 4; d++ {
			q, walk := m.target(p, d)
			if !walk {
				continue
			}
			to, _ := m.resolve(q)
			if i := idx(to); prev[i] == -2 {
				prev[i] = int32(idx(p))
				via[i] = int8(d)
				queue = append(queue, to)
			}
		}
	}
	return nil, false
}

// MinLevelChanges is the fewest stairs any winning route takes, or -1 if the
// finish is unreachable.
func (m *Map) MinLevelChanges() int {
	if m.ForcedDescents < 0 {
		return -1
	}
	return m.LevelCount() - 1 + 2*m.ForcedDescents
}

// fewestDescents is a layered BFS where only D stairs cost 1: layer k holds
// every position reachable with exactly k descents and no fewer. It returns -1
// if the finish is unreachable.
func (m *Map) fewestDescents() int {
	idx := func(p Position) int { return (p.Level*m.Height+p.Y)*m.Width + p.X }
	seen := make([]bool, len(m.tiles))
	seen[idx(m.Start)] = true
	layer := []Position{m.Start}
	for k := 0; len(layer) > 0; k++ {
		var next []Position
		for i := 0; i < len(layer); i++ {
			p := layer[i]
			if p == m.Finish {
				return k
			}
			for d := Direction(0); d < 4; d++ {
				q, walk := m.target(p, d)
				if !walk {
					continue
				}
				to, via := m.resolve(q)
				if seen[idx(to)] {
					continue
				}
				if via == ViaStairsDown {
					next = append(next, to)
					continue
				}
				seen[idx(to)] = true
				layer = append(layer, to)
			}
		}
		// A position first found across a descent may have been reached for
		// free later in the same layer; only the rest open the next layer.
		layer = layer[:0]
		for _, p := range next {
			if !seen[idx(p)] {
				seen[idx(p)] = true
				layer = append(layer, p)
			}
		}
	}
	return -1
}

// Reachable returns every position a player can stand on, starting from the
// start tile.
func (m *Map) Reachable() []Position {
	seen := make(map[Position]bool)
	queue := []Position{m.Start}
	seen[m.Start] = true
	var out []Position
	for len(queue) > 0 {
		p := queue[0]
		queue = queue[1:]
		out = append(out, p)
		for d := Direction(0); d < 4; d++ {
			q, walk := m.target(p, d)
			if !walk {
				continue
			}
			to, _ := m.resolve(q)
			if !seen[to] {
				seen[to] = true
				queue = append(queue, to)
			}
		}
	}
	return out
}
