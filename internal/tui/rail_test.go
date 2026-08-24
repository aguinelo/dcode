package tui

import (
	"reflect"
	"strings"

	"github.com/aguinelo/dcode/internal/protocol"
	"testing"
)

func railModel(entries ...Entry) Model {
	return Model{Lang: En, Cursor: -1, Entries: entries}
}

func railGeometry(w int) Geometry {
	g := DefaultGeometry(w, 20)
	g.Palette = Palette{}
	return g
}

// A turn that touched nothing gets no column. A column of nothing costs the
// stream twenty characters and tells the reader something is missing.
func TestATurnThatTouchedNothingGetsNoSidebar(t *testing.T) {
	g := railGeometry(140)
	if g.ShowRail(false) {
		t.Error("an empty tree still opened the sidebar")
	}
	if g.StreamWidth(false, false) != 140 {
		t.Errorf("the stream lost width to a sidebar that is not drawn: %d", g.StreamWidth(false, false))
	}
}

// Below a hundred columns it goes, because the stream is what the work is in.
func TestTheSidebarDisappearsOnANarrowTerminal(t *testing.T) {
	if railGeometry(90).ShowRail(true) {
		t.Error("the sidebar survived a 90-column terminal")
	}
	if !railGeometry(120).ShowRail(true) {
		t.Error("the sidebar is missing at 120 columns")
	}
}

// And an explicit choice wins at any width, in BOTH directions. A keypress is
// the user noticing; responsiveness answers the case where they did not.
func TestAnExplicitChoiceWinsAtAnyWidthBothWays(t *testing.T) {
	forced := railGeometry(80)
	forced.RailMode = RailShown
	if !forced.ShowRail(true) {
		t.Error("an explicitly shown sidebar was hidden by the width rule")
	}

	hidden := railGeometry(200)
	hidden.RailMode = RailHidden
	if hidden.ShowRail(true) {
		t.Error("an explicitly hidden sidebar came back on a wide terminal")
	}
}

// Its width is the design's clamp, in cells.
func TestTheSidebarKeepsItsFloorAndItsCeiling(t *testing.T) {
	if got := railGeometry(100).railWidth(); got != 20 {
		t.Errorf("a fifth of 100 is 20, floor is 20; got %d", got)
	}
	if got := railGeometry(400).railWidth(); got != 30 {
		t.Errorf("the ceiling did not hold: %d", got)
	}
}

// The stream pays for both columns, counted once, in the function the layout
// and the renderer both read.
func TestTheStreamPaysForEveryColumnExactlyOnce(t *testing.T) {
	g := railGeometry(140)
	full := g.StreamWidth(true, true)
	if want := 140 - g.railWidth() - 1 - g.panelWidth() - 1; full != want {
		t.Errorf("got %d, want %d", full, want)
	}
	if railOnly := g.StreamWidth(true, false); railOnly != 140-g.railWidth()-1 {
		t.Errorf("the sidebar alone cost %d", 140-railOnly)
	}
}

// Movement is never the only clue. In ASCII a running row, a finished one and a
// failed one still differ by character — a pulse cannot be the difference on a
// terminal where nothing pulses.
func TestTheStatesStayApartWithoutUnicode(t *testing.T) {
	gl := railGlyphs(false)
	seen := map[string]string{}
	for name, mark := range map[string]string{
		"folder": gl.folder, "running": gl.running, "done": gl.done, "failed": gl.failed,
	} {
		if other, clash := seen[mark]; clash {
			t.Errorf("%s and %s are both %q; blocked would read as finished", name, other, mark)
		}
		seen[mark] = name
	}
}

// The sidebar emits no escape without colour, like everything else on screen.
func TestTheSidebarEmitsNoEscapeWithoutColour(t *testing.T) {
	m := railModel(
		Entry{Kind: KindTool, Tool: "write", Target: "a/b.go", Added: 12},
		Entry{Kind: KindTool, Tool: "read", Target: "a/c.go", Running: true},
		Entry{Kind: KindTool, Tool: "edit", Target: "d.go", IsError: true},
	)
	for _, line := range renderRail(m, railGeometry(120), 8) {
		if strings.ContainsRune(line, 0x1b) {
			t.Errorf("an escape survived a monochrome palette: %q", line)
		}
	}
}

// No row is wider than the column, whatever the path.
func TestNoSidebarRowOverflowsTheColumn(t *testing.T) {
	g := railGeometry(120)
	m := railModel(Entry{
		Kind: KindTool, Tool: "write", Added: 123456,
		Target: "internal/very/deeply/nested/directory/with/a/long/name/file.go",
	})
	for _, line := range renderRail(m, g, 6) {
		if w := visibleWidth(line); w > g.railWidth() {
			t.Errorf("a row is %d wide in a %d column: %q", w, g.railWidth(), line)
		}
	}
}

