package orchestrate

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/aguinelo/dcode/internal/protocol"
	"github.com/aguinelo/dcode/internal/session"
)

// Dispatch: a batch of children, validated whole before the first worktree,
// each one a session in its own worktree on its own branch.
//
// Spec: docs/specs/architecture/orchestration/202610132000-orchestration.p.spec.md §6, §10.

// The states of a child that step 6 moves through (.p §6). in_review, pr and
// merged arrive with the pipeline.
const (
	stateDispatched = "dispatched"
	stateWorking    = "working"
	stateAwaiting   = "awaiting_person"
	stateFailed     = "failed"
	stateClosed     = "closed"
	stateInReview   = "in_review"
	statePR         = "pr"
	stateMerged     = "merged"
)

// terminal states hold nothing: no owns, no worktree, no place in max_children.
func terminal(state string) bool { return state == stateClosed || state == stateMerged }

// dispatchDeadline bounds the git one child's dispatch asks.
const dispatchDeadline = time.Minute

// ChildRequest is what the daemon needs to build a child: its id, chosen here
// because the worktree is named by it before the session exists, the
// orchestrator it answers to, its worktree, the profile it runs on, its task
// and its owns.
type ChildRequest struct {
	ID, ParentID, Worktree, Profile, Task string
	Owns                                  []string
}

// Item is one entry of a delegate batch, as the orchestrator's model wrote it.
type Item struct {
	Title, Task     string
	Owns            []string
	Model           string
	PRTitle, PRBody string
	FixesMain       bool
	// Why is the model's reason for this child, and goes to the log.
	Why string
}

// Delegate dispatches a batch, all or nothing: owns, overlap with each other
// and with every live child, max_children, the role's profiles and the token
// ceiling are checked for the whole batch before any worktree is made. It
// returns once the children's first turns are submitted, never waiting for
// one to end: the orchestrator stays idle while they run.
func (o *Orchestrator) Delegate(items []Item) ([]protocol.Child, error) {
	o.dispatch.Lock()
	defer o.dispatch.Unlock()
	profiles, err := o.admit(items)
	if err != nil {
		o.refuse(items, err)
		return nil, err
	}
	out := make([]protocol.Child, 0, len(items))
	for i, it := range items {
		out = append(out, o.dispatchOne(it, profiles[i]))
	}
	return out, nil
}

// admit checks the batch whole and gives back each item's profile.
func (o *Orchestrator) admit(items []Item) ([]string, error) {
	st := o.snapshot()
	if len(items) == 0 {
		return nil, protocol.Errorf(protocol.CodeInvalidInput, "delegate needs at least one item")
	}
	if st.Spent >= o.limits.TokenCeiling {
		return nil, protocol.Errorf(protocol.CodeCeilingReached,
			"the project has spent %d tokens, and orchestrator.token_ceiling is %d: no child is dispatched until the person answers the ceiling question",
			st.Spent, o.limits.TokenCeiling)
	}
	profiles := make([]string, len(items))
	for i, it := range items {
		if strings.TrimSpace(it.Title) == "" || strings.TrimSpace(it.Task) == "" || strings.TrimSpace(it.Why) == "" {
			return nil, protocol.Errorf(protocol.CodeInvalidInput,
				"item %d needs a title, a task and a why; the batch is refused whole", i+1)
		}
		if err := validOwns(it.Title, it.Owns); err != nil {
			return nil, err
		}
		p, ok := o.profileFor(it.Model)
		if !ok {
			return nil, protocol.Errorf(protocol.CodeModelNotAllowed,
				"%q asks for the model %q, which is not in the child role's list (%s); the batch is refused whole",
				it.Title, it.Model, o.allowedProfiles())
		}
		profiles[i] = p
	}
	if err := overlapIn(items, leasesOf(st.Children)); err != nil {
		return nil, err
	}
	live := 0
	for _, c := range st.Children {
		if !terminal(c.State) {
			live++
		}
	}
	if live+len(items) > o.limits.MaxChildren {
		return nil, protocol.Errorf(protocol.CodeMaxChildrenReached,
			"%d children are live and the batch has %d; orchestrator.max_children is %d, so the batch is refused whole",
			live, len(items), o.limits.MaxChildren)
	}
	return profiles, nil
}

// child is the allow-list of the child role: child_profiles, or the
// orchestrator's own profile when none is declared.
func (o *Orchestrator) childProfiles() []string {
	if len(o.limits.ChildProfiles) > 0 {
		return o.limits.ChildProfiles
	}
	return []string{o.ownProfile}
}

// profileFor is the profile a requested model runs on: empty is the first of
// the list, anything else must be in it.
func (o *Orchestrator) profileFor(model string) (string, bool) {
	allowed := o.childProfiles()
	if model == "" {
		return allowed[0], true
	}
	for _, p := range allowed {
		if p == model {
			return p, true
		}
	}
	return "", false
}

func (o *Orchestrator) allowedProfiles() string {
	var names []string
	for _, p := range o.childProfiles() {
		if p == "" {
			p = "the orchestrator's model"
		}
		names = append(names, p)
	}
	return strings.Join(names, ", ")
}

