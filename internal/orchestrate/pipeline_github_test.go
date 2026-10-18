package orchestrate

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/aguinelo/dcode/internal/protocol"
)

// originMoves puts a commit on origin's main that the local main does not
// have, as a merge on GitHub does, and returns it.
func originMoves(t *testing.T, root, remote string) string {
	t.Helper()
	tree := strings.TrimSpace(gitOut(t, root, "rev-parse", "main^{tree}"))
	commit := strings.TrimSpace(gitOut(t, root, "commit-tree", "-p", "main", "-m", "merged on GitHub", tree))
	gitOut(t, root, "push", "-q", remote, commit+":refs/heads/main")
	return commit
}

// Spec: orchestration .p §7.4.1 — with forge github, main's tip is origin's,
// fetched first; the local main, which a merge on GitHub never moves, is not.
// A fetch that fails is said, never a quiet fall back to the local main.
func TestOnGitHubMainsTipIsOriginsAfterAFetch(t *testing.T) {
	p, _, f, _ := githubProject(t)
	local := revParse(t, p.root, "main")
	moved := originMoves(t, p.root, p.remote)
	tip, err := f.MainTip(context.Background())
	if err != nil || tip != moved {
		t.Fatalf("MainTip = %s, %v; origin is at %s, the local main at %s", tip, err, moved, local)
	}
	if revParse(t, p.root, "main") != local {
		t.Error("reading the tip moved the local main")
	}
	gitOut(t, p.root, "remote", "set-url", "origin", p.remote+"-gone")
	if _, err := f.MainTip(context.Background()); err == nil || !strings.Contains(err.Error(), "fetch") {
		t.Errorf("a fetch that fails = %v, want it said", err)
	}
}

// Spec: orchestration .p §7.4 — the pipeline measures over main as the forge
// has it: with github, origin's tip after a fetch, merged into the branch.
func TestThePipelineMeasuresOverOriginsMainOnGitHub(t *testing.T) {
	fakeGh(t)
	p := openWith(t, map[string]string{"orchestrator.gate_command": "true", "orchestrator.forge": "github"})
	remote := t.TempDir()
	gitOut(t, remote, "init", "-q", "--bare", "-b", "main")
	gitOut(t, p.root, "remote", "add", "origin", remote)
	gitOut(t, p.root, "push", "-q", "origin", "main")
	c := p.one(t, item("index", "internal/index"))
	put(t, c, "internal/index/i.go", "package index\n")
	p.end(t, c.ID, protocol.StopDone, "Done.")
	moved := originMoves(t, p.root, remote)
	p.review(t, approve)

	got := p.o.child(c.ID)
	if got.Gate == nil || got.Gate.Base != moved {
		t.Fatalf("gate = %+v, want it over origin's main %s", got.Gate, moved)
	}
	gitOut(t, p.root, "merge-base", "--is-ancestor", moved, got.Gate.Commit)
}

// Spec: orchestration .p §7.5 — a child that already has its PR, measured
// again after gate_stale, is pushed (never forced) and its PR's evidence is
// refreshed; no second PR is asked for.
func TestAChildWithAPRIsPushedAndItsPRRefreshedNotOpenedAgain(t *testing.T) {
	p, gh, f, c := githubProject(t)
	m, _ := newMerges(t, p, f, &events{})
	c, _, err := m.Propose(context.Background(), Proposal{Child: c, Title: c.Title, Body: "Adds a."})
	if err != nil || c.PR == nil || c.PR.Number != 7 {
		t.Fatalf("first Propose = %+v, %v", c.PR, err)
	}
	wt := strings.TrimSpace(gitOut(t, p.root, "worktree", "list", "--porcelain"))
	path := ""
	for _, l := range strings.Split(wt, "\n") {
		if strings.HasPrefix(l, "worktree ") && strings.Contains(l, "dcode-a") {
			path = strings.TrimPrefix(l, "worktree ")
		}
	}
	writeFile(t, path, "b.txt", "b\n")
	gitOut(t, path, "add", "b.txt")
	gitOut(t, path, "commit", "-q", "-m", "more")
	head := revParse(t, path, "HEAD")
	again := measured(c.ID, c.Branch, head, c.Gate.Base)
	again.PR = c.PR
	c2, q, err := m.Propose(context.Background(), Proposal{Child: again, Title: c.Title, Body: "Adds a, and b."})
	if err != nil {
		t.Fatalf("second Propose = %v", err)
	}
	calls := ghCalls(t, gh)
	if n := strings.Count(calls, "pr\ncreate\n"); n != 1 {
		t.Errorf("gh pr create ran %d times:\n%s", n, calls)
	}
	if !strings.Contains(calls, "pr\nedit\n7\n") || !strings.Contains(ghBody(t, gh), short(head)) {
		t.Errorf("the PR's evidence was not refreshed for %s:\n%s", short(head), calls)
	}
	if c2.PR == nil || c2.PR.Number != 7 || q.ID == "" {
		t.Errorf("after the refresh: pr %+v, question %q", c2.PR, q.ID)
	}
	remoteHead := strings.TrimSpace(gitOut(t, p.root, "ls-remote", p.remote, "refs/heads/dcode/a"))
	if !strings.HasPrefix(remoteHead, head) {
		t.Errorf("origin's branch is %q, want %s", remoteHead, head)
	}
}

// Spec: orchestration .p §9.1 — child.returned has one shape whoever writes
// it: {"return": Return, "returned": n}.
func TestEveryChildReturnedLineHasOneShape(t *testing.T) {
	p := openWith(t, map[string]string{"orchestrator.gate_command": "true"})
	c := p.one(t, item("index", "internal/index"))
	put(t, c, "internal/index/i.go", "package index // the child's\n")
	p.end(t, c.ID, protocol.StopDone, "Done.")
	put(t, protocol.Child{Worktree: p.root}, "internal/index/i.go", "package index // main's\n")
	gitOut(t, p.root, "add", "internal/index/i.go")
	gitOut(t, p.root, "commit", "-q", "-m", "main moves")
	p.review(t, approve) // the conflict, written by the integration's step

	put(t, c, "README.md", "outside\n")
	put(t, c, "internal/index/i.go", "package index // both\n")
	p.end(t, c.ID, protocol.StopDone, "Resolved.") // outside owns, written by the hand-back

	lines := p.lines(t, "child.returned")
	if len(lines) != 2 {
		t.Fatalf("%d child.returned lines", len(lines))
	}
	for i, d := range lines {
		var shape struct {
			Return   *protocol.Return `json:"return"`
			Returned int              `json:"returned"`
		}
		if err := json.Unmarshal(d.Data, &shape); err != nil || shape.Return == nil || shape.Return.Why == "" || shape.Returned != i+1 {
			t.Errorf("child.returned %d = %s", i+1, d.Data)
		}
	}
}
