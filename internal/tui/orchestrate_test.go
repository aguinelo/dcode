package tui

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/aguinelo/dcode/internal/protocol"
)

// `dcode orchestrate` and its two keys (orchestration .i step 13): the list
// read from /children and /questions when the orchestrator opens, ^Q answering
// the oldest question through the POST, and ^R attaching to a child.

// fakeOrchestration stands in for the daemon's orchestration routes.
type fakeOrchestration struct {
	mu        sync.Mutex
	children  []protocol.Child
	questions []protocol.Question
	answerErr error
	reads     []string
	answers   [][3]string // session, question, answer
}

func (f *fakeOrchestration) Children(_ context.Context, id string) ([]protocol.Child, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.reads = append(f.reads, "children:"+id)
	return append([]protocol.Child(nil), f.children...), nil
}

func (f *fakeOrchestration) Questions(_ context.Context, id string) ([]protocol.Question, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.reads = append(f.reads, "questions:"+id)
	return append([]protocol.Question(nil), f.questions...), nil
}

func (f *fakeOrchestration) AnswerQuestion(_ context.Context, id, qid string, req protocol.AnswerQuestionRequest) (protocol.Question, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.answers = append(f.answers, [3]string{id, qid, req.Answer})
	if f.answerErr != nil {
		return protocol.Question{}, f.answerErr
	}
	for _, q := range f.questions {
		if q.ID == qid {
			now := time.Now()
			q.Answer, q.Answered = req.Answer, &now
			return q, nil
		}
	}
	return protocol.Question{}, protocol.Errorf(protocol.CodeQuestionNotFound, "no question %s", qid)
}

func (f *fakeOrchestration) answered() [][3]string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([][3]string(nil), f.answers...)
}

func mergeQuestion(id, child string, asked time.Time) protocol.Question {
	return protocol.Question{
		ID: id, Kind: "merge", Text: "merge " + child + "?", Recommended: "merge", Why: "the gate is green",
		ChildID: child, Asked: asked,
		Options: []protocol.QuestionOption{
			{ID: "merge", Label: "Merge"}, {ID: "return", Label: "Return to the child"},
			{ID: "close", Label: "Close the child"},
		},
	}
}

func ceilingQuestion(id string, asked time.Time) protocol.Question {
	return protocol.Question{
		ID: id, Kind: "ceiling", Text: "the token ceiling was reached", Recommended: "stop", Why: "spent",
		Asked:   asked,
		Options: []protocol.QuestionOption{{ID: "continue", Label: "Continue"}, {ID: "stop", Label: "Stop"}},
	}
}

// orchestrating is a program opened by `dcode orchestrate` on the orchestrator
// "orq", in lang.
func orchestrating(t *testing.T, lang Lang, f *fakeOrchestration) (*program, *fakeTransport) {
	t.Helper()
	p, tr := newProgram(t, func(o *Options) {
		o.SessionID, o.Lang, o.Orchestration = "orq", lang, f
		o.Geometry = agentsGeometry()
	})
	p.model = apply(t, p.model, ev(t, 1, protocol.EventSessionCreated, protocol.Session{
		ID: "orq", Workspace: "/w", Role: protocol.RoleOrchestrator,
	}))
	return p, tr
}

// messagesOf runs a command — every command of a batch, each on its own,
// since the stream reader blocks until the test ends — and returns the
// messages that came back within a moment.
func messagesOf(t *testing.T, cmd tea.Cmd) []tea.Msg {
	t.Helper()
	if cmd == nil {
		return nil
	}
	out := make(chan tea.Msg, 16)
	var spawn func(c tea.Cmd)
	spawn = func(c tea.Cmd) {
		go func() {
			m := c()
			if b, ok := m.(tea.BatchMsg); ok {
				for _, c := range b {
					if c != nil {
						spawn(c)
					}
				}
				return
			}
			out <- m
		}()
	}
	spawn(cmd)
	var msgs []tea.Msg
	deadline := time.After(300 * time.Millisecond)
	for {
		select {
		case m := <-out:
			msgs = append(msgs, m)
		case <-deadline:
			return msgs
		}
	}
}

// loads applies the list's answer if the messages carry one.
func loads(p *program, msgs []tea.Msg) bool {
	for _, m := range msgs {
		if _, ok := m.(agentsLoadedMsg); ok {
			p.Update(m)
			return true
		}
	}
	return false
}

// screen is the whole frame, wide enough for the side column to be drawn.
func screen(p *program) string {
	g := DefaultGeometry(140, 40)
	g.Palette, g.Unicode = Palette{}, true
	return Render(p.model, g)
}

