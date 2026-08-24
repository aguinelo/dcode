package tui

import (
	"strings"
)

// The conversation list, summoned.
//
// It used to be the lower half of the sidebar: twenty-six columns, permanently
// resident, holding something a person opens once in an afternoon. `^R` in
// readline is a search you summon — it appears, you choose, it goes — and
// borrowing the key while making it a standing column contradicted the very
// convention that justified borrowing it.
//
// Drawn as an overlay, over the stream, the way the approval modal already is.
// Nothing about RailNav moved: the cursor that stops at both ends, the filter
// that pulls it back to the top, the naming mode that owns the keyboard, and
// every test over them are unchanged. Only the drawing moved.

// sessionOverlayRows is how many conversations are listed at once.
//
// Ten, and then it says how many more there are. A list that fills the terminal
// is a list that hides the conversation it was summoned over, and the filter is
// the way to reach the eleventh — which is what the filter is for.
const sessionOverlayRows = 10

// renderSessionList draws the overlay, already boxed.
func renderSessionList(m Model, g Geometry) []string {
	gl := glyphs(g.Unicode)
	glr := railGlyphs(g.Unicode)
	p := g.Palette
	t := Text(m.Lang)

	w := g.Width - 8
	if w > 64 {
		w = 64
	}
	if w < 24 {
		w = 24
	}

	// The header says where the typing goes. It is only ever drawn while the
	// list has the keyboard — the overlay does not exist otherwise — so there
	// is no passive form to carry, and the key hint moved to the foot where a
	// hint belongs.
	head := strings.ToUpper(t.RailSessions)
	if m.Nav.Naming {
		head += "  " + t.RailNaming
	} else {
		head += "  " + t.RailFilter + m.Nav.Filter + glr.caret
	}
	top := gl.boxTL + gl.boxH + " " + head + " "
	out := []string{
		top + strings.Repeat(gl.boxH, maxInt(0, w-visibleWidth(top)+1)) + gl.boxTR,
	}

	visible := m.Nav.Visible(m.Sessions)
	if len(visible) == 0 {
		// Said, rather than left blank. A list that empties itself under a
		// filter reads as a list that lost its contents.
		out = append(out, gl.gutter+pad("  "+t.RailNoMatch, w)+gl.gutter)
	}

	// The window follows the cursor rather than starting at the top, so a
	// cursor driven past the tenth row does not walk off a list that stopped
	// drawing at ten.
	first := 0
	if m.Nav.Cursor >= sessionOverlayRows {
		first = m.Nav.Cursor - sessionOverlayRows + 1
	}
	shown := visible[first:]
	if len(shown) > sessionOverlayRows {
		shown = shown[:sessionOverlayRows]
	}
	for i, c := range shown {
		at := first + i
		// The cursor belongs to a list that has the keyboard. A model without
		// it would otherwise mark row zero, which is a claim nobody made.
		under := m.Nav.Active && at == m.Nav.Cursor
		if under && m.Nav.Naming {
			out = append(out, gl.gutter+padStyled(namingRow(m.Nav.Draft, glr, p, w), w)+gl.gutter)
			continue
		}
		row := sessionRow(c, c.ID == m.SessionID, under, glr, p, w-2,
			sessionMeta(c, m.Now, m.Lang, glr))
		out = append(out, gl.gutter+padStyled(" "+row, w)+gl.gutter)
	}
	if rest := len(visible) - first - len(shown); rest > 0 {
		out = append(out, gl.gutter+pad("  "+plural(rest, t.SessionsMoreOne, t.SessionsMoreMany), w)+gl.gutter)
	}

	out = append(out,
		gl.gutter+pad("", w)+gl.gutter,
		gl.gutter+padStyled("  "+p.Apply(StyleHint, t.SessionsKeys), w)+gl.gutter,
		gl.boxBL+strings.Repeat(gl.boxH, w)+gl.boxBR,
	)
	return out
}
