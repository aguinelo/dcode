package tui

import (
	"fmt"
	"strings"

	"github.com/aguinelo/dcode/internal/protocol"
	"github.com/mattn/go-runewidth"
)

// Geometry is the terminal size and the layout knobs.
type Geometry struct {
	Width  int
	Height int

	PanelWidth         int
	PanelMinTotalWidth int
	PanelHidden        bool

	DiffMaxLines int
	Unicode      bool
}

// DefaultGeometry returns the documented defaults.
func DefaultGeometry(w, h int) Geometry {
	return Geometry{
		Width: w, Height: h,
		PanelWidth: 24, PanelMinTotalWidth: 100,
		DiffMaxLines: 40, Unicode: true,
	}
}

// ShowPanel reports whether the plan panel fits and is wanted.
//
// Responsive before configurable: at 80 columns a 24-wide panel leaves 56 for
// the stream, and a diff in 56 columns is bad. Configuration answers a
// preference; responsiveness answers the case where the user never noticed the
// window got narrow.
func (g Geometry) ShowPanel(hasPlan bool) bool {
	if g.PanelHidden || !hasPlan {
		return false
	}
	return g.Width >= g.PanelMinTotalWidth
}

// StreamWidth is the columns available to the stream.
func (g Geometry) StreamWidth(showPanel bool) int {
	if !showPanel {
		return g.Width
	}
	w := g.Width - g.PanelWidth - 1
	if w < 20 {
		return 20
	}
	return w
}

// marks carries the glyphs for a status, with an ASCII fallback.
type marks struct{ pending, active, done, blocked, bullet string }

// glyphs returns the mark set.
//
// The ASCII fallback must keep blocked and done *distinguishable*. If they
// collapse to the same character, a blocked item reads as finished, which is
// the worst possible error in this panel.
func glyphs(unicode bool) marks {
	if unicode {
		return marks{pending: " ", active: "▸", done: "✓", blocked: "⊘", bullet: "⏺"}
	}
	return marks{pending: " ", active: ">", done: "x", blocked: "!", bullet: "*"}
}

// Render draws the whole screen. Pure over model and geometry, which is what
// allows exact golden tests with no TTY anywhere.
func Render(m Model, g Geometry) string {
	if g.Width <= 0 || g.Height <= 0 {
		return ""
	}
	showPanel := g.ShowPanel(len(m.Plan) > 0)
	streamW := g.StreamWidth(showPanel)

	var b strings.Builder
	b.WriteString(renderStatus(m, g, showPanel))
	b.WriteString("\n")

	bodyH := g.Height - 2
	if bodyH < 1 {
		bodyH = 1
	}

	var body []string
	if m.ShowEmptyState() {
		body = emptyState(m, g, streamW)
	} else {
		body = renderStream(m, g, streamW)
	}

	var panel []string
	if showPanel {
		panel = renderPanel(m, g)
	}

	// Show the tail: during a turn the newest output is what matters.
	if len(body) > bodyH {
		body = body[len(body)-bodyH:]
	}

	for i := 0; i < bodyH; i++ {
		left := ""
		if i < len(body) {
			left = body[i]
		}
		if !showPanel {
			b.WriteString(clip(left, g.Width))
			b.WriteString("\n")
			continue
		}
		right := ""
		if i < len(panel) {
			right = panel[i]
		}
		b.WriteString(pad(clip(left, streamW), streamW))
		b.WriteString("│")
		b.WriteString(clip(right, g.PanelWidth))
		b.WriteString("\n")
	}

	b.WriteString(renderInput(m, g))

	if m.Pending != nil {
		return overlay(b.String(), renderApproval(*m.Pending, g), g)
	}
	return b.String()
}

// renderStatus is the always-visible bar.
func renderStatus(m Model, g Geometry, showPanel bool) string {
	gl := glyphs(g.Unicode)
	state := gl.done
	switch m.State {
	case protocol.SessionStateRunning:
		state = gl.active
	case protocol.SessionStateBlocked:
		state = "!"
	}

	// The sandbox mode is not information, it is a safety indicator: the one
	// piece of state where being wrong is dangerous. full-access is always
	// loud, and must survive with colour switched off.
	mode := m.Sandbox
	if mode == "full-access" {
		mode = "!! FULL-ACCESS !!"
	}

	parts := []string{state + " dcode", m.Model, mode}
	if m.ContextPct > 0 {
		parts = append(parts, fmt.Sprintf("ctx %d%%", m.ContextPct))
	}
	if !showPanel {
		if s := m.PlanSummary(); s != "" {
			parts = append(parts, s)
		}
	}
	return clip(strings.Join(parts, "  "), g.Width)
}

