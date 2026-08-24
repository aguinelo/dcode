package tui

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"github.com/aguinelo/dcode/internal/session"
	"os"
	"path/filepath"
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
	Submit(ctx context.Context, id, text string, images ...protocol.TurnImage) error
	Interrupt(ctx context.Context, id string) error
	Steer(ctx context.Context, id, text string) error
	Undo(ctx context.Context, id string) (protocol.UndoResult, error)
	RenameSession(ctx context.Context, id, name string) error
	Resolve(ctx context.Context, id, approvalID string, d protocol.ApprovalDecision) error
	Subscribe(ctx context.Context, id string, from uint64) (<-chan protocol.Event, <-chan error)
}

var _ Transport = (*client.Client)(nil)

// MaxImageBytes is the largest picture that goes to the model.
//
// Ten megabytes, which is what the providers take. Refusing here rather than on
// the wire means the person hears "that file is too big" while they can still
// pick another one, instead of a rejected request after the turn started.
const MaxImageBytes = 10 << 20

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
	// Lang is the interface language, resolved once by the caller. Zero lands
	// on the fallback.
	Lang Lang

	// Sessions is what this workspace has recorded, read once at start by the
	// caller. Only the conversations something was asked in — the rest is what
	// a record directory mostly holds, and burying four real ones under thirty
	// empty ones is what the picker already refuses to do.
	Sessions []SessionChoice

	// Commands is the user's discovered command set. Frozen at start, like the
	// instruction chain, so behaviour cannot change mid-session.
	Commands config.CommandSet
	// AcceptsImages says whether this session's model reads pictures. Passed
	// in rather than guessed: the client cannot know, and guessing wrong turns
	// a refusal it could have given into a provider error it cannot explain.
	AcceptsImages bool

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

	// ticking says whether an animation tick is in flight. It exists so the
	// tick can stop when the session goes idle and be started again exactly
	// once when a turn begins.
	ticking bool
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
		opts: opts,
		model: func() Model {
			m := NewModel(opts.SessionID, opts.Workspace, opts.Model, opts.Sandbox, opts.Lang)
			m.Sessions = opts.Sessions
			return m
		}(),
		geo: opts.Geometry,
		ctx: runCtx,
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

// renamedMsg is a name that stuck.
type renamedMsg struct{ id, name string }
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
	p.ticking = true
	return tea.Tick(tickInterval, func(t time.Time) tea.Msg { return tickMsg(t) })
}

// resumeTicking starts the animation again when a turn begins.
//
// The tick stops itself when the session goes idle, so something has to start
// it, and the guard is what keeps two clocks from running: every event would
// otherwise add a tick, and the frame counter would sprint.
func (p *program) resumeTicking() tea.Cmd {
	if p.ticking || p.model.State != protocol.SessionStateRunning {
		return nil
	}
	return p.tick()
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
		// Idle: stop, rather than reschedule with the counter held still.
		//
		// The frame already stopped advancing here, and the comment said an
		// idle screen that keeps repainting burns a laptop battery for no
		// information — while the tick rescheduled anyway and the screen
		// repainted eight times a second for a number that never moved. The
		// sentence was right; it just was not being kept.
		//
		// Nothing is lost by stopping: Now is refreshed on every event, so the
		// clock a turn starts from is fresh whether or not a tick just ran.
		if p.model.State != protocol.SessionStateRunning {
			p.ticking = false
			return p, nil
		}
		p.model.Frame++
		return p, p.tick()

	case renamedMsg:
		for i := range p.model.Sessions {
			if p.model.Sessions[i].ID == msg.id {
				p.model.Sessions[i].Name = msg.name
				break
			}
		}
		return p, nil

	case noteMsg:
		p.model.Entries = append(p.model.Entries, Entry{Kind: KindNote, Summary: string(msg)})
		return p, nil

	case switchedMsg:
		// A new session means a new event log, so the view is rebuilt rather
		// than carried over: keeping entries from the old one would show a
		// history the model no longer has.
		p.opts.SessionID = msg.session.ID
		p.model = NewModel(msg.session.ID, msg.session.Workspace, msg.session.Model, msg.session.SandboxMode, p.opts.Lang)
		// The recorded conversations do not belong to the session that was
		// replaced: they are the workspace's, and the sidebar would otherwise
		// empty itself the first time somebody ran /clear.
		p.model.Sessions = p.opts.Sessions
		p.attach(msg.session.ID)
		return p, p.waitForEvent()

	case eventMsg:
		p.model.Now = p.now()
		p.model = p.model.Apply(protocol.Event(msg))
		// The queue drains when the session goes idle: the protocol refuses a
		// concurrent turn, so waiting here is what turns a refusal into a
		// usable experience.
		var cmds []tea.Cmd
		if c := p.resumeTicking(); c != nil {
			cmds = append(cmds, c)
		}
		if p.model.State == protocol.SessionStateIdle && len(p.model.Queue) > 0 {
			m, text := p.model.DrainQueue()
			p.model = m
			cmds = append(cmds, p.submit(text))
		}
		cmds = append(cmds, p.waitForEvent())
		return p, tea.Batch(cmds...)

	case steerLateMsg:
		// The turn ended between the keystroke and the request. Queue it, which
		// is what would have happened had they typed it a moment later.
		if p.model.State == protocol.SessionStateIdle {
			return p, p.submit(msg.text)
		}
		m, _ := p.model.Enqueue(msg.text, p.opts.QueueMax)
		p.model = m
		return p, nil

	case errMsg:
		p.fatal = msg.err.Error()
		return p, tea.Quit

	case streamClosedMsg:
		return p, tea.Quit

	case tea.KeyPressMsg:
		return p.onKey(msg)

	case tea.PasteMsg:
		// A terminal with bracketed paste sends the whole block as one message
		// rather than as a burst of key presses, so without this case a paste
		// fell through and vanished — nothing on screen changed, which reads as
		// a broken terminal rather than a missing case.
		//
		// The modal swallows it like any other input: it is the one moment the
		// user has to read, and a paste is not consent.
		if p.model.Pending != nil {
			return p, nil
		}
		p.model = p.model.Insert(normalisePaste(msg.Content)).Refresh(p.opts.Commands)
		return p, nil
	}
	return p, nil
}

