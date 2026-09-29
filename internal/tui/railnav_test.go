package tui

import (
	"errors"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/aguinelo/dcode/internal/protocol"
)

func navProgram(open string, titles ...string) *program {
	m := Model{Lang: En, Cursor: -1, SessionID: open}
	for i, ti := range titles {
		m.Sessions = append(m.Sessions, SessionChoice{ID: string(rune('a' + i)), Title: ti})
	}
	return &program{model: m, geo: DefaultGeometry(140, 24)}
}

// The mode has to be reachable, and it has to own the keyboard once it is.
// Copy mode's block was placed inside the completion-menu guard and never ran
// at all — the menu only opens once something is typed, and copy mode only
// opens on an empty line, so the two were never true together.
func TestTheRailTakesTheKeyboardAndGivesItBack(t *testing.T) {
	p := navProgram("b", "first", "second")

	p.onKey(ctrl('r'))
	if !p.model.Nav.Active {
		t.Fatal("^r did not focus the list")
	}

	// A letter filters here rather than typing into the line: this is a mode
	// that owns the keyboard, which is the case RN-16 leaves room for.
	p.onKey(key("s"))
	if p.model.Nav.Filter != "s" {
		t.Errorf("a letter did not reach the filter: %q", p.model.Nav.Filter)
	}
	if p.model.Input != "" {
		t.Errorf("a letter leaked into the input line: %q", p.model.Input)
	}

	// Esc backs out of one thing at a time.
	p.onKey(special(tea.KeyEscape))
	if p.model.Nav.Filter != "" || !p.model.Nav.Active {
		t.Errorf("esc left the mode instead of clearing the filter first: %+v", p.model.Nav)
	}
	p.onKey(special(tea.KeyEscape))
	if p.model.Nav.Active {
		t.Error("a second esc did not close the mode")
	}
}

// Nothing to choose from means nothing to focus: a mode that opens onto an
// empty list swallows the next keystroke for no reason.
func TestTheRailDoesNotOpenOntoAnEmptyList(t *testing.T) {
	p := &program{model: Model{Lang: En, Cursor: -1}, geo: DefaultGeometry(140, 24)}
	p.onKey(ctrl('r'))
	if p.model.Nav.Active {
		t.Error("the mode opened with no conversations to show")
	}
}

// It does not wrap. Wrapping turns "one too far" into "somewhere else
// entirely", and this is a list where landing somewhere else opens the wrong
// afternoon's work.
func TestTheRailCursorStopsAtBothEnds(t *testing.T) {
	n := RailNav{Active: true}
	if got := n.Move(-1, 3); got.Cursor != 0 {
		t.Errorf("moving up from the top landed at %d", got.Cursor)
	}
	n.Cursor = 2
	if got := n.Move(1, 3); got.Cursor != 2 {
		t.Errorf("moving down from the bottom landed at %d", got.Cursor)
	}
}

// A filter that matched nothing chooses nothing. Empty has to be
// distinguishable from the first row, or it would open the newest conversation
// instead of doing nothing.
func TestAFilterThatMatchesNothingChoosesNothing(t *testing.T) {
	all := []SessionChoice{{ID: "a", Title: "alpha"}, {ID: "b", Title: "bravo"}}
	n := RailNav{Active: true, Filter: "zzz"}
	if got := n.Chosen(all); got != "" {
		t.Errorf("an empty list chose %q", got)
	}
}

// Typing pulls the cursor back to the top: after a keystroke the person is
// looking at a different list, and holding a position in it would be holding a
// position in something they have not read.
func TestTypingReturnsTheCursorToTheTop(t *testing.T) {
	all := []SessionChoice{{ID: "a", Title: "alpha"}, {ID: "b", Title: "bravo"}}
	n := RailNav{Active: true, Cursor: 1}.Type("a", all)
	if n.Cursor != 0 {
		t.Errorf("the cursor stayed at %d after the list changed", n.Cursor)
	}
}

