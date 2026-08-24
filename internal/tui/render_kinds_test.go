package tui

import (
	"strings"
	"testing"

	"github.com/aguinelo/dcode/internal/loop"
)

// Every kind of entry has to reach the screen, and each one has to be
// distinguishable from the others without colour: a monochrome terminal, a
// screenshot pasted into a bug report, and a colour-blind reader all have to
// get the same answer. A kind that renders as nothing is a turn the person
// cannot see happened.
func TestEveryKindOfEntryReachesTheScreen(t *testing.T) {
	g := DefaultGeometry(80, 24)
	g.Palette = Palette{Enabled: false}

	for _, tc := range []struct {
		name string
		e    Entry
		want string
	}{
		{"user", Entry{Kind: KindUser, Summary: "make the check pass"}, "make the check pass"},
		{"assistant", Entry{Kind: KindAssistant, Summary: "I read the file."}, "I read the file."},
		{"error", Entry{Kind: KindError, Summary: "the suite failed"}, "the suite failed"},
		{"note", Entry{Kind: KindNote, Summary: "the session was continued"}, "the session was continued"},
		{"reasoning", Entry{Kind: KindReasoning, Summary: "weighing two options"}, "weighing"},
		{"completion ok", Entry{Kind: KindCompletion, Summary: "verified: 2 checks"}, "verified"},
		{"completion failed", Entry{Kind: KindCompletion, Summary: "NOT verified: the lint"}, "NOT verified"},
		{"completion unavailable", Entry{Kind: KindCompletion, Summary: "not verified: nothing to run"}, "not verified"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			out := renderStream(Model{Entries: []Entry{tc.e}, Cursor: 0}, g, g.Width)
			if !strings.Contains(strings.Join(out, "\n"), tc.want) {
				t.Errorf("%s rendered as %q, want it to carry %q", tc.name, out, tc.want)
			}
		})
	}
}

// A failing completion says so in the words as well as in the colour, and the
// three verdicts must not share a mark. Reading the mark is the only channel
// left when colour is gone.
func TestTheThreeCompletionVerdictsAreToldApartWithoutColour(t *testing.T) {
	g := DefaultGeometry(80, 24)
	g.Palette = Palette{Enabled: false}

	marks := map[string]string{}
	for name, summary := range map[string]string{
		"passed":      "verified: 2 checks",
		"failed":      "NOT verified: the lint",
		"unavailable": "not verified: nothing to run",
	} {
		// Cursor away from the entry: the selection arrow would otherwise be
		// the first character of every line and hide the mark being compared.
		out := renderStream(Model{Entries: []Entry{{Kind: KindCompletion, Summary: summary}}, Cursor: -1}, g, g.Width)
		marks[name] = strings.TrimSpace(out[0])[:1]
	}
	if marks["failed"] == marks["unavailable"] || marks["failed"] == marks["passed"] {
		t.Errorf("two verdicts share a mark: %v", marks)
	}
}

// A long line wraps rather than being cut, and every continuation row keeps a
// prefix that lines up under the first. A continuation that starts at column
// zero reads as a new message.
func TestLongEntriesWrapWithAlignedContinuations(t *testing.T) {
	g := DefaultGeometry(40, 24)
	g.Palette = Palette{Enabled: false}
	long := strings.Repeat("palavra ", 20)

	for _, k := range []Kind{KindUser, KindNote} {
		out := renderStream(Model{Entries: []Entry{{Kind: k, Summary: long}}}, g, g.Width)
		if len(out) < 2 {
			t.Fatalf("kind %v did not wrap: %q", k, out)
		}
		for i, line := range out[1:] {
			if line != "" && !strings.HasPrefix(line, " ") {
				t.Errorf("kind %v continuation %d starts at column zero: %q", k, i+1, line)
			}
		}
	}
}

// An expanded entry shows its detail and a collapsed one does not. That is the
// whole contract of Tab, and it holds for the kinds that carry a detail.
func TestExpandingShowsTheDetailAndCollapsingHidesIt(t *testing.T) {
	g := DefaultGeometry(80, 24)
	g.Palette = Palette{Enabled: false}

	for _, k := range []Kind{KindError, KindCompletion} {
		e := Entry{Kind: k, Summary: "a summary", Detail: "the detail nobody sees collapsed"}

		collapsed := strings.Join(renderStream(Model{Entries: []Entry{e}}, g, g.Width), "\n")
		if strings.Contains(collapsed, "nobody sees") {
			t.Errorf("kind %v showed its detail while collapsed", k)
		}

		e.Expanded = true
		expanded := strings.Join(renderStream(Model{Entries: []Entry{e}}, g, g.Width), "\n")
		if !strings.Contains(expanded, "nobody sees") {
			t.Errorf("kind %v hid its detail while expanded: %q", k, expanded)
		}
	}
}

// The label on the status line is the one claim the model's prose cannot
// contradict. Each verdict that means something says something, and the state
// that means nothing says nothing — a permanent label stops being read.
func TestTheVerificationLabelClaimsOnlyWhatRan(t *testing.T) {
	for _, v := range []string{
		string(loop.VerificationPassed),
		string(loop.VerificationFailed),
		string(loop.VerificationStale),
		string(loop.VerificationUnavailable),
	} {
		label, style := VerificationLabel(v, En)
		if strings.TrimSpace(label) == "" {
			t.Errorf("%s has no label", v)
		}
		if style == StyleDim {
			t.Errorf("%s is styled as nothing happened", v)
		}
	}

	// Clean, or no definition of done: nothing changed, so there is nothing to
	// claim either way.
	if label, style := VerificationLabel("", En); label != "" || style != StyleDim {
		t.Errorf("a session with nothing to verify claimed %q", label)
	}
}

// ---------- the caret moving between lines ----------

// Down from the last line is nowhere, and down onto a shorter line lands at its
// end rather than past it. A caret past the end of a line is an index the
// renderer then paints outside the box.
func TestMovingTheCaretDownStopsAtTheEndOfAShorterLine(t *testing.T) {
	text := "a long first line\nshort"

	// Column 12 on the first line; the second line has only five characters.
	at := LineDown(text, 12)
	if at != len(text) {
		t.Errorf("landed at %d, want the end of the shorter line (%d)", at, len(text))
	}

	// Straight down where the column exists on both lines.
	if got := LineDown("abcdef\nghijkl", 2); got != 9 {
		t.Errorf("landed at %d, want the same column one row down", got)
	}

	// From the last line there is nowhere further down.
	if got := LineDown(text, len(text)); got != -1 {
		t.Errorf("got %d from the last line, want -1", got)
	}
	if got := LineDown("one line", 3); got != -1 {
		t.Errorf("got %d from a single line, want -1", got)
	}
}

// A caret outside the text is a bug upstream, and every movement clamps rather
// than indexing out of the slice while somebody holds an arrow key.
func TestCaretMovementsSurviveACaretOutsideTheText(t *testing.T) {
	text := "one\ntwo"
	for _, tc := range []struct {
		name string
		got  int
	}{
		{"down from before the start", LineDown(text, -9)},
		{"down from past the end", LineDown(text, 999)},
		{"up from before the start", LineUp(text, -9)},
		{"up from past the end", LineUp(text, 999)},
		{"start from past the end", LineStart(text, 999)},
		{"end from before the start", LineEnd(text, -9)},
	} {
		if tc.got < -1 || tc.got > len([]rune(text)) {
			t.Errorf("%s landed at %d, outside the text", tc.name, tc.got)
		}
	}
}
