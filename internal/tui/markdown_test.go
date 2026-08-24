package tui

import (
	"strings"
	"testing"
)

func plainProse(t *testing.T, text string, w int) []string {
	t.Helper()
	g := DefaultGeometry(w+4, 24)
	g.Palette = Palette{} // no colour: what is left is the text itself
	return renderProse(text, w, g)
}

// The model writes markdown and the screen printed it raw: `**A**` arrived with
// its asterisks, a package name arrived with its backticks. Every answer with
// emphasis in it read as though the product had forgotten to finish something.
func TestEmphasisMarkersDoNotReachTheScreen(t *testing.T) {
	lines := plainProse(t, "Eu voto em **B** usando `node:sqlite`: já é nativo.", 80)
	got := strings.Join(lines, "\n")

	for _, marker := range []string{"**", "`"} {
		if strings.Contains(got, marker) {
			t.Errorf("the marker %q reached the screen:\n%s", marker, got)
		}
	}
	for _, want := range []string{"B", "node:sqlite", "já é nativo"} {
		if !strings.Contains(got, want) {
			t.Errorf("the text lost %q:\n%s", want, got)
		}
	}
}

// Prose is dim and the technical term returns to normal brightness — the
// design's own way of separating them, and it costs no new colour.
func TestProseIsDimAndCodeIsNot(t *testing.T) {
	g := DefaultGeometry(84, 24)
	lines := renderProse("o projeto só usa `better-sqlite3` hoje", 80, g)
	got := strings.Join(lines, "\n")

	dim := g.Palette.Apply(StyleDim, "x")
	if dim == "x" {
		t.Skip("no colour in this environment")
	}
	if !strings.Contains(got, "\x1b[2m") {
		t.Errorf("the prose is not dim:\n%q", got)
	}
	// The code run is not inside a dim sequence: it closes before, and reopens
	// after. Otherwise "brighter than the prose" is not what the eye gets.
	before := strings.Index(got, "better-sqlite3")
	if before < 0 {
		t.Fatalf("the code is missing:\n%q", got)
	}
	if !strings.Contains(got[:before], "\x1b[0m") {
		t.Errorf("the dim run was never closed before the code:\n%q", got)
	}
}

// A bold phrase that crosses a wrap boundary stays bold on both lines. Styling
// whole runs and then wrapping would put the escape on one line and the reset
// on another, which paints the rest of the screen.
func TestEmphasisSurvivesAWrapBoundary(t *testing.T) {
	text := "isso é **uma frase inteira em negrito que não cabe numa linha só** e segue"
	lines := plainProse(t, text, 24)

	if len(lines) < 2 {
		t.Fatalf("the text did not wrap: %v", lines)
	}
	joined := strings.Join(lines, " ")
	if strings.Contains(joined, "**") {
		t.Errorf("a marker survived the wrap:\n%v", lines)
	}
	for _, want := range []string{"negrito", "linha", "segue"} {
		if !strings.Contains(joined, want) {
			t.Errorf("the wrap lost %q:\n%v", want, lines)
		}
	}
}

// No line is ever wider than the column it was given, styled or not. Measuring
// the escape sequences would report a coloured line as too wide and a plain one
// as fitting, which is the layout bug that only appears on colour terminals.
func TestStyledProseNeverExceedsItsColumn(t *testing.T) {
	text := "a lib que conversa com SQLite é **`better-sqlite3`**, e ela precisa ser " +
		"compilada — o que esbarra no `Xcode/CLT` ausente na máquina"

	for _, w := range []int{20, 32, 48, 72, 100} {
		g := DefaultGeometry(w+4, 24)
		for _, line := range renderProse(text, w, g) {
			if n := visibleWidth(line); n > w {
				t.Errorf("width %d: a line is %d cells:\n%q", w, n, line)
			}
		}
	}
}

// A fenced block is what the person is meant to run. It gets the same
// continuation bar a diff gets — the screen already means "this is verbatim"
// with that shape — and the fences themselves never show.
func TestAFencedBlockGetsTheContinuationBarAndLosesItsFences(t *testing.T) {
	text := "Você roda uma vez:\n\n```sh\nxcode-select --install\n```\n\nDepois me avisa."
	lines := plainProse(t, text, 60)
	got := strings.Join(lines, "\n")

	if strings.Contains(got, "```") || strings.Contains(got, "sh\n") {
		t.Errorf("a fence reached the screen:\n%s", got)
	}
	var barred bool
	for _, l := range lines {
		if strings.Contains(l, "xcode-select --install") {
			barred = strings.Contains(l, "│") || strings.Contains(l, "|")
		}
	}
	if !barred {
		t.Errorf("the command has no continuation bar:\n%s", got)
	}
	if !strings.Contains(got, "Depois me avisa") {
		t.Errorf("the prose after the block is gone:\n%s", got)
	}
}

