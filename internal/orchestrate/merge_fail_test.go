package orchestrate

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/aguinelo/dcode/internal/protocol"
)

// Spec: orchestration .p §5.3 — the forge's own refusal is merge.refused,
// said with what GitHub said; nothing moves and the child leaves the queue.
func TestAMergeGitHubRefusesIsSaidAndLeavesTheQueue(t *testing.T) {
	p, gh, f, c := githubProject(t)
	m, _ := newMerges(t, p, f, &events{})
	c = proposeAndAccept(t, m, c)
	ghSays(t, gh, "merge-fails", "Head branch was modified. Review and try the merge again.")
	out, err := m.Merge(context.Background(), c, false)
	if err != nil || out.Refused == nil || out.Refused.Code != "forge" || !strings.Contains(out.Refused.Reason, "Head branch was modified") {
		t.Fatalf("Merge = %+v, %v; want the forge's refusal with what it said", out, err)
	}
	if m.queued(c.ID) {
		t.Error("a refused child stays in the queue")
	}
}

// Spec: AGENTS.md, every failure said — a merge that stands with the branch
// still on origin is completed, and the why says the branch was left.
func TestABranchLeftOnOriginIsSaidBesideAMergeThatStands(t *testing.T) {
	p, gh, f, c := githubProject(t)
	m, _ := newMerges(t, p, f, &events{})
	c = proposeAndAccept(t, m, c)
	ghMergesTo(t, gh, c.Head)
	ghSays(t, gh, "delete-branch", "dcode/a")
	out, err := m.Merge(context.Background(), c, false)
	if err != nil || out.Completed == nil {
		t.Fatalf("Merge = %+v, %v", out, err)
	}
	if d := decisionOf(t, p, "merge.completed"); !strings.Contains(d.Why, "still on origin") {
		t.Errorf("merge.completed why = %q, want the branch left said", d.Why)
	}
}

// Spec: AGENTS.md, every failure said — gh that says the PR is not merged
// after the merge, a PR that cannot be opened, an address that is not a PR,
// and CI that cannot be read are errors or unknown, never a quiet success.
func TestWhatGhCannotSayIsAnErrorNeverASuccess(t *testing.T) {
	p, gh, f, c := githubProject(t)
	m, _ := newMerges(t, p, f, &events{})

	ghSays(t, gh, "create-fails", "a pull request for branch already exists")
	if _, _, err := m.Propose(context.Background(), Proposal{Child: c, Title: "t", Body: "b"}); err == nil ||
		!strings.Contains(err.Error(), "already exists") {
		t.Errorf("a PR gh did not open = %v", err)
	}
	if len(m.Questions()) != 0 {
		t.Error("a question was asked for a PR that was not opened")
	}
	if err := os.Remove(filepath.Join(gh, "create-fails")); err != nil {
		t.Fatal(err)
	}
	c = proposeAndAccept(t, m, c)
	ghSays(t, gh, "view.json", `{"state":"OPEN","mergeCommit":null}`)
	if _, err := m.Merge(context.Background(), c, false); err == nil || !strings.Contains(err.Error(), "not merged") {
		t.Errorf("gh saying OPEN after the merge = %v, want an error", err)
	}
	noPR := c
	noPR.PR = nil
	if _, err := f.Merge(context.Background(), noPR, c.Head); err == nil {
		t.Error("a child with no PR was merged on GitHub")
	}
	ghSays(t, gh, "runs.json", `not json`)
	if h, err := f.MainHealth(context.Background(), c.Head); err == nil || h.Health != "unknown" {
		t.Errorf("CI that does not read = %+v, %v; want unknown with an error", h, err)
	}
	if _, err := f.OpenPR(context.Background(), "dcode/a", "t", strings.Repeat("x", 70000)); err == nil {
		t.Error("a body over GitHub's limit was sent")
	}
}

// Spec: AGENTS.md, every failure said — main's CI that cannot be read after a
// merge is unknown in the log, said with why, and stops the queue.
func TestMainThatCannotBeReadIsUnknownAndSaid(t *testing.T) {
	p, gh, f, c := githubProject(t)
	m, _ := newMerges(t, p, f, &events{})
	c = proposeAndAccept(t, m, c)
	ghMergesTo(t, gh, c.Head)
	ghSays(t, gh, "runs-"+c.Head+".json", `not json`)
	out, err := m.Merge(context.Background(), c, false)
	if err == nil || out.Main.Health != "unknown" || m.health().Health != "unknown" {
		t.Fatalf("Merge = %+v, %v; want main unknown with the error", out, err)
	}
	var last Decision
	for _, d := range readLines(t, filepath.Dir(p.dir)) {
		if d.Kind == "main.health" {
			last = d
		}
	}
	if !strings.Contains(last.Why, "could not be read") {
		t.Errorf("main.health why = %q", last.Why)
	}
}