// refuse says a refused batch: in the log, and to the person as a session
// error on the orchestrator. The ceiling also asks its question, once a window.
func (o *Orchestrator) refuse(items []Item, err error) {
	e, ok := err.(*protocol.Error)
	if !ok {
		e = &protocol.Error{Code: protocol.CodeInternal, Message: err.Error()}
	}
	if e.Code == protocol.CodeCeilingReached {
		o.reachCeiling(e.Message)
	}
	titles := make([]string, 0, len(items))
	for _, it := range items {
		titles = append(titles, it.Title)
	}
	data, _ := json.Marshal(map[string]any{"code": e.Code, "titles": titles})
	if rerr := o.record(Decision{Kind: "dispatch.refused", Session: o.orchestratorID(), Why: e.Message, Data: data}); rerr != nil {
		o.sayError(protocol.CodeInternal, "the refusal of a batch did not reach the decision log: "+rerr.Error())
	}
	o.sayError(e.Code, e.Message)
}

// dispatchOne makes one child of an admitted batch. What fails after
// admission fails that child alone, said: dispatched to failed, reason error.
func (o *Orchestrator) dispatchOne(it Item, profile string) protocol.Child {
	ctx, cancel := context.WithTimeout(context.Background(), dispatchDeadline)
	defer cancel()
	p, orq := o.Project(), o.orchestratorID()
	id := session.NewID(o.reg.now, randomUint32)
	c := protocol.Child{
		ID: id, ParentID: orq, Title: it.Title, Profile: profile, State: stateDispatched,
		Owns: append([]string(nil), it.Owns...), Branch: branchFor(o.limits.BranchPrefix, it.Title, id),
		Worktree: worktreeOf(o.reg.dir, p.ID, id), Started: o.reg.now().UTC(),
	}
	base, berr := mainTip(ctx, p.Root, p.Main)
	c.Base = base
	data, _ := json.Marshal(map[string]any{"title": it.Title, "task": it.Task, "owns": it.Owns,
		"pr_title": it.PRTitle, "pr_body": it.PRBody, "fixes_main": it.FixesMain, "branch": c.Branch})
	if err := o.commit(Decision{Kind: "dispatch", Child: id, Session: orq, Why: it.Why, Data: data},
		func(st *State) { st.Children = append(st.Children, c) }); err != nil {
		c.State, c.Reason = stateFailed, protocol.StopError
		o.sayError(protocol.CodeInternal, fmt.Sprintf("%q was not dispatched: the decision log did not take it: %v", it.Title, err))
		return c
	}
	modelWhy := it.Why
	if it.Model == "" {
		modelWhy = "the item named no model, so the harness took the first of the child role's list"
	}
	model, _ := json.Marshal(map[string]string{"profile": profile, "asked": it.Model})
	o.note(Decision{Kind: "model.chosen", Child: id, Session: orq, Why: modelWhy, Data: model})
	if berr != nil {
		return o.fail(id, fmt.Errorf("main %s has no commit to branch from: %w", p.Main, berr))
	}
	before := hooksPath(ctx, p.Root)
	if err := addWorktree(ctx, p.Root, c.Worktree, c.Branch, base); err != nil {
		return o.fail(id, err)
	}
	made, _ := json.Marshal(map[string]string{"worktree": c.Worktree, "branch": c.Branch, "base": base})
	o.note(Decision{Kind: "worktree.created", Child: id, Data: made,
		Why: "a child works in its own worktree, from main's tip, never in the person's tree"})
	if after := hooksPath(ctx, p.Root); after != before {
		hooks, _ := json.Marshal(map[string]string{"before": before, "after": after})
		o.note(Decision{Kind: "hooks.changed", Child: id, Data: hooks,
			Why: "core.hooksPath read after making the worktree differs from the one read before; the orchestrator does not rewrite it"})
	}
	sess, err := o.reg.spawner.SpawnChild(ChildRequest{ID: id, ParentID: orq, Worktree: c.Worktree,
		Profile: profile, Task: it.Task, Owns: c.Owns})
	if err != nil {
		return o.fail(id, fmt.Errorf("the child's session could not be built: %w", err))
	}
	o.setModel(id, sess.Model)
	o.emit(protocol.EventChildStarted, protocol.ChildChanged{Child: o.child(id)})
	if err := o.reg.spawner.Submit(id, childTask(it, c), ""); err != nil {
		return o.fail(id, fmt.Errorf("the child's first turn could not start: %w", err))
	}
	return o.move(id, stateWorking, "", "the worktree and the session exist and the first turn was submitted")
}

// childTask is the first turn's text: the task, and what the harness holds the
// child to. In English, as everything the model reads.
func childTask(it Item, c protocol.Child) string {
	return it.Task + "\n\n---\n" +
		"You are a child of an orchestrator, working in your own git worktree on the branch " + c.Branch + ".\n" +
		"You may write only these paths: " + strings.Join(c.Owns, ", ") + ". A write outside them is refused.\n" +
		"Do not run git commands that write (commit, merge, rebase, push, reset, checkout, switch, stash, config, worktree, tag, branch): " +
		"the harness commits what you changed at the end of each of your turns, and pushes and merges for you.\n" +
		"When the work is done, end your turn with a short report: what you changed, and how you checked it."
}

func randomUint32() uint32 {
	var b [4]byte
	_, _ = rand.Read(b[:])
	return binary.BigEndian.Uint32(b[:])
}
