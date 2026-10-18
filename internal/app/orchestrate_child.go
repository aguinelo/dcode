package app

import (
	"context"
	"os"
	"path/filepath"

	"github.com/aguinelo/dcode/internal/orchestrate"
	"github.com/aguinelo/dcode/internal/policy"
	"github.com/aguinelo/dcode/internal/protocol"
	"github.com/aguinelo/dcode/internal/session"
	"github.com/aguinelo/dcode/internal/tools"
)

// The daemon's half of a child: the session it is, the tools the orchestrator
// conducts it with, and the events of both that the registry reads.
//
// Spec: docs/specs/architecture/orchestration/202610132000-orchestration.p.spec.md §5.1, §8, §10.

// orchestratorMode is plan mode with the orchestrator's tools over project id.
func (d *Daemon) orchestratorMode(projectID string) func(*Options) {
	return func(o *Options) {
		o.Conductor = conductor{d: d, project: projectID}
		planMode(o)
	}
}

// childMode is child id's boundary, set last so no request widens it: it
// writes inside its worktree and its owns alone, asks the person without a
// deadline, runs no git that writes, and delegates nothing. Its red_test
// reaches the pipeline of its project.
func (d *Daemon) childMode(id string, owns []string) func(*Options) {
	return func(o *Options) {
		o.Child, o.Owns, o.Conductor = true, append([]string(nil), owns...), nil
		o.RedTest = func(_ context.Context, command string) (string, error) { return d.redTest(id, command) }
		o.SandboxMode, o.Policy = policy.ModeWorkspaceWrite, policy.PolicyOnRequest
		o.Writable, o.Delegate, o.Qualify, o.LoopSpec = nil, false, false, ""
	}
}

// SpawnChild opens a child under req.ID in its worktree: a recorded session
// with the child role, its parent and project, and its owns. One that is not
// being recorded is closed and refused, as an orchestrator is.
func (d *Daemon) SpawnChild(req orchestrate.ChildRequest) (protocol.Session, error) {
	if err := session.ValidID(req.ID); err != nil {
		return protocol.Session{}, err
	}
	if _, err := d.manager.Get(req.ID); err == nil {
		return protocol.Session{}, protocol.Errorf(protocol.CodeInvalidInput, "a session %s is already open", req.ID)
	}
	parent, err := d.manager.Get(req.ParentID)
	if err != nil {
		return protocol.Session{}, err
	}
	if req.Profile != "" {
		opts, err := d.optionsFor(req.Worktree)
		if err != nil {
			return protocol.Session{}, err
		}
		if _, ok := opts.Profiles[req.Profile]; !ok {
			return protocol.Session{}, protocol.Errorf(protocol.CodeModelNotAllowed,
				"the child role's profile %q is no profile of models.toml", req.Profile)
		}
	}
	sess, err := d.buildAs(req.ID, protocol.CreateSessionRequest{Workspace: req.Worktree, Model: req.Profile}, d.childMode(req.ID, req.Owns))
	if err != nil {
		return protocol.Session{}, err
	}
	sess.Role, sess.ParentID, sess.ProjectID = protocol.RoleChild, req.ParentID, parent.ProjectID
	sess.Owns = append([]string(nil), req.Owns...)
	if err := d.manager.Add(sess); err != nil {
		sess.Close()
		return protocol.Session{}, err
	}
	sess.Emit(protocol.EventSessionCreated, sess.Describe())
	if _, err := os.Stat(filepath.Join(d.opts.RecordDir, sess.ID+".jsonl")); err != nil {
		_ = d.manager.Remove(sess.ID)
		return protocol.Session{}, protocol.Errorf(protocol.CodeOrchestratorUnavailable,
			"child %s is not being recorded, so it could not survive a restart: %v", sess.ID, err)
	}
	d.watch(sess)
	return sess.Describe(), nil
}

// Submit starts a turn in a session; origin is who started it.
func (d *Daemon) Submit(sessionID, text, origin string) error {
	s, err := d.manager.Get(sessionID)
	if err != nil {
		return err
	}
	return s.SubmitOrigin(text, origin)
}

