package engine

// PortalView is a portal hole as seen from one level. A portal's two holes
// are always on the same level, so both A and B are set whenever the portal
// is visible at all.
type PortalView struct {
	Glyph string
	A     Position
	B     *Position
}

// LevelView is everything a player standing on one level may see. The server
// never exposes more than this to players (fog): other levels' layouts stay
// hidden.
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
		if p.A.Level != l {
			continue
		}
		b := p.B
		v.Portals = append(v.Portals, PortalView{Glyph: p.Glyph, A: p.A, B: &b})
	}
	return v
}
