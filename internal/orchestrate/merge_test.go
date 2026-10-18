package orchestrate

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/aguinelo/dcode/internal/protocol"
)

// githubProject is a project on the forge: origin a bare repository, gh the
// stand-in, and one child measured green on its own commit over main.
func githubProject(t *testing.T) (mergeProject, string, Forge, protocol.Child) {
	t.Helper()
	gh := fakeGh(t)
	p := newProject(t, true)
	main := revParse(t, p.root, "main")
	_, head := p.child(t, "dcode/a", "main", "a.txt", "a\n")
	f, err := NewForge(context.Background(), ForgeSetup{Declared: ForgeGitHub, Root: p.root, Main: "main"})
	if err != nil {
		t.Fatal(err)
	}
	return p, gh, f, measured("c1", "dcode/a", head, main)
}

// Spec: orchestration .p §7.5, §7.6; .i step 10 — in step-by-step every merge
// waits for the person's answer: before a yes nothing merges and the forge is
// never asked; after it, the child merges.
func TestAMergeWaitsForThePersonsYes(t *testing.T) {
	p, gh, f, c := githubProject(t)
	ev := &events{}
	m, _ := newMerges(t, p, f, ev)
	c, q, err := m.Propose(context.Background(), Proposal{Child: c, Title: c.Title, Body: "Adds a."})
	if err != nil {
		t.Fatal(err)
	}
	if q.Kind != "merge" || q.Recommended != "merge" || q.ChildID != c.ID || len(q.Options) != 3 {
		t.Fatalf("question = %+v, want a merge question with merge, return and close", q)
	}
	if _, err := m.Merge(context.Background(), c, false); codeOf(err) != protocol.CodeInvalidInput {
		t.Fatalf("Merge before the answer = %v, want invalid_input: nothing merges without the person's yes", err)
	}
	if strings.Contains(ghCalls(t, gh), "merge\n") {
		t.Fatalf("the forge was asked to merge before the person answered:\n%s", ghCalls(t, gh))
	}
	if _, err := m.Answer(q.ID, "merge", ""); err != nil {
		t.Fatal(err)
	}
	ghMergesTo(t, gh, c.Head)
	out, err := m.Merge(context.Background(), c, false)
	if err != nil {
		t.Fatal(err)
	}
	if out.Completed == nil || out.Completed.Merged != c.Head || out.Completed.Measured != c.Head || out.Completed.PR != 7 {
		t.Fatalf("outcome = %+v, want merge.completed of the measured commit through PR 7", out)
	}
	if out.Main.Health != "green" || !ev.has(protocol.EventMergeCompleted) || !ev.has(protocol.EventMainHealth) {
		t.Errorf("main = %+v, events = %v", out.Main, ev.got)
	}
	want := []string{"pr.opened", "question.asked", "question.answered", "merge.queued", "main.health", "merge.completed", "main.health"}
	if got := logKinds(t, p); strings.Join(got, " ") != strings.Join(want, " ") {
		t.Errorf("log = %v, want %v", got, want)
	}
}

// Spec: orchestration .i step 10 — `gh pr merge` carries --match-head-commit
// with GateResult.Commit, so the server merges the commit measured or nothing.
func TestTheMergeOnGitHubPassesMatchHeadCommitWithTheMeasuredCommit(t *testing.T) {
	p, gh, f, c := githubProject(t)
	m, _ := newMerges(t, p, f, &events{})
	c = proposeAndAccept(t, m, c)
	ghMergesTo(t, gh, c.Head)
	if _, err := m.Merge(context.Background(), c, false); err != nil {
		t.Fatal(err)
	}
	want := "pr\nmerge\n7\n--squash\n--match-head-commit\n" + c.Gate.Commit + "\n"
	if !strings.Contains(ghCalls(t, gh), want) {
		t.Fatalf("gh calls:\n%s\nwant a block:\n%s", ghCalls(t, gh), want)
	}
	if remoteHas(t, p.remote, "dcode/a") {
		t.Error("the harness's branch is still on origin after the merge")
	}
}

// Spec: orchestration .i step 10 — main that moved after the gate is
// gate_stale: no merge, and the child goes back to be measured again.
func TestAMainThatMovedAfterTheGateIsStaleAndNothingMerges(t *testing.T) {
	p, gh, f, c := githubProject(t)
	m, _ := newMerges(t, p, f, &events{})
	c = proposeAndAccept(t, m, c)
	writeFile(t, p.root, "file.txt", "moved\n")
	git(t, p.root, "commit", "-qam", "main moves")
	git(t, p.root, "push", "-q", "origin", "main")
	out, err := m.Merge(context.Background(), c, false)
	if err != nil {
		t.Fatal(err)
	}
	if out.Refused == nil || out.Refused.Code != protocol.CodeGateStale {
		t.Fatalf("outcome = %+v, want merge.refused{gate_stale}", out)
	}
	if strings.Contains(ghCalls(t, gh), "pr\nmerge\n") {
		t.Fatalf("the forge was asked to merge a stale measurement:\n%s", ghCalls(t, gh))
	}
	if refusedCode(t, decisionOf(t, p, "merge.refused")) != protocol.CodeGateStale {
		t.Error("the refusal is not in the log as gate_stale")
	}
	if m.queued(c.ID) {
		t.Error("a stale child stays in the queue; it goes back to be measured")
	}
}

