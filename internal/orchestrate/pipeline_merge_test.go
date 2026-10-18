package orchestrate

import (
	"os"
	"strings"
	"testing"

	"github.com/aguinelo/dcode/internal/protocol"
)

// proposed is a child of p taken through the pipeline to its merge question,
// with main free to move: the person's tree is detached, as a local merge
// requires.
func proposed(t *testing.T, p *project) (protocol.Child, protocol.Question) {
	t.Helper()
	gitOut(t, p.root, "checkout", "-q", "--detach")
	it := item("index", "internal/index")
	it.PRTitle = "feat(index): count words"
	c := p.one(t, it)
	put(t, c, "internal/index/i.go", "package index\n")
	p.end(t, c.ID, protocol.StopDone, "Done.")
	p.review(t, approve)
	got := p.o.child(c.ID)
	if got.State != statePR || got.Waiting != "merge" || got.Gate == nil || got.Gate.Result != GateGreen {
		t.Fatalf("after a green gate the child is %s waiting %q, gate %+v", got.State, got.Waiting, got.Gate)
	}
	var q protocol.Question
	for _, x := range p.o.Questions() {
		if x.Kind == "merge" && x.ChildID == c.ID {
			q = x
		}
	}
	if q.ID == "" {
		t.Fatalf("no merge question: %+v", p.o.Questions())
	}
	return got, q
}

// Spec: orchestration .p §7.5, §7.6 — a child measured green is proposed and
// asked about; nothing merges before the person's merge; after it the queue
// merges the measured commit, and the child is merged with its worktree gone.
func TestAGreenChildMergesOnlyAfterThePersonsYes(t *testing.T) {
	p := openWith(t, map[string]string{"orchestrator.gate_command": "true"})
	before := strings.TrimSpace(gitOut(t, p.root, "rev-parse", "main"))
	c, q := proposed(t, p)
	if k := p.kinds(t); strings.Contains(k, "merge.queued") || strings.Contains(k, "merge.completed") {
		t.Fatalf("a merge before the person's answer: %s", k)
	}
	if strings.TrimSpace(gitOut(t, p.root, "rev-parse", "main")) != before {
		t.Fatal("main moved before the person answered")
	}
	if _, err := p.o.Answer(q.ID, "merge", ""); err != nil {
		t.Fatal(err)
	}
	got := p.o.child(c.ID)
	if got.State != stateMerged {
		t.Fatalf("after the yes the child is %s; log %s", got.State, p.kinds(t))
	}
	if strings.TrimSpace(gitOut(t, p.root, "rev-parse", "main")) == before {
		t.Error("main did not move")
	}
	if _, err := os.Stat(c.Worktree); !os.IsNotExist(err) {
		t.Errorf("the merged child's worktree stays: %v", err)
	}
	k := p.kinds(t)
	for _, want := range []string{"gate.measured", "question.asked", "question.answered", "merge.queued", "merge.completed"} {
		if !strings.Contains(k, want) {
			t.Errorf("log lacks %s: %s", want, k)
		}
	}
	if st := p.o.snapshot(); len(st.Queue) != 0 || st.Project.Forge != ForgeLocal || !st.LocalSaid {
		t.Errorf("state after the merge: queue %v, forge %q, local said %v", st.Queue, st.Project.Forge, st.LocalSaid)
	}
	if _, err := p.o.Answer(q.ID, "merge", ""); err == nil {
		t.Error("a second answer was taken")
	}
}

// Spec: orchestration .p §6 — the person's return hands the work back to the
// child, said as merge.refused.
func TestThePersonsReturnHandsTheWorkBack(t *testing.T) {
	p := openWith(t, map[string]string{"orchestrator.gate_command": "true"})
	c, q := proposed(t, p)
	if _, err := p.o.Answer(q.ID, "return", "name it Count"); err != nil {
		t.Fatal(err)
	}
	got := p.o.child(c.ID)
	if got.State != stateWorking || got.Return == nil || got.Return.Why != "person" {
		t.Fatalf("after return the child is %s with %+v", got.State, got.Return)
	}
	if turns := p.toChild(c.ID); !strings.Contains(turns[len(turns)-1], "name it Count") {
		t.Errorf("the hand-back said %q", turns[len(turns)-1])
	}
	if !strings.Contains(p.kinds(t), "merge.refused") {
		t.Errorf("log = %s", p.kinds(t))
	}
}

