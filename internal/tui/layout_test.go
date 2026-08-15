package tui

import (
	"strings"
	"testing"

	"github.com/mattn/go-runewidth"
)

// The panel gives ground first on a narrow terminal — it holds short lines and
// the stream holds diffs — and takes a little more on a wide one. Both ends are
// bounded, because a panel that eats the stream and a panel too narrow to read
// are the same bug seen from opposite sides.
func TestThePanelStaysBetweenItsFloorAndItsCeiling(t *testing.T) {
	for _, w := range []int{20, 40, 80, 100, 200, 400} {
		g := Geometry{Width: w}
		got := g.panelWidth()
		if got < 16 {
			t.Errorf("width %d: panel %d, below the readable floor", w, got)
		}
		if got > 34 {
			t.Errorf("width %d: panel %d, above the ceiling", w, got)
		}
		if got >= w && w > 34 {
			t.Errorf("width %d: panel %d takes the whole screen", w, got)
		}
	}

	// Configured bounds win over the defaults, in both directions.
	if got := (Geometry{Width: 400, PanelMaxWidth: 20}).panelWidth(); got != 20 {
		t.Errorf("panel = %d, want the configured ceiling", got)
	}
	if got := (Geometry{Width: 10, PanelMinWidth: 30, PanelWidth: 30, PanelMaxWidth: 40}).panelWidth(); got != 30 {
		t.Errorf("panel = %d, want the configured floor", got)
	}
}

// The end is what identifies a file; the directories leading to it are what
// everything in a repository has in common. So the cut takes the front, and the
// result never comes out wider than the column it had to fit.
func TestATooLongPathIsCutAtTheFrontAndFitsItsColumn(t *testing.T) {
	long := "internal/tui/very/deep/path/render.go"

	got := ellipsis(long, 20)
	if runewidth.StringWidth(got) > 20 {
		t.Errorf("%q is %d wide, want at most 20", got, runewidth.StringWidth(got))
	}
	if !strings.HasSuffix(got, "render.go") {
		t.Errorf("got %q, want the filename kept", got)
	}
	if !strings.HasPrefix(got, "…") {
		t.Errorf("got %q, want the cut marked", got)
	}

	// Already short enough: untouched, and no marker added.
	if got := ellipsis("a.go", 20); got != "a.go" {
		t.Errorf("got %q, want it unchanged", got)
	}
	// A column too narrow to hold even the marker still returns something that
	// fits, rather than a string wider than the space it has. Width zero is not
	// a narrow column but the absence of one, and passes through untouched.
	if got := ellipsis(long, 0); got != long {
		t.Errorf("got %q with no column given, want it unchanged", got)
	}
	for _, w := range []int{1, 2, 3} {
		if got := ellipsis(long, w); runewidth.StringWidth(got) > w {
			t.Errorf("width %d: %q is wider than the column", w, got)
		}
	}
}

// Non-ASCII must be cut by display width, not by byte, or a wide glyph pushes
// the line one column past the box and the frame breaks.
func TestCuttingAPathRespectsWideGlyphs(t *testing.T) {
	got := ellipsis("caminho/muito/comprido/日本語のファイル.go", 20)
	if runewidth.StringWidth(got) > 20 {
		t.Errorf("%q is %d wide, want at most 20", got, runewidth.StringWidth(got))
	}
	if strings.Contains(got, "�") {
		t.Errorf("the path was cut mid-character: %q", got)
	}
}

// The window never hands back a slice outside the body, whatever the scroll
// position says. Every one of these was reachable: a resize shrinks the body
// under a scroll position taken before it, and following pins to the end.
func TestTheWindowNeverReachesOutsideTheBody(t *testing.T) {
	g := DefaultGeometry(80, 24)
	body := make([]string, 8)
	for i := range body {
		body[i] = "line"
	}

	for _, tc := range []struct {
		name string
		m    Model
	}{
		{"scrolled past the end", Model{ScrollTop: 999}},
		{"scrolled before the start", Model{ScrollTop: -50}},
		{"following", Model{Follow: true}},
		{"following an empty body", Model{Follow: true}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			b := body
			if strings.Contains(tc.name, "empty") {
				b = nil
			}
			visible, top, total, height := Window(tc.m, g, b)
			if top < 0 || top > total {
				t.Fatalf("top = %d with %d lines", top, total)
			}
			if len(visible) > height {
				t.Errorf("%d visible lines in a body of %d rows", len(visible), height)
			}
			if total != len(b) {
				t.Errorf("total = %d, want %d", total, len(b))
			}
		})
	}
}

// A body shorter than the window shows all of it and scrolls nowhere: the
// alternative is a screen of blank rows above content the person is looking at.
func TestABodyShorterThanTheWindowShowsAllOfIt(t *testing.T) {
	g := DefaultGeometry(80, 24)
	visible, top, total, _ := Window(Model{Follow: true}, g, []string{"one", "two"})
	if top != 0 || total != 2 || len(visible) != 2 {
		t.Errorf("visible=%d top=%d total=%d, want both lines from the top", len(visible), top, total)
	}
}

// Markdown reaches the screen as text, not as asterisks: the model writes
// markdown, and printing it raw made every answer with emphasis in it look
// unfinished. A fenced block keeps its lines as they were, because re-flowing a
// command produces a second line that looks runnable.
func TestProseIsReadAsMarkdownAndFencesAreLeftAlone(t *testing.T) {
	g := DefaultGeometry(80, 24)
	g.Palette = Palette{Enabled: false}

	out := strings.Join(renderProse("Read **the file** and then `make check`.", 60, g), "\n")
	if strings.Contains(out, "**") || strings.Contains(out, "`") {
		t.Errorf("the markers reached the screen: %q", out)
	}
	if !strings.Contains(out, "the file") || !strings.Contains(out, "make check") {
		t.Errorf("the text did not survive: %q", out)
	}

	fenced := renderProse("before\n```\ngo test ./...\n```\nafter", 60, g)
	joined := strings.Join(fenced, "\n")
	if !strings.Contains(joined, "go test ./...") {
		t.Errorf("the fenced command did not survive: %q", joined)
	}

	// Nothing at all is still one line: returning no lines leaves the caller
	// computing a height of zero for something that occupies a row.
	if got := renderProse("", 60, g); len(got) != 1 {
		t.Errorf("empty prose rendered as %d lines, want one", len(got))
	}
}

// A word wider than the column is broken rather than merely given its own line:
// a path longer than the stream would otherwise wrap in the terminal itself and
// destroy the layout the renderer just computed.
func TestAWordWiderThanTheColumnIsBrokenNotOverflowed(t *testing.T) {
	g := DefaultGeometry(80, 24)
	g.Palette = Palette{Enabled: false}

	long := strings.Repeat("x", 100)
	for _, w := range []int{10, 20, 40} {
		for i, line := range renderProse(long, w, g) {
			if runewidth.StringWidth(line) > w {
				t.Errorf("width %d: line %d is %d wide: %q", w, i, runewidth.StringWidth(line), line)
			}
		}
	}

	// A column with no room at all still answers with a line rather than
	// looping forever trying to break a word into zero-width pieces.
	if got := renderProse("word", 0, g); len(got) != 1 {
		t.Errorf("a zero-width column rendered %d lines", len(got))
	}
}
