package orchestrate

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/aguinelo/dcode/internal/protocol"
)

// A project's children after dispatch: listed, steered, closed, and every
// change of theirs in the log before the state, and in the state before the
// wire (.p §8).
//
// Spec: docs/specs/architecture/orchestration/202610132000-orchestration.p.spec.md §5.1, §6, §8, §11.

// snapshot is a copy of the state, safe to read without the lock.
func (o *Orchestrator) snapshot() State {
	o.mu.Lock()
	defer o.mu.Unlock()
	return cloneState(o.state)
}

func cloneState(st State) State {
	st.Children = append([]protocol.Child(nil), st.Children...)
	st.Questions = append([]protocol.Question(nil), st.Questions...)
	return st
}

// commit writes d to the log, then the state mutate makes of the current one:
// the log before state.json (.p §8). A line that does not write changes
// nothing, and neither does a state that does not save.
func (o *Orchestrator) commit(d Decision, mutate func(*State)) error {
	o.mu.Lock()
	defer o.mu.Unlock()
	if d.Child != "" && d.Session == "" {
		d.Session = o.state.Project.OrchestratorID
	}
	seq, err := o.log.Record(d)
	if err != nil {
		return err
	}
	st := cloneState(o.state)
	if mutate != nil {
		mutate(&st)
	}
	st.LogSeq = seq
	if err := o.store.Save(st); err != nil {
		return err
	}
	o.state = st
	return nil
}

// record writes d and the state that follows it, unchanged but for the seq.
func (o *Orchestrator) record(d Decision) error { return o.commit(d, nil) }

// note records d, and says it when the log does not take it.
func (o *Orchestrator) note(d Decision) {
	if err := o.record(d); err != nil {
		o.sayError(protocol.CodeInternal, fmt.Sprintf("a %s decision did not reach the log: %v", d.Kind, err))
	}
}

func (o *Orchestrator) orchestratorID() string { return o.Project().OrchestratorID }

// emit sends an event on the orchestrator's log: the wire, last.
func (o *Orchestrator) emit(t protocol.EventType, payload any) {
	if o.reg != nil && o.reg.emit != nil {
		o.reg.emit(o.orchestratorID(), t, payload)
	}
}

// sayError is a failure told to the person, on the orchestrator.
func (o *Orchestrator) sayError(code, msg string) {
	o.emit(protocol.EventSessionError, protocol.Error{Code: code, Message: msg})
}

// child is child id as the state holds it; zero when there is none.
func (o *Orchestrator) child(id string) protocol.Child {
	o.mu.Lock()
	defer o.mu.Unlock()
	if i := indexOf(o.state.Children, id); i >= 0 {
		return o.state.Children[i]
	}
	return protocol.Child{}
}

func indexOf(children []protocol.Child, id string) int {
	for i, c := range children {
		if c.ID == id {
			return i
		}
	}
	return -1
}

// edit is a mutation of one child.
func edit(id string, f func(*protocol.Child)) func(*State) {
	return func(st *State) {
		if i := indexOf(st.Children, id); i >= 0 {
			f(&st.Children[i])
		}
	}
}

// setModel keeps the model the child's session was built with. Not a
// decision: model.chosen already said which profile and why.
func (o *Orchestrator) setModel(id, model string) {
	o.mu.Lock()
	defer o.mu.Unlock()
	if i := indexOf(o.state.Children, id); i >= 0 {
		o.state.Children[i].Model = model
	}
}

// move is a transition of the state machine: a child.state line, the state,
// then child.state on the wire. reason is how the last turn ended, or empty
// for a turn now running.
func (o *Orchestrator) move(id, to, reason, why string) protocol.Child {
	from := o.child(id).State
	data, _ := json.Marshal(map[string]string{"from": from, "to": to, "reason": reason})
	err := o.commit(Decision{Kind: "child.state", Child: id, Why: why, Data: data}, edit(id, func(c *protocol.Child) {
		c.State, c.Reason = to, reason
		if to != stateAwaiting {
			c.Waiting = ""
		}
		if terminal(to) {
			now := o.reg.now().UTC()
			c.Ended = &now
		}
	}))
	if err != nil {
		o.sayError(protocol.CodeInternal, fmt.Sprintf("child %s did not move from %s to %s: the log or the state did not take it: %v", id, from, to, err))
		return o.child(id)
	}
	c := o.child(id)
	o.emit(protocol.EventChildState, protocol.ChildChanged{Child: c})
	return c
}

