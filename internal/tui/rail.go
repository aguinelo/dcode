package tui

import (
	"fmt"
	"strings"

	"github.com/mattn/go-runewidth"
)

// The sidebar: what this turn has touched, in the order a reader can scan.
//
// Derived from the entries on every draw rather than stored, for the reason
// tree.go gives — one reduction of the log rather than two that can disagree.

// railMarks are the sidebar's glyphs, with an ASCII set that keeps the states
// apart. Movement is never the only clue: a pulsing row and a still one must
// still differ when nothing can pulse, so a running row carries a mark of its
// own rather than only a colour or an animation.
type railMarks struct{ folder, file, running, done, failed, open, cursor, caret, ell, dot string }

func railGlyphs(unicode bool) railMarks {
	if !unicode {
		return railMarks{folder: "+-", file: "|", running: "*", done: "x", failed: "!", open: ">", cursor: "*", caret: "_", ell: "...", dot: "-"}
	}
	return railMarks{folder: "▾", file: "◦", running: "◦", done: "✓", failed: "⊘", open: "●", cursor: "▸", caret: "▌", ell: "…", dot: "·"}
}

// renderRail draws the sidebar, one row per line, already clipped to width.
func renderRail(m Model, g Geometry, height int) []string {
	p := g.Palette
	gl := railGlyphs(g.Unicode)
	w := g.railWidth()
	rows := FileTree(m.Entries)

	out := make([]string, 0, height)

	touched := 0
	for _, r := range rows {
		if !r.Folder {
			touched++
		}
	}
	// The label in caps, the count as it is written. Shouting a number reads as
	// emphasis on the number, and what is being labelled is the column.
	t := Text(m.Lang)
	head := fmt.Sprintf("%s %s", strings.ToUpper(t.RailFiles),
		plural(touched, t.RailTouchedOne, t.RailTouchedMany))
	out = append(out, clipStyled(p.Apply(StyleDim, head), w))

	for _, r := range rows {
		if len(out) >= height {
			break
		}
		out = append(out, clipStyled(railRow(r, gl, p, w), w))
	}

	// The conversations of this workspace are NOT here any more.
	//
	// They were a second section of this column, twenty-six characters wide and
	// permanently resident, holding a list somebody opens once in an afternoon.
	// On a real session the four rows read `Write DCODE.md at the r…` four
	// times: the titles derive from the first question, the four conversations
	// began with the same one, and the cut fell in the same place. Nothing
	// there told them apart.
	//
	// `^R` in readline is a SUMMONED search — it appears, you choose, it goes.
	// Borrowing the key and then making it a permanent column contradicted the
	// convention the borrowing was justified by. The list is an overlay now,
	// which is what the key already meant.
	//
	// RailNav is untouched by the move: the cursor, the filter, the naming mode
	// and every rule they carry are the same, and so are their tests. What
	// changed is where the list is drawn.

	for len(out) < height {
		out = append(out, "")
	}
	return out
}

// sessionRow is one recorded conversation. The open one is marked by a
// character and not only by colour, so it is still the open one on a terminal
// without any.
func sessionRow(c SessionChoice, open, under bool, gl railMarks, p Palette, w int) string {
	mark, style := " ", StyleDim
	if open {
		mark, style = gl.open, StyleAccent
	}
	// The cursor is a CHARACTER, never only a colour. A row picked out in
	// colour alone is not picked out at all on a terminal without any, and this
	// is a list where choosing the wrong row opens the wrong afternoon's work.
	//
	// It wins over the open mark, because while the rail has the keyboard the
	// question on screen is "which one am I about to open", not "which one am I
	// in" — and the open one still carries its own colour.
	if under {
		mark = gl.cursor
		if !open {
			style = StyleAccent
		}
	}
	// A name a person gave beats the one derived from the first question, and
	// says which it is. Without the mark, a listing shows two kinds of claim in
	// one column and nothing tells them apart.
	title, given := c.Title, false
	if c.Name != "" {
		title, given = c.Name, true
	}
	if title == "" {
		title = c.ID
	}
	if given {
		// Trimmed to leave room for the mark, so the mark never pushes the
		// name off its own row.
		if room := w - 4; room > 1 && visibleWidth(title) > room {
			title = trimTo(title, room-visibleWidth(gl.ell)) + gl.ell
		}
		title += " " + gl.dot
	}
	// Cut with a mark, never silently. A title that merely stops leaves the
	// reader unable to tell a short conversation from a truncated one, and the
	// start is what identifies it — so the end is what gives way.
	if room := w - 2; room > 1 && visibleWidth(title) > room {
		title = trimTo(title, room-visibleWidth(gl.ell)) + gl.ell
	}
	return p.Apply(style, mark) + " " + title
}

// namingRow is the row while a name is being typed.
//
// It replaces the row rather than sitting above it: the thing being named has
// to be the thing under the cursor, and a field somewhere else makes that a
// guess.
func namingRow(draft string, gl railMarks, p Palette, w int) string {
	room := w - 3
	shown := draft
	if room > 1 && visibleWidth(shown)+1 > room {
		// The END is kept while typing, because the end is where the caret is.
		r := []rune(shown)
		for len(r) > 0 && visibleWidth(string(r))+1 > room {
			r = r[1:]
		}
		shown = string(r)
	}
	return p.Apply(StyleAccent, gl.cursor) + " " + p.Apply(StyleAccent, shown+gl.caret)
}

// trimTo cuts to a width in cells, never in bytes: a rune is not a column, and
// a path or a title with an accent in it is where that goes wrong.
func trimTo(s string, w int) string {
	if w <= 0 {
		return ""
	}
	out, used := make([]rune, 0, len(s)), 0
	for _, r := range s {
		rw := runewidth.RuneWidth(r)
		if used+rw > w {
			break
		}
		out = append(out, r)
		used += rw
	}
	return string(out)
}

func railRow(r FileRow, gl railMarks, p Palette, w int) string {
	// Two cells per level, which is the design's eleven pixels in the unit a
	// terminal actually has.
	indent := strings.Repeat("  ", r.Depth)

	mark, style := gl.file, StyleDim
	switch {
	case r.Folder:
		mark, style = gl.folder, StyleDim
	case r.State == FileFailed:
		mark, style = gl.failed, StyleError
	case r.State == FileDone:
		mark, style = gl.done, StyleOK
	case r.State == FileWriting, r.State == FileReading:
		mark, style = gl.running, StyleAccent
	}

	meta := ""
	if !r.Folder && r.State == FileDone && r.Added > 0 {
		meta = fmt.Sprintf("+%d", r.Added)
	}

	// The label gives way with a mark, the way the conversation titles below
	// it already do. A name that merely stops reads as a short name, and in a
	// column of paths that is the difference between `client.py` and
	// `client.pyi` — one of which exists.
	//
	// The END gives way even though this is a path, because the sidebar has
	// already put the directory on its own row: what is left here is the name,
	// and a name is identified by how it starts.
	label := r.Label
	if room := w - visibleWidth(indent) - 2 - visibleWidth(meta); room > 1 {
		label = ellipsisTail(label, room, gl.ell)
	}

	left := indent + p.Apply(style, mark) + " " + label
	if meta == "" {
		return left
	}
	// The count sits at the right edge and gives way before the name does: a
	// path that has lost its end identifies nothing. One column of gutter stays
	// before the divider, so the count is not read as part of the frame.
	room := w - 1 - visibleWidth(left) - visibleWidth(meta)
	if room < 1 {
		return left
	}
	return left + strings.Repeat(" ", room) + p.Apply(StyleDim, meta)
}
