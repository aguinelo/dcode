package tui

import (
	"encoding/base64"
	"strings"
	"testing"
)

// The anchor stays put so dragging back past the start shrinks the selection
// rather than inverting it. An anchor that moves is how a selection ends up off
// by one at one end.
func TestTheSelectionExtendsInBothDirectionsFromItsAnchor(t *testing.T) {
	m := Model{Cursor: 5}.EnterCopy(9)
	if lo, hi := m.Copy.Range(); lo != 5 || hi != 5 {
		t.Fatalf("a fresh selection is %d..%d, want a single line", lo, hi)
	}

	m = m.ExtendCopy(3, 9)
	if lo, hi := m.Copy.Range(); lo != 5 || hi != 8 {
		t.Errorf("extending down gave %d..%d, want 5..8", lo, hi)
	}

	m = m.ExtendCopy(-6, 9)
	if lo, hi := m.Copy.Range(); lo != 2 || hi != 5 {
		t.Errorf("dragging back past the anchor gave %d..%d, want 2..5 — the anchor must not move", lo, hi)
	}
}

func TestTheSelectionCannotLeaveTheStream(t *testing.T) {
	m := Model{Cursor: 0}.EnterCopy(3)
	m = m.ExtendCopy(-10, 3)
	if lo, _ := m.Copy.Range(); lo != 0 {
		t.Errorf("selection went above the first line: %d", lo)
	}
	m = m.ExtendCopy(99, 3)
	if _, hi := m.Copy.Range(); hi != 3 {
		t.Errorf("selection went past the last line: %d", hi)
	}
}

