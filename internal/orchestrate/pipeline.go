package orchestrate

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/aguinelo/dcode/internal/protocol"
	"github.com/aguinelo/dcode/internal/session"
	"github.com/aguinelo/dcode/internal/tools"
)

// The pipeline of one child: when a turn of its ends done, the harness runs
// the owns check, the commit, the review and the gate, in that order and
// skipping none; the model reads the results and decides (owner's decision 2).
//
// Spec: docs/specs/architecture/orchestration/202610132000-orchestration.p.spec.md §6, §7.1–§7.4.

// Why a child was handed its work back (protocol.Return.Why), and the two
// phases of a return that asks for a red test first (.p §7.3).
const (
	returnReview  = "review"
	returnGateRed = "gate_red"
	phaseRed      = "red"
	phaseFix      = "fix"
)

const (
	// reviewAttempts is how many reviews without a verdict the pipeline
	// takes before it stops and reports (.config §4).
	reviewAttempts = 2
	// reviewDeadline bounds assembling one review: a few git reads.
	reviewDeadline = 3 * reviewGitDeadline
	// idleWait is how long a hand-back waits for the child's session to be
	// idle: its turn.completed is sent just before the turn lets go of it.
	idleWait = 10 * time.Second
)

// reviewing is a reviewer session at work on one child.
type reviewing struct {
	child   string
	review  *Review
	attempt int
}

// brief is what child id was dispatched with.
func (o *Orchestrator) brief(id string) Brief {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.state.Briefs[id]
}

func setBrief(st *State, id string, f func(*Brief)) {
	if st.Briefs == nil {
		st.Briefs = map[string]Brief{}
	}
	b := st.Briefs[id]
	f(&b)
	st.Briefs[id] = b
}

// checkOwns is .p §7.1 before the commit: every path git status lists in the
// worktree against the child's owns. What the child wrote through bash is on
// the list too. owns.checked says the paths.
func (o *Orchestrator) checkOwns(ctx context.Context, c protocol.Child) (changed, outside []string, err error) {
	changed, err = Uncommitted(ctx, c.Worktree)
	if err != nil || len(changed) == 0 {
		return changed, nil, err
	}
	mine, err := childsOwn(ctx, c.Worktree, changed)
	if err != nil {
		return changed, nil, err
	}
	outside = OutsideOwns(c.Owns, mine)
	data, _ := json.Marshal(map[string]any{"paths": changed, "outside": outside, "owns": c.Owns})
	why := fmt.Sprintf("the harness checked %d changed path(s) against owns before committing", len(changed))
	if len(outside) > 0 {
		why = fmt.Sprintf("%s changed outside owns, so nothing is committed and the work goes back without a review", strings.Join(outside, ", "))
	}
	o.note(Decision{Kind: "owns.checked", Child: c.ID, Why: why, Data: data})
	return changed, outside, nil
}

// childsOwn is what of changed the child wrote. In a merge of main the
// integration left in the worktree for a conflict, a path that is as main
// has it came with main, not from the child: only the rest is checked against
// owns, and the commit that concludes the merge takes both.
func childsOwn(ctx context.Context, worktree string, changed []string) ([]string, error) {
	if _, err := reviewGit(ctx, worktree, "rev-parse", "-q", "--verify", "MERGE_HEAD"); err != nil {
		return changed, nil // no merge in progress: every change is the child's
	}
	differs, err := reviewGit(ctx, worktree, "diff", "--name-only", "-z", "--no-renames", "MERGE_HEAD", "--")
	if err != nil {
		return nil, err
	}
	untracked, err := reviewGit(ctx, worktree, "ls-files", "-z", "--others", "--exclude-standard")
	if err != nil {
		return nil, err
	}
	byChild := map[string]bool{}
	for _, p := range strings.Split(differs+untracked, "\x00") {
		byChild[p] = p != ""
	}
	var mine []string
	for _, p := range changed {
		if byChild[p] {
			mine = append(mine, p)
		}
	}
	return mine, nil
}

// afterDone is the pipeline from a turn that ended done and committed: the
// red test of a red phase, or the review.
func (o *Orchestrator) afterDone(id string) {
	c := o.child(id)
	if c.Return != nil && c.Return.Phase == phaseRed {
		o.reg.async(func() { o.confirmRed(id) })
		return
	}
	if c.Head == "" {
		o.move(id, stateFailed, protocol.StopIncomplete, "the child's turn ended done and nothing was ever committed: there is nothing to review")
		o.reportChild(id, "its turn ended done without changing anything, so there is nothing to review; it failed incomplete.")
		return
	}
	o.move(id, stateInReview, protocol.StopDone, "the child's turn ended done; the pipeline reviews commit "+short(c.Head))
	o.startReview(id, 1)
}