// A block is verbatim. Wrapping a command would produce a second line that
// looks runnable and is not, and re-flowing indentation would change what the
// person copies.
func TestAFencedBlockIsNotReflowed(t *testing.T) {
	text := "```\n  indented line kept as written\n```"
	lines := plainProse(t, text, 60)
	var found bool
	for _, l := range lines {
		if strings.Contains(l, "  indented line kept as written") {
			found = true
		}
	}
	if !found {
		t.Errorf("the block was reflowed:\n%v", lines)
	}
}

// A marker with no partner is text. Eating it would delete something the model
// wrote, and the model writes `**` in prose about markdown often enough.
func TestAnUnpairedMarkerIsLeftAlone(t *testing.T) {
	for _, text := range []string{
		"2 ** 8 é potência",
		"use a crase ` para código",
		"**começa e não fecha",
	} {
		got := strings.Join(plainProse(t, text, 80), "\n")
		if !strings.Contains(got, strings.Fields(text)[0]) {
			t.Errorf("%q lost its opening: %q", text, got)
		}
	}
}

// Lists start with `*` and are not italics. Treating a single asterisk as
// emphasis would eat the bullet of every list the model writes.
func TestASingleAsteriskIsNotEmphasis(t *testing.T) {
	got := strings.Join(plainProse(t, "* primeiro item\n* segundo item", 60), "\n")
	if strings.Count(got, "*") != 2 {
		t.Errorf("the bullets were eaten or doubled:\n%s", got)
	}
}

// End to end: what the model wrote reaches the screen formatted, not marked up.
// The unit tests above prove the renderer; this proves the stream uses it.
func TestTheStreamRendersWhatTheModelWroteAsFormatting(t *testing.T) {
	m := NewModel("s1", "/w/proj", "m", "workspace-write", PtBR)
	m.Entries = []Entry{{
		Kind: KindAssistant,
		Summary: "Eu voto em **B** usando `node:sqlite`: já é nativo no seu Node 24.\n\n" +
			"```sh\nnpm install\n```\n\nDepois eu subo o servidor.",
	}}

	g := DefaultGeometry(100, 30)
	g.Palette = Palette{}
	out := Render(m, g)

	for _, marker := range []string{"**", "```", "`node"} {
		if strings.Contains(out, marker) {
			t.Errorf("the marker %q is on screen:\n%s", marker, out)
		}
	}
	for _, want := range []string{"node:sqlite", "npm install", "Depois eu subo"} {
		if !strings.Contains(out, want) {
			t.Errorf("the screen lost %q:\n%s", want, out)
		}
	}
}

// While a turn is streaming, every emphasised word arrives as `**` first and
// its partner some deltas later, so the screen flashed a pair of asterisks
// before each of them. On a real session `1. **` sat alone on a line as the
// last thing the reader saw of every heading the model wrote.
func TestAMarkerStillArrivingIsNotDrawn(t *testing.T) {
	g := DefaultGeometry(60, 20)
	g.Palette = Palette{}
	got := func(src string) string {
		return strings.Join(renderProse(src, 50, g), "|")
	}

	for _, c := range []struct{ src, want string }{
		// Opened at the end, nothing after it: it has not finished arriving.
		{"1. **", "1."},
		{"veja `", "veja"},
		{"a **b** c **", "a b c"},

		// A PAIR that closes at the end is finished, and the first version of
		// this took the closing marker off it.
		{"1. **Alvo**", "1. Alvo"},
		{"veja `x`", "veja x"},

		// Opened and left, with words after it: written on purpose, so it
		// stays. Deleting it would delete something somebody typed.
		{"1. **Alvo", "1. **Alvo"},
		{"a ** b", "a ** b"},
	} {
		if g := got(c.src); g != c.want {
			t.Errorf("renderProse(%q) = %q, want %q", c.src, g, c.want)
		}
	}
}