// fail moves a child to failed with reason error, saying why to the person.
func (o *Orchestrator) fail(id string, err error) protocol.Child {
	o.sayError(protocol.CodeInternal, fmt.Sprintf("child %s failed: %v", id, err))
	return o.move(id, stateFailed, protocol.StopError, err.Error())
}

// Children are the children /children answers: every one but a closed one
// whose retention has passed. A failed one stays until someone closes it.
func (o *Orchestrator) Children() []protocol.Child {
	st := o.snapshot()
	now := o.reg.now()
	out := make([]protocol.Child, 0, len(st.Children))
	for _, c := range st.Children {
		if c.State == stateClosed && c.Ended != nil && now.Sub(*c.Ended) >= o.reg.retention {
			continue
		}
		out = append(out, c)
	}
	return out
}

// Steer hands a child more work: a new turn when it is idle or failed, or
// steering of the turn it is in. A child the pipeline holds is the
// pipeline's (child_in_pipeline).
func (o *Orchestrator) Steer(id, text, why string) error {
	c := o.child(id)
	switch {
	case c.ID == "":
		return protocol.Errorf(protocol.CodeSessionNotFound, "this orchestrator has no child %s", id)
	case strings.TrimSpace(text) == "" || strings.TrimSpace(why) == "":
		return protocol.Errorf(protocol.CodeInvalidInput, "steering a child needs a text and a why")
	case terminal(c.State):
		return protocol.Errorf(protocol.CodeInvalidInput, "child %s is %s, and a %s child takes no more work", id, c.State, c.State)
	case c.State == stateInReview || c.State == statePR:
		return protocol.Errorf(protocol.CodeChildInPipeline, "child %s is %s: the pipeline holds it until it returns", id, c.State)
	}
	data, _ := json.Marshal(map[string]string{"steer": text})
	if err := o.record(Decision{Kind: "dispatch", Child: id, Why: why, Data: data}); err != nil {
		return protocol.Errorf(protocol.CodeInternal, "the steering of %s did not reach the log: %v", id, err)
	}
	running := c.Reason == "" && (c.State == stateWorking || c.State == stateAwaiting)
	if running {
		return o.reg.spawner.Steer(id, text)
	}
	if err := o.reg.spawner.Submit(id, text, ""); err != nil {
		return err
	}
	o.move(id, stateWorking, "", "the orchestrator gave the child a new turn: "+why)
	return nil
}

// Close closes a child that is not terminal: its session, its worktree, its
// owns. by says who — the orchestrator's close_child, or the person's DELETE.
func (o *Orchestrator) Close(id, why string) error {
	o.dispatch.Lock()
	defer o.dispatch.Unlock()
	c := o.child(id)
	switch {
	case c.ID == "":
		return protocol.Errorf(protocol.CodeSessionNotFound, "this orchestrator has no child %s", id)
	case strings.TrimSpace(why) == "":
		return protocol.Errorf(protocol.CodeInvalidInput, "closing a child needs a why")
	case terminal(c.State):
		return protocol.Errorf(protocol.CodeInvalidInput, "child %s is already %s", id, c.State)
	}
	if err := o.reg.spawner.Close(id); err != nil && !isCode(err, protocol.CodeSessionNotFound) {
		o.sayError(protocol.CodeInternal, fmt.Sprintf("the session of child %s did not close: %v", id, err))
	}
	data, _ := json.Marshal(map[string]string{"from": c.State, "worktree": c.Worktree, "branch": c.Branch})
	if err := o.commit(Decision{Kind: "child.closed", Child: id, Why: why, Data: data}, edit(id, func(ch *protocol.Child) {
		now := o.reg.now().UTC()
		ch.State, ch.Reason, ch.Waiting, ch.Ended = stateClosed, "cancelled", "", &now
	})); err != nil {
		return protocol.Errorf(protocol.CodeInternal, "the close of %s did not reach the log: %v", id, err)
	}
	o.emit(protocol.EventChildState, protocol.ChildChanged{Child: o.child(id)})
	ctx, cancel := context.WithTimeout(context.Background(), dispatchDeadline)
	defer cancel()
	if err := removeWorktree(ctx, o.Project().Root, c.Worktree); err != nil {
		o.sayError(protocol.CodeInternal, fmt.Sprintf("the worktree of closed child %s stays at %s: %v", id, c.Worktree, err))
		return nil
	}
	gone, _ := json.Marshal(map[string]string{"worktree": c.Worktree, "branch": c.Branch})
	o.note(Decision{Kind: "worktree.removed", Child: id, Data: gone,
		Why: "the child was closed; its branch stays, with what the harness committed"})
	return nil
}