// startReview assembles the review of the child's Base..Head and starts a
// reviewer on it: a session of its own, which only reads, whose first turn is
// the brief.
func (o *Orchestrator) startReview(id string, attempt int) {
	c, b := o.child(id), o.brief(id)
	ctx, cancel := context.WithTimeout(context.Background(), reviewDeadline)
	defer cancel()
	base, err := o.reviewBase(ctx, c)
	if err != nil {
		o.stall(id, err)
		return
	}
	req, err := AssembleReview(ctx, ReviewInput{ChildID: id, Worktree: c.Worktree, Base: base, Head: c.Head,
		Task: b.Task, Owns: c.Owns, MaxDiffBytes: o.review.MaxDiffBytes})
	var ov *OwnsViolation
	if errors.As(err, &ov) {
		o.giveBack(id, ov.Return(), true, ownsText(ov.Paths, c.Owns))
		return
	}
	if err != nil {
		o.stall(id, fmt.Errorf("the review of %s could not be assembled: %w", short(c.Head), err))
		return
	}
	profile, err := o.reviewerProfile()
	if err != nil {
		o.stall(id, err)
		return
	}
	rid := session.NewID(o.reg.now, randomUint32)
	req.ID, req.ParentID, req.Profile = rid, o.orchestratorID(), profile
	rv := NewReview(req, rid)
	req.Submit = rv.Submit
	o.mu.Lock()
	if o.reviews == nil {
		o.reviews = map[string]*reviewing{}
	}
	o.reviews[rid] = &reviewing{child: id, review: rv, attempt: attempt}
	o.mu.Unlock()
	if _, err := o.reg.spawner.SpawnReviewer(req); err != nil {
		o.dropReviewer(rid)
		o.stall(id, fmt.Errorf("the reviewer's session could not be built: %w", err))
		return
	}
	if err := o.reg.spawner.Submit(rid, req.Prompt(), ""); err != nil {
		o.dropReviewer(rid)
		o.closeSession(rid)
		o.stall(id, fmt.Errorf("the reviewer's turn could not start: %w", err))
	}
}

// reviewBase is where the child's own work starts: the merge-base of its head
// and main's tip. It is the branch's Base until the integration merges main
// into it; after that, what main brought is not the child's to review.
func (o *Orchestrator) reviewBase(ctx context.Context, c protocol.Child) (string, error) {
	p := o.Project()
	tip, err := o.mainNow(ctx)
	if err != nil {
		return "", fmt.Errorf("main %s cannot be read for the review: %w", p.Main, err)
	}
	base, err := wtGit(ctx, p.Root, "merge-base", tip, c.Head)
	if err != nil {
		return "", fmt.Errorf("the review finds no base for %s over main: %w", short(c.Head), err)
	}
	return base, nil
}

// reviewerProfile is the reviewer's profile: the first of
// reviewer_profiles, else the first of the child role's list, else the
// orchestrator's own model.
func (o *Orchestrator) reviewerProfile() (string, error) {
	fallback := o.childProfiles()[0]
	if len(o.review.Profiles) == 0 && strings.TrimSpace(fallback) == "" {
		return "", nil // the orchestrator's own model: nothing is declared
	}
	return o.review.Profile("", fallback)
}

func (o *Orchestrator) dropReviewer(rid string) *reviewing {
	o.mu.Lock()
	defer o.mu.Unlock()
	rv := o.reviews[rid]
	delete(o.reviews, rid)
	return rv
}

func (o *Orchestrator) reviewer(rid string) bool {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.reviews[rid] != nil
}

// closeSession closes a session the pipeline opened, saying a close that fails.
func (o *Orchestrator) closeSession(id string) {
	if err := o.reg.spawner.Close(id); err != nil && !isCode(err, protocol.CodeSessionNotFound) {
		o.sayError(protocol.CodeInternal, fmt.Sprintf("the pipeline's session %s did not close: %v", id, err))
	}
}

// reviewerEnded is the end of a reviewer's turn: the review is finished, its
// verdict in the log and on the wire, and the pipeline goes on from it. A
// review without a verdict runs once more; the second leaves the child
// in_review, reported.
func (o *Orchestrator) reviewerEnded(rid string) {
	rv := o.dropReviewer(rid)
	if rv == nil {
		return
	}
	o.closeSession(rid)
	id := rv.child
	v, err := rv.review.Finish(o.log)
	if v.Verdict != "" {
		if aerr := o.apply(edit(id, func(c *protocol.Child) { c.Review = &v })); aerr != nil {
			o.sayError(protocol.CodeInternal, fmt.Sprintf("the verdict on child %s is in the log and not in the state: %v", id, aerr))
		}
		o.emit(protocol.EventReviewVerdict, protocol.ReviewVerdict{ChildID: id, Verdict: v})
	}
	switch {
	case errors.Is(err, ErrNoVerdict) && rv.attempt < reviewAttempts:
		o.startReview(id, rv.attempt+1)
	case errors.Is(err, ErrNoVerdict):
		o.reportChild(id, fmt.Sprintf("its reviewer ended %d times without a verdict, which is no review; the child stays in_review. Close it, or wait for the person.", reviewAttempts))
	case err != nil:
		o.stall(id, fmt.Errorf("the verdict did not reach the log: %w", err))
	case v.Verdict == tools.VerdictApprove:
		o.reg.async(func() { o.measure(id) })
	default:
		o.giveBack(id, protocol.Return{Why: returnReview, Phase: phaseRed}, true, redText(v.Findings))
	}
}

