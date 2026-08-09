package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/aguinelo/dcode/internal/protocol"
	"github.com/mattn/go-runewidth"
)

// PanelMode is how the plan panel decides whether to show.
type PanelMode int

const (
	// PanelAuto lets the width decide. The default, because it answers the
	// case the user has not thought about.
	PanelAuto PanelMode = iota
	// PanelShown and PanelHidden are the user having thought about it.
	PanelShown
	PanelHidden
)

// Geometry is the terminal size and the layout knobs.
type Geometry struct {
	Width  int
	Height int

	PanelWidth         int
	PanelMinWidth      int
	PanelMinTotalWidth int
	PanelMode          PanelMode

	DiffMaxLines int
	Unicode      bool
	Palette      Palette
}

// DefaultGeometry returns the documented defaults.
func DefaultGeometry(w, h int) Geometry {
	return Geometry{
		Width: w, Height: h,
		PanelWidth: 24, PanelMinWidth: 16, PanelMinTotalWidth: 100,
		DiffMaxLines: 40, Unicode: true,
	}
}

// ShowPanel reports whether the plan panel is drawn.
//
// Responsive by default: at 80 columns a 24-wide panel leaves 56 for the
// stream, and a diff in 56 columns is bad. But responsiveness answers the case
// where the user never noticed the window got narrow — and a keypress *is* the
// user noticing, so an explicit choice wins over the default at any width.
//
// No plan means no panel in every mode: an empty panel is worse than none.
func (g Geometry) ShowPanel(hasPlan bool) bool {
	if !hasPlan {
		return false
	}
	switch g.PanelMode {
	case PanelShown:
		return true
	case PanelHidden:
		return false
	}
	return g.Width >= g.PanelMinTotalWidth
}

// panelWidth is how wide the panel actually draws.
//
// It gives ground before it disappears: asked for on a narrow terminal, it
// shrinks rather than taking the room the stream needs. Below its own minimum
// an item is unreadable, so that is the floor.
func (g Geometry) panelWidth() int {
	w := g.PanelWidth
	if w <= 0 {
		w = 24
	}
	floor := g.PanelMinWidth
	if floor <= 0 {
		floor = 16
	}
	// A quarter of the screen, never more. At 80 columns that trades four
	// panel cells for four stream cells, which is the right way round: the
	// panel holds short lines and the stream holds diffs.
	if quarter := g.Width / 4; w > quarter {
		w = quarter
	}
	if w < floor {
		w = floor
	}
	return w
}

// StreamWidth is the columns available to the stream.
func (g Geometry) StreamWidth(showPanel bool) int {
	if !showPanel {
		return g.Width
	}
	w := g.Width - g.panelWidth() - 1
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

	var b strings.Builder
	b.WriteString(renderStatus(m, g, showPanel))
	b.WriteString("\n")

	body := StreamLines(m, g)
	visible, top, total, height := Window(m, g, body)

	var panel []string
	if showPanel {
		panel = renderPanel(m, g)
	}

	streamW := g.StreamWidth(showPanel)
	for i := 0; i < height; i++ {
		left := ""
		if i < len(visible) {
			left = visible[i]
		}
		if !showPanel {
			b.WriteString(left)
			b.WriteString("\n")
			continue
		}
		right := ""
		if i < len(panel) {
			right = panel[i]
		}
		b.WriteString(padStyled(left, streamW))
		b.WriteString("│")
		b.WriteString(clip(right, g.panelWidth()))
		b.WriteString("\n")
	}

	if m.workingVisible() {
		b.WriteString(renderWorking(m, g))
		b.WriteString("\n")
	}
	b.WriteString(renderInput(m, g, ScrollHint(m, g, top, total, height)))

	if m.Pending != nil {
		return overlay(b.String(), renderApproval(*m.Pending, g), g)
	}
	return b.String()
}

// StreamLines renders the whole stream, not just what fits.
//
// Everything, because the window is taken from it afterwards: rendering only
// the tail is what made scrolling impossible, since there was nothing above the
// screen to scroll back to.
func StreamLines(m Model, g Geometry) []string {
	w := g.StreamWidth(g.ShowPanel(len(m.Plan) > 0))
	if m.ShowEmptyState() {
		return emptyState(m, g, w)
	}
	return renderStream(m, g, w)
}

