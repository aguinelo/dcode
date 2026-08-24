package tui

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/aguinelo/dcode/internal/config"
	"github.com/aguinelo/dcode/internal/protocol"
)

// fakeTransport records what the client asked the daemon to do.
type fakeTransport struct {
	mu         sync.Mutex
	submitted  []string
	steered    []string
	interrupts int
	resolved   []protocol.ApprovalDecision
	created    []protocol.CreateSessionRequest
	renamed    [][2]string
	sessions   []protocol.Session

	submitErr error
	steerErr  error
	listErr   error
	createErr error
	renameErr error
	getErr    error

	submittedImages []int
	undos           int
	undoResult      protocol.UndoResult
	undoErr         error

	events chan protocol.Event
	errs   chan error
}

func newFakeTransport() *fakeTransport {
	return &fakeTransport{
		events: make(chan protocol.Event, 16),
		errs:   make(chan error, 4),
	}
}

func (f *fakeTransport) CreateSession(_ context.Context, req protocol.CreateSessionRequest) (protocol.Session, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.created = append(f.created, req)
	if f.createErr != nil {
		return protocol.Session{}, f.createErr
	}
	return protocol.Session{
		ID: "new", Workspace: req.Workspace, Model: req.Model,
		SandboxMode: req.SandboxMode, State: protocol.SessionStateIdle,
	}, nil
}

func (f *fakeTransport) ListSessions(context.Context) ([]protocol.Session, error) {
	return f.sessions, f.listErr
}

func (f *fakeTransport) GetSession(_ context.Context, id string) (protocol.Session, error) {
	if f.getErr != nil {
		return protocol.Session{}, f.getErr
	}
	return protocol.Session{ID: id, Workspace: "/w", Model: "m", SandboxMode: "read-only"}, nil
}

func (f *fakeTransport) Submit(_ context.Context, _, text string, imgs ...protocol.TurnImage) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.submitted = append(f.submitted, text)
	f.submittedImages = append(f.submittedImages, len(imgs))
	return f.submitErr
}

func (f *fakeTransport) Steer(_ context.Context, _, text string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.steered = append(f.steered, text)
	return f.steerErr
}

func (f *fakeTransport) steers() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string(nil), f.steered...)
}

// renamed records what a name was set to, so a test can assert the client
// sent what the person typed rather than only that it sent something.
func (f *fakeTransport) RenameSession(_ context.Context, id, name string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.renameErr != nil {
		return f.renameErr
	}
	f.renamed = append(f.renamed, [2]string{id, name})
	return nil
}

func (f *fakeTransport) Undo(context.Context, string) (protocol.UndoResult, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.undos++
	return f.undoResult, f.undoErr
}

func (f *fakeTransport) Interrupt(context.Context, string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.interrupts++
	return nil
}

func (f *fakeTransport) Resolve(_ context.Context, _, _ string, d protocol.ApprovalDecision) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.resolved = append(f.resolved, d)
	return nil
}

func (f *fakeTransport) Subscribe(context.Context, string, uint64) (<-chan protocol.Event, <-chan error) {
	return f.events, f.errs
}

func (f *fakeTransport) submits() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string(nil), f.submitted...)
}

func newProgram(t *testing.T, opts ...func(*Options)) (*program, *fakeTransport) {
	t.Helper()
	tr := newFakeTransport()
	o := Options{
		SessionID: "s1", Workspace: "/w", Model: "m", Sandbox: "read-only",
		Transport: tr, Geometry: DefaultGeometry(100, 24), QueueMax: 2,
		Commands: config.CommandSet{Commands: map[string]config.Command{}},
	}
	for _, mut := range opts {
		mut(&o)
	}
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	p := &program{
		opts: o, model: NewModel(o.SessionID, o.Workspace, o.Model, o.Sandbox, En),
		geo: o.Geometry, ctx: ctx, cancel: cancel,
	}
	p.attach(o.SessionID)
	return p, tr
}

// key builds a printable keypress.
//
// Code is the FIRST RUNE, not the first byte. `rune(s[0])` on "á" takes half of
// a two-byte character and produces a rune that is not the one typed — which is
// how a test helper can quietly stop representing the thing it is named after.
func key(s string) tea.KeyPressMsg {
	return tea.KeyPressMsg{Code: []rune(s)[0], Text: s}
}

func special(code rune) tea.KeyPressMsg { return tea.KeyPressMsg{Code: code} }

func ctrl(code rune) tea.KeyPressMsg {
	return tea.KeyPressMsg{Code: code, Mod: tea.ModCtrl}
}

// run drains a command, applying whatever message it produces.
func run(t *testing.T, p *program, cmd tea.Cmd) tea.Msg {
	t.Helper()
	if cmd == nil {
		return nil
	}
	return cmd()
}

func typeLine(t *testing.T, p *program, text string) tea.Cmd {
	t.Helper()
	for _, r := range text {
		if r == ' ' {
			p.Update(special(' '))
			continue
		}
		p.Update(tea.KeyPressMsg{Code: r, Text: string(r)})
	}
	_, cmd := p.Update(special(tea.KeyEnter))
	return cmd
}

func TestEnterSubmitsWhenIdleAndEchoesTheLine(t *testing.T) {
	p, tr := newProgram(t)
	run(t, p, typeLine(t, p, "hello"))

	if got := tr.submits(); len(got) != 1 || got[0] != "hello" {
		t.Fatalf("got %v", got)
	}
	// The user's own line must be visible before the model answers, or a long
	// turn looks like nothing happened. It arrives on turn.started rather than
	// being echoed when it was typed: the client that echoed was the only one
	// that had it, so a second client — or this one after a replay — saw
	// answers to questions it never saw.
	p.model = p.model.Apply(ev(t, 1, protocol.EventTurnStarted,
		protocol.TurnStarted{TurnID: "t1", Text: "hello"}))
	if len(p.model.Entries) != 1 || p.model.Entries[0].Kind != KindUser {
		t.Errorf("got %+v", p.model.Entries)
	}
	if p.model.Entries[0].Summary != "hello" {
		t.Errorf("the line shown is %q", p.model.Entries[0].Summary)
	}
	if p.model.Input != "" {
		t.Errorf("the input must clear, got %q", p.model.Input)
	}
}

func TestBlankInputSubmitsNothing(t *testing.T) {
	p, tr := newProgram(t)
	p.Update(special(tea.KeyEnter))
	if len(tr.submits()) != 0 {
		t.Error("an empty line is not a turn")
	}
}

// Words typed while a turn runs steer it: they reach the running turn at its
// next round rather than waiting for it to end.
//
// This replaces queuing for ordinary input, and the invariant it replaces was
// true for a good reason that no longer holds — the protocol refuses a second
// TURN, and a correction is not one. Watching a turn go wrong with two options,
// let it finish or kill it and lose what it learned, is what queuing cost.
func TestWordsTypedMidTurnSteerIt(t *testing.T) {
	p, tr := newProgram(t)
	p.model.State = protocol.SessionStateRunning

	run(t, p, typeLine(t, p, "use tabs"))
	run(t, p, typeLine(t, p, "and stop at three"))

	if got := tr.submits(); len(got) != 0 {
		t.Errorf("a correction opened a second turn: %v", got)
	}
	got := tr.steers()
	if len(got) != 2 || got[0] != "use tabs" || got[1] != "and stop at three" {
		t.Errorf("steered %v, want both in typing order", got)
	}
	if len(p.model.Queue) != 0 {
		t.Errorf("a correction was queued as well: %v", p.model.Queue)
	}
	// Not echoed locally: turn.steered carries it, exactly as turn.started does
	// for the question that opens a turn. Echoing too would show it twice.
	for _, e := range p.model.Entries {
		if e.Kind == KindUser && strings.Contains(e.Summary, "use tabs") {
			t.Error("the correction was echoed locally as well as by the event")
		}
	}
}

