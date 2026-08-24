package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/aguinelo/dcode/internal/protocol"
)

func geo(w, h int) Geometry { return Geometry{Width: w, Height: h} }

func protocolIdle() protocol.SessionState { return protocol.SessionStateIdle }

// Enter sends, so a list could not be typed: every line collapsed into one
// paragraph. This is the whole feature.
func TestABreakMakesARow(t *testing.T) {
	g := geo(80, 24)
	if got := InputRows(Model{Input: "one"}, g); got != 1 {
		t.Errorf("a single line takes %d rows", got)
	}
	if got := InputRows(Model{Input: "one\ntwo\nthree"}, g); got != 3 {
		t.Errorf("three lines take %d rows, want 3", got)
	}
	// A trailing break is a row the caret sits on, and losing it would put the
	// caret on the line above what the user is typing.
	if got := InputRows(Model{Input: "one\n"}, g); got != 2 {
		t.Errorf("a trailing break takes %d rows, want 2", got)
	}
}

// A pasted essay must not eat the window.
func TestTheBoxStopsGrowing(t *testing.T) {
	g := geo(80, 40)
	tall := strings.Repeat("x\n", 100)
	got := InputRows(Model{Input: tall}, g)
	if got != MaxInputRows {
		t.Errorf("a hundred lines took %d rows, want the cap of %d", got, MaxInputRows)
	}
}

// On a short terminal the cap has to yield, or the box leaves no stream.
func TestTheBoxNeverTakesTheWholeWindow(t *testing.T) {
	g := geo(80, 8)
	tall := strings.Repeat("x\n", 100)
	rows := InputRows(Model{Input: tall}, g)
	if rows >= g.Height-3 {
		t.Errorf("the box took %d of %d rows, leaving nothing to read", rows, g.Height)
	}
	if rows < 1 {
		t.Errorf("the box vanished: %d rows", rows)
	}
}

// The assertion that would have caught the ghosting: what is drawn has to be
// exactly the height the layout reserved. Two places computing it is the bug.
func TestTheFrameReservesExactlyWhatTheBoxDraws(t *testing.T) {
	for _, in := range []string{"", "one", "one\ntwo", strings.Repeat("y\n", 50)} {
		m := Model{Input: in}
		g := geo(80, 24)
		if got, want := len(renderInputLines(m, g, "")), InputHeight(m, g); got != want {
			t.Errorf("input %q drew %d rows and the layout reserved %d", in, got, want)
		}
	}
}

// Every row is exactly the frame's width, or the terminal keeps whatever was
// under the short ones — which is the ghosting this project already fixed once.
func TestEveryRowOfTheBoxCoversItsWidth(t *testing.T) {
	m := Model{Input: "one\ntwo longer line\nx"}
	g := geo(40, 24)
	for i, line := range renderInputLines(m, g, "") {
		if w := clipWidth(line); w != g.Width {
			t.Errorf("row %d is %d cells wide, want %d: %q", i, w, g.Width, line)
		}
	}
}

// The prompt marks where typing starts, and it belongs on the first row only.
// Repeating it would read as three separate messages.
func TestOnlyTheFirstRowCarriesThePrompt(t *testing.T) {
	lines := renderInputLines(Model{Input: "one\ntwo\nthree"}, geo(40, 24), "")
	// Row zero is the rule above the area now, so the prompt is on the first
	// row INSIDE the frame.
	body := lines[1 : len(lines)-1]
	if !strings.Contains(body[0], "> ") {
		t.Errorf("the first row has no prompt: %q", body[0])
	}
	for _, l := range body[1:] {
		if strings.HasPrefix(strings.TrimSpace(strings.Trim(l, "|")), ">") {
			t.Errorf("a continuation row carries a prompt: %q", l)
		}
	}
}

