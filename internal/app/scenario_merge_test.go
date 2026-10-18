package app

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/aguinelo/dcode/internal/protocol"
)

// The scenarios of the merge: what happens after a green gate — the PR, the
// person's yes, the serial queue, a main that moved or went red, and the
// local merge without a forge — read from the log, the fake gh's calls and
// git, never from what a model said.
//
// Spec: docs/specs/architecture/orchestration/202610132000-orchestration.i.spec.md, step 12;
// .p §7.5, §7.6.

// mergeAll is the person saying yes to every merge question about ids, as
// they come, until every one of them is merged.
func (s *scenario) mergeAll(ids ...string) {
	s.t.Helper()
	want := map[string]bool{}
	for _, id := range ids {
		want[id] = true
	}
	s.waitFor("not every child merged", func() bool {
		for _, q := range s.questions() {
			if want[q.ChildID] && q.Kind == "merge" && q.Answer == "" {
				s.answer(q, "merge")
			}
		}
		for _, id := range ids {
			if s.count(id, "merge.completed") == 0 {
				return false
			}
		}
		return true
	})
}

// merges is every `gh pr merge` the fake gh got, each as its argument lines.
func (s *scenario) merges() [][]string {
	var out [][]string
	for _, block := range strings.Split(s.ghCalls(), "\n\n") {
		args := strings.Split(strings.TrimSpace(block), "\n")
		if len(args) > 1 && args[0] == "pr" && args[1] == "merge" {
			out = append(out, args)
		}
	}
	return out
}

func after(args []string, flag string) string {
	for i := range args[:len(args)-1] {
		if args[i] == flag {
			return args[i+1]
		}
	}
	return ""
}

// lastGate is the last gate measured on child id.
func (s *scenario) lastGate(id string) protocol.GateResult {
	s.t.Helper()
	g := s.gates(id)
	if len(g) == 0 {
		s.t.Fatalf("no gate was measured on %s", tail(id))
	}
	return g[len(g)-1]
}

// TestScenarioThreeThemesBecomeThreeChildren: one batch of three, owns
// disjoint, three PRs, and the merges one at a time, each after the person's
// yes, each squashed at the commit its gate measured.
func TestScenarioThreeThemesBecomeThreeChildren(t *testing.T) {
	s := newScenario(t, scenarioOptions{forge: "github"})
	s.model.orchestrator(dispatch(item("docs", "docs"), item("api", "api"), item("web", "web")), say("Three children."))
	for _, k := range []string{"docs", "api", "web"} {
		s.model.child(k, writes(k+"/"+k+".txt", k+"\n"), say("Done."))
		s.model.reviewer(k, approve(k+" is fine"))
	}
	s.open()
	s.ask("Docs, api and web, please.")

	var ids []string
	s.waitFor("the three children were not dispatched", func() bool { return len(s.children()) == 3 })
	branches := map[string]bool{}
	for _, c := range s.children() {
		ids = append(ids, c.ID)
		branches[c.Branch] = true
	}
	if len(branches) != 3 {
		t.Errorf("three children on %d branches", len(branches))
	}
	if n := s.count("", "dispatch"); n != 3 {
		t.Errorf("%d dispatch decisions, want the batch's three", n)
	}
	for _, id := range ids {
		s.waitQuestion(id)
	}
	if n := strings.Count(s.ghCalls(), "pr\ncreate\n"); n != 3 {
		t.Errorf("gh opened %d PRs, want three", n)
	}
	if n := s.count("", "merge.completed"); n != 0 {
		t.Fatalf("%d merges before the person said yes", n)
	}
	s.mergeAll(ids...)

	for _, k := range []string{"docs", "api", "web"} {
		if got := s.show(s.mainTip(), k+"/"+k+".txt"); got != k+"\n" {
			t.Errorf("main holds %s/%s.txt = %q", k, k, got)
		}
	}
	for _, args := range s.merges() {
		if after(args, "--match-head-commit") == "" || !contains(args, "--squash") {
			t.Errorf("gh pr merge without --squash and the measured commit: %q", args)
		}
	}
	for _, id := range ids {
		s.inOrder(id, "dispatch", "child.completed", "review.verdict", "gate.measured", "pr.opened",
			"question.asked", "question.answered", "merge.queued", "merge.completed")
		done := s.all(id, "merge.completed")[0]
		m := decode[protocol.MergeCompleted](t, done.Data)
		if m.Measured != s.lastGate(id).Commit {
			t.Errorf("%s merged %s, and its last gate measured %s", tail(id), m.Measured, s.lastGate(id).Commit)
		}
	}
	// One at a time: each merge's health is read before the next merges.
	var seq []string
	for _, d := range s.decisions() {
		if d.Kind == "merge.completed" || d.Kind == "main.health" {
			seq = append(seq, d.Kind)
		}
	}
	for i := 1; i < len(seq); i++ {
		if seq[i] == "merge.completed" && seq[i-1] == "merge.completed" {
			t.Errorf("two merges with no health read between them: %v", seq)
		}
	}
	s.quiet()
}