// The event is what puts it on screen, so a second client watching the same
// session sees the turn change direction and why.
func TestASteeredTurnShowsWhatWasSaid(t *testing.T) {
	p, _ := newProgram(t)
	p.Update(eventMsg(ev(t, 1, protocol.EventTurnSteered,
		protocol.TurnSteered{TurnID: "t1", Text: "use tabs"})))

	var found bool
	for _, e := range p.model.Entries {
		if e.Kind == KindUser && strings.Contains(e.Summary, "use tabs") {
			found = true
		}
	}
	if !found {
		t.Error("a correction reached the session and nothing showed it")
	}
}

// An attached picture holds the message back. An image is about a question, and
// the steering path carries text only — sending half the pair now would ask the
// model about something it cannot see.
func TestAnAttachedImageHoldsTheMessageBack(t *testing.T) {
	p, tr := newProgram(t)
	p.model.State = protocol.SessionStateRunning
	p.model.Attached = []protocol.TurnImage{{MediaType: "image/png", Data: "eA=="}}

	run(t, p, typeLine(t, p, "what is wrong here"))

	if len(tr.steers()) != 0 {
		t.Error("a message with a picture was steered without its picture")
	}
	if len(p.model.Queue) != 1 {
		t.Errorf("queue = %v, want the message held for the next turn", p.model.Queue)
	}
}

// The turn can end between the keystroke and the request. Losing the message
// there would lose the most deliberate thing a person does during a turn, so a
// refusal becomes what would have happened had they typed it a moment later.
func TestACorrectionThatArrivedTooLateBecomesAMessage(t *testing.T) {
	p, tr := newProgram(t)
	p.model.State = protocol.SessionStateIdle

	run(t, p, func() tea.Cmd {
		_, cmd := p.Update(steerLateMsg{"use tabs"})
		return cmd
	}())

	got := tr.submits()
	if len(got) != 1 || got[0] != "use tabs" {
		t.Errorf("submitted %v, want the late correction sent as a message", got)
	}
}

// drainBatch runs a command and every command in the batch it produced.
//
// The commands run concurrently because one of them is the blocking wait for
// the next event, which by design never returns on its own.
func drainBatch(t *testing.T, p *program, cmd tea.Cmd) {
	t.Helper()
	if cmd == nil {
		return
	}
	msg := cmd()
	batch, ok := msg.(tea.BatchMsg)
	if !ok {
		if msg != nil {
			p.Update(msg)
		}
		return
	}
	var wg sync.WaitGroup
	for _, c := range batch {
		if c == nil {
			continue
		}
		wg.Add(1)
		go func(c tea.Cmd) { defer wg.Done(); c() }(c)
	}
	done := make(chan struct{})
	go func() { wg.Wait(); close(done) }()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		// The event wait is still pending, which is expected; everything else
		// has had its chance to run.
	}
}

// Interrupting is what is wanted in the overwhelming majority of cases, and
// quitting mid-turn loses work.
func TestCtrlCInterruptsMidTurnAndQuitsWhenIdle(t *testing.T) {
	p, tr := newProgram(t)
	p.model.State = protocol.SessionStateRunning
	_, cmd := p.Update(ctrl('c'))
	run(t, p, cmd)
	if tr.interrupts != 1 {
		t.Fatalf("got %d interrupts", tr.interrupts)
	}

	p.model.State = protocol.SessionStateIdle
	_, cmd = p.Update(ctrl('c'))
	if _, ok := run(t, p, cmd).(tea.QuitMsg); !ok {
		t.Error("ctrl+c when idle quits")
	}
	_, cmd = p.Update(ctrl('d'))
	if _, ok := run(t, p, cmd).(tea.QuitMsg); !ok {
		t.Error("ctrl+d quits")
	}
}

// The modal is the one moment the user has to read, and letting keystrokes fall
// through would let them dismiss it by accident.
func TestApprovalKeysAndTheirDefault(t *testing.T) {
	for name, tc := range map[string]struct {
		msg  tea.KeyPressMsg
		want protocol.ApprovalDecision
	}{
		"allow once":    {key("a"), protocol.ApprovalAllow},
		"allow session": {tea.KeyPressMsg{Code: 'a', ShiftedCode: 'A', Text: "A"}, protocol.ApprovalAllowSession},
		"deny":          {key("d"), protocol.ApprovalDeny},
		"enter denies":  {special(tea.KeyEnter), protocol.ApprovalDeny},
		"esc denies":    {special(tea.KeyEscape), protocol.ApprovalDeny},
		"ctrl+c denies": {ctrl('c'), protocol.ApprovalDeny},
	} {
		p, tr := newProgram(t)
		p.model.Pending = &protocol.ApprovalRequest{ApprovalID: "a1", Tool: "bash"}
		_, cmd := p.Update(tc.msg)
		run(t, p, cmd)
		if len(tr.resolved) != 1 || tr.resolved[0] != tc.want {
			t.Errorf("%s: got %v", name, tr.resolved)
		}
	}
}

func TestKeystrokesDoNotFallThroughTheModal(t *testing.T) {
	p, tr := newProgram(t)
	p.model.Pending = &protocol.ApprovalRequest{ApprovalID: "a1", Tool: "bash"}
	p.Update(key("x"))
	if p.model.Input != "" {
		t.Errorf("the modal must swallow ordinary input, got %q", p.model.Input)
	}
	if len(tr.resolved) != 0 {
		t.Errorf("an unrelated key must not answer, got %v", tr.resolved)
	}
}

func TestEditingKeys(t *testing.T) {
	p, _ := newProgram(t)
	p.Update(key("a"))
	p.Update(key("b"))
	p.Update(special(tea.KeyBackspace))
	if p.model.Input != "a" {
		t.Errorf("got %q", p.model.Input)
	}
	p.Update(special(tea.KeyBackspace))
	p.Update(special(tea.KeyBackspace))
	if p.model.Input != "" {
		t.Errorf("backspace on an empty line is a no-op, got %q", p.model.Input)
	}
	p.Update(special(' '))
	if p.model.Input != " " {
		t.Errorf("got %q", p.model.Input)
	}
}

func TestCursorMovementAndExpansion(t *testing.T) {
	p, _ := newProgram(t)
	p.model.Entries = []Entry{
		{Kind: KindTool, Tool: "read", Detail: "x"},
		{Kind: KindTool, Tool: "grep", Detail: "y"},
	}
	// `esc` from an empty line steps into the transcript, at the newest entry:
	// the newest is what the user is looking at. It used to be `up`, which
	// walked in silently — see TestSteppingIntoTheTranscriptIsDeliberate.
	p.Update(special(tea.KeyEsc))
	p.Update(key("k"))
	if p.model.Cursor != 0 {
		t.Fatalf("got %d", p.model.Cursor)
	}
	p.Update(key("k"))
	if p.model.Cursor != 0 {
		t.Errorf("the cursor must not run off the top, got %d", p.model.Cursor)
	}
	p.Update(special(tea.KeyDown))
	p.Update(special(tea.KeyDown))
	if p.model.Cursor != 1 {
		t.Errorf("the cursor must not run off the bottom, got %d", p.model.Cursor)
	}
	p.Update(special(tea.KeyTab))
	if !p.model.Entries[1].Expanded {
		t.Error("tab expands the selected entry")
	}
}

func TestWindowResizeIsHonoured(t *testing.T) {
	p, _ := newProgram(t)
	p.Update(tea.WindowSizeMsg{Width: 42, Height: 13})
	if p.geo.Width != 42 || p.geo.Height != 13 {
		t.Errorf("got %dx%d", p.geo.Width, p.geo.Height)
	}
}

// ---------- built-ins ----------

func TestHelpAndPlanAnswerLocally(t *testing.T) {
	p, tr := newProgram(t)
	run(t, p, typeLine(t, p, "/help"))
	if len(tr.submits()) != 0 {
		t.Error("/help must not cost a turn")
	}
	last := p.model.Entries[len(p.model.Entries)-1]
	if last.Kind != KindNote || !strings.Contains(last.Summary, "/config") {
		t.Errorf("got %+v", last)
	}

	p.model.Plan = modelWithPlan().Plan
	run(t, p, typeLine(t, p, "/plan"))
	if len(tr.submits()) != 0 {
		t.Error("/plan without an argument is a local question")
	}
	if !strings.Contains(p.model.Entries[len(p.model.Entries)-1].Summary, "read the parser") {
		t.Errorf("got %+v", p.model.Entries[len(p.model.Entries)-1])
	}
}

