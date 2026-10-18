package orchestrate

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/aguinelo/dcode/internal/protocol"
)

// The merge step wired into the pipeline: a child measured green is proposed —
// pushed, its PR opened, the person asked — and only the person's merge puts
// it in the serial queue. What the merges keep lives in state.json too, so a
// restart gives the questions, the queue and main's health back.
//
// Spec: docs/specs/architecture/orchestration/202610132000-orchestration.p.spec.md §6, §7.5, §7.6, §8.

// mergeOpenDeadline bounds opening the merge step: the integration place, a git remote and gh auth status.
const mergeOpenDeadline = forgeDeadline

// merges is the project's merge step, opened the first time a child needs it:
// the integration place, the forge, and the Merges given back what the state
// kept of them.
func (o *Orchestrator) merges() (*Merges, *Integration, error) {
	o.mergeMu.Lock()
	defer o.mergeMu.Unlock()
	if o.mg != nil {
		return o.mg, o.in, nil
	}
	p := o.Project()
	ctx, cancel := context.WithTimeout(context.Background(), mergeOpenDeadline)
	defer cancel()
	in, err := OpenIntegration(ctx, p.Root, filepath.Join(o.reg.dir, p.ID))
	if err != nil {
		return nil, nil, err
	}
	forge, err := NewForge(ctx, ForgeSetup{Declared: o.mergeCfg.Forge, Root: p.Root, Main: p.Main,
		Integration: in, Gate: o.gate, Now: o.reg.now})
	if err != nil {
		return nil, nil, err
	}
	st := o.snapshot()
	var asked []protocol.Question
	for _, q := range st.Questions {
		if q.Kind == "merge" {
			asked = append(asked, q)
		}
	}
	mg, err := NewMerges(MergeOptions{Log: o.log, Forge: forge, Config: o.mergeCfg, Poll: o.reg.healthPoll,
		Emit: o.emit, Now: o.reg.now, LocalSaid: st.LocalSaid, Questions: asked, Queue: st.Queue, Main: st.Main})
	if err != nil {
		return nil, nil, err
	}
	if err := o.apply(func(s *State) {
		s.LocalSaid = s.LocalSaid || forge.Name() == ForgeLocal
		s.Project.Forge = forge.Name()
	}); err != nil {
		o.sayError(protocol.CodeInternal, "the project's forge could not be kept in its state: "+err.Error())
	}
	o.mg, o.in, o.forge = mg, in, forge
	return mg, in, nil
}

// mainNow is main's tip as the project's forge has it now — origin's after a
// fetch with github, the local branch with local: what the pipeline measures
// and reviews over is what the merge will compare with (.p §7.4.1).
func (o *Orchestrator) mainNow(ctx context.Context) (string, error) {
	if _, _, err := o.merges(); err != nil {
		return "", fmt.Errorf("the merge step could not open: %w", err)
	}
	ctx, cancel := context.WithTimeout(ctx, forgeDeadline)
	defer cancel()
	return o.forge.MainTip(ctx)
}

// saveMerges keeps in the state what the merges hold: their questions, the
// queue and main's health. Called after each step of theirs, whose lines are
// in the log already.
func (o *Orchestrator) saveMerges(mg *Merges) {
	asked := mg.Questions()
	if err := o.apply(func(st *State) {
		kept := st.Questions[:0:0]
		for _, q := range st.Questions {
			if q.Kind != "merge" {
				kept = append(kept, q)
			}
		}
		st.Questions, st.Queue, st.Main = append(kept, asked...), mg.Queued(), mg.Main()
	}); err != nil {
		o.sayError(protocol.CodeInternal, "the merge questions and queue are in the log and not in the state: "+err.Error())
	}
}

// propose is the pipeline's last step: the child's review approved and its
// gate measured green on its head, so the merge step pushes it, opens its PR
// and asks the person; the child goes to pr, waiting on the merge.
func (o *Orchestrator) propose(id string) {
	c, b := o.child(id), o.brief(id)
	if terminal(c.State) {
		return
	}
	if c.Gate == nil || c.Gate.Result != GateGreen || c.Gate.Commit != c.Head {
		o.stall(id, fmt.Errorf("%s has no green measurement of its head %s, so it is not proposed", id, short(c.Head)))
		return
	}
	mg, _, err := o.merges()
	if err != nil {
		o.stall(id, fmt.Errorf("the merge step could not open: %w", err))
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), reviewDeadline)
	defer cancel()
	paths, lines, err := diffSize(ctx, c.Worktree, c.Gate.Base, c.Head)
	if err != nil {
		o.stall(id, err)
		return
	}
	title := b.PRTitle
	if strings.TrimSpace(title) == "" {
		title = c.Title
	}
	got, q, err := mg.Propose(ctx, Proposal{Child: c, Title: title, Body: b.PRBody, Paths: paths, Lines: lines})
	if err != nil {
		o.stall(id, fmt.Errorf("the merge step did not propose it: %w", err))
		return
	}
	o.saveMerges(mg)
	data, _ := json.Marshal(map[string]any{"from": stateInReview, "to": statePR, "pr": got.PR, "question": q.ID})
	if err := o.commit(Decision{Kind: "child.state", Child: id, Data: data,
		Why: "review approved and gate green on its head: the merge waits for the person's answer to " + q.ID},
		edit(id, func(ch *protocol.Child) { ch.State, ch.Waiting, ch.PR = statePR, "merge", got.PR })); err != nil {
		o.sayError(protocol.CodeInternal, fmt.Sprintf("child %s did not move to pr: %v", id, err))
		return
	}
	o.emit(protocol.EventChildState, protocol.ChildChanged{Child: o.child(id)})
	o.reportChild(id, fmt.Sprintf("its review approved and the gate measured green on %s; the person is asked whether it merges (%s).", short(c.Head), q.ID))
}

