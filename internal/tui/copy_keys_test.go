package tui

import (
	"fmt"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

// enterCopy opens copy mode the way a person does: with the chord.
func enterCopy(t *testing.T, p *program) {
	t.Helper()
	p.model.Entries = []Entry{
		{Kind: KindUser, Summary: "first"},
		{Kind: KindAssistant, Summary: "second"},
		{Kind: KindAssistant, Summary: "third"},
	}
	p.model.Cursor = len(p.model.Entries) - 1
	p.Update(ctrl('o'))
	if !p.model.Copy.Active {
		t.Fatal("^O did not open copy mode")
	}
}

// Copy mode owns the keyboard while it is open, and nothing else does.
//
// It did not. The whole block handling these keys sat inside
// `if len(p.model.Completions) > 0`, so it ran only while the completion menu
// was open — which is the one moment copy mode cannot be, because opening it
// requires an empty input line and the menu only appears once something is
// typed. In every real use the keys fell straight through to the stream
// bindings underneath.
//
// The comment above that block asserted the invariant this test now checks. It
// is the shape this project keeps finding: something declared that one side
// reads and no side writes.
func TestCopyModeOwnsTheKeyboardWhileItIsOpen(t *testing.T) {
	p, _ := newProgram(t)
	enterCopy(t, p)
	anchor := p.model.Copy.Anchor

	// Upwards first: copy mode opens on the last line when no entry is under
	// the cursor, so that is the direction with room to move.
	p.Update(special(tea.KeyUp))
	if p.model.Copy.Head == anchor {
		t.Fatalf("up did not extend the selection: %+v", p.model.Copy)
	}
	if p.model.Copy.Anchor != anchor {
		t.Errorf("the anchor moved to %d; dragging must shrink, not invert", p.model.Copy.Anchor)
	}

	extended := p.model.Copy.Head
	p.Update(special(tea.KeyDown))
	if p.model.Copy.Head == extended {
		t.Errorf("down did not move the head back: %+v", p.model.Copy)
	}
}

// The stream bindings underneath must not also fire. A key that both extends a
// selection and scrolls the stream moves the ground under what is being
// selected.
func TestTheStreamDoesNotMoveWhileASelectionIsBeingMade(t *testing.T) {
	p, _ := newProgram(t)
	enterCopy(t, p)
	cursor, scroll := p.model.Cursor, p.model.ScrollTop

	p.Update(special(tea.KeyDown))
	p.Update(special(tea.KeyUp))

	if p.model.Cursor != cursor {
		t.Errorf("the cursor moved from %d to %d while selecting", cursor, p.model.Cursor)
	}
	if p.model.ScrollTop != scroll {
		t.Errorf("the stream scrolled from %d to %d while selecting", scroll, p.model.ScrollTop)
	}
}

// Copying writes to the terminal's own clipboard through OSC 52, which is what
// reaches it over ssh and inside tmux, and then leaves the mode. A mode that
// stays open after the copy is one people press escape at, wondering whether it
// worked.
func TestCopyingSendsTheSelectionAndLeavesTheMode(t *testing.T) {
	p, _ := newProgram(t)
	enterCopy(t, p)
	p.Update(special(tea.KeyDown))

	_, cmd := p.Update(key("y"))
	if cmd == nil {
		t.Fatal("copying produced no command; nothing reached the terminal")
	}
	if p.model.Copy.Active {
		t.Error("copy mode stayed open after copying")
	}
	if p.model.Flash == "" {
		t.Error("copying said nothing; the person cannot tell whether it worked")
	}

	// The command carries the selection as an OSC 52 sequence rather than the
	// raw text, because the clipboard is the terminal's and not the program's.
	// The message type bubbletea returns for a printed line is unexported, so
	// this reads its rendering rather than its shape. What matters is the bytes
	// that reach the terminal, and those are visible either way.
	printed := fmt.Sprintf("%v", cmd())
	if !strings.Contains(printed, "\x1b]52;c;") {
		t.Errorf("what was written is not an OSC 52 sequence: %q", printed)
	}
}

// Escape leaves without copying. A mode with no visible way out is a mode
// people force-quit the program to escape, and `q` and ctrl+c mean the same
// thing here.
func TestEveryWayOutOfCopyModeWorks(t *testing.T) {
	for _, k := range []tea.KeyPressMsg{special(tea.KeyEscape), key("q"), ctrl('c')} {
		p, _ := newProgram(t)
		enterCopy(t, p)
		p.Update(k)
		if p.model.Copy.Active {
			t.Errorf("%s did not leave copy mode", k.String())
		}
	}
}

// Every other key is swallowed while copying. Typing a letter mid-selection
// must not put it in the input line, waiting to be sent when the person presses
// enter to copy.
func TestTypingWhileCopyingDoesNotReachTheInputLine(t *testing.T) {
	p, _ := newProgram(t)
	enterCopy(t, p)

	p.Update(key("a"))
	p.Update(key("b"))

	if p.model.Input != "" {
		t.Errorf("input = %q; keys leaked through copy mode", p.model.Input)
	}
	if !p.model.Copy.Active {
		t.Error("an ordinary letter closed copy mode")
	}
}

// Copying an empty selection says so rather than writing an empty clipboard.
// Silently replacing what somebody had copied before with nothing is worse than
// refusing.
func TestCopyingNothingSaysSoRatherThanEmptyingTheClipboard(t *testing.T) {
	p, _ := newProgram(t)
	// Copy mode open over nothing. Set directly rather than through the key,
	// because what this measures is the empty selection, not the binding.
	p.model.Entries = nil
	p.model.Copy = CopyState{Active: true}

	_, cmd := p.Update(key("y"))
	if cmd != nil {
		t.Error("an empty selection still wrote to the clipboard")
	}
	if p.model.Flash == "" {
		t.Error("copying nothing said nothing")
	}
	if p.model.Copy.Active {
		t.Error("copy mode stayed open")
	}
}

// A letter people type is not a shortcut.
//
// `v` opened copy mode whenever the input line was empty — which is where every
// message starts. The first character of anything beginning with v was eaten:
// "voce" arrived as "oce". Reported from use.
func TestVIsALetterWhileTyping(t *testing.T) {
	p, _ := newProgram(t)
	p.model.Entries = []Entry{{Kind: KindUser, Summary: "something"}}
	p.model.Cursor = -1 // the input line has focus, which is where typing happens

	p.Update(key("v"))
	if p.model.Copy.Active {
		t.Fatal("typing v opened copy mode")
	}
	if p.model.Input != "v" {
		t.Errorf("input = %q, want the letter that was typed", p.model.Input)
	}

	for _, r := range "oce" {
		p.Update(key(string(r)))
	}
	if p.model.Input != "voce" {
		t.Errorf("input = %q, want the whole word", p.model.Input)
	}
}

// Copy mode is a CHORD, and this test asserts what its two predecessors could
// not: `v` is a letter, always, wherever the cursor happens to be.
//
// The first version of this bound copy mode to a bare `v` on an empty line, and
// "voce" arrived as "oce". The fix required the stream cursor to be in the
// stream — which narrowed the rule rather than applying it, and the same report
// came back, by the path this test walks: `↑` on a session with no history
// walks into the stream, and the next `v` typed there was a shortcut again.
func TestVIsALetterWhereverTheCursorIs(t *testing.T) {
	for _, start := range []int{-1, 0} {
		p, _ := newProgram(t)
		p.model.Entries = []Entry{{Kind: KindUser, Summary: "something"}}
		p.model.Cursor = start

		for _, r := range "voce" {
			p.Update(key(string(r)))
		}
		if p.model.Copy.Active {
			t.Errorf("cursor=%d: typing opened copy mode", start)
		}
		if p.model.Input != "voce" {
			t.Errorf("cursor=%d: input = %q, want the whole word", start, p.model.Input)
		}
	}
}

// And the path the report came in by, walked end to end: a fresh session with
// no history, one press of up, then a message that starts with a v.
func TestUpThenTypingKeepsEveryLetter(t *testing.T) {
	p, _ := newProgram(t)
	p.model.Entries = []Entry{{Kind: KindUser, Summary: "algo"}, {Kind: KindAssistant, Summary: "resposta"}}
	p.model.Cursor = -1

	p.Update(special(tea.KeyUp))
	if p.model.Cursor < 0 {
		t.Fatal("up did not walk into the stream; the fixture no longer reproduces the report")
	}
	for _, r := range "voce" {
		p.Update(key(string(r)))
	}
	if p.model.Input != "voce" {
		t.Errorf("input = %q, want the whole word", p.model.Input)
	}
	// And typing put the focus back where the typing goes: browsing and
	// writing were two states at once, with nothing on screen saying which.
	if p.model.Cursor >= 0 {
		t.Errorf("the stream kept the cursor at %d while the line was typed on", p.model.Cursor)
	}
}

// The chord opens it, and it does not need the stream to have focus first:
// requiring that was half of what made the letter ambiguous.
func TestTheChordOpensCopyMode(t *testing.T) {
	for _, start := range []int{-1, 0} {
		p, _ := newProgram(t)
		p.model.Entries = []Entry{{Kind: KindUser, Summary: "something"}}
		p.model.Cursor = start

		p.Update(ctrl('o'))
		if !p.model.Copy.Active {
			t.Errorf("cursor=%d: the chord did not open copy mode", start)
		}
	}
}