// Spec: orchestration .p §7.6; .i step 10 — a red main stops the queue: the
// next merge is merge.refused{main_red} and waits; only fixes_main passes.
func TestARedMainStopsTheQueueAndFixesMainPasses(t *testing.T) {
	p, gh, f, c1 := githubProject(t)
	m, _ := newMerges(t, p, f, &events{})
	main := c1.Gate.Base
	_, h2 := p.child(t, "dcode/b", "main", "b.txt", "b\n")
	c1 = proposeAndAccept(t, m, c1)
	c2 := proposeAndAccept(t, m, measured("c2", "dcode/b", h2, main))

	ghMergesTo(t, gh, c1.Head)
	ghSays(t, gh, "runs-"+c1.Head+".json", `[{"status":"completed","conclusion":"failure"}]`)
	out, err := m.Merge(context.Background(), c1, false)
	if err != nil || out.Completed == nil || out.Main.Health != "red" {
		t.Fatalf("first merge = %+v, %v; want completed with main red", out, err)
	}

	out, err = m.Merge(context.Background(), c2, false)
	if err != nil || out.Refused == nil || out.Refused.Code != protocol.CodeMainRed {
		t.Fatalf("second merge = %+v, %v; want merge.refused{main_red}", out, err)
	}
	if !m.queued(c2.ID) {
		t.Error("the refused child left the queue; with main red it waits in it")
	}

	_, h3 := p.child(t, "dcode/fix", c1.Head, "fix.txt", "fix\n")
	c3 := proposeAndAccept(t, m, measured("c3", "dcode/fix", h3, c1.Head))
	ghMergesTo(t, gh, c3.Head)
	out, err = m.Merge(context.Background(), c3, true)
	if err != nil || out.Completed == nil || out.Main.Health != "green" {
		t.Fatalf("fixes_main merge = %+v, %v; want it through, and main green", out, err)
	}
}

// Spec: orchestration .p §7.6 — pending past ci_timeout is unknown, and
// unknown stops the queue as red does.
func TestPendingCIPastTheTimeoutIsUnknownAndStopsTheQueue(t *testing.T) {
	p, gh, f, c := githubProject(t)
	l := openLog(t, filepath.Dir(p.dir), protocol.Project{ID: testProjectID, Root: p.root})
	m, err := NewMerges(MergeOptions{Log: l, Forge: f, Poll: 5 * time.Millisecond,
		Config: MergeConfig{CITimeout: 60 * time.Millisecond, Sensitive: defaultSensitive, SensitiveDiffLines: 800}})
	if err != nil {
		t.Fatal(err)
	}
	c = proposeAndAccept(t, m, c)
	ghMergesTo(t, gh, c.Head)
	ghSays(t, gh, "runs-"+c.Head+".json", `[{"status":"in_progress","conclusion":""}]`)
	out, err := m.Merge(context.Background(), c, false)
	if err != nil || out.Main.Health != "unknown" {
		t.Fatalf("merge = %+v, %v; want main unknown after ci_timeout", out, err)
	}
	if got := m.health(); got.Health != "unknown" || got.Commit != c.Head {
		t.Errorf("health() = %+v", got)
	}
}

// Spec: orchestration .p §7.5 — an answer other than merge is the person's
// refusal: merge.refused with the reason, and nothing merges.
func TestAnAnswerOtherThanMergeIsRefusedWithTheReason(t *testing.T) {
	p, gh, f, c := githubProject(t)
	ev := &events{}
	m, _ := newMerges(t, p, f, ev)
	c, q, err := m.Propose(context.Background(), Proposal{Child: c, Title: c.Title, Body: "Adds a."})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := m.Answer(q.ID, "return", "the name is wrong"); err != nil {
		t.Fatal(err)
	}
	d := decisionOf(t, p, "merge.refused")
	if refusedCode(t, d) != "person" || !strings.Contains(d.Why, "the name is wrong") {
		t.Errorf("merge.refused = %+v, want the person's refusal with the note", d)
	}
	if m.queued(c.ID) || !ev.has(protocol.EventMergeRefused) {
		t.Errorf("queued = %v, events = %v", m.queued(c.ID), ev.got)
	}
	if _, err := m.Merge(context.Background(), c, false); err == nil {
		t.Error("a refused child merged")
	}
	if strings.Contains(ghCalls(t, gh), "pr\nmerge\n") {
		t.Error("the forge was asked to merge what the person refused")
	}
}

