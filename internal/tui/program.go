package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/aguinelo/dcode/internal/config"
	"github.com/aguinelo/dcode/internal/protocol"
	"github.com/aguinelo/dcode/pkg/client"
)

// Transport is what the TUI needs from a daemon. An interface rather than the
// concrete client so the program can be driven in tests without a socket.
type Transport interface {
	CreateSession(ctx context.Context, req protocol.CreateSessionRequest) (protocol.Session, error)
	ListSessions(ctx context.Context) ([]protocol.Session, error)
	GetSession(ctx context.Context, id string) (protocol.Session, error)
	Submit(ctx context.Context, id, text string) error
	Interrupt(ctx context.Context, id string) error
	Resolve(ctx context.Context, id, approvalID string, d protocol.ApprovalDecision) error
	Subscribe(ctx context.Context, id string, from uint64) (<-chan protocol.Event, <-chan error)
}

var _ Transport = (*client.Client)(nil)

// Options configure the program.
type Options struct {
	SessionID string
	Workspace string
	Model     string
	Sandbox   string
	// Window is the model's context window, so a token count can become the
	// percentage a person can act on.
	Window    int
	Transport Transport
	Geometry  Geometry
	QueueMax  int

	// Commands is the user's discovered command set. Frozen at start, like the
	// instruction chain, so behaviour cannot change mid-session.
	Commands config.CommandSet

	// Lookup answers `/config <key>`. Injected rather than read here, because
	// the client is not where configuration is resolved.
	Lookup func(key string) (string, bool)

	// Notice is the passive version check. It runs off the critical path and
	// its failure is silent by contract.
	Notice func(context.Context) string

	// Now is the clock for elapsed time. Injected so a test can assert an
	// exact duration instead of sleeping for one.
	Now func() time.Time
}

type program struct {
	opts   Options
	model  Model
	geo    Geometry
	ctx    context.Context
	cancel context.CancelFunc

	events <-chan protocol.Event
	errs   <-chan error
	fatal  string

	// unsubscribe tears down the current subscription when the session is
	// replaced by /clear or /model.
	unsubscribe context.CancelFunc
}

// Run starts the TUI. It takes the alternate screen, which is what a fixed
// panel requires — there is no way to hold a region while appending to terminal
// flow. The cost is scroll-back and mouse selection, reimplemented here.
func Run(ctx context.Context, opts Options) error {
	if opts.QueueMax <= 0 {
		opts.QueueMax = 10
	}
	if opts.Geometry.Width == 0 {
		opts.Geometry = DefaultGeometry(100, 30)
	}

	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	p := &program{
		opts:  opts,
		model: NewModel(opts.SessionID, opts.Workspace, opts.Model, opts.Sandbox),
		geo:   opts.Geometry,
		ctx:   runCtx,
	}
	p.cancel = cancel
	p.model.Window = opts.Window
	p.model.Now = p.now()
	p.attach(opts.SessionID)

	// A user command that shadows a built-in is reported rather than obeyed:
	// the override that did not happen is exactly what would otherwise be spent
	// an afternoon on.
	for _, msg := range ShadowedBuiltins(opts.Commands) {
		p.model.Entries = append(p.model.Entries, Entry{Kind: KindNote, Summary: msg})
	}

	prog := tea.NewProgram(p, tea.WithContext(runCtx))
	_, err := prog.Run()
	return err
}

// attach subscribes to a session's event log from the beginning, which is what
// makes reattaching identical to having watched it live.
func (p *program) attach(id string) {
	if p.unsubscribe != nil {
		p.unsubscribe()
	}
	subCtx, cancel := context.WithCancel(p.ctx)
	p.unsubscribe = cancel
	p.events, p.errs = p.opts.Transport.Subscribe(subCtx, id, 1)
}

// ---------- bubbletea ----------

type eventMsg protocol.Event
type errMsg struct{ err error }
type streamClosedMsg struct{}
type noteMsg string
type switchedMsg struct{ session protocol.Session }