func TestOpeningTheOrchestratorReadsChildrenAndQuestionsIntoTheList(t *testing.T) {
	f := &fakeOrchestration{
		children: []protocol.Child{orchChild("c1", "índice", "working", 41200, 3100, 12000)},
		questions: []protocol.Question{
			mergeQuestion("q1", "c1", time.Unix(100, 0)),
		},
	}
	p, _ := orchestrating(t, PtBR, f)
	if !loads(p, messagesOf(t, p.Init())) {
		t.Fatalf("opening the orchestrator never read its list; reads: %v", f.reads)
	}
	if len(p.model.Agents.Children) != 1 || p.model.Agents.Children[0].ID != "c1" {
		t.Fatalf("the list is not the daemon's /children: %+v", p.model.Agents.Children)
	}
	if len(p.model.Agents.Questions) != 1 {
		t.Fatalf("the list is not the daemon's /questions: %+v", p.model.Agents.Questions)
	}
	s := screen(p)
	for _, want := range []string{"índice", "trabalhando", Text(PtBR).AgentsQuestionOne} {
		if !strings.Contains(s, want) {
			t.Errorf("the column does not show %q:\n%s", want, s)
		}
	}
}

func TestTheListIsReadOnlyAfterTheBacklogSoReplayCannotResurrectAChild(t *testing.T) {
	f := &fakeOrchestration{}
	p, tr := orchestrating(t, En, f)
	p.opts.Backlog = 3
	if loads(p, messagesOf(t, p.Init())) {
		t.Fatal("the list was read while the backlog was still arriving")
	}
	_ = tr
	// The last event of the backlog arrives: now it is read.
	_, cmd := p.Update(eventMsg{ev: started(t, 3, orchChild("old", "gone", "closed", 0, 0, 0))})
	loaded := loads(p, messagesOf(t, cmd))
	if !loaded {
		t.Fatal("the list was never read once the backlog was in")
	}
	if len(p.model.Agents.Children) != 0 {
		t.Fatalf("a child the daemon no longer lists came back from the replay: %+v", p.model.Agents.Children)
	}
}

func loadedWith(t *testing.T, p *program, f *fakeOrchestration) {
	t.Helper()
	p.Update(agentsLoadedMsg{session: "orq", children: f.children, questions: f.questions})
}

func TestCtrlQAnswersTheOldestQuestionThroughThePost(t *testing.T) {
	f := &fakeOrchestration{questions: []protocol.Question{
		ceilingQuestion("q-new", time.Unix(200, 0)),
		mergeQuestion("q-old", "c1", time.Unix(100, 0)),
	}}
	p, _ := orchestrating(t, En, f)
	loadedWith(t, p, f)

	p.Update(ctrl('q'))
	if p.model.Ask == nil || p.model.Ask.Question.ID != "q-old" {
		t.Fatalf("^Q did not open the oldest question: %+v", p.model.Ask)
	}
	s := screen(p)
	tx := Text(En)
	for _, want := range []string{"merge c1?", "the gate is green", tx.AnswerMerge, tx.AnswerReturn, tx.AnswerClose, tx.AskRecommended} {
		if !strings.Contains(s, want) {
			t.Errorf("the question does not show %q:\n%s", want, s)
		}
	}

	// The cursor opens on the recommended answer; one down is "return".
	p.Update(special(tea.KeyDown))
	_, cmd := p.Update(special(tea.KeyEnter))
	if cmd == nil {
		t.Fatal("enter sent nothing")
	}
	p.Update(one(t, cmd))

	got := f.answered()
	if len(got) != 1 || got[0] != [3]string{"orq", "q-old", "return"} {
		t.Fatalf("the answer was not POSTed to the orchestrator's question: %v", got)
	}
	if p.model.Ask != nil {
		t.Fatal("the question stayed open after it was answered")
	}
	if len(p.model.Agents.Questions) != 1 || p.model.Agents.Questions[0].ID != "q-new" {
		t.Fatalf("the answered question is still counted: %+v", p.model.Agents.Questions)
	}
}

func TestEnterOnAFreshQuestionAnswersTheRecommendedOne(t *testing.T) {
	f := &fakeOrchestration{questions: []protocol.Question{ceilingQuestion("q1", time.Unix(1, 0))}}
	p, _ := orchestrating(t, En, f)
	loadedWith(t, p, f)
	p.Update(ctrl('q'))
	_, cmd := p.Update(special(tea.KeyEnter))
	p.Update(one(t, cmd))
	if got := f.answered(); len(got) != 1 || got[0][2] != "stop" {
		t.Fatalf("enter did not answer the recommended option: %v", got)
	}
}

