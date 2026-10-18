package orchestrate

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/aguinelo/dcode/internal/protocol"
	"github.com/aguinelo/dcode/internal/tools"
)

// put writes rel in the child's worktree, as the child would.
func put(t *testing.T, c protocol.Child, rel, content string) {
	t.Helper()
	path := filepath.Join(c.Worktree, rel)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// end is one turn of the child: it says said, and ends with reason.
func (p *project) end(t *testing.T, id, reason, said string) {
	t.Helper()
	p.reg.Observe(id, event(t, protocol.EventTurnStarted, protocol.TurnStarted{TurnID: "t"}))
	p.reg.Observe(id, event(t, protocol.EventMessageDelta, protocol.MessageDelta{Text: said}))
	p.reg.Observe(id, event(t, protocol.EventTurnCompleted, protocol.TurnCompleted{TurnID: "t", Reason: reason}))
}

func (p *project) reviewerReqs() []ReviewRequest {
	p.sp.mu.Lock()
	defer p.sp.mu.Unlock()
	return append([]ReviewRequest(nil), p.sp.reviewers...)
}

// review is the last reviewer's turn: verdict when one is given, then its end.
func (p *project) review(t *testing.T, in *tools.VerdictInput) {
	t.Helper()
	reqs := p.reviewerReqs()
	if len(reqs) == 0 {
		t.Fatal("no reviewer was spawned")
	}
	req := reqs[len(reqs)-1]
	if in != nil {
		if _, err := req.Submit(context.Background(), *in); err != nil {
			t.Fatalf("verdict: %v", err)
		}
	}
	p.reg.Observe(req.ID, event(t, protocol.EventTurnCompleted, protocol.TurnCompleted{TurnID: "r", Reason: protocol.StopDone}))
}

var (
	approve = &tools.VerdictInput{Verdict: tools.VerdictApprove, Why: "it does what the task asks"}
	changes = &tools.VerdictInput{Verdict: tools.VerdictChanges, Why: "a defect",
		Findings: []tools.VerdictFinding{{Path: "internal/index/i.go", What: "counts zero", Why: "the task asks for the count"}}}
)

// toChild is every turn submitted to session id, in order.
func (p *project) toChild(id string) []string {
	p.sp.mu.Lock()
	defer p.sp.mu.Unlock()
	var out []string
	for _, s := range p.sp.submitted {
		if strings.HasPrefix(s, id+": ") {
			out = append(out, strings.TrimPrefix(s, id+": "))
		}
	}
	return out
}

func (p *project) kinds(t *testing.T) string {
	return strings.Join(kindsIn(t, p.dir, p.o.Project().ID), " ")
}

// lines are the log's decisions of kind.
func (p *project) lines(t *testing.T, kind string) []Decision {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(p.dir, p.o.Project().ID, decisionsFile))
	if err != nil {
		t.Fatal(err)
	}
	var out []Decision
	for _, l := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		var d Decision
		if json.Unmarshal([]byte(l), &d) == nil && d.Kind == kind {
			out = append(out, d)
		}
	}
	return out
}

// Spec: orchestration .i step 8 — a diff with a file outside owns goes back
// as Return{outside_owns} without a commit and without a reviewer session.
func TestADiffOutsideOwnsIsNeverCommittedAndReturnsWithoutAReviewer(t *testing.T) {
	p := openWith(t, nil)
	c := p.one(t, item("index", "internal/index"))
	put(t, c, "internal/index/i.go", "package index\n")
	put(t, c, "README.md", "written through bash\n")
	p.end(t, c.ID, protocol.StopDone, "Done.")

	got := p.o.child(c.ID)
	if got.Head != "" || strings.TrimSpace(gitOut(t, c.Worktree, "rev-parse", "HEAD")) != c.Base {
		t.Fatalf("something was committed: head %q", got.Head)
	}
	if len(p.reviewerReqs()) != 0 {
		t.Fatal("a reviewer was spawned for a diff outside owns")
	}
	if got.State != stateWorking || got.Return == nil || got.Return.Why != protocol.CodeOutsideOwns ||
		strings.Join(got.Return.Paths, ",") != "README.md" || got.Returned != 1 {
		t.Fatalf("child = %+v, return %+v", got, got.Return)
	}
	turns := p.toChild(c.ID)
	if last := turns[len(turns)-1]; !strings.Contains(last, "README.md") || !strings.Contains(last, "did not commit") {
		t.Errorf("the hand-back said %q", last)
	}
	if k := p.kinds(t); !strings.Contains(k, "owns.checked child.completed") || !strings.Contains(k, "child.returned") {
		t.Errorf("log = %s", k)
	}
}