// Backspace drops a rune, not a byte.
func TestBackspaceDropsARuneNotAByte(t *testing.T) {
	n := RailNav{Active: true, Filter: "sessão"}.Backspace()
	if n.Filter != "sessã" {
		t.Errorf("got %q", n.Filter)
	}
}

// The cursor is a character. A row picked out in colour alone is not picked out
// at all on a terminal without any, and choosing the wrong row here opens the
// wrong afternoon's work.
func TestTheCursorIsACharacterAndNotOnlyAColour(t *testing.T) {
	m := Model{Lang: En, Cursor: -1, SessionID: "z",
		Sessions: []SessionChoice{{ID: "a", Title: "alpha"}, {ID: "b", Title: "bravo"}},
		Nav:      RailNav{Active: true, Cursor: 1}}
	g := railGeometry(140)
	g.Palette = Palette{}
	lines := strings.Join(renderSessionList(m, g), "\n")
	if !strings.Contains(lines, "▸ bravo") {
		t.Errorf("the cursor is not drawn as a character:\n%s", lines)
	}
}

// A filter that empties the list says so. A list that empties itself reads as a
// list that lost its contents.
func TestAnEmptyResultSaysSoRatherThanGoingBlank(t *testing.T) {
	m := Model{Lang: En, Cursor: -1,
		Sessions: []SessionChoice{{ID: "a", Title: "alpha"}},
		Nav:      RailNav{Active: true, Filter: "zzz"}}
	g := railGeometry(140)
	g.Palette = Palette{}
	if !strings.Contains(strings.Join(renderSessionList(m, g), "\n"), "nothing matches") {
		t.Error("an empty filter result went silently blank")
	}
}

// Choosing the conversation already open does nothing rather than reloading it.
func TestChoosingTheOpenConversationDoesNothing(t *testing.T) {
	p := navProgram("a", "alpha", "bravo")
	p.onKey(ctrl('r'))
	_, cmd := p.onKey(special(tea.KeyEnter))
	if cmd != nil {
		t.Error("the conversation already open was reloaded")
	}
	if p.model.Nav.Active {
		t.Error("the mode stayed open after choosing")
	}
}

// -- naming ------------------------------------------------------------------

// Naming is its own mode inside the list, because it is the one thing here that
// changes something. Every key means the name while it is open, so nothing else
// is reachable by accident halfway through.
func TestNamingTakesEveryKeyWhileItIsOpen(t *testing.T) {
	p := navProgram("b", "first", "second")
	p.onKey(ctrl('r'))
	p.onKey(key("r"))
	if !p.model.Nav.Naming {
		t.Fatal("r did not open naming")
	}

	// `r` would have re-opened naming and `j` would have moved a cursor in
	// another mode. Here they are letters in a name.
	for _, c := range []string{"r", "j", "x"} {
		p.onKey(key(c))
	}
	if p.model.Nav.Draft != "rjx" {
		t.Errorf("the draft is %q", p.model.Nav.Draft)
	}
	if p.model.Nav.Cursor != 0 {
		t.Errorf("a letter moved the cursor to %d", p.model.Nav.Cursor)
	}
}

// The draft is seeded with the NAME and not the derived title. Offering the
// title would turn "give this a name" into "confirm the one you were given",
// and the first Enter would quietly promote a derived title into a chosen one.
func TestNamingStartsFromTheNameAndNotTheDerivedTitle(t *testing.T) {
	n := RailNav{Active: true, Cursor: 0}
	all := []SessionChoice{{ID: "a", Title: "derived from the question"}}
	if got := n.StartNaming(all); got.Draft != "" {
		t.Errorf("the derived title was offered as a draft: %q", got.Draft)
	}

	named := []SessionChoice{{ID: "a", Title: "derived", Name: "chosen"}}
	if got := n.StartNaming(named); got.Draft != "chosen" {
		t.Errorf("the existing name was not offered for editing: %q", got.Draft)
	}
}

