package orchestrate

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/aguinelo/dcode/internal/protocol"
	"github.com/aguinelo/dcode/internal/tools"
)

// The pipeline's measurements: the red test that has to fail before a fix is
// taken (RN-5), and the gate on the child's commit, after an approval.
//
// Spec: docs/specs/architecture/orchestration/202610132000-orchestration.p.spec.md §7.3, §7.4, §8.

// RedTest takes the command a child in its red phase names with red_test.
// The harness runs it on the commit of the turn, at the turn's end.
func (o *Orchestrator) RedTest(id, command string) (string, error) {
	c := o.child(id)
	command = strings.TrimSpace(command)
	switch {
	case c.ID == "":
		return "", fmt.Errorf("this session is no child of the project")
	case c.Return == nil || c.Return.Phase != phaseRed || c.State != stateWorking:
		return "", fmt.Errorf("red_test belongs to the red phase, after a review asked for changes; this child is not in it")
	case command == "":
		return "", fmt.Errorf("red_test needs the command that runs the failing test")
	}
	if err := o.apply(func(st *State) { setBrief(st, id, func(b *Brief) { b.RedTest = command }) }); err != nil {
		return "", fmt.Errorf("the red test could not be kept: %w", err)
	}
	return "Red test recorded: `" + command + "`. End your turn now; the harness commits your test and runs it, " +
		"and the test must fail before you fix anything.", nil
}

// confirmRed is the end of a red phase: the harness ran the child's test on
// the commit of the turn, before any fix. A failing test is red_test.confirmed
// and the fix phase; a passing one shows no defect, and the red phase again.
func (o *Orchestrator) confirmRed(id string) {
	c, b := o.child(id), o.brief(id)
	why := c.Return.Why
	if b.RedTest == "" {
		o.giveBack(id, protocol.Return{Why: why, Phase: phaseRed}, true, "You ended the red phase without calling "+
			"`red_test`. Write the failing test, call `red_test` with the command that runs it, then end your turn. "+
			"No fix is taken before the test is confirmed red.")
		return
	}
	if err := o.apply(func(st *State) { setBrief(st, id, func(b *Brief) { b.RedTest = "" }) }); err != nil {
		o.sayError(protocol.CodeInternal, fmt.Sprintf("the red test of child %s could not be cleared: %v", id, err))
	}
	// Its output beside the gates', not over one: a red test and a gate on
	// one commit are two measurements.
	g := Gate{Command: b.RedTest, Timeout: o.gate.Timeout}
	if o.gate.Logs != "" {
		g.Logs = filepath.Join(filepath.Dir(o.gate.Logs), "red_tests")
	}
	if g.Timeout <= 0 {
		g.Timeout = gateTimeoutDefault
	}
	r, err := g.Measure(context.Background(), o.Project().Root, c.Head)
	if err != nil {
		o.stall(id, fmt.Errorf("the red test could not run on %s: %w", short(c.Head), err))
		return
	}
	r.Base = c.Base
	if r.Result != GateRed {
		o.giveBack(id, protocol.Return{Why: why, Phase: phaseRed}, true, fmt.Sprintf("Your test `%s` did not fail on %s "+
			"(%s, exit %d), so it does not show the defect. Write a test that fails because of it, call `red_test` "+
			"again, and end your turn.\n\n%s", b.RedTest, short(c.Head), r.Result, r.ExitCode, outputOf(r)))
		return
	}
	measured, _ := json.Marshal(r)
	if err := o.record(Decision{Kind: "red_test.confirmed", Child: id, Measured: measured,
		Why: fmt.Sprintf("the harness ran %q on %s, before the fix: it fails (exit %d), so it shows the defect", b.RedTest, short(c.Head), r.ExitCode)}); err != nil {
		o.stall(id, fmt.Errorf("the confirmed red test did not reach the log: %w", err))
		return
	}
	var fix strings.Builder
	fmt.Fprintf(&fix, "Your test fails as it should (`%s`, exit %d). Now the fix phase: fix the defect so the test "+
		"passes, and end your turn. The pipeline reviews and measures it again.\n\n", b.RedTest, r.ExitCode)
	if c.Review != nil {
		writeFindings(&fix, c.Review.Findings)
	}
	o.giveBack(id, protocol.Return{Why: why, Phase: phaseFix}, false, fix.String())
}