// The header says how much was touched even when the column is too narrow for
// the list, which is what makes a collapsed column still worth a glance.
func TestTheSidebarHeaderCountsWhatWasTouched(t *testing.T) {
	m := railModel(
		Entry{Kind: KindTool, Tool: "read", Target: "a/b.go"},
		Entry{Kind: KindTool, Tool: "read", Target: "a/c.go"},
	)
	head := renderRail(m, railGeometry(160), 5)[0]
	if !strings.Contains(head, "2 touched") {
		t.Errorf("the header does not count the files: %q", head)
	}
	if strings.Contains(head, "2 2") {
		t.Errorf("the count is printed twice: %q", head)
	}
}

// -- the conversations of this workspace -------------------------------------

func sessionsModel(open string, titles ...string) Model {
	m := railModel(Entry{Kind: KindTool, Tool: "read", Target: "a.go"})
	m.SessionID = open
	for i, ti := range titles {
		m.Sessions = append(m.Sessions, SessionChoice{ID: string(rune('a' + i)), Title: ti})
	}
	return m
}

// The open conversation is marked by a character, not only by colour. A
// selection shown in colour alone is no selection on a terminal without any —
// the rule copy mode already carries.
func TestTheOpenConversationIsMarkedByACharacter(t *testing.T) {
	m := sessionsModel("b", "first", "second")
	g := railGeometry(140)
	g.Palette = Palette{}
	lines := strings.Join(renderRail(m, g, 12), "\n")

	for _, want := range []string{"● second", "  first"} {
		if !strings.Contains(lines, want) {
			t.Errorf("expected %q in:\n%s", want, lines)
		}
	}
}

// A workspace with conversations and a turn that has touched nothing still has
// a column worth drawing. Asking only about files emptied it for the first
// minute of every session.
func TestConversationsAloneAreEnoughToOpenTheSidebar(t *testing.T) {
	m := Model{Lang: En, Cursor: -1, Sessions: []SessionChoice{{ID: "a", Title: "x"}}}
	if !m.railHasContent() {
		t.Error("a workspace with recorded conversations got no sidebar")
	}
	empty := Model{Lang: En, Cursor: -1}
	if empty.railHasContent() {
		t.Error("a session with nothing at all opened a column")
	}
}

// A title that had to be cut says so. One that merely stops leaves the reader
// unable to tell a short conversation from a truncated one.
func TestATruncatedTitleSaysItWasTruncated(t *testing.T) {
	m := sessionsModel("z", strings.Repeat("catalogar os contratos ", 4))
	g := railGeometry(120)
	g.Palette = Palette{}
	var row string
	for _, l := range renderRail(m, g, 12) {
		if strings.Contains(l, "catalogar") {
			row = l
			break
		}
	}
	if row == "" {
		t.Fatal("the conversation is not listed")
	}
	if !strings.HasSuffix(strings.TrimRight(row, " "), "…") {
		t.Errorf("a cut title does not say it was cut: %q", row)
	}
	if w := visibleWidth(row); w > g.railWidth() {
		t.Errorf("the row is %d wide in a %d column", w, g.railWidth())
	}
}

// Cut in cells, never in bytes: a rune is not a column.
func TestATitleIsCutInCellsAndNotInBytes(t *testing.T) {
	if got := trimTo("sessões", 4); visibleWidth(got) > 4 {
		t.Errorf("%q is %d columns, wanted at most 4", got, visibleWidth(got))
	}
	if trimTo("abc", 0) != "" {
		t.Error("a zero width produced text")
	}
}

// -- no box-drawing rune reaches a terminal that cannot draw one -------------

// Four separate literals in this package assumed Unicode: the column divider,
// the diff gutter, the running marker and the path ellipsis. Each was found by
// looking at an ASCII render rather than at the code, and each was found AFTER
// the previous one had been fixed.
//
// So the assertion is over the whole screen rather than over one of them: a
// fifth would otherwise wait for a fifth pair of eyes.
func TestNoBoxDrawingRuneSurvivesAsciiMode(t *testing.T) {
	m := railModel(
		Entry{Kind: KindTool, Tool: "read", Target: "docs/ROADMAP.md", Running: true},
		Entry{Kind: KindTool, Tool: "edit", Target: "internal/tui/rail.go", Added: 38,
			Diff: "--- a/x\n+++ b/x\n@@ -1,2 +1,3 @@\n+added\n"},
		Entry{Kind: KindTool, Tool: "read",
			Target: "internal/very/deeply/nested/path/that/must/be/shortened.go"},
	)
	m.Sessions = []SessionChoice{{ID: "s1", Title: "uma conversa"}}
	m.SessionID = "s1"

	g := DefaultGeometry(118, 18)
	g.Palette = Palette{}
	g.Unicode = false

	// The forbidden set is DERIVED from the Unicode glyph sets rather than
	// listed here. Listing them is what let the filter caret slip in as the
	// fifth of these: a new glyph would have to be remembered in two places,
	// and the second place is a test nobody edits when adding a mark.
	m.Nav = RailNav{Active: true, Filter: "co"}
	out := Render(m, g)
	for _, forbidden := range unicodeGlyphs() {
		if forbidden == "" || forbidden == " " {
			continue
		}
		if strings.Contains(out, forbidden) {
			t.Errorf("%q reached a terminal that declared it cannot draw one:\n%s",
				forbidden, out)
		}
	}
	// The two the layout reaches for directly, outside either set.
	for _, forbidden := range []string{"⋯", "│"} {
		if strings.Contains(out, forbidden) {
			t.Errorf("%q reached a terminal that declared it cannot draw one:\n%s",
				forbidden, out)
		}
	}
}