// Steer hands the turn a session is in more of what to do.
func (d *Daemon) Steer(sessionID, text string) error {
	s, err := d.manager.Get(sessionID)
	if err != nil {
		return err
	}
	return s.Steer(text)
}

// watch hands the registry every event of a project's session, from now on,
// in order, until the session's log closes.
func (d *Daemon) watch(s *session.Session) {
	if d.projects == nil || d.projects.reg == nil {
		return
	}
	events, cancel, err := s.Log.Subscribe(context.Background(), s.Log.LastSeq()+1)
	if err != nil {
		if d.opts.Log != nil {
			d.opts.Log("the orchestrator cannot follow session " + s.ID + ": " + err.Error())
		}
		return
	}
	reg := d.projects.reg
	go func() {
		defer cancel()
		for ev := range events {
			reg.Observe(s.ID, ev)
		}
	}()
}

// emitTo sends an event on a live session's log; one that is gone is said.
func (d *Daemon) emitTo(sessionID string, t protocol.EventType, payload any) {
	s, err := d.manager.Get(sessionID)
	if err != nil {
		if d.opts.Log != nil {
			d.opts.Log("a " + string(t) + " for session " + sessionID + ", which is not open, was not sent")
		}
		return
	}
	s.Emit(t, payload)
}

// Children answers /children of a session: an orchestrator's, or none.
func (p *daemonProjects) Children(sessionID string) []protocol.Child {
	if p.reg == nil {
		return nil
	}
	kids, _ := p.reg.ChildrenOf(sessionID)
	return kids
}

// CloseChild is the person's DELETE of a child; held is false for a session
// that is no child here.
func (p *daemonProjects) CloseChild(sessionID string) (bool, error) {
	if p.reg == nil {
		return false, nil
	}
	return p.reg.CloseChild(sessionID)
}

// Questions answers /questions of a session: an orchestrator's, or none.
func (p *daemonProjects) Questions(sessionID string) ([]protocol.Question, bool) {
	if p.reg == nil {
		return nil, false
	}
	return p.reg.QuestionsOf(sessionID)
}

// Answer is the person's answer to a question of an orchestrator.
func (p *daemonProjects) Answer(sessionID, questionID string, req protocol.AnswerQuestionRequest) (protocol.Question, error) {
	if p.reg == nil {
		return protocol.Question{}, p.unavailable()
	}
	return p.reg.AnswerOf(sessionID, questionID, req.Answer, req.Note)
}

// Cards are the cards an orchestrator's questions are shown in.
func (p *daemonProjects) Cards(sessionID string) []protocol.QuestionCard {
	if p.reg == nil {
		return nil
	}
	cards, _ := p.reg.CardsOf(sessionID)
	return cards
}

// AnswerAll is the person's confirm of a card: every answer in one POST.
func (p *daemonProjects) AnswerAll(sessionID string, req protocol.AnswerQuestionsRequest) (protocol.AnswerQuestionsResponse, error) {
	if p.reg == nil {
		return protocol.AnswerQuestionsResponse{}, p.unavailable()
	}
	return p.reg.AnswersOf(sessionID, req)
}

// conductor is the project an orchestrator's tools reach, found when a tool
// runs: the session is built before its project is held.
type conductor struct {
	d       *Daemon
	project string
}

func (c conductor) held() (*orchestrate.Orchestrator, error) {
	if c.d.projects == nil || c.d.projects.reg == nil {
		return nil, protocol.Errorf(protocol.CodeOrchestratorUnavailable, "this daemon conducts no projects")
	}
	o, ok := c.d.projects.reg.Get(c.project)
	if !ok {
		return nil, protocol.Errorf(protocol.CodeProjectNotFound, "this daemon does not hold project %s", c.project)
	}
	return o, nil
}

func (c conductor) Delegate(items []tools.DelegateItem) ([]protocol.Child, error) {
	o, err := c.held()
	if err != nil {
		return nil, err
	}
	batch := make([]orchestrate.Item, 0, len(items))
	for _, it := range items {
		batch = append(batch, orchestrate.Item{Title: it.Title, Task: it.Task, Owns: it.Owns, Model: it.Model,
			PRTitle: it.PRTitle, PRBody: it.PRBody, FixesMain: it.FixesMain, Why: it.Why})
	}
	return o.Delegate(batch)
}

