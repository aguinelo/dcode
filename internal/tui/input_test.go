package tui

import (
	"strings"
	"testing"
)

// typed builds a line with the caret at a known offset.
func typed(text string, cursor int) Model {
	m := NewModel("s", "/w", "m", "read-only", En)
	m.Input = text
	m.InputCursor = cursor
	return m
}

// The caret and the text can never disagree about where they are, which is why
// editing lives on the model rather than in the key handler.
func TestInsertAtTheCaret(t *testing.T) {
	m := typed("olá", 3).Insert(" mundo")
	if m.Input != "olá mundo" || m.InputCursor != 9 {
		t.Fatalf("got %q at %d", m.Input, m.InputCursor)
	}

	// In the middle, not appended to the end.
	m = typed("ac", 1).Insert("b")
	if m.Input != "abc" || m.InputCursor != 2 {
		t.Errorf("got %q at %d", m.Input, m.InputCursor)
	}
}

// Runes, not bytes: the failure only shows with the accented input a
// Portuguese-speaking user types.
func TestEditingCountsRunesNotBytes(t *testing.T) {
	m := typed("configuração", 12)
	m = m.Backspace()
	if m.Input != "configuraçã" {
		t.Errorf("backspace ate more than one character: %q", m.Input)
	}
	if m.InputCursor != 11 {
		t.Errorf("got caret at %d", m.InputCursor)
	}

	m = typed("日本語", 3).Backspace()
	if m.Input != "日本" {
		t.Errorf("got %q", m.Input)
	}
}

func TestBackspaceAndDeleteAtTheEdges(t *testing.T) {
	if got := typed("abc", 0).Backspace(); got.Input != "abc" {
		t.Errorf("backspace at the start is a no-op, got %q", got.Input)
	}
	if got := typed("abc", 3).DeleteForward(); got.Input != "abc" {
		t.Errorf("delete at the end is a no-op, got %q", got.Input)
	}
	if got := typed("abc", 1).DeleteForward(); got.Input != "ac" {
		t.Errorf("got %q", got.Input)
	}
	// A caret past the end must not panic; it is clamped.
	if got := typed("abc", 99).Backspace(); got.Input != "ab" {
		t.Errorf("got %q", got.Input)
	}
}

func TestDeleteWordTakesTrailingSpacesWithIt(t *testing.T) {
	for _, tc := range []struct{ in, want string }{
		{"refatore o desconto", "refatore o "},
		{"refatore o    ", "refatore "},
		{"palavra", ""},
		{"", ""},
	} {
		got := typed(tc.in, len([]rune(tc.in))).DeleteWord()
		if got.Input != tc.want {
			t.Errorf("%q: got %q want %q", tc.in, got.Input, tc.want)
		}
		if got.InputCursor != len([]rune(tc.want)) {
			t.Errorf("%q: caret at %d", tc.in, got.InputCursor)
		}
	}

	// Only what is behind the caret goes.
	got := typed("um dois tres", 7).DeleteWord()
	if got.Input != "um  tres" {
		t.Errorf("got %q", got.Input)
	}
}

// ---------- history ----------

func TestHistoryWalksBackAndForwards(t *testing.T) {
	m := NewModel("s", "/w", "m", "read-only", En)
	m = m.Remember("primeiro").Remember("segundo")

	m = m.HistoryPrev()
	if m.Input != "segundo" {
		t.Fatalf("up must reach the newest first, got %q", m.Input)
	}
	m = m.HistoryPrev()
	if m.Input != "primeiro" {
		t.Fatalf("got %q", m.Input)
	}
	// Past the oldest there is nothing more to show.
	m = m.HistoryPrev()
	if m.Input != "primeiro" {
		t.Errorf("got %q", m.Input)
	}

	m = m.HistoryNext()
	if m.Input != "segundo" {
		t.Errorf("got %q", m.Input)
	}
	m = m.HistoryNext()
	if m.Input != "" {
		t.Errorf("past the newest returns to the empty line, got %q", m.Input)
	}
}

// Entering the history must not silently discard a half-written message.
func TestHistoryKeepsTheDraft(t *testing.T) {
	m := NewModel("s", "/w", "m", "read-only", En).Remember("antigo")
	m = m.SetInput("meio escrito")

	m = m.HistoryPrev()
	if m.Input != "antigo" {
		t.Fatalf("got %q", m.Input)
	}
	m = m.HistoryNext()
	if m.Input != "meio escrito" {
		t.Errorf("the draft must come back, got %q", m.Input)
	}
}

// Pressing up twice should reach two different commands, not the same one.
func TestHistoryCollapsesConsecutiveDuplicates(t *testing.T) {
	m := NewModel("s", "/w", "m", "read-only", En)
	m = m.Remember("igual").Remember("igual").Remember("outro")
	if len(m.History) != 2 {
		t.Fatalf("got %v", m.History)
	}
	if got := m.Remember("   "); len(got.History) != 2 {
		t.Errorf("blank input is not a command: %v", got.History)
	}
}

func TestHistoryOnAnEmptyHistory(t *testing.T) {
	m := NewModel("s", "/w", "m", "read-only", En)
	if got := m.HistoryPrev(); got.Input != "" {
		t.Errorf("got %q", got.Input)
	}
	if got := m.HistoryNext(); got.Input != "" {
		t.Errorf("got %q", got.Input)
	}
}

// Typing leaves the history: what is on the line is now the user's.
func TestTypingLeavesTheHistory(t *testing.T) {
	m := NewModel("s", "/w", "m", "read-only", En).Remember("antigo").HistoryPrev()
	if m.HistoryAt < 0 {
		t.Fatal("setup: the model should be browsing")
	}
	if got := m.Insert("x"); got.HistoryAt >= 0 {
		t.Error("typing must leave the history behind")
	}
}

// ---------- the caret on screen ----------

func TestTheCaretIsDrawnWhereTypingLands(t *testing.T) {
	g := DefaultGeometry(80, 10)
	g.Palette = Palette{Enabled: true}

	m := typed("abc", 1)
	m.Entries = []Entry{{Kind: KindAssistant, Summary: "x"}}
	got := Render(m, g)

	// Stripped, because the caret escape sits between the characters — which
	// is exactly what it is supposed to do.
	if !strings.Contains(stripANSI(got), "abc") {
		t.Fatalf("the text must be there:\n%q", got)
	}
	// Reverse video marks the character under the caret, so it survives any
	// colour theme.
	if !strings.Contains(got, "\x1b[7m") {
		t.Errorf("the caret must be visible:\n%q", got)
	}
	if visibleWidth(lines(got)[len(lines(got))-1]) > 80 {
		t.Error("the caret must not widen the line")
	}
}

func TestTheCaretAtTheEndOfTheLine(t *testing.T) {
	p := Palette{Enabled: true}
	got := renderCaret(typed("ab", 2), p)
	if stripANSI(got) != "ab " {
		t.Errorf("a caret past the last character is a block after it: %q", stripANSI(got))
	}
	// Out of range is clamped rather than panicking: the caret comes from a
	// keystroke and the line can be replaced between the two.
	if got := renderCaret(typed("ab", 99), p); stripANSI(got) != "ab " {
		t.Errorf("got %q", stripANSI(got))
	}
	if got := renderCaret(typed("ab", -1), p); stripANSI(got) != "ab" {
		t.Errorf("got %q", stripANSI(got))
	}
}