// Spec: orchestration .i step 10 — the first answer wins; the second is
// question_already_answered. An answer the question does not offer is
// invalid_input, and an id the queue does not have question_not_found.
func TestASecondAnswerIsQuestionAlreadyAnswered(t *testing.T) {
	p, _, f, c := githubProject(t)
	m, _ := newMerges(t, p, f, &events{})
	_, q, err := m.Propose(context.Background(), Proposal{Child: c, Title: c.Title, Body: "Adds a."})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := m.Answer(q.ID, "maybe", ""); codeOf(err) != protocol.CodeInvalidInput {
		t.Errorf("an answer not offered = %v, want invalid_input", err)
	}
	if _, err := m.Answer("q-none", "merge", ""); codeOf(err) != protocol.CodeQuestionNotFound {
		t.Errorf("an unknown id = %v, want question_not_found", err)
	}
	if _, err := m.Answer(q.ID, "merge", ""); err != nil {
		t.Fatal(err)
	}
	if _, err := m.Answer(q.ID, "close", ""); codeOf(err) != protocol.CodeQuestionAlreadyAnswered {
		t.Errorf("a second answer = %v, want question_already_answered", err)
	}
	if got := m.Questions(); len(got) != 1 || got[0].Answer != "merge" || got[0].Answered == nil {
		t.Errorf("Questions() = %+v, want the first answer kept", got)
	}
}

// Spec: orchestration .p §7.5 — no PR and no question without a green
// measurement of the child's head; the PR's body carries the evidence, and
// never the gate's output, which may hold what the gate printed of a secret.
func TestAProposalWithoutAGreenGateOpensNothing(t *testing.T) {
	p, gh, f, c := githubProject(t)
	m, _ := newMerges(t, p, f, &events{})
	red := c
	red.Gate = &protocol.GateResult{Commit: c.Head, Base: c.Gate.Base, Result: GateRed}
	if _, _, err := m.Propose(context.Background(), Proposal{Child: red, Title: "t", Body: "b"}); err == nil {
		t.Fatal("a red gate was proposed")
	}
	if strings.Contains(ghCalls(t, gh), "pr\ncreate") || len(m.Questions()) != 0 {
		t.Fatalf("a red gate reached the forge or the queue: %q", ghCalls(t, gh))
	}
	c.Gate.OutputTail = "TOKEN=hunter2"
	c, _, err := m.Propose(context.Background(), Proposal{Child: c, Title: c.Title, Body: "Adds a.", Paths: []string{"go.mod", "a.txt"}})
	if err != nil {
		t.Fatal(err)
	}
	body := ghBody(t, gh)
	for _, want := range []string{"Adds a.", c.Gate.Commit, c.Gate.Base, "`true`", "green", "approve"} {
		if !strings.Contains(body, want) {
			t.Errorf("the PR body lacks %q:\n%s", want, body)
		}
	}
	if strings.Contains(body, "hunter2") {
		t.Errorf("the gate's output reached the PR body:\n%s", body)
	}
	if c.PR == nil || c.PR.Number != 7 || remoteHas(t, p.remote, "dcode/a") != true {
		t.Errorf("PR = %+v, pushed = %v", c.PR, remoteHas(t, p.remote, "dcode/a"))
	}
	if strings.Contains(ghCalls(t, gh), "--force") {
		t.Error("something was forced")
	}
	q := m.Questions()[0]
	if len(q.Sensitive) != 1 || q.Sensitive[0] != "go.mod" {
		t.Errorf("Sensitive = %v, want go.mod alone", q.Sensitive)
	}
}

// Spec: orchestration .config §3 — a diff over sensitive_diff_lines makes the
// question sensitive whatever paths it touches.
func TestADiffOverTheLineLimitIsSensitive(t *testing.T) {
	p, _, f, c := githubProject(t)
	m, _ := newMerges(t, p, f, &events{})
	_, q, err := m.Propose(context.Background(), Proposal{Child: c, Title: c.Title, Body: "b", Paths: []string{"a.txt"}, Lines: 801})
	if err != nil {
		t.Fatal(err)
	}
	if len(q.Sensitive) != 1 || q.Sensitive[0] != "a.txt" || !strings.Contains(q.Text, "801") {
		t.Errorf("question = %+v, want a.txt sensitive and the line count said", q)
	}
}