func renderStream(m Model, g Geometry, w int) []string {
	gl := glyphs(g.Unicode)
	var out []string

	for i, e := range m.Entries {
		cursor := "  "
		if i == m.Cursor {
			cursor = "> "
		}
		switch e.Kind {
		case KindAssistant:
			for _, line := range wrap(e.Summary, w-2) {
				out = append(out, "  "+line)
			}
		case KindTool:
			head := fmt.Sprintf("%s%s %-6s %s", cursor, gl.bullet, e.Tool, e.Target)
			if e.Summary != "" && e.Summary != "…" {
				head += "  " + e.Summary
			}
			out = append(out, clip(head, w))
			if e.Expanded && e.Detail != "" {
				out = append(out, detailLines(e.Detail, w, g.DiffMaxLines)...)
			}
		case KindError:
			out = append(out, clip(cursor+"! "+e.Summary, w))
			if e.Expanded && e.Detail != "" {
				out = append(out, detailLines(e.Detail, w, g.DiffMaxLines)...)
			}
		case KindNote:
			out = append(out, clip("  ~ "+e.Summary, w))
		case KindUser:
			out = append(out, clip("> "+e.Summary, w))
		}
	}
	return out
}

func detailLines(detail string, w, max int) []string {
	lines := strings.Split(strings.TrimRight(detail, "\n"), "\n")
	truncated := false
	if max > 0 && len(lines) > max {
		lines = lines[:max]
		truncated = true
	}
	out := make([]string, 0, len(lines)+1)
	for _, l := range lines {
		out = append(out, clip("    │ "+l, w))
	}
	if truncated {
		out = append(out, clip("    │ … output truncated", w))
	}
	return out
}

// renderPanel draws the plan.
func renderPanel(m Model, g Geometry) []string {
	gl := glyphs(g.Unicode)
	w := g.PanelWidth
	out := []string{clip(" PLAN", w), ""}

	for _, it := range m.Plan {
		mark := gl.pending
		switch it.Status {
		case protocol.PlanActive:
			mark = gl.active
		case protocol.PlanDone:
			mark = gl.done
		case protocol.PlanBlocked:
			mark = gl.blocked
		}
		out = append(out, clip(fmt.Sprintf(" %s %d %s", mark, it.ID, it.Text), w))
		if it.Status == protocol.PlanBlocked && it.Blocked != "" {
			// A block with no visible cause is worse than no block at all.
			for _, line := range wrap(it.Blocked, w-6) {
				out = append(out, clip("     "+line, w))
			}
		}
	}

	out = append(out, "")
	if s := m.PlanSummary(); s != "" {
		out = append(out, clip(" "+s, w))
	}
	out = append(out, "", clip(" [p] hide panel", w))
	return out
}

func renderInput(m Model, g Geometry) string {
	if len(m.Queue) > 0 {
		return clip(fmt.Sprintf("(%d queued) > %s", len(m.Queue), m.Input), g.Width)
	}
	return clip("> "+m.Input, g.Width)
}

// renderApproval is the modal.
//
// The screen it takes over is deliberate: as a line in the stream it would
// scroll past during a long turn and be approved unread. It is also the only
// moment the user *must* read, so it gets more care than the rest.
func renderApproval(req protocol.ApprovalRequest, g Geometry) []string {
	w := g.Width - 8
	if w > 60 {
		w = 60
	}
	if w < 24 {
		w = 24
	}

	lines := []string{
		"┌─ Approval needed " + strings.Repeat("─", maxInt(0, w-19)) + "┐",
		"│" + pad("", w) + "│",
		"│" + pad("  "+req.Tool+" crosses: "+req.BoundaryCrossed, w) + "│",
	}
	if req.Command != "" {
		// The rendered command, never a description. Asking for consent to
		// "access the network" without showing what runs is asking blind.
		lines = append(lines, "│"+pad("", w)+"│")
		for _, l := range wrap(req.Command, w-6) {
			lines = append(lines, "│"+pad("    "+l, w)+"│")
		}
	}
	lines = append(lines,
		"│"+pad("", w)+"│",
		// Deny first and as the default: the safe action is the one that costs
		// least effort. The capital A is harder to press by accident, which is
		// right for the option with the largest consequence.
		"│"+pad("  [d] deny   [a] allow   [A] whole session", w)+"│",
		"│"+pad("  Enter denies.", w)+"│",
		"│"+pad("", w)+"│",
		"└"+strings.Repeat("─", w)+"┘",
	)
	return lines
}

