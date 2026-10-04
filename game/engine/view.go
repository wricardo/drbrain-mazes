package engine

// PortalView is a portal hole as seen from one level. A is on the viewed
// level; B is set only when the twin is on the same level too.
type PortalView struct {
	Glyph string
	A     Position
	B     *Position
}

// LevelView is everything a player standing on one level may see. The server
// never exposes more than this to players (fog): other levels' layouts and the
// location of a portal's twin on another level stay hidden.
type LevelView struct {
	Level      int
	LevelCount int
	Rows       []string
	Portals    []PortalView
}

// View returns the fogged view of level l. An out-of-range level yields an
// empty view with no rows.
func (m *Map) View(l int) LevelView {
	v := LevelView{Level: l, LevelCount: len(m.Levels), Rows: []string{}, Portals: []PortalView{}}
	if l < 0 || l >= len(m.Levels) {
		return v
	}
	v.Rows = append(v.Rows, m.Levels[l]...)
	for _, p := range m.Portals {
		switch {
		case p.A.Level == l && p.B.Level == l:
			b := p.B
			v.Portals = append(v.Portals, PortalView{Glyph: p.Glyph, A: p.A, B: &b})
		case p.A.Level == l:
			v.Portals = append(v.Portals, PortalView{Glyph: p.Glyph, A: p.A})
		case p.B.Level == l:
			v.Portals = append(v.Portals, PortalView{Glyph: p.Glyph, A: p.B})
		}
	}
	return v
}
