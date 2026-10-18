package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/aguinelo/dcode/internal/protocol"
	"github.com/aguinelo/dcode/internal/tools"
)

// The scenarios of the pipeline before the merge: what a child's done turn
// goes through — the owns check, the review, the red phase, the gate — read
// from the log and from git, never from what a model said.
//
// Spec: docs/specs/architecture/orchestration/202610132000-orchestration.i.spec.md, step 12;
// .p §7.1–§7.4.

// item is one entry of a delegate batch; the task carries [[key]], which is
// how the scripted model knows whose script a request is.
func item(key string, owns ...string) map[string]any {
	return map[string]any{"title": key, "task": "[[" + key + "]] the work of " + key, "owns": owns,
		"pr_title": key + ": the work", "why": key + " is a theme of its own"}
}

func dispatch(items ...map[string]any) step {
	return use("delegate", map[string]any{"items": items})
}

func writes(path, content string) step {
	return use("write", map[string]string{"path": path, "content": content})
}

func approve(why string) step {
	return use("verdict", map[string]any{"verdict": tools.VerdictApprove, "why": why})
}

// TestScenarioAChildThatClaimsTheGatePassedIsMeasured: the child says the
// gate passed; the gate the harness runs is red; nothing is proposed, and the
// log keeps the claim beside the measurement.
func TestScenarioAChildThatClaimsTheGatePassedIsMeasured(t *testing.T) {
	s := newScenario(t, scenarioOptions{forge: "github"})
	hold := make(chan struct{})
	s.model.orchestrator(dispatch(item("docs", "docs")), say("Dispatched."))
	s.model.child("docs",
		writes("docs/a.md", "BROKEN\n"),
		say("Done. I ran make check and it passed, all green."),
		say("Looking at the gate's output.").heldUntil(hold))
	s.model.reviewer("docs", approve("the doc reads well"))
	s.open()
	s.ask("Write the docs.")

	c := s.waitState("docs", "working")
	s.waitFor("the gate was never measured", func() bool { return s.count(c.ID, "gate.measured") > 0 })
	s.waitFor("the red gate did not hand the work back", func() bool { return s.count(c.ID, "child.returned") > 0 })

	g := s.gates(c.ID)[0]
	if g.Result != "red" || g.ExitCode == 0 || g.Commit != s.kid("docs").Head {
		t.Errorf("the gate measured %+v, want red on the child's head %s", g, s.kid("docs").Head)
	}
	measured := s.all(c.ID, "gate.measured")[0]
	if !strings.Contains(measured.Claim, "it passed") {
		t.Errorf("the child's claim is not beside the measurement: claim %q", measured.Claim)
	}
	if r := s.returns(c.ID)[0]; r.Why != "gate_red" || r.Phase != "fix" {
		t.Errorf("the red gate handed back %+v, want gate_red in the fix phase", r)
	}
	if n := s.count(c.ID, "pr.opened") + s.count(c.ID, "question.asked"); n != 0 {
		t.Errorf("a red gate opened a PR or asked a merge (%d decisions)", n)
	}
	if calls := s.ghCalls(); strings.Contains(calls, "pr\ncreate") {
		t.Errorf("gh was asked for a PR behind a red gate:\n%s", calls)
	}
	s.inOrder(c.ID, "dispatch", "worktree.created", "owns.checked", "child.completed",
		"review.verdict", "gate.measured", "child.returned")
	close(hold)
	s.closeChild(c.ID)
	s.quiet()
}

