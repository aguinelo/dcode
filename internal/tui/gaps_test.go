package tui

import (
	"strings"
	"testing"

	"github.com/aguinelo/dcode/internal/loop"
	"github.com/aguinelo/dcode/internal/protocol"
)

// ---------- copy mode ----------

// Copy mode is entered on the cursor, closed, and closing must put the model
// back exactly where it was — a lingering selection paints highlight over a
// stream nobody is selecting from any more.
func TestCopyModeOpensOnTheCursorAndClosesCompletely(t *testing.T) {
	m := Model{Cursor: 2, Flash: "copied"}
	m = m.EnterCopy(5)
	if !m.Copy.Active || m.Copy.Anchor != 2 || m.Copy.Head != 2 {
		t.Fatalf("copy = %+v, want it anchored on the cursor", m.Copy)
	}
	if m.Flash != "" {
		t.Error("an old flash survived into copy mode")
	}

	m = m.LeaveCopy()
	if m.Copy.Active || m.Copy.Anchor != 0 || m.Copy.Head != 0 {
		t.Errorf("copy = %+v after leaving, want it empty", m.Copy)
	}
}

// With no cursor there is still a sensible place to start, and with nothing on
// screen at all it is the first line rather than a negative index.
func TestCopyModeWithNoCursorStartsSomewhereReal(t *testing.T) {
	if got := (Model{Cursor: -1}).EnterCopy(7); got.Copy.Anchor != 7 {
		t.Errorf("anchor = %d, want the last line", got.Copy.Anchor)
	}
	if got := (Model{Cursor: -1}).EnterCopy(-1); got.Copy.Anchor != 0 {
		t.Errorf("anchor = %d, want the first line", got.Copy.Anchor)
	}
}

// The anchor stays put so dragging back past the start shrinks the selection
// rather than inverting it, and neither end may leave the stream.
func TestExtendingASelectionKeepsItsAnchorAndItsBounds(t *testing.T) {
	m := (Model{Cursor: 3}).EnterCopy(9)

	m = m.ExtendCopy(2, 9)
	if m.Copy.Anchor != 3 || m.Copy.Head != 5 {
		t.Fatalf("copy = %+v, want the anchor kept and the head moved", m.Copy)
	}
	if got := m.ExtendCopy(-99, 9); got.Copy.Head != 0 {
		t.Errorf("head = %d, want it stopped at the first line", got.Copy.Head)
	}
	if got := m.ExtendCopy(99, 9); got.Copy.Head != 9 {
		t.Errorf("head = %d, want it stopped at the last line", got.Copy.Head)
	}
	// A limit of -1 means "unknown", and must not clamp everything to zero.
	if got := m.ExtendCopy(4, -1); got.Copy.Head != 9 {
		t.Errorf("head = %d with no known last line, want it moved freely", got.Copy.Head)
	}

	// Extending when copy mode is closed changes nothing.
	if got := (Model{}).ExtendCopy(3, 9); got.Copy.Active {
		t.Error("extending outside copy mode opened it")
	}
}

// What is copied is what is on screen, and a selection reaching past the end of
// it is clamped rather than panicking. Nothing is worse here than an index
// error taking the client down while somebody selects text.
func TestCopyingClampsToWhatIsActuallyOnScreen(t *testing.T) {
	lines := []string{"one", "two", "three"}

	if got := CopyText(lines, CopyState{}); got != "" {
		t.Errorf("copying outside copy mode returned %q", got)
	}
	if got := CopyText(nil, CopyState{Active: true}); got != "" {
		t.Errorf("copying an empty screen returned %q", got)
	}

	got := CopyText(lines, CopyState{Active: true, Anchor: 1, Head: 99})
	if got != "two\nthree" {
		t.Errorf("got %q, want the selection clamped to the last line", got)
	}
	got = CopyText(lines, CopyState{Active: true, Anchor: -5, Head: 0})
	if got != "one" {
		t.Errorf("got %q, want the selection clamped to the first line", got)
	}
}

// ---------- the input line ----------

// Deleting forward at the end of the line has nothing to delete, and must not
// reach past it.
func TestDeletingForwardAtTheEndOfTheLineDoesNothing(t *testing.T) {
	m := Model{Input: "abc", InputCursor: 3}
	if got := m.DeleteForward(); got.Input != "abc" {
		t.Errorf("input = %q, want it unchanged", got.Input)
	}
	m = Model{Input: "abc", InputCursor: 1}
	if got := m.DeleteForward(); got.Input != "ac" {
		t.Errorf("input = %q, want the character under the caret gone", got.Input)
	}
}