func TestPlanWithAnArgumentAndInitBecomeTurns(t *testing.T) {
	p, tr := newProgram(t)
	run(t, p, typeLine(t, p, "/plan drop the docs"))
	run(t, p, typeLine(t, p, "/init"))

	got := tr.submits()
	if len(got) != 2 {
		t.Fatalf("got %v", got)
	}
	if !strings.Contains(got[0], "drop the docs") {
		t.Errorf("got %q", got[0])
	}
	if !strings.Contains(got[1], "DCODE.md") {
		t.Errorf("got %q", got[1])
	}
}

func TestBuiltinTurnsAreQueuedWhileRunning(t *testing.T) {
	p, tr := newProgram(t)
	p.model.State = protocol.SessionStateRunning
	run(t, p, typeLine(t, p, "/init"))
	if len(tr.submits()) != 0 {
		t.Error("a built-in that costs a turn must respect the one-turn rule")
	}
	if len(p.model.Queue) != 1 {
		t.Errorf("got %v", p.model.Queue)
	}
}

func TestConfigLookup(t *testing.T) {
	p, _ := newProgram(t, func(o *Options) {
		o.Lookup = func(key string) (string, bool) {
			if key == "model.name" {
				return "model.name = m\n  from: env", true
			}
			return "", false
		}
	})
	run(t, p, typeLine(t, p, "/config model.name"))
	if !strings.Contains(p.model.Entries[0].Summary, "from: env") {
		t.Errorf("the provenance is the point of the answer: %+v", p.model.Entries[0])
	}

	run(t, p, typeLine(t, p, "/config nope"))
	if !strings.Contains(p.model.Entries[1].Summary, "not set") {
		t.Errorf("got %+v", p.model.Entries[1])
	}

	run(t, p, typeLine(t, p, "/config"))
	if !strings.Contains(p.model.Entries[2].Summary, "Usage") {
		t.Errorf("got %+v", p.model.Entries[2])
	}
}

func TestConfigWithoutALookup(t *testing.T) {
	p, _ := newProgram(t)
	run(t, p, typeLine(t, p, "/config model.name"))
	if !strings.Contains(p.model.Entries[0].Summary, "not available") {
		t.Errorf("got %+v", p.model.Entries[0])
	}
}

// The system prompt is part of the prefix and the prefix cannot be rewritten,
// so changing the model means a new session — not a cleared screen.
func TestClearAndModelOpenANewSession(t *testing.T) {
	p, tr := newProgram(t)
	p.model.Entries = []Entry{{Kind: KindNote, Summary: "old history"}}

	msg := run(t, p, func() tea.Cmd { _, c := p.runBuiltin(Resolved{Kind: CmdBuiltin, Name: "clear"}); return c }())
	sw, ok := msg.(switchedMsg)
	if !ok {
		t.Fatalf("got %T", msg)
	}
	p.Update(sw)
	if len(p.model.Entries) != 0 {
		t.Error("a new session starts from an empty view; the old log is not its own")
	}
	if p.opts.SessionID != "new" {
		t.Errorf("got %q", p.opts.SessionID)
	}

	run(t, p, func() tea.Cmd {
		_, c := p.runBuiltin(Resolved{Kind: CmdBuiltin, Name: "model", Args: "claude"})
		return c
	}())
	if len(tr.created) != 2 || tr.created[1].Model != "claude" {
		t.Errorf("got %+v", tr.created)
	}
}

func TestModelWithoutAnArgumentReportsTheCurrentOne(t *testing.T) {
	p, tr := newProgram(t)
	run(t, p, typeLine(t, p, "/model"))
	if len(tr.created) != 0 {
		t.Error("no name, no switch")
	}
	if !strings.Contains(p.model.Entries[0].Summary, "Currently m") {
		t.Errorf("got %+v", p.model.Entries[0])
	}
}

func TestResumeListsAndReattaches(t *testing.T) {
	p, tr := newProgram(t)
	tr.sessions = []protocol.Session{{ID: "s9", State: protocol.SessionStateIdle, Workspace: "/x"}}

	msg := run(t, p, func() tea.Cmd { _, c := p.runBuiltin(Resolved{Name: "resume"}); return c }())
	if note, ok := msg.(noteMsg); !ok || !strings.Contains(string(note), "s9") {
		t.Fatalf("got %v", msg)
	}

	msg = run(t, p, func() tea.Cmd { _, c := p.runBuiltin(Resolved{Name: "resume", Args: "s9"}); return c }())
	if sw, ok := msg.(switchedMsg); !ok || sw.session.ID != "s9" {
		t.Fatalf("got %v", msg)
	}
}

func TestBuiltinFailuresBecomeNotesRatherThanCrashes(t *testing.T) {
	p, tr := newProgram(t)
	tr.createErr = errors.New("no room")
	tr.listErr = errors.New("offline")
	tr.getErr = errors.New("gone")

	for _, r := range []Resolved{
		{Name: "clear"}, {Name: "resume"}, {Name: "resume", Args: "s1"},
	} {
		_, cmd := p.runBuiltin(r)
		if _, ok := run(t, p, cmd).(noteMsg); !ok {
			t.Errorf("%s: a failure must become a note the user can read", r.Name)
		}
	}
}

func TestAnUnknownCommandIsRefusedAndPointsAtHelp(t *testing.T) {
	p, tr := newProgram(t)
	run(t, p, typeLine(t, p, "/nope"))
	if len(tr.submits()) != 0 {
		t.Error("an unknown command must not reach the model")
	}
	if !strings.Contains(p.model.Entries[0].Summary, "/help") {
		t.Errorf("got %+v", p.model.Entries[0])
	}
}

func TestAUserCommandExpandsIntoATurn(t *testing.T) {
	p, tr := newProgram(t, func(o *Options) {
		o.Commands = userCommands(config.Command{Name: "rev", Body: "Review $ARGUMENTS."})
	})
	run(t, p, typeLine(t, p, "/rev main.go"))
	if got := tr.submits(); len(got) != 1 || got[0] != "Review main.go." {
		t.Fatalf("got %v", got)
	}
}

// ---------- the stream ----------

func TestEventsFoldIntoTheModelAndAStreamErrorQuits(t *testing.T) {
	p, _ := newProgram(t)
	p.Update(eventMsg(ev(t, 1, protocol.EventMessageDelta, protocol.MessageDelta{Text: "hi"})))
	if len(p.model.Entries) != 1 {
		t.Fatalf("got %+v", p.model.Entries)
	}

	_, cmd := p.Update(errMsg{errors.New("connection lost")})
	if _, ok := run(t, p, cmd).(tea.QuitMsg); !ok {
		t.Error("a fatal stream error ends the program")
	}
	if !strings.Contains(p.View().Content, "connection lost") {
		t.Errorf("the reason must be visible:\n%s", p.View().Content)
	}

	_, cmd = p.Update(streamClosedMsg{})
	if _, ok := run(t, p, cmd).(tea.QuitMsg); !ok {
		t.Error("a closed stream ends the program")
	}
}

func TestSubmitFailureSurfacesAsAFatalMessage(t *testing.T) {
	p, tr := newProgram(t)
	tr.submitErr = errors.New("daemon went away")
	msg := run(t, p, typeLine(t, p, "hello"))
	if _, ok := msg.(errMsg); !ok {
		t.Fatalf("got %T", msg)
	}
}

func TestWaitForEventDeliversAndNoticesClosure(t *testing.T) {
	p, tr := newProgram(t)
	tr.events <- ev(t, 1, protocol.EventTurnStarted, protocol.TurnStarted{TurnID: "t1"})
	if _, ok := p.waitForEvent()().(eventMsg); !ok {
		t.Error("an event must arrive as an eventMsg")
	}

	tr.errs <- errors.New("boom")
	if _, ok := p.waitForEvent()().(errMsg); !ok {
		t.Error("an error must arrive as an errMsg")
	}

	close(tr.events)
	if _, ok := p.waitForEvent()().(streamClosedMsg); !ok {
		t.Error("a closed channel ends the stream")
	}
}

