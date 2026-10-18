package tui

import (
	"context"
	"fmt"

	tea "charm.land/bubbletea/v2"

	"github.com/aguinelo/dcode/internal/protocol"
	"github.com/aguinelo/dcode/pkg/client"
)

// What `dcode orchestrate` adds to the client (orchestration .i step 13,
// client-tui .p §2.2): the live list read from the daemon when the
// orchestrator opens, ^Q answering the oldest question it waits on, and ^R
// attaching to one of its children — where a child's approval is answered.
//
// Spec: docs/specs/architecture/orchestration/202610132000-orchestration.p.spec.md §5.1, §11.

// Orchestration is what the client reads and answers under an orchestrator:
// three of the daemon's orchestration routes. Its own interface rather than
// more of Transport, because only `dcode orchestrate` hands one in, and an
// ordinary conversation has no use for any of it.
type Orchestration interface {
	Children(ctx context.Context, sessionID string) ([]protocol.Child, error)
	Questions(ctx context.Context, sessionID string) ([]protocol.Question, error)
	AnswerQuestion(ctx context.Context, sessionID, questionID string, req protocol.AnswerQuestionRequest) (protocol.Question, error)
}

var _ Orchestration = (*client.Client)(nil)

// orchFollow is what the program keeps about the orchestrator it was opened
// on: its id, which outlives attaching to a child, and whether its list is due
// a read from the daemon.
type orchFollow struct {
	id  string
	due bool
}

// agentsLoadedMsg is the daemon's answer to /children and /questions.
type agentsLoadedMsg struct {
	session   string
	children  []protocol.Child
	questions []protocol.Question
	err       error
}

// answeredMsg is the daemon's answer to one POST of /questions.
type answeredMsg struct {
	question protocol.Question
	answer   string
	err      error
}

// AskView is the question ^Q opened, and where the cursor is among its
// answers. It opens on the recommended one, so Enter takes the advice and any
// other answer is a deliberate move.
type AskView struct {
	Question protocol.Question
	Cursor   int
	// Sending is true while the answer is in flight: a second Enter would be
	// a second POST, and the daemon would refuse it as already answered.
	Sending bool
}

// AgentPick is the list ^R opens under an orchestrator: its children, and the
// orchestrator itself when the client is attached to one of them.
type AgentPick struct {
	Active bool
	Cursor int
	// Rows are the list as it was when ^R opened it: what Enter chooses from
	// is what is on screen, even when a child changes state underneath.
	Rows []pickRow
}

// pickRow is one row of that list.
type pickRow struct {
	id, title, state string
	orchestrator     bool
}

// orchestrating reports whether this client was opened on an orchestrator,
// and takes that session as the orchestrator the first time it is asked.
func (p *program) orchestrating() bool {
	p.startOrchestrating()
	return p.opts.Orchestration != nil
}

// startOrchestrating takes the session the client opened on as the
// orchestrator, once, and marks its list as due. Once: after ^R the client is
// on a child, and the orchestrator is still the session it opened on.
func (p *program) startOrchestrating() {
	if p.opts.Orchestration != nil && p.orch.id == "" {
		p.orch = orchFollow{id: p.opts.SessionID, due: true}
	}
}

// loadAgentsIfDue reads the list when it is due, the client is on the
// orchestrator, and the backlog is in. Not before the backlog: the replay
// folds every child.* the log ever had, and a read answered in the middle of
// it would have rows the daemon has since let go put back by the events after
// it.
func (p *program) loadAgentsIfDue() tea.Cmd {
	p.startOrchestrating()
	if !p.orch.due || p.opts.SessionID != p.orch.id || p.catchingUp() {
		return nil
	}
	p.orch.due = false
	o, id := p.opts.Orchestration, p.orch.id
	return func() tea.Msg {
		children, err := o.Children(p.ctx, id)
		if err != nil {
			return agentsLoadedMsg{session: id, err: err}
		}
		questions, err := o.Questions(p.ctx, id)
		return agentsLoadedMsg{session: id, children: children, questions: questions, err: err}
	}
}

