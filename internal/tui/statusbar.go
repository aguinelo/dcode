package tui

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/aguinelo/dcode/internal/protocol"
)

// RenderStatusBar draws the bottom bar: where you are, what has changed, and
// what is waiting.
//
// One line, always, and never two. It is the only region of the screen that is
// true regardless of what the stream happens to be showing, so it gives ground
// by dropping segments rather than by wrapping — a second row would take a line
// from the layout the rest of the screen owns, and it would take it at exactly
// the moment the terminal was already too narrow.
//
// Pure over the model and the geometry, like everything else that draws here.
// Segments are measured in display cells and assembled until they fit.
func RenderStatusBar(m Model, g Geometry) string {
	// A segment nobody can populate does not render. That is the rule the
	// design states for the pending badge — "some por completo quando zero, sem
	// badge vazio" — and it generalises: a bar of empty slots describes the bar
	// rather than the session.
	segs := []segment{navSegment(m, g), worktreeSegment(m, g)}
	if seg, ok := modeSegment(m, g); ok {
		segs = append(segs, seg)
	}
	if seg, ok := diffSegment(m, g); ok {
		segs = append(segs, seg)
	}
	if seg, ok := ceilingSegment(m, g); ok {
		segs = append(segs, seg)
	}
	if seg, ok := waitingSegment(m, g); ok {
		segs = append(segs, seg)
	}
	if seg, ok := posSegment(m, g); ok {
		segs = append(segs, seg)
	}

	// Dropped from the right, cheapest first: what a person can reconstruct
	// somewhere else goes before what exists nowhere else. The diff is on the
	// diff of the turn; where you are, and what is blocked on you, are not.
	for {
		if !fits(segs, g.Width) {
			if !dropOne(&segs) {
				break
			}
			continue
		}
		// It fits, and the path still has nowhere to go. Give up only what the
		// path outranks — the key hints, which `?` restates in full.
		//
		// Taking the leftover space alone was not enough: the segments fit at
		// eighty cells and left nine, so the path vanished between roughly
		// seventy and ninety-five and came back below it, once the hints were
		// dropped for width. It disappeared at the one width most terminals
		// actually are, and reappeared as the window got smaller.
		if roomForPath(segs, g.Width) || !dropOneAbove(&segs, pathOutranks) {
			break
		}
	}
	return withWorkspacePath(assemble(segs, g), m, g)
}

// pathOutranks is the drop order the path beats.
//
// Above the key hints, which `?` restates in full, and below everything else:
// the diff, the position and the mode are each the only place their fact
// appears, and a bar that keeps a path by dropping what changed has chosen the
// address over the news.
const pathOutranks = 3

// roomForPath reports whether the bar could still draw a usable path.
func roomForPath(segs []segment, width int) bool {
	return barWidth(segs)+minPathCells+2 <= width
}

// dropOneAbove gives up the most expendable segment ranked above a floor, and
// reports whether it could.
func dropOneAbove(segs *[]segment, floor int) bool {
	worst, at := floor, -1
	for i, s := range *segs {
		if s.drop > worst {
			worst, at = s.drop, i
		}
	}
	if at < 0 {
		return false
	}
	*segs = append((*segs)[:at], (*segs)[at+1:]...)
	return true
}

// minPathCells is the narrowest a path may be drawn at.
//
// Below this it is an ellipsis and two letters, which answers nothing and costs
// the room the segments were dropped to free.
const minPathCells = 14

// withWorkspacePath puts the working directory at the right-hand end, in
// whatever the bar has left.
//
// The worktree segment on the left carries the BASE name, and that is the fast
// answer — until two checkouts share one. `dcode` and `dcode` in different
// parents read identically, and the session that ran in the wrong one looks
// exactly like the session that ran in the right one.
//
// Right-aligned rather than appended, because it is a standing fact about the
// window and not another segment competing in the flow. It is drawn only when
// there is room after everything droppable has already gone: it is the most
// reconstructible thing on the line — `pwd` answers it — so it gives way to
// anything that exists nowhere else.
func withWorkspacePath(left string, m Model, g Geometry) string {
	if strings.TrimSpace(m.Workspace) == "" {
		return left
	}
	used := visibleWidth(left)
	room := g.Width - used - 2 // a cell of air on each side
	if room < minPathCells {
		return left
	}
	path := elideLeft(m.Workspace, room, g.Unicode)
	gap := g.Width - used - visibleWidth(path) - 1
	if gap < 1 {
		return left
	}
	return left + strings.Repeat(" ", gap) + g.Palette.Apply(StyleHint, path)
}

// elideLeft drops the FRONT of a path rather than the back.
//
// The tail is what distinguishes two worktrees; the head is what they have in
// common. `/Users/…/dreibox/dcode` cut from the right becomes `/Users/agui…`,
// which is the half every path on the machine shares.
func elideLeft(s string, w int, unicode bool) string {
	if visibleWidth(s) <= w {
		return s
	}
	mark := "…"
	if !unicode {
		mark = "..."
	}
	keep := w - visibleWidth(mark)
	if keep <= 0 {
		return ""
	}
	r := []rune(s)
	out := ""
	for i := len(r) - 1; i >= 0; i-- {
		next := string(r[i]) + out
		if visibleWidth(next) > keep {
			break
		}
		out = next
	}
	return mark + out
}

