package orchestrate

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/aguinelo/dcode/internal/protocol"
)

func (p *project) reported(t *testing.T, want string) {
	t.Helper()
	if got := strings.Join(p.o.snapshot().PendingReports, "\n"); !strings.Contains(got, want) {
		t.Errorf("no report saying %q:\n%s", want, got)
	}
}

// A reviewer that cannot be built stops the pipeline, said to the person and
// to the orchestrator, and the child stays in_review.
func TestAReviewerThatCannotBeBuiltStopsThePipelineAndSaysWhy(t *testing.T) {
	p := openWith(t, nil)
	c := p.one(t, item("index", "internal/index"))
	p.sp.failReviewer = errors.New("no models.toml")
	put(t, c, "internal/index/i.go", "package index\n")
	p.end(t, c.ID, protocol.StopDone, "Done.")
	if got := p.o.child(c.ID); got.State != stateInReview {
		t.Fatalf("child = %s", got.State)
	}
	p.reported(t, "no models.toml")
	if p.w.count(protocol.EventSessionError) == 0 {
		t.Error("the stop was not said to the person")
	}
}

// A reviewer whose first turn does not start is closed, and the pipeline stops.
func TestAReviewerWhoseTurnDoesNotStartIsClosed(t *testing.T) {
	p := openWith(t, nil)
	c := p.one(t, item("index", "internal/index"))
	p.sp.failSubmit = errors.New("the model is down")
	put(t, c, "internal/index/i.go", "package index\n")
	p.end(t, c.ID, protocol.StopDone, "Done.")
	reqs := p.reviewerReqs()
	if len(reqs) != 1 || !strings.Contains(strings.Join(p.sp.closed, " "), reqs[0].ID) {
		t.Fatalf("reviewers %d, closed %v", len(reqs), p.sp.closed)
	}
	if p.o.reviewer(reqs[0].ID) {
		t.Error("the reviewer is still held")
	}
	p.reported(t, "the model is down")
}

// A child that ends done having never changed anything has nothing to review.
func TestADoneTurnWithNothingCommittedFailsIncomplete(t *testing.T) {
	p := openWith(t, nil)
	c := p.one(t, item("index", "internal/index"))
	p.end(t, c.ID, protocol.StopDone, "Nothing to do.")
	if got := p.o.child(c.ID); got.State != stateFailed || got.Reason != protocol.StopIncomplete {
		t.Fatalf("child = %s/%s", got.State, got.Reason)
	}
	p.reported(t, "nothing to review")
}

// Spec: orchestration .p §10 — ask refuses what is no question, and the routes'
// half finds the project's questions and answers them.
func TestAskRefusesWhatIsNoQuestionAndTheRouteAnswersIt(t *testing.T) {
	p := openWith(t, map[string]string{"orchestrator.report_debounce": "1m"})
	two := []protocol.QuestionOption{{ID: "a", Label: "A"}, {ID: "b", Label: "B"}}
	for name, call := range map[string]func() (string, error){
		"no text":    func() (string, error) { return p.o.Ask("", two, "a", "w", "") },
		"one option": func() (string, error) { return p.o.Ask("x?", two[:1], "a", "w", "") },
		"no label": func() (string, error) {
			return p.o.Ask("x?", []protocol.QuestionOption{{ID: "a"}, {ID: "b", Label: "B"}}, "a", "w", "")
		},
		"twice an id":  func() (string, error) { return p.o.Ask("x?", []protocol.QuestionOption{two[0], two[0]}, "a", "w", "") },
		"off the list": func() (string, error) { return p.o.Ask("x?", two, "c", "w", "") },
		"no child":     func() (string, error) { return p.o.Ask("x?", two, "a", "w", "nobody") },
	} {
		if _, err := call(); err == nil {
			t.Errorf("%s: asked", name)
		}
	}
	id, err := p.o.Ask("Which store?", two, "a", "fewer moving parts", "")
	if err != nil {
		t.Fatal(err)
	}
	orq := p.o.orchestratorID()
	if qs, held := p.reg.QuestionsOf(orq); !held || len(qs) != 1 || qs[0].Kind != "design" {
		t.Fatalf("QuestionsOf = %+v, %v", qs, held)
	}
	if _, held := p.reg.QuestionsOf("nobody"); held {
		t.Error("a session that conducts nothing has questions")
	}
	if _, err := p.reg.AnswerOf("nobody", id, "a", ""); err == nil {
		t.Error("an answer to a session that conducts nothing was taken")
	}
	if _, err := p.reg.AnswerOf(orq, id, "", ""); err == nil {
		t.Error("an empty answer was taken")
	}
	if q, err := p.reg.AnswerOf(orq, id, "b", "fine"); err != nil || q.Answer != "b" {
		t.Fatalf("AnswerOf = %+v, %v", q, err)
	}
	p.reported(t, `with "b"`)
}