func TestCancellingTheContextEndsTheWait(t *testing.T) {
	p, _ := newProgram(t)
	p.cancel()
	if _, ok := p.waitForEvent()().(streamClosedMsg); !ok {
		t.Error("a cancelled context ends the stream")
	}
}

func TestTheVersionNoticeArrivesAsANote(t *testing.T) {
	p, _ := newProgram(t, func(o *Options) {
		o.Notice = func(context.Context) string { return "dcode 0.2.0 is available" }
	})
	msg := run(t, p, p.checkVersion())
	if note, ok := msg.(noteMsg); !ok || !strings.Contains(string(note), "0.2.0") {
		t.Fatalf("got %v", msg)
	}
	p.Update(msg)
	if p.model.Entries[0].Kind != KindNote {
		t.Errorf("got %+v", p.model.Entries[0])
	}

	// Nothing to say means nothing on screen.
	quiet, _ := newProgram(t, func(o *Options) {
		o.Notice = func(context.Context) string { return "" }
	})
	if msg := run(t, quiet, quiet.checkVersion()); msg != nil {
		t.Errorf("got %v", msg)
	}
	if cmd := quiet.Init(); cmd == nil {
		t.Error("Init must at least start the event wait")
	}
}

func TestViewTakesTheAlternateScreen(t *testing.T) {
	p, _ := newProgram(t)
	if !p.View().AltScreen {
		t.Error("a fixed panel is only possible on the alternate screen")
	}
}

func TestUnknownMessagesAreIgnored(t *testing.T) {
	p, _ := newProgram(t)
	if _, cmd := p.Update(struct{ x int }{1}); cmd != nil {
		t.Errorf("got %v", cmd)
	}
}

// ---------- scrolling and navigation through the keyboard ----------

func withStream(t *testing.T, n int) (*program, *fakeTransport) {
	t.Helper()
	p, tr := newProgram(t, func(o *Options) { o.Geometry = DefaultGeometry(80, 12) })
	p.model = longStream(n)
	p.model.Follow = true
	return p, tr
}

// A key that sometimes scrolls and sometimes types is a key nobody trusts, so
// these never depend on what is in the input line.
func TestScrollKeysWorkWhateverIsTyped(t *testing.T) {
	for _, input := range []string{"", "meio escrito"} {
		p, _ := withStream(t, 100)
		p.model = p.model.SetInput(input)

		p.Update(special(tea.KeyPgUp))
		if p.model.Follow {
			t.Errorf("input %q: PgUp must scroll", input)
		}
		before := p.model.ScrollTop

		p.Update(special(tea.KeyPgDown))
		if p.model.ScrollTop <= before && !p.model.Follow {
			t.Errorf("input %q: PgDown must come back", input)
		}

		p.Update(special(tea.KeyHome))
		if p.model.ScrollTop != 0 {
			t.Errorf("input %q: Home goes to the beginning, got %d", input, p.model.ScrollTop)
		}
		p.Update(special(tea.KeyEnd))
		if !p.model.Follow {
			t.Errorf("input %q: End resumes following", input)
		}
		// And the line was never touched.
		if p.model.Input != input {
			t.Errorf("scrolling must not edit the line: %q", p.model.Input)
		}
	}
}

// Empty input means the user is reaching for what they typed before; input with
// text means they are working on the stream. Getting it wrong the other way
// would destroy what they were writing.
func TestArrowsAreContextSensitive(t *testing.T) {
	p, tr := newProgram(t)
	run(t, p, typeLine(t, p, "primeiro comando"))
	if len(tr.submits()) != 1 {
		t.Fatal("setup failed")
	}

	// Empty line: up recalls.
	p.Update(special(tea.KeyUp))
	if p.model.Input != "primeiro comando" {
		t.Fatalf("up on an empty line must recall, got %q", p.model.Input)
	}
	p.Update(special(tea.KeyDown))
	if p.model.Input != "" {
		t.Errorf("down must return to the empty line, got %q", p.model.Input)
	}

	// With text on the line, up neither browses history nor walks into the
	// transcript: it moves within the line, and at the border it scrolls.
	p.model.Navigating, p.model.Cursor = false, -1
	p.model = p.model.SetInput("escrevendo")
	p.model.Entries = append(p.model.Entries, Entry{Kind: KindNote, Summary: "a"})
	p.Update(special(tea.KeyUp))
	if p.model.Input != "escrevendo" {
		t.Errorf("history must not eat what is being written: %q", p.model.Input)
	}
	if p.model.Cursor >= 0 {
		t.Errorf("up walked into the transcript from a line being typed on: %d", p.model.Cursor)
	}
}

func TestEnteringTheStreamStartsAtTheNewestEntry(t *testing.T) {
	p, _ := withStream(t, 20)
	p.model.Cursor = -1
	p.model.History = nil

	p.Update(special(tea.KeyEsc))
	if !p.model.Navigating {
		t.Fatal("esc from an empty line did not step into the transcript")
	}
	if p.model.Cursor != len(p.model.Entries)-1 {
		t.Errorf("got %d, want the last entry", p.model.Cursor)
	}
	// And the selection is on screen.
	if !strings.Contains(Render(p.model, p.geo), "linha 19") {
		t.Error("the selected entry must be visible")
	}
}

// Escape backs out of what was opened, innermost first.
func TestEscapeClosesTheExpansionThenTheSelection(t *testing.T) {
	p, _ := newProgram(t)
	p.model.Entries = []Entry{{Kind: KindTool, Tool: "read", Detail: "x", Expanded: true}}
	p.model.Cursor = 0
	p.model.Navigating = true

	p.Update(special(tea.KeyEscape))
	if p.model.Entries[0].Expanded {
		t.Fatal("escape must close the expansion first")
	}
	if p.model.Cursor != 0 {
		t.Fatal("and keep the selection")
	}
	p.Update(special(tea.KeyEscape))
	if p.model.Cursor != -1 || p.model.Navigating {
		t.Error("a second escape leaves the mode")
	}
}

func TestLineEditingKeys(t *testing.T) {
	p, _ := newProgram(t)
	p.model = p.model.SetInput("um dois tres")

	p.Update(ctrl('a'))
	if p.model.InputCursor != 0 {
		t.Errorf("ctrl+a goes to the start, got %d", p.model.InputCursor)
	}
	p.Update(ctrl('e'))
	if p.model.InputCursor != 12 {
		t.Errorf("ctrl+e goes to the end, got %d", p.model.InputCursor)
	}
	p.Update(special(tea.KeyLeft))
	if p.model.InputCursor != 11 {
		t.Errorf("got %d", p.model.InputCursor)
	}
	p.Update(special(tea.KeyRight))
	if p.model.InputCursor != 12 {
		t.Errorf("got %d", p.model.InputCursor)
	}
	p.Update(ctrl('w'))
	if p.model.Input != "um dois " {
		t.Errorf("ctrl+w deletes a word, got %q", p.model.Input)
	}
	p.Update(ctrl('u'))
	if p.model.Input != "" || p.model.InputCursor != 0 {
		t.Errorf("ctrl+u clears the line, got %q", p.model.Input)
	}

	p.model = p.model.SetInput("cortar aqui")
	p.model.InputCursor = 7
	p.Update(ctrl('k'))
	if p.model.Input != "cortar " {
		t.Errorf("ctrl+k cuts to the end, got %q", p.model.Input)
	}
	p.model = p.model.SetInput("abc")
	p.model.InputCursor = 0
	p.Update(special(tea.KeyDelete))
	if p.model.Input != "bc" {
		t.Errorf("delete removes under the caret, got %q", p.model.Input)
	}
}