// Backing out leaves what was there rather than clearing it, which is the
// design's own rule for esc in this mode.
func TestEscapingNamingKeepsWhatWasThere(t *testing.T) {
	p := navProgram("b", "first", "second")
	p.onKey(ctrl('r'))
	p.onKey(key("r"))
	p.onKey(key("z"))
	p.onKey(special(tea.KeyEscape))

	if p.model.Nav.Naming || p.model.Nav.Draft != "" {
		t.Errorf("naming survived esc: %+v", p.model.Nav)
	}
	if !p.model.Nav.Active {
		t.Error("esc left the list entirely instead of just the naming")
	}
	if p.model.Sessions[0].Name != "" {
		t.Errorf("a cancelled name was kept: %q", p.model.Sessions[0].Name)
	}
}

// Enter sends what was typed, to the conversation under the cursor.
func TestEnterSendsTheNameForTheRowUnderTheCursor(t *testing.T) {
	p := navProgram("z", "first", "second")
	p.onKey(ctrl('r'))
	p.onKey(special(tea.KeyDown))
	p.onKey(key("r"))
	for _, c := range []string{"n", "e", "w"} {
		p.onKey(key(c))
	}
	_, cmd := p.onKey(special(tea.KeyEnter))
	if cmd == nil {
		t.Fatal("enter sent nothing")
	}
	if p.model.Nav.Naming {
		t.Error("the mode stayed open after committing")
	}
}

// The keyboard stops where the daemon would refuse. Letting somebody type past
// the limit and then reporting a failure wastes the typing.
func TestTheDraftStopsAtTheLimitTheDaemonEnforces(t *testing.T) {
	n := RailNav{Active: true, Naming: true}
	for i := 0; i < NameLimit+20; i++ {
		n = n.TypeName("a")
	}
	if got := len([]rune(n.Draft)); got != NameLimit {
		t.Errorf("the draft grew to %d, limit is %d", got, NameLimit)
	}
}

func TestBackspaceInANameDropsARuneNotAByte(t *testing.T) {
	n := RailNav{Active: true, Naming: true, Draft: "sessão"}.BackspaceName()
	if n.Draft != "sessã" {
		t.Errorf("got %q", n.Draft)
	}
}

// A name a person gave says which it is. Without the mark, a listing shows two
// kinds of claim in one column and nothing tells them apart.
func TestAGivenNameIsMarkedAsGiven(t *testing.T) {
	m := Model{Lang: En, Cursor: -1,
		Sessions: []SessionChoice{
			{ID: "a", Title: "derived one"},
			{ID: "b", Title: "derived two", Name: "chosen"},
		}}
	g := railGeometry(160)
	g.Palette = Palette{}
	lines := strings.Join(renderRail(m, g, 12), "\n")

	for _, l := range strings.Split(lines, "\n") {
		if strings.Contains(l, "chosen") && !strings.Contains(l, "·") {
			t.Errorf("a given name is not marked: %q", l)
		}
		if strings.Contains(l, "derived one") && strings.Contains(l, "·") {
			t.Errorf("a derived title was marked as given: %q", l)
		}
	}
}

// -- opening -----------------------------------------------------------------

// listProgram is a program whose list holds these recorded conversations, the
// way the edge hands them over at start.
func listProgram(t *testing.T, choices ...SessionChoice) (*program, *fakeTransport) {
	t.Helper()
	p, tr := newProgram(t, func(o *Options) { o.Sessions = choices })
	p.model.Sessions = p.opts.Sessions
	return p, tr
}

