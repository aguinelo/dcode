package tui

import (
	"math"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/aguinelo/dcode/internal/protocol"
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
		if visibleWidth(styled) != ruler.StringWidth(text) {
			t.Errorf("style %d changed the measured width: %d vs %d",
				s, visibleWidth(styled), ruler.StringWidth(text))
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
		if w := ruler.StringWidth(Spinner(i, true)); w != 1 {
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
	m := NewModel("s1", "/w", "MiniMax-M3", "read-only", En)
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

// The model's prose is most of what is on the screen, and it was the one thing
// on it drawn faint.
//
// It used to be asserted as "prose carries no attribute at all", which was the
// only way to say "normal weight" while the product did not choose its ground.
// With a theme the claim is the one that was always meant: prose is BRIGHTER
// than the things that qualify it, measured against the ground both are drawn
// on.
func TestTheAnswerIsBrighterThanWhatQualifiesIt(t *testing.T) {
	th := Neon()
	prose := contrast(th.Role[StyleProse].fg, th.Ground)
	for _, c := range []struct {
		style Style
		name  string
	}{{StyleMeta, "meta"}, {StyleHint, "hint"}, {StyleChrome, "chrome"}} {
		if got := contrast(th.Role[c.style].fg, th.Ground); got >= prose {
			t.Errorf("%s is %.2f:1 and prose is %.2f:1; the answer must be the brighter",
				c.name, got, prose)
		}
	}
	// And the contrast inside a sentence is still bought, with the term.
	if th.Role[StyleCode].fg == th.Role[StyleProse].fg {
		t.Error("a technical term inside a sentence is not picked out at all")
	}
}

// Every role in the hierarchy maps to something, and to one of the three
// weights a terminal has that survive an unknown background.
//
// Every role in the hierarchy is legible against the ground the theme paints.
//
// This test asserted the opposite rule until the theme arrived: that no role
// may pick a colour, because "a grey chosen for a dark theme is unreadable on a
// light one". That was right for exactly as long as the product did not choose
// the ground. It does now, so the constraint is no longer "pick no colour" — it
// is "pick one that can be read against the ground you picked", which is a
// stronger claim and a measurable one.
//
// The ratios are WCAG relative luminance. Body text at 4.5, anything meant to
// be read at 3, and chrome at 1.5 — a rule is meant to be SEEN and not read,
// and holding it to text contrast would make every gutter shout.
func TestEveryRoleIsLegibleAgainstTheGround(t *testing.T) {
	ground := 0
	for _, th := range Themes() {
		if th.Ground.zero() {
			// A theme with no ground has nothing to measure against, and
			// skipping it would leave a whole theme outside every guard. So it
			// is held to the condition that makes the TERMINAL's legibility
			// hold instead: no role picks an RGB. An RGB here is the hard-coded
			// grey coming back in through the side door — the exact thing the
			// ground was what made safe.
			checkNoRGB(t, th)
			continue
		}
		ground++
		checkLegible(t, th)
	}
	if ground == 0 {
		t.Fatal("no theme paints a ground; the contrast half of this guard measured nothing")
	}
}

// A theme with no ground carries no RGB, in fg or in bg, in any role.
func checkNoRGB(t *testing.T, th Theme) {
	t.Helper()
	if len(th.Role) == 0 {
		t.Fatalf("%s has no roles at all", th.Name)
	}
	for style, paint := range th.Role {
		if !paint.fg.zero() {
			t.Errorf("%s: role %v picks a foreground RGB %v and paints no ground to read it against",
				th.Name, style, paint.fg)
		}
		if !paint.bg.zero() {
			t.Errorf("%s: role %v picks a background RGB %v and paints no ground to read it against",
				th.Name, style, paint.bg)
		}
	}
}

// A theme with no ground emits only weight and indexed colour.
//
// The RGB check above is about the table; this one is about what reaches the
// terminal, because the two are only the same while nothing between them
// invents a colour. 38;2 on a terminal that cannot draw it is not a fallback,
// it is text the reader cannot see.
func TestAThemeWithNoGroundEmitsOnlyWeightAndIndexedColour(t *testing.T) {
	var th Theme
	for _, c := range Themes() {
		if c.Ground.zero() {
			th = c
			break
		}
	}
	if th.Name == "" {
		t.Fatal("no theme without a ground; this guard is asking about nothing")
	}

	allowed := map[string]bool{"1": true, "2": true, "3": true}
	for n := 30; n <= 37; n++ {
		allowed[strconv.Itoa(n)] = true
	}
	for n := 40; n <= 47; n++ {
		allowed[strconv.Itoa(n)] = true
	}
	for n := 90; n <= 97; n++ {
		allowed[strconv.Itoa(n)] = true
	}
	for n := 100; n <= 107; n++ {
		allowed[strconv.Itoa(n)] = true
	}

	// Both depths, because a theme that is drawable anywhere must not change
	// what it emits when the terminal happens to be able to take more.
	for _, depth := range []Depth{DepthTrue, Depth256} {
		p := Palette{Enabled: true, Theme: th, Depth: depth}
		if got := p.Ground(); got != "" {
			t.Errorf("%s paints a ground: %q", th.Name, got)
		}
		for style := range th.Role {
			for _, param := range strings.Split(th.Role[style].sgr(depth), ";") {
				if param == "" {
					continue
				}
				if !allowed[param] {
					t.Errorf("%s: role %v emits SGR %q, which is neither a weight nor one of the sixteen",
						th.Name, style, param)
				}
			}
		}
	}
}

// Italic is given by the theme's table, and only one theme gives it.
//
// Stated as a guard rather than left to reading, because the failure it
// prevents is silent: an attribute added for one theme and set in the shared
// mapping changes all five, and nobody notices until the theme they use looks
// different for a reason nobody wrote down.
func TestItalicIsAnAttributeOfTheThemeTable(t *testing.T) {
	italic := map[string][]Style{}
	for _, th := range Themes() {
		for style, paint := range th.Role {
			if paint.italic {
				italic[th.Name] = append(italic[th.Name], style)
			}
		}
	}
	for _, th := range Themes() {
		if th.Ground.zero() {
			continue
		}
		if got := italic[th.Name]; len(got) > 0 {
			t.Errorf("%s paints a ground and still uses italic, in %v; the four with a ground do not",
				th.Name, got)
		}
	}
	claude := italic["claude"]
	if len(claude) != 1 || claude[0] != StyleReasoning {
		t.Errorf("the claude theme gives italic to %v; it is for the model's reasoning and nothing else", claude)
	}

	// And it reaches the screen as SGR 3, closed by SGR 23 — not by a reset,
	// which would take the row's ground with it in every other theme.
	p := Palette{Enabled: true, Theme: claudeThemeForTest(t)}
	got := p.Apply(StyleReasoning, "x")
	if !strings.Contains(got, "\x1b[2;3m") || !strings.Contains(got, "\x1b[22;23m") {
		t.Errorf("reasoning renders as %q, want it opened with 2;3 and closed with 22;23", got)
	}
}

func claudeThemeForTest(t *testing.T) Theme {
	t.Helper()
	for _, th := range Themes() {
		if th.Name == "claude" {
			return th
		}
	}
	t.Fatal("no claude theme")
	return Theme{}
}

func checkLegible(t *testing.T, th Theme) {
	t.Helper()
	for _, c := range []struct {
		style Style
		name  string
		min   float64
	}{
		{StyleProse, "prose", 4.5},
		{StyleHeading, "heading", 4.5},
		{StyleBold, "bold", 4.5},
		{StyleCode, "code", 3},
		{StyleMeta, "meta", 3},
		{StyleAccent, "accent", 3},
		{StyleOK, "ok", 3},
		{StyleError, "error", 3},
		{StyleWarn, "warn", 3},
		{StyleLaneYou, "lane you", 3},
		{StyleLaneAnswer, "lane answer", 3},
		{StyleHint, "hint", 1.8},
		{StyleLaneProcess, "lane process", 1.8},
		{StyleChrome, "chrome", 1.5},
	} {
		paint, ok := th.Role[c.style]
		if !ok {
			t.Errorf("%s has no colour in %s", c.name, th.Name)
			continue
		}
		if got := contrast(paint.fg, th.Ground); got < c.min {
			t.Errorf("%s: %s is %.2f:1 against the ground, want at least %.1f",
				th.Name, c.name, got, c.min)
		}
	}
}

// contrast is the WCAG ratio between two colours.
func contrast(a, b rgb) float64 {
	la, lb := luminance(a), luminance(b)
	if lb > la {
		la, lb = lb, la
	}
	return (la + 0.05) / (lb + 0.05)
}

func luminance(c rgb) float64 {
	f := func(v uint8) float64 {
		x := float64(v) / 255
		if x <= 0.03928 {
			return x / 12.92
		}
		return math.Pow((x+0.055)/1.055, 2.4)
	}
	return 0.2126*f(c.r) + 0.7152*f(c.g) + 0.0722*f(c.b)
}

// Colour switched off emits no escape at all, the ground included.
//
// The ground is the one that could have slipped through: it is painted once per
// row rather than around a run, so a palette that forgot to ask whether colour
// was on would tint every row of a NO_COLOR terminal and leave the reset
// nowhere.
func TestColourOffPaintsNoGround(t *testing.T) {
	off := Palette{}
	if got := off.Ground(); got != "" {
		t.Errorf("a disabled palette paints a ground: %q", got)
	}
	for _, s := range []Style{StyleProse, StyleAccent, StyleError, StyleChrome} {
		if got := off.Apply(s, "x"); got != "x" {
			t.Errorf("style %v emitted %q with colour off", s, got)
		}
	}

	m := modelWithPlan()
	m.Entries = append(m.Entries, Entry{Kind: KindTool, Tool: "read", Target: "a.go", Summary: "ok"})
	g := DefaultGeometry(100, 24)
	g.Palette = off
	if got := Render(m, g); strings.ContainsRune(got, 0x1b) {
		t.Errorf("an escape reached a screen with colour off:\n%q", got)
	}
}
