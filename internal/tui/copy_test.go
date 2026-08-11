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