// `?` is a shortcut on an empty line and a character everywhere else: a key
// that eats what you type is worse than one you have to reach for.
func TestQuestionMarkOpensHelpOnlyOnAnEmptyLine(t *testing.T) {
	p, _ := newProgram(t)
	p.Update(key("?"))
	if len(p.model.Entries) == 0 || p.model.Entries[0].Kind != KindNote {
		t.Fatalf("? on an empty line opens help, got %+v", p.model.Entries)
	}

	p2, _ := newProgram(t)
	p2.model = p2.model.SetInput("porque")
	p2.Update(key("?"))
	if p2.model.Input != "porque?" {
		t.Errorf("mid-word it is a character, got %q", p2.model.Input)
	}
}

// An idle screen that keeps repainting burns a battery for no information.
func TestTheFrameOnlyAdvancesWhileRunning(t *testing.T) {
	p, _ := newProgram(t)
	p.model.State = protocol.SessionStateIdle
	p.Update(tickMsg(time.Unix(1, 0)))
	if p.model.Frame != 0 {
		t.Errorf("an idle session must not animate, got frame %d", p.model.Frame)
	}

	p.model.State = protocol.SessionStateRunning
	p.Update(tickMsg(time.Unix(2, 0)))
	if p.model.Frame != 1 {
		t.Errorf("got frame %d", p.model.Frame)
	}
	// The tick reschedules itself, or the animation stops after one frame.
	if _, cmd := p.Update(tickMsg(time.Unix(3, 0))); cmd == nil {
		t.Error("the tick must reschedule")
	}
}

func TestElapsedTimeComesFromTheInjectedClock(t *testing.T) {
	now := time.Unix(1000, 0)
	p, _ := newProgram(t, func(o *Options) { o.Now = func() time.Time { return now } })

	p.Update(eventMsg(ev(t, 1, protocol.EventTurnStarted, protocol.TurnStarted{TurnID: "t1"})))
	now = time.Unix(1042, 0)
	p.Update(tickMsg(now))

	if !strings.Contains(Render(p.model, p.geo), "42.0s") {
		t.Errorf("the turn's elapsed time must show:\n%s", Render(p.model, p.geo))
	}
}

// A sent line joins the history so it can be recalled.
func TestSendingRemembersTheLine(t *testing.T) {
	p, _ := newProgram(t)
	run(t, p, typeLine(t, p, "lembrar disso"))
	if len(p.model.History) != 1 || p.model.History[0] != "lembrar disso" {
		t.Errorf("got %v", p.model.History)
	}
}

// ---------- the `/` menu through the keyboard ----------

func withCommands(t *testing.T) (*program, *fakeTransport) {
	t.Helper()
	return newProgram(t, func(o *Options) {
		o.Commands = userCommands(config.Command{
			Name: "revisar", Description: "revisa o diff", Body: "Revise.",
		})
	})
}

// The menu owns the arrows, Tab and Esc while it is open — and nothing else, so
// every other key keeps doing what it always does.
func TestTheMenuOpensOnSlashAndOwnsItsKeys(t *testing.T) {
	p, tr := withCommands(t)

	p.Update(key("/"))
	if len(p.model.Completions) == 0 {
		t.Fatal("typing / opens the menu")
	}

	first := p.model.Completions[p.model.CompletionAt].Name
	p.Update(special(tea.KeyDown))
	if p.model.Completions[p.model.CompletionAt].Name == first {
		t.Error("down must move the highlight")
	}
	p.Update(special(tea.KeyUp))
	if got := p.model.Completions[p.model.CompletionAt].Name; got != first {
		t.Errorf("up must come back, got %q", got)
	}

	p.Update(special(tea.KeyTab))
	if !strings.HasPrefix(p.model.Input, "/"+first) {
		t.Errorf("tab completes the highlight, got %q", p.model.Input)
	}
	if len(p.model.Completions) != 0 {
		t.Error("and closes the menu")
	}
	if len(tr.submits()) != 0 {
		t.Error("completing is not sending")
	}
}

func TestEscapeClosesTheMenuWithoutClearingTheLine(t *testing.T) {
	p, _ := withCommands(t)
	for _, r := range "/pl" {
		p.Update(tea.KeyPressMsg{Code: r, Text: string(r)})
	}
	if len(p.model.Completions) == 0 {
		t.Fatal("setup")
	}

	p.Update(special(tea.KeyEscape))
	if len(p.model.Completions) != 0 {
		t.Fatal("esc closes the menu")
	}
	if p.model.Input != "/pl" {
		t.Errorf("and leaves the line alone, got %q", p.model.Input)
	}
	// A second escape does nothing while the line still has text on it:
	// abandoning a line somebody is halfway through writing is not what escape
	// is for here, and stepping into the transcript from a written line would
	// be worse.
	p.Update(special(tea.KeyEscape))
	if p.model.Navigating {
		t.Error("esc stepped into the transcript from a line with text on it")
	}
	if p.model.Input != "/pl" {
		t.Errorf("and the line is still there, got %q", p.model.Input)
	}
}

// Deleting reopens it, because the line changed.
func TestBackspaceReopensTheMenu(t *testing.T) {
	p, _ := withCommands(t)
	for _, r := range "/plan" {
		p.Update(tea.KeyPressMsg{Code: r, Text: string(r)})
	}
	p.Update(special(tea.KeyEscape))
	p.Update(special(tea.KeyBackspace))
	if len(p.model.Completions) == 0 {
		t.Errorf("editing the line reopens the menu, input is %q", p.model.Input)
	}
}

// Enter still sends. The menu must not swallow the one key that matters most.
func TestEnterSendsWithTheMenuOpen(t *testing.T) {
	p, tr := withCommands(t)
	for _, r := range "/help" {
		p.Update(tea.KeyPressMsg{Code: r, Text: string(r)})
	}
	_, cmd := p.Update(special(tea.KeyEnter))
	run(t, p, cmd)

	if len(p.model.Completions) != 0 {
		t.Error("sending closes the menu")
	}
	// /help is answered locally, so nothing reaches the daemon — but the line
	// was consumed, which is the point.
	if p.model.Input != "" {
		t.Errorf("got %q", p.model.Input)
	}
	_ = tr
}

// ---------- the queue ----------

// A message the user cannot take back is worse than one that was refused.
func TestCtrlXRemovesTheOldestQueuedMessage(t *testing.T) {
	p, _ := newProgram(t)
	p.model.State = protocol.SessionStateRunning
	// A picture is what still puts typed text in the queue: ordinary words now
	// steer, and half a pair is not sent.
	p.model.Attached = []protocol.TurnImage{{MediaType: "image/png", Data: "eA=="}}
	run(t, p, typeLine(t, p, "primeira"))
	p.model.Attached = []protocol.TurnImage{{MediaType: "image/png", Data: "eA=="}}
	run(t, p, typeLine(t, p, "segunda"))
	if len(p.model.Queue) != 2 {
		t.Fatalf("setup: %v", p.model.Queue)
	}

	p.Update(ctrl('x'))
	if len(p.model.Queue) != 1 || p.model.Queue[0] != "segunda" {
		t.Errorf("the oldest goes first, got %v", p.model.Queue)
	}
	p.Update(ctrl('x'))
	p.Update(ctrl('x'))
	if len(p.model.Queue) != 0 {
		t.Errorf("emptying is a no-op past the end, got %v", p.model.Queue)
	}
}

func TestTheQueueIsVisibleWithItsKey(t *testing.T) {
	m := NewModel("s", "/w", "m", "read-only", En)
	m.Entries = []Entry{{Kind: KindAssistant, Summary: "x"}}
	m, _ = m.Enqueue("depois roda o benchmark", 10)
	m, _ = m.Enqueue("e atualiza o godoc", 10)

	got := Render(m, DefaultGeometry(90, 14))
	for _, want := range []string{"depois roda o benchmark", "e atualiza o godoc", "^X remove"} {
		if !strings.Contains(got, want) {
			t.Errorf("%q missing from:\n%s", want, got)
		}
	}
	// The key is stated once, not on every row.
	if n := strings.Count(got, "^X remove"); n != 1 {
		t.Errorf("the key must be stated once, got %d", n)
	}
}