// onAgentsLoaded takes the answer as the list, or says why there is none.
func (p *program) onAgentsLoaded(msg agentsLoadedMsg) {
	if msg.err != nil {
		p.note(fmt.Sprintf(Text(p.model.Lang).AgentsLoadFailed, msg.err))
		return
	}
	if msg.session != p.opts.SessionID {
		// Read for the orchestrator after the client had moved to a child; the
		// next return to it reads again.
		p.orch.due = true
		return
	}
	p.model = p.model.loadAgents(msg.children, msg.questions)
	p.syncAsk()
}

// carryAgents keeps the orchestrator's list across a switch: from a child,
// ^R still lists its siblings and ^Q still answers the orchestrator's
// questions. Back on the orchestrator, the list is read again.
func (p *program) carryAgents(old Agents, to string) {
	if !p.orchestrating() {
		return
	}
	p.model.Agents = old
	if to == p.orch.id {
		p.orch.due = true
	}
}

func (p *program) note(text string) {
	p.model.Entries = append(p.model.Entries, Entry{Kind: KindNote, Summary: text, Expanded: true})
}

// clearOrchFlash takes down a flash of this file's at the next key: it said
// why a key did nothing, and the next key is a new attempt.
func (p *program) clearOrchFlash() {
	t := Text(p.model.Lang)
	switch p.model.Flash {
	case t.AskNone, t.AskNotHere, t.PickNone:
		p.model.Flash = ""
	}
}

// ---------- ^Q ----------

// openAsk opens the oldest question waiting. Nothing waiting is said, as is a
// conversation with no orchestrator to answer: a key that does nothing and
// says nothing reads as a key that is broken.
func (p *program) openAsk() (tea.Model, tea.Cmd) {
	t := Text(p.model.Lang)
	if !p.orchestrating() {
		p.model.Flash = t.AskNotHere
		return p, nil
	}
	if len(p.model.Agents.Questions) == 0 {
		p.model.Flash = t.AskNone
		return p, nil
	}
	q := p.model.Agents.Questions[0]
	cursor := 0
	for i, o := range q.Options {
		if o.ID == q.Recommended {
			cursor = i
		}
	}
	p.model.Ask = &AskView{Question: q, Cursor: cursor}
	return p, nil
}

// onAskKey owns the keyboard while a question is open. Every other key is
// swallowed: a mode that lets keys through is a mode people leave by accident,
// halfway through answering.
func (p *program) onAskKey(k tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	a := p.model.Ask
	switch k.String() {
	case "up":
		if a.Cursor > 0 {
			a.Cursor--
		}
	case "down":
		if a.Cursor < len(a.Question.Options)-1 {
			a.Cursor++
		}
	case "esc", "ctrl+c", "ctrl+q":
		p.model.Ask = nil
	case "enter":
		if a.Sending || a.Cursor >= len(a.Question.Options) {
			return p, nil
		}
		a.Sending = true
		return p, p.answer(a.Question, a.Question.Options[a.Cursor].ID)
	}
	return p, nil
}

func (p *program) answer(q protocol.Question, answer string) tea.Cmd {
	o, id := p.opts.Orchestration, p.orch.id
	return func() tea.Msg {
		got, err := o.AnswerQuestion(p.ctx, id, q.ID, protocol.AnswerQuestionRequest{Answer: answer})
		if err == nil {
			q = got
		}
		return answeredMsg{question: q, answer: answer, err: err}
	}
}

// onAnswered says what came of the answer. The first answer wins: one that
// lost to an answer given elsewhere is said, the question leaves the list,
// and the list is read again for what the daemon holds now.
func (p *program) onAnswered(msg answeredMsg) tea.Cmd {
	t := Text(p.model.Lang)
	pe, typed := protocol.AsError(msg.err)
	switch {
	case msg.err == nil:
		p.model.Ask = nil
		p.dropQuestion(msg.question.ID)
		p.note(fmt.Sprintf(t.AskAnswered, msg.question.Text, answerLabel(msg.question, msg.answer, t)))
		return nil
	case typed && (pe.Code == protocol.CodeQuestionAlreadyAnswered || pe.Code == protocol.CodeQuestionNotFound):
		p.model.Ask = nil
		p.dropQuestion(msg.question.ID)
		p.note(t.AskAlreadyAnswered)
		p.orch.due = true
		return p.loadAgentsIfDue()
	}
	// Any other failure leaves the question where it was, open, for another
	// try: nothing was answered, and nothing may look as if it had been.
	if p.model.Ask != nil {
		p.model.Ask.Sending = false
	}
	p.note(fmt.Sprintf(t.AskFailed, msg.err))
	return nil
}