// TestScenarioARedGateReturnsToTheSameChild: the red gate goes back to the
// child that made it, in its own session; the fix passes, and the merge goes
// on after the person's yes.
func TestScenarioARedGateReturnsToTheSameChild(t *testing.T) {
	s := newScenario(t, scenarioOptions{forge: "local"})
	s.model.orchestrator(dispatch(item("api", "api")), say("Dispatched."))
	s.model.child("api",
		writes("api/handler.txt", "BROKEN\n"),
		say("Done."),
		use("read", map[string]string{"path": "api/handler.txt"}),
		writes("api/handler.txt", "serves\n"),
		say("Fixed what the gate showed."))
	s.model.reviewer("api", approve("fine"), approve("fine again"))
	s.open()
	s.ask("Build the api.")

	c := s.waitState("api", "working")
	s.merge(c.ID)
	s.waitState("api", "merged")

	gates := s.gates(c.ID)
	if len(gates) != 2 || gates[0].Result != "red" || gates[1].Result != "green" {
		t.Fatalf("the gates measured %+v, want red then green", gates)
	}
	completed := s.all(c.ID, "child.completed")
	for _, d := range completed {
		if d.Session != c.ID {
			t.Errorf("a turn of the work ended in session %s, not the child's %s", d.Session, c.ID)
		}
	}
	if got := s.show("main", "api/handler.txt"); got != "serves\n" {
		t.Errorf("main holds api/handler.txt = %q, want the fix", got)
	}
	s.inOrder(c.ID, "dispatch", "child.completed", "review.verdict", "gate.measured", "child.returned",
		"child.completed", "review.verdict", "gate.measured", "question.asked", "question.answered",
		"merge.queued", "merge.completed")
	s.quiet()
}

// TestScenarioAReviewDefectReturnsRedFirst: the reviewer points at a defect;
// the child first writes a test that fails on it, the harness runs it and
// confirms it red on a commit without the fix, and only then the fix.
func TestScenarioAReviewDefectReturnsRedFirst(t *testing.T) {
	s := newScenario(t, scenarioOptions{forge: "local"})
	s.model.orchestrator(dispatch(item("docs", "docs", "tests/docs")), say("Dispatched."))
	s.model.child("docs",
		writes("docs/a.md", "wrong\n"),
		say("Done."),
		writes("tests/docs/check.sh", "grep -qx right docs/a.md\n"),
		use("red_test", map[string]string{"command": "sh tests/docs/check.sh"}),
		say("The test fails on the defect."),
		use("read", map[string]string{"path": "docs/a.md"}),
		writes("docs/a.md", "right\n"),
		say("Fixed."))
	s.model.reviewer("docs",
		use("verdict", map[string]any{"verdict": tools.VerdictChanges, "why": "the doc is wrong",
			"findings": []map[string]any{{"path": "docs/a.md", "line": 1, "what": "it says wrong", "why": "the task wants right"}}}),
		approve("now it is right"))
	s.open()
	s.ask("Write the docs.")

	c := s.waitState("docs", "working")
	s.waitQuestion(c.ID)

	rets := s.returns(c.ID)
	if len(rets) != 2 || rets[0].Phase != "red" || rets[1].Phase != "fix" || rets[0].Why != "review" {
		t.Fatalf("the hand-backs were %+v, want review in the red phase, then the fix", rets)
	}
	red := s.all(c.ID, "red_test.confirmed")
	if len(red) != 1 {
		t.Fatalf("red_test.confirmed %d times, want once", len(red))
	}
	g := decode[protocol.GateResult](t, red[0].Measured)
	if g.Result != "red" || s.show(g.Commit, "tests/docs/check.sh") == "" || s.show(g.Commit, "docs/a.md") != "wrong\n" {
		t.Errorf("the red test was confirmed on %s (%s): it must be the commit with the test and without the fix", g.Commit, g.Result)
	}
	if last := s.gates(c.ID); len(last) != 1 || last[0].Result != "green" {
		t.Errorf("the gates measured %+v, want one green, after the fix", last)
	}
	s.inOrder(c.ID, "review.verdict", "child.returned", "red_test.confirmed", "child.returned",
		"child.completed", "review.verdict", "gate.measured", "question.asked")
	s.quiet()
}