func (p *program) Init() tea.Cmd {
	cmds := []tea.Cmd{p.waitForEvent(), p.tick()}
	if p.opts.Notice != nil {
		cmds = append(cmds, p.checkVersion())
	}
	return tea.Batch(cmds...)
}

// tickInterval is the animation rate: fast enough to read as motion, slow
// enough that an idle session is not repainting ten times a second.
const tickInterval = 120 * time.Millisecond

type tickMsg time.Time

func (p *program) tick() tea.Cmd {
	return tea.Tick(tickInterval, func(t time.Time) tea.Msg { return tickMsg(t) })
}

func (p *program) now() time.Time {
	if p.opts.Now != nil {
		return p.opts.Now()
	}
	return time.Now()
}

func (p *program) checkVersion() tea.Cmd {
	return func() tea.Msg {
		if msg := p.opts.Notice(p.ctx); msg != "" {
			return noteMsg(msg)
		}
		return nil
	}
}

func (p *program) waitForEvent() tea.Cmd {
	events, errs := p.events, p.errs
	return func() tea.Msg {
		select {
		case ev, open := <-events:
			if !open {
				return streamClosedMsg{}
			}
			return eventMsg(ev)
		case err, open := <-errs:
			if !open {
				return streamClosedMsg{}
			}
			return errMsg{err}
		case <-p.ctx.Done():
			return streamClosedMsg{}
		}
	}
}

func (p *program) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		p.geo.Width, p.geo.Height = msg.Width, msg.Height
		return p, nil

	case tickMsg:
		p.model.Now = p.now()
		// The frame only advances while something is running: an idle screen
		// that keeps repainting burns a laptop battery for no information.
		if p.model.State == protocol.SessionStateRunning {
			p.model.Frame++
		}
		return p, p.tick()

	case noteMsg:
		p.model.Entries = append(p.model.Entries, Entry{Kind: KindNote, Summary: string(msg)})
		return p, nil

	case switchedMsg:
		// A new session means a new event log, so the view is rebuilt rather
		// than carried over: keeping entries from the old one would show a
		// history the model no longer has.
		p.opts.SessionID = msg.session.ID
		p.model = NewModel(msg.session.ID, msg.session.Workspace, msg.session.Model, msg.session.SandboxMode)
		p.attach(msg.session.ID)
		return p, p.waitForEvent()

	case eventMsg:
		p.model.Now = p.now()
		p.model = p.model.Apply(protocol.Event(msg))
		// The queue drains when the session goes idle: the protocol refuses a
		// concurrent turn, so waiting here is what turns a refusal into a
		// usable experience.
		var cmds []tea.Cmd
		if p.model.State == protocol.SessionStateIdle && len(p.model.Queue) > 0 {
			m, text := p.model.DrainQueue()
			p.model = m
			cmds = append(cmds, p.submit(text))
		}
		cmds = append(cmds, p.waitForEvent())
		return p, tea.Batch(cmds...)

	case errMsg:
		p.fatal = msg.err.Error()
		return p, tea.Quit

	case streamClosedMsg:
		return p, tea.Quit

	case tea.KeyPressMsg:
		return p.onKey(msg)
	}
	return p, nil
}

