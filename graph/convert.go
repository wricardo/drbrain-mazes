package graph

import (
	"github.com/wricardo/drbrain-mazes/game/engine"
	"github.com/wricardo/drbrain-mazes/game/service"
	"github.com/wricardo/drbrain-mazes/graph/model"
)

var directionNames = [...]model.Direction{
	engine.DirUp:    model.DirectionUp,
	engine.DirRight: model.DirectionRight,
	engine.DirDown:  model.DirectionDown,
	engine.DirLeft:  model.DirectionLeft,
}

func toDirection(d engine.Direction) model.Direction {
	if d < 0 || int(d) >= len(directionNames) {
		return model.DirectionUp
	}
	return directionNames[d]
}

// fromDirection maps the GraphQL enum to the engine direction; unknown names
// (rejected by GraphQL enum validation before reaching here) map to -1, which
// the service refuses as INVALID_ARGUMENT.
func fromDirection(d model.Direction) engine.Direction {
	if v, ok := engine.ParseDirection(string(d)); ok {
		return v
	}
	return -1
}

func fromDirections(ds []model.Direction) []engine.Direction {
	out := make([]engine.Direction, len(ds))
	for i, d := range ds {
		out[i] = fromDirection(d)
	}
	return out
}

func toPosition(p engine.Position) *model.Position {
	return &model.Position{Level: p.Level, X: p.X, Y: p.Y}
}

func toPositions(ps []engine.Position) []*model.Position {
	out := make([]*model.Position, len(ps))
	for i, p := range ps {
		out[i] = toPosition(p)
	}
	return out
}

func toLossReason(r engine.LossReason) *model.LossReason {
	if r == engine.LossNone {
		return nil
	}
	lr := model.LossReason(r)
	return &lr
}

func toPortals(ps []engine.Portal) []*model.Portal {
	out := make([]*model.Portal, len(ps))
	for i, p := range ps {
		out[i] = &model.Portal{Glyph: p.Glyph, A: toPosition(p.A), B: toPosition(p.B)}
	}
	return out
}

// nullIfNegative hides the engine's -1 "unreachable" marker as null.
func nullIfNegative(n int) *int {
	if n < 0 {
		return nil
	}
	return &n
}

// toMap converts a map to the public catalog entry. It deliberately exposes
// only level 0 (the start level, via Preview) and counts: no other level, no
// finish and no portal twin may leave through this type (fog).
func toMap(m *engine.Map) *model.Map {
	if m == nil {
		return nil
	}
	return &model.Map{
		ID: m.ID, Name: m.Name, Description: m.Description, Difficulty: m.Difficulty,
		Width: m.Width, Height: m.Height, LevelCount: m.LevelCount(),
		Preview:         append([]string{}, m.View(0).Rows...),
		MaxMoves:        m.MaxMoves,
		ParMoves:        nullIfNegative(m.Par),
		MinLevelChanges: nullIfNegative(m.MinLevelChanges()),
		ForcedDescents:  nullIfNegative(m.ForcedDescents),
	}
}

// toAdminMap converts the full map for the editor. Only admin-gated resolvers
// may call it.
func toAdminMap(m *engine.Map) *model.AdminMap {
	if m == nil {
		return nil
	}
	levels := make([][]string, len(m.Levels))
	for i, rows := range m.Levels {
		levels[i] = append([]string{}, rows...)
	}
	return &model.AdminMap{
		ID: m.ID, Name: m.Name, Description: m.Description, Difficulty: m.Difficulty,
		Width: m.Width, Height: m.Height, LevelCount: m.LevelCount(),
		Levels:          levels,
		Start:           toPosition(m.Start),
		Finish:          toPosition(m.Finish),
		Portals:         toPortals(m.Portals),
		MaxMoves:        m.MaxMoves,
		ParMoves:        nullIfNegative(m.Par),
		MinLevelChanges: nullIfNegative(m.MinLevelChanges()),
		ForcedDescents:  nullIfNegative(m.ForcedDescents),
	}
}

// toLevelView converts the fogged view of level l of m: the rows of that
// level and the portal ends on it. Both ends of a portal are always on the
// same level, so a visible portal always shows its twin too.
func toLevelView(m *engine.Map, l int) *model.LevelView {
	v := m.View(l)
	portals := make([]*model.Portal, len(v.Portals))
	for i, p := range v.Portals {
		portals[i] = &model.Portal{Glyph: p.Glyph, A: toPosition(p.A)}
		if p.B != nil {
			portals[i].B = toPosition(*p.B)
		}
	}
	return &model.LevelView{Level: v.Level, LevelCount: v.LevelCount, Rows: v.Rows, Portals: portals}
}

func toMaps(ms []*engine.Map) []*model.Map {
	out := make([]*model.Map, len(ms))
	for i, m := range ms {
		out[i] = toMap(m)
	}
	return out
}

func fromMapInput(in model.MapInput) engine.MapConfig {
	levels := make([][]string, len(in.Levels))
	for i, rows := range in.Levels {
		levels[i] = append([]string{}, rows...)
	}
	cfg := engine.MapConfig{ID: in.ID, Name: in.Name, Levels: levels}
	if in.Description != nil {
		cfg.Description = *in.Description
	}
	if in.Difficulty != nil {
		cfg.Difficulty = *in.Difficulty
	}
	if in.MaxMoves != nil {
		cfg.MaxMoves = *in.MaxMoves
	}
	return cfg
}

func toGameState(st *engine.State) *model.GameState {
	return &model.GameState{
		Pos:        toPosition(st.Pos),
		Moves:      st.Moves,
		Status:     model.Status(st.Status),
		LossReason: toLossReason(st.LossReason),
		Path:       toPositions(st.Path),
	}
}

func toMoveEvent(e *engine.MoveEvent) *model.MoveEvent {
	if e == nil {
		return nil
	}
	return &model.MoveEvent{
		Move: e.Move, Direction: toDirection(e.Direction),
		From: toPosition(e.From), Stepped: toPosition(e.Stepped), To: toPosition(e.To),
		Blocked: e.Blocked, Via: model.Via(e.Via),
		Status: model.Status(e.Status), LossReason: toLossReason(e.LossReason),
	}
}

// toSession converts an immutable session snapshot (a Clone). It is the only
// Session conversion (queries, mutations and subscriptions all go through it),
// so every path exposes just the fogged view of the current level.
func toSession(s *service.Session) *model.Session {
	if s == nil {
		return nil
	}
	return &model.Session{
		ID: s.ID, DisplayName: s.DisplayName, MapID: s.MapID,
		Map:       toMap(s.Map),
		View:      toLevelView(s.Map, s.State.Pos.Level),
		State:     toGameState(&s.State),
		Seq:       int(s.Seq),
		CreatedAt: s.CreatedAt, LastActionAt: s.LastActionAt,
		Attempts:  s.Attempts,
		BestMoves: s.BestMoves,
		LastEvent: toMoveEvent(s.LastEvent),
	}
}

func toSessionUpdate(s *service.Session) *model.SessionUpdate {
	return &model.SessionUpdate{Seq: int(s.Seq), Session: toSession(s), Event: toMoveEvent(s.LastEvent)}
}