// renderStatus is the always-visible bar.
func renderStatus(m Model, g Geometry, showPanel bool) string {
	gl := glyphs(g.Unicode)
	p := g.Palette

	state, stateStyle := gl.done, StyleOK
	switch m.State {
	case protocol.SessionStateRunning:
		state, stateStyle = Spinner(m.Frame, g.Unicode), StyleAccent
	case protocol.SessionStateBlocked:
		state, stateStyle = "!", StyleWarn
	}

	parts := []string{
		p.Apply(stateStyle, state) + " " + p.Apply(StyleBold, "dcode"),
		p.Apply(StyleDim, m.Model),
	}

	// The sandbox mode is not information, it is a safety indicator: the one
	// piece of state where being wrong is dangerous. full-access is always
	// loud, and must survive with colour switched off — which is why it also
	// changes the text and not only the colour.
	if m.Sandbox == "full-access" {
		parts = append(parts, p.Apply(StyleDanger, " !! FULL-ACCESS !! "))
	} else {
		parts = append(parts, p.Apply(StyleDim, m.Sandbox))
	}

	if label := ContextLabel(m.InputTokens, m.Window); label != "" {
		parts = append(parts, p.Apply(ContextStyle(m.ContextPct), label))
	}
	if !showPanel {
		// A collapsed panel that says nothing is indistinguishable from a
		// broken one, and the key that brings it back is only documented
		// inside the panel itself.
		if s := m.PlanSummary(); s != "" {
			parts = append(parts, p.Apply(StyleDim, s), p.Apply(StyleDim, "[^p] plan"))
		}
	}
	return clipStyled(strings.Join(parts, "  "), g.Width)
}

func renderStream(m Model, g Geometry, w int) []string {
	gl := glyphs(g.Unicode)
	p := g.Palette
	var out []string

	for i, e := range m.Entries {
		selected := i == m.Cursor
		cursor := "  "
		if selected {
			cursor = p.Apply(StyleAccent, "> ")
		}

		switch e.Kind {
		case KindAssistant:
			for _, line := range wrap(e.Summary, w-2) {
				out = append(out, "  "+line)
			}

		case KindTool:
			out = append(out, clipStyled(renderToolLine(e, cursor, gl, p), w))
			if e.Expanded && e.Detail != "" {
				out = append(out, detailLines(e.Detail, w, g)...)
			}

		case KindError:
			head := cursor + p.Apply(StyleError, "! "+e.Summary)
			out = append(out, clipStyled(head, w))
			if e.Expanded && e.Detail != "" {
				out = append(out, detailLines(e.Detail, w, g)...)
			}

		case KindNote:
			for j, line := range wrap(e.Summary, w-4) {
				prefix := "  ~ "
				if j > 0 {
					prefix = "    "
				}
				out = append(out, clipStyled(p.Apply(StyleDim, prefix+line), w))
			}

		case KindUser:
			for j, line := range wrap(e.Summary, w-2) {
				prefix := "> "
				if j > 0 {
					prefix = "  "
				}
				out = append(out, clipStyled(p.Apply(StyleBold, prefix+line), w))
			}
		}
	}
	return out
}

// renderToolLine is the one-line form of a call: what ran, on what, how it went
// and how long it took.
func renderToolLine(e Entry, cursor string, gl marks, p Palette) string {
	bullet := p.Apply(StyleAccent, gl.bullet)
	if e.IsError {
		bullet = p.Apply(StyleError, gl.bullet)
	}
	head := fmt.Sprintf("%s%s %-6s %s", cursor, bullet, e.Tool, e.Target)

	switch {
	case e.Running:
		// No summary yet, and saying nothing reads as finished-with-no-output.
		head += "  " + p.Apply(StyleDim, "…")
	case e.Summary != "":
		style := StyleDim
		if e.IsError {
			style = StyleError
		}
		head += "  " + p.Apply(style, e.Summary)
	}
	// Only when it was slow enough to be worth a glance. Every call carrying a
	// duration turns the column into noise nobody reads.
	if d := FormatDuration(e.Duration); d != "" && e.Duration >= 500*time.Millisecond {
		head += "  " + p.Apply(StyleDim, d)
	}
	return head
}

