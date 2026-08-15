package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/aguinelo/dcode/internal/loop"
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
	PanelMaxWidth      int
	PanelMinTotalWidth int
	PanelMode          PanelMode

	// DiffPreviewLines is how much of a diff shows without asking. A diff is
	// what gets reviewed, so some of it is always visible — but a whole-file
	// rewrite must not bury the conversation it belongs to.
	DiffPreviewLines int
	DiffMaxLines     int
	// CompletionRows is how many candidates the `/` menu shows at once.
	CompletionRows int
	// ThoughtLines is how much of a live thought stays on screen. Enough to
	// read where it is going, not enough to push the work off the top.
	ThoughtLines int
	Unicode      bool
	Palette      Palette
}

// DefaultGeometry returns the documented defaults.
func DefaultGeometry(w, h int) Geometry {
	return Geometry{
		Width: w, Height: h,
		PanelWidth: 24, PanelMinWidth: 16, PanelMaxWidth: 34, PanelMinTotalWidth: 100,
		DiffPreviewLines: 8, DiffMaxLines: 40, CompletionRows: 5,
		ThoughtLines: 4, Unicode: true,
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
	// A quarter of the screen, between the floor and the ceiling. It gives
	// ground first on a narrow terminal — the panel holds short lines and the
	// stream holds diffs — and takes a little more on a wide one, where a plan
	// item was being cut for no reason.
	if quarter := g.Width / 4; quarter > w {
		w = quarter
	}
	ceiling := g.PanelMaxWidth
	if ceiling <= 0 {
		ceiling = 34
	}
	if w > ceiling {
		w = ceiling
	}
	if q := g.Width / 4; w > q {
		w = q
	}
	if w < floor {
		w = floor
	}
	return w
}

// StreamWidth is the columns available to the stream.
// copyGutterWidth is the cell copy mode borrows from the stream, and the reason
// it is borrowed rather than added: growing the line would push it past the
// terminal, and the layout the alternate screen buys is the thing copy mode is
// compensating for in the first place.
const copyGutterWidth = 2

// copyGutter marks one line as inside or outside the selection.
func copyGutter(c CopyState, line int, g Geometry) string {
	mark := "▌"
	if !g.Unicode {
		mark = ">"
	}
	if !c.Contains(line) {
		return "  "
	}
	return g.Palette.Apply(StyleAccent, mark) + " "
}

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
type marks struct{ pending, active, done, blocked, bullet, thought string }

// glyphs returns the mark set.
//
// The ASCII fallback must keep blocked and done *distinguishable*. If they
// collapse to the same character, a blocked item reads as finished, which is
// the worst possible error in this panel.
func glyphs(unicode bool) marks {
	if unicode {
		return marks{pending: " ", active: "▸", done: "✓", blocked: "⊘", bullet: "⏺", thought: "✻"}
	}
	return marks{pending: " ", active: ">", done: "x", blocked: "!", bullet: "*", thought: "~"}
}

// Render draws the whole screen. Pure over model and geometry, which is what
// allows exact golden tests with no TTY anywhere.
func Render(m Model, g Geometry) string {
	if g.Width <= 0 || g.Height <= 0 {
		return ""
	}
	return fill(render(m, g), g)
}

// fill pads every line to the full width.
//
// A frame owns every cell it covers. Twenty-nine lines in thirty ended before
// the right edge, and every cell past them kept whatever the terminal had there
// — the previous command's output showing through the gaps of a running
// session. Whether the alternate screen was entered is not knowable from in
// here, and with this it stops mattering: a line that reaches the edge cannot
// have anything behind it.
//
// Padding rather than an erase-to-end-of-line escape, because the escape is the
// renderer's business and this has to hold whatever renderer draws it.
func fill(body string, g Geometry) string {
	lines := strings.Split(body, "\n")
	for i, line := range lines {
		if pad := g.Width - visibleWidth(line); pad > 0 {
			lines[i] = line + strings.Repeat(" ", pad)
		}
	}
	return strings.Join(lines, "\n")
}

func render(m Model, g Geometry) string {
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
		// While copy mode is open, every line carries a gutter: a marker where
		// it is selected, a space where it is not. A selection shown only in
		// colour is no selection on a terminal without any, and the gutter is
		// on every line so the text does not shift as the selection moves.
		if m.Copy.Active {
			left = copyGutter(m.Copy, top+i, g) + clipStyled(left, streamW-copyGutterWidth)
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
	for _, l := range renderQueue(m, g) {
		b.WriteString(l)
		b.WriteString("\n")
	}
	for _, l := range renderCompletions(m, g) {
		b.WriteString(l)
		b.WriteString("\n")
	}
	for _, l := range renderInputLines(m, g, ScrollHint(m, g, top, total, height)) {
		b.WriteString(l)
		b.WriteString("\n")
	}
	b.WriteString(RenderStatusBar(m, g))

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
//
// What it drops when the terminal is narrow is a safety decision, not a layout
// one. The order below is deliberate and the sandbox mode is never in it: it is
// the only field where being wrong is dangerous, so a narrow terminal loses the
// model name before it loses the mode.
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

	// The sandbox mode is not information, it is a safety indicator. full-access
	// is always loud, and it changes the *text* rather than only the colour so
	// it survives a monochrome terminal.
	mode := p.Apply(StyleDim, m.Sandbox)
	if m.Sandbox == "full-access" {
		mode = p.Apply(StyleDanger, " !! FULL-ACCESS !! ")
	}

	// Two orders, and they are not the same one. Reading order puts the model
	// beside the name; drop order gives the model up first. Conflating them is
	// how the mode ends up being what disappears.
	type field struct {
		text string
		// drop is the order fields are given up in as the terminal narrows.
		// Zero means never: the mode is a safety indicator, not a field.
		drop int
	}

	fields := []field{
		{p.Apply(stateStyle, state) + " " + p.Apply(StyleBold, "dcode"), 0},
		{p.Apply(StyleDim, m.Model), 3},
		{mode, 0},
	}
	// The seal is undroppable, for the same reason the sandbox mode is: it is
	// not information about the session, it is the one thing on screen that a
	// model claiming success in prose cannot contradict. A field that vanishes
	// on a narrow terminal is a guarantee that vanishes on a narrow terminal.
	if label, style := VerificationLabel(m.Verification, m.Lang); label != "" {
		fields = append(fields, field{p.Apply(style, label), 0})
	}
	if m.Copy.Active {
		// While copying, the status line says what is selected and how to get
		// out. A mode with no visible way out is a mode people force-quit the
		// program to escape.
		fields = append(fields, field{p.Apply(StyleAccent, CopyHint(m.Copy, m.Lang)), 0})
	} else if m.Flash != "" {
		fields = append(fields, field{p.Apply(StyleDim, m.Flash), 1})
	}
	if label := ContextLabel(m.InputTokens, m.Window); label != "" {
		fields = append(fields, field{p.Apply(ContextStyle(m.ContextPct), label), 2})
	}
	if !showPanel {
		// A collapsed panel that says nothing is indistinguishable from a
		// broken one, and the key that brings it back is documented only inside
		// the panel that is not on screen.
		if s := m.PlanSummary(); s != "" {
			fields = append(fields, field{p.Apply(StyleDim, s+" · ^p"), 1})
		}
	}

	render := func(fs []field) string {
		parts := make([]string, 0, len(fs))
		for _, f := range fs {
			parts = append(parts, f.text)
		}
		return strings.Join(parts, "  ")
	}

	// Drop the least important field still present until the line fits.
	for visibleWidth(render(fields)) > g.Width {
		worst, at := 0, -1
		for i, f := range fields {
			if f.drop > worst {
				worst, at = f.drop, i
			}
		}
		if at < 0 {
			break // only undroppable fields left; clipping takes it from here
		}
		fields = append(fields[:at], fields[at+1:]...)
	}
	line := render(fields)

	return clipStyled(line, g.Width)
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
			// The model writes markdown, so the screen reads it. Printing it
			// raw put the asterisks and the backticks on screen, and every
			// answer with emphasis in it looked unfinished.
			for _, line := range renderProse(e.Summary, w-2, g) {
				out = append(out, "  "+line)
			}

		case KindTool:
			out = append(out, clipStyled(renderToolLine(e, cursor, gl, p, w), w))
			// The diff is what gets reviewed, so it wins over the raw output
			// and shows without being asked for. Collapsed it is a preview;
			// Tab reveals the rest, which is what makes the hint honest.
			if body := e.Diff; body != "" {
				limit := g.DiffPreviewLines
				if e.Expanded {
					limit = g.DiffMaxLines
				}
				out = append(out, detailLines(body, w, g, limit)...)
			} else if e.Expanded && e.Detail != "" {
				out = append(out, detailLines(e.Detail, w, g, g.DiffMaxLines)...)
			}

		case KindError:
			head := cursor + p.Apply(StyleError, "! "+e.Summary)
			out = append(out, clipStyled(head, w))
			if e.Expanded && e.Detail != "" {
				out = append(out, detailLines(e.Detail, w, g, g.DiffMaxLines)...)
			}

		case KindCompletion:
			// The one line the model's prose cannot contradict. Styled by what
			// actually ran, and the failing states say so in the words as well
			// as in the colour.
			style := StyleOK
			mark := gl.done
			if strings.HasPrefix(e.Summary, "NOT verified") {
				style, mark = StyleError, "!"
			} else if strings.HasPrefix(e.Summary, "not verified") {
				style, mark = StyleWarn, "?"
			}
			head := cursor + p.Apply(style, mark+" "+e.Summary)
			out = append(out, clipStyled(head, w))
			if e.Expanded && e.Detail != "" {
				out = append(out, detailLines(e.Detail, w, g, g.DiffMaxLines)...)
			}

		case KindReasoning:
			out = append(out, renderThought(e, cursor, gl, g)...)

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

// Column widths for a tool call. Fixed, because the point of the line is that
// the summaries stack into a column the eye can run down — ragged summaries are
// read one at a time, which is exactly what a wall of tool calls must not be.
const (
	toolNameWidth   = 6
	toolTargetWidth = 26
)

// renderThought draws the model thinking.
//
// Open, it streams: the tail of it, dim, because while the model is working
// this is the most informative thing on the screen and the only answer to "is
// it doing something sensible". Closed, it is one line — thinking runs several
// times the length of the answer, and left expanded it buries the result it
// was leading to.
func renderThought(e Entry, cursor string, gl marks, g Geometry) []string {
	p := g.Palette
	w := g.StreamWidth(g.ShowPanel(true))

	if !e.Closed && !e.Expanded {
		lines := wrap(strings.TrimSpace(e.Summary), w-4)
		if n := g.ThoughtLines; n > 0 && len(lines) > n {
			// The tail, not the head: what it is thinking now is what matters,
			// the same reason the stream follows its own end.
			lines = lines[len(lines)-n:]
		}
		out := make([]string, 0, len(lines))
		for _, l := range lines {
			out = append(out, clipStyled(p.Apply(StyleDim, "  │ "+l), w))
		}
		return out
	}

	head := cursor + p.Apply(StyleDim, gl.thought+" thought")
	if d := FormatDuration(e.Duration); d != "" {
		head += p.Apply(StyleDim, " for "+d)
	}
	if !e.Expanded {
		head += p.Apply(StyleDim, " · Tab")
		return []string{clipStyled(head, w)}
	}

	out := []string{clipStyled(head, w)}
	for _, l := range wrap(strings.TrimSpace(e.Summary), w-4) {
		out = append(out, clipStyled(p.Apply(StyleDim, "  │ "+l), w))
	}
	return out
}

// renderToolLine is the one-line form of a call: what ran, on what, how it went
// and how long it took.
func renderToolLine(e Entry, cursor string, gl marks, p Palette, w int) string {
	bullet := p.Apply(StyleAccent, gl.bullet)
	if e.IsError {
		bullet = p.Apply(StyleError, gl.bullet)
	}

	// The target gives ground first when the terminal is narrow: the tool name
	// and the summary are short and load-bearing, a path is neither.
	targetW := toolTargetWidth
	if room := w - len(cursor) - toolNameWidth - 24; room < targetW {
		targetW = room
	}
	if targetW < 8 {
		targetW = 8
	}
	target := ellipsis(e.Target, targetW)

	head := fmt.Sprintf("%s%s %-*s %-*s",
		cursor, bullet, toolNameWidth, e.Tool, targetW, target)

	switch {
	case e.Running:
		// No summary yet, and saying nothing reads as finished-with-no-output.
		head += " " + p.Apply(StyleDim, "…")
	case e.Summary != "":
		style := StyleDim
		if e.IsError {
			style = StyleError
		}
		head += " " + p.Apply(style, e.Summary)
	}
	// Only when it was slow enough to be worth a glance. Every call carrying a
	// duration turns the column into noise nobody reads.
	if d := FormatDuration(e.Duration); d != "" && e.Duration >= 500*time.Millisecond {
		head += "  " + p.Apply(StyleDim, d)
	}
	return strings.TrimRight(head, " ")
}

// ellipsis shortens the middle of a path, keeping the end.
//
// The end is the part that identifies a file; the directories leading to it are
// what everything in a repository has in common.
func ellipsis(s string, w int) string {
	if w <= 0 || clipWidth(s) <= w {
		return s
	}
	if w <= 2 {
		return clip(s, w)
	}
	tail := runewidth.Truncate(reverse(s), w-1, "")
	return "…" + reverse(tail)
}

func reverse(s string) string {
	r := []rune(s)
	for i, j := 0, len(r)-1; i < j; i, j = i+1, j-1 {
		r[i], r[j] = r[j], r[i]
	}
	return string(r)
}

// detailLines renders expanded output, colouring it as a diff when it looks
// like one. The diff is what gets reviewed, so it is the one place where colour
// is doing work rather than decorating.
func detailLines(detail string, w int, g Geometry, limit int) []string {
	lines := strings.Split(strings.TrimRight(detail, "\n"), "\n")
	hidden := 0
	if limit > 0 && len(lines) > limit {
		hidden = len(lines) - limit
		lines = lines[:limit]
	}
	out := make([]string, 0, len(lines)+1)
	for _, l := range lines {
		body := g.Palette.Apply(DiffStyle(l), l)
		out = append(out, clipStyled("    │ "+body, w))
	}
	if hidden > 0 {
		// How much is hidden and how to see it. "truncated" alone leaves the
		// reader unable to judge whether it matters.
		mark := "⋯"
		if !g.Unicode {
			mark = "..."
		}
		note := fmt.Sprintf("    %s %s · Tab expande", mark, plural(hidden, "line", "lines"))
		out = append(out, clipStyled(g.Palette.Apply(StyleDim, note), w))
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

// MaxInputRows is how tall the input box is allowed to get.
//
// Ten, and the number is a compromise. One row made a list impossible to type,
// which is what this whole thing is about. No cap at all means a pasted essay
// takes the terminal and the person loses the conversation they were pasting
// it into.
const MaxInputRows = 10

// InputRows is how many rows the box occupies.
//
// The layout and the renderer both read this, and that is the point. BodyHeight
// used to subtract a literal 3 — status, input, bottom bar — with the input's
// share hard-coded at one. A box that grew without that number growing with it
// would paint over the stream, which is the ghosting already fixed once in "a
// painted frame owns every cell it covers". Two places computing a height is
// the bug; the symptom is only where it shows up.
func InputRows(m Model, g Geometry) int {
	rows := strings.Count(m.Input, "\n") + 1
	if rows > MaxInputRows {
		rows = MaxInputRows
	}
	// The stream keeps at least one row whatever the box wants. A box that
	// fills a short terminal leaves the person typing into nothing.
	if room := g.Height - 3; rows > room-1 {
		rows = room - 1
	}
	if rows < 1 {
		rows = 1
	}
	return rows
}

// inputWindow is the slice of lines the box shows, and where the caret sits
// inside it.
//
// Past the cap the box scrolls rather than truncating: the row being typed on
// has to be visible, and it is usually the last one.
func inputWindow(m Model, rows int) (lines []string, caretRow, caretCol int) {
	lines = strings.Split(m.Input, "\n")
	caretRow, caretCol = caretAt(m.Input, m.InputCursor)

	top := 0
	if caretRow >= rows {
		top = caretRow - rows + 1
	}
	end := top + rows
	if end > len(lines) {
		end = len(lines)
	}
	return lines[top:end], caretRow - top, caretCol
}

// caretAt turns a rune offset into a row and a column.
func caretAt(text string, at int) (row, col int) {
	runes := []rune(text)
	if at < 0 {
		at = 0
	}
	if at > len(runes) {
		at = len(runes)
	}
	for _, r := range runes[:at] {
		if r == '\n' {
			row++
			col = 0
			continue
		}
		col++
	}
	return row, col
}

// renderInputLines draws the box, one string per row.
//
// The prompt is on the first row only: repeating it would read as separate
// messages. The hint is on the last, right-aligned, so it never moves as you
// type.
func renderInputLines(m Model, g Geometry, hint string) []string {
	p := g.Palette
	prompt := "> "
	if len(m.Queue) > 0 {
		prompt = fmt.Sprintf("(%d %s) > ", len(m.Queue), Text(m.Lang).Queued)
	}

	rows := InputRows(m, g)
	lines, caretRow, caretCol := inputWindow(m, rows)

	out := make([]string, 0, rows)
	for i := 0; i < rows; i++ {
		text := ""
		if i < len(lines) {
			text = lines[i]
		}
		head := strings.Repeat(" ", clipWidth(prompt))
		if i == 0 {
			head = prompt
		}
		body := text
		if p.Enabled && i == caretRow {
			body = renderCaretIn(text, caretCol, p)
		}
		line := head + body

		// The hint rides the last row, and only when there is room for it.
		if i == rows-1 && hint != "" {
			room := g.Width - clipWidth(head+text) - 1
			if room >= clipWidth(hint) {
				line += strings.Repeat(" ", room-clipWidth(hint)+1) + p.Apply(StyleDim, hint)
			}
		}
		out = append(out, padStyled(clipStyled(line, g.Width), g.Width))
	}
	return out
}

// renderCaretIn marks where typing will land on one row.
func renderCaretIn(text string, at int, p Palette) string {
	runes := []rune(text)
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
// standingScope reports the crossings whose answer can outlive the session.
//
// Only the network. A standing answer to "write outside the workspace" would be
// a grant over the whole filesystem given once, months ago, for a reason nobody
// records — and the path in that question is what makes it answerable at all.
func standingScope(req protocol.ApprovalRequest) bool {
	return req.BoundaryCrossed == "network"
}

// approvalKeys lists the answers available for this crossing.
//
// Deny first and as the default: the safe action is the one that costs least
// effort. The capitals are harder to press by accident, which is right for the
// options with the largest consequence — and the two that are written down are
// the two that need the most deliberate keystroke.
func approvalKeys(req protocol.ApprovalRequest) string {
	if standingScope(req) {
		return "  [d] no   [a] once   [P] this project   [G] always"
	}
	return "  [d] deny   [a] allow   [A] whole session"
}

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
	// The network question is about the PROJECT, not this command. A shell
	// command is opaque, so answering yes opens the boundary for everything
	// that runs here — saying "allow this command" would promise something
	// narrower than what the answer does.
	if standingScope(req) {
		lines = append(lines,
			"│"+pad("", w)+"│",
			"│"+pad("  Commands in this project may reach the network.", w)+"│")
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
		"│"+pad(approvalKeys(req), w)+"│",
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
	p := g.Palette

	// Each row carries the role its voxels have in the brand: lit face, front
	// face, shaded face. The eye is the one terracotta in the whole interface.
	type row struct {
		text  string
		style Style
	}
	art := []row{
		{"    ▄▄▄▄", StyleHighlight},
		{"    █", StyleBody}, // the eye row is assembled below
		{"    ████", StyleBody},
		{"  ▄▄▄▄▄▄▄▄", StyleHighlight},
		{"  ████████", StyleBody},
		{"▄▄▄▄▄▄▄▄▄▄▄▄", StyleHighlight},
		{"████████████", StyleBody},
		{" ▀▀      ▀▀ ", StyleShadow},
	}
	if !g.Unicode {
		art = []row{
			{"    ####", StyleHighlight},
			{"    #", StyleBody},
			{"    ####", StyleBody},
			{"  ########", StyleHighlight},
			{"  ########", StyleBody},
			{"############", StyleHighlight},
			{"############", StyleBody},
			{" ##      ## ", StyleShadow},
		}
	}

	// The eye row is the only one that mixes roles.
	eye, body := "▀▀", "█"
	if !g.Unicode {
		eye, body = "oo", "#"
	}
	eyeRow := p.Apply(StyleBody, "    "+body) +
		p.Apply(StyleEye, eye) + p.Apply(StyleBody, body)

	info := []string{
		"",
		p.Apply(StyleBold, "dcode"),
		"",
		p.Apply(StyleDim, m.Model),
		p.Apply(StyleDim, m.Sandbox),
		"",
		p.Apply(StyleDim, "? help    ^C interrupt"),
	}
	if m.Sandbox == "full-access" {
		info[4] = p.Apply(StyleDanger, " !! FULL-ACCESS !! ")
	}

	out := []string{""}
	for i, r := range art {
		left := p.Apply(r.style, r.text)
		if i == 1 {
			left = eyeRow
		}
		right := ""
		if i < len(info) {
			right = info[i]
		}
		out = append(out, clipStyled("  "+padStyled(left, 14)+"  "+right, w))
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

// renderQueue lists what is waiting to be sent.
//
// Visible and removable: a message the user typed and cannot see is a message
// they will type again, and one they cannot take back is worse than one that
// was refused.
func renderQueue(m Model, g Geometry) []string {
	if len(m.Queue) == 0 {
		return nil
	}
	p := g.Palette
	mark := "⇥"
	if !g.Unicode {
		mark = ">>"
	}

	out := make([]string, 0, len(m.Queue))
	for i, text := range m.Queue {
		row := fmt.Sprintf("%s %d %s", p.Apply(StyleDim, mark), i+1, text)
		if i == 0 {
			// The key goes on the first row only: repeated on every row it is
			// noise, and absent it is a feature nobody finds.
			row += "  " + p.Apply(StyleDim, "^X remove")
		}
		out = append(out, clipStyled(p.Apply(StyleDim, row), g.Width))
	}
	return out
}

// renderCompletions draws the `/` menu above the input.
func renderCompletions(m Model, g Geometry) []string {
	if len(m.Completions) == 0 {
		return nil
	}
	p := g.Palette

	visible := m.Completions
	shown := g.CompletionRows
	if shown <= 0 {
		shown = 5
	}
	// A window around the highlight, so walking past the end keeps moving.
	from := 0
	if len(visible) > shown {
		from = m.CompletionAt - shown/2
		if from < 0 {
			from = 0
		}
		if from > len(visible)-shown {
			from = len(visible) - shown
		}
		visible = visible[from : from+shown]
	}

	out := make([]string, 0, len(visible)+1)
	for i, c := range visible {
		name := "/" + c.Name
		if c.Args != "" {
			name += " " + c.Args
		}
		row := fmt.Sprintf("  %-22s %s", name, c.Description)
		if from+i == m.CompletionAt {
			out = append(out, clipStyled(p.Apply(StyleCursor, padStyled(row, g.Width)), g.Width))
			continue
		}
		out = append(out, clipStyled("  "+p.Apply(StyleAccent, name)+
			strings.Repeat(" ", maxInt(1, 22-clipWidth(name)))+
			p.Apply(StyleDim, c.Description), g.Width))
	}

	footer := fmt.Sprintf("  %d de %d · ↑↓ navegar · ⇥ completar · esc fechar",
		m.CompletionAt+1, len(m.Completions))
	if !g.Unicode {
		footer = fmt.Sprintf("  %d de %d · up/down navegar · tab completar · esc fechar",
			m.CompletionAt+1, len(m.Completions))
	}
	return append(out, clipStyled(p.Apply(StyleDim, footer), g.Width))
}

// VerificationLabel is the seal shown on the status line.
//
// Short enough to survive beside everything else, and worded so the failing
// states read as failing without colour: a monochrome terminal, a screenshot in
// a bug report, and a colour-blind reader all have to get the same answer.
func VerificationLabel(v string, lang Lang) (string, Style) {
	t := Text(lang)
	switch v {
	case string(loop.VerificationPassed):
		return t.VerifiedLabel, StyleOK
	case string(loop.VerificationFailed):
		return t.NotVerifiedLabel, StyleDanger
	case string(loop.VerificationStale):
		return t.UnverifiedLabel, StyleWarn
	case string(loop.VerificationUnavailable):
		return t.UnverifiedLabel, StyleWarn
	default:
		// Clean, or no definition of done. Nothing changed, so there is nothing
		// to claim either way, and a permanent label would stop being read.
		return "", StyleDim
	}
}

// LineStart is the offset of the beginning of the line the caret is on.
//
// Home means this line, not the whole buffer. On one line the two are the same
// and nothing changes; on three they are the difference between correcting a
// word and jumping to the top.
func LineStart(text string, at int) int {
	runes := []rune(text)
	at = clampRune(runes, at)
	for i := at - 1; i >= 0; i-- {
		if runes[i] == '\n' {
			return i + 1
		}
	}
	return 0
}

// LineEnd is the offset just before the next break, or the end of the text.
func LineEnd(text string, at int) int {
	runes := []rune(text)
	at = clampRune(runes, at)
	for i := at; i < len(runes); i++ {
		if runes[i] == '\n' {
			return i
		}
	}
	return len(runes)
}

// LineUp is where the caret lands one row up, or -1 when there is no row above.
//
// The -1 matters: with nothing above, up is not a movement at all, and the
// caller falls back to walking the command history — which is what up has
// always done on an empty line.
//
// The column is kept where the shorter line allows and clamped to its end
// otherwise, because overshooting would land on a row the user did not aim at.
func LineUp(text string, at int) int {
	start := LineStart(text, at)
	if start == 0 {
		return -1
	}
	col := at - start
	prevStart := LineStart(text, start-1)
	prevEnd := start - 1
	if prevStart+col > prevEnd {
		return prevEnd
	}
	return prevStart + col
}

// LineDown is where the caret lands one row down, or -1 when there is none.
func LineDown(text string, at int) int {
	runes := []rune(text)
	at = clampRune(runes, at)
	end := LineEnd(text, at)
	if end >= len(runes) {
		return -1
	}
	col := at - LineStart(text, at)
	nextStart := end + 1
	nextEnd := LineEnd(text, nextStart)
	if nextStart+col > nextEnd {
		return nextEnd
	}
	return nextStart + col
}

func clampRune(runes []rune, at int) int {
	if at < 0 {
		return 0
	}
	if at > len(runes) {
		return len(runes)
	}
	return at
}