func contains(xs []string, x string) bool {
	for _, v := range xs {
		if v == x {
			return true
		}
	}
	return false
}

// TestScenarioMainMovedBetweenGateAndMerge: another change lands on main
// between the gate and the person's yes; the merge is refused gate_stale,
// the child is measured again over the main of now, and only that
// measurement merges.
func TestScenarioMainMovedBetweenGateAndMerge(t *testing.T) {
	s := newScenario(t, scenarioOptions{forge: "github"})
	s.model.orchestrator(dispatch(item("docs", "docs")), say("Dispatched."))
	s.model.child("docs", writes("docs/a.md", "docs\n"), say("Done."))
	s.model.reviewer("docs", approve("fine"))
	s.open()
	s.ask("Write the docs.")

	c := s.waitState("docs", "working")
	q := s.waitQuestion(c.ID)
	first := s.lastGate(c.ID)
	moved := s.commitOnMain("other.txt", "somebody else\n")
	if code, body := s.answer(q, "merge"); code != 200 {
		t.Fatalf("the person's merge was refused: %d %s", code, body)
	}
	s.waitFor("the stale gate was not refused", func() bool { return s.count(c.ID, "merge.refused") > 0 })
	refused := decode[protocol.MergeRefused](t, s.all(c.ID, "merge.refused")[0].Data)
	if refused.Code != protocol.CodeGateStale {
		t.Errorf("the merge over a moved main was refused %q, want gate_stale", refused.Code)
	}
	s.mergeAll(c.ID)

	again := s.lastGate(c.ID)
	if again.Commit == first.Commit || again.Base != moved {
		t.Errorf("measured again on %s over %s, want a new commit over main's tip %s", again.Commit, again.Base, moved)
	}
	if !s.ancestor(moved, again.Commit) {
		t.Errorf("the second measurement %s does not contain main's tip %s", again.Commit, moved)
	}
	ms := s.merges()
	if len(ms) != 1 || after(ms[0], "--match-head-commit") != again.Commit {
		t.Errorf("gh pr merge was called %q, want once, at the second measurement %s", ms, again.Commit)
	}
	tip := s.mainTip()
	if s.show(tip, "other.txt") == "" || s.show(tip, "docs/a.md") != "docs\n" {
		t.Errorf("main at %s lost one of the two changes", tip)
	}
	s.inOrder(c.ID, "gate.measured", "question.asked", "question.answered", "merge.queued", "merge.refused",
		"gate.measured", "merge.completed")
	s.quiet()
}

// ancestor is whether commit a is in b's history.
func (s *scenario) ancestor(a, b string) bool {
	s.t.Helper()
	if s.origin != "" {
		s.git(s.root, "fetch", "-q", "origin")
	}
	cmd := exec.Command("git", "merge-base", "--is-ancestor", a, b)
	cmd.Dir = s.root
	return cmd.Run() == nil
}