// The area is delimited on all four sides, and by a glyph rather than a colour:
// a boundary drawn in colour alone is no boundary on a terminal without any,
// and this one answers the question that has no other answer on the screen —
// where do the letters I type go.
func TestTheInputAreaIsDelimited(t *testing.T) {
	for _, unicode := range []bool{true, false} {
		g := geo(40, 24)
		g.Unicode = unicode
		gl := glyphs(unicode)
		lines := renderInputLines(Model{Input: "one\ntwo"}, g, "")

		if !strings.HasPrefix(lines[0], gl.boxTL) || !strings.HasSuffix(lines[0], gl.boxTR) {
			t.Errorf("unicode=%v: no rule above the area: %q", unicode, lines[0])
		}
		last := lines[len(lines)-1]
		if !strings.HasPrefix(last, gl.boxBL) || !strings.HasSuffix(last, gl.boxBR) {
			t.Errorf("unicode=%v: no rule below the area: %q", unicode, last)
		}
		for i, l := range lines[1 : len(lines)-1] {
			if !strings.HasPrefix(l, gl.gutter) || !strings.HasSuffix(l, gl.gutter) {
				t.Errorf("unicode=%v: row %d has no sides: %q", unicode, i, l)
			}
		}
	}
}

// The frame draws the same shape with and without colour.
//
// An earlier version of it dimmed while the stream had the keyboard. The test
// asked the obvious next question — does that distinction survive without
// colour — and the answer was no: it was a colour and nothing else, which is
// the one thing a state indicator here may not be. So the frame stopped
// carrying state, and the state stayed where it is already drawn, on the entry
// under the stream cursor.
func TestTheFrameIsTheSameShapeWithAndWithoutColour(t *testing.T) {
	mono, colour := geo(40, 24), geo(40, 24)
	colour.Palette = Palette{Enabled: true}

	for _, cursor := range []int{-1, 0} {
		m := Model{Input: "x", Cursor: cursor}
		a := renderInputLines(m, mono, "")
		b := renderInputLines(m, colour, "")
		if len(a) != len(b) {
			t.Fatalf("cursor=%d: %d rows in mono, %d in colour", cursor, len(a), len(b))
		}
		for i := range a {
			if a[i] != stripANSI(b[i]) {
				t.Errorf("cursor=%d row %d: colour changed the shape\n mono: %q\n col:  %q",
					cursor, i, a[i], stripANSI(b[i]))
			}
		}
	}
}

// The caret has to know about rows, or home and end jump to the ends of the
// whole buffer and the arrows walk through the break instead of over it.
func TestTheCaretMovesByLine(t *testing.T) {
	m := Model{Input: "abc\ndefgh", InputCursor: 5} // before "e", row 1 col 1

	if row, col := caretAt(m.Input, m.InputCursor); row != 1 || col != 1 {
		t.Fatalf("caret at row %d col %d, want 1,1", row, col)
	}

	// Home is the start of THIS line, not of everything typed.
	if got := LineStart(m.Input, m.InputCursor); got != 4 {
		t.Errorf("home went to %d, want the start of the second line", got)
	}
	if got := LineEnd(m.Input, m.InputCursor); got != 9 {
		t.Errorf("end went to %d, want the end of the second line", got)
	}

	// Up keeps the column where it can.
	up := LineUp(m.Input, m.InputCursor)
	if row, col := caretAt(m.Input, up); row != 0 || col != 1 {
		t.Errorf("up landed at row %d col %d, want 0,1", row, col)
	}
	// And clamps to a shorter line rather than overshooting into the next one.
	short := Model{Input: "ab\nlonger", InputCursor: 9}
	if row, _ := caretAt(short.Input, LineUp(short.Input, short.InputCursor)); row != 0 {
		t.Errorf("up from a long line left row 0")
	}
	if got := LineUp(short.Input, short.InputCursor); got != 2 {
		t.Errorf("up landed at %d, want the end of the shorter line", got)
	}
}

func TestOnASingleLineTheCaretKeepsItsOldBehaviour(t *testing.T) {
	const s = "just one line"
	if got := LineStart(s, 5); got != 0 {
		t.Errorf("home went to %d", got)
	}
	if got := LineEnd(s, 5); got != len(s) {
		t.Errorf("end went to %d", got)
	}
	// Nowhere to go, so it stays put and the caller falls back to history.
	if got := LineUp(s, 5); got != -1 {
		t.Errorf("up reported %d, want -1 for nothing above", got)
	}
	if got := LineDown(s, 5); got != -1 {
		t.Errorf("down reported %d, want -1 for nothing below", got)
	}
}

