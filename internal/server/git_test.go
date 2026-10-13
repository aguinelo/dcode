package server

import (
	"encoding/json"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/aguinelo/dcode/internal/protocol"
)

// gitRepo builds a real repository with one commit and a dirty tree: a.txt
// changed, b.txt new. The route shells out to git, so a fake would test the
// fake.
func gitRepo(t *testing.T) string {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("no git here")
	}
	dir := t.TempDir()
	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	put := func(name, body string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	run("init", "-q", "-b", "main")
	run("config", "user.email", "t@example.com")
	run("config", "user.name", "T")
	run("config", "commit.gpgsign", "false")
	put("a.txt", "one\ntwo\n")
	run("add", "-A")
	run("commit", "-q", "-m", "first")
	put("a.txt", "one\nTWO\nthree\n")
	put("b.txt", "new\n")
	return dir
}

func gitPath(ws string, extra ...string) string {
	q := url.Values{"workspace": {ws}}
	for i := 0; i+1 < len(extra); i += 2 {
		q.Set(extra[i], extra[i+1])
	}
	return "/git?" + q.Encode()
}

// GET /git answers where the folder is in git and what changed in it now:
// each file with its status and its lines, the branch as it is now.
func TestTheGitRouteListsTheTreeAsItIsNow(t *testing.T) {
	dir := gitRepo(t)
	srv, _ := newServer(t, 4)
	rec := get(srv, gitPath(dir))
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}
	var out protocol.GitResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if !out.Repository || out.Branch != "main" || out.Head == "" || out.Root == "" || out.Worktree {
		t.Errorf("where came back as %+v", out)
	}
	want := []protocol.GitFile{
		{Path: "a.txt", Status: protocol.GitModified, Added: 2, Removed: 1},
		{Path: "b.txt", Status: protocol.GitUntracked, Added: 1},
	}
	if len(out.Files) != len(want) {
		t.Fatalf("files %+v, want %+v", out.Files, want)
	}
	for i := range want {
		if out.Files[i] != want[i] {
			t.Errorf("file %d: %+v, want %+v", i, out.Files[i], want[i])
		}
	}
}

// A folder outside any repository answers that, with an empty list — said,
// not mistaken for a clean tree by a client reading only the files.
func TestAFolderThatIsNotARepositoryIsSaidAsSuch(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("no git here")
	}
	srv, _ := newServer(t, 4)
	rec := get(srv, gitPath(t.TempDir()))
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"repository":false`) || !strings.Contains(rec.Body.String(), `"files":[]`) {
		t.Errorf("a plain folder came back as %s", rec.Body.String())
	}
}

// Without git on the daemon's PATH there is no answer, and the route says why
// rather than answer "not a repository" about a folder nobody looked at.
func TestWithoutGitTheRouteSaysSo(t *testing.T) {
	dir := gitRepo(t)
	t.Setenv("PATH", t.TempDir())
	srv, _ := newServer(t, 4)
	for _, path := range []string{gitPath(dir), "/git/diff?" + url.Values{"workspace": {dir}, "path": {"a.txt"}}.Encode()} {
		rec := get(srv, path)
		if rec.Code == http.StatusOK || !strings.Contains(rec.Body.String(), "git is not installed") {
			t.Errorf("%s without git answered %d %s", path, rec.Code, rec.Body.String())
		}
	}
}

// GET /git/diff gives one changed file's unified diff, cut at the cap and
// saying so when it is.
func TestTheDiffRouteGivesOneChangedFile(t *testing.T) {
	dir := gitRepo(t)
	srv, _ := newServer(t, 4)
	rec := get(srv, "/git/diff?"+url.Values{"workspace": {dir}, "path": {"a.txt"}}.Encode())
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}
	var out protocol.GitDiffResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if out.Path != "a.txt" || out.Status != protocol.GitModified || !strings.Contains(out.Diff, "+TWO\n") ||
		out.Truncated || out.Limit != protocol.GitDiffLimit {
		t.Errorf("the diff came back as %+v", out)
	}
}

// A path git does not list as changed has no diff here: the route reads
// changes, never any file by name.
func TestTheDiffRouteRefusesAFileThatDidNotChange(t *testing.T) {
	dir := gitRepo(t)
	srv, _ := newServer(t, 4)
	for _, p := range []string{"", "c.txt", "../x", "/etc/hosts"} {
		rec := get(srv, "/git/diff?"+url.Values{"workspace": {dir}, "path": {p}}.Encode())
		if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), protocol.CodeInvalidInput) {
			t.Errorf("path %q gave %d %s, want 400 %s", p, rec.Code, rec.Body.String(), protocol.CodeInvalidInput)
		}
	}
}

// A workspace is what both routes read, so none, a relative one or one that
// does not exist is refused before git is run.
func TestTheGitRoutesRefuseAWorkspaceTheyCannotRead(t *testing.T) {
	srv, _ := newServer(t, 4)
	missing := filepath.Join(t.TempDir(), "nowhere")
	for _, path := range []string{"/git", "/git?workspace=projects/dcode", gitPath(missing),
		"/git/diff?path=a.txt", "/git/diff?workspace=projects/dcode&path=a.txt"} {
		rec := get(srv, path)
		if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), protocol.CodeWorkspaceInvalid) {
			t.Errorf("%s gave %d %s, want 400 %s", path, rec.Code, rec.Body.String(), protocol.CodeWorkspaceInvalid)
		}
	}
}