// TestScenarioARedMainStopsTheQueue: main's CI is red after the first merge;
// the next merge in the queue is refused main_red, and gh is never asked for
// it.
func TestScenarioARedMainStopsTheQueue(t *testing.T) {
	s := newScenario(t, scenarioOptions{forge: "github"})
	s.model.orchestrator(dispatch(item("docs", "docs"), item("api", "api")), say("Two children."))
	for _, k := range []string{"docs", "api"} {
		s.model.child(k, writes(k+"/"+k+".txt", k+"\n"), say("Done."))
		s.model.reviewer(k, approve("fine"))
	}
	s.open()
	s.ask("Docs and api.")
	s.waitFor("two children were not dispatched", func() bool { return len(s.children()) == 2 })
	docs, api := s.kid("docs"), s.kid("api")
	s.waitQuestion(docs.ID)
	qa := s.waitQuestion(api.ID)
	if err := os.WriteFile(filepath.Join(s.gh, "red-after-merge"), nil, 0o600); err != nil {
		t.Fatal(err)
	}

	s.mergeAll(docs.ID)
	s.waitFor("main's health after the merge was never read red", func() bool {
		for _, d := range s.all("", "main.health") {
			if decode[protocol.MainHealth](t, d.Measured).Health == "red" {
				return true
			}
		}
		return false
	})
	if code, body := s.answer(qa, "merge"); code != 200 {
		t.Fatalf("the person's merge was refused: %d %s", code, body)
	}
	s.waitFor("the merge over a red main was not refused", func() bool { return s.count(api.ID, "merge.refused") > 0 })
	refused := decode[protocol.MergeRefused](t, s.all(api.ID, "merge.refused")[0].Data)
	if refused.Code != protocol.CodeMainRed {
		t.Errorf("the second merge was refused %q, want main_red", refused.Code)
	}
	if n := len(s.merges()); n != 1 {
		t.Errorf("gh pr merge was called %d times, want once: the red main stops the queue", n)
	}
	if k := s.kid("api"); k.State != "pr" || s.count(api.ID, "merge.completed") != 0 {
		t.Errorf("behind a red main the second child is %s, want it waiting in pr", k.State)
	}
	s.inOrder("", "merge.completed", "main.health", "merge.refused")
	s.quiet()
}

// TestScenarioNoForgeMergesLocally: without an origin the merge is local —
// a squash on the integration tree and a conditional update of main — and
// forge.local says what is lost, once.
func TestScenarioNoForgeMergesLocally(t *testing.T) {
	s := newScenario(t, scenarioOptions{forge: "local", env: map[string]string{"DCODE_ORCHESTRATOR_FORGE": "auto"}})
	s.model.orchestrator(dispatch(item("docs", "docs")), say("Dispatched."))
	s.model.child("docs", writes("docs/a.md", "docs\n"), say("Done."))
	s.model.reviewer("docs", approve("fine"))
	s.open()
	before := s.mainTip()
	s.ask("Write the docs.")

	c := s.waitState("docs", "working")
	s.mergeAll(c.ID)

	tip := s.mainTip()
	if tip == before || s.show(tip, "docs/a.md") != "docs\n" {
		t.Errorf("main is at %s (was %s) without the child's file", tip, before)
	}
	if parent := s.git(s.root, "rev-parse", tip+"^"); parent != before || s.git(s.root, "rev-list", "--count", before+".."+tip) != "1" {
		t.Errorf("the local merge is not one squash on the old main: parent %s, old main %s", parent, before)
	}
	m := decode[protocol.MergeCompleted](t, s.all(c.ID, "merge.completed")[0].Data)
	if m.Merged != tip || m.Measured != s.lastGate(c.ID).Commit {
		t.Errorf("merge.completed says %+v, want the measured commit squashed to %s", m, tip)
	}
	if n := s.count("", "forge.local"); n != 1 {
		t.Errorf("forge.local said %d times, want once", n)
	}
	if k := s.kid("docs"); k.PR != nil && k.PR.Number != 0 {
		t.Errorf("a local merge opened PR %+v", k.PR)
	}
	s.inOrder("", "forge.local", "gate.measured", "question.asked", "question.answered", "merge.completed", "main.health")
	s.quiet()
}
