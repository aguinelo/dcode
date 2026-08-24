package tui

import (
	"fmt"
	"strings"
)

// The side column: what changed, and where the session stands.
//
// It replaces the file list, and the difference is what got the file list
// hidden by default in the first place — that column repeated what the stream
// had just said. These two panes do not: a bar of added against removed, and a
// session's own arithmetic, exist nowhere else on the screen.
//
// Two panes stacked rather than one long list, because they answer different
// questions and a reader goes to one or the other. The design splits them in
// half; so does this.

// sidePaneGap is the rule between the two panes.
const sidePaneGap = 1

// renderSide draws the whole column, already clipped, exactly height rows.
func renderSide(m Model, g Geometry, height int) []string {
	gl := glyphs(g.Unicode)
	w := g.railWidth()

	// The diff takes the top half and the session the bottom, and the diff
	// keeps the odd row: it grows with the work, and the session does not.
	bottom := (height - sidePaneGap) / 2
	top := height - sidePaneGap - bottom
	if top < 2 || bottom < 2 {
		// Too short to be two panes. The diff alone, because it is the half
		// that changes while a turn runs.
		return clipTo(renderDiffPane(m, g, w, height), height, w)
	}

	out := clipTo(renderDiffPane(m, g, w, top), top, w)
	out = append(out, g.Palette.Apply(StyleChrome, strings.Repeat(gl.boxH, w)))
	return append(out, clipTo(renderSessionPane(m, g, w, bottom), bottom, w)...)
}

func clipTo(rows []string, height, w int) []string {
	if len(rows) > height {
		rows = rows[:height]
	}
	for len(rows) < height {
		rows = append(rows, "")
	}
	for i := range rows {
		rows[i] = clipStyled(rows[i], w)
	}
	return rows
}

// paneHead is a pane's title row: its key, its name, and one fact.
//
// The key in brackets is the design's, and it is a LABEL rather than a binding:
// pressing `d` on a line where you type is the defect this product has fixed
// twice. It says which pane this is; the nav mode is what will reach it.
func paneHead(gl marks, p Palette, key, name, meta string, w int) string {
	left := p.Apply(StyleMeta, "["+key+"]") + " " + p.Apply(StyleHeading, strings.ToUpper(name))
	if meta == "" {
		return left
	}
	gap := w - visibleWidth(left) - visibleWidth(meta)
	if gap < 1 {
		return left
	}
	return left + strings.Repeat(" ", gap) + p.Apply(StyleMeta, meta)
}

// renderDiffPane is what this turn changed, per file, with the shape of the
// change beside it.
//
// The bar is the reason this pane is not the old file list: `+188` says how
// much, and a bar of added against removed against untouched says WHAT KIND of
// change it was — a rewrite, an addition, a surgical edit — which is the thing
// a reader is actually asking when they glance at a file list.
func renderDiffPane(m Model, g Geometry, w, height int) []string {
	gl := glyphs(g.Unicode)
	p := g.Palette
	t := Text(m.Lang)

	total := ""
	if m.DiffAdded > 0 || m.DiffRemoved > 0 {
		total = p.Apply(StyleOK, fmt.Sprintf("+%d", m.DiffAdded)) + " " +
			p.Apply(StyleError, fmt.Sprintf("%s%d", gl.minus, m.DiffRemoved))
	}
	out := []string{paneHead(gl, p, "d", t.SideDiff, total, w)}

	rows := FileTree(m.Entries)
	scale := 0
	for _, r := range rows {
		if n := r.Added + r.Removed; n > scale {
			scale = n
		}
	}
	if scale > 0 {
		// The denominator, said. A bar whose scale nobody can see is a bar that
		// will be read as a percentage of the file.
		out = append(out, p.Apply(StyleHint,
			fmt.Sprintf("%s %d", t.SideBarScale, scale)))
	}
	for _, r := range rows {
		if len(out) >= height {
			break
		}
		if r.Folder {
			continue
		}
		out = append(out, diffRow(r, scale, gl, p, w)...)
	}
	if len(rows) == 0 {
		out = append(out, p.Apply(StyleMeta, t.SideNothingYet))
	}
	return out
}