// overlay centres the modal over the screen and blocks nothing else from being
// drawn beneath it.
func overlay(screen string, modal []string, g Geometry) string {
	rows := strings.Split(strings.TrimRight(screen, "\n"), "\n")
	top := (len(rows) - len(modal)) / 2
	if top < 0 {
		top = 0
	}
	for i, ml := range modal {
		r := top + i
		if r >= len(rows) {
			break
		}
		left := (g.Width - runewidth.StringWidth(ml)) / 2
		if left < 0 {
			left = 0
		}
		rows[r] = clip(strings.Repeat(" ", left)+ml, g.Width)
	}
	return strings.Join(rows, "\n") + "\n"
}

// emptyState draws the mascot and the essentials.
//
// It is the test that the identity belongs to the product: a mark that cannot
// render in its own terminal is external decoration.
func emptyState(m Model, g Geometry, w int) []string {
	art := []string{
		"    ▄▄▄▄",
		"    █▀▀█",
		"    ████",
		"  ▄▄▄▄▄▄▄▄",
		"  ████████",
		"▄▄▄▄▄▄▄▄▄▄▄▄",
		"████████████",
		" ▀▀      ▀▀ ",
	}
	if !g.Unicode {
		art = []string{
			"    ####",
			"    #oo#",
			"    ####",
			"  ########",
			"  ########",
			"############",
			"############",
			" ##      ## ",
		}
	}

	info := []string{
		"", "dcode", "",
		m.Model,
		m.Sandbox,
		"",
		"? help    ^C interrupt",
	}

	out := []string{""}
	for i := 0; i < len(art); i++ {
		right := ""
		if i < len(info) {
			right = info[i]
		}
		out = append(out, clip("  "+pad(art[i], 14)+"  "+right, w))
	}
	return out
}

// ---------- width-aware helpers ----------
//
// Every one of these measures display cells rather than bytes. Byte width
// breaks on accents and on CJK, and the failure only shows with non-ASCII
// input — which is exactly what a Portuguese-speaking user types.

func clip(s string, w int) string {
	if w <= 0 {
		return ""
	}
	if runewidth.StringWidth(s) <= w {
		return s
	}
	return runewidth.Truncate(s, w, "")
}

func pad(s string, w int) string {
	d := w - runewidth.StringWidth(s)
	if d <= 0 {
		return clip(s, w)
	}
	return s + strings.Repeat(" ", d)
}

func wrap(s string, w int) []string {
	if w <= 0 {
		return []string{""}
	}
	var out []string
	for _, para := range strings.Split(s, "\n") {
		if para == "" {
			out = append(out, "")
			continue
		}
		var line strings.Builder
		for _, word := range strings.Fields(para) {
			// A word wider than the column has to be broken, not merely placed
			// on a line of its own: a path or a stack frame longer than the
			// panel would otherwise overflow and wrap in the terminal, which is
			// what destroys the fixed layout the panel depends on.
			for runewidth.StringWidth(word) > w {
				if line.Len() > 0 {
					out = append(out, line.String())
					line.Reset()
				}
				head := runewidth.Truncate(word, w, "")
				out = append(out, head)
				word = word[len(head):]
			}
			if word == "" {
				continue
			}
			switch {
			case line.Len() == 0:
				line.WriteString(word)
			case runewidth.StringWidth(line.String())+1+runewidth.StringWidth(word) <= w:
				line.WriteString(" ")
				line.WriteString(word)
			default:
				out = append(out, line.String())
				line.Reset()
				line.WriteString(word)
			}
		}
		if line.Len() > 0 {
			out = append(out, line.String())
		}
	}
	if len(out) == 0 {
		return []string{""}
	}
	return out
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