func isCode(err error, code string) bool {
	var e *protocol.Error
	return errors.As(err, &e) && e.Code == code
}

// Questions are the questions of the project, answered or not.
func (o *Orchestrator) Questions() []protocol.Question { return o.snapshot().Questions }

// reachCeiling says the ceiling in the log and asks its question, unless one
// is already waiting.
func (o *Orchestrator) reachCeiling(why string) {
	for _, q := range o.snapshot().Questions {
		if q.Kind == "ceiling" && q.Answer == "" {
			return
		}
	}
	st := o.snapshot()
	data, _ := json.Marshal(map[string]int64{"spent": st.Spent, "ceiling": o.limits.TokenCeiling})
	o.note(Decision{Kind: "ceiling.reached", Session: st.Project.OrchestratorID, Why: why, Data: data})
	o.ask(protocol.Question{Kind: "ceiling", Recommended: "stop",
		Text: fmt.Sprintf("The project has spent %d tokens, past orchestrator.token_ceiling (%d). Continue with a new window, or stop dispatching?", st.Spent, o.limits.TokenCeiling),
		Why:  "children already running finish what they do; no new child is dispatched until this is answered",
		Options: []protocol.QuestionOption{
			{ID: "continue", Label: "Continue: start a new window of tokens"},
			{ID: "stop", Label: "Stop dispatching"},
		}})
}

// ask puts a question in the queue: question.asked, the state, the wire.
func (o *Orchestrator) ask(q protocol.Question) {
	st := o.snapshot()
	q.ID = fmt.Sprintf("q%d", len(st.Questions)+1)
	q.Asked = o.reg.now().UTC()
	data, _ := json.Marshal(q)
	if err := o.commit(Decision{Kind: "question.asked", Child: q.ChildID, Session: st.Project.OrchestratorID, Why: q.Why, Data: data},
		func(s *State) { s.Questions = append(s.Questions, q) }); err != nil {
		o.sayError(protocol.CodeInternal, "a question did not reach the log, and is not asked: "+err.Error())
		return
	}
	o.emit(protocol.EventQuestionAsked, q)
}

// Answer answers a question; the first answer wins. continue on the ceiling
// question opens a new window: Spent goes back to zero.
func (o *Orchestrator) Answer(id, answer, note string) (protocol.Question, error) {
	st := o.snapshot()
	i := -1
	for j, q := range st.Questions {
		if q.ID == id {
			i = j
		}
	}
	if i < 0 {
		return protocol.Question{}, protocol.Errorf(protocol.CodeQuestionNotFound, "this project has no question %s", id)
	}
	q := st.Questions[i]
	if q.Answer != "" {
		return q, protocol.Errorf(protocol.CodeQuestionAlreadyAnswered, "question %s was already answered %q", id, q.Answer)
	}
	known := false
	for _, opt := range q.Options {
		known = known || opt.ID == answer
	}
	if !known {
		return q, protocol.Errorf(protocol.CodeInvalidInput, "%q is not one of the options of question %s", answer, id)
	}
	now := o.reg.now().UTC()
	q.Answer, q.Note, q.Answered = answer, note, &now
	why := "the person answered " + answer
	if q.Kind == "ceiling" && answer == "continue" {
		why += ": the token window starts again from zero"
	}
	data, _ := json.Marshal(q)
	if err := o.commit(Decision{Kind: "question.answered", Session: st.Project.OrchestratorID, Child: q.ChildID, Why: why, Data: data},
		func(s *State) {
			for j := range s.Questions {
				if s.Questions[j].ID == id {
					s.Questions[j] = q
				}
			}
			if q.Kind == "ceiling" && answer == "continue" {
				s.Spent = 0
			}
		}); err != nil {
		return protocol.Question{}, protocol.Errorf(protocol.CodeInternal, "the answer did not reach the log: %v", err)
	}
	o.emit(protocol.EventQuestionAnswered, q)
	return q, nil
}