// diffRow is one file: what happened to it, its name, its counts, and its bar.
func diffRow(r FileRow, scale int, gl marks, p Palette, w int) []string {
	// The mark is the file's STATE, not whether it was added or modified.
	//
	// The design's column is M/A/D, and that is a fact this client does not
	// have: a tool reports what it changed, never whether the path existed
	// before. Drawing an A for "no removals" was the first version of this, and
	// it labelled every new test file and every append to an old one the same
	// way — a guess dressed as a fact, in a column whose whole job is to be
	// glanceable.
	mark, style := gl.done, StyleOK
	switch r.State {
	case FileFailed:
		mark, style = gl.blocked, StyleError
	case FileWriting, FileReading:
		mark, style = gl.active, StyleWarn
	}

	counts := ""
	if r.Added > 0 || r.Removed > 0 {
		counts = p.Apply(StyleOK, fmt.Sprintf("+%d", r.Added))
		if r.Removed > 0 {
			counts += p.Apply(StyleError, fmt.Sprintf("%s%d", gl.minus, r.Removed))
		}
	}

	name := r.Path[strings.LastIndex(r.Path, "/")+1:]
	room := w - 2 - visibleWidth(counts) - 1
	if room > 1 {
		name = ellipsisTail(name, room, gl.ell)
	}
	head := p.Apply(style, mark) + " " + name
	if counts != "" {
		if gap := w - visibleWidth(head) - visibleWidth(counts); gap >= 1 {
			head += strings.Repeat(" ", gap) + counts
		}
	}
	// No counts, no bar, and no blank row where the bar would have been: a
	// file the turn read rather than changed has nothing to draw a bar of, and
	// an empty row per such file is the pane spending half its height on
	// nothing.
	if b := bar(r.Added, r.Removed, scale, gl, p, w); b != "" {
		return []string{head, b}
	}
	return []string{head}
}

// bar draws one file's change against the largest change in the turn.
//
// The design's bar is added, removed, and the rest of the FILE — which needs
// the file's length, and a tool reports what it changed, never how long the
// thing it changed was. The first version of this used the change as its own
// denominator, so every file with no removals drew a full-width green bar: a
// row of identical bars, each saying "100% of what I did to this file, I did to
// this file".
//
// Against the turn's largest change the bars say something true and useful:
// which files this turn actually went to work on. The denominator is stated in
// the pane rather than assumed, because a bar whose scale nobody can see is a
// bar that will be read as a percentage of something else.
func bar(added, removed, scale int, gl marks, p Palette, w int) string {
	touched := added + removed
	if w < 4 || touched == 0 || scale <= 0 {
		return ""
	}
	span := touched * w / scale
	if span < 1 {
		span = 1
	}
	if span > w {
		span = w
	}
	a := added * span / touched
	if added > 0 && a == 0 {
		a = 1
	}
	r := span - a
	return p.Apply(StyleOK, strings.Repeat(gl.barFull, a)) +
		p.Apply(StyleError, strings.Repeat(gl.barFull, maxInt(0, r))) +
		p.Apply(StyleTrack, strings.Repeat(gl.barTrack, maxInt(0, w-span)))
}

// renderSessionPane is where the session stands: how much room is left, how
// much of what the model asked for was allowed, and what it has been doing.
func renderSessionPane(m Model, g Geometry, w, height int) []string {
	gl := glyphs(g.Unicode)
	p := g.Palette
	t := Text(m.Lang)

	tools := 0
	for _, e := range m.Entries {
		if e.Kind == KindTool {
			tools++
		}
	}
	out := []string{paneHead(gl, p, "s", t.SideSession,
		plural(tools, t.SideToolOne, t.SideToolMany), w)}

	// The context, as a pair and a gauge. The pair because the question a
	// ceiling answers is how much is left, and a share cannot answer it.
	if m.Window > 0 {
		out = append(out,
			keyValue(t.SideContext, fmt.Sprintf("%s / %s",
				humanTokens(m.InputTokens), humanTokens(m.Window)), p, w),
			gauge(m.ContextPct, gl, p, w))
	}
	if m.Asked > 0 {
		out = append(out, keyValue(t.SideAllowed,
			fmt.Sprintf("%d / %d", m.Allowed, m.Asked), p, w))
	}

	// What it has been doing, newest first, by the daemon's clock.
	recent := recentCalls(m.Entries, height-len(out)-2)
	if len(recent) > 0 {
		out = append(out, "", p.Apply(StyleMeta, strings.ToUpper(t.SideRecent)))
		for _, e := range recent {
			out = append(out, recentRow(e, m, p, w))
		}
	}
	return out
}

