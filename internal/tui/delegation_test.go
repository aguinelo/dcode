package tui

import (
	"strings"
	"testing"
)

func child(name string, owns []string, running, failed bool, summary string) Entry {
	return Entry{Kind: KindTool, Tool: "explore", Target: name, Owns: owns,
		Running: running, IsError: failed, Summary: summary}
}

func delegationLines(t *testing.T, entries ...Entry) string {
	t.Helper()
	m := Model{Lang: En, Cursor: -1, Entries: entries}
	g := DefaultGeometry(110, 30)
	g.Palette = Palette{}
	return strings.Join(StreamLines(m, g), "\n")
}

// The promise this whole block exists to make visible. delegated-writing says a
// child that never answered is named with its reason, never summarised in with
// the ones that answered — and the client drew a failed explore exactly like any
// other failed call, one line among siblings, where it read as the sixth line
// rather than as the one that matters.
func TestTheChildThatDidNotAnswerIsNamedOnItsOwnLine(t *testing.T) {
	out := delegationLines(t,
		child("alpha", []string{"docs/a"}, false, false, "read 9 · wrote 1"),
		child("bravo", []string{"docs/b"}, false, true, ""),
		child("charlie", []string{"docs/c"}, false, false, "read 4"),
	)
	if !strings.Contains(out, "bravo") {
		t.Errorf("the child that did not answer is not named:\n%s", out)
	}
	for _, line := range strings.Split(out, "\n") {
		if strings.Contains(line, "bravo") && !strings.Contains(line, "no answer") {
			t.Errorf("the reason is not on the child's own line: %q", line)
		}
		if strings.Contains(line, "bravo") && strings.Contains(line, "alpha") {
			t.Errorf("the failure was folded in with a child that answered: %q", line)
		}
	}
}

// And the header counts it, so somebody scrolling past learns that one is
// missing without reading every sub-line.
func TestTheHeaderCountsTheChildrenAndTheOnesMissing(t *testing.T) {
	out := delegationLines(t,
		child("a", nil, false, false, "ok"),
		child("b", nil, false, true, ""),
	)
	if !strings.Contains(out, "2 children") {
		t.Errorf("the header does not count the children:\n%s", out)
	}
	if !strings.Contains(out, "1 no answer") {
		t.Errorf("the header does not count what is missing:\n%s", out)
	}
}

// The boundary each child was given, shown whether or not it came back. A child
// that never answered still declared what it would write, and saying which is
// half of why its refusal is legible.
func TestEveryChildShowsTheBoundaryItWasGiven(t *testing.T) {
	out := delegationLines(t,
		child("alpha", []string{"docs/a", "docs/x"}, false, false, "ok"),
		child("bravo", []string{"docs/b"}, false, true, ""),
	)
	for _, want := range []string{"owns docs/a docs/x", "owns docs/b"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q:\n%s", want, out)
		}
	}
}

// One child is a call, not a delegation. A frame and a count around something
// the ordinary line already says is noise.
func TestASingleExploreIsNotDrawnAsADelegation(t *testing.T) {
	out := delegationLines(t, child("only", []string{"docs/a"}, false, false, "ok"))
	if strings.Contains(out, "1 child") {
		t.Errorf("a lone call was drawn as a delegation:\n%s", out)
	}
}

// Children are drawn once. The header covers them, so the ordinary tool line
// must not draw them again underneath.
func TestChildrenAreNotDrawnTwice(t *testing.T) {
	out := delegationLines(t,
		child("alpha", nil, false, false, "ok"),
		child("bravo", nil, false, false, "ok"),
	)
	if n := strings.Count(out, "alpha"); n != 1 {
		t.Errorf("alpha appears %d times:\n%s", n, out)
	}
}

// What follows a delegation is separated from it, and what precedes it too: it
// is one decision and reads as one block.
func TestADelegationIsSeparatedFromWhatSurroundsIt(t *testing.T) {
	out := delegationLines(t,
		Entry{Kind: KindTool, Tool: "glob", Target: "**/*.go", Summary: "184 files"},
		child("alpha", nil, false, false, "ok"),
		child("bravo", nil, false, false, "ok"),
		Entry{Kind: KindTool, Tool: "edit", Target: "x.go", Summary: "+1 −0"},
	)
	lines := strings.Split(out, "\n")
	for i, l := range lines {
		if strings.Contains(l, "children") {
			if i == 0 || strings.TrimSpace(lines[i-1]) != "" {
				t.Errorf("the delegation is not separated from what came before:\n%s", out)
			}
			return
		}
	}
	t.Fatalf("no delegation header:\n%s", out)
}

// A running child says so rather than looking finished with no output.
func TestARunningChildIsDistinguishableFromAFinishedOne(t *testing.T) {
	out := delegationLines(t,
		child("alpha", nil, false, false, "read 9"),
		child("bravo", nil, true, false, ""),
	)
	for _, line := range strings.Split(out, "\n") {
		if strings.Contains(line, "bravo") && strings.Contains(line, "read 9") {
			t.Errorf("a running child borrowed a finished one's result: %q", line)
		}
	}
}