// Spec: orchestration .i step 8 — missing runs once more; the second leaves
// the child in_review, reported to the orchestrator.
func TestAReviewWithoutAVerdictRunsOnceMoreThenStaysInReview(t *testing.T) {
	p := openWith(t, nil)
	c := p.one(t, item("index", "internal/index"))
	put(t, c, "internal/index/i.go", "package index\n")
	p.end(t, c.ID, protocol.StopDone, "Done.")
	p.review(t, nil)
	if n := len(p.reviewerReqs()); n != 2 {
		t.Fatalf("after one missing verdict %d reviewers ran, want 2", n)
	}
	p.review(t, nil)
	reqs := p.reviewerReqs()
	if len(reqs) != 2 {
		t.Fatalf("after two missing verdicts %d reviewers ran, want no third", len(reqs))
	}
	got := p.o.child(c.ID)
	if got.State != stateInReview || got.Review == nil || got.Review.Verdict != VerdictMissing {
		t.Fatalf("child = %s, review %+v", got.State, got.Review)
	}
	if n := len(p.lines(t, "review.verdict")); n != 2 || p.w.count(protocol.EventReviewVerdict) != 2 {
		t.Errorf("review.verdict: %d lines, %d events", n, p.w.count(protocol.EventReviewVerdict))
	}
	closed := strings.Join(p.sp.closed, " ")
	if !strings.Contains(closed, reqs[0].ID) || !strings.Contains(closed, reqs[1].ID) {
		t.Errorf("reviewers left open: closed %s", closed)
	}
	if reqs[0].Prompt() == "" || reqs[0].Commit != got.Head || reqs[0].Worktree != c.Worktree || reqs[0].ParentID != c.ParentID {
		t.Errorf("the reviewer was given %+v", reqs[0])
	}
	pending := strings.Join(p.o.snapshot().PendingReports, "\n")
	if !strings.Contains(pending, "without a verdict") {
		t.Errorf("no report of the missing verdicts: %q", pending)
	}
}

// Spec: orchestration .i step 8 — a red-phase test that passes is not taken as
// red, and the child goes back to the red phase; a fix handed in without the
// red phase confirmed never reaches the gate.
func TestARedTestThatPassesIsNotTakenAndAnUnconfirmedFixNeverReachesTheGate(t *testing.T) {
	p := openWith(t, map[string]string{"orchestrator.gate_command": "true", "orchestrator.max_returns": "5"})
	c := p.one(t, item("index", "internal/index"))
	put(t, c, "internal/index/i.go", "package index\n")
	p.end(t, c.ID, protocol.StopDone, "Done.")
	p.review(t, changes)
	if got := p.o.child(c.ID); got.Return == nil || got.Return.Phase != phaseRed || got.Return.Why != returnReview {
		t.Fatalf("after changes_requested the child is %+v", got.Return)
	}

	put(t, c, "internal/index/i.go", "package index\n\nfunc Count() int { return 1 }\n")
	p.end(t, c.ID, protocol.StopDone, "Fixed it.")
	if got := p.o.child(c.ID); got.Return.Phase != phaseRed || got.Returned != 2 {
		t.Fatalf("a fix without red_test went on: %+v, returned %d", got.Return, got.Returned)
	}

	if _, err := p.o.RedTest(c.ID, "true"); err != nil {
		t.Fatal(err)
	}
	put(t, c, "internal/index/i_test.go", "package index\n")
	p.end(t, c.ID, protocol.StopDone, "The test is red.")
	got := p.o.child(c.ID)
	if got.Return.Phase != phaseRed || got.Returned != 3 {
		t.Fatalf("a red test that passes was taken: %+v, returned %d", got.Return, got.Returned)
	}
	if n := len(p.lines(t, "red_test.confirmed")) + len(p.lines(t, "gate.measured")); n != 0 {
		t.Errorf("%d red_test.confirmed or gate.measured lines without a red test that fails", n)
	}
	if turns := p.toChild(c.ID); !strings.Contains(turns[len(turns)-1], "did not fail") {
		t.Errorf("the hand-back said %q", turns[len(turns)-1])
	}
	if _, err := p.o.RedTest(c.ID, ""); err == nil {
		t.Error("red_test without a command was taken")
	}
}

