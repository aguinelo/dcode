package tui

import (
	"os"
	"strings"
	"testing"

	"github.com/mattn/go-runewidth"
)

// The glyphs this layout is drawn out of, and what they measure.
//
// Every one of these is AMBIGUOUS in Unicode's East Asian Width table, which is
// what makes them the defect's surface: `go-runewidth` reads the locale in its
// own init() and doubles all of them at once.
//
// The list is measured rather than recalled, and the Fatal below is what keeps
// it that way — writing it from memory put `◦` in it, which is narrow
// everywhere, and a guard that asserts nothing about a character it thinks it
// is testing is worse than no guard. Not every mark is affected: `⏺`, `✻`, `⊘`,
// `✓`, `❯`, `╎`, `▸`, `▾`, `◦` stay one cell in every locale.
var ambiguous = []string{"│", "└", "┌", "┐", "┘", "─", "├", "┃", "·", "▌", "▏", "━", "…", "●"}

// The locale says which language the person reads. It does not say how many
// cells their terminal gives a vertical rule.
//
// This is the whole defect in one assertion. With LANG=ja_JP.UTF-8,
// supportsUnicode reaches for the box-drawing set — the locale is UTF-8 — and
// runewidth measures that same set at two cells, from the same variable. Every
// frame came out at twice the terminal's width: the input frame, the approval
// modal, the rail, the mascot, every rule.
func TestTheLocaleDoesNotDecideHowWideAGlyphIs(t *testing.T) {
	// The hostile global, restored afterwards. Set rather than mocked: the
	// point is that this package's measurements survive a process whose locale
	// says East Asian, and only the real global can say that.
	defer func(prev bool) {
		runewidth.DefaultCondition.EastAsianWidth = prev
		runewidth.EastAsianWidth = prev
	}(runewidth.EastAsianWidth)
	runewidth.DefaultCondition.EastAsianWidth = true
	runewidth.EastAsianWidth = true

	for _, g := range ambiguous {
		if got := runewidth.StringWidth(g); got != 2 {
			t.Fatalf("%q measures %d through the global; this guard is no longer hostile and proves nothing", g, got)
		}
		if got := ruler.StringWidth(g); got != 1 {
			t.Errorf("%q measures %d cells through this package's ruler, want 1", g, got)
		}
	}

	// And the screen itself, which is the thing the reader loses. Widths from
	// the guard that found it: at 80 columns every framed row came back at 160.
	m := modelWithPlan()
	m.Entries = append(m.Entries,
		Entry{Kind: KindTool, Tool: "read", Target: "internal/tui/render.go", Summary: "240 linhas"},
		Entry{Kind: KindReasoning, Summary: "olhando o layout", Closed: false})
	for _, w := range []int{40, 60, 80, 100, 200} {
		g := DefaultGeometry(w, 24)
		g.Unicode = true
		for i, line := range strings.Split(Render(m, g), "\n") {
			if got := ruler.StringWidth(stripANSI(line)); got > w {
				t.Errorf("width %d: line %d is %d cells; anything past the edge overwrites what was there",
					w, i, got)
			}
		}
	}
}

// Nothing in this package measures with anything but the package's ruler.
//
// The assertion above is about behaviour and would keep passing if half the
// call sites went back to the global: the failure only appears in a locale the
// suite does not run in, which is precisely how this arrived. So the guard is
// asked about the SOURCE, the same way the box-drawing guard is asked about the
// whole screen rather than about a list of glyphs somebody remembered.
//
// Tests are held to it too. A guard that measures with a different ruler than
// the product draws with is a guard reporting failures nobody has and missing
// the ones they do.
func TestNothingMeasuresWithTheGlobalRuler(t *testing.T) {
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	checked := 0
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") || e.Name() == "width.go" {
			continue
		}
		data, err := os.ReadFile(e.Name())
		if err != nil {
			t.Fatal(err)
		}
		src := string(data)
		checked++
		// width_test.go holds the hostile global on purpose: it is the only
		// place that has to reach the thing it is defending against.
		if e.Name() == "width_test.go" {
			continue
		}
		for _, call := range []string{"runewidth.StringWidth(", "runewidth.RuneWidth(", "runewidth.Truncate("} {
			if strings.Contains(src, call) {
				t.Errorf("%s calls %s…; measure with `ruler`, which the process's locale cannot move",
					e.Name(), call)
			}
		}
	}
	if checked == 0 {
		t.Fatal("no source files were read; the guard would pass vacuously")
	}
}