// Deleting a word takes the trailing spaces with it, and at the start of the
// line there is nothing to take.
func TestDeletingAWordTakesItsTrailingSpaces(t *testing.T) {
	m := Model{Input: "make the check   ", InputCursor: 17}
	got := m.DeleteWord()
	if got.Input != "make the " {
		t.Errorf("input = %q, want the word and its spaces gone", got.Input)
	}
	if got.InputCursor != len([]rune(got.Input)) {
		t.Errorf("caret at %d, want %d", got.InputCursor, len([]rune(got.Input)))
	}

	if got := (Model{Input: "abc", InputCursor: 0}).DeleteWord(); got.Input != "abc" {
		t.Errorf("input = %q, want it unchanged at the start of the line", got.Input)
	}
}

// A caret outside the line is a bug somewhere upstream, and the input operations
// survive it rather than turning it into a panic in the render loop.
func TestACaretOutsideTheLineIsClampedRatherThanFatal(t *testing.T) {
	if got := (Model{Input: "abc", InputCursor: 99}).DeleteForward(); got.Input != "abc" {
		t.Errorf("input = %q", got.Input)
	}
	if got := (Model{Input: "abc", InputCursor: -4}).DeleteWord(); got.Input != "abc" {
		t.Errorf("input = %q", got.Input)
	}
}

// ---------- input history ----------

// Pressing up twice should reach two different commands, not the same one
// again, so consecutive duplicates collapse. Blank lines are not history.
func TestHistoryCollapsesRepeatsAndIgnoresBlanks(t *testing.T) {
	m := Model{}
	m = m.Remember("make check")
	m = m.Remember("  make check  ")
	m = m.Remember("   ")
	if len(m.History) != 1 {
		t.Fatalf("history = %v, want the repeat and the blank kept out", m.History)
	}
	m = m.Remember("go test ./...")
	if len(m.History) != 2 {
		t.Fatalf("history = %v", m.History)
	}
}

// Walking into the history keeps whatever was being typed, and walking back out
// past the newest entry hands it back — a half-written message discarded by an
// arrow key is work lost with no way to ask for it again.
func TestWalkingTheHistoryReturnsTheDraftItInterrupted(t *testing.T) {
	m := Model{Input: "half writ"}
	m = m.Remember("first").Remember("second")
	m = Model{Input: "half writ", History: m.History, HistoryAt: -1}

	m = m.HistoryPrev()
	if m.Input != "second" {
		t.Fatalf("input = %q, want the newest entry", m.Input)
	}
	m = m.HistoryPrev()
	if m.Input != "first" {
		t.Fatalf("input = %q, want the older entry", m.Input)
	}
	// Past the oldest there is nothing further back.
	if got := m.HistoryPrev(); got.Input != "first" {
		t.Errorf("input = %q, want it held at the oldest entry", got.Input)
	}

	m = m.HistoryNext()
	if m.Input != "second" {
		t.Fatalf("input = %q walking forward", m.Input)
	}
	m = m.HistoryNext()
	if m.Input != "half writ" || m.HistoryAt != -1 {
		t.Errorf("input = %q at %d, want the draft back and the history left", m.Input, m.HistoryAt)
	}
}

// Both walks are answers, not errors, when there is no history to walk.
func TestWalkingAnEmptyHistoryChangesNothing(t *testing.T) {
	if got := (Model{Input: "x"}).HistoryPrev(); got.Input != "x" {
		t.Errorf("input = %q", got.Input)
	}
	if got := (Model{Input: "x", HistoryAt: -1}).HistoryNext(); got.Input != "x" {
		t.Errorf("input = %q", got.Input)
	}
}

// ---------- what the turn says it verified ----------

// The summary is the last thing a person reads about a turn, and each verdict
// has to say a different thing. An unknown verdict is shown as it came rather
// than swallowed: a blank line where the result should be is worse than a word
// nobody recognises.
func TestEveryVerificationVerdictSaysSomethingDifferent(t *testing.T) {
	seen := map[string]bool{}
	for _, v := range []string{
		string(loop.VerificationPassed),
		string(loop.VerificationFailed),
		string(loop.VerificationUnavailable),
		string(loop.VerificationStale),
	} {
		got := completionSummary(&protocol.Completion{
			Verification: v,
			Met:          []string{"the suite"},
			Unmet:        []string{"the lint"},
		}, En)
		if strings.TrimSpace(got) == "" {
			t.Errorf("%s renders as nothing", v)
		}
		if seen[got] {
			t.Errorf("%s renders the same as another verdict: %q", v, got)
		}
		seen[got] = true
	}

	if got := completionSummary(&protocol.Completion{Verification: "something new"}, En); got != "something new" {
		t.Errorf("an unrecognised verdict rendered as %q, want it shown as it came", got)
	}
}