// What goes to the clipboard is what the person meant to copy, not the cursor
// marks and colour around it. Pasting a diff with a gutter of escapes into an
// issue is the failure this avoids.
func TestCopiedTextCarriesNoDecoration(t *testing.T) {
	lines := []string{
		"\x1b[1m> the question\x1b[0m",
		"\x1b[2m  read internal/config/toml.go\x1b[0m   ",
		"plain line",
	}
	got := CopyText(lines, CopyState{Active: true, Anchor: 0, Head: 2})
	if strings.Contains(got, "\x1b") {
		t.Fatalf("escapes reached the clipboard: %q", got)
	}
	want := "> the question\n  read internal/config/toml.go\nplain line"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestCopyingNothingProducesNothing(t *testing.T) {
	if got := CopyText([]string{"a"}, CopyState{}); got != "" {
		t.Errorf("an inactive selection produced %q", got)
	}
	if got := CopyText(nil, CopyState{Active: true}); got != "" {
		t.Errorf("an empty stream produced %q", got)
	}
}

func TestASelectionPastTheEndIsClamped(t *testing.T) {
	got := CopyText([]string{"a", "b"}, CopyState{Active: true, Anchor: 0, Head: 99})
	if got != "a\nb" {
		t.Fatalf("got %q", got)
	}
}

// OSC 52 rather than pbcopy or xclip, because the terminal the person is
// looking at is not always on the machine dcode runs on, and a clipboard that
// only works locally fails exactly when it is most wanted.
func TestTheClipboardSequenceCarriesTheTextEncoded(t *testing.T) {
	const text = "line one\nline two"
	seq := OSC52(text)
	if !strings.HasPrefix(seq, "\x1b]52;c;") || !strings.HasSuffix(seq, "\x07") {
		t.Fatalf("not an OSC 52 sequence: %q", seq)
	}
	payload := strings.TrimSuffix(strings.TrimPrefix(seq, "\x1b]52;c;"), "\x07")
	got, err := base64.StdEncoding.DecodeString(payload)
	if err != nil {
		t.Fatalf("payload is not base64: %v", err)
	}
	if string(got) != text {
		t.Errorf("decoded %q, want %q", got, text)
	}
}

func TestContainsAnswersForEachLine(t *testing.T) {
	c := CopyState{Active: true, Anchor: 4, Head: 2}
	for i, want := range map[int]bool{1: false, 2: true, 3: true, 4: true, 5: false} {
		if got := c.Contains(i); got != want {
			t.Errorf("Contains(%d) = %v, want %v", i, got, want)
		}
	}
	if (CopyState{}).Contains(0) {
		t.Error("an inactive selection contains a line")
	}
}

// A mode with no visible way out is a mode people force-quit the program to
// escape.
func TestTheHintNamesEveryKeyAndTheCount(t *testing.T) {
	for _, lang := range Languages() {
		h := CopyHint(CopyState{Active: true, Anchor: 2, Head: 5}, lang)
		if !strings.Contains(h, "4") {
			t.Errorf("%s: the hint does not say how many lines are selected: %s", lang, h)
		}
		for _, key := range []string{"y", "esc"} {
			if !strings.Contains(h, key) {
				t.Errorf("%s: the hint does not name %q: %s", lang, key, h)
			}
		}
	}
}

// Copy mode existed and showed nothing. `Contains` was written to say which
// rendered lines are inside the selection, and had no caller: the user pressed
// keys, a selection moved, and the screen looked identical.
//
// That is worse than not having copy mode. Without it the answer is "select
// with the mouse, it does not work here"; with an invisible selection the answer
// is "press some keys and hope", and the only way to find out what was taken is
// to paste it somewhere else.
func TestCopyModeShowsWhichLinesAreSelected(t *testing.T) {
	m := NewModel("s1", "/w", "m", "workspace-write", En)
	for i := 0; i < 6; i++ {
		m.Entries = append(m.Entries, Entry{Kind: KindAssistant, Summary: "LINE-" + string(rune('A'+i))})
	}
	g := DefaultGeometry(100, 30)

	plain := Render(m, g)

	m.Copy = CopyState{Active: true, Anchor: 0, Head: 1}
	selected := Render(m, g)

	if selected == plain {
		t.Fatal("opening a selection changed nothing on screen; the user is dragging " +
			"something they cannot see")
	}
	// Moving the selection has to move what is highlighted, or the first
	// difference was just the mode indicator.
	m.Copy = CopyState{Active: true, Anchor: 3, Head: 4}
	moved := Render(m, g)
	if moved == selected {
		t.Error("moving the selection changed nothing; only the mode is being shown")
	}
}

// The highlight is decoration, and decoration must not reach the clipboard.
// What is pasted is what the person meant to copy, not what the terminal drew
// around it.
func TestTheHighlightNeverReachesTheClipboard(t *testing.T) {
	lines := []string{"first line", "second line", "third line"}
	got := CopyText(lines, CopyState{Active: true, Anchor: 0, Head: 1})
	if got != "first line\nsecond line" {
		t.Errorf("copied %q", got)
	}
	if strings.Contains(got, "\x1b") {
		t.Errorf("an escape sequence reached the clipboard: %q", got)
	}
}

// A monochrome terminal has no colour to highlight with, so the selection has
// to be visible some other way. Marking it only by colour means copy mode does
// not work at all where colour is off.
func TestTheSelectionIsVisibleWithoutColour(t *testing.T) {
	m := NewModel("s1", "/w", "m", "workspace-write", En)
	for i := 0; i < 4; i++ {
		m.Entries = append(m.Entries, Entry{Kind: KindAssistant, Summary: "LINE"})
	}
	g := DefaultGeometry(100, 30)
	g.Palette = Palette{} // no colour at all

	// Both renders are IN copy mode, differing only in which lines are chosen.
	// Comparing against copy mode being closed would pass on the gutter merely
	// existing, which says nothing about whether the selection can be seen.
	m.Copy = CopyState{Active: true, Anchor: 0, Head: 0}
	first := Render(m, g)
	m.Copy = CopyState{Active: true, Anchor: 2, Head: 2}
	second := Render(m, g)

	if first == second {
		t.Error("with colour off the selection is invisible; copy mode would not work " +
			"on a terminal that has none")
	}
}

// The gutter borrows a cell rather than adding one. Adding would push every
// line one past the terminal, wrapping it — and the stable layout the alternate
// screen buys is exactly what copy mode exists to compensate for.
func TestTheGutterNeverPushesALinePastTheTerminal(t *testing.T) {
	m := modelWithPlan()
	m.Entries = append(m.Entries, Entry{
		Kind: KindTool, Tool: "bash", Target: strings.Repeat("long-path/", 20),
		Summary: strings.Repeat("output ", 30), Expanded: true,
		Detail: strings.Repeat("detail line\n", 5),
	})
	m.Copy = CopyState{Active: true, Anchor: 1, Head: 3}

	for _, w := range []int{40, 60, 80, 100, 200} {
		for _, unicode := range []bool{true, false} {
			g := DefaultGeometry(w, 24)
			g.Unicode = unicode
			out := Render(m, g)
			for _, line := range strings.Split(out, "\n") {
				if n := visibleWidth(line); n > w {
					t.Fatalf("width %d, unicode %v: a line is %d cells wide:\n%q", w, unicode, n, line)
				}
			}
		}
	}
}
