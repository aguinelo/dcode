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
		// The lane gutter is dropped for the same reason — it is the same
		// character on all three, which is the point of a lane.
		out := renderStream(Model{Entries: []Entry{{Kind: KindCompletion, Summary: summary}}, Cursor: -1}, g, g.Width)
		marks[name] = strings.TrimSpace(dropCells(out[0], 2))[:1]
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
		out := renderStream(Model{Entries: []Entry{{Kind: k, Summary: long}, {}}, Cursor: -1}, g, g.Width)
		if len(out) < 2 {
			t.Fatalf("kind %v did not wrap: %q", k, out)
		}
		// Every row keeps the two-cell lead — the lane and the selection
		// column — and the text after it lines up. Asserting the LEAD rather
		// than "starts with a space" is the same claim made against what the
		// rows are actually built from: a continuation that lost its lane
		// would read as a different kind of thing, not just as a new message.
		lane := leadCells(out[0], 1)
		for i, line := range out[1:] {
			if line == "" {
				continue
			}
			if got := leadCells(line, 1); got != lane {
				t.Errorf("kind %v continuation %d changed lane: %q after %q", k, i+1, got, lane)
			}
			// And past the lane the row still starts with the blank selection
			// column, so its text lines up under the first row rather than
			// beginning at the edge and reading as a new message.
			if !strings.HasPrefix(dropCells(line, 1), " ") {
				t.Errorf("kind %v continuation %d starts at the edge: %q", k, i+1, line)
			}
		}
	}
}

// leadCells is the first n display cells of a rendered row.
func leadCells(s string, n int) string {
	r := []rune(stripANSI(s))
	if len(r) < n {
		return s
	}
	return string(r[:n])
}

// dropCells removes the first n display cells of a rendered row.
func dropCells(s string, n int) string {
	r := []rune(stripANSI(s))
	if len(r) < n {
		return s
	}
	return string(r[n:])
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

// Every row of the stream says which of three things it is: what you asked,
// what the model did on the way, and what it says.
//
// Without it a long turn is prose and tool calls alternating with nothing
// structural between them, so catching up means reading every row to find out
// which rows were worth reading.
func TestEveryStreamRowIsInALane(t *testing.T) {
	for _, unicode := range []bool{true, false} {
		g := DefaultGeometry(80, 30)
		g.Palette = Palette{}
		g.Unicode = unicode
		gl := glyphs(unicode)
		lanes := map[string]bool{gl.laneYou: true, gl.laneProcess: true, gl.laneAnswer: true}

		m := Model{Lang: En, Cursor: -1, Entries: []Entry{
			{Kind: KindUser, Summary: "a question"},
			{Kind: KindAssistant, Summary: "an answer long enough to wrap across the stream at least once, twice"},
			{Kind: KindTool, Tool: "read", Target: "a.go", Summary: "24 lines"},
			{Kind: KindReasoning, Summary: "a thought", Closed: true, Expanded: true},
			{Kind: KindNote, Summary: "a note"},
			{Kind: KindError, Summary: "boom"},
			{Kind: KindCompletion, Summary: "verified: 2 checks"},
		}}

		rows := renderStream(m, g, g.Width)
		for i, l := range rows {
			if l == "" {
				continue
			}
			if lane := leadCells(l, 1); !lanes[lane] {
				t.Errorf("unicode=%v: row %d is in no lane (%q): %q", unicode, i, lane, l)
			}
		}

		// The two that were indistinguishable must differ, and the lane must
		// be a CHARACTER: a lane told apart by colour alone is no lane on a
		// terminal without any, which is the rule every other mark here obeys.
		var answer, process string
		for _, l := range rows {
			switch {
			case strings.Contains(l, "an answer") && answer == "":
				answer = leadCells(l, 1)
			case strings.Contains(l, "read") && process == "":
				process = leadCells(l, 1)
			}
		}
		if answer == "" || process == "" {
			t.Fatalf("unicode=%v: fixture did not produce both lanes", unicode)
		}
		if answer == process {
			t.Errorf("unicode=%v: the answer and the work share a lane (%q)", unicode, answer)
		}
	}
}

// The lane costs no width. Every row already reserved two columns — the
// selection marker, or two spaces where there was none — so the lane takes the
// first and the marker keeps the second.
func TestTheLaneCostsNoColumns(t *testing.T) {
	g := DefaultGeometry(80, 30)
	g.Palette = Palette{}
	m := Model{Lang: En, Cursor: -1, Entries: []Entry{
		{Kind: KindAssistant, Summary: strings.Repeat("palavra ", 40)},
		{Kind: KindTool, Tool: "read", Target: "a.go", Summary: "24 lines"},
	}}

	for i, l := range renderStream(m, g, g.Width) {
		if l == "" {
			continue
		}
		if w := visibleWidth(l); w > g.Width {
			t.Errorf("row %d is %d cells in a %d column stream: %q", i, w, g.Width, l)
		}
		// The text still starts where it started: the lane displaced the
		// padding, not the content.
		if body := dropCells(l, 2); strings.HasPrefix(body, " ") && strings.TrimSpace(body) != "" {
			t.Errorf("row %d pushed its text right: %q", i, l)
		}
	}
}

// A legend is worth its row when what it explains is a CHARACTER whose meaning
// cannot be guessed — `▏` and `╎` say nothing on their own, and a reader who
// has not been told reads them as decoration and stops seeing them.
//
// And only when the screen is making the distinction: a legend for three lanes
// on a screen with one is a row spent on nothing.
func TestTheLaneLegendAppearsOnlyWhenItExplainsSomething(t *testing.T) {
	g := DefaultGeometry(120, 30)
	g.Palette = Palette{}

	one := Model{Lang: En, Cursor: -1, Entries: []Entry{
		{Kind: KindAssistant, Summary: "just an answer"},
	}}
	if got := renderLanes(one, g, 120); len(got) != 0 {
		t.Errorf("a single-lane screen drew a legend: %q", got)
	}

	many := one
	many.Entries = append(many.Entries, Entry{Kind: KindTool, Tool: "read", Target: "a.go"})
	got := strings.Join(renderLanes(many, g, 120), "\n")
	for _, want := range []string{"YOU", "WORK", "ANSWER"} {
		if !strings.Contains(got, want) {
			t.Errorf("%q is missing from the legend: %q", want, got)
		}
	}
	// It names each lane with the character the stream actually draws.
	gl := glyphs(g.Unicode)
	for _, mark := range []string{gl.laneYou, gl.laneProcess, gl.laneAnswer} {
		if !strings.Contains(got, mark) {
			t.Errorf("the legend does not show %q: %q", mark, got)
		}
	}
	// And it never takes a row it cannot fill honestly.
	if narrow := renderLanes(many, g, 12); len(narrow) != 0 {
		t.Errorf("a legend was drawn in 12 columns: %q", narrow)
	}
}