func (p *program) onKey(k tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	// The modal blocks input: it is the one moment the user has to read, and
	// letting keystrokes fall through would let them dismiss it by accident.
	if p.model.Pending != nil {
		return p.onApprovalKey(k)
	}

	// The menu owns the arrows, Tab and Esc while it is open — and nothing
	// else, so every other key keeps doing what it always does.
	if len(p.model.Completions) > 0 {
		switch k.String() {
		case "up":
			p.model = p.model.MoveCompletion(-1)
			return p, nil
		case "down":
			p.model = p.model.MoveCompletion(1)
			return p, nil
		case "tab":
			p.model = p.model.AcceptCompletion()
			return p, nil
		case "esc":
			p.model = p.model.CloseCompletions()
			return p, nil
		}
	}

	switch k.String() {
	case "ctrl+c":
		// Interrupt before quit. Interrupting is what is wanted in the
		// overwhelming majority of cases, and quitting mid-turn loses work.
		if p.model.State != protocol.SessionStateIdle {
			return p, p.interrupt()
		}
		return p, tea.Quit

	case "ctrl+d":
		return p, tea.Quit

	case "enter":
		return p.onEnter()

	// ---------- scrolling ----------
	//
	// These never depend on what is in the input line: a key that sometimes
	// scrolls and sometimes types is a key nobody trusts.
	case "pgup":
		p.model = p.model.ScrollBy(-PageSize(p.model, p.geo), p.geo)
		return p, nil
	case "pgdown":
		p.model = p.model.ScrollBy(PageSize(p.model, p.geo), p.geo)
		return p, nil
	case "home":
		p.model = p.model.ScrollToTop()
		return p, nil
	case "end":
		p.model = p.model.ScrollToBottom(p.geo)
		return p, nil
	case "ctrl+x":
		// The oldest goes: it is the one the user has had longest to change
		// their mind about, and the one nearest the top of the list.
		p.model = p.model.RemoveFromQueue(0)
		return p, nil

	case "ctrl+p":
		// The panel toggle is a control key, not a letter. As a bare `p` it ate
		// the first character of every message starting with one — "primeiro",
		// "please", "por favor" — which is the exact failure the rest of this
		// switch is written to avoid.
		if p.geo.ShowPanel(len(p.model.Plan) > 0) {
			p.geo.PanelMode = PanelHidden
		} else {
			p.geo.PanelMode = PanelShown
		}
		return p, nil

	case "shift+up", "ctrl+up":
		p.model = p.model.ScrollBy(-1, p.geo)
		return p, nil
	case "shift+down", "ctrl+down":
		p.model = p.model.ScrollBy(1, p.geo)
		return p, nil

	// ---------- the disputed arrows ----------
	//
	// Empty input means the user is reaching for what they typed before; input
	// with text means they are working on the stream. It is the reading that
	// costs nothing to be wrong about: with text on the line, history would
	// destroy what they were writing.
	case "up":
		if p.model.Input == "" && len(p.model.History) > 0 && p.model.Cursor < 0 {
			p.model = p.model.HistoryPrev()
			return p, nil
		}
		return p.moveCursor(-1)
	case "down":
		if p.model.HistoryAt >= 0 {
			p.model = p.model.HistoryNext()
			return p, nil
		}
		return p.moveCursor(1)

	// ---------- line editing ----------
	case "left":
		if p.model.InputCursor > 0 {
			p.model.InputCursor--
		}
		return p, nil
	case "right":
		if p.model.InputCursor < len([]rune(p.model.Input)) {
			p.model.InputCursor++
		}
		return p, nil
	case "ctrl+a":
		p.model.InputCursor = 0
		return p, nil
	case "ctrl+e":
		p.model.InputCursor = len([]rune(p.model.Input))
		return p, nil
	case "ctrl+u":
		p.model = p.model.SetInput("").Refresh(p.opts.Commands)
		return p, nil
	case "ctrl+w":
		p.model = p.model.DeleteWord().Refresh(p.opts.Commands)
		return p, nil
	case "ctrl+k":
		runes := []rune(p.model.Input)
		if p.model.InputCursor <= len(runes) {
			p.model.Input = string(runes[:p.model.InputCursor])
		}
		return p, nil

	case "backspace":
		p.model = p.model.Backspace().Refresh(p.opts.Commands)
		return p, nil
	case "delete":
		p.model = p.model.DeleteForward().Refresh(p.opts.Commands)
		return p, nil

	case "esc":
		// Closes the expansion first, then the selection. Escape means "back
		// out of what I opened", and the outermost thing opened is the last
		// thing it should abandon.
		if p.model.Cursor >= 0 && p.model.Cursor < len(p.model.Entries) &&
			p.model.Entries[p.model.Cursor].Expanded {
			p.model = p.model.ToggleAt(p.model.Cursor)
			return p, nil
		}
		p.model.Cursor = -1
		return p, nil

	case "tab":
		p.model = p.model.ToggleAt(p.model.Cursor)
		p.model = p.model.EnsureCursorVisible(p.geo)
		return p, nil
	}

	if s := k.String(); len(s) == 1 {
		// `?` opens help only as the first character of an empty line. It is
		// the convention every pager and monitor already uses, and a message
		// that opens with a question mark is rare enough to be worth the trade
		// — unlike a letter, which is not.
		if s == "?" && p.model.Input == "" {
			return p.runBuiltin(Resolved{Kind: CmdBuiltin, Name: "help"})
		}
		p.model = p.model.Insert(s).Refresh(p.opts.Commands)
	} else if k.String() == "space" {
		p.model = p.model.Insert(" ").Refresh(p.opts.Commands)
	}
	return p, nil
}