// Spec: orchestration .p §7.6 — a wait for main's CI that is stopped is
// unknown, never green.
func TestAStoppedWaitForCIIsUnknown(t *testing.T) {
	p, gh, f, c := githubProject(t)
	l := openLog(t, filepath.Dir(p.dir), protocolProject(p))
	m, err := NewMerges(MergeOptions{Log: l, Forge: f, Poll: time.Hour,
		Config: MergeConfig{CITimeout: time.Hour}})
	if err != nil {
		t.Fatal(err)
	}
	c = proposeAndAccept(t, m, c)
	ghMergesTo(t, gh, c.Head)
	ghSays(t, gh, "runs-"+c.Head+".json", `[]`)
	ctx, cancel := context.WithCancel(context.Background())
	m.o.Emit = func(typ protocol.EventType, payload any) {
		if h, ok := payload.(protocol.MainHealth); ok && h.Health == "pending" {
			cancel()
		}
	}
	out, err := m.Merge(ctx, c, false)
	if err == nil || out.Main.Health != "unknown" {
		t.Fatalf("a stopped wait = %+v, %v; want unknown with the error", out, err)
	}
}

// Spec: AGENTS.md, every failure said — a squash with nothing in it, a tip
// the forge does not have and a push the remote refuses are errors.
func TestASquashOrAPushThatCannotHappenIsAnError(t *testing.T) {
	p, f, c := localProject(t)
	main := revParse(t, p.root, "main")
	empty := c
	empty.Gate = &protocol.GateResult{Commit: main, Base: main, Result: GateGreen}
	if _, err := f.Merge(context.Background(), empty, main); err == nil {
		t.Error("a squash with nothing in it made a commit")
	}
	if revParse(t, p.root, "main") != main {
		t.Error("main moved")
	}

	gp, _, gf, _ := githubProject(t)
	if _, _, err := gf.Tips(context.Background(), "never-pushed"); err == nil {
		t.Error("the tip of a branch origin does not have was read")
	}
	writeFile(t, gp.root, "file.txt", "other\n")
	git(t, gp.root, "commit", "-qam", "other")
	git(t, gp.root, "push", "-q", "origin", "main:dcode/a")
	if _, err := gf.OpenPR(context.Background(), "dcode/a", "t", "b"); err == nil {
		t.Error("a push the remote refused opened a PR")
	}
}

// Spec: orchestration .p §4 — what cannot be built is refused, saying why.
func TestWhatTheMergeNeedsIsRefusedWhenMissing(t *testing.T) {
	p := newProject(t, false)
	ctx := context.Background()
	if _, err := NewForge(ctx, ForgeSetup{Declared: ForgeLocal}); err == nil {
		t.Error("a forge without a root was made")
	}
	if _, err := NewForge(ctx, ForgeSetup{Declared: "gitlab", Root: p.root, Main: "main"}); err == nil {
		t.Error("an unknown forge was made")
	}
	if _, err := NewForge(ctx, ForgeSetup{Declared: ForgeLocal, Root: p.root, Main: "main"}); err == nil {
		t.Error("a local forge without an integration place was made")
	}
	l := openLog(t, filepath.Dir(p.dir), protocolProject(p))
	if _, err := NewMerges(MergeOptions{Log: l}); err == nil {
		t.Error("merges without a forge were made")
	}
	in, err := OpenIntegration(ctx, p.root, p.dir)
	if err != nil {
		t.Fatal(err)
	}
	f, err := NewForge(ctx, ForgeSetup{Declared: ForgeLocal, Root: p.root, Main: "main", Integration: in})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := NewMerges(MergeOptions{Log: l, Forge: f}); err == nil {
		t.Error("merges without a ci_timeout were made")
	}
	if _, err := in.Update(ctx, nil, "c", p.root, "a", "b"); err == nil {
		t.Error("an update without a log ran")
	}
	if _, err := in.Update(ctx, l, "c", p.root, "HEAD", "main"); err == nil {
		t.Error("an update of refs, not exact commits, ran")
	}
	if _, _, err := f.Tips(ctx, "no-such-branch"); err == nil {
		t.Error("the tip of a branch that does not exist was read")
	}
	if _, err := f.Merge(ctx, protocol.Child{ID: "c"}, "x"); err == nil {
		t.Error("a child with no measurement was merged")
	}
	h, err := f.MainHealth(ctx, revParse(t, p.root, "main"))
	if err != nil || h.Health != "unknown" {
		t.Errorf("main's health with no gate declared = %+v, %v; want unknown", h, err)
	}
	other := filepath.Join(t.TempDir(), "proj")
	mkdir(t, filepath.Join(other, "integration"))
	if _, err := OpenIntegration(ctx, p.root, other); err == nil {
		t.Error("a folder that is not the integration worktree was taken for it")
	}
}