// The frame is the assertion that would have caught the ghosting: a taller box
// takes its rows from the stream and from nowhere else, and the whole thing
// still fills the terminal exactly.
func TestATallerBoxTakesItsRowsFromTheStream(t *testing.T) {
	g := geo(60, 24)
	base := Model{Lang: En, State: protocolIdle()}
	tall := base
	tall.Input = "one\ntwo\nthree\nfour"

	if a, b := BodyHeight(base, g), BodyHeight(tall, g); b != a-3 {
		t.Errorf("a box three rows taller left the stream at %d, was %d", b, a)
	}

	for _, m := range []Model{base, tall} {
		lines := strings.Split(strings.TrimSuffix(Render(m, g), "\n"), "\n")
		if len(lines) != g.Height {
			t.Errorf("input %q rendered %d rows, want exactly %d",
				m.Input, len(lines), g.Height)
		}
	}
}

// Past the cap the box scrolls rather than truncating: the row being typed on
// has to be visible, and it is usually the last one.
func TestTheBoxScrollsToKeepTheCaretVisible(t *testing.T) {
	g := geo(40, 40)
	var b strings.Builder
	for i := 0; i < 30; i++ {
		b.WriteString("line\n")
	}
	m := Model{Input: b.String()}
	m.InputCursor = len([]rune(m.Input)) // the empty row at the end

	lines, caretRow, _ := inputWindow(m, InputRows(m, g))
	if len(lines) != MaxInputRows {
		t.Fatalf("the window shows %d rows, want the cap of %d", len(lines), MaxInputRows)
	}
	if caretRow < 0 || caretRow >= MaxInputRows {
		t.Errorf("the caret is on row %d, outside the visible window", caretRow)
	}
	// And at the top nothing scrolls, or the first line is unreachable.
	m.InputCursor = 0
	_, caretRow, _ = inputWindow(m, InputRows(m, g))
	if caretRow != 0 {
		t.Errorf("with the caret at the start it sits on row %d", caretRow)
	}
}

// Ctrl+A and Ctrl+E move within the line the caret is on.
func TestTheLineKeysStayOnTheirLine(t *testing.T) {
	const text = "first\nsecond line\nthird"
	at := 10 // inside "second line"
	if got := LineStart(text, at); got != 6 {
		t.Errorf("ctrl+a went to %d, want the start of the second line", got)
	}
	if got := LineEnd(text, at); got != 17 {
		t.Errorf("ctrl+e went to %d, want the end of the second line", got)
	}
}

// A clipboard with nothing in it, and a machine with no way to look, are
// different answers and need different notes.
func TestTheClipboardDistinguishesEmptyFromUnavailable(t *testing.T) {
	if ErrNoImageInClipboard == ErrNoClipboardTool {
		t.Fatal("the two outcomes are the same value, so the caller cannot tell them apart")
	}
	for _, e := range []error{ErrNoImageInClipboard, ErrNoClipboardTool} {
		if e == nil || e.Error() == "" {
			t.Error("an error with nothing to say")
		}
	}
}

// The reader answers on this machine, whatever the answer is. What it must
// never do is hang or panic — it runs on a keypress, in the interface.
func TestReadingTheClipboardAnswers(t *testing.T) {
	body, media, err := ClipboardImage()
	switch {
	case err != nil:
		// Empty or unavailable are both fine here; a CI runner has neither a
		// clipboard nor a tool.
	case len(body) == 0:
		t.Error("success with no bytes")
	case media == "":
		t.Error("bytes with no media type")
	}
}

// A screenshot path is typed by a person, and that is how a person writes it.
func TestATildePathFindsTheFile(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("no home directory here")
	}
	if got := expandHome("~/Desktop/shot.png"); got != filepath.Join(home, "Desktop/shot.png") {
		t.Errorf("expanded to %q", got)
	}
	// Only a leading ~/. A path with a tilde inside it is a path.
	for _, p := range []string{"/tmp/a~b.png", "relative/shot.png", "~weird"} {
		if got := expandHome(p); got != p {
			t.Errorf("%q was rewritten to %q", p, got)
		}
	}
}