// Spec: orchestration .p §8 — a merge question pending when the daemon stops
// is pending after it starts again, and forge.local is said once a project.
func TestAMergeQuestionSurvivesARestart(t *testing.T) {
	keys := map[string]string{"orchestrator.gate_command": "true"}
	p := openWith(t, keys)
	c, q := proposed(t, p)
	p.reg.Close()

	again := p.registry(t, keys)
	if errs := again.Reload(t.Context()); len(errs) != 0 {
		t.Fatalf("Reload: %v", errs)
	}
	o, _ := again.Get(p.o.Project().ID)
	if _, err := o.Answer(q.ID, "merge", ""); err != nil {
		t.Fatalf("the question did not survive the restart: %v", err)
	}
	if got := o.child(c.ID); got.State != stateMerged {
		t.Errorf("after the restart and the yes the child is %s", got.State)
	}
	if n := strings.Count(p.kinds(t), "forge.local"); n != 1 {
		t.Errorf("forge.local said %d times", n)
	}
}

// Spec: orchestration .i step 9 — gate.measured.commit is the H that holds
// main's tip, and base is that tip: a main that moved is merged into the
// branch first, and the child's branch moves forward to H.
func TestTheGateMeasuresTheCommitThatHoldsMainAsItIsNow(t *testing.T) {
	p := openWith(t, map[string]string{"orchestrator.gate_command": "true"})
	c := p.one(t, item("index", "internal/index"))
	put(t, c, "internal/index/i.go", "package index\n")
	p.end(t, c.ID, protocol.StopDone, "Done.")
	put(t, protocol.Child{Worktree: p.root}, "docs/other.md", "main moves on\n")
	gitOut(t, p.root, "add", "docs/other.md")
	gitOut(t, p.root, "commit", "-q", "-m", "main moves")
	tip := strings.TrimSpace(gitOut(t, p.root, "rev-parse", "main"))
	p.review(t, approve)

	got := p.o.child(c.ID)
	if got.Gate == nil || got.Gate.Base != tip || got.Gate.Commit != got.Head || got.Gate.Tree == "" {
		t.Fatalf("gate = %+v, head %s, main %s", got.Gate, got.Head, tip)
	}
	gitOut(t, p.root, "merge-base", "--is-ancestor", tip, got.Gate.Commit)
	if branch := strings.TrimSpace(gitOut(t, p.root, "rev-parse", got.Branch)); branch != got.Gate.Commit {
		t.Errorf("the branch is at %s, the gate measured %s", branch, got.Gate.Commit)
	}
	if !strings.Contains(p.kinds(t), "branch.updated gate.measured") {
		t.Errorf("log = %s", p.kinds(t))
	}
}

// Spec: orchestration .i step 9 — a main that moved and conflicts goes back
// to the child that owns the branch with the paths; what main brought outside
// the child's owns is not the child's, and the resolution is committed as the
// merge, then reviewed again.
func TestAConflictWithMainGoesBackToTheChildAndItsResolutionIsReviewed(t *testing.T) {
	p := openWith(t, map[string]string{"orchestrator.gate_command": "true"})
	c := p.one(t, item("index", "internal/index"))
	put(t, c, "internal/index/i.go", "package index // the child's\n")
	p.end(t, c.ID, protocol.StopDone, "Done.")

	main := protocol.Child{Worktree: p.root}
	put(t, main, "internal/index/i.go", "package index // main's\n")
	put(t, main, "docs/other.md", "main's own change\n")
	gitOut(t, p.root, "add", "internal/index/i.go", "docs/other.md")
	gitOut(t, p.root, "commit", "-q", "-m", "main moves")
	p.review(t, approve)

	got := p.o.child(c.ID)
	if got.State != stateWorking || got.Return == nil || got.Return.Why != "conflict" ||
		strings.Join(got.Return.Paths, ",") != "internal/index/i.go" {
		t.Fatalf("after the conflict the child is %s with %+v", got.State, got.Return)
	}
	put(t, c, "internal/index/i.go", "package index // both\n")
	p.end(t, c.ID, protocol.StopDone, "Resolved.")
	after := p.o.child(c.ID)
	if after.State != stateInReview || len(p.reviewerReqs()) != 2 {
		for _, d := range p.lines(t, "child.returned") {
			t.Logf("returned: %s %s", d.Why, d.Data)
		}
		for _, d := range p.lines(t, "owns.checked") {
			t.Logf("owns: %s %s", d.Why, d.Data)
		}
		t.Fatalf("the resolution was not reviewed: %s, %d reviewers; log %s", after.State, len(p.reviewerReqs()), p.kinds(t))
	}
	if parents := strings.Fields(gitOut(t, c.Worktree, "log", "-1", "--format=%P")); len(parents) != 2 {
		t.Errorf("the resolution is not the merge's commit: parents %v", parents)
	}
}