func TestASecondAnswerIsSaidAsAlreadyAnswered(t *testing.T) {
	f := &fakeOrchestration{
		questions: []protocol.Question{mergeQuestion("q1", "c1", time.Unix(1, 0))},
		answerErr: protocol.Errorf(protocol.CodeQuestionAlreadyAnswered, "q1 was answered merge"),
	}
	p, _ := orchestrating(t, PtBR, f)
	loadedWith(t, p, f)
	p.Update(ctrl('q'))
	_, cmd := p.Update(special(tea.KeyEnter))
	p.Update(one(t, cmd))

	if p.model.Ask != nil {
		t.Fatal("a question somebody else answered stayed open")
	}
	if len(p.model.Agents.Questions) != 0 {
		t.Fatalf("a question somebody else answered is still counted: %+v", p.model.Agents.Questions)
	}
	if !strings.Contains(screen(p), Text(PtBR).AskAlreadyAnswered) {
		t.Fatalf("question_already_answered was not said:\n%s", screen(p))
	}
}

func TestAFailedAnswerIsSaidAndTheQuestionStays(t *testing.T) {
	f := &fakeOrchestration{
		questions: []protocol.Question{mergeQuestion("q1", "c1", time.Unix(1, 0))},
		answerErr: protocol.Errorf(protocol.CodeInternal, "disk full"),
	}
	p, _ := orchestrating(t, En, f)
	loadedWith(t, p, f)
	p.Update(ctrl('q'))
	_, cmd := p.Update(special(tea.KeyEnter))
	p.Update(one(t, cmd))
	if len(p.model.Agents.Questions) != 1 {
		t.Fatal("a question whose answer failed left the list as if answered")
	}
	if !strings.Contains(screen(p), "disk full") {
		t.Fatalf("the failure was not said:\n%s", screen(p))
	}
}

func TestCtrlQWithNothingWaitingSaysSo(t *testing.T) {
	p, _ := orchestrating(t, En, &fakeOrchestration{})
	p.Update(ctrl('q'))
	if p.model.Ask != nil {
		t.Fatal("^Q opened a question that does not exist")
	}
	if p.model.Flash != Text(En).AskNone {
		t.Fatalf("^Q with nothing waiting said %q", p.model.Flash)
	}
}

func TestCtrlQOutsideAnOrchestratorSaysWhereItWorks(t *testing.T) {
	p, _ := newProgram(t)
	p.Update(ctrl('q'))
	if p.model.Ask != nil || p.model.Flash != Text(En).AskNotHere {
		t.Fatalf("^Q in an ordinary conversation: ask %+v, flash %q", p.model.Ask, p.model.Flash)
	}
}

func TestEscClosesTheQuestionWithoutAnswering(t *testing.T) {
	f := &fakeOrchestration{questions: []protocol.Question{ceilingQuestion("q1", time.Unix(1, 0))}}
	p, _ := orchestrating(t, En, f)
	loadedWith(t, p, f)
	p.Update(ctrl('q'))
	p.Update(special(tea.KeyEscape))
	if p.model.Ask != nil || len(f.answered()) != 0 {
		t.Fatalf("esc answered or stayed: %+v %v", p.model.Ask, f.answered())
	}
}

func TestAQuestionAnsweredElsewhereClosesTheOpenOne(t *testing.T) {
	q := mergeQuestion("q1", "c1", time.Unix(1, 0))
	f := &fakeOrchestration{questions: []protocol.Question{q}}
	p, _ := orchestrating(t, En, f)
	loadedWith(t, p, f)
	p.Update(ctrl('q'))
	now := time.Now()
	q.Answer, q.Answered = "merge", &now
	p.Update(eventMsg{ev: ev(t, 2, protocol.EventQuestionAnswered, q)})
	if p.model.Ask != nil {
		t.Fatal("a question the desktop answered is still open here")
	}
	if !strings.Contains(screen(p), Text(En).AskAlreadyAnswered) {
		t.Fatalf("the close was not said:\n%s", screen(p))
	}
}