// Spec: orchestration .i step 8 — the red test must fail without the fix:
// red_test.confirmed on the test's commit, the fix phase, and then the
// pipeline from the owns check again.
func TestTheRedTestMustFailWithoutTheFixBeforeTheFixIsTaken(t *testing.T) {
	p := openWith(t, map[string]string{"orchestrator.gate_command": "true"})
	c := p.one(t, item("index", "internal/index"))
	put(t, c, "internal/index/i.go", "package index\n")
	p.end(t, c.ID, protocol.StopDone, "Done.")
	p.review(t, changes)

	if _, err := p.o.RedTest(c.ID, "test -f internal/index/fixed"); err != nil {
		t.Fatal(err)
	}
	put(t, c, "internal/index/i_test.go", "package index\n")
	p.end(t, c.ID, protocol.StopDone, "Red.")
	got := p.o.child(c.ID)
	if got.Return.Phase != phaseFix || got.Returned != 1 {
		t.Fatalf("after a red test that fails: %+v, returned %d", got.Return, got.Returned)
	}
	confirmed := p.lines(t, "red_test.confirmed")
	if len(confirmed) != 1 || !strings.Contains(string(confirmed[0].Measured), got.Head) {
		t.Fatalf("red_test.confirmed = %+v, want one on %s", confirmed, got.Head)
	}

	put(t, c, "internal/index/fixed", "yes\n")
	p.end(t, c.ID, protocol.StopDone, "Fixed.")
	if n := len(p.reviewerReqs()); n != 2 {
		t.Fatalf("the fix was not reviewed again: %d reviewers", n)
	}
	p.review(t, approve)
	if got := p.o.child(c.ID); got.Gate == nil || got.Gate.Result != GateGreen {
		t.Errorf("after the approval the gate is %+v", got.Gate)
	}
}

// Spec: orchestration .i step 9 — gate.measured.commit is the child's exact
// commit and base the main tip under it; a child that claims its gate passed
// is recorded red beside its claim, and goes back in the fix phase.
func TestTheGateMeasuresTheExactCommitAndALyingChildIsRecordedRed(t *testing.T) {
	p := openWith(t, map[string]string{"orchestrator.gate_command": "echo broken; exit 3"})
	c := p.one(t, item("index", "internal/index"))
	put(t, c, "internal/index/i.go", "package index\n")
	p.end(t, c.ID, protocol.StopDone, "Done; make check passed.")
	p.review(t, approve)

	got := p.o.child(c.ID)
	lines := p.lines(t, "gate.measured")
	if len(lines) != 1 {
		t.Fatalf("%d gate.measured lines", len(lines))
	}
	var r protocol.GateResult
	if err := json.Unmarshal(lines[0].Measured, &r); err != nil {
		t.Fatal(err)
	}
	tip := strings.TrimSpace(gitOut(t, p.root, "rev-parse", "main"))
	if r.Commit != got.Head || r.Base != tip || r.Result != GateRed || r.ExitCode != 3 {
		t.Errorf("measured %+v; head %s, main %s", r, got.Head, tip)
	}
	if lines[0].Claim != "Done; make check passed." {
		t.Errorf("claim = %q", lines[0].Claim)
	}
	if got.Return == nil || got.Return.Why != returnGateRed || got.Return.Phase != phaseFix || got.State != stateWorking {
		t.Errorf("after a red gate the child is %s with %+v", got.State, got.Return)
	}
	if turns := p.toChild(c.ID); !strings.Contains(turns[len(turns)-1], "broken") {
		t.Errorf("the hand-back does not carry the gate's output: %q", turns[len(turns)-1])
	}
	if p.w.count(protocol.EventGateMeasured) != 1 {
		t.Errorf("gate.measured events: %d", p.w.count(protocol.EventGateMeasured))
	}
}