func keyValue(k, v string, p Palette, w int) string {
	gap := w - visibleWidth(k) - visibleWidth(v)
	if gap < 1 {
		return p.Apply(StyleMeta, k)
	}
	return p.Apply(StyleMeta, k) + strings.Repeat(" ", gap) + p.Apply(StyleProse, v)
}

// gauge is the context as a bar, coloured by how close the ceiling is.
func gauge(pct int, gl marks, p Palette, w int) string {
	if pct < 0 {
		pct = 0
	}
	if pct > 100 {
		pct = 100
	}
	full := pct * w / 100
	// Anything at all shows as something: a gauge that reads empty at 1% says
	// the session has used nothing, which is a different claim.
	if pct > 0 && full == 0 {
		full = 1
	}
	return p.Apply(ContextStyle(pct), strings.Repeat(gl.barFull, full)) +
		p.Apply(StyleTrack, strings.Repeat(gl.barTrack, maxInt(0, w-full)))
}

// recentCalls is the last few tool calls, newest first.
func recentCalls(entries []Entry, n int) []Entry {
	if n < 1 {
		return nil
	}
	var out []Entry
	for i := len(entries) - 1; i >= 0 && len(out) < n; i-- {
		if entries[i].Kind == KindTool && entries[i].Tool != "" {
			out = append(out, entries[i])
		}
	}
	return out
}

func recentRow(e Entry, m Model, p Palette, w int) string {
	when := "  :  "
	if !e.At.IsZero() {
		when = e.At.Format("15:04")
	}
	head := p.Apply(StyleHint, when) + " " + p.Apply(StyleAccent, e.Tool)
	room := w - visibleWidth(head) - 1
	if room < 2 {
		return head
	}
	target := e.Target[strings.LastIndex(e.Target, "/")+1:]
	return head + " " + p.Apply(StyleMeta, ellipsisTail(target, room, glyphs(true).ell))
}

// renderLanes is the legend above the stream: three marks and three words,
// once, at the top.
//
// A key is worth its row when the thing it explains is a CHARACTER whose
// meaning cannot be guessed. The lane gutters are exactly that: `▏` and `╎` say
// nothing on their own, and a reader who has not been told will read them as
// decoration and stop seeing them.
//
// One row, and only when the stream has more than one lane in it — a legend for
// a distinction the screen is not currently making is a row spent on nothing.
func renderLanes(m Model, g Geometry, w int) []string {
	gl := glyphs(g.Unicode)
	p := g.Palette
	t := Text(m.Lang)

	seen := map[Lane]bool{}
	for _, e := range m.Entries {
		seen[laneOf(e)] = true
	}
	if len(seen) < 2 {
		return nil
	}

	parts := []string{
		p.Apply(StyleLaneYou, gl.laneYou) + " " + p.Apply(StyleHint, strings.ToUpper(t.LaneYou)),
		p.Apply(StyleLaneProcess, gl.laneProcess) + " " + p.Apply(StyleHint, strings.ToUpper(t.LaneProcess)),
		p.Apply(StyleLaneAnswer, gl.laneAnswer) + " " + p.Apply(StyleHint, strings.ToUpper(t.LaneAnswer)),
	}
	line := "  " + strings.Join(parts, "   ")
	if visibleWidth(line) > w {
		return nil
	}
	return []string{line, ""}
}
