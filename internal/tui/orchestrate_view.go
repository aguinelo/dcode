package tui

import (
	"fmt"
	"strings"
)

// The two overlays `dcode orchestrate` adds, drawn: the question ^Q opened,
// and the list of agents ^R opened. Both are summoned over the screen, like
// the conversation list, rather than part of the stream.

// overlayWidth is the inner width of a modal on a terminal of width.
func overlayWidth(width int) int {
	w := width - 8
	if w > 64 {
		w = 64
	}
	if w < 24 {
		w = 24
	}
	return w
}

// boxed frames rows under a head, with a hint at the foot.
func boxed(head string, rows []string, hint string, g Geometry, w int) []string {
	gl := glyphs(g.Unicode)
	p := g.Palette
	top := gl.boxTL + gl.boxH + " " + head + " "
	out := []string{top + strings.Repeat(gl.boxH, maxInt(0, w-visibleWidth(top)+1)) + gl.boxTR}
	for _, r := range rows {
		out = append(out, gl.gutter+padStyled(clipStyled(r, w), w)+gl.gutter)
	}
	return append(out,
		gl.gutter+pad("", w)+gl.gutter,
		gl.gutter+padStyled("  "+p.Apply(StyleHint, hint), w)+gl.gutter,
		gl.boxBL+strings.Repeat(gl.boxH, w)+gl.boxBR,
	)
}

// renderAsk draws the open question: what is asked, why, of which child, and
// its answers, one per row. The recommended one is marked by a word, and the
// cursor by a character — neither by colour alone, which a terminal without
// any does not show.
func renderAsk(m Model, g Geometry) []string {
	a := m.Ask
	q := a.Question
	p := g.Palette
	t := Text(m.Lang)
	glr := railGlyphs(g.Unicode)
	w := overlayWidth(g.Width)

	var rows []string
	for _, l := range wrap(q.Text, w-2) {
		rows = append(rows, " "+p.Apply(StyleProse, l))
	}
	if q.Why != "" {
		for i, l := range wrap(q.Why, w-4-visibleWidth(t.AskWhy)) {
			lead := "   " + strings.Repeat(" ", visibleWidth(t.AskWhy))
			if i == 0 {
				lead = " " + p.Apply(StyleMeta, t.AskWhy+":") + " "
			}
			rows = append(rows, lead+p.Apply(StyleMeta, l))
		}
	}
	if q.ChildID != "" {
		rows = append(rows, " "+p.Apply(StyleMeta, t.AskChild+": "+childTitle(m, q.ChildID)))
	}
	if len(q.Sensitive) > 0 {
		rows = append(rows, " "+p.Apply(StyleWarn, t.AskSensitive+": "+strings.Join(q.Sensitive, ", ")))
	}
	rows = append(rows, "")
	if len(q.Options) == 0 {
		rows = append(rows, " "+p.Apply(StyleError, t.AskNoOptions))
	}
	for i, o := range q.Options {
		mark, style := " ", StyleProse
		if i == a.Cursor {
			mark, style = glr.cursor, StyleAccent
		}
		row := " " + p.Apply(StyleAccent, mark) + " " + p.Apply(style, answerLabel(q, o.ID, t))
		if o.ID == q.Recommended {
			row += "  " + p.Apply(StyleOK, "("+t.AskRecommended+")")
		}
		rows = append(rows, row)
	}
	head := strings.ToUpper(t.AskHead)
	if n := len(m.Agents.Questions); n > 1 {
		head += fmt.Sprintf("  1/%d", n)
	}
	return boxed(head, rows, t.AskKeys, g, w)
}

// childTitle names a child by its title when the list knows it, and by its
// id otherwise.
func childTitle(m Model, id string) string {
	for _, c := range m.Agents.Children {
		if c.ID == id && c.Title != "" {
			return c.Title
		}
	}
	return id
}

// renderPick draws the list ^R opened: the cursor by character, the session
// the client is on by its own mark, and each child's state in the person's
// language.
func renderPick(m Model, g Geometry) []string {
	p := g.Palette
	t := Text(m.Lang)
	glr := railGlyphs(g.Unicode)
	am := agentGlyphs(g.Unicode)
	w := overlayWidth(g.Width)

	first := 0
	if m.Pick.Cursor >= sessionOverlayRows {
		first = m.Pick.Cursor - sessionOverlayRows + 1
	}
	shown := m.Pick.Rows[first:]
	if len(shown) > sessionOverlayRows {
		shown = shown[:sessionOverlayRows]
	}
	var rows []string
	for i, r := range shown {
		cursor := " "
		if first+i == m.Pick.Cursor {
			cursor = glr.cursor
		}
		open := " "
		if r.id == m.SessionID {
			open = glr.open
		}
		state, style, mark := "", StyleAccent, am.ask
		if !r.orchestrator {
			state, style = agentStateWord(r.state, t), agentStateStyle(r.state)
			var ok bool
			if mark, ok = am.state[r.state]; !ok {
				mark = am.unknown
			}
		}
		left := " " + p.Apply(StyleAccent, cursor) + p.Apply(StyleAccent, open) + " " +
			p.Apply(style, mark) + " "
		room := w - visibleWidth(left) - visibleWidth(state) - 2
		title := r.title
		if room > 1 {
			title = ellipsisTail(title, room, glyphs(g.Unicode).ell)
		}
		rows = append(rows, rightAlign(left+p.Apply(StyleProse, title), p.Apply(style, state)+" ", w))
	}
	if rest := len(m.Pick.Rows) - first - len(shown); rest > 0 {
		rows = append(rows, "  "+p.Apply(StyleMeta, plural(rest, t.SessionsMoreOne, t.SessionsMoreMany)))
	}
	return boxed(strings.ToUpper(t.SideAgents), rows, t.PickKeys, g, w)
}