// detailLines renders expanded output, colouring it as a diff when it looks
// like one. The diff is what gets reviewed, so it is the one place where colour
// is doing work rather than decorating.
func detailLines(detail string, w int, g Geometry) []string {
	lines := strings.Split(strings.TrimRight(detail, "\n"), "\n")
	truncated := false
	if max := g.DiffMaxLines; max > 0 && len(lines) > max {
		lines = lines[:max]
		truncated = true
	}
	out := make([]string, 0, len(lines)+1)
	for _, l := range lines {
		body := g.Palette.Apply(DiffStyle(l), l)
		out = append(out, clipStyled("    │ "+body, w))
	}
	if truncated {
		out = append(out, clipStyled(g.Palette.Apply(StyleDim, "    │ … output truncated"), w))
	}
	return out
}

// renderPanel draws the plan.
func renderPanel(m Model, g Geometry) []string {
	gl := glyphs(g.Unicode)
	w := g.panelWidth()
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
	out = append(out, "", clip(" [^p] hide panel", w))
	return out
}

// renderWorking is the line Claude Code taught everyone to expect: something is
// happening, this is how long it has been happening, and this is how to stop it.
//
// Without it a long turn is indistinguishable from a hung one, and the user's
// only move is to kill the process.
func renderWorking(m Model, g Geometry) string {
	p := g.Palette
	parts := []string{p.Apply(StyleAccent, Spinner(m.Frame, g.Unicode))}

	verb := "working"
	for i := len(m.Entries) - 1; i >= 0; i-- {
		if m.Entries[i].Running {
			verb = m.Entries[i].Tool
			if tgt := m.Entries[i].Target; tgt != "" {
				verb += " " + tgt
			}
			break
		}
	}
	parts = append(parts, p.Apply(StyleBold, verb))

	if !m.TurnStartedAt.IsZero() && !m.Now.IsZero() {
		if d := m.Now.Sub(m.TurnStartedAt); d > 0 {
			parts = append(parts, p.Apply(StyleDim, FormatDuration(d)))
		}
	}
	if tk := humanTokens(m.OutputTokens); tk != "" {
		parts = append(parts, p.Apply(StyleDim, tk+" tok"))
	}
	// The way out belongs next to the thing you want out of.
	parts = append(parts, p.Apply(StyleDim, "^C interrupts"))

	return clipStyled(strings.Join(parts, "  "), g.Width)
}

func renderInput(m Model, g Geometry, hint string) string {
	p := g.Palette
	prompt := "> "
	if len(m.Queue) > 0 {
		prompt = fmt.Sprintf("(%d queued) > ", len(m.Queue))
	}

	line := prompt + m.Input
	// The caret is drawn rather than left to the terminal: the input sits on a
	// line the renderer owns, and a hardware cursor would be wherever the last
	// write left it.
	if p.Enabled {
		line = prompt + renderCaret(m, p)
	}
	if hint == "" {
		return clipStyled(line, g.Width)
	}

	// The hint is right-aligned so it never moves as you type.
	room := g.Width - clipWidth(prompt+m.Input) - 1
	if room < clipWidth(hint) {
		return clipStyled(line, g.Width)
	}
	pad := strings.Repeat(" ", room-clipWidth(hint)+1)
	return clipStyled(line+pad+p.Apply(StyleDim, hint), g.Width)
}

// renderCaret marks where typing will land.
func renderCaret(m Model, p Palette) string {
	runes := []rune(m.Input)
	at := m.InputCursor
	if at < 0 {
		at = 0
	}
	if at > len(runes) {
		at = len(runes)
	}
	if at == len(runes) {
		return string(runes) + p.Apply(StyleCursor, " ")
	}
	return string(runes[:at]) + p.Apply(StyleCursor, string(runes[at])) + string(runes[at+1:])
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
