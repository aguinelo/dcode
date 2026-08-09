package tui

import (
	"strings"
	"testing"
	"time"

	"github.com/aguinelo/dcode/internal/protocol"
	"github.com/mattn/go-runewidth"
)

func envOf(m map[string]string) func(string) string {
	return func(k string) string { return m[k] }
}

// NO_COLOR is the convention users already know, and a terminal that says it
// cannot should be believed.
func TestColorEnabled(t *testing.T) {
	for name, tc := range map[string]struct {
		env  map[string]string
		want bool
	}{
		"ordinary terminal": {map[string]string{"TERM": "xterm-256color"}, true},
		"NO_COLOR":          {map[string]string{"TERM": "xterm", "NO_COLOR": "1"}, false},
		"dumb terminal":     {map[string]string{"TERM": "dumb"}, false},
		"no TERM at all":    {map[string]string{}, false},
		// The user answering for their own terminal beats every heuristic:
		// this is exactly the case where they know better.
		"forced on over NO_COLOR": {
			map[string]string{"TERM": "dumb", "NO_COLOR": "1", "DCODE_COLOR": "always"}, true},
		"forced off": {map[string]string{"TERM": "xterm", "DCODE_COLOR": "never"}, false},
	} {
		if got := ColorEnabled(envOf(tc.env)); got != tc.want {
			t.Errorf("%s: got %v", name, got)
		}
	}
}

// A style must never change how wide something renders, or a coloured line
// overflows a column the monochrome one fits — the hardest bug to see.
func TestStylingDoesNotChangeDisplayWidth(t *testing.T) {
	p := Palette{Enabled: true}
	for _, s := range []Style{StyleDim, StyleBold, StyleAccent, StyleAdded, StyleError, StyleDanger} {
		text := "configuração"
		styled := p.Apply(s, text)
		if visibleWidth(styled) != runewidth.StringWidth(text) {
			t.Errorf("style %d changed the measured width: %d vs %d",
				s, visibleWidth(styled), runewidth.StringWidth(text))
		}
	}
}

// The zero palette writes nothing at all: one rendering path, not two.
func TestADisabledPaletteEmitsNoEscapes(t *testing.T) {
	p := Palette{}
	for _, s := range []Style{StyleDim, StyleError, StyleDanger} {
		if got := p.Apply(s, "x"); got != "x" {
			t.Errorf("got %q", got)
		}
	}
	if got := (Palette{Enabled: true}).Apply(StyleNone, "x"); got != "x" {
		t.Errorf("StyleNone is not a style: %q", got)
	}
	if got := (Palette{Enabled: true}).Apply(StyleDim, ""); got != "" {
		t.Errorf("empty text needs no escapes: %q", got)
	}
}

// Regression: clipStyled appended a reset even when nothing was styled, so a
// NO_COLOR user got escape bytes in what should be plain text — and every
// width measurement counted them.
func TestClippingPlainTextNeverIntroducesEscapes(t *testing.T) {
	got := clipStyled(strings.Repeat("a", 50), 10)
	if strings.ContainsRune(got, 0x1b) {
		t.Errorf("plain text must stay plain: %q", got)
	}
	if visibleWidth(got) != 10 {
		t.Errorf("got %d cells", visibleWidth(got))
	}
}

// Truncating inside an escape would leave the terminal in that style for the
// rest of the screen.
func TestClippingStyledTextClosesTheStyle(t *testing.T) {
	p := Palette{Enabled: true}
	got := clipStyled(p.Apply(StyleError, strings.Repeat("a", 50)), 10)
	if visibleWidth(got) != 10 {
		t.Errorf("got %d cells: %q", visibleWidth(got), got)
	}
	if !strings.HasSuffix(got, "\x1b[0m") {
		t.Errorf("the style must be closed: %q", got)
	}
}

func TestPadStyledMeasuresCellsNotBytes(t *testing.T) {
	p := Palette{Enabled: true}
	got := padStyled(p.Apply(StyleDim, "日本"), 10)
	if visibleWidth(got) != 10 {
		t.Errorf("got %d cells", visibleWidth(got))
	}
	// Already too wide: pad has to clip rather than grow.
	if got := padStyled("abcdef", 3); visibleWidth(got) != 3 {
		t.Errorf("got %q", got)
	}
}

func TestStripANSI(t *testing.T) {
	if got := stripANSI("\x1b[31mred\x1b[0m"); got != "red" {
		t.Errorf("got %q", got)
	}
	if got := stripANSI("plain"); got != "plain" {
		t.Errorf("got %q", got)
	}
	// An unterminated escape must not eat the rest of the string forever.
	if got := stripANSI("a\x1b["); got != "a" {
		t.Errorf("got %q", got)
	}
}

// Colour on the context meter is a warning, not decoration.
func TestContextStyleGradesTheMeter(t *testing.T) {
	for pct, want := range map[int]Style{
		10: StyleDim, 74: StyleDim,
		75: StyleWarn, 89: StyleWarn,
		90: StyleError, 99: StyleError,
	} {
		if got := ContextStyle(pct); got != want {
			t.Errorf("%d%%: got %d want %d", pct, got, want)
		}
	}
}