func (p *program) dropQuestion(id string) {
	qs := make([]protocol.Question, 0, len(p.model.Agents.Questions))
	for _, q := range p.model.Agents.Questions {
		if q.ID != id {
			qs = append(qs, q)
		}
	}
	p.model.Agents.Questions = qs
}

// syncAsk closes the open question when the list no longer has it — answered
// from the desktop, or by another terminal — and says so, rather than leave a
// question on screen that a POST would only be refused for.
func (p *program) syncAsk() {
	a := p.model.Ask
	if a == nil || a.Sending {
		return
	}
	for _, q := range p.model.Agents.Questions {
		if q.ID == a.Question.ID {
			return
		}
	}
	p.model.Ask = nil
	p.note(Text(p.model.Lang).AskAlreadyAnswered)
}

// answerLabel is an answer in the person's language. The daemon's labels are
// English (the wire's language); the answers of the kinds it asks — merge
// and ceiling — are this client's to translate, and any other is shown as the
// daemon wrote it.
func answerLabel(q protocol.Question, id string, t Strings) string {
	switch id {
	case "merge":
		return t.AnswerMerge
	case "return":
		return t.AnswerReturn
	case "close":
		return t.AnswerClose
	case "continue":
		return t.AnswerContinue
	case "stop":
		return t.AnswerStop
	}
	for _, o := range q.Options {
		if o.ID == id && o.Label != "" {
			return o.Label
		}
	}
	return id
}

// ---------- ^R ----------

// pickRows is the list ^R opens: the orchestrator first when the client is on
// a child, then every child the list holds.
func (p *program) pickRows() []pickRow {
	t := Text(p.model.Lang)
	var rows []pickRow
	if p.opts.SessionID != p.orch.id {
		rows = append(rows, pickRow{id: p.orch.id, title: t.PickOrchestrator, orchestrator: true})
	}
	for _, c := range p.model.Agents.Children {
		title := c.Title
		if title == "" {
			title = t.ChildUnnamed
		}
		rows = append(rows, pickRow{id: c.ID, title: title, state: c.State})
	}
	return rows
}

func (p *program) openPick() (tea.Model, tea.Cmd) {
	rows := p.pickRows()
	if len(rows) == 0 {
		// An empty list does not take the keyboard, and says why it did not.
		p.model.Flash = Text(p.model.Lang).PickNone
		return p, nil
	}
	p.model.Pick = AgentPick{Active: true, Rows: rows}
	return p, nil
}

func (p *program) onPickKey(k tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	rows := p.model.Pick.Rows
	switch k.String() {
	case "up":
		if p.model.Pick.Cursor > 0 {
			p.model.Pick.Cursor--
		}
	case "down":
		if p.model.Pick.Cursor < len(rows)-1 {
			p.model.Pick.Cursor++
		}
	case "esc", "ctrl+c", "ctrl+r":
		p.model.Pick = AgentPick{}
	case "enter":
		at := p.model.Pick.Cursor
		p.model.Pick = AgentPick{}
		if at >= len(rows) || rows[at].id == p.opts.SessionID {
			return p, nil
		}
		return p, p.attachTo(rows[at].id)
	}
	return p, nil
}

// attachTo joins a session the daemon holds, and nothing else. A child is a
// live session or it is gone; a gone one is said, never continued into a new
// session by guess — that would be a second writer in a worktree the harness
// owns.
func (p *program) attachTo(id string) tea.Cmd {
	t := Text(p.model.Lang)
	return func() tea.Msg {
		s, err := p.opts.Transport.GetSession(p.ctx, id)
		if err != nil {
			return noteMsg(fmt.Sprintf(t.PickOpenFailed, id, err))
		}
		return switchedMsg{session: s}
	}
}
