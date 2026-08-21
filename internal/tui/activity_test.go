package tui

import (
	"strings"
	"testing"
	"time"

	"github.com/aguinelo/dcode/internal/protocol"
)

// working builds a model mid-turn with one running tool.
func working(tool, target string, frame int) Model {
	m := Model{State: protocol.SessionStateRunning, Frame: frame, Lang: En}
	if tool != "" {
		m.Entries = []Entry{{Kind: KindTool, Tool: tool, Target: target, Running: true}}
	}
	return m
}

func plainGeometry() Geometry {
	g := DefaultGeometry(120, 40)
	g.ActivityVerbs = true
	return g
}

// The rule the whole feature hangs on. A gerund with nothing beside it is
// motion pretending to be information: the screen looks alive and the reader
// learns nothing, which is worse than a still screen that says one true word.
func TestTheVerbNeverAppearsWithoutTheToolItDescribes(t *testing.T) {
	for _, frame := range []int{0, 20, 40, 60, 120, 999} {
		line := renderWorking(working("", "", frame), plainGeometry())
		for _, verb := range activityVerbs[En][phaseOther] {
			if strings.Contains(line, verb) {
				t.Errorf("frame %d: a verb was drawn with no running tool: %q", frame, line)
			}
		}
	}
}

// And where there IS a tool, the fact stays on the line beside the verb — the
// verb accompanies it, never replaces it.
func TestTheVerbRidesBesideTheToolAndItsTarget(t *testing.T) {
	line := renderWorking(working("grep", `\.Save\(`, 0), plainGeometry())
	if !strings.Contains(line, "grep") {
		t.Errorf("the tool left the line: %q", line)
	}
	if !strings.Contains(line, `\.Save\(`) {
		t.Errorf("the target left the line: %q", line)
	}
	if !strings.Contains(line, "reading") {
		t.Errorf("no verb for a reading tool: %q", line)
	}
}

// 2.4 seconds at a 120ms tick. Asserted in frames because that is what the
// render reads: a duration here and a counter there is two sources for one
// rhythm, and that is how an animation drifts from what it animates.
func TestTheVerbHoldsForTwentyFramesAndThenChanges(t *testing.T) {
	first := ActivityVerb("read", 0, En)
	if ActivityVerb("read", verbFrames-1, En) != first {
		t.Error("the verb changed before its twenty frames were up")
	}
	if second := ActivityVerb("read", verbFrames, En); second == first {
		t.Errorf("the verb did not change at frame %d: still %q", verbFrames, first)
	}
}

// It cycles rather than running out, and it never panics on a frame counter
// that has been running all afternoon.
func TestTheVerbCyclesAndSurvivesAnyFrame(t *testing.T) {
	n := len(activityVerbs[En][phaseReading])
	if got := ActivityVerb("read", verbFrames*n, En); got != ActivityVerb("read", 0, En) {
		t.Errorf("the set did not wrap around: %q", got)
	}
	for _, frame := range []int{-1, 0, 1 << 20} {
		if ActivityVerb("read", frame, En) == "" {
			t.Errorf("frame %d produced no verb for a known tool", frame)
		}
	}
}

// A tool nobody has grouped yet reads as ordinary work rather than vanishing
// from the line. A tool added later must not silently lose its verb.
func TestAnUnknownToolStillGetsAVerb(t *testing.T) {
	if got := ActivityVerb("some-tool-invented-tomorrow", 0, En); got == "" {
		t.Error("an unknown tool produced no verb")
	}
}

// Off means off: the line keeps its facts and loses only the word.
func TestTurningTheVerbsOffLeavesTheFactsAlone(t *testing.T) {
	g := plainGeometry()
	g.ActivityVerbs = false
	line := renderWorking(working("grep", "needle", 0), g)
	if strings.Contains(line, "reading") {
		t.Errorf("a verb was drawn with verbs turned off: %q", line)
	}
	if !strings.Contains(line, "grep") || !strings.Contains(line, "needle") {
		t.Errorf("turning verbs off took the facts with them: %q", line)
	}
}

func TestTheVerbSettingDefaultsToOn(t *testing.T) {
	if !ActivityVerbsEnabled(func(string) string { return "" }) {
		t.Error("verbs are off by default")
	}
	for _, off := range []string{"0", "false", "off", "no", "FALSE", " off "} {
		if ActivityVerbsEnabled(func(string) string { return off }) {
			t.Errorf("%q did not turn the verbs off", off)
		}
	}
}

// Every declared language covers every phase. Strings has a guard for its own
// fields and it reads them with reflect.Value.String(), which returns something
// non-empty for a slice whatever it holds — so a catalogue of lists that leaned
// on that guard would be checked by nothing at all.
func TestEveryLanguageHasAVerbForEveryPhase(t *testing.T) {
	for _, lang := range Languages() {
		set, ok := activityVerbs[lang]
		if !ok {
			t.Errorf("%s has no verbs at all", lang)
			continue
		}
		for _, phase := range activityPhases() {
			if len(set[phase]) == 0 {
				t.Errorf("%s has no verb for %s", lang, phase)
			}
		}
	}
}