// The queue and the menu take their rows from the stream, or the last lines of
// output are drawn underneath the input.
func TestTheQueueAndMenuTakeRowsFromTheStream(t *testing.T) {
	g := DefaultGeometry(90, 14)
	plain := longStream(100)
	base := BodyHeight(plain, g)

	queued := plain
	queued, _ = queued.Enqueue("um", 10)
	queued, _ = queued.Enqueue("dois", 10)
	if got := BodyHeight(queued, g); got != base-2 {
		t.Errorf("two queued rows cost two stream rows, got %d want %d", got, base-2)
	}

	withMenu := plain.SetInput("/").Refresh(userCommands())
	if BodyHeight(withMenu, g) >= base {
		t.Error("the menu costs rows too")
	}
	for _, m := range []Model{queued, withMenu} {
		if n := len(lines(Render(m, g))); n > 14 {
			t.Errorf("the render must still fit the height, got %d lines", n)
		}
	}
}

// Regression: pasting did nothing at all.
//
// A terminal with bracketed paste sends the whole block as one PasteMsg rather
// than as a burst of key presses, and Update only handled key presses — so the
// paste fell through to the default case and was dropped in silence. Nothing on
// screen changed, which reads as a broken terminal rather than a missing case.
func TestPastingInsertsTheText(t *testing.T) {
	p, _ := newProgram(t)
	p.model = p.model.SetInput("antes ")

	p.Update(tea.PasteMsg{Content: "texto colado"})
	if p.model.Input != "antes texto colado" {
		t.Fatalf("got %q", p.model.Input)
	}
	if p.model.InputCursor != len([]rune(p.model.Input)) {
		t.Errorf("the caret follows the paste, got %d", p.model.InputCursor)
	}
}

// A pasted block routinely carries newlines — a stack trace, a diff, a log.
// Each one would otherwise be an Enter, so a three-line paste would send three
// turns and lose the last line.
func TestAMultiLinePasteDoesNotSendAnything(t *testing.T) {
	p, tr := newProgram(t)
	p.Update(tea.PasteMsg{Content: "primeira\nsegunda\nterceira"})

	if len(tr.submits()) != 0 {
		t.Fatalf("pasting is not sending, got %v", tr.submits())
	}
	if !strings.Contains(p.model.Input, "primeira") || !strings.Contains(p.model.Input, "terceira") {
		t.Errorf("no line may be lost: %q", p.model.Input)
	}
	// The breaks are kept. They used to become spaces, because the input was
	// one row and a raw newline would have broken the render — a pasted stack
	// trace arriving as one long line was the lesser damage.
	//
	// The box holds rows now, so a pasted stack trace reads as a stack trace.
	// What is still true is that pasting sends nothing, which is what the
	// first assertion above is for.
	if strings.Count(p.model.Input, "\n") != 2 {
		t.Errorf("the pasted breaks were flattened: %q", p.model.Input)
	}
}

// The modal is the one moment the user has to read, and a paste is not consent.
func TestPastingIsIgnoredWhileTheModalIsOpen(t *testing.T) {
	p, tr := newProgram(t)
	p.model.Pending = &protocol.ApprovalRequest{ApprovalID: "a1", Tool: "bash"}

	p.Update(tea.PasteMsg{Content: "qualquer coisa"})
	if p.model.Input != "" {
		t.Errorf("the modal swallows a paste too, got %q", p.model.Input)
	}
	if len(tr.resolved) != 0 {
		t.Errorf("and it certainly does not answer, got %v", tr.resolved)
	}
}

// A paste is an edit, so the completion menu follows it like any other.
func TestPastingASlashOpensTheMenu(t *testing.T) {
	p, _ := withCommands(t)
	p.Update(tea.PasteMsg{Content: "/pl"})
	if len(p.model.Completions) == 0 {
		t.Errorf("got input %q with no menu", p.model.Input)
	}
}

// Typing Portuguese produced Portuguese with the accents missing, silently.
//
// The insertion path asked `len(k.String()) == 1`, and len on a string counts
// BYTES: "á", "ç" and "ã" are two bytes each in UTF-8, so every accented
// character failed the test and was dropped on the floor. Nothing reported it —
// the letter simply did not appear, and the sentence that reached the model was
// not the sentence that was typed.
//
// The cost is not cosmetic. "não" becomes "no", which reverses it.
func TestAccentedCharactersReachTheInput(t *testing.T) {
	p, _ := newProgram(t)

	for _, s := range []string{"n", "ã", "o", " ", "é", " ", "ç", "ã", "o"} {
		p.onKey(key(s))
	}
	if got := p.model.Input; got != "não ção" && got != "não é ção" {
		t.Errorf("input = %q, want the accented text that was typed", got)
	}
}

// Every letter Portuguese needs, plus the ones the other languages the
// interface ships in need. A table because the failure was per-character: the
// ASCII ones always worked, which is why it looked like the input was "mostly"
// fine.
func TestEveryAccentedLetterSurvivesBeingTyped(t *testing.T) {
	for _, r := range []string{
		"á", "à", "â", "ã", "ä", "é", "ê", "í", "ó", "ô", "õ", "ú", "ü", "ç",
		"Á", "Ã", "Ç", "É", "Ô",
		"ñ", "ß", "ø", "å", "€", "—", "…",
	} {
		p, _ := newProgram(t)
		p.onKey(key(r))
		if got := p.model.Input; got != r {
			t.Errorf("typing %q produced %q", r, got)
		}
	}
}

// `?` opens help only as the first character of an empty line, and that rule is
// about the character rather than its byte count. An accented letter must not
// inherit the special case, and `?` must keep it.
func TestTheHelpShortcutStillOnlyFiresOnAnEmptyLine(t *testing.T) {
	p, _ := newProgram(t)
	p.onKey(key("?"))
	if p.model.Input != "" {
		t.Errorf("? on an empty line typed a character instead of opening help: %q", p.model.Input)
	}

	p, _ = newProgram(t)
	p.onKey(key("á"))
	p.onKey(key("?"))
	if got := p.model.Input; got != "á?" {
		t.Errorf("input = %q; ? mid-line is a character, not a shortcut", got)
	}
}

// Taking the keypress's text widens what counts as typing, so the other half
// has to hold: a key that produces no text types nothing. Otherwise the fix for
// dropped letters becomes stray characters from arrows and function keys.
func TestAKeyThatProducesNoTextTypesNothing(t *testing.T) {
	p, _ := newProgram(t)
	for _, code := range []rune{tea.KeyUp, tea.KeyDown, tea.KeyLeft, tea.KeyRight, tea.KeyEsc} {
		p.onKey(special(code))
	}
	if got := p.model.Input; got != "" {
		t.Errorf("navigation keys typed %q", got)
	}
}

// A paste arrives as one keypress carrying many characters. The old branch
// rejected anything longer than a byte, so pasting a path or an error message
// put nothing on the line at all.
func TestAPasteArrivesWhole(t *testing.T) {
	p, _ := newProgram(t)
	p.onKey(key("corrigir o parser de configuração"))
	if got := p.model.Input; got != "corrigir o parser de configuração" {
		t.Errorf("input = %q", got)
	}
}

// The network question is the one whose answer can be written down, so it is
// the one that offers the two standing options.
func TestTheNetworkApprovalOffersTheAnswersThatOutliveTheSession(t *testing.T) {
	p, tr := newProgram(t)
	p.model.Pending = &protocol.ApprovalRequest{
		ApprovalID: "a1", Tool: "bash", Command: "go test ./...",
		BoundaryCrossed: "network",
	}

	screen := Render(p.model, DefaultGeometry(100, 30))
	// It says what the answer does. "Allow this command" would promise
	// something narrower than what saying yes actually opens.
	if !strings.Contains(screen, "this project may reach the network") &&
		!strings.Contains(screen, "project may reach the network") {
		t.Errorf("the modal does not say the answer is about the project:\n%s", screen)
	}
	for _, key := range []string{"[P]", "[G]"} {
		if !strings.Contains(screen, key) {
			t.Errorf("the modal does not offer %s:\n%s", key, screen)
		}
	}

	_, cmd := p.onApprovalKey(key("P"))
	run(t, p, cmd)
	if got := tr.resolvedDecisions(); len(got) != 1 || got[0] != protocol.ApprovalAllowProject {
		t.Errorf("decisions = %v, want one allow_project", got)
	}
}