// measure brings the child's branch up to date with main in the integration
// place — a merge, never a rebase — and runs the gate on the result H, in a
// throwaway tree at that exact commit, with Base the main tip it holds.
// gate.measured keeps what the child said beside what was measured.
func (o *Orchestrator) measure(id string) {
	c, p := o.child(id), o.Project()
	ctx := context.Background()
	_, in, err := o.merges()
	if err != nil {
		o.stall(id, fmt.Errorf("the merge step could not open: %w", err))
		return
	}
	gctx, cancel := context.WithTimeout(ctx, gateGitDeadline)
	tip, err := mainTip(gctx, p.Root, p.Main)
	cancel()
	if err != nil {
		o.stall(id, fmt.Errorf("main %s cannot be read: %w", p.Main, err))
		return
	}
	h, err := in.Update(ctx, o.log, id, c.Worktree, c.Head, tip)
	var cf *Conflict
	if errors.As(err, &cf) {
		o.handBack(id, cf.Return(), true, conflictText(cf.Paths), false)
		return
	}
	if err != nil {
		o.stall(id, fmt.Errorf("main could not be merged into the branch: %w", err))
		return
	}
	if h != c.Head {
		if aerr := o.apply(edit(id, func(ch *protocol.Child) { ch.Head = h })); aerr != nil {
			o.sayError(protocol.CodeInternal, fmt.Sprintf("the updated head of child %s is not in the state: %v", id, aerr))
		}
	}
	r, err := o.gate.Measure(ctx, p.Root, h)
	if err != nil && r.Result != GateUndeclared {
		o.stall(id, fmt.Errorf("the gate did not measure %s: %w", short(h), err))
		return
	}
	r.Base = tip
	if _, rerr := RecordGate(o.log, id, o.brief(id).Claim, r); rerr != nil {
		o.stall(id, fmt.Errorf("the measurement of %s did not reach the log: %w", short(h), rerr))
		return
	}
	if aerr := o.apply(edit(id, func(ch *protocol.Child) { g := r; ch.Gate = &g })); aerr != nil {
		o.sayError(protocol.CodeInternal, fmt.Sprintf("the gate of child %s is in the log and not in the state: %v", id, aerr))
	}
	o.emit(protocol.EventGateMeasured, protocol.GateMeasured{ChildID: id, GateResult: r})
	switch r.Result {
	case GateGreen:
		o.propose(id)
	case GateUndeclared:
		o.reportChild(id, "its review approved, and no gate is declared (orchestrator.gate_command, verify.command): nothing was measured, and nothing merges.")
	default:
		o.giveBack(id, protocol.Return{Why: returnGateRed, Phase: phaseFix}, true, fmt.Sprintf("The harness ran the gate "+
			"`%s` on your commit %s and it is %s (exit %d). Fix what it shows and end your turn; the pipeline "+
			"reviews and measures again.\n\n%s", r.Command, short(r.Commit), r.Result, r.ExitCode, outputOf(r)))
	}
}

// conflictText is the turn that hands back a branch main does not merge into.
func conflictText(paths []string) string {
	return "Main moved, and merging it into your branch conflicts in " + strings.Join(paths, ", ") + ". " +
		"The merge is in progress in your worktree with the conflict markers in those files. Resolve them, " +
		"keeping what main changed and what your task needs, and end your turn: the harness concludes the merge " +
		"with its commit, and the pipeline reviews and measures it again. Run no git command yourself."
}

// RedTest is a child's red_test, handed to the project it is a child of.
func (r *Registry) RedTest(childID, command string) (string, error) {
	r.mu.Lock()
	var owner *Orchestrator
	for _, o := range r.projects {
		if o.child(childID).ID != "" {
			owner = o
		}
	}
	r.mu.Unlock()
	if owner == nil {
		return "", fmt.Errorf("session %s is no child of a project this daemon conducts", childID)
	}
	return owner.RedTest(childID, command)
}

// outputOf is a measurement's output for a turn the model reads.
func outputOf(r protocol.GateResult) string {
	if strings.TrimSpace(r.OutputTail) == "" {
		return "(no output)"
	}
	return "The end of its output:\n\n```\n" + strings.TrimRight(r.OutputTail, "\n") + "\n```"
}

// resume takes the pipeline up again after a restart, from the step the state
// shows: a review or a gate that was running runs again; only a finished
// measurement counts (.p §8).
func (o *Orchestrator) resume() {
	for _, c := range o.snapshot().Children {
		if c.State != stateInReview {
			continue
		}
		id := c.ID
		if c.Review != nil && c.Review.Verdict == tools.VerdictApprove && c.Review.Commit == c.Head {
			switch {
			case c.Gate == nil || c.Gate.Commit != c.Head:
				o.reg.async(func() { o.measure(id) })
			case c.Gate.Result == GateGreen:
				o.reg.async(func() { o.propose(id) })
			}
			continue
		}
		o.reg.async(func() { o.startReview(id, 1) })
	}
}
