package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/aguinelo/dcode/internal/loop"
	"github.com/aguinelo/dcode/internal/protocol"
	"github.com/mattn/go-runewidth"
)

// RailMode is the sidebar's visibility.
type RailMode int

const (
	// RailHidden is where a terminal starts, and the zero value says so.
	//
	// The rule that used to live beside this — the panel's own mode, mirroring
	// this one — is gone with the panel. The mirror was the defect: the two
	// columns answered "should I be here?" the same way with the same
	// threshold, and their two hundreds compounded, so crossing from 99 to 100
	// columns cost the conversation 46 of them at once.
	RailHidden RailMode = iota
	// RailShown is the user having asked, and it holds at any width.
	RailShown
)

// Geometry is the terminal size and the layout knobs.
type Geometry struct {
	Width  int
	Height int

	// The sidebar. clamp(20, w/5, 30) wide when it is asked for, and asked for
	// is the only way it appears — there is no width rule here any more, and
	// RailMode says why.
	RailWidth    int
	RailMinWidth int
	RailMaxWidth int
	RailMode     RailMode

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
	// ActivityVerbs draws the gerund beside the running tool on the activity
	// line. Presentation, resolved at the edge like Unicode and the palette,
	// because this package never reads the environment.
	ActivityVerbs bool
	Palette       Palette
}

// DefaultGeometry returns the documented defaults.
func DefaultGeometry(w, h int) Geometry {
	return Geometry{
		Width: w, Height: h,
		RailWidth: 22, RailMinWidth: 20, RailMaxWidth: 30,
		// Written out rather than left to the zero value: the default is a
		// decision, and a decision nobody can find in the defaults is one the
		// next reader has to infer from a constant's position in a list.
		//
		// HIDDEN, not auto. Measured on a real session: at 132 columns the
		// column and the panel together took 61 of them and left 71 for the
		// conversation, while the same session at 99 columns — where both
		// disappear — gave it 99. Widening the terminal made the text NARROWER,
		// and the crossing was a single column: 99 gave the stream 99, and 100
		// gave it 53.
		//
		// What the column held was also a second copy of what the stream had
		// just said: every `⏺ write DCODE.md` was followed by a row saying
		// `✓ DCODE.md`. Twenty-six columns is a lot to pay for a repetition.
		//
		// So it is summoned rather than resident, which is what `^B` means in
		// the editor the key was borrowed from — you toggle it, and it stays as
		// you left it.
		RailMode:         RailHidden,
		DiffPreviewLines: 8, DiffMaxLines: 40, CompletionRows: 5,
		ThoughtLines: 4, Unicode: true, ActivityVerbs: true,
	}
}

// ShowRail reports whether the sidebar is drawn.
//
// Nothing to put in it means no sidebar, for the reason an empty panel is worse
// than none: a column of nothing costs the stream twenty characters and tells
// the reader that something is missing.
func (g Geometry) ShowRail(hasContent bool) bool {
	if !hasContent {
		return false
	}
	return g.RailMode == RailShown
}

