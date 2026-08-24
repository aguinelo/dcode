package tui

import (
	"fmt"
	"path/filepath"
	"strings"
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
	segs := []segment{worktreeSegment(m, g)}
	if seg, ok := diffSegment(m, g); ok {
		segs = append(segs, seg)
	}
	if seg, ok := waitingSegment(m, g); ok {
		segs = append(segs, seg)
	}

	// Dropped from the right, cheapest first: what a person can reconstruct
	// somewhere else goes before what exists nowhere else. The diff is on the
	// diff of the turn; where you are, and what is blocked on you, are not.
	for {
		if fits(segs, g.Width) || !dropOne(&segs) {
			break
		}
	}
	return assemble(segs, g)
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
	name := filepath.Base(strings.TrimRight(m.Workspace, string(filepath.Separator)))
	if name == "" || name == "." || name == string(filepath.Separator) {
		name = m.Workspace
	}
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