// answerMerge is the person's answer to a merge question: merge puts the child
// in the queue, which then runs; return hands the work back; close closes it.
func (o *Orchestrator) answerMerge(q protocol.Question, answer, note string) (protocol.Question, error) {
	mg, _, err := o.merges()
	if err != nil {
		return q, protocol.Errorf(protocol.CodeInternal, "the merge step could not open: %v", err)
	}
	got, err := mg.Answer(q.ID, answer, note)
	o.saveMerges(mg)
	if err != nil {
		return got, err
	}
	id := got.ChildID
	o.Report(fmt.Sprintf("The person answered %s to the merge of child %s (%s).", answer, id, got.ID))
	switch answer {
	case answerMerge:
		if aerr := o.apply(edit(id, func(c *protocol.Child) { c.Waiting = "" })); aerr != nil {
			o.sayError(protocol.CodeInternal, "the child's wait could not be cleared: "+aerr.Error())
		}
		o.reg.async(o.drain)
	case answerReturn:
		text := "The person read your pull request and handed it back to you."
		if strings.TrimSpace(note) != "" {
			text += " Their note: " + note
		}
		o.giveBack(id, protocol.Return{Why: "person"}, true, text+"\n\nChange what they ask and end your turn; the pipeline reviews and measures it again.")
	case answerClose:
		if cerr := o.Close(id, "the person answered close to the merge question"); cerr != nil {
			o.sayError(protocol.CodeInternal, fmt.Sprintf("child %s could not be closed: %v", id, cerr))
		}
	}
	return got, nil
}

// drain merges the queue, one child at a time and in order, until it is
// empty or a refusal stops it: a red or unknown main waits for green.
func (o *Orchestrator) drain() {
	mg, _, err := o.merges()
	if err != nil {
		o.sayError(protocol.CodeInternal, "the merge queue could not run: "+err.Error())
		return
	}
	o.drainMu.Lock()
	defer o.drainMu.Unlock()
	for _, id := range mg.Queued() {
		c := o.child(id)
		out, err := mg.Merge(context.Background(), c, o.brief(id).FixesMain)
		o.saveMerges(mg)
		switch {
		case err != nil:
			o.stall(id, fmt.Errorf("the merge did not run: %w", err))
			return
		case out.Completed != nil:
			o.merged(id, *out.Completed)
			if out.Main.Health != healthGreen {
				o.Report(fmt.Sprintf("Main is %s after the merge of child %s; the merge queue stops until main is green.", out.Main.Health, id))
				return
			}
		case out.Refused != nil && out.Refused.Code == protocol.CodeMainRed:
			o.reportChild(id, "its merge waits: "+out.Refused.Reason)
			return
		case out.Refused != nil && out.Refused.Code == protocol.CodeGateStale:
			o.move(id, stateInReview, "", "main or the branch moved after the gate: the pipeline measures again")
			o.reportChild(id, "its measurement went stale before the merge ("+out.Refused.Reason+"); the pipeline measures it again.")
			o.reg.async(func() { o.measure(id) })
		case out.Refused != nil:
			o.reportChild(id, "its merge was refused: "+out.Refused.Reason)
		}
	}
}

// merged is a child whose merge completed: merged, its session closed, its
// worktree removed and its owns free.
func (o *Orchestrator) merged(id string, done protocol.MergeCompleted) {
	c := o.child(id)
	o.move(id, stateMerged, "", fmt.Sprintf("merged into main as %s, the squash of the measured %s", short(done.Merged), short(done.Measured)))
	o.closeSession(id)
	ctx, cancel := context.WithTimeout(context.Background(), dispatchDeadline)
	defer cancel()
	if err := removeWorktree(ctx, o.Project().Root, c.Worktree); err != nil {
		o.sayError(protocol.CodeInternal, fmt.Sprintf("the worktree of merged child %s stays at %s: %v", id, c.Worktree, err))
	} else {
		gone, _ := json.Marshal(map[string]string{"worktree": c.Worktree, "branch": c.Branch})
		o.note(Decision{Kind: "worktree.removed", Child: id, Data: gone, Why: "the child merged; its owns are free"})
	}
	o.reportChild(id, fmt.Sprintf("it merged into main as %s.", short(done.Merged)))
}

// diffSize is the paths and the changed lines of base..head, which say
// whether a merge question is sensitive.
func diffSize(ctx context.Context, worktree, base, head string) ([]string, int, error) {
	out, err := reviewGit(ctx, worktree, "diff", "--numstat", "--no-renames", base, head)
	if err != nil {
		return nil, 0, err
	}
	var paths []string
	lines := 0
	for _, row := range strings.Split(strings.TrimSpace(out), "\n") {
		f := strings.SplitN(row, "\t", 3)
		if len(f) != 3 {
			continue
		}
		paths = append(paths, f[2])
		for _, n := range f[:2] {
			if v, err := strconv.Atoi(n); err == nil { // "-" is a binary file
				lines += v
			}
		}
	}
	return paths, lines, nil
}