// moveCursor walks the stream and keeps the selection on screen.
func (p *program) moveCursor(delta int) (tea.Model, tea.Cmd) {
	n := len(p.model.Entries)
	if n == 0 {
		return p, nil
	}
	switch {
	case p.model.Cursor < 0:
		// Entering the stream from the input starts at the newest entry, which
		// is what the user is looking at.
		p.model.Cursor = n - 1
	default:
		p.model.Cursor += delta
	}
	if p.model.Cursor < 0 {
		p.model.Cursor = 0
	}
	if p.model.Cursor >= n {
		p.model.Cursor = n - 1
	}
	p.model = p.model.EnsureCursorVisible(p.geo)
	return p, nil
}

// onEnter is where an input line becomes either a turn, a queued turn, or a
// client-side answer.
func (p *program) onEnter() (tea.Model, tea.Cmd) {
	line := strings.TrimSpace(p.model.Input)
	if line == "" {
		return p, nil
	}
	p.model = p.model.SetInput("")
	p.model.Completions = nil
	p.model = p.model.Remember(line)

	r := ResolveInput(line, p.opts.Commands)
	switch r.Kind {
	case CmdBuiltin:
		return p.runBuiltin(r)
	case CmdUnknown:
		p.model.Entries = append(p.model.Entries, Entry{
			Kind: KindError, IsError: true, Expanded: true,
			Summary: fmt.Sprintf("/%s is not a command. Try /help.", r.Name),
		})
		return p, nil
	}

	// The user's own line, echoed before it is sent: a command that expanded
	// into three paragraphs should not vanish into the void while the model
	// thinks.
	p.model.Entries = append(p.model.Entries, Entry{Kind: KindUser, Summary: line})

	if p.model.State != protocol.SessionStateIdle {
		m, ok := p.model.Enqueue(r.Text, p.opts.QueueMax)
		p.model = m
		if !ok {
			// Refusing loudly beats dropping silently.
			p.model.Entries = append(p.model.Entries, Entry{
				Kind: KindError, Summary: "the queue is full; wait for the current turn",
				IsError: true, Expanded: true,
			})
		}
		return p, nil
	}
	return p, p.submit(r.Text)
}

func (p *program) runBuiltin(r Resolved) (tea.Model, tea.Cmd) {
	note := func(text string) (tea.Model, tea.Cmd) {
		p.model.Entries = append(p.model.Entries, Entry{
			Kind: KindNote, Summary: text, Expanded: true,
		})
		return p, nil
	}

	switch r.Name {
	case "help":
		return note(HelpText(p.opts.Commands))

	case "plan":
		if strings.TrimSpace(r.Args) == "" {
			return note(PlanText(p.model))
		}
		return p.sendOrQueue(ReplanPrompt(r.Args))

	case "init":
		return p.sendOrQueue(InitPrompt)

	case "config":
		key := strings.TrimSpace(r.Args)
		if key == "" {
			return note("Usage: /config <key>")
		}
		if p.opts.Lookup == nil {
			return note("configuration is not available in this client")
		}
		v, ok := p.opts.Lookup(key)
		if !ok {
			return note(key + " is not set")
		}
		return note(v)

	case "clear":
		// A fresh session rather than a cleared screen: context is server-side
		// and append-only, so there is no way to unsay something to the model.
		// Pretending otherwise by wiping the view would be a lie about what the
		// model still remembers.
		return p, p.newSession(p.opts.Model)

	case "model":
		name := strings.TrimSpace(r.Args)
		if name == "" {
			return note("Usage: /model <name>. Currently " + p.model.Model)
		}
		return p, p.newSession(name)

	case "resume":
		if id := strings.TrimSpace(r.Args); id != "" {
			return p, p.resume(id)
		}
		return p, p.listSessions()
	}
	return note("/" + r.Name + " is not implemented")
}