// red_test reaches the child's project through the registry, and only in
// the red phase.
func TestRedTestReachesTheProjectOnlyInTheRedPhase(t *testing.T) {
	p := openWith(t, nil)
	c := p.one(t, item("index", "internal/index"))
	if _, err := p.reg.RedTest("nobody", "true"); err == nil {
		t.Error("a session that is no child named a red test")
	}
	if _, err := p.reg.RedTest(c.ID, "true"); err == nil {
		t.Error("a child outside its red phase named a red test")
	}
}

// Spec: orchestration .p §8 — the pipeline takes up again after a restart: a
// review that was running runs again, and an approval not yet measured is.
func TestThePipelineTakesUpAgainAfterARestart(t *testing.T) {
	keys := map[string]string{"orchestrator.gate_command": "true"}
	p := openWith(t, keys)
	a := p.one(t, item("a", "a"))
	b := p.one(t, item("b", "b"))
	put(t, a, "a/x", "x\n")
	put(t, b, "b/x", "x\n")
	p.end(t, a.ID, protocol.StopDone, "Done.")
	p.end(t, b.ID, protocol.StopDone, "Done.")
	v := protocol.Verdict{Verdict: "approve", Commit: p.o.child(b.ID).Head}
	if err := p.o.apply(edit(b.ID, func(c *protocol.Child) { c.Review = &v })); err != nil {
		t.Fatal(err)
	}
	p.reg.Close()

	again := p.registry(t, keys)
	if errs := again.Reload(t.Context()); len(errs) != 0 {
		t.Fatalf("Reload: %v", errs)
	}
	o, _ := again.Get(p.o.Project().ID)
	if n := len(p.reviewerReqs()); n != 3 {
		t.Errorf("after the restart %d reviewers ran, want the running review again (3)", n)
	}
	if got := o.child(b.ID); got.Gate == nil || got.Gate.Result != GateGreen || got.State != statePR {
		t.Errorf("the approved child after the restart: %s, gate %+v", got.State, got.Gate)
	}
}

// Spec: orchestration .p §7.6 — with main red the queue stops: the child the
// person said merge to waits in pr, queued, and is told why.
func TestARedMainStopsTheQueueAndTheChildWaits(t *testing.T) {
	// Main's health on a local forge is the gate on main: main lacks the
	// child's file, so main is red while the child's commit is green.
	p := openWith(t, map[string]string{"orchestrator.gate_command": "test -f internal/index/i.go"})
	c, q := proposed(t, p)
	if _, err := p.o.Answer(q.ID, "merge", ""); err != nil {
		t.Fatal(err)
	}
	got := p.o.child(c.ID)
	if got.State != statePR || len(p.o.snapshot().Queue) != 1 || p.o.snapshot().Main.Health != healthRed {
		t.Fatalf("with main red the child is %s, queue %v, main %+v", got.State, p.o.snapshot().Queue, p.o.snapshot().Main)
	}
	p.reported(t, "waits")
	if !strings.Contains(p.kinds(t), "merge.refused") {
		t.Errorf("log = %s", p.kinds(t))
	}
}