// A language nobody declared falls back rather than losing the word, the same
// way Text does for every other string.
func TestAnUndeclaredLanguageStillGetsAVerb(t *testing.T) {
	if got := ActivityVerb("read", 0, Lang("kl-GL")); got == "" {
		t.Error("an undeclared language lost the verb instead of falling back")
	}
}

// -- the tick stops when there is nothing to animate --------------------------

// The comment on this path has promised it since it was written: "an idle
// screen that keeps repainting burns a laptop battery for no information."
// The frame counter did stop; the tick rescheduled anyway, so the screen
// repainted eight times a second for a number that never moved.
//
// A sentence that describes what the code does not do is the shape this
// repository keeps finding, and it is cheaper to catch here than in a battery
// report nobody files.
func TestTheTickStopsWhenTheSessionIsIdle(t *testing.T) {
	p := &program{model: Model{State: protocol.SessionStateIdle}, opts: Options{Now: func() time.Time { return time.Time{} }}}
	p.ticking = true

	_, cmd := p.Update(tickMsg(time.Time{}))
	if cmd != nil {
		t.Error("an idle session scheduled another tick")
	}
	if p.ticking {
		t.Error("the program still believes a tick is in flight")
	}
}

func TestTheTickKeepsGoingWhileATurnRuns(t *testing.T) {
	p := &program{model: Model{State: protocol.SessionStateRunning}, opts: Options{Now: func() time.Time { return time.Time{} }}}
	p.ticking = true

	_, cmd := p.Update(tickMsg(time.Time{}))
	if cmd == nil {
		t.Fatal("a running turn stopped animating")
	}
	if p.model.Frame != 1 {
		t.Errorf("the frame did not advance: %d", p.model.Frame)
	}
}

// And exactly one tick comes back when work starts again. Without the guard
// every event would add one, and the frame counter would sprint — motion that
// says the machine is busier than it is.
func TestWorkStartingAgainRestartsExactlyOneTick(t *testing.T) {
	p := &program{model: Model{State: protocol.SessionStateRunning}}
	p.ticking = false

	if c := p.resumeTicking(); c == nil {
		t.Fatal("a running turn did not restart the tick")
	}
	if c := p.resumeTicking(); c != nil {
		t.Error("a second tick was scheduled while one was already in flight")
	}
}

func TestAnIdleSessionDoesNotRestartTheTick(t *testing.T) {
	p := &program{model: Model{State: protocol.SessionStateIdle}}
	if c := p.resumeTicking(); c != nil {
		t.Error("an idle session started animating")
	}
}

// -- the two invariants that make colour removable ---------------------------

// Monochrome emits no escape at all, not even a reset. The verb is new text on
// a line that already had some, and new text is exactly where this slips.
func TestTheActivityLineEmitsNoEscapeWithoutColour(t *testing.T) {
	g := plainGeometry()
	g.Palette = Palette{}
	for _, tool := range []string{"", "grep", "write", "explore", "bash"} {
		line := renderWorking(working(tool, "some/path.go", 3*verbFrames), g)
		if strings.ContainsRune(line, 0x1b) {
			t.Errorf("tool %q: an escape survived a monochrome palette: %q", tool, line)
		}
	}
}

// Style never changes measured width. Colour on and colour off must lay out the
// same, or a monochrome terminal wraps where a colour one does not.
func TestStylingTheActivityLineDoesNotChangeItsWidth(t *testing.T) {
	coloured := plainGeometry()
	coloured.Palette = Palette{Enabled: true}
	plain := plainGeometry()
	plain.Palette = Palette{}

	for _, tool := range []string{"", "grep", "explore"} {
		m := working(tool, "internal/tui/render.go", verbFrames)
		a := visibleWidth(renderWorking(m, coloured))
		b := visibleWidth(renderWorking(m, plain))
		if a != b {
			t.Errorf("tool %q: %d columns with colour, %d without", tool, a, b)
		}
	}
}

// The whole line speaks one language, the way-out included.
//
// It said "^C interrupts" in English under a Portuguese interface, which the
// verb made obvious by sitting right beside it: `lendo grep … ^C interrupts`.
// Half a sentence in each language is worse than either alone, because it reads
// as a bug in the product rather than as a missing translation.
func TestTheWayOutSpeaksTheSameLanguageAsTheLine(t *testing.T) {
	for _, c := range []struct {
		lang Lang
		want string
	}{{PtBR, "^C interrompe"}, {En, "^C interrupts"}} {
		m := working("grep", "needle", 0)
		m.Lang = c.lang
		line := renderWorking(m, plainGeometry())
		if !strings.Contains(line, c.want) {
			t.Errorf("%s: the way out is not in the interface language: %q", c.lang, line)
		}
	}
}