// spend adds tokens to the window. Accounting, not a decision: no log line,
// the state alone, saved so a restart keeps the window.
func (o *Orchestrator) spend(u *protocol.Usage) {
	n := tokens(u)
	if n == 0 {
		return
	}
	o.mu.Lock()
	defer o.mu.Unlock()
	st := cloneState(o.state)
	st.Spent += n
	if err := o.store.Save(st); err != nil {
		go o.sayError(protocol.CodeInternal, "the token window could not be saved: "+err.Error())
		return
	}
	o.state = st
}

// tokens are what the ceiling counts of a turn: input, output and cache read.
func tokens(u *protocol.Usage) int64 {
	if u == nil {
		return 0
	}
	return int64(u.InputTokens + u.OutputTokens + u.CacheReadTokens)
}

// reopenChildren brings back every child that is not terminal under its own
// id, after the orchestrator. A child whose turn was open when the daemon
// stopped — or that never got past dispatch — fails with daemon_restart, and
// one that could not be reopened fails with error; both are moves run after
// daemon.restart is in the log, so the line that says the restart comes first.
func (o *Orchestrator) reopenChildren() (reopened []string, failed map[string]string, moves []func()) {
	failed = map[string]string{}
	for _, c := range o.snapshot().Children {
		if terminal(c.State) {
			continue
		}
		id := c.ID
		if _, err := o.reg.spawner.Reopen(id, protocol.StopDaemonRestart); err != nil {
			failed[id] = err.Error()
			msg := fmt.Sprintf("the child could not be reopened after the daemon started again: %v", err)
			moves = append(moves, func() { o.move(id, stateFailed, protocol.StopError, msg) })
			continue
		}
		reopened = append(reopened, id)
		open := c.State == stateDispatched || (c.Reason == "" && (c.State == stateWorking || c.State == stateAwaiting))
		if open {
			moves = append(moves, func() {
				o.move(id, stateFailed, protocol.StopDaemonRestart,
					"the child's turn was open when the daemon stopped, and ended daemon_restart")
			})
		}
	}
	return reopened, failed, moves
}

// ChildrenOf answers /children of a session: an orchestrator's children, and
// held false for a session that conducts nothing here.
func (r *Registry) ChildrenOf(sessionID string) (children []protocol.Child, held bool) {
	if o := r.orchestratorOf(sessionID); o != nil {
		return o.Children(), true
	}
	return nil, false
}

// CloseChild is the person's DELETE of a child: held false when sessionID is
// no child of a project here.
func (r *Registry) CloseChild(sessionID string) (held bool, err error) {
	r.mu.Lock()
	var owner *Orchestrator
	for _, o := range r.projects {
		if o.child(sessionID).ID != "" {
			owner = o
		}
	}
	r.mu.Unlock()
	if owner == nil {
		return false, nil
	}
	return true, owner.Close(sessionID, "the person closed the child")
}

// orchestratorOf is the project whose orchestrator sessionID is.
func (r *Registry) orchestratorOf(sessionID string) *Orchestrator {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, o := range r.projects {
		if o.orchestratorID() == sessionID {
			return o
		}
	}
	return nil
}