// Spec: orchestration .p §7.6 — main that moved between the gate and the
// merge is gate_stale: the child goes back to in_review, is measured again
// over the new main, and is proposed again.
func TestAMainThatMovedAfterTheGateIsMeasuredAgain(t *testing.T) {
	p := openWith(t, map[string]string{"orchestrator.gate_command": "true"})
	c, q := proposed(t, p)
	gitOut(t, p.root, "checkout", "-q", "main")
	put(t, protocol.Child{Worktree: p.root}, "docs/other.md", "main moves on\n")
	gitOut(t, p.root, "add", "docs/other.md")
	gitOut(t, p.root, "commit", "-q", "-m", "main moves")
	gitOut(t, p.root, "checkout", "-q", "--detach")
	if _, err := p.o.Answer(q.ID, "merge", ""); err != nil {
		t.Fatal(err)
	}
	got := p.o.child(c.ID)
	if got.State != statePR || got.Gate == nil || got.Gate.Commit == c.Gate.Commit {
		t.Fatalf("after gate_stale the child is %s, gate %+v (was %s)", got.State, got.Gate, c.Gate.Commit)
	}
	merges := 0
	for _, x := range p.o.Questions() {
		if x.Kind == "merge" && x.ChildID == c.ID {
			merges++
		}
	}
	if merges != 2 {
		t.Errorf("%d merge questions, want the stale one and a new one", merges)
	}
	p.reported(t, "stale")
}

// Spec: orchestration .i step 9 — no gate declared: the approval is measured
// as undeclared, reported, and nothing is proposed.
func TestAnApprovalWithNoGateDeclaredIsReportedAndNothingFollows(t *testing.T) {
	p := openWith(t, nil)
	c := p.one(t, item("index", "internal/index"))
	put(t, c, "internal/index/i.go", "package index\n")
	p.end(t, c.ID, protocol.StopDone, "Done.")
	p.review(t, approve)
	got := p.o.child(c.ID)
	if got.State != stateInReview || got.Gate == nil || got.Gate.Result != GateUndeclared {
		t.Fatalf("child = %s, gate %+v", got.State, got.Gate)
	}
	p.reported(t, "no gate is declared")
	if strings.Contains(p.kinds(t), "question.asked") {
		t.Error("an unmeasured child was proposed")
	}
}

// Spec: orchestration .p §6 — the person's close of the merge question closes
// the child.
func TestThePersonsCloseClosesTheChild(t *testing.T) {
	p := openWith(t, map[string]string{"orchestrator.gate_command": "true"})
	c, q := proposed(t, p)
	if _, err := p.o.Answer(q.ID, "close", ""); err != nil {
		t.Fatal(err)
	}
	if got := p.o.child(c.ID); got.State != stateClosed {
		t.Errorf("after close the child is %s", got.State)
	}
}

// A hand-back whose turn cannot start fails the child, said.
func TestAHandBackThatCannotStartFailsTheChild(t *testing.T) {
	p := openWith(t, nil)
	c := p.one(t, item("index", "internal/index"))
	put(t, c, "README.md", "outside\n")
	p.sp.failSubmit = errors.New("gone")
	p.end(t, c.ID, protocol.StopDone, "Done.")
	if got := p.o.child(c.ID); got.State != stateFailed {
		t.Fatalf("child = %s", got.State)
	}
	p.reported(t, "could not start")
}

// Spec: orchestration .p §8 — a line that does not write stops the step: the
// child does not move, and the failure is said.
func TestAPipelineStepWhoseLineDoesNotWriteMovesNothing(t *testing.T) {
	p := openWith(t, map[string]string{"orchestrator.gate_command": "true"})
	c := p.one(t, item("index", "internal/index"))
	put(t, c, "internal/index/i.go", "package index\n")
	p.end(t, c.ID, protocol.StopDone, "Done.")
	if err := p.o.log.Close(); err != nil {
		t.Fatal(err)
	}
	errorsBefore := p.w.count(protocol.EventSessionError)
	p.review(t, approve)
	p.o.giveBack(c.ID, protocol.Return{Why: returnReview}, true, "again")
	p.o.measure(c.ID)
	p.o.propose(c.ID)
	if _, err := p.o.Ask("x?", []protocol.QuestionOption{{ID: "a", Label: "A"}, {ID: "b", Label: "B"}}, "a", "w", ""); err == nil {
		t.Error("a question was asked with no log")
	}
	if got := p.o.child(c.ID); got.State != stateInReview || got.Returned != 0 {
		t.Errorf("with no log the child moved: %s, returned %d", got.State, got.Returned)
	}
	if p.w.count(protocol.EventSessionError) <= errorsBefore {
		t.Error("the failures were not said")
	}
}

