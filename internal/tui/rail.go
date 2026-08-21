package tui

import (
	"fmt"
	"strings"
)

// The sidebar: what this turn has touched, in the order a reader can scan.
//
// Derived from the entries on every draw rather than stored, for the reason
// tree.go gives — one reduction of the log rather than two that can disagree.

// railMarks are the sidebar's glyphs, with an ASCII set that keeps the states
// apart. Movement is never the only clue: a pulsing row and a still one must
// still differ when nothing can pulse, so a running row carries a mark of its
// own rather than only a colour or an animation.
type railMarks struct{ folder, file, running, done, failed string }

func railGlyphs(unicode bool) railMarks {
	if !unicode {
		return railMarks{folder: "+-", file: "|", running: "*", done: "x", failed: "!"}
	}
	return railMarks{folder: "▾", file: "◦", running: "◦", done: "✓", failed: "⊘"}
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
	for len(out) < height {
		out = append(out, "")
	}
	return out
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

	left := indent + p.Apply(style, mark) + " " + r.Label
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
