package tui

import (
	"strings"
	"testing"
)

// The caret is drawn on the character it sits on, and at the end of the line on
// a space that is not there. A caret drawn as an extra character mid-line would
// push everything after it one column right, which reads as the text moving
// while it is being typed.
func TestTheCaretSitsOnACharacterWithoutMovingTheRest(t *testing.T) {
	p := Palette{Enabled: false}

	if got := renderCaretIn("abc", 1, p); got != "abc" {
		t.Errorf("got %q, want the line unchanged with no palette", got)
	}
	// At the end there is nothing under it, so a space is added — and that is
	// the one case where the line does get longer.
	if got := renderCaretIn("abc", 3, p); got != "abc " {
		t.Errorf("got %q, want a space for the caret past the last character", got)
	}
	// A caret outside the line is clamped rather than indexing out of it.
	if got := renderCaretIn("abc", -5, p); got != "abc" {
		t.Errorf("got %q from a negative position", got)
	}
	if got := renderCaretIn("abc", 99, p); got != "abc " {
		t.Errorf("got %q from a position past the end", got)
	}
	// An empty line still shows where typing would go.
	if got := renderCaretIn("", 0, p); got != " " {
		t.Errorf("got %q for an empty line", got)
	}
}

// The modal is painted over the screen it takes, and every row it covers is
// replaced whole — a frame that leaves part of the row underneath is the
// ghosting this codebase already fixed once.
func TestTheModalCoversTheRowsItPaintsAndKeepsTheScreensShape(t *testing.T) {
	g := Geometry{Width: 20}
	screen := strings.Repeat("xxxxxxxxxxxxxxxxxxxx\n", 10)

	got := overlay(screen, []string{"one", "two"}, g)
	rows := strings.Split(strings.TrimRight(got, "\n"), "\n")
	if len(rows) != 10 {
		t.Fatalf("the screen is %d rows after the modal, want 10", len(rows))
	}
	joined := strings.Join(rows, "\n")
	if !strings.Contains(joined, "one") || !strings.Contains(joined, "two") {
		t.Errorf("the modal did not land: %q", joined)
	}
	for i, r := range rows {
		if len([]rune(r)) > g.Width {
			t.Errorf("row %d is %d wide, past the frame: %q", i, len([]rune(r)), r)
		}
	}

	// A modal taller than the screen paints what fits and stops, rather than
	// indexing past the last row.
	tall := make([]string, 40)
	for i := range tall {
		tall[i] = "line"
	}
	out := overlay(screen, tall, g)
	if n := strings.Count(strings.TrimRight(out, "\n"), "\n") + 1; n != 10 {
		t.Errorf("the screen grew to %d rows under a taller modal", n)
	}

	// A modal wider than the frame is cut rather than pushing the frame out.
	wide := overlay(screen, []string{strings.Repeat("w", 60)}, g)
	for _, r := range strings.Split(strings.TrimRight(wide, "\n"), "\n") {
		if len([]rune(r)) > g.Width {
			t.Errorf("a wide modal pushed a row to %d columns", len([]rune(r)))
		}
	}
}
