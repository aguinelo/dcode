package tui

import (
	"strings"
	"testing"
)

func toolEntry(tool, target, diff string) Entry {
	return Entry{Kind: KindTool, Tool: tool, Target: target, Summary: "done", Diff: diff}
}

func streamOf(t *testing.T, entries ...Entry) []string {
	t.Helper()
	m := Model{Lang: En, Entries: entries, Cursor: -1}
	g := DefaultGeometry(100, 40)
	g.Palette = Palette{}
	return StreamLines(m, g)
}

// A call that carries a body reads as a block. The gutter already ties the body
// to its header; what was missing was the room around it.
func TestACallWithABodyIsSeparatedFromWhatCameBefore(t *testing.T) {
	lines := streamOf(t,
		toolEntry("read", "a.go", ""),
		toolEntry("edit", "b.go", "--- a/b.go\n+++ b/b.go\n-old\n+new\n"),
	)
	head := -1
	for i, l := range lines {
		if strings.Contains(l, "edit") {
			head = i
			break
		}
	}
	if head <= 0 {
		t.Fatalf("the edit header is not where expected:\n%s", strings.Join(lines, "\n"))
	}
	if strings.TrimSpace(lines[head-1]) != "" {
		t.Errorf("no blank line before a call with a body:\n%s", strings.Join(lines, "\n"))
	}
}

// And a call with no body stays a single line. Most calls are one line, and a
// stream that puts air around every one of them stops fitting on a screen.
func TestCallsWithoutBodiesStayPacked(t *testing.T) {
	lines := streamOf(t,
		toolEntry("read", "a.go", ""),
		toolEntry("read", "b.go", ""),
		toolEntry("read", "c.go", ""),
	)
	for _, l := range lines {
		if strings.TrimSpace(l) == "" {
			t.Errorf("bodyless calls were spaced out:\n%s", strings.Join(lines, "\n"))
		}
	}
}

// Never two. A block following a block would otherwise open with the gap the
// last one closed with, and double spacing reads as something missing.
func TestTwoBlocksAreSeparatedByExactlyOneBlankLine(t *testing.T) {
	diff := "--- a/x\n+++ b/x\n-a\n+b\n"
	lines := streamOf(t, toolEntry("edit", "x.go", diff), toolEntry("edit", "y.go", diff))
	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "" && strings.TrimSpace(lines[i-1]) == "" {
			t.Errorf("two blank lines in a row at %d:\n%s", i, strings.Join(lines, "\n"))
		}
	}
}

// The gap goes before, never after — measured, not guessed.
//
// A trailing blank costs a row of the window the stream is anchored to, so the
// newest thing scrolls off to make room for nothing. Put after, it dropped the
// changed line out of a diff on a 40-row terminal.
func TestTheStreamDoesNotEndOnABlankLine(t *testing.T) {
	lines := streamOf(t, toolEntry("edit", "x.go", "--- a/x\n+++ b/x\n-a\n+b\n"))
	if n := len(lines); n > 0 && strings.TrimSpace(lines[n-1]) == "" {
		t.Errorf("the stream ends on a blank line, wasting a row of the window:\n%s",
			strings.Join(lines, "\n"))
	}
}

// What comes after a block is separated from it too, or the body runs straight
// into the next thing and the block loses its bottom edge.
func TestWhatFollowsABlockIsSeparatedFromIt(t *testing.T) {
	lines := streamOf(t,
		toolEntry("edit", "x.go", "--- a/x\n+++ b/x\n-a\n+b\n"),
		toolEntry("read", "after.go", ""),
	)
	at := -1
	for i, l := range lines {
		if strings.Contains(l, "after.go") {
			at = i
			break
		}
	}
	if at <= 0 {
		t.Fatalf("the following call is missing:\n%s", strings.Join(lines, "\n"))
	}
	if strings.TrimSpace(lines[at-1]) != "" {
		t.Errorf("the body ran straight into what came after:\n%s", strings.Join(lines, "\n"))
	}
}

// The hint under a collapsed body said "Tab expande" in Portuguese beside a
// count in English — one line, two languages, in BOTH interfaces. Neither half
// followed what the user had chosen.
func TestTheExpansionHintSpeaksTheInterfaceLanguage(t *testing.T) {
	long := strings.Repeat("a line of output\n", 60)
	for _, c := range []struct {
		lang       Lang
		want, gone string
	}{
		{PtBR, "Tab expande", "lines"},
		{En, "Tab expands", "linhas"},
	} {
		m := Model{Lang: c.lang, Cursor: -1, Entries: []Entry{
			{Kind: KindTool, Tool: "bash", Target: "x", Summary: "ok", Detail: long, Expanded: true},
		}}
		g := DefaultGeometry(100, 40)
		g.Palette = Palette{}
		out := strings.Join(StreamLines(m, g), "\n")
		if !strings.Contains(out, c.want) {
			t.Errorf("%s: the hint is not in the interface language:\n%s", c.lang, out)
		}
		if strings.Contains(out, c.gone) {
			t.Errorf("%s: the other language leaked into the hint:\n%s", c.lang, out)
		}
	}
}