// Everything else keeps the three answers it had. A standing grant over "write
// outside the workspace" would be a permission given once for a reason nobody
// records, and the path in that question is what makes it answerable.
func TestAnyOtherCrossingOffersNoStandingAnswer(t *testing.T) {
	p, tr := newProgram(t)
	p.model.Pending = &protocol.ApprovalRequest{
		ApprovalID: "a1", Tool: "write", Command: "",
		BoundaryCrossed: "filesystem_write",
	}

	screen := Render(p.model, DefaultGeometry(100, 30))
	for _, key := range []string{"[P]", "[G]"} {
		if strings.Contains(screen, key) {
			t.Errorf("a crossing nobody can grant standing offered %s:\n%s", key, screen)
		}
	}

	// And pressing them does nothing rather than resolving by accident.
	_, first := p.onApprovalKey(key("P"))
	_, second := p.onApprovalKey(key("G"))
	run(t, p, first)
	run(t, p, second)
	if got := tr.resolvedDecisions(); len(got) != 0 {
		t.Errorf("a key that is not offered resolved the approval: %v", got)
	}
}

// resolvedDecisions returns what the client answered.
func (f *fakeTransport) resolvedDecisions() []protocol.ApprovalDecision {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]protocol.ApprovalDecision(nil), f.resolved...)
}

// Enter sends, so a list could not be typed: every line collapsed into one
// paragraph. This is the feature, driven through the keyboard rather than
// asserted on the helpers underneath it.
//
// Three bindings, because most terminals send the same bytes for enter and
// shift+enter — the modifier only survives where the terminal answers Bubble
// Tea's disambiguation request. ctrl+j needs nothing anywhere, and alt+enter
// covers the middle.
func TestBreakingALineDoesNotSend(t *testing.T) {
	for _, k := range []tea.KeyPressMsg{
		{Code: tea.KeyEnter, Mod: tea.ModShift},
		{Code: tea.KeyEnter, Mod: tea.ModAlt},
		{Code: 'j', Mod: tea.ModCtrl},
	} {
		p, tr := newProgram(t)
		p.Update(key("a"))
		p.Update(k)
		p.Update(key("b"))

		if got := p.model.Input; got != "a\nb" {
			t.Errorf("%v produced %q, want a break between the letters", k, got)
		}
		if len(tr.submits()) != 0 {
			t.Errorf("%v sent the turn: %v", k, tr.submits())
		}
	}
}

// And plain enter still sends, which is the half nobody would notice breaking
// until it did.
func TestPlainEnterStillSends(t *testing.T) {
	p, tr := newProgram(t)
	run(t, p, typeLine(t, p, "a"))

	if len(tr.submits()) != 1 {
		t.Fatalf("enter sent %d turns, want 1", len(tr.submits()))
	}
	if p.model.Input != "" {
		t.Errorf("the line was not cleared: %q", p.model.Input)
	}
}

// A client that did not do the typing still sees what was asked.
//
// This is the case the local echo could never cover: the only copy of the
// question lived in the client that typed it, so a second client attaching —
// or this one replaying after a reconnect — saw answers to questions it never
// saw, and a recorded session read the same way.
func TestAnAttachingClientSeesTheQuestion(t *testing.T) {
	p, _ := newProgram(t)

	p.model = p.model.Apply(ev(t, 1, protocol.EventTurnStarted,
		protocol.TurnStarted{TurnID: "t1", Text: "what does Rows do?"}))

	if len(p.model.Entries) != 1 || p.model.Entries[0].Kind != KindUser {
		t.Fatalf("nothing was shown for a turn this client did not start: %+v", p.model.Entries)
	}
	if p.model.Entries[0].Summary != "what does Rows do?" {
		t.Errorf("shown %q", p.model.Entries[0].Summary)
	}
}

// A turn with no text announces no line. Nothing else emits turn.started
// today, but a blank entry would be a visible artefact of an invisible cause.
func TestATurnWithoutTextShowsNoLine(t *testing.T) {
	p, _ := newProgram(t)
	p.model = p.model.Apply(ev(t, 1, protocol.EventTurnStarted, protocol.TurnStarted{TurnID: "t1"}))
	if len(p.model.Entries) != 0 {
		t.Errorf("a textless turn produced %+v", p.model.Entries)
	}
}

// Both halves are reported. A silent undo leaves the person guessing which of
// seven files went back, and the one that did NOT is the one they most need to
// hear about: it stayed because they had changed it themselves.
func TestUndoSaysWhatWentBackAndWhatDidNot(t *testing.T) {
	p, tr := newProgram(t)
	tr.undoResult = protocol.UndoResult{
		Restored: []string{"stats.go", "report.go"},
		Refused:  []string{"notes.md"},
	}

	msg := run(t, p, p.undo())
	note, ok := msg.(noteMsg)
	if !ok {
		t.Fatalf("got %T", msg)
	}
	for _, want := range []string{"stats.go", "report.go", "notes.md"} {
		if !strings.Contains(string(note), want) {
			t.Errorf("the note does not mention %q:\n%s", want, note)
		}
	}
	if tr.undos != 1 {
		t.Errorf("asked the session %d times", tr.undos)
	}
}

// A turn that changed nothing says so rather than looking like it worked.
func TestUndoingNothingSaysSo(t *testing.T) {
	p, tr := newProgram(t)
	tr.undoResult = protocol.UndoResult{}

	note, ok := run(t, p, p.undo()).(noteMsg)
	if !ok {
		t.Fatal("expected a note")
	}
	if !strings.Contains(string(note), "no files") {
		t.Errorf("got %q", note)
	}
}

// A refused undo is reported, not swallowed. The commonest reason is a turn
// still running, and somebody who is not told will assume it worked.
func TestAFailedUndoIsReported(t *testing.T) {
	p, tr := newProgram(t)
	tr.undoErr = errors.New("a turn is running; interrupt it before undoing")

	note, ok := run(t, p, p.undo()).(noteMsg)
	if !ok {
		t.Fatal("expected a note")
	}
	if !strings.Contains(string(note), "a turn is running") {
		t.Errorf("got %q", note)
	}
}

// Every outcome is said out loud. A key that sometimes does nothing for three
// different reasons looks identical each time, and an empty clipboard, a
// machine with no way to read it, and a model that cannot see are three
// different things to do next.
func TestPastingAnImageSaysWhatHappened(t *testing.T) {
	p, _ := newProgram(t, func(o *Options) { o.AcceptsImages = false })
	p.model.Model = "text-only-model"

	if _, _ = p.pasteImage(); len(p.model.Entries) != 1 {
		t.Fatal("nothing was said")
	}
	if !strings.Contains(p.model.Entries[0].Summary, "text-only-model") {
		t.Errorf("the note does not name the model: %q", p.model.Entries[0].Summary)
	}
	if !strings.Contains(p.model.Entries[0].Summary, "/model") {
		t.Errorf("the note does not say what to do: %q", p.model.Entries[0].Summary)
	}
}

// A picture waits for a question. Sending it alone is a turn the model has to
// guess the point of, and the attachment rides with the next message.
func TestAnAttachedImageGoesWithTheNextMessage(t *testing.T) {
	p, tr := newProgram(t)
	p.model.Attached = []protocol.TurnImage{{MediaType: "image/png", Data: "eA=="}}

	run(t, p, typeLine(t, p, "why is this cut off?"))

	if len(tr.submittedImages) != 1 || tr.submittedImages[0] != 1 {
		t.Errorf("the image did not go with the message: %v", tr.submittedImages)
	}
	// And it is not sent twice: the next message starts clean.
	if len(p.model.Attached) != 0 {
		t.Errorf("the attachment survived the send: %+v", p.model.Attached)
	}
}

