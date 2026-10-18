package orchestrate

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/aguinelo/dcode/internal/protocol"
)

// The merge tests run on real repositories in temporary folders. gh talks to
// GitHub, which a test cannot, so it is the one stand-in: testdata/fakegh,
// first on PATH, answering from files and writing down every call.

// mergeEnv makes every git the code under test runs commit as somebody, with
// no configuration of the machine's leaking in.
func mergeEnv(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed")
	}
	for k, v := range map[string]string{"GIT_CONFIG_GLOBAL": "/dev/null", "GIT_CONFIG_NOSYSTEM": "1",
		"GIT_AUTHOR_NAME": "t", "GIT_AUTHOR_EMAIL": "t@t", "GIT_COMMITTER_NAME": "t", "GIT_COMMITTER_EMAIL": "t@t"} {
		t.Setenv(k, v)
	}
}

// fakeGh puts the stand-in first on PATH and returns the folder it answers
// from. Every main is green unless a test says otherwise.
func fakeGh(t *testing.T) string {
	t.Helper()
	bin, err := filepath.Abs(filepath.Join("testdata", "fakegh"))
	if err != nil {
		t.Fatal(err)
	}
	d := t.TempDir()
	t.Setenv("FAKE_GH", d)
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	ghSays(t, d, "runs.json", `[{"status":"completed","conclusion":"success"}]`)
	return d
}

func ghSays(t *testing.T, d, name, body string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(d, name), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
}

// ghCalls is every call the stand-in got, each a block of one argument per line.
func ghCalls(t *testing.T, d string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(d, "calls"))
	if err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}
	return string(b)
}

// ghMergesTo makes the next `gh pr merge` move origin's main to commit, as
// GitHub's squash would, and `gh pr view` say so.
func ghMergesTo(t *testing.T, d, commit string) {
	t.Helper()
	ghSays(t, d, "merge-to", commit)
	ghSays(t, d, "view.json", `{"state":"MERGED","mergeCommit":{"oid":"`+commit+`"}}`)
}

// mergeProject is a repository with main at one commit, and the folder the daemon
// keeps the project in. With origin, a bare repository elsewhere holds main too.
type mergeProject struct {
	root, remote, dir string
}

func newProject(t *testing.T, origin bool) mergeProject {
	t.Helper()
	mergeEnv(t)
	base, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	p := mergeProject{root: filepath.Join(base, "repo"), dir: filepath.Join(base, "orchestrate", testProjectID)}
	mkdir(t, p.root)
	mkdir(t, p.dir)
	git(t, p.root, "init", "-q", "-b", "main")
	writeFile(t, p.root, "file.txt", "one\n")
	git(t, p.root, "add", "file.txt")
	git(t, p.root, "commit", "-q", "-m", "first")
	if origin {
		p.remote = filepath.Join(base, "origin.git")
		git(t, base, "init", "-q", "--bare", "-b", "main", p.remote)
		git(t, p.root, "remote", "add", "origin", p.remote)
		git(t, p.root, "push", "-q", "origin", "main")
	}
	return p
}

// child is a worktree on a branch of its own, off from, with one commit
// writing name; it returns the worktree and the commit.
func (p mergeProject) child(t *testing.T, branch, from, name, body string) (string, string) {
	t.Helper()
	wt := filepath.Join(filepath.Dir(p.dir), "children", strings.ReplaceAll(branch, "/", "-"))
	git(t, p.root, "worktree", "add", "-q", "-b", branch, wt, from)
	writeFile(t, wt, name, body)
	git(t, wt, "add", name)
	git(t, wt, "commit", "-q", "-m", "child writes "+name)
	return wt, revParse(t, wt, "HEAD")
}

func mkdir(t *testing.T, dir string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
}

func writeFile(t *testing.T, dir, name, body string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

// measured is a child whose gate was green on head, over main.
func measured(id, branch, head, main string) protocol.Child {
	return protocol.Child{ID: id, Title: "Child " + id, Branch: branch, Head: head, State: "in_review",
		Gate:   &protocol.GateResult{Commit: head, Base: main, Command: "true", Result: GateGreen, At: time.Now().UTC()},
		Review: &protocol.Verdict{SessionID: "r-" + id, Commit: head, Verdict: "approve"}}
}

// events gathers what Merges sends to the wire, in order.
type events struct{ got []protocol.EventType }

func (e *events) emit(t protocol.EventType, _ any) { e.got = append(e.got, t) }

func (e *events) has(t protocol.EventType) bool {
	for _, g := range e.got {
		if g == t {
			return true
		}
	}
	return false
}

func newMerges(t *testing.T, p mergeProject, f Forge, ev *events) (*Merges, *DecisionLog) {
	t.Helper()
	l := openLog(t, filepath.Dir(p.dir), protocol.Project{ID: testProjectID, Root: p.root})
	m, err := NewMerges(MergeOptions{Log: l, Forge: f, Emit: ev.emit, Poll: 5 * time.Millisecond,
		Config: MergeConfig{CITimeout: 2 * time.Second, Sensitive: defaultSensitive, SensitiveDiffLines: 800}})
	if err != nil {
		t.Fatal(err)
	}
	return m, l
}

// logKinds is the kinds of the log, in order.
func logKinds(t *testing.T, p mergeProject) []string {
	t.Helper()
	var out []string
	for _, d := range readLines(t, filepath.Dir(p.dir)) {
		out = append(out, d.Kind)
	}
	return out
}

func decisionOf(t *testing.T, p mergeProject, kind string) Decision {
	t.Helper()
	for _, d := range readLines(t, filepath.Dir(p.dir)) {
		if d.Kind == kind {
			return d
		}
	}
	t.Fatalf("no %s in the log: %v", kind, logKinds(t, p))
	return Decision{}
}

func refusedCode(t *testing.T, d Decision) string {
	t.Helper()
	var r protocol.MergeRefused
	if err := json.Unmarshal(d.Data, &r); err != nil {
		t.Fatalf("merge.refused data: %v", err)
	}
	return r.Code
}

// proposeAndAccept opens the child's PR, asks, and answers merge.
func proposeAndAccept(t *testing.T, m *Merges, c protocol.Child) protocol.Child {
	t.Helper()
	c, q, err := m.Propose(context.Background(), Proposal{Child: c, Title: c.Title, Body: "What " + c.ID + " does."})
	if err != nil {
		t.Fatalf("Propose: %v", err)
	}
	if _, err := m.Answer(q.ID, "merge", ""); err != nil {
		t.Fatalf("Answer: %v", err)
	}
	return c
}

// remoteHas says whether the bare origin holds a branch.
func remoteHas(t *testing.T, remote, branch string) bool {
	t.Helper()
	cmd := exec.Command("git", "--git-dir", remote, "rev-parse", "--verify", "--quiet", "refs/heads/"+branch)
	cmd.Env = gitEnv(os.Environ())
	return cmd.Run() == nil
}

// ghBody is the last PR body the stand-in was handed.
func ghBody(t *testing.T, d string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(d, "body"))
	if err != nil {
		t.Fatalf("no PR body was handed to gh: %v", err)
	}
	return string(b)
}

func codeOf(err error) string {
	if pe, ok := err.(*protocol.Error); ok {
		return pe.Code
	}
	return ""
}