func TestTheAnswerOptionsSpeakThePersonsLanguage(t *testing.T) {
	f := &fakeOrchestration{questions: []protocol.Question{
		mergeQuestion("q1", "c1", time.Unix(1, 0)), ceilingQuestion("q2", time.Unix(2, 0)),
	}}
	p, _ := orchestrating(t, PtBR, f)
	loadedWith(t, p, f)
	tx := Text(PtBR)
	p.Update(ctrl('q'))
	s := screen(p)
	for _, want := range []string{tx.AnswerMerge, tx.AnswerReturn, tx.AnswerClose} {
		if !strings.Contains(s, want) {
			t.Errorf("the merge question does not offer %q:\n%s", want, s)
		}
	}
	if strings.Contains(s, "Return to the child") {
		t.Errorf("the daemon's English label survived a Portuguese screen:\n%s", s)
	}
	p.Update(special(tea.KeyEscape))
	p.model.Agents.Questions = p.model.Agents.Questions[1:]
	p.Update(ctrl('q'))
	s = screen(p)
	for _, want := range []string{tx.AnswerContinue, tx.AnswerStop} {
		if !strings.Contains(s, want) {
			t.Errorf("the ceiling question does not offer %q:\n%s", want, s)
		}
	}
}

func TestCtrlRUnderTheOrchestratorAttachesToAChildAndBack(t *testing.T) {
	f := &fakeOrchestration{children: []protocol.Child{
		orchChild("c1", "índice", "working", 1, 1, 0),
		orchChild("c2", "api", "awaiting_person", 1, 1, 0),
	}}
	p, tr := orchestrating(t, En, f)
	tr.sessions = []protocol.Session{
		{ID: "orq", Workspace: "/w", Role: protocol.RoleOrchestrator},
		{ID: "c2", Workspace: "/w/children/c2", Role: protocol.RoleChild, ParentID: "orq"},
	}
	loadedWith(t, p, f)

	p.Update(ctrl('r'))
	if !p.model.Pick.Active {
		t.Fatal("^R did not open the list of children")
	}
	s := screen(p)
	for _, want := range []string{"índice", "api", Text(En).AgentAwaiting} {
		if !strings.Contains(s, want) {
			t.Errorf("the list does not show %q:\n%s", want, s)
		}
	}
	p.Update(special(tea.KeyDown))
	_, cmd := p.Update(special(tea.KeyEnter))
	if cmd == nil {
		t.Fatal("choosing a child did nothing")
	}
	p.Update(one(t, cmd))
	if p.opts.SessionID != "c2" {
		t.Fatalf("the client is not attached to the child: %s", p.opts.SessionID)
	}
	if len(tr.created) != 0 {
		t.Fatalf("attaching to a child started a session: %+v", tr.created)
	}

	// From the child, the orchestrator is the first row, and choosing it goes back.
	p.Update(ctrl('r'))
	if !p.model.Pick.Active || !strings.Contains(screen(p), Text(En).PickOrchestrator) {
		t.Fatalf("the child's list does not offer the orchestrator:\n%s", screen(p))
	}
	_, cmd = p.Update(special(tea.KeyEnter))
	p.Update(one(t, cmd))
	if p.opts.SessionID != "orq" {
		t.Fatalf("the client did not go back to the orchestrator: %s", p.opts.SessionID)
	}
}

func TestAChildThatCannotBeOpenedIsSaidAndNothingIsStarted(t *testing.T) {
	f := &fakeOrchestration{children: []protocol.Child{orchChild("c1", "índice", "merged", 1, 1, 0)}}
	p, tr := orchestrating(t, En, f)
	loadedWith(t, p, f)
	p.Update(ctrl('r'))
	_, cmd := p.Update(special(tea.KeyEnter))
	p.Update(one(t, cmd))
	if p.opts.SessionID != "orq" || len(tr.created) != 0 {
		t.Fatalf("a child the daemon does not hold was opened by guess: %s %+v", p.opts.SessionID, tr.created)
	}
	if !strings.Contains(screen(p), "c1") {
		t.Fatalf("the failure does not name the child:\n%s", screen(p))
	}
}

func TestCtrlRWithNoChildrenDoesNotOpen(t *testing.T) {
	p, _ := orchestrating(t, En, &fakeOrchestration{})
	p.Update(ctrl('r'))
	if p.model.Pick.Active {
		t.Fatal("an empty list of children took the keyboard")
	}
	if p.model.Flash != Text(En).PickNone {
		t.Fatalf("an empty list said %q", p.model.Flash)
	}
}

// one runs the single command a key produced and returns its message. A key
// that produced none, or one that never answers, fails the test rather than
// hanging it.
func one(t *testing.T, cmd tea.Cmd) tea.Msg {
	t.Helper()
	if cmd == nil {
		t.Fatal("the key produced no command")
	}
	done := make(chan tea.Msg, 1)
	go func() { done <- cmd() }()
	select {
	case m := <-done:
		return m
	case <-time.After(time.Second):
		t.Fatal("the key's command never answered")
	}
	return nil
}
