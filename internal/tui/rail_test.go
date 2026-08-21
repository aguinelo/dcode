package tui

import (
	"strings"
	"testing"
)

func railModel(entries ...Entry) Model {
	return Model{Lang: En, Cursor: -1, Entries: entries}
}

func railGeometry(w int) Geometry {
	g := DefaultGeometry(w, 20)
	g.Palette = Palette{}
	return g
}

// A turn that touched nothing gets no column. A column of nothing costs the
// stream twenty characters and tells the reader something is missing.
func TestATurnThatTouchedNothingGetsNoSidebar(t *testing.T) {
	g := railGeometry(140)
	if g.ShowRail(false) {
		t.Error("an empty tree still opened the sidebar")
	}
	if g.StreamWidth(false, false) != 140 {
		t.Errorf("the stream lost width to a sidebar that is not drawn: %d", g.StreamWidth(false, false))
	}
}

// Below a hundred columns it goes, because the stream is what the work is in.
func TestTheSidebarDisappearsOnANarrowTerminal(t *testing.T) {
	if railGeometry(90).ShowRail(true) {
		t.Error("the sidebar survived a 90-column terminal")
	}
	if !railGeometry(120).ShowRail(true) {
		t.Error("the sidebar is missing at 120 columns")
	}
}

// And an explicit choice wins at any width, in BOTH directions. A keypress is
// the user noticing; responsiveness answers the case where they did not.
func TestAnExplicitChoiceWinsAtAnyWidthBothWays(t *testing.T) {
	forced := railGeometry(80)
	forced.RailMode = RailShown
	if !forced.ShowRail(true) {
		t.Error("an explicitly shown sidebar was hidden by the width rule")
	}

	hidden := railGeometry(200)
	hidden.RailMode = RailHidden
	if hidden.ShowRail(true) {
		t.Error("an explicitly hidden sidebar came back on a wide terminal")
	}
}

// Its width is the design's clamp, in cells.
func TestTheSidebarKeepsItsFloorAndItsCeiling(t *testing.T) {
	if got := railGeometry(100).railWidth(); got != 20 {
		t.Errorf("a fifth of 100 is 20, floor is 20; got %d", got)
	}
	if got := railGeometry(400).railWidth(); got != 30 {
		t.Errorf("the ceiling did not hold: %d", got)
	}
}

// The stream pays for both columns, counted once, in the function the layout
// and the renderer both read.
func TestTheStreamPaysForEveryColumnExactlyOnce(t *testing.T) {
	g := railGeometry(140)
	full := g.StreamWidth(true, true)
	if want := 140 - g.railWidth() - 1 - g.panelWidth() - 1; full != want {
		t.Errorf("got %d, want %d", full, want)
	}
	if railOnly := g.StreamWidth(true, false); railOnly != 140-g.railWidth()-1 {
		t.Errorf("the sidebar alone cost %d", 140-railOnly)
	}
}

// Movement is never the only clue. In ASCII a running row, a finished one and a
// failed one still differ by character — a pulse cannot be the difference on a
// terminal where nothing pulses.
func TestTheStatesStayApartWithoutUnicode(t *testing.T) {
	gl := railGlyphs(false)
	seen := map[string]string{}
	for name, mark := range map[string]string{
		"folder": gl.folder, "running": gl.running, "done": gl.done, "failed": gl.failed,
	} {
		if other, clash := seen[mark]; clash {
			t.Errorf("%s and %s are both %q; blocked would read as finished", name, other, mark)
		}
		seen[mark] = name
	}
}

// The sidebar emits no escape without colour, like everything else on screen.
func TestTheSidebarEmitsNoEscapeWithoutColour(t *testing.T) {
	m := railModel(
		Entry{Kind: KindTool, Tool: "write", Target: "a/b.go", Added: 12},
		Entry{Kind: KindTool, Tool: "read", Target: "a/c.go", Running: true},
		Entry{Kind: KindTool, Tool: "edit", Target: "d.go", IsError: true},
	)
	for _, line := range renderRail(m, railGeometry(120), 8) {
		if strings.ContainsRune(line, 0x1b) {
			t.Errorf("an escape survived a monochrome palette: %q", line)
		}
	}
}

// No row is wider than the column, whatever the path.
func TestNoSidebarRowOverflowsTheColumn(t *testing.T) {
	g := railGeometry(120)
	m := railModel(Entry{
		Kind: KindTool, Tool: "write", Added: 123456,
		Target: "internal/very/deeply/nested/directory/with/a/long/name/file.go",
	})
	for _, line := range renderRail(m, g, 6) {
		if w := visibleWidth(line); w > g.railWidth() {
			t.Errorf("a row is %d wide in a %d column: %q", w, g.railWidth(), line)
		}
	}
}

// The header says how much was touched even when the column is too narrow for
// the list, which is what makes a collapsed column still worth a glance.
func TestTheSidebarHeaderCountsWhatWasTouched(t *testing.T) {
	m := railModel(
		Entry{Kind: KindTool, Tool: "read", Target: "a/b.go"},
		Entry{Kind: KindTool, Tool: "read", Target: "a/c.go"},
	)
	head := renderRail(m, railGeometry(160), 5)[0]
	if !strings.Contains(head, "2 touched") {
		t.Errorf("the header does not count the files: %q", head)
	}
	if strings.Contains(head, "2 2") {
		t.Errorf("the count is printed twice: %q", head)
	}
}