// /image takes a path, for a file that is already on disk and for a daemon on
// another machine where a clipboard is not a shared thing.
func TestAttachingAnImageByPath(t *testing.T) {
	dir := t.TempDir()
	png := filepath.Join(dir, "shot.png")
	if err := os.WriteFile(png, []byte("\x89PNG\r\n\x1a\n....."), 0o644); err != nil {
		t.Fatal(err)
	}

	p, _ := newProgram(t, func(o *Options) { o.AcceptsImages = true })
	p.attachImage(png)

	if len(p.model.Attached) != 1 {
		t.Fatalf("nothing was attached: %+v", p.model.Entries)
	}
	if p.model.Attached[0].MediaType != "image/png" {
		t.Errorf("media type is %q", p.model.Attached[0].MediaType)
	}
}

// A path that is not a picture is refused with the reason, while the person can
// still pick another file.
func TestAttachingSomethingThatIsNotAnImage(t *testing.T) {
	dir := t.TempDir()
	notes := filepath.Join(dir, "notes.txt")
	if err := os.WriteFile(notes, []byte("text"), 0o644); err != nil {
		t.Fatal(err)
	}

	p, _ := newProgram(t, func(o *Options) { o.AcceptsImages = true })
	p.attachImage(notes)

	if len(p.model.Attached) != 0 {
		t.Error("a text file was attached as a picture")
	}
	if len(p.model.Entries) != 1 || !strings.Contains(p.model.Entries[0].Summary, "not an image") {
		t.Errorf("the refusal does not say why: %+v", p.model.Entries)
	}
}

// No path at all says how to use it rather than doing nothing.
func TestAttachingWithNoPathExplainsItself(t *testing.T) {
	p, _ := newProgram(t, func(o *Options) { o.AcceptsImages = true })
	p.attachImage("")
	if len(p.model.Entries) != 1 || !strings.Contains(p.model.Entries[0].Summary, "Usage") {
		t.Errorf("got %+v", p.model.Entries)
	}
}

// The clipboard path answers whatever the machine says, and never leaves the
// person guessing which of the outcomes they got.
func TestPastingReportsEveryOutcome(t *testing.T) {
	p, _ := newProgram(t, func(o *Options) { o.AcceptsImages = true })
	p.pasteImage()

	if len(p.model.Entries) != 1 {
		t.Fatalf("pasting said nothing: %+v", p.model.Entries)
	}
	said := p.model.Entries[0].Summary
	// On a machine with no clipboard and on one with an empty clipboard the
	// note differs, and either is a real answer. What must never happen is
	// silence, or an attachment appearing from nowhere.
	if said == "" {
		t.Error("an empty note")
	}
	if len(p.model.Attached) != 0 && !strings.Contains(said, "image") {
		t.Errorf("something was attached without saying so: %q", said)
	}
}

// A model that cannot see is told before anything is read, so the refusal
// names the model rather than the clipboard.
func TestPastingIntoAModelThatCannotSeeNamesTheModel(t *testing.T) {
	p, _ := newProgram(t, func(o *Options) { o.AcceptsImages = false })
	p.model.Model = "some-text-model"
	p.pasteImage()

	if len(p.model.Attached) != 0 {
		t.Error("a picture was attached for a model that cannot read one")
	}
	if !strings.Contains(p.model.Entries[0].Summary, "some-text-model") {
		t.Errorf("got %q", p.model.Entries[0].Summary)
	}
}

// NAV owns the keyboard, and every key it does not name is swallowed.
//
// That is what makes a letter safe inside it, and it is the rule copy mode
// already carries: a mode that lets unknown keys through is a mode people leave
// by accident, and the keys that reach the input line arrive as text nobody
// meant to type.
func TestNavModeSwallowsEveryKeyItDoesNotName(t *testing.T) {
	p, _ := newProgram(t)
	p.model.Entries = []Entry{{Kind: KindUser, Summary: "algo"}, {Kind: KindAssistant, Summary: "b"}}
	p.Update(special(tea.KeyEsc))
	if !p.model.Navigating {
		t.Fatal("esc did not open the mode")
	}

	for _, r := range "voce escreveu isto" {
		p.Update(key(string(r)))
	}
	if p.model.Input != "" {
		t.Errorf("keys reached the input line from inside the mode: %q", p.model.Input)
	}
	if !p.model.Navigating {
		t.Error("an unnamed key left the mode")
	}

	// And the keys it DOES name work, letters included.
	p.model.Cursor = 1
	p.Update(key("k"))
	if p.model.Cursor != 0 {
		t.Errorf("k did not move the cursor: %d", p.model.Cursor)
	}
	p.Update(key("j"))
	if p.model.Cursor != 1 {
		t.Errorf("j did not move the cursor: %d", p.model.Cursor)
	}
}

// The theme key is a letter, and it exists only where a letter is safe.
//
// Outside the mode `t` is the first character of "tenta", "test", "the" — which
// is the defect this product has fixed twice, at the user's report both times.
func TestTheThemeKeyOnlyExistsInsideTheMode(t *testing.T) {
	p, _ := newProgram(t)
	p.model.Entries = []Entry{{Kind: KindUser, Summary: "algo"}}
	first := p.geo.Palette.Theme.Name

	// Outside: a letter.
	p.Update(key("t"))
	if p.model.Input != "t" {
		t.Errorf("t was eaten outside the mode: input %q", p.model.Input)
	}
	if p.geo.Palette.Theme.Name != first {
		t.Error("typing t changed the theme")
	}

	// Inside: the theme, and it comes back round.
	p.model = p.model.SetInput("")
	p.Update(special(tea.KeyEsc))
	seen := map[string]bool{}
	for i := 0; i < len(Themes()); i++ {
		p.Update(key("t"))
		seen[p.geo.Palette.Theme.Name] = true
	}
	if len(seen) != len(Themes()) {
		t.Errorf("cycling reached %d of %d themes: %v", len(seen), len(Themes()), seen)
	}
	if p.geo.Palette.Theme.Name != first {
		t.Errorf("a full cycle did not come back to %q, got %q", first, p.geo.Palette.Theme.Name)
	}
}

// Resuming paints one screen, not one per event.
//
// Continuing a conversation writes the whole of the old log into the new
// session, so attaching replays every event of it — 3544, on a real session of
// this machine. Each arrives as its own message and Bubble Tea paints after
// every message, so resuming redrew the screen 3544 times with the window
// following its own end. That is the screen that would not stop scrolling.
func TestResumingPaintsALoadingLineUntilTheBacklogIsRead(t *testing.T) {
	p, _ := newProgram(t)
	p.opts.Backlog = 40
	p.model.Lang = En

	// Mid-backlog: one line, and it says what it is doing rather than showing
	// a screen built from half a conversation.
	p.model.LastSeq = 12
	p.model.Entries = []Entry{{Kind: KindUser, Summary: "algo"}}
	got := p.View().Content
	if !strings.Contains(got, "reading the conversation") {
		t.Errorf("the screen does not say it is reading:\n%s", got)
	}
	if n := len(lines(strings.TrimRight(got, "\n "))); n != 1 {
		t.Errorf("the loading screen is %d lines, want 1", n)
	}

	// Caught up: the conversation.
	p.model.LastSeq = 40
	got = p.View().Content
	if strings.Contains(got, "reading the conversation") {
		t.Errorf("the loading line outlived the backlog:\n%s", got)
	}
	if !strings.Contains(got, "algo") {
		t.Errorf("the conversation is not on screen:\n%s", got)
	}
}

// And the line moves while it reads. The session is IDLE during a replay —
// nothing is running — and the tick stops when idle, so without this the
// spinner froze on a screen that says "reading", which reads as stuck.
func TestTheLoadingLineKeepsTicking(t *testing.T) {
	p, _ := newProgram(t)
	p.opts.Backlog = 40
	p.model.LastSeq, p.model.State = 5, protocol.SessionStateIdle

	if p.resumeTicking() == nil {
		t.Error("the tick does not run while history is being read")
	}
	p.model.LastSeq = 40
	p.ticking = false
	if p.resumeTicking() != nil {
		t.Error("the tick outlived the backlog on an idle session")
	}
}