// Spec: orchestration .i step 8 — past max_returns the child fails, reason
// incomplete, and the orchestrator is told.
func TestPastMaxReturnsTheChildFailsIncomplete(t *testing.T) {
	p := openWith(t, map[string]string{"orchestrator.max_returns": "1"})
	c := p.one(t, item("index", "internal/index"))
	for i := 0; i < 2; i++ {
		put(t, c, "README.md", "outside\n")
		p.end(t, c.ID, protocol.StopDone, "Done.")
	}
	got := p.o.child(c.ID)
	if got.State != stateFailed || got.Reason != protocol.StopIncomplete || got.Returned != 1 {
		t.Fatalf("child = %s/%s, returned %d", got.State, got.Reason, got.Returned)
	}
	if !strings.Contains(strings.Join(p.o.snapshot().PendingReports, "\n"), "max_returns") {
		t.Errorf("reports = %q", p.o.snapshot().PendingReports)
	}
}

type fakeMerge struct {
	got   []protocol.Child
	brief []Brief
}

func (m *fakeMerge) Ready(_ *Orchestrator, c protocol.Child, b Brief) (protocol.ChildPR, error) {
	m.got, m.brief = append(m.got, c), append(m.brief, b)
	return protocol.ChildPR{Number: 7, URL: "https://example.test/pr/7"}, nil
}

// Spec: orchestration .p §7.5 — review approved and gate green hand the child
// to the merge step, which opens its pull request; the child goes to pr.
func TestAGreenChildIsHandedToTheMergeStepAndWaitsInPR(t *testing.T) {
	m := &fakeMerge{}
	p := openMerging(t, map[string]string{"orchestrator.gate_command": "true"}, m)
	it := item("index", "internal/index")
	it.PRTitle = "feat(index): count words"
	c := p.one(t, it)
	put(t, c, "internal/index/i.go", "package index\n")
	p.end(t, c.ID, protocol.StopDone, "Done.")
	p.review(t, approve)

	if len(m.got) != 1 || m.got[0].Gate == nil || m.got[0].Gate.Commit != m.got[0].Head || m.brief[0].PRTitle != it.PRTitle {
		t.Fatalf("the merge step got %+v / %+v", m.got, m.brief)
	}
	got := p.o.child(c.ID)
	if got.State != statePR || got.PR == nil || got.PR.Number != 7 || got.Waiting != "merge" {
		t.Errorf("child = %s, pr %+v, waiting %q", got.State, got.PR, got.Waiting)
	}
	k := p.kinds(t)
	if i, j := strings.Index(k, "review.verdict"), strings.Index(k, "gate.measured"); i < 0 || j < i {
		t.Errorf("review and gate are out of order: %s", k)
	}
	if strings.Contains(k, "merge.queued") {
		t.Errorf("merge.queued is the person's answer, never the pipeline's: %s", k)
	}
}

// A pipeline step that ends after the child was closed moves nothing: a
// closed child is terminal, and a late failure must not reopen it.
func TestAStepThatEndsAfterACloseMovesNothing(t *testing.T) {
	p := openWith(t, nil)
	c := p.one(t, item("index", "internal/index"))
	if err := p.o.Close(c.ID, "enough"); err != nil {
		t.Fatal(err)
	}
	p.o.move(c.ID, stateFailed, protocol.StopIncomplete, "a late step")
	p.o.giveBack(c.ID, protocol.Return{Why: returnReview}, true, "late")
	p.o.ready(c.ID, false)
	if got := p.o.child(c.ID); got.State != stateClosed || got.Returned != 0 {
		t.Errorf("after the close the child is %s, returned %d", got.State, got.Returned)
	}
}

// waitFor polls until cond holds, or fails after a while.
func waitFor(t *testing.T, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for !cond() {
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for %s", what)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

// reportTurns are the turns the inbox started on the orchestrator.
func (p *project) reportTurns() []string {
	p.sp.mu.Lock()
	defer p.sp.mu.Unlock()
	var out []string
	for i, s := range p.sp.submitted {
		if p.sp.origins[i] == protocol.OriginChildren {
			out = append(out, s)
		}
	}
	return out
}
