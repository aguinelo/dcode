package tui

import (
	"errors"
	"strings"
	"testing"

	"github.com/aguinelo/dcode/internal/protocol"
)

// A steer that fails for any reason other than "the turn already ended" is a
// real failure and is reported as one.
//
// Only the one refusal becomes the queue. Swallowing the rest would turn a
// broken socket into a message that silently never arrived — which is the
// failure this whole path exists to prevent.
func TestASteerThatFailsForAnyOtherReasonIsReported(t *testing.T) {
	p, tr := newProgram(t)
	tr.steerErr = errors.New("the socket is gone")
	p.model.State = protocol.SessionStateRunning

	msg := p.steer("use tabs")()
	if _, ok := msg.(steerLateMsg); ok {
		t.Fatal("a transport failure was treated as a turn that had ended")
	}
	e, ok := msg.(errMsg)
	if !ok {
		t.Fatalf("got %T, want the failure reported", msg)
	}
	if !strings.Contains(e.err.Error(), "socket") {
		t.Errorf("the reported error lost what happened: %v", e.err)
	}
}

// A steer that succeeds says nothing: the turn.steered event is what puts it on
// screen, and a second message would show it twice.
func TestASteerThatWorksProducesNoMessage(t *testing.T) {
	p, tr := newProgram(t)
	p.model.State = protocol.SessionStateRunning

	if msg := p.steer("use tabs")(); msg != nil {
		t.Errorf("got %T, want nothing — the event carries it", msg)
	}
	if got := tr.steers(); len(got) != 1 || got[0] != "use tabs" {
		t.Errorf("steered %v", got)
	}
}

// A late correction arriving while ANOTHER turn is already running goes back to
// the queue rather than being steered into work it was not about.
func TestALateCorrectionDuringANewTurnIsQueued(t *testing.T) {
	p, tr := newProgram(t)
	p.model.State = protocol.SessionStateRunning

	p.Update(steerLateMsg{"use tabs"})
	if len(tr.submits()) != 0 {
		t.Error("a late correction opened a turn while one was running")
	}
	if len(p.model.Queue) != 1 || p.model.Queue[0] != "use tabs" {
		t.Errorf("queue = %v", p.model.Queue)
	}
}

// ---------- the palette ----------

// A style number nothing knows how to draw comes out as the plain text. Emitting an
// empty escape sequence would put invisible bytes in what gets copied.
func TestAnUnknownStyleLeavesTheTextAlone(t *testing.T) {
	p := Palette{Enabled: true}
	if got := p.Apply(Style(200), "hello"); got != "hello" {
		t.Errorf("got %q, want the text untouched", got)
	}
	// And the ordinary refusals: colour off, no style, nothing to draw.
	if got := (Palette{}).Apply(StyleError, "hello"); got != "hello" {
		t.Errorf("with colour off: %q", got)
	}
	if got := p.Apply(StyleNone, "hello"); got != "hello" {
		t.Errorf("with no style: %q", got)
	}
	if got := p.Apply(StyleError, ""); got != "" {
		t.Errorf("with no text: %q", got)
	}
	// A style it does know wraps, and closes exactly what it opened.
	//
	// A plain reset would undo the GROUND as well — the background is painted
	// once per row — so a styled run would punch a hole in it. Closing what was
	// opened leaves the row's ground alone, and costs four bytes instead of a
	// second background escape per run.
	got := p.Apply(StyleError, "hello")
	if !strings.HasPrefix(got, "\x1b[") {
		t.Errorf("a known style did not wrap: %q", got)
	}
	if strings.Contains(got, "\x1b[0m") {
		t.Errorf("a styled run reset everything and took the ground with it: %q", got)
	}
	if !strings.HasSuffix(got, "\x1b[39m") {
		t.Errorf("a coloured run did not put the foreground back: %q", got)
	}
}

// ---------- keeping the cursor on screen ----------

// With no entry under the cursor there is nothing to scroll to, and the view
// stays where the person left it.
func TestNothingToScrollToLeavesTheViewAlone(t *testing.T) {
	g := DefaultGeometry(80, 24)
	for _, m := range []Model{
		{Cursor: -1, ScrollTop: 5},
		{Cursor: 3, ScrollTop: 5}, // no entries at all
	} {
		if got := m.EnsureCursorVisible(g); got.ScrollTop != 5 {
			t.Errorf("scroll moved from 5 to %d with nothing to move to", got.ScrollTop)
		}
	}
}

// An entry already on screen does not move the view. Scrolling on every cursor
// move would make the stream jump under someone reading it.
func TestAnEntryAlreadyOnScreenDoesNotScroll(t *testing.T) {
	g := DefaultGeometry(80, 24)
	m := Model{
		Entries: []Entry{
			{Kind: KindUser, Summary: "one"},
			{Kind: KindUser, Summary: "two"},
		},
		Cursor: 1,
	}
	if got := m.EnsureCursorVisible(g); got.ScrollTop != 0 {
		t.Errorf("scroll moved to %d for an entry already visible", got.ScrollTop)
	}
}

// A cursor outside the view brings the view to it, in both directions.
//
// Asserted as the property rather than as a number: what the function promises
// is that the cursor's rows land inside the window, and pinning the exact scroll
// position would be a test of the arithmetic instead of the guarantee.
func TestTheViewFollowsTheCursorBothWays(t *testing.T) {
	g := DefaultGeometry(80, 10)
	var entries []Entry
	for i := 0; i < 40; i++ {
		entries = append(entries, Entry{Kind: KindUser, Summary: "line"})
	}

	for _, tc := range []struct {
		name   string
		cursor int
		from   int
	}{
		{"far above the view", 0, 30},
		{"far below the view", 39, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := Model{Entries: entries, Cursor: tc.cursor, ScrollTop: tc.from}
			got := m.EnsureCursorVisible(g)

			start, end := entryRows(got, g, tc.cursor)
			if start < 0 {
				t.Fatalf("the cursor's entry has no rows")
			}
			h := BodyHeight(got, g)
			if start < got.ScrollTop || end >= got.ScrollTop+h {
				t.Errorf("entry rows %d-%d are outside the window [%d,%d)",
					start, end, got.ScrollTop, got.ScrollTop+h)
			}
			if got.ScrollTop < 0 {
				t.Errorf("scrolled to %d, outside the stream", got.ScrollTop)
			}
		})
	}
}