// normalisePaste makes a pasted block's line endings uniform.
//
// The breaks are kept now. They used to become spaces, because the input was
// one row and letting them through would have broken the render — a stack
// trace, a diff or a log pasted as one long line was the lesser damage.
//
// The box holds rows, so the reason is gone and keeping them is plainly
// better: a pasted stack trace reads as a stack trace. Treating each break as
// Enter is still wrong for the same reason as before — it would send one turn
// per line and lose the last — and nothing here does that.
func normalisePaste(s string) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	return strings.ReplaceAll(s, "\r", "\n")
}

func (p *program) onKey(k tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	// The modal blocks input: it is the one moment the user has to read, and
	// letting keystrokes fall through would let them dismiss it by accident.
	if p.model.Pending != nil {
		return p.onApprovalKey(k)
	}

	// Copy mode owns the keyboard while it is open, and nothing else does. A
	// mode that lets other keys through is a mode people leave by accident,
	// halfway through a selection.
	//
	// Ahead of the completion menu, not inside it: this block used to sit under
	// the menu's guard, so it ran only while the menu was open — which is the
	// one moment copy mode cannot be, because opening it needs an empty input
	// line and the menu only appears once something is typed. Every key fell
	// through to the stream bindings underneath instead.
	if p.model.Copy.Active {
		last := len(p.renderedStream()) - 1
		switch k.String() {
		case "up", "k":
			p.model = p.model.ExtendCopy(-1, last)
			return p, nil
		case "down", "j":
			p.model = p.model.ExtendCopy(1, last)
			return p, nil
		case "y", "enter":
			text := CopyText(p.renderedStream(), p.model.Copy)
			t := Text(p.model.Lang)
			if text == "" {
				p.model.Flash = t.CopyEmpty
				p.model = p.model.LeaveCopy()
				return p, nil
			}
			p.model.Flash = t.CopyDone
			p.model = p.model.LeaveCopy()
			// Written straight to the terminal: the clipboard is the
			// terminal's, not the program's, and OSC 52 is what reaches it
			// over ssh and inside tmux.
			return p, tea.Printf("%s", OSC52(text))
		case "esc", "q", "ctrl+c":
			p.model = p.model.LeaveCopy()
			return p, nil
		}
		// Every other key is swallowed: a mode that lets them through is a
		// mode people leave by accident, halfway through a selection.
		return p, nil
	}

	// The rail owns the keyboard while it is open, and it sits HERE — above the
	// completion menu — for the reason the copy-mode changelog records: a block
	// placed inside that guard only ever runs while the menu is open, and the
	// menu only opens once something has been typed. The mode would have been
	// decorative and nothing would have said so.
	if p.model.Nav.Active {
		visible := p.model.Nav.Visible(p.model.Sessions)

		// Naming is its own mode inside the list, because it is the one thing
		// here that CHANGES something. Every key means the name while it is
		// open, so nothing else can be reached by accident halfway through.
		if p.model.Nav.Naming {
			switch k.String() {
			case "enter":
				id, name := p.model.Nav.Chosen(p.model.Sessions), p.model.Nav.Draft
				p.model.Nav.Naming, p.model.Nav.Draft = false, ""
				if id == "" {
					return p, nil
				}
				return p, p.rename(id, name)
			case "esc":
				p.model.Nav = p.model.Nav.Escape()
				return p, nil
			case "backspace":
				p.model.Nav = p.model.Nav.BackspaceName()
				return p, nil
			case "ctrl+c":
				p.model.Nav = RailNav{}
				return p, nil
			}
			if t := k.String(); len(t) == 1 || t == " " {
				p.model.Nav = p.model.Nav.TypeName(t)
			}
			return p, nil
		}

		switch key := k.String(); key {
		case "r", "f2":
			p.model.Nav = p.model.Nav.StartNaming(p.model.Sessions)
			return p, nil
		case "up":
			p.model.Nav = p.model.Nav.Move(-1, len(visible))
			return p, nil
		case "down":
			p.model.Nav = p.model.Nav.Move(1, len(visible))
			return p, nil
		case "enter":
			id := p.model.Nav.Chosen(p.model.Sessions)
			p.model.Nav = RailNav{}
			if id == "" || id == p.model.SessionID {
				// Already here, or a filter that matched nothing. Doing
				// nothing beats reloading the conversation somebody is in.
				return p, nil
			}
			return p, p.resume(id)
		case "esc", "ctrl+r":
			p.model.Nav = p.model.Nav.Escape()
			return p, nil
		case "backspace":
			p.model.Nav = p.model.Nav.Backspace()
			return p, nil
		case "ctrl+c":
			// The way out of everything stays the way out of everything.
			p.model.Nav = RailNav{}
			return p, nil
		}
		// A letter is a filter here, not a shortcut: this is a mode that owns
		// the keyboard, which is the exact case RN-16 leaves room for.
		if len(k.String()) == 1 {
			p.model.Nav = p.model.Nav.Type(k.String(), p.model.Sessions)
			return p, nil
		}
		// Everything else is swallowed. A mode that lets keys through is a mode
		// people leave by accident, halfway through choosing.
		return p, nil
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

	// ---------- breaking a line ----------
	//
	// Enter sends, so there was no way to type a list: every line collapsed
	// into one paragraph.
	//
	// Three bindings for one act, and it is not generosity. Most terminals
	// send the same bytes for enter and shift+enter, so the modifier only
	// survives where the terminal answers Bubble Tea's disambiguation request.
	// alt+enter and ctrl+j need nothing — ctrl+j IS a newline, and has been on
	// every terminal ever made — so the feature works everywhere and reads
	// naturally where it can.
	// Pasting a picture. The terminal cannot deliver one — a paste arrives as
	// bracketed text or as a key press, and an image on the clipboard produces
	// neither — so the key is the signal to go and ask the system itself.
	case "ctrl+v":
		return p.pasteImage()

	case "shift+enter", "alt+enter", "ctrl+j":
		p.model = p.model.Insert("\n")
		return p, nil

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

	case "ctrl+r":
		// Focus the conversation list. Nothing to choose from means nothing to
		// focus: a mode that opens onto an empty list is a mode that swallows
		// the next keystroke for no reason.
		if len(p.model.Sessions) == 0 {
			return p, nil
		}
		// The list is an overlay, so the file column is not disturbed by
		// asking for it. Opening a twenty-six-column sidebar to choose a
		// conversation was the borrowed key contradicting what it borrowed:
		// `^R` in readline summons a search, it does not pin a panel.
		p.model.Nav = RailNav{Active: true}
		return p, nil

	case "ctrl+b":
		// The sidebar toggle, and ^B because that is what VS Code made the word
		// "sidebar" mean. A control key rather than a letter, for the reason the
		// rest of this switch is written around: `b` on an empty line would eat
		// the first character of "bom", "build", "bash".
		//
		// An explicit choice wins at any width, both ways — the same manners as
		// the panel, because answering one question two ways would give the two
		// columns different behaviour on the same terminal.
		if p.geo.ShowRail(p.model.railHasContent()) {
			p.geo.RailMode = RailHidden
		} else {
			p.geo.RailMode = RailShown
		}
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
		// A row above is a row to move to, and that beats every other reading:
		// somebody editing the second line of a list is not reaching for what
		// they typed yesterday.
		if at := LineUp(p.model.Input, p.model.InputCursor); at >= 0 {
			p.model.InputCursor = at
			return p, nil
		}
		if p.model.Input == "" && len(p.model.History) > 0 && p.model.Cursor < 0 {
			p.model = p.model.HistoryPrev()
			return p, nil
		}
		return p.moveCursor(-1)
	case "down":
		if at := LineDown(p.model.Input, p.model.InputCursor); at >= 0 {
			p.model.InputCursor = at
			return p, nil
		}
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
	// home and end stay on the stream, deliberately. The rule a few cases up
	// holds: a key that sometimes scrolls and sometimes types is a key nobody
	// trusts. ctrl+a and ctrl+e are the line keys, and they now mean THIS
	// line rather than the whole buffer.
	case "ctrl+a":
		p.model.InputCursor = LineStart(p.model.Input, p.model.InputCursor)
		return p, nil
	case "ctrl+e":
		p.model.InputCursor = LineEnd(p.model.Input, p.model.InputCursor)
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

	case "ctrl+o":
		// The debt RN-1 took on: the alternate screen costs the terminal's own
		// selection, and this is it given back.
		//
		// A CHORD, and it was `v` twice. The first time, a plain `v` on an empty
		// line ate the first character of anything starting with one — "voce"
		// arrived as "oce". That was fixed by requiring the stream cursor to be
		// in the stream, which NARROWED the rule instead of applying it, so the
		// same report came back: `↑` on a session with no history walks into
		// the stream, and the next `v` typed there is a shortcut again.
		//
		// RN-16 says a shortcut on a line where you type is a control key. The
		// input line is ALWAYS a line where you type — typing while browsing
		// inserts — so the rule was never satisfied by a condition, only by
		// giving the letter back.
		//
		// ^O because it is the one chord in this neighbourhood with no meaning
		// to fight: ^Y is yank, ^S is XOFF, ^L clears, ^A/^E/^F/^B/^N are
		// readline motions, and ^V is already the picture paste.
		p.model = p.model.EnterCopy(len(p.renderedStream()) - 1)
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

	// The text the keypress produced, which is the terminal's answer to "what
	// did the person type". It was `len(k.String()) == 1` before, and len on a
	// string counts BYTES: every accented character is two or three of them in
	// UTF-8, so "á", "ç" and "ã" all failed that test and were dropped in
	// silence. Someone typing Portuguese watched "não" arrive as "no", which
	// reverses it, with nothing anywhere reporting a lost letter.
	//
	// Taking the text also means a paste arrives whole rather than as a run of
	// single characters the old branch would have rejected for being longer
	// than one byte.
	if text := k.Text; text != "" {
		// `?` opens help only as the first character of an empty line. It is
		// the convention every pager and monitor already uses, and a message
		// that opens with a question mark is rare enough to be worth the trade
		// — unlike a letter, which is not.
		if text == "?" && p.model.Input == "" {
			return p.runBuiltin(Resolved{Kind: CmdBuiltin, Name: "help"})
		}
		// Typing returns the focus to the line being typed on.
		//
		// Browsing the stream and writing a message were two states at once,
		// and nothing on screen said which one you were in: `↑` walks into the
		// stream, and from there a character still went into the input line
		// while the stream kept the cursor. The state that produced this
		// report — half in one place, half in the other — cannot be reached
		// now, because typing is what ends it.
		p.model.Cursor = -1
		p.model = p.model.Insert(text).Refresh(p.opts.Commands)
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

	// A queued line is echoed here, because no turn will announce it until it
	// leaves the queue and there is nothing else to show the user meanwhile.
	// A line that starts a turn is NOT echoed: turn.started carries it, and
	// echoing as well would show it twice.
	if p.model.State != protocol.SessionStateIdle {
		// Mid-turn, ordinary words steer: they reach the running turn at its
		// next round instead of waiting for it to end. Watching a turn go wrong
		// with two options — let it finish, or kill it and lose what it learned
		// — is the friction this removes.
		//
		// A picture still waits. An image is about a question and the steering
		// path carries text only; sending half the pair now would ask the model
		// about something it cannot see.
		if len(p.model.Attached) == 0 {
			// Not echoed here: turn.steered carries it, and echoing as well
			// would show it twice — the same rule turn.started already follows.
			return p, p.steer(r.Text)
		}
		p.model.Entries = append(p.model.Entries, Entry{Kind: KindUser, Summary: line})
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
		return note(HelpText(p.opts.Commands, p.model.Lang))

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

	case "image":
		return p.attachImage(strings.TrimSpace(r.Args))

	case "undo":
		// Not a tool. The model does not undo its own work, because the
		// judgment undo exists for is the person's.
		return p, p.undo()

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

// sendOrQueue is for built-ins, which cost a whole turn.
//
// They queue rather than steer, and the difference is the point: `/init` mid-turn
// is not a correction to the work in flight, it is a second job. Steering carries
// what the person wants CHANGED about what is happening; a built-in carries
// something else to do next.
func (p *program) sendOrQueue(text string) (tea.Model, tea.Cmd) {
	if p.model.State != protocol.SessionStateIdle {
		m, _ := p.model.Enqueue(text, p.opts.QueueMax)
		p.model = m
		return p, nil
	}
	return p, p.submit(text)
}

// steer sends a correction to the running turn, and falls back to the queue
// when the turn ended in the meantime.
//
// The race is real and unavoidable: the turn can finish between the keystroke
// and the request. Dropping the message there would lose the most deliberate
// thing a person does during a turn, so a refusal becomes the queue — which is
// what would have happened had they typed it a moment later.
func (p *program) steer(text string) tea.Cmd {
	return func() tea.Msg {
		err := p.opts.Transport.Steer(p.ctx, p.opts.SessionID, text)
		if err == nil {
			return nil
		}
		var perr *protocol.Error
		if errors.As(err, &perr) && perr.Code == protocol.CodeNoActiveTurn {
			return steerLateMsg{text}
		}
		return errMsg{err}
	}
}

// steerLateMsg is a correction that arrived after its turn ended.
type steerLateMsg struct{ text string }

func (p *program) onApprovalKey(k tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	id := p.model.Pending.ApprovalID
	switch k.String() {
	case "a":
		return p, p.resolve(id, protocol.ApprovalAllow)
	case "A", "shift+a":
		return p, p.resolve(id, protocol.ApprovalAllowSession)
	case "P", "shift+p":
		// Written down, and only offered where an answer can be. The two
		// standing options take a capital: harder to press by accident, which
		// is what the largest consequence deserves.
		if standingScope(*p.model.Pending) {
			return p, p.resolve(id, protocol.ApprovalAllowProject)
		}
	case "G", "shift+g":
		if standingScope(*p.model.Pending) {
			return p, p.resolve(id, protocol.ApprovalAllowAlways)
		}
	case "d", "enter", "esc":
		// Enter denies. The safe action is the default and the least effort.
		return p, p.resolve(id, protocol.ApprovalDeny)
	case "ctrl+c":
		return p, p.resolve(id, protocol.ApprovalDeny)
	}
	return p, nil
}

func (p *program) submit(text string) tea.Cmd {
	images := p.model.Attached
	p.model.Attached = nil
	return func() tea.Msg {
		if err := p.opts.Transport.Submit(p.ctx, p.opts.SessionID, text, images...); err != nil {
			return errMsg{err}
		}
		return nil
	}
}

// attachImage holds a picture for the next question.
//
// Attached rather than sent on its own: a picture with no question is a turn
// the model has to guess the point of, and "what am I looking at" is rarely
// what the person meant.
//
// The refusal when the model cannot read pictures happens HERE, before
// anything is sent. dcode speaks to several providers and this is a capability
// only some have — a provider error thirty seconds later is a failure the
// person cannot connect to what they did.
func (p *program) attachImage(path string) (tea.Model, tea.Cmd) {
	t := Text(p.model.Lang)
	note := func(s string) (tea.Model, tea.Cmd) {
		p.model.Entries = append(p.model.Entries, Entry{Kind: KindNote, Summary: s, Expanded: true})
		return p, nil
	}
	if path == "" {
		return note(t.ImageUsage)
	}
	if !p.opts.AcceptsImages {
		return note(fmt.Sprintf(t.ImageUnsupported, p.model.Model))
	}
	img, err := session.ReadImage(expandHome(path), MaxImageBytes)
	if err != nil {
		return note(t.ImageFailed + " " + err.Error())
	}
	p.model.Attached = append(p.model.Attached, protocol.TurnImage{
		MediaType: img.MediaType,
		Data:      base64.StdEncoding.EncodeToString(img.Data),
	})
	return note(fmt.Sprintf(t.ImageAttached, path, len(p.model.Attached)))
}

// pasteImage attaches whatever picture is on the clipboard.
//
// Every outcome is said out loud, because the alternative is a key that
// sometimes does nothing for three different reasons and looks identical each
// time: an empty clipboard, a machine with no way to read it, and a model that
// cannot see pictures are three different things to do next.
func (p *program) pasteImage() (tea.Model, tea.Cmd) {
	t := Text(p.model.Lang)
	note := func(s string) (tea.Model, tea.Cmd) {
		p.model.Entries = append(p.model.Entries, Entry{Kind: KindNote, Summary: s, Expanded: true})
		return p, nil
	}
	if !p.opts.AcceptsImages {
		return note(fmt.Sprintf(t.ImageUnsupported, p.model.Model))
	}

	body, media, err := ClipboardImage()
	switch {
	case errors.Is(err, ErrNoImageInClipboard):
		return note(t.ClipboardEmpty)
	case errors.Is(err, ErrNoClipboardTool):
		return note(t.ClipboardMissing)
	case err != nil:
		return note(t.ImageFailed + " " + err.Error())
	}
	if len(body) > MaxImageBytes {
		return note(fmt.Sprintf(t.ImageTooBig, len(body)/(1<<20), MaxImageBytes/(1<<20)))
	}

	p.model.Attached = append(p.model.Attached, protocol.TurnImage{
		MediaType: media,
		Data:      base64.StdEncoding.EncodeToString(body),
	})
	return note(fmt.Sprintf(t.ImagePasted, len(p.model.Attached)))
}

// expandHome turns a leading ~ into the home directory, because a screenshot
// path is typed by a person and that is how a person writes it.
func expandHome(path string) string {
	if !strings.HasPrefix(path, "~/") {
		return path
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}
	return filepath.Join(home, path[2:])
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

// rename names a conversation and puts the answer on screen.
//
// The list is updated from what was SENT rather than re-read: the daemon
// answers with nothing, and going back for the whole listing to learn one
// string is a round trip for a fact already in hand. A failure says so and
// leaves the row as it was — a name that did not stick must not look like it
// did.
func (p *program) rename(id, name string) tea.Cmd {
	return func() tea.Msg {
		if err := p.opts.Transport.RenameSession(p.ctx, id, name); err != nil {
			return errMsg{err: err}
		}
		return renamedMsg{id: id, name: name}
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

// renderedStream is the stream as lines, undecorated enough to be copied.
//
// Rendered rather than stored: what a person wants to copy is what they can
// see, including the wrapping, and reconstructing that from the entries would
// be a second renderer to keep in step with the first.
func (p *program) renderedStream() []string {
	g := p.geo
	g.Width = max(20, g.Width)
	return renderStream(p.model, g, g.Width)
}

// undo asks the session to put the last turn back and says what happened.
//
// Both halves are reported. A silent undo leaves the person guessing which of
// seven files went back, and the one that did NOT is the one they most need to
// know about — it did not go back because they had changed it themselves.
func (p *program) undo() tea.Cmd {
	return func() tea.Msg {
		out, err := p.opts.Transport.Undo(p.ctx, p.opts.SessionID)
		if err != nil {
			return noteMsg(Text(p.model.Lang).UndoFailed + " " + err.Error())
		}
		t := Text(p.model.Lang)
		switch {
		case len(out.Restored) == 0 && len(out.Refused) == 0:
			return noteMsg(t.UndoNothing)
		case len(out.Refused) == 0:
			return noteMsg(fmt.Sprintf("%s\n  %s", t.UndoRestored, strings.Join(out.Restored, "\n  ")))
		default:
			body := t.UndoRestored + "\n  " + strings.Join(out.Restored, "\n  ")
			if len(out.Restored) == 0 {
				body = ""
			}
			return noteMsg(strings.TrimSpace(body + "\n\n" + t.UndoRefused + "\n  " +
				strings.Join(out.Refused, "\n  ")))
		}
	}
}