// A state that cannot be saved is said, and the report is said instead of kept.
func TestAReportThatCannotBeKeptIsSaid(t *testing.T) {
	p := openWith(t, nil)
	dir := filepath.Join(p.dir, p.o.Project().ID)
	if err := os.Chmod(dir, 0o500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(dir, 0o700) })
	p.o.Report("Child x: it failed.")
	if len(p.o.snapshot().PendingReports) != 0 || p.w.count(protocol.EventSessionError) != 1 {
		t.Errorf("reports %q, errors %d", p.o.snapshot().PendingReports, p.w.count(protocol.EventSessionError))
	}
}

// The reviewer runs on the first of reviewer_profiles, else the first child
// profile, else the orchestrator's own model.
func TestTheReviewerProfileFallsBackInOrder(t *testing.T) {
	p := openWith(t, nil)
	if got, err := p.o.reviewerProfile(); err != nil || got != "" {
		t.Errorf("nothing declared = %q, %v", got, err)
	}
	p.o.limits.ChildProfiles = []string{"child"}
	if got, err := p.o.reviewerProfile(); err != nil || got != "child" {
		t.Errorf("child profiles = %q, %v", got, err)
	}
	p.o.review.Profiles = []string{"strong"}
	if got, err := p.o.reviewerProfile(); err != nil || got != "strong" {
		t.Errorf("reviewer profiles = %q, %v", got, err)
	}
}

// A report the orchestrator's turn cannot take for another reason than a
// running turn stays in the inbox, said.
func TestAReportTurnThatDoesNotStartStaysInTheInbox(t *testing.T) {
	p := openWith(t, map[string]string{"orchestrator.report_debounce": "1m"})
	p.o.Report("Child x: it failed.")
	p.sp.failSubmit = errors.New("the session is gone")
	p.o.flushReports()
	if len(p.o.snapshot().PendingReports) != 1 || p.w.count(protocol.EventSessionError) != 1 {
		t.Errorf("reports %q, errors %d", p.o.snapshot().PendingReports, p.w.count(protocol.EventSessionError))
	}
}

// A conclusion past DelegateMaxResultBytes is cut on a whole character and
// said truncated; the claim the gate keeps is the cut one.
func TestALongConclusionIsCutAndSaid(t *testing.T) {
	p := openWith(t, nil)
	p.reg.conclusionMax = 5
	c := p.one(t, item("index", "internal/index"))
	p.end(t, c.ID, protocol.StopError, "abcdé and much more")
	cc, ok := p.w.last[protocol.EventChildCompleted].(protocol.ChildCompleted)
	if !ok || !cc.Truncated || cc.Conclusion != "abcd" || p.o.brief(c.ID).Claim != "abcd" {
		t.Errorf("child.completed = %+v, claim %q", cc, p.o.brief(c.ID).Claim)
	}
}

// Session events of nobody the registry holds are let go, and a child's
// end that does not read is said.
func TestEventsOfNobodyAreLetGoAndAnUnreadableEndIsSaid(t *testing.T) {
	p := openWith(t, nil)
	p.reg.Observe("nobody", event(t, protocol.EventTurnCompleted, protocol.TurnCompleted{}))
	c := p.one(t, item("index", "internal/index"))
	p.reg.Observe(c.ID, protocol.Event{Type: protocol.EventTurnCompleted, Payload: []byte("{")})
	if p.w.count(protocol.EventSessionError) != 1 {
		t.Errorf("an unreadable end was not said: %d errors", p.w.count(protocol.EventSessionError))
	}
}

// A close the daemon refuses is said; an empty report is no report.
func TestAPipelineSessionThatDoesNotCloseIsSaid(t *testing.T) {
	p := openWith(t, nil)
	p.sp.closeErr = errors.New("busy")
	p.o.closeSession("r1")
	if p.w.count(protocol.EventSessionError) != 1 {
		t.Error("a close that failed was not said")
	}
	p.o.Report("  ")
	if len(p.o.snapshot().PendingReports) != 0 {
		t.Error("an empty report was kept")
	}
}
