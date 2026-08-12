package tui

import (
	"strings"
	"testing"

	"github.com/aguinelo/dcode/internal/protocol"
)

func barModel() Model {
	m := NewModel("s1", "/Users/ada/work/acme/checkout-flow-v2", "MiniMax-M3", "workspace-write", PtBR)
	m.DiffAdded, m.DiffRemoved, m.DiffFiles = 96, 14, 8
	return m
}

// The bar is one line and stays one line. It is the only thing on screen that
// is always true — where you are, what has changed, what is waiting — and a
// second row would push it into the layout the rest of the screen owns.
func TestTheStatusBarIsAlwaysOneLineAndNeverOverflows(t *testing.T) {
	m := barModel()
	m.Pending = &protocol.ApprovalRequest{ApprovalID: "a1", Tool: "bash"}

	for _, w := range []int{20, 30, 40, 60, 80, 100, 140, 200} {
		for _, unicode := range []bool{true, false} {
			g := DefaultGeometry(w, 24)
			g.Unicode = unicode
			out := RenderStatusBar(m, g)

			if strings.Contains(out, "\n") {
				t.Fatalf("width %d: the bar wrapped:\n%q", w, out)
			}
			if n := visibleWidth(out); n > w {
				t.Fatalf("width %d, unicode %v: the bar is %d cells:\n%q", w, unicode, n, out)
			}
		}
	}
}

// Where you are is the most expensive information on the screen once there is
// more than one worktree, so it is the segment that never gives ground.
func TestTheWorktreeSurvivesEveryWidth(t *testing.T) {
	m := barModel()
	for _, w := range []int{20, 30, 50, 100, 200} {
		out := RenderStatusBar(m, DefaultGeometry(w, 24))
		if !strings.Contains(out, "checkout") {
			t.Errorf("width %d: the worktree is gone:\n%q", w, out)
		}
	}
}

// Something waiting on a person outranks everything except where they are. It
// is the one segment whose absence is the message — a badge showing zero is a
// badge people learn to ignore.
func TestWhatIsWaitingNeverDropsAndVanishesWhenThereIsNothing(t *testing.T) {
	m := barModel()
	if out := RenderStatusBar(m, DefaultGeometry(120, 24)); strings.Contains(out, "◉") {
		t.Errorf("nothing is waiting and the bar shows a badge:\n%q", out)
	}

	m.Pending = &protocol.ApprovalRequest{ApprovalID: "a1", Tool: "bash"}
	for _, w := range []int{24, 40, 80, 160} {
		out := RenderStatusBar(m, DefaultGeometry(w, 24))
		if !strings.Contains(out, "◉") && !strings.Contains(out, "1") {
			t.Errorf("width %d: something is waiting and the bar does not say so:\n%q", w, out)
		}
	}
}

// The drop order is from the right and it is documented: what a person can
// reconstruct elsewhere goes first. The diff is on the diff tab; where you are
// is nowhere else.
func TestTheDiffGivesGroundBeforeTheWorktreeDoes(t *testing.T) {
	m := barModel()
	wide := RenderStatusBar(m, DefaultGeometry(120, 24))
	if !strings.Contains(wide, "96") {
		t.Fatalf("the diff is missing on a wide terminal:\n%q", wide)
	}

	narrow := RenderStatusBar(m, DefaultGeometry(28, 24))
	if strings.Contains(narrow, "96") {
		t.Errorf("the diff survived a terminal too narrow for it:\n%q", narrow)
	}
	if !strings.Contains(narrow, "checkout") {
		t.Errorf("the worktree was dropped before the diff:\n%q", narrow)
	}
}

// A session that has changed nothing has no diff to report, and reporting
// `+0 −0` would be noise dressed as information.
func TestASessionThatChangedNothingShowsNoDiff(t *testing.T) {
	m := NewModel("s1", "/w/proj", "m", "workspace-write", PtBR)
	out := RenderStatusBar(m, DefaultGeometry(120, 24))
	if strings.Contains(out, "+0") || strings.Contains(out, "0 arq") {
		t.Errorf("an untouched session reports a diff:\n%q", out)
	}
}

// Amber carries structure here, and a terminal without colour has to carry the
// same structure some other way. The text says it, so the bar survives being
// stripped.
func TestTheBarSaysWhatItMeansWithoutColour(t *testing.T) {
	m := barModel()
	m.Pending = &protocol.ApprovalRequest{ApprovalID: "a1", Tool: "bash"}

	g := DefaultGeometry(120, 24)
	g.Palette = Palette{}
	out := RenderStatusBar(m, g)

	if strings.Contains(out, "\x1b") {
		t.Errorf("monochrome emitted an escape: %q", out)
	}
	for _, want := range []string{"checkout-flow-v2", "96", "14"} {
		if !strings.Contains(out, want) {
			t.Errorf("without colour the bar lost %q:\n%q", want, out)
		}
	}
}

// The accumulated diff is what the agent has changed in this worktree, summed
// from what each tool reported rather than parsed back out of its text.
func TestTheDiffAccumulatesAcrossTheTurn(t *testing.T) {
	m := NewModel("s1", "/w/proj", "m", "workspace-write", PtBR)
	m = m.Apply(ev(t, 1, protocol.EventToolCompleted, protocol.ToolCompleted{
		ToolCallID: "c1", OK: true, Added: 8, Removed: 3,
	}))
	m = m.Apply(ev(t, 2, protocol.EventToolCompleted, protocol.ToolCompleted{
		ToolCallID: "c2", OK: true, Added: 12, Removed: 1,
	}))

	if m.DiffAdded != 20 || m.DiffRemoved != 4 {
		t.Errorf("accumulated +%d −%d, want +20 −4", m.DiffAdded, m.DiffRemoved)
	}
	if m.DiffFiles != 2 {
		t.Errorf("touched %d files, want 2", m.DiffFiles)
	}

	// A call that changed nothing is not a file touched.
	m = m.Apply(ev(t, 3, protocol.EventToolCompleted, protocol.ToolCompleted{
		ToolCallID: "c3", OK: true, Lines: 200,
	}))
	if m.DiffFiles != 2 {
		t.Errorf("a read counted as a file touched: %d", m.DiffFiles)
	}
}

// The bar is drawn, on the last line, on every screen. A region that is true
// regardless of what the stream shows has to be somewhere a person can look
// without moving — and that is worth a row taken from the stream.
func TestTheBarIsTheLastLineOfEveryScreen(t *testing.T) {
	for _, m := range []Model{
		barModel(),
		modelWithPlan(),
		NewModel("s1", "/w/fresh", "m", "workspace-write", PtBR), // the empty state
	} {
		g := DefaultGeometry(100, 24)
		lines := strings.Split(Render(m, g), "\n")
		last := lines[len(lines)-1]
		if !strings.Contains(last, "⎇") {
			t.Errorf("the last line is not the bar:\n%q", last)
		}
		if len(lines) > g.Height {
			t.Errorf("the screen is %d lines in a %d-line terminal", len(lines), g.Height)
		}
	}
}