// WindowTitle is what the terminal window is called.
//
// The session's name when someone gave it one, its derived title when they did
// not, and where it is running when there is neither. All three answer the
// question a row of identical terminal tabs asks, and the name is the best
// answer because it is the only one a person chose.
//
// Pure over the model, like everything else that draws here: the title is a
// field of the view, not an escape written on the side.
func WindowTitle(m Model) string {
	for _, s := range m.Sessions {
		if s.ID != m.SessionID {
			continue
		}
		if n := strings.TrimSpace(s.Name); n != "" {
			return "dcode · " + n
		}
		if t := strings.TrimSpace(s.Title); t != "" {
			return "dcode · " + t
		}
		break
	}
	if base := workspaceName(m.Workspace); base != "" {
		return "dcode · " + base
	}
	return "dcode"
}

// workspaceName is the short name of a workspace: its last path element, or the
// whole thing when that says nothing.
func workspaceName(workspace string) string {
	name := filepath.Base(strings.TrimRight(workspace, string(filepath.Separator)))
	if name == "" || name == "." || name == string(filepath.Separator) {
		return workspace
	}
	return name
}

// segment is one region of the bar.
type segment struct {
	text string
	// solid marks a segment drawn in reversed amber. The design gives that
	// treatment to structure and to nothing else, so two of them next to each
	// other would say that everything is structure.
	solid bool
	// drop is the order segments are given up in. Zero means never.
	drop int
}

// worktreeSegment says where you are.
//
// Never dropped, and drawn solid. With one workspace it is nearly free
// information; with forty it is the most expensive thing on the screen, and the
// bar is where a person looks to answer it without moving.
func worktreeSegment(m Model, g Geometry) segment {
	name := workspaceName(m.Workspace)
	mark := "⎇"
	if !g.Unicode {
		mark = "wt"
	}
	return segment{text: mark + " " + name, solid: true, drop: 0}
}

// diffSegment is what has changed here, summed from what each tool reported.
//
// Never parsed back out of a tool's text: the numbers are stated once, by the
// tool that knows them, and re-deriving them from prose breaks the day the
// wording changes.
func diffSegment(m Model, g Geometry) (segment, bool) {
	if m.DiffAdded == 0 && m.DiffRemoved == 0 {
		return segment{}, false
	}
	gl := glyphs(g.Unicode)
	files := ""
	if m.DiffFiles > 0 {
		files = fmt.Sprintf(" %s %d %s", gl.dot, m.DiffFiles, Text(m.Lang).BarFiles)
	}
	return segment{
		text: fmt.Sprintf("+%d %s%d%s", m.DiffAdded, gl.minus, m.DiffRemoved, files),
		drop: 1,
	}, true
}

// navSegment is the design's NAV badge and the keys beside it.
//
// Solid, first, and never dropped. It is the one region that says what this
// keyboard can do, and a product whose keys are only documented inside a help
// screen is a product whose keys nobody finds.
//
// The keys it names are the ones that are BINDINGS. The design's footer also
// offers `j/k move` and `t theme`, which are letters, and a letter on a line
// where you type is the defect this product has fixed twice. They belong to a
// mode that owns the keyboard — the design implies one by putting a NAV badge
// there at all — and until that mode exists, naming them here would advertise
// keys that eat what you are typing.
func navSegment(m Model, g Geometry) segment {
	gl := glyphs(g.Unicode)
	t := Text(m.Lang)
	p := g.Palette

	// The badge is LIT while the mode owns the keyboard, and quiet otherwise.
	// It is the only thing on screen that says which of two keyboards a key is
	// about to reach, and that state used to be invisible — which is half of
	// why a letter bound to a mode could eat a keystroke with no visible cause.
	badge, keys := StyleMeta, []string{
		keyHint(p, "esc", t.NavEnter),
		keyHint(p, "^r", t.NavSessions),
		keyHint(p, "^b", t.NavColumn),
		keyHint(p, "?", t.NavKeys),
	}
	if m.Navigating {
		badge = StyleOnAccent
		keys = []string{
			keyHint(p, "j/k", t.NavMove),
			keyHint(p, gl.enter, t.NavOpen),
			keyHint(p, "t", p.theme().Name),
			keyHint(p, "/", t.NavPrompt),
			keyHint(p, "esc", t.NavLeave),
		}
	}
	// Dropped FIRST when the bar runs out of room. Every key it names is
	// reachable from `?`, which makes it the most reconstructible thing on the
	// line — and a bar that keeps its hints by dropping where you are has
	// chosen the hint over the fact.
	return segment{
		text: p.Apply(badge, " "+strings.ToUpper(t.NavBadge)+" ") + " " +
			strings.Join(keys, "  "+gl.dot+"  "),
		drop: 4,
	}
}

// keyHint is one hint: the keystroke, then what it does.
func keyHint(p Palette, stroke, what string) string {
	return p.Apply(StyleProse, stroke) + " " + p.Apply(StyleHint, what)
}