// railWidth is how wide the sidebar actually draws: a fifth of the screen,
// between its floor and its ceiling, as the design asks.
func (g Geometry) railWidth() int {
	w := g.Width / 5
	floor, ceil := g.RailMinWidth, g.RailMaxWidth
	if floor <= 0 {
		floor = 20
	}
	if ceil <= 0 {
		ceil = 30
	}
	if w < floor {
		w = floor
	}
	if w > ceil {
		w = ceil
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

// StreamWidth is what the stream gets once the columns have taken theirs.
//
// One function, read by the layout and by the renderer both. Two places
// computing a width is the defect; where it shows up is only the symptom, and
// this family has paid for that once already with a painted frame.
func (g Geometry) StreamWidth(showRail bool) int {
	w := g.Width
	if showRail {
		w -= g.railWidth() + 1
	}
	if w < 20 {
		return 20
	}
	return w
}

// marks carries the glyphs for a status, with an ASCII fallback.
type marks struct {
	pending, active, done, blocked, bullet, thought, gutter, ell string
	// dot is the separator between two facts on one line, and minus the sign
	// of a removal. Both were literals until an ASCII terminal got them: they
	// are typography, and typography is the renderer's, never the model's.
	dot, minus, prompt string
	// The lane gutter: one column down the left of every row saying which of
	// the three kinds of thing it is. The `you` lane has no glyph — the rule
	// and the prompt already mark it twice, and a third mark on the one block
	// nobody can miss is ink spent where there is no confusion.
	laneProcess, laneAnswer string
	// The plan tree.
	treeTee, treeEnd string
	// The frame of the approval modal. It was drawn from literals with no
	// fallback at all — the one screen where being unreadable costs the most.
	boxTL, boxTR, boxBL, boxBR, boxH string
}

// glyphs returns the mark set.
//
// The ASCII fallback must keep blocked and done *distinguishable*. If they
// collapse to the same character, a blocked item reads as finished, which is
// the worst possible error in this panel.
func glyphs(unicode bool) marks {
	if unicode {
		return marks{pending: " ", active: "▸", done: "✓", blocked: "⊘", bullet: "⏺", thought: "✻",
			dot: "·", minus: "−", prompt: "❯",
			laneProcess: "╎", laneAnswer: "▏", treeTee: "├", treeEnd: "└",
			boxTL: "┌", boxTR: "┐", boxBL: "└", boxBR: "┘", boxH: "─",
			gutter: "│", ell: "…"}
	}
	return marks{pending: " ", active: ">", done: "x", blocked: "!", bullet: "*", thought: "~",
		dot: "-", minus: "-", prompt: ">",
		laneProcess: ":", laneAnswer: "|", treeTee: "+", treeEnd: "\\",
		boxTL: "+", boxTR: "+", boxBL: "+", boxBR: "+", boxH: "-",
		gutter: "|", ell: "..."}
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
	showRail := g.ShowRail(m.railHasContent())

	var b strings.Builder
	b.WriteString(renderStatus(m, g, false))
	b.WriteString("\n")

	body := StreamLines(m, g)
	visible, top, total, height := Window(m, g, body)

	// The divider follows the same rule as every other glyph: ASCII when the
	// terminal cannot draw the box character. It was a literal here, so a
	// terminal in ASCII mode got a box-drawing rune anyway — visible only once
	// a second column made the same mistake twice.
	divider := "│"
	if !g.Unicode {
		divider = "|"
	}

	var rail []string
	if showRail {
		rail = renderRail(m, g, height)
	}

	streamW := g.StreamWidth(showRail)
	for i := 0; i < height; i++ {
		// The sidebar first, and it never scrolls with the stream: it is a
		// standing answer to "what has this turn touched", not part of the
		// conversation.
		if showRail {
			row := ""
			if i < len(rail) {
				row = rail[i]
			}
			b.WriteString(padStyled(row, g.railWidth()))
			b.WriteString(divider)
		}
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
		b.WriteString(left)
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

	// The approval wins when both are open: one is a question the turn is
	// blocked on, the other is a list somebody summoned.
	if m.Pending != nil {
		return overlay(b.String(), renderApproval(*m.Pending, g, Text(m.Lang)), g)
	}
	if m.Nav.Active {
		return overlay(b.String(), renderSessionList(m, g), g)
	}
	return b.String()
}

// StreamLines renders the whole stream, not just what fits.
//
// Everything, because the window is taken from it afterwards: rendering only
// the tail is what made scrolling impossible, since there was nothing above the
// screen to scroll back to.
func StreamLines(m Model, g Geometry) []string {
	w := g.StreamWidth(g.ShowRail(m.railHasContent()))
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
			// No key beside it any more: the panel it opened is gone, and a
			// hint for a key that does nothing is worse than no hint.
			fields = append(fields, field{p.Apply(StyleMeta, s), 1})
		}
	}
	// The same debt, and it cost more. The sidebar disappears below a hundred
	// columns — which is most terminals — and said nothing at all, so a column
	// that had been built read as a column that had not. The key that brings it
	// back was documented only inside the column that was not on screen.
	//
	// It gives ground before the plan summary does: what the sidebar holds is
	// still reachable another way (`dcode sessions`, the stream itself), and a
	// plan nobody can see has no second route.
	if !g.ShowRail(m.railHasContent()) && m.railHasContent() {
		fields = append(fields, field{
			p.Apply(StyleHint, Text(m.Lang).RailHidden+" · ^b"), 4})
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

// toolBody is what a call carries under its header, and how much of it shows.
//
// The diff wins over the raw output and shows without being asked for: it is
// what gets reviewed. Collapsed it is a preview; Tab reveals the rest, which is
// what makes the hint honest.
func toolBody(e Entry, g Geometry) (string, int) {
	if e.Diff != "" {
		if e.Expanded {
			return e.Diff, g.DiffMaxLines
		}
		return e.Diff, g.DiffPreviewLines
	}
	if e.Expanded && e.Detail != "" {
		return e.Detail, g.DiffMaxLines
	}
	return "", 0
}

// gapBefore puts one blank line in, and never two.
//
// Before rather than after, always. A trailing blank costs a row of the window
// the stream is anchored to, so the last thing that happened scrolls off to
// make room for nothing — measured, not guessed: it dropped the changed line
// out of a diff on a 40-row terminal.
func gapBefore(out []string) []string {
	if len(out) == 0 || out[len(out)-1] == "" {
		return out
	}
	return append(out, "")
}

func renderStream(m Model, g Geometry, w int) []string {
	gl := glyphs(g.Unicode)
	p := g.Palette
	var out []string

	// A call that carries a body reads as a block: a blank line separates it
	// from whatever is around it, so the header, the gutter and the body hold
	// together as one thing.
	//
	// This is the design's card, in the units a terminal has. The spine is
	// already there — detailLines draws a `│` down the left of every body line
	// — so what was missing was the breathing room, not a frame. A frame would
	// cost two columns and two rows per call, join what copy mode selects, and
	// need an ASCII variant, and it would do nothing the gutter does not.
	//
	// A call with no body stays a single line. Most calls are one line, and a
	// card around one line is a box around nothing.
	blocked, skip := false, 0
	for i, e := range m.Entries {
		isBlock := false
		if e.Kind == KindTool {
			b, _ := toolBody(e, g)
			isBlock = b != ""
		}
		if isBlock || blocked {
			out = gapBefore(out)
		}
		blocked = isBlock

		// A delegation is one decision, so it is one block: the header, then a
		// line per child. Skipping the children the header covered is what
		// keeps them from being drawn twice.
		if skip > 0 {
			skip--
			continue
		}
		if n := delegationRun(m.Entries, i); n > 0 {
			out = gapBefore(out)
			out = append(out, renderDelegation(m.Entries[i:i+n], gl, p, w, Text(m.Lang))...)
			skip = n - 1
			blocked = true
			continue
		}

		// Two columns, as before: the lane in the first, the selection marker
		// in the second. `cursor` leads the first row of an entry and `cont`
		// every row after it, so a wrapped answer stays in its lane.
		gut := laneGutter(laneOf(e), gl, p)
		cursor, cont := gut+" ", gut+" "
		if i == m.Cursor {
			cursor = gut + p.Apply(StyleAccent, gl.active)
		}

		switch e.Kind {
		case KindAssistant:
			// The model writes markdown, so the screen reads it. Printing it
			// raw put the asterisks and the backticks on screen, and every
			// answer with emphasis in it looked unfinished.
			lead := cursor
			for _, line := range renderProse(e.Summary, w-2, g) {
				// A paragraph break is an EMPTY row, not two spaces. Indenting
				// it made it whitespace, and the guard that forbids two blank
				// rows in a row compares against "" — so every double blank in
				// prose was invisible to the one test written to catch it.
				//
				// It carries no lane either: a lane on an empty row draws a
				// gutter beside nothing, and the block it belongs to is
				// unambiguous from the rows above and below.
				if line == "" {
					out = gapBefore(out)
					continue
				}
				out = append(out, lead+line)
				lead = cont
			}

		case KindTool:
			body, limit := toolBody(e, g)
			out = append(out, clipStyled(renderToolLine(e, cursor, gl, p, w), w))
			if body != "" {
				out = append(out, detailLines(body, cont, w, g, limit, Text(m.Lang))...)
			}

		case KindError:
			head := cursor + p.Apply(StyleError, "! "+e.Summary)
			out = append(out, clipStyled(head, w))
			if e.Expanded && e.Detail != "" {
				out = append(out, detailLines(e.Detail, cont, w, g, g.DiffMaxLines, Text(m.Lang))...)
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
				out = append(out, detailLines(e.Detail, cont, w, g, g.DiffMaxLines, Text(m.Lang))...)
			}

		case KindPlan:
			out = gapBefore(out)
			out = append(out, renderPlanBlock(e, cursor, cont, gl, g, m.Lang, w)...)
			blocked = true

		case KindReasoning:
			out = append(out, renderThought(e, cursor, cont, gl, g)...)

		case KindNote:
			for j, line := range wrap(e.Summary, w-4) {
				prefix := cursor + "~ "
				if j > 0 {
					prefix = cont + "  "
				}
				out = append(out, clipStyled(prefix+p.Apply(StyleDim, line), w))
			}

		case KindUser:
			// Where the exchange begins, said with a rule.
			//
			// Without it a question is a `>` in the same weight as the prose
			// around it, and a screen of scrollback has no boundary anywhere:
			// on a real session it was impossible to see where one turn ended
			// and the next began. This is the single change that costs the
			// fewest columns — none — for the most reading.
			//
			// A rule and not a colour: a question picked out by colour alone is
			// not picked out at all on a monochrome terminal, and this is the
			// landmark the eye scrolls to.
			out = gapBefore(out)
			// Inset to the same two-column gutter everything else uses. Drawn
			// edge to edge it touched the dividers on both sides and read as a
			// row of a table rather than as the seam between two exchanges.
			out = append(out, cont+p.Apply(StyleChrome, strings.Repeat(gl.boxH, maxInt(0, w-4))))
			for j, line := range wrap(e.Summary, w-4) {
				prefix := cursor + p.Apply(StyleAccent, gl.prompt) + " "
				if j > 0 {
					prefix = cont + "  "
				}
				out = append(out, clipStyled(prefix+p.Apply(StyleBold, line), w))
			}
			// The answer gets its gap from the next pass, the way every other
			// block here does: the gap is drawn BEFORE, never after, so the
			// stream can never end on a blank row.
			blocked = true
		}
	}
	return out
}

// renderPlanBlock draws the plan where the model made it.
//
// A tree, not a list: the design draws `├` down the items and `└` on the last,
// and the shape is doing work — it says these are steps of one thing rather
// than four unrelated lines that happen to be adjacent.
//
// The heading counts the plan the way the model would say it — `plan 2/4` — so
// the block answers "how far along" without the reader counting ticks.
func renderPlanBlock(e Entry, cursor, cont string, gl marks, g Geometry, lang Lang, w int) []string {
	p := g.Palette
	t := Text(lang)

	done := 0
	for _, it := range e.Plan {
		if it.Status == protocol.PlanDone {
			done++
		}
	}
	head := fmt.Sprintf("%s %d/%d", strings.ToLower(t.PanelPlan), done, len(e.Plan))
	out := []string{clipStyled(cursor+p.Apply(StyleHeading, head), w)}

	for i, it := range e.Plan {
		branch := gl.treeTee
		if i == len(e.Plan)-1 {
			branch = gl.treeEnd
		}
		mark, style := gl.pending, StyleMeta
		switch it.Status {
		case protocol.PlanActive:
			mark, style = gl.active, StyleAccent
		case protocol.PlanDone:
			mark, style = gl.done, StyleOK
		case protocol.PlanBlocked:
			mark, style = gl.blocked, StyleError
		}
		room := w - visibleWidth(cont) - 4
		line := cont + p.Apply(StyleChrome, branch) + " " + p.Apply(style, mark) + " " +
			p.Apply(textStyleFor(it.Status), ellipsisTail(it.Text, room, gl.ell))
		out = append(out, clipStyled(line, w))

		// A block with no visible cause is worse than no block at all — the
		// rule the panel carried, kept when the plan moved into the stream.
		if it.Status == protocol.PlanBlocked && it.Blocked != "" {
			for _, l := range wrap(it.Blocked, room-2) {
				out = append(out, clipStyled(cont+"   "+p.Apply(StyleError, l), w))
			}
		}
	}
	return out
}

// textStyleFor keeps a finished step quieter than the one being worked on: the
// eye should land on where the plan IS, not on what it has left behind.
func textStyleFor(s string) Style {
	switch s {
	case protocol.PlanActive:
		return StyleNone
	case protocol.PlanBlocked:
		return StyleProse
	default:
		return StyleMeta
	}
}

// Lane is which of the three things a row is: what you asked, what the model
// did on the way, and what it says.
//
// It is the one idea worth taking whole from the v2 design, and the reason is
// what a long turn looks like: prose and tool calls alternate down the screen
// with nothing structural telling them apart, so catching up means reading
// every row to find out which rows were worth reading. With a lane in the
// gutter the eye can run down the answer lane alone.
//
// It costs NOTHING. Every row of the stream already reserved two columns — the
// selection marker, or two spaces where there was none. The lane takes the
// first of them and the marker keeps the second.
type Lane int

const (
	// LaneProcess is the zero value because it is what an unknown kind should
	// read as: work on the way to an answer, not an answer.
	LaneProcess Lane = iota
	LaneYou
	LaneAnswer
)

// laneOf reads the lane off the kind.
//
// KindAssistant is the answer lane whether the model was narrating mid-turn or
// concluding, because nothing in the event stream tells those apart — the
// design's RESULT block, with its badge and its file list, needs a fact the
// protocol does not carry yet. Recorded in the roadmap rather than guessed at.
func laneOf(e Entry) Lane {
	switch e.Kind {
	case KindUser:
		return LaneYou
	case KindAssistant, KindCompletion:
		return LaneAnswer
	default:
		return LaneProcess
	}
}

// laneGutter is the single column that marks a lane, already styled.
func laneGutter(l Lane, gl marks, p Palette) string {
	switch l {
	case LaneYou:
		return " "
	case LaneAnswer:
		return p.Apply(StyleOK, gl.laneAnswer)
	default:
		return p.Apply(StyleChrome, gl.laneProcess)
	}
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
func renderThought(e Entry, cursor, cont string, gl marks, g Geometry) []string {
	p := g.Palette
	w := g.StreamWidth(false)

	if !e.Closed && !e.Expanded {
		lines := wrap(strings.TrimSpace(e.Summary), w-4)
		if n := g.ThoughtLines; n > 0 && len(lines) > n {
			// The tail, not the head: what it is thinking now is what matters,
			// the same reason the stream follows its own end.
			lines = lines[len(lines)-n:]
		}
		out := make([]string, 0, len(lines))
		for _, l := range lines {
			out = append(out, clipStyled(cont+p.Apply(StyleChrome, gl.gutter+" "+l), w))
		}
		return out
	}

	head := cursor + p.Apply(StyleDim, gl.thought+" thought")
	if d := FormatDuration(e.Duration); d != "" {
		head += p.Apply(StyleMeta, " for "+d)
	}
	if !e.Expanded {
		head += p.Apply(StyleHint, " · Tab")
		return []string{clipStyled(head, w)}
	}

	out := []string{clipStyled(head, w)}
	for _, l := range wrap(strings.TrimSpace(e.Summary), w-4) {
		out = append(out, clipStyled(cont+p.Apply(StyleChrome, gl.gutter+" "+l), w))
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
	target := elide(e.Target, targetW, gl.ell)

	head := fmt.Sprintf("%s%s %-*s %-*s",
		cursor, bullet, toolNameWidth, e.Tool, targetW, target)

	switch {
	case e.Running:
		// What it has got through, when it says. The ellipsis is the fallback
		// rather than the rule: saying nothing reads as finished-with-no-output,
		// and a count says the same thing with a number attached.
		head += " " + p.Apply(StyleMeta, runningMeta(e, gl))
	case e.Summary != "":
		style := StyleMeta
		if e.IsError {
			style = StyleError
		}
		head += " " + p.Apply(style, e.Summary)
	}
	// Only when it was slow enough to be worth a glance. Every call carrying a
	// duration turns the column into noise nobody reads.
	if d := FormatDuration(e.Duration); d != "" && e.Duration >= 500*time.Millisecond {
		head += "  " + p.Apply(StyleMeta, d)
	}
	return strings.TrimRight(head, " ")
}

// runningMeta is what a call in flight has to show for itself.
//
// `184/900` when the total is known, `184` when the walk is still discovering
// it, and the ellipsis when the tool has said nothing. Never a percentage: the
// question of a scan is how much is left, and a share cannot answer it while
// the denominator is still moving.
func runningMeta(e Entry, gl marks) string {
	// A call still arriving is counted in bytes of itself, and says so: it has
	// not run, and "184" beside a tool that has done nothing would read as work
	// already done.
	if e.Arriving {
		if e.Done <= 0 {
			return gl.ell
		}
		return humanBytes(e.Done)
	}
	switch {
	case e.Done > 0 && e.Total > 0:
		return fmt.Sprintf("%d/%d", e.Done, e.Total)
	case e.Done > 0:
		return fmt.Sprintf("%d", e.Done)
	}
	return gl.ell
}

// humanBytes is a size somebody can read at a glance while it grows.
func humanBytes(n int) string {
	if n < 1024 {
		return fmt.Sprintf("%dB", n)
	}
	return fmt.Sprintf("%.1fk", float64(n)/1024)
}

// elide shortens a target, keeping the half that identifies it.
//
// Which half that is depends on what the target IS, and the two answers are
// opposite. A path is identified by its end — the directories leading to it are
// what everything in a repository has in common. A command is identified by its
// beginning: `grep -o '"/[a-z...' chunks` and `grep -rn func src` differ in the
// first twenty characters and end alike, so keeping the tail produced four
// consecutive rows reading `… | sort -u | head -40` for four different searches.
//
// It asks the value, not the tool. `looksLikePath` is already the package's
// answer to "is this a path", and asking it here keeps one definition instead of
// a second list of tool names that would drift from the first. A grep pattern
// and a delegated child's name land on the command side, which is right: both
// are identified by how they start.
func elide(s string, w int, mark string) string {
	if looksLikePath(s) {
		return ellipsis(s, w, mark)
	}
	return ellipsisTail(s, w, mark)
}

// ellipsisTail keeps the beginning and marks what was dropped off the end.
func ellipsisTail(s string, w int, mark string) string {
	if w <= 0 || clipWidth(s) <= w {
		return s
	}
	mw := runewidth.StringWidth(mark)
	if w <= mw {
		return clip(s, w)
	}
	return runewidth.Truncate(s, w-mw, "") + mark
}

// ellipsis shortens the middle of a path, keeping the end.
func ellipsis(s string, w int, mark string) string {
	if w <= 0 || clipWidth(s) <= w {
		return s
	}
	mw := runewidth.StringWidth(mark)
	if w <= mw {
		return clip(s, w)
	}
	tail := runewidth.Truncate(reverse(s), w-mw, "")
	return mark + reverse(tail)
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
func detailLines(detail, lead string, w int, g Geometry, limit int, t Strings) []string {
	gl := glyphs(g.Unicode)
	lines := strings.Split(strings.TrimRight(detail, "\n"), "\n")
	hidden := 0
	if limit > 0 && len(lines) > limit {
		hidden = len(lines) - limit
		lines = lines[:limit]
	}
	out := make([]string, 0, len(lines)+1)
	for _, l := range lines {
		body := g.Palette.Apply(DiffStyle(l), l)
		out = append(out, clipStyled(lead+"  "+gl.gutter+" "+body, w))
	}
	if hidden > 0 {
		// How much is hidden and how to see it. "truncated" alone leaves the
		// reader unable to judge whether it matters.
		note := fmt.Sprintf("  %s %s · %s", gl.ell,
			plural(hidden, t.LineOne, t.LineMany), t.ExpandHint)
		out = append(out, clipStyled(lead+g.Palette.Apply(StyleDim, note), w))
	}
	return out
}

// renderPanel draws the plan.
// renderWorking is the line Claude Code taught everyone to expect: something is
// happening, this is how long it has been happening, and this is how to stop it.
//
// Without it a long turn is indistinguishable from a hung one, and the user's
// only move is to kill the process.
func renderWorking(m Model, g Geometry) string {
	p := g.Palette
	parts := []string{p.Apply(StyleAccent, Spinner(m.Frame, g.Unicode))}

	tool, fact := "", ""
	for i := len(m.Entries) - 1; i >= 0; i-- {
		if m.Entries[i].Running {
			tool = m.Entries[i].Tool
			fact = tool
			if tgt := m.Entries[i].Target; tgt != "" {
				fact += " " + tgt
			}
			break
		}
	}

	// The verb rides BESIDE the fact and never instead of it. Dim against the
	// fact's bold for the same reason: what moves is the accompaniment, what is
	// true is the emphasis. With no tool running there is no fact for a verb to
	// be about, so the line says its one plain word and stays still.
	if fact == "" {
		parts = append(parts, p.Apply(StyleBold, Text(m.Lang).Working))
	} else {
		if g.ActivityVerbs {
			if v := ActivityVerb(tool, m.Frame, m.Lang); v != "" {
				parts = append(parts, p.Apply(StyleMeta, v))
			}
		}
		parts = append(parts, p.Apply(StyleBold, fact))
	}

	if !m.TurnStartedAt.IsZero() && !m.Now.IsZero() {
		if d := m.Now.Sub(m.TurnStartedAt); d > 0 {
			parts = append(parts, p.Apply(StyleMeta, FormatDuration(d)))
		}
	}
	if tk := humanTokens(m.OutputTokens); tk != "" {
		parts = append(parts, p.Apply(StyleMeta, tk+" tok"))
	}
	// The way out belongs next to the thing you want out of.
	parts = append(parts, p.Apply(StyleHint, Text(m.Lang).WorkingInterrupt))

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
	if room := g.Height - 3 - inputFrameRows; rows > room-1 {
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
// inputFrameRows is the rule above the input area and the rule below it.
//
// A box here and not around a tool call, and the difference is what a box is
// for: the input is a FIELD — a fixed region that does not scroll, that you
// return to, and that has to be findable without reading. A tool call is
// content, and a frame around content is a frame around the thing you were
// already reading.
const inputFrameRows = 2

// InputHeight is every row the input area occupies, its frame included. The
// stream's height is computed from this, so a box that grew used to paint over
// the last lines of output.
func InputHeight(m Model, g Geometry) int { return InputRows(m, g) + inputFrameRows }

func renderInputLines(m Model, g Geometry, hint string) []string {
	p := g.Palette
	gl := glyphs(g.Unicode)
	prompt := "> "
	if len(m.Queue) > 0 {
		prompt = fmt.Sprintf("(%d %s) > ", len(m.Queue), Text(m.Lang).Queued)
	}

	// The frame is chrome and stays chrome, whichever region has the keyboard.
	//
	// The first version dimmed it while the stream had focus, and a test asked
	// the obvious next question: does that distinction survive without colour?
	// It did not — it was a colour and nothing else, which is the one thing a
	// state indicator here may not be. And the state is already drawn where it
	// belongs: the entry under the stream cursor carries its own mark.
	//
	// What the frame answers is the question that has no other answer on the
	// screen — where do the letters I type go — and that answer does not change.
	const frame = StyleChrome

	inner := g.Width - 2
	if inner < 1 {
		inner = 1
	}
	rule := func(left, right string) string {
		return p.Apply(frame, left+strings.Repeat(gl.boxH, inner)+right)
	}

	rows := InputRows(m, g)
	lines, caretRow, caretCol := inputWindow(m, rows)

	out := make([]string, 0, rows+inputFrameRows)
	out = append(out, rule(gl.boxTL, gl.boxTR))
	for i := 0; i < rows; i++ {
		text := ""
		if i < len(lines) {
			text = lines[i]
		}
		// One column of gutter inside the frame. Without it the prompt sits
		// against the left rule and the two read as one mark.
		head := strings.Repeat(" ", clipWidth(prompt)+1)
		if i == 0 {
			head = " " + p.Apply(StyleAccent, prompt)
		}
		body := text
		if p.Enabled && i == caretRow {
			body = renderCaretIn(text, caretCol, p)
		}
		line := head + body

		// The hint rides the last row, and only when there is room for it.
		if i == rows-1 && hint != "" {
			// One column of gutter on the right too, so the hint does not
			// read as part of the frame.
			room := inner - clipWidth(prompt) - clipWidth(text) - 3
			if room >= clipWidth(hint) {
				line += strings.Repeat(" ", room-clipWidth(hint)+1) + p.Apply(StyleHint, hint)
			}
		}
		out = append(out, p.Apply(frame, gl.gutter)+
			padStyled(clipStyled(line, inner), inner)+
			p.Apply(frame, gl.gutter))
	}
	out = append(out, rule(gl.boxBL, gl.boxBR))
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
func approvalKeys(req protocol.ApprovalRequest, t Strings) string {
	if standingScope(req) {
		return "  " + t.ApprovalStanding
	}
	return "  " + t.ApprovalOnce
}

func renderApproval(req protocol.ApprovalRequest, g Geometry, t Strings) []string {
	w := g.Width - 8
	if w > 60 {
		w = 60
	}
	if w < 24 {
		w = 24
	}

	gl := glyphs(g.Unicode)
	v := gl.gutter
	// Every piece of the frame comes from the glyph set now. It was drawn from
	// literals with no fallback, which made this the ONE screen a terminal in
	// ASCII could not read — and it is the screen that asks whether a boundary
	// may be crossed.
	head := gl.boxTL + gl.boxH + " " + t.ApprovalTitle + " "
	lines := []string{
		head + strings.Repeat(gl.boxH, maxInt(0, w-visibleWidth(head)+1)) + gl.boxTR,
		v + pad("", w) + v,
		v + pad("  "+req.Tool+" "+t.ApprovalCrosses+" "+req.BoundaryCrossed, w) + v,
	}
	// The network question is about the PROJECT, not this command. A shell
	// command is opaque, so answering yes opens the boundary for everything
	// that runs here — saying "allow this command" would promise something
	// narrower than what the answer does.
	if standingScope(req) {
		lines = append(lines,
			v+pad("", w)+v,
			v+pad("  "+t.ApprovalNetwork, w)+v)
	}
	if req.Command != "" {
		// The rendered command, never a description. Asking for consent to
		// "access the network" without showing what runs is asking blind.
		lines = append(lines, v+pad("", w)+v)
		for _, l := range wrap(req.Command, w-6) {
			lines = append(lines, v+pad("    "+l, w)+v)
		}
	}
	lines = append(lines,
		v+pad("", w)+v,
		// Deny first and as the default: the safe action is the one that costs
		// least effort. The capital A is harder to press by accident, which is
		// right for the option with the largest consequence.
		v+pad(approvalKeys(req, t), w)+v,
		v+pad("  "+t.ApprovalEnter, w)+v,
		v+pad("", w)+v,
		gl.boxBL+strings.Repeat(gl.boxH, w)+gl.boxBR,
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
		p.Apply(StyleHint, "? help    ^C interrupt"),
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
			row += "  " + p.Apply(StyleHint, "^X remove")
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