// TestScenarioADiffOutsideOwnsReturnsWithoutReview: the child writes outside
// its owns through bash, where the write tool cannot refuse it; the harness
// commits nothing, opens no review, and hands the work back with the paths.
func TestScenarioADiffOutsideOwnsReturnsWithoutReview(t *testing.T) {
	s := newScenario(t, scenarioOptions{forge: "local"})
	s.model.orchestrator(dispatch(item("web", "web")), say("Dispatched."))
	s.model.child("web",
		writes("web/index.txt", "home\n"),
		use("bash", map[string]string{"command": "echo stray > outside.txt"}),
		say("Done."),
		use("bash", map[string]string{"command": "rm outside.txt"}),
		say("Removed what was not mine."))
	s.model.reviewer("web", approve("only web changed"))
	s.open()
	s.ask("Build the web page.")

	c := s.waitState("web", "working")
	s.waitFor("the owns check never handed the work back", func() bool { return s.count(c.ID, "child.returned") > 0 })
	r := s.returns(c.ID)[0]
	if r.Why != protocol.CodeOutsideOwns || len(r.Paths) != 1 || r.Paths[0] != "outside.txt" {
		t.Errorf("the hand-back is %+v, want outside_owns with outside.txt", r)
	}
	first := s.all(c.ID, "child.completed")[0]
	if head := decode[struct {
		Head string `json:"head"`
	}](t, first.Data).Head; head != "" {
		t.Errorf("the harness committed %s over a change outside owns", head)
	}

	s.waitQuestion(c.ID)
	// One review is two answers: the verdict, and the turn's last.
	if n := s.model.requests("reviewer/web"); n != 2 {
		t.Errorf("the reviewer answered %d times, want one review: after the stray file went, never before", n)
	}
	head := s.kid("web").Head
	if s.show(head, "outside.txt") != "" || s.show(head, "web/index.txt") != "home\n" {
		t.Errorf("the child's commit %s holds the stray file, or not its own", head)
	}
	s.inOrder(c.ID, "owns.checked", "child.completed", "child.returned", "owns.checked", "child.completed",
		"review.verdict", "gate.measured", "question.asked")
	s.quiet()
}

// TestScenarioAMissingHookDoesNotSkipTheGate: core.hooksPath changes under
// the orchestrator while it makes the worktree, to a folder that is not
// there; hooks.changed says both values, and the gate runs all the same.
func TestScenarioAMissingHookDoesNotSkipTheGate(t *testing.T) {
	gone := "/nonexistent/scenario-hooks"
	hooks := t.TempDir()
	if err := os.WriteFile(filepath.Join(hooks, "post-checkout"), []byte("#!/bin/sh\ngit config core.hooksPath "+gone+"\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	s := newScenario(t, scenarioOptions{forge: "local", after: func(root string) {
		gitIn(t, root, "config", "core.hooksPath", hooks)
	}})
	s.model.orchestrator(dispatch(item("docs", "docs")), say("Dispatched."))
	s.model.child("docs", writes("docs/a.md", "hello\n"), say("Done."))
	s.model.reviewer("docs", approve("fine"))
	s.open()
	s.ask("Write the docs.")

	c := s.waitState("docs", "working")
	s.waitQuestion(c.ID)

	changed := s.all(c.ID, "hooks.changed")
	if len(changed) != 1 {
		t.Fatalf("hooks.changed %d times, want once", len(changed))
	}
	said := decode[map[string]string](t, changed[0].Data)
	if said["before"] != hooks || said["after"] != gone {
		t.Errorf("hooks.changed says %v, want %s then %s", said, hooks, gone)
	}
	if g := s.gates(c.ID); len(g) != 1 || g[0].Result != "green" {
		t.Errorf("with the hooks gone the gate measured %+v, want it run and green", g)
	}
	s.inOrder(c.ID, "worktree.created", "hooks.changed", "child.completed", "review.verdict", "gate.measured", "question.asked")
	s.quiet()
}