func (c conductor) Children() ([]protocol.Child, error) {
	o, err := c.held()
	if err != nil {
		return nil, err
	}
	return o.Children(), nil
}

func (c conductor) SteerChild(id, text, why string) error {
	o, err := c.held()
	if err != nil {
		return err
	}
	return o.Steer(id, text, why)
}

func (c conductor) CloseChild(id, why string) error {
	o, err := c.held()
	if err != nil {
		return err
	}
	return o.Close(id, why)
}

func (c conductor) Ask(in tools.AskInput) (string, error) {
	o, err := c.held()
	if err != nil {
		return "", err
	}
	options := make([]protocol.QuestionOption, 0, len(in.Options))
	for _, opt := range in.Options {
		options = append(options, protocol.QuestionOption{ID: opt.ID, Label: opt.Label, Description: opt.Description})
	}
	return o.AskWithHeader(in.Header, in.Text, options, in.Recommended, in.Why, in.ChildID)
}

func (c conductor) Summarize(qid, summary string) error {
	o, err := c.held()
	if err != nil {
		return err
	}
	return o.Summarize(qid, summary)
}

// ProposeAnswers makes a proposed card; the person's POST alone answers it.
func (c conductor) ProposeAnswers(in tools.ProposeInput) (string, error) {
	o, err := c.held()
	if err != nil {
		return "", err
	}
	answers := make([]protocol.AnswerItem, 0, len(in.Answers))
	for _, a := range in.Answers {
		answers = append(answers, protocol.AnswerItem{QuestionID: a.QuestionID, Answer: a.Answer, Note: a.Note})
	}
	card, err := o.ProposeAnswers(in.From, in.Why, answers)
	return card.ID, err
}

// reviewerMode is a reviewer's boundary: plan mode, the reviewer's registry
// with submit as its verdict, and nothing that conducts or delegates.
func reviewerMode(submit func(context.Context, tools.VerdictInput) (string, error)) func(*Options) {
	return func(o *Options) {
		o.Child, o.Owns, o.Conductor, o.RedTest = false, nil, nil, nil
		o.Delegate, o.Qualify, o.LoopSpec, o.Memory = false, false, "", false
		o.Reviewer = submit
		planMode(o)
	}
}

// SpawnReviewer opens a reviewer under req.ID in the child's worktree: a
// recorded session with the reviewer role and the orchestrator as parent,
// which only reads and ends its review with verdict.
func (d *Daemon) SpawnReviewer(req orchestrate.ReviewRequest) (protocol.Session, error) {
	if err := session.ValidID(req.ID); err != nil {
		return protocol.Session{}, err
	}
	if req.Submit == nil {
		return protocol.Session{}, protocol.Errorf(protocol.CodeInvalidInput, "a reviewer needs the review its verdict goes to")
	}
	parent, err := d.manager.Get(req.ParentID)
	if err != nil {
		return protocol.Session{}, err
	}
	sess, err := d.buildAs(req.ID, protocol.CreateSessionRequest{Workspace: req.Worktree, Model: req.Profile}, reviewerMode(req.Submit))
	if err != nil {
		return protocol.Session{}, err
	}
	sess.Role, sess.ParentID, sess.ProjectID = protocol.RoleReviewer, req.ParentID, parent.ProjectID
	if err := d.manager.Add(sess); err != nil {
		sess.Close()
		return protocol.Session{}, err
	}
	sess.Emit(protocol.EventSessionCreated, sess.Describe())
	d.watch(sess)
	return sess.Describe(), nil
}

// redTest hands a child's red_test to its project's pipeline.
func (d *Daemon) redTest(childID, command string) (string, error) {
	if d.projects == nil || d.projects.reg == nil {
		return "", protocol.Errorf(protocol.CodeOrchestratorUnavailable, "this daemon conducts no projects")
	}
	return d.projects.reg.RedTest(childID, command)
}

// knownProfile says whether models.toml at root has profile name.
func (d *Daemon) knownProfile(root, name string) bool {
	opts, err := d.optionsFor(root)
	if err != nil {
		return false
	}
	_, ok := opts.Profiles[name]
	return ok
}