func TestDiffStyle(t *testing.T) {
	for line, want := range map[string]Style{
		"+ added":       StyleAdded,
		"- removed":     StyleRemoved,
		"+++ b/x.go":    StyleDim,
		"--- a/x.go":    StyleDim,
		"@@ -1,2 +1 @@": StyleDim,
		" context":      StyleNone,
	} {
		if got := DiffStyle(line); got != want {
			t.Errorf("%q: got %d want %d", line, got, want)
		}
	}
}

// A static glyph cannot tell a long turn from a hung one, which is the whole
// question the user is asking the screen.
func TestSpinnerAnimatesAndDegrades(t *testing.T) {
	seen := map[string]struct{}{}
	for i := 0; i < 10; i++ {
		seen[Spinner(i, true)] = struct{}{}
	}
	if len(seen) < 4 {
		t.Errorf("the spinner must actually move, got %d frames", len(seen))
	}
	// One cell wide in every frame, or the line jitters as it turns.
	for i := 0; i < 10; i++ {
		if w := runewidth.StringWidth(Spinner(i, true)); w != 1 {
			t.Errorf("frame %d is %d cells", i, w)
		}
		if got := Spinner(i, false); got > "\x7f" {
			t.Errorf("the ASCII spinner must be ASCII, got %q", got)
		}
	}
	// Pure over the counter, so a golden test can pin a frame.
	if Spinner(3, true) != Spinner(13, true) {
		t.Error("the spinner must cycle deterministically")
	}
	if Spinner(-3, true) == "" {
		t.Error("a negative counter must still produce a frame")
	}
}

func TestHumanTokens(t *testing.T) {
	for n, want := range map[int]string{
		0: "", -5: "", 1: "1", 999: "999",
		1000: "1.0k", 12400: "12.4k", 1_500_000: "1.5M",
	} {
		if got := humanTokens(n); got != want {
			t.Errorf("%d: got %q want %q", n, got, want)
		}
	}
}

func TestFormatDuration(t *testing.T) {
	for d, want := range map[time.Duration]string{
		0:                       "",
		-time.Second:            "",
		250 * time.Millisecond:  "250ms",
		1500 * time.Millisecond: "1.5s",
		90 * time.Second:        "1m30s",
	} {
		if got := FormatDuration(d); got != want {
			t.Errorf("%v: got %q want %q", d, got, want)
		}
	}
}

// ---------- the working line ----------

func workingModel() Model {
	m := NewModel("s1", "/w", "MiniMax-M3", "read-only")
	m.State = protocol.SessionStateRunning
	m.Entries = []Entry{{Kind: KindUser, Summary: "vai"}}
	m.TurnStartedAt = time.Unix(1000, 0)
	m.Now = time.Unix(1012, 0)
	m.OutputTokens = 1234
	return m
}

// Without it a long turn is indistinguishable from a hung one, and the user's
// only move is to kill the process.
func TestTheWorkingLineSaysWhatHowLongAndHowToStop(t *testing.T) {
	got := Render(workingModel(), DefaultGeometry(100, 12))

	if !strings.Contains(got, "12.0s") {
		t.Errorf("elapsed time must be on screen:\n%s", got)
	}
	if !strings.Contains(got, "1.2k tok") {
		t.Errorf("what it is costing must be on screen:\n%s", got)
	}
	if !strings.Contains(got, "^C") {
		t.Errorf("the way out belongs next to the thing you want out of:\n%s", got)
	}
}

// While a tool runs, the line names the tool: "working" says nothing a spinner
// has not already said.
func TestTheWorkingLineNamesTheRunningTool(t *testing.T) {
	m := workingModel()
	m.Entries = append(m.Entries, Entry{
		Kind: KindTool, Tool: "bash", Target: "go test ./...", Running: true,
	})
	got := Render(m, DefaultGeometry(100, 12))
	if !strings.Contains(got, "bash go test ./...") {
		t.Errorf("got:\n%s", got)
	}
}

func TestTheWorkingLineIsAbsentWhenIdle(t *testing.T) {
	m := workingModel()
	m.State = protocol.SessionStateIdle
	if strings.Contains(Render(m, DefaultGeometry(100, 12)), "^C interrupts") {
		t.Error("an idle session has nothing to interrupt")
	}
}

func TestTheWorkingLineFitsANarrowTerminal(t *testing.T) {
	for _, w := range []int{40, 60, 100} {
		g := DefaultGeometry(w, 12)
		g.Palette = Palette{Enabled: true}
		if n := widest(Render(workingModel(), g)); n > w {
			t.Errorf("width %d: a line is %d cells", w, n)
		}
	}
}

func TestContextLabel(t *testing.T) {
	for name, tc := range map[string]struct {
		used, window int
		want         string
	}{
		"a third":            {34000, 100000, "ctx 34%"},
		"tiny of a million":  {5200, 1_000_000, "ctx <1%"},
		"nothing used yet":   {0, 100000, ""},
		"no window reported": {5200, 0, ""},
		"full":               {100000, 100000, "ctx 100%"},
	} {
		if got := ContextLabel(tc.used, tc.window); got != tc.want {
			t.Errorf("%s: got %q want %q", name, got, tc.want)
		}
	}
}
