package tui

import (
	"context"
	"errors"
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
	interrupts int
	resolved   []protocol.ApprovalDecision
	created    []protocol.CreateSessionRequest
	sessions   []protocol.Session

	submitErr error
	listErr   error
	createErr error
	getErr    error

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

func (f *fakeTransport) Submit(_ context.Context, _, text string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.submitted = append(f.submitted, text)
	return f.submitErr
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
		opts: o, model: NewModel(o.SessionID, o.Workspace, o.Model, o.Sandbox),
		geo: o.Geometry, ctx: ctx, cancel: cancel,
	}
	p.attach(o.SessionID)
	return p, tr
}

// key builds a printable keypress.
func key(s string) tea.KeyPressMsg {
	return tea.KeyPressMsg{Code: rune(s[0]), Text: s}
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
	// turn looks like nothing happened.
	if len(p.model.Entries) != 1 || p.model.Entries[0].Kind != KindUser {
		t.Errorf("got %+v", p.model.Entries)
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

// The protocol refuses a concurrent turn; queueing is what turns the refusal
// into a usable experience.
func TestInputIsQueuedWhileRunningAndDrainsAsOneTurn(t *testing.T) {
	p, tr := newProgram(t)
	p.model.State = protocol.SessionStateRunning

	run(t, p, typeLine(t, p, "one"))
	run(t, p, typeLine(t, p, "two"))
	if len(tr.submits()) != 0 {
		t.Fatalf("nothing may be sent mid-turn, got %v", tr.submits())
	}

	// A third is refused loudly rather than dropped.
	run(t, p, typeLine(t, p, "three"))
	var refused bool
	for _, e := range p.model.Entries {
		if e.Kind == KindError && strings.Contains(e.Summary, "queue is full") {
			refused = true
		}
	}
	if !refused {
		t.Error("a full queue must say so")
	}

	// Going idle drains everything into one turn.
	_, cmd := p.Update(eventMsg(ev(t, 1, protocol.EventTurnCompleted,
		protocol.TurnCompleted{TurnID: "t1"})))
	drainBatch(t, p, cmd)

	got := tr.submits()
	if len(got) != 1 {
		t.Fatalf("the queue must become exactly one turn, got %v", got)
	}
	if !strings.Contains(got[0], "one") || !strings.Contains(got[0], "two") {
		t.Errorf("got %q", got[0])
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
	// Up from nothing selects the last entry: the newest is what the user is
	// looking at.
	p.Update(special(tea.KeyUp))
	if p.model.Cursor != 1 {
		t.Fatalf("got %d", p.model.Cursor)
	}
	p.Update(special(tea.KeyUp))
	if p.model.Cursor != 0 {
		t.Fatalf("got %d", p.model.Cursor)
	}
	p.Update(special(tea.KeyUp))
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

// `p` toggles the panel only when it is not being typed into a message.
func TestPToggleOnlyAppliesToAnEmptyInput(t *testing.T) {
	p, _ := newProgram(t)
	p.model.Plan = modelWithPlan().Plan

	// The panel is showing at 100 columns, so the first press hides it.
	p.Update(key("p"))
	if p.geo.ShowPanel(true) {
		t.Error("p on an empty line hides the panel")
	}
	p.Update(key("p"))
	if !p.geo.ShowPanel(true) {
		t.Error("and shows it again")
	}

	// On a terminal too narrow for the default, the key still works: an
	// explicit request beats the responsive default.
	narrow, _ := newProgram(t, func(o *Options) { o.Geometry = DefaultGeometry(80, 24) })
	narrow.model.Plan = modelWithPlan().Plan
	if narrow.geo.ShowPanel(true) {
		t.Fatal("80 columns hides it by default")
	}
	narrow.Update(key("p"))
	if !narrow.geo.ShowPanel(true) {
		t.Error("pressing p on a narrow terminal must show the panel anyway")
	}

	p.Update(key("x"))
	p.Update(key("p"))
	if p.model.Input != "xp" {
		t.Errorf("p mid-word is a letter, got %q", p.model.Input)
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