// stall is a pipeline step that could not run: the child stays where it is,
// the person is told, and so is the orchestrator.
func (o *Orchestrator) stall(id string, err error) {
	o.sayError(protocol.CodeInternal, fmt.Sprintf("the pipeline of child %s stopped: %v", id, err))
	o.reportChild(id, "the pipeline stopped: "+err.Error())
}

// giveBack hands the child its work back: child.returned in the log, the
// child working again with the Return, and a turn with what to do. count is
// false for the red phase becoming the fix phase, which is one return in two
// halves. Past max_returns the child fails incomplete instead.
func (o *Orchestrator) giveBack(id string, ret protocol.Return, count bool, text string) {
	c := o.child(id)
	if terminal(c.State) {
		return
	}
	n := c.Returned
	if count {
		n++
	}
	if n > o.limits.MaxReturns {
		why := fmt.Sprintf("returned %d times, and orchestrator.max_returns is %d: the child fails rather than going round again", c.Returned, o.limits.MaxReturns)
		o.move(id, stateFailed, protocol.StopIncomplete, why)
		o.reportChild(id, why+"; the last return was "+ret.Why+".")
		return
	}
	data, _ := json.Marshal(map[string]any{"return": ret, "returned": n})
	why := fmt.Sprintf("the pipeline hands the work back (%s", ret.Why)
	if ret.Phase != "" {
		why += ", phase " + ret.Phase
	}
	why += ")"
	if len(ret.Paths) > 0 {
		why += ": " + strings.Join(ret.Paths, ", ")
	}
	back := edit(id, func(ch *protocol.Child) {
		r := ret
		ch.State, ch.Reason, ch.Waiting, ch.Return, ch.Returned = stateWorking, "", "", &r, n
	})
	if err := o.commit(Decision{Kind: "child.returned", Child: id, Why: why, Data: data}, back); err != nil {
		o.sayError(protocol.CodeInternal, fmt.Sprintf("child %s was not handed back: the log did not take it: %v", id, err))
		return
	}
	o.emit(protocol.EventChildState, protocol.ChildChanged{Child: o.child(id)})
	if err := o.submitIdle(id, text); err != nil {
		o.fail(id, fmt.Errorf("the hand-back's turn could not start: %w", err))
		o.reportChild(id, "it was handed back and its turn could not start: "+err.Error())
		return
	}
	o.reportChild(id, why+"; it is working on it.")
}

// submitIdle starts a turn in a session whose last turn just ended, waiting
// the moment it takes the session to be idle again.
func (o *Orchestrator) submitIdle(id, text string) error {
	deadline := time.Now().Add(idleWait)
	for {
		err := o.reg.spawner.Submit(id, text, "")
		if !isCode(err, protocol.CodeTurnAlreadyActive) || time.Now().After(deadline) {
			return err
		}
		time.Sleep(25 * time.Millisecond)
	}
}

// ownsText is the turn that hands back a change outside owns.
func ownsText(outside, owns []string) string {
	return "The harness did not commit your turn: you changed " + strings.Join(outside, ", ") +
		", outside the paths you own (" + strings.Join(owns, ", ") + "). Nothing was reviewed. " +
		"Undo the changes outside what you own — restore or delete those files — and keep the rest. " +
		"If the task needs them changed, say so in your report instead. End your turn when the tree holds only your paths."
}

// redText is the red phase's turn: the findings, and a test that shows them.
func redText(findings []protocol.Finding) string {
	var b strings.Builder
	b.WriteString("The reviewer asked for changes. First, the red phase: write only a test that fails because of " +
		"the defect below, without fixing it. Then call `red_test` with the command that runs that test, and end " +
		"your turn. The harness commits the test, runs the command on that commit, and accepts it only if it fails.\n\n")
	writeFindings(&b, findings)
	return b.String()
}

func writeFindings(b *strings.Builder, findings []protocol.Finding) {
	for _, f := range findings {
		at := f.Path
		if f.Line > 0 {
			at = fmt.Sprintf("%s:%d", f.Path, f.Line)
		}
		fmt.Fprintf(b, "- %s — %s (%s)\n", at, f.What, f.Why)
	}
}