// posSegment is where the cursor is in the stream, as the design's `1 / 7`.
//
// Only while the stream HAS the cursor. On the input line there is no position
// to report, and a counter that reads `0 / 7` when nobody is browsing is a
// number that has to be explained.
func posSegment(m Model, g Geometry) (segment, bool) {
	if m.Cursor < 0 || len(m.Entries) == 0 {
		return segment{}, false
	}
	return segment{
		text: g.Palette.Apply(StyleProse,
			fmt.Sprintf("%d / %d", m.Cursor+1, len(m.Entries))),
		drop: 2,
	}, true
}

// ceilingSegment is the turn against a ceiling it is approaching.
//
// It used to be a section of the plan panel. The panel is gone with the plan
// that justified it, and this is the half of it that exists nowhere else — the
// round count and the in-flight pair, which nothing else on the screen carries.
//
// It appears on the same terms the panel section did: from half the ceiling,
// and whenever every in-flight slot is taken. Far from a ceiling it is a number
// nobody acts on, and a bar of numbers nobody acts on is a bar people stop
// reading.
//
// Never dropped: it is drawn only when it is worth acting on, and a warning
// that gives way to a diff summary is a warning that goes missing at exactly
// the width where the screen is already tight.
func ceilingSegment(m Model, g Geometry) (segment, bool) {
	if !m.turnSectionWorthDrawing() {
		return segment{}, false
	}
	gl := glyphs(g.Unicode)
	t := Text(m.Lang)
	p := g.Palette

	var parts []string
	if m.MaxRounds > 0 {
		style := StyleWarn
		if m.Rounds*4 >= m.MaxRounds*3 {
			style = StyleError
		}
		parts = append(parts, p.Apply(style,
			fmt.Sprintf("%s %d/%d", t.PanelRounds, m.Rounds, m.MaxRounds)))
	}
	if m.MaxInFlight > 0 && m.InFlight >= m.MaxInFlight {
		parts = append(parts, p.Apply(StyleWarn,
			fmt.Sprintf("%s %d%s%d", t.PanelInFlight, m.InFlight, gl.dot, m.MaxInFlight)))
	}
	if len(parts) == 0 {
		return segment{}, false
	}
	return segment{text: strings.Join(parts, " "+gl.dot+" "), drop: 0}, true
}

// waitingSegment is what is blocked on a person.
//
// Solid, undroppable, and absent at zero. The absence is the message: a badge
// that shows nothing waiting is a badge people stop reading, and then it is
// still there on the day something is.
func waitingSegment(m Model, g Geometry) (segment, bool) {
	if m.Pending == nil {
		return segment{}, false
	}
	mark := "◉"
	if !g.Unicode {
		mark = Text(m.Lang).BarWaiting
	}
	return segment{text: mark + " 1", solid: true, drop: 0}, true
}

// modeSegment shows the session behavioural mode (plan, assist, auto).
//
// auto gets the warning style, because the only thing worth highlighting on a
// bar full of facts is "no boundary between the agent and the machine".
// plan and assist stay quiet — they are the ordinary states.
//
// drop: 3, after diff and ceilings. The mode is the second most important
// fact on the bar after "where am I", but a bar that keeps its mode by
// dropping what changed cannot be trusted.
func modeSegment(m Model, g Geometry) (segment, bool) {
	if m.Mode == "" {
		return segment{}, false
	}
	style := StyleMeta
	if m.Mode == protocol.ModeAuto {
		style = StyleWarn
	}
	return segment{
		text: g.Palette.Apply(style, "["+m.Mode+"]"),
		drop: 3,
	}, true
}

// fits reports whether every segment still has room, separators included.
func fits(segs []segment, width int) bool {
	return barWidth(segs) <= width
}

func barWidth(segs []segment) int {
	w := 0
	for i, s := range segs {
		w += visibleWidth(s.text) + 2 // one cell of padding either side
		if i > 0 {
			w++ // the separator between segments
		}
	}
	return w
}

// dropOne gives up the most expendable segment, and reports whether it could.
func dropOne(segs *[]segment) bool {
	worst, at := 0, -1
	for i, s := range *segs {
		if s.drop > worst {
			worst, at = s.drop, i
		}
	}
	if at < 0 {
		// Everything left is undroppable. The worktree is clipped instead,
		// because a truncated answer to "where am I" still answers it.
		return false
	}
	*segs = append((*segs)[:at], (*segs)[at+1:]...)
	return true
}

func assemble(segs []segment, g Geometry) string {
	p := g.Palette
	gl := glyphs(g.Unicode)
	var b strings.Builder
	for i, s := range segs {
		if i > 0 {
			b.WriteString(p.Apply(StyleChrome, gl.gutter))
		}
		body := " " + s.text + " "
		if s.solid {
			b.WriteString(p.Apply(StyleOnAccent, body))
			continue
		}
		b.WriteString(p.Apply(StyleMeta, body))
	}
	// The last resort. Clipping keeps the bar one line when even the
	// undroppable segments do not fit, which a narrow enough terminal
	// guarantees.
	return clipStyled(b.String(), g.Width)
}
