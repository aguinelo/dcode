package orchestrate

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/aguinelo/dcode/internal/protocol"
)

// gitLine is what git said, trimmed.
func gitLine(t *testing.T, dir string, args ...string) string {
	t.Helper()
	return strings.TrimSpace(gitOut(t, dir, args...))
}

func protocolProject(p mergeProject) protocol.Project {
	return protocol.Project{ID: testProjectID, Root: p.root}
}

// Spec: orchestration .p §7.4, §8; .i step 9 — the integration place is the
// project's integration/, a detached worktree of the repository, made once
// and never the person's tree.
func TestTheIntegrationIsADetachedWorktreeOfItsOwn(t *testing.T) {
	p := newProject(t, false)
	in, err := OpenIntegration(context.Background(), p.root, p.dir)
	if err != nil {
		t.Fatal(err)
	}
	place := filepath.Join(p.dir, "integration")
	if in.place != place {
		t.Fatalf("place = %s, want %s", in.place, place)
	}
	if b := gitLine(t, place, "rev-parse", "--abbrev-ref", "HEAD"); b != "HEAD" {
		t.Errorf("the integration is on %q, want detached", b)
	}
	if _, err := OpenIntegration(context.Background(), p.root, p.dir); err != nil {
		t.Fatalf("opening again: %v", err)
	}
	if n := worktrees(t, p.root); n != 2 {
		t.Errorf("%d worktrees, want the person's and one integration", n)
	}
	if _, err := OpenIntegration(context.Background(), p.root, filepath.Join(p.root, "inside")); err == nil {
		t.Error("an integration inside the person's tree was made")
	}
}

// Spec: orchestration .p §7.4; .i step 9 — the branch is brought up to date by
// merging main into it, never by rewriting it: H has the child's head and
// main as parents, and the branch moves forward to H.
func TestUpdatingMergesMainIntoTheBranchAndNeverRewritesIt(t *testing.T) {
	p := newProject(t, false)
	wt, head := p.child(t, "dcode/a", "main", "a.txt", "a\n")
	writeFile(t, p.root, "file.txt", "main moved\n")
	git(t, p.root, "commit", "-qam", "main moves")
	main := revParse(t, p.root, "main")
	l := openLog(t, filepath.Dir(p.dir), protocolProject(p))
	in, err := OpenIntegration(context.Background(), p.root, p.dir)
	if err != nil {
		t.Fatal(err)
	}
	h, err := in.Update(context.Background(), l, "c1", wt, head, main)
	if err != nil {
		t.Fatal(err)
	}
	if parents := gitLine(t, p.root, "rev-list", "--parents", "-n1", h); parents != h+" "+head+" "+main {
		t.Errorf("H's parents = %q, want the child's head then main", parents)
	}
	if revParse(t, wt, "HEAD") != h || revParse(t, p.root, "dcode/a") != h {
		t.Error("the branch did not move forward to H")
	}
	if d := decisionOf(t, p, "branch.updated"); !strings.Contains(string(d.Data), h) || !strings.Contains(string(d.Data), main) {
		t.Errorf("branch.updated = %s", d.Data)
	}
	// Up to date already: H is the head, and nothing is made.
	again, err := in.Update(context.Background(), l, "c1", wt, h, main)
	if err != nil || again != h {
		t.Errorf("a second update = %s, %v; want %s", again, err, h)
	}
}

// Spec: AGENTS.md, every failure said — a child's worktree that cannot move
// forward to H, an edit of its own in the way, is an error and nothing is
// recorded as updated.
func TestABranchThatCannotMoveForwardIsAnError(t *testing.T) {
	p := newProject(t, false)
	wt, head := p.child(t, "dcode/a", "main", "a.txt", "a\n")
	writeFile(t, p.root, "file.txt", "main moved\n")
	git(t, p.root, "commit", "-qam", "main moves")
	writeFile(t, wt, "file.txt", "the child's own edit\n")
	l := openLog(t, filepath.Dir(p.dir), protocolProject(p))
	in, err := OpenIntegration(context.Background(), p.root, p.dir)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := in.Update(context.Background(), l, "c1", wt, head, revParse(t, p.root, "main")); err == nil {
		t.Fatal("the branch moved over the child's own edit")
	}
	if revParse(t, p.root, "dcode/a") != head {
		t.Error("the branch moved")
	}
	for _, d := range readLines(t, filepath.Dir(p.dir)) {
		if d.Kind == "branch.updated" {
			t.Error("branch.updated was recorded for an update that failed")
		}
	}
}

// Spec: orchestration .i step 9, Q-H — a conflict goes back to the child that
// owns the branch with the paths, the markers left in its worktree; the
// integration ends clean and the orchestrator resolves nothing.
func TestAConflictReturnsToTheChildWithItsPathsAndTheIntegrationEndsClean(t *testing.T) {
	p := newProject(t, false)
	wt, head := p.child(t, "dcode/a", "main", "file.txt", "child\n")
	writeFile(t, p.root, "file.txt", "main\n")
	git(t, p.root, "commit", "-qam", "main moves")
	main := revParse(t, p.root, "main")
	l := openLog(t, filepath.Dir(p.dir), protocolProject(p))
	in, err := OpenIntegration(context.Background(), p.root, p.dir)
	if err != nil {
		t.Fatal(err)
	}
	_, err = in.Update(context.Background(), l, "c1", wt, head, main)
	var c *Conflict
	if !errors.As(err, &c) || strings.Join(c.Paths, ",") != "file.txt" {
		t.Fatalf("Update = %v, want a conflict on file.txt", err)
	}
	if r := c.Return(); r.Why != "conflict" || len(r.Paths) != 1 {
		t.Errorf("Return = %+v", r)
	}
	if s := gitLine(t, in.place, "status", "--porcelain"); s != "" {
		t.Errorf("the integration is not clean:\n%s", s)
	}
	b, _ := os.ReadFile(filepath.Join(wt, "file.txt"))
	if !strings.Contains(string(b), "<<<<<<<") {
		t.Errorf("the child's worktree has no markers:\n%s", b)
	}
	if revParse(t, p.root, "dcode/a") != head {
		t.Error("the branch moved on a conflict")
	}
}