// unicodeGlyphs is every mark either set draws when the terminal can.
func unicodeGlyphs() []string {
	var out []string
	for _, v := range []any{glyphs(true), railGlyphs(true)} {
		rv := reflect.ValueOf(v)
		for i := 0; i < rv.NumField(); i++ {
			out = append(out, rv.Field(i).String())
		}
	}
	return out
}

// -- a column that hides itself says so --------------------------------------

// The sidebar disappears below a hundred columns, which is most terminals, and
// it said nothing at all. A column that had been built read as a column that
// had not — and the key that brings it back was documented only inside the
// column that was not on screen.
//
// The plan panel already carried this debt and already paid it; the sidebar
// inherited the behaviour and not the hint.
func TestASidebarHiddenByWidthSaysSoAndNamesTheKey(t *testing.T) {
	m := railModel(Entry{Kind: KindTool, Tool: "read", Target: "a.go"})
	g := railGeometry(80)

	if g.ShowRail(m.railHasContent()) {
		t.Fatal("the fixture is not narrow enough to hide the column")
	}
	status := renderStatus(m, g, false)
	if !strings.Contains(status, "^b") {
		t.Errorf("a hidden column does not name the key that brings it back: %q", status)
	}
}

// And it stops saying so once it is on screen: a hint for something already
// visible is noise, and noise is what teaches people to stop reading the line.
func TestAVisibleSidebarSaysNothingAboutItself(t *testing.T) {
	m := railModel(Entry{Kind: KindTool, Tool: "read", Target: "a.go"})
	g := railGeometry(140)

	if !g.ShowRail(m.railHasContent()) {
		t.Fatal("the fixture should show the column")
	}
	if status := renderStatus(m, g, false); strings.Contains(status, "^b") {
		t.Errorf("a visible column advertises itself: %q", status)
	}
}

// Nothing to show is not something to advertise. Offering a key for an empty
// column sends somebody to look at nothing.
func TestAnEmptySidebarIsNotAdvertised(t *testing.T) {
	m := Model{Lang: En, Cursor: -1}
	if status := renderStatus(m, railGeometry(80), false); strings.Contains(status, "^b") {
		t.Errorf("an empty column was advertised: %q", status)
	}
}

// The box-drawing guard above asks whether a known set of glyphs got through.
// That set is derived from the two glyph tables, so it only ever covers what
// those tables know about — and the approval modal, which is drawn entirely
// from literals and never appeared in a model this test built, was outside it
// from the day it was written.
//
// This asks the decisive question instead: in ASCII mode, is every rune ASCII?
// The model is built entirely from ASCII so that any rune above 127 in the
// output came from the layout drawing it, and it is checked with the modal
// open, which is the one screen where being unreadable is most expensive.
func TestAsciiModeDrawsNothingButAscii(t *testing.T) {
	m := railModel(
		Entry{Kind: KindTool, Tool: "read", Target: "docs/ROADMAP.md", Running: true},
		Entry{Kind: KindTool, Tool: "edit", Target: "internal/tui/rail.go", Added: 38, Removed: 2,
			Diff: "--- a/x\n+++ b/x\n@@ -1,2 +1,3 @@\n+added\n", Expanded: true},
		Entry{Kind: KindReasoning, Summary: "a thought", Expanded: true},
		Entry{Kind: KindAssistant, Summary: "a paragraph with `code` and a - list item"},
		Entry{Kind: KindTool, Tool: "read",
			Target: "internal/very/deeply/nested/path/that/must/be/shortened.go"},
	)
	m.Sessions = []SessionChoice{{ID: "s1", Title: "a conversation with a title too long for any column"}}
	m.SessionID = "s1"
	m.DiffAdded, m.DiffRemoved, m.DiffFiles = 38, 2, 3
	m.Rounds, m.MaxRounds, m.InFlight, m.MaxInFlight = 16, 20, 1, 4
	m.Plan = []protocol.PlanItem{
		{ID: 1, Text: "a plan item longer than the panel can hold", Status: protocol.PlanBlocked,
			Blocked: "the reason it is blocked, at length"},
	}

	for _, name := range []string{"stream", "modal"} {
		m := m
		if name == "modal" {
			m.Pending = &protocol.ApprovalRequest{
				ApprovalID: "a1", Tool: "bash", Command: "rm -rf /tmp/x",
				BoundaryCrossed: "the workspace", Reason: "it writes outside", Rule: "deny:rm",
			}
		}
		g := DefaultGeometry(118, 24)
		g.Palette = Palette{}
		g.Unicode = false

		for i, line := range lines(Render(m, g)) {
			for _, r := range line {
				if r > 127 {
					t.Errorf("%s: %q on line %d reached a terminal that declared ASCII:\n%s",
						name, r, i, line)
					break
				}
			}
		}
	}
}