func (p *program) sendOrQueue(text string) (tea.Model, tea.Cmd) {
	if p.model.State != protocol.SessionStateIdle {
		m, _ := p.model.Enqueue(text, p.opts.QueueMax)
		p.model = m
		return p, nil
	}
	return p, p.submit(text)
}

func (p *program) onApprovalKey(k tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	id := p.model.Pending.ApprovalID
	switch k.String() {
	case "a":
		return p, p.resolve(id, protocol.ApprovalAllow)
	case "A", "shift+a":
		return p, p.resolve(id, protocol.ApprovalAllowSession)
	case "d", "enter", "esc":
		// Enter denies. The safe action is the default and the least effort.
		return p, p.resolve(id, protocol.ApprovalDeny)
	case "ctrl+c":
		return p, p.resolve(id, protocol.ApprovalDeny)
	}
	return p, nil
}

func (p *program) submit(text string) tea.Cmd {
	return func() tea.Msg {
		if err := p.opts.Transport.Submit(p.ctx, p.opts.SessionID, text); err != nil {
			return errMsg{err}
		}
		return nil
	}
}

func (p *program) interrupt() tea.Cmd {
	return func() tea.Msg {
		_ = p.opts.Transport.Interrupt(p.ctx, p.opts.SessionID)
		return nil
	}
}

func (p *program) resolve(approvalID string, d protocol.ApprovalDecision) tea.Cmd {
	return func() tea.Msg {
		_ = p.opts.Transport.Resolve(p.ctx, p.opts.SessionID, approvalID, d)
		return nil
	}
}

// newSession opens another session, which is the only honest way to change the
// model or start over: the system prompt is part of the prefix, and the prefix
// cannot be rewritten (ADR-03).
func (p *program) newSession(model string) tea.Cmd {
	return func() tea.Msg {
		s, err := p.opts.Transport.CreateSession(p.ctx, protocol.CreateSessionRequest{
			Workspace:   p.model.Workspace,
			Model:       model,
			SandboxMode: p.model.Sandbox,
		})
		if err != nil {
			return noteMsg("could not open a session: " + err.Error())
		}
		return switchedMsg{session: s}
	}
}

func (p *program) resume(id string) tea.Cmd {
	return func() tea.Msg {
		s, err := p.opts.Transport.GetSession(p.ctx, id)
		if err != nil {
			return noteMsg("could not resume " + id + ": " + err.Error())
		}
		return switchedMsg{session: s}
	}
}

func (p *program) listSessions() tea.Cmd {
	return func() tea.Msg {
		list, err := p.opts.Transport.ListSessions(p.ctx)
		if err != nil {
			return noteMsg("could not list sessions: " + err.Error())
		}
		return noteMsg(SessionList(list, p.opts.SessionID))
	}
}

// SessionList renders `/resume`. Pure, so the shape is testable.
func SessionList(list []protocol.Session, current string) string {
	if len(list) == 0 {
		return "There are no other sessions."
	}
	var b strings.Builder
	b.WriteString("Sessions — /resume <id> to reattach\n")
	for _, s := range list {
		marker := "  "
		if s.ID == current {
			marker = "* "
		}
		fmt.Fprintf(&b, "%s%s  %-14s %s\n", marker, s.ID, s.State, s.Workspace)
	}
	return strings.TrimRight(b.String(), "\n")
}

func (p *program) View() tea.View {
	body := Render(p.model, p.geo)
	if p.fatal != "" {
		body += "\n" + p.fatal + "\n"
	}
	v := tea.NewView(body)
	// The alternate screen is what makes a fixed panel possible at all: there
	// is no way to hold a region while appending to terminal flow. The cost -
	// native scroll-back and mouse selection - is accepted deliberately.
	v.AltScreen = true
	return v
}