// A conversation from an earlier run opens, as a continuation.
//
// The list is what this workspace RECORDED, read from disk when the interface
// started, and enter asked GetSession for the choice — which answers only for
// the sessions the daemon holds right now. A conversation from an earlier run
// is on disk and not in the daemon, so choosing one, which is what the list is
// for, came back as "could not resume …: no session …". Nothing caught it: the
// fake transport answered GetSession for any id, so a test here could not
// have failed.
//
// Opening it is what `dcode -r` does: a new session, with an id of its own,
// carrying the conversation. The marker at the top of that session's log is
// what says so on the screen.
func TestChoosingARecordedConversationContinuesIt(t *testing.T) {
	p, tr := listProgram(t,
		SessionChoice{ID: "1a02c3d", Title: "rename the flags", Turns: 1},
		SessionChoice{ID: "1a015fb", Title: "fix the parser", Turns: 3})

	p.onKey(ctrl('r'))
	p.onKey(special(tea.KeyDown))
	_, cmd := p.onKey(special(tea.KeyEnter))
	msg := run(t, p, cmd)
	sw, ok := msg.(switchedMsg)
	if !ok {
		t.Fatalf("choosing a recorded conversation did not open it: %#v", msg)
	}
	if len(tr.created) != 1 {
		t.Fatalf("want one session opened to carry it, got %+v", tr.created)
	}
	if req := tr.created[0]; req.Resume != "1a015fb" || req.Workspace != "/w" {
		t.Errorf("the session opened does not continue the one chosen, here: %+v", req)
	}

	p.Update(sw)
	if p.opts.SessionID != sw.session.ID || p.opts.SessionID == "1a015fb" {
		t.Errorf("attached to %q; a continuation is a session of its own", p.opts.SessionID)
	}

	// What the daemon opens a continuation with. Without the line it draws, the
	// replayed conversation reads as work this session did.
	for _, e := range []protocol.Event{
		ev(t, 1, protocol.EventSessionCreated, protocol.Session{ID: sw.session.ID, Workspace: "/w"}),
		ev(t, 2, protocol.EventSessionResumed, protocol.SessionResumed{SourceID: "1a015fb", Turns: 3}),
		ev(t, 3, protocol.EventTurnStarted, protocol.TurnStarted{TurnID: "t1", Text: "fix the parser"}),
	} {
		p.Update(eventMsg{ev: e, gen: p.generation})
	}
	if screen := p.View().Content; !strings.Contains(screen, "continuing 1a015fb") {
		t.Errorf("the screen does not say it is a continuation:\n%s", screen)
	}
}

// A conversation still open somewhere is joined, not continued.
//
// A daemon can be shared by two terminals, and then a recorded conversation may
// also be live. Continuing it would put the same conversation in two sessions
// that go on separately, and the one on this screen would not be the one being
// worked in over there.
func TestChoosingALiveConversationAttachesToIt(t *testing.T) {
	p, tr := listProgram(t, SessionChoice{ID: "1a02c3d", Title: "rename the flags", Turns: 1})
	tr.sessions = []protocol.Session{{ID: "1a02c3d", Workspace: "/w", LastSeq: 40, FirstSeq: 1}}

	p.onKey(ctrl('r'))
	_, cmd := p.onKey(special(tea.KeyEnter))
	msg := run(t, p, cmd)
	if sw, ok := msg.(switchedMsg); !ok || sw.session.ID != "1a02c3d" {
		t.Fatalf("the live conversation was not attached: %#v", msg)
	}
	if len(tr.created) != 0 {
		t.Errorf("a live conversation was continued into a second session: %+v", tr.created)
	}
}

// Only the daemon saying it holds no such session makes a choice a
// continuation. Any other failure is said, and nothing is opened on a guess:
// continuing a session that was live after all would split it in two.
func TestAChoiceTheDaemonCouldNotAnswerOpensNothing(t *testing.T) {
	p, tr := listProgram(t, SessionChoice{ID: "1a015fb", Title: "fix the parser", Turns: 3})
	tr.getErr = errors.New("connection refused")

	p.onKey(ctrl('r'))
	_, cmd := p.onKey(special(tea.KeyEnter))
	msg := run(t, p, cmd)
	if note, ok := msg.(noteMsg); !ok || !strings.Contains(string(note), "connection refused") {
		t.Fatalf("the failure was not said: %#v", msg)
	}
	if len(tr.created) != 0 {
		t.Errorf("a session was opened on a guess: %+v", tr.created)
	}
}
