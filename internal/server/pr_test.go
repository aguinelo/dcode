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

// prWorkspace is gitRepo with its changes committed on a branch `feature`, an
// origin that is a bare repository holding main, and the stand-in gh from
// internal/vcs first on PATH. It returns the workspace and the folder the
// stand-in answers from.
func prWorkspace(t *testing.T) (dir, gh string) {
	t.Helper()
	dir = gitRepo(t)
	remote := t.TempDir()
	run := func(in string, args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = in
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	run(remote, "init", "-q", "--bare", "-b", "main")
	run(dir, "remote", "add", "origin", remote)
	run(dir, "push", "-q", "origin", "main")
	run(dir, "remote", "set-head", "origin", "main")
	run(dir, "switch", "-q", "-c", "feature")
	run(dir, "add", "-A")
	run(dir, "commit", "-q", "-m", "second")

	bin, err := filepath.Abs(filepath.Join("..", "vcs", "testdata", "fakegh"))
	if err != nil {
		t.Fatal(err)
	}
	gh = t.TempDir()
	t.Setenv("FAKE_GH", gh)
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	writeFile(t, gh, "repo.json", `{"defaultBranchRef":{"name":"main"}}`)
	return dir, gh
}

func writeFile(t *testing.T, dir, name, body string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
}

const ghPR = `{"number":7,"url":"https://github.com/o/r/pull/7","title":"Add b","state":"OPEN","isDraft":false,` +
	`"baseRefName":"main","headRefName":"feature","statusCheckRollup":[` +
	`{"__typename":"CheckRun","name":"test","status":"COMPLETED","conclusion":"FAILURE"}]}`

func prPath(ws string) string { return "/git/pr?" + url.Values{"workspace": {ws}}.Encode() }

// GET /git/pr answers the PR of the branch checked out now, its checks said as
// one word, and no PR when the branch has none.
func TestThePRRouteAnswersTheBranchsPullRequest(t *testing.T) {
	dir, gh := prWorkspace(t)
	srv, _ := newServer(t, 4)

	rec := get(srv, prPath(dir))
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"branch":"feature"`) ||
		!strings.Contains(rec.Body.String(), `"base":"main"`) || strings.Contains(rec.Body.String(), `"pr"`) {
		t.Fatalf("a branch with no PR answered %d %s", rec.Code, rec.Body.String())
	}

	writeFile(t, gh, "pr.json", ghPR)
	rec = get(srv, prPath(dir))
	var out protocol.GitPRResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil || rec.Code != http.StatusOK {
		t.Fatalf("%d %s", rec.Code, rec.Body.String())
	}
	want := protocol.GitPR{Number: 7, URL: "https://github.com/o/r/pull/7", Title: "Add b", State: protocol.GitPROpen,
		Base: "main", Head: "feature", Checks: protocol.GitChecks{State: protocol.GitChecksFail, Failed: 1}}
	if out.PR == nil || *out.PR != want {
		t.Errorf("the PR came back as %+v, want %+v", out.PR, want)
	}
}

// POST /git/pr opens the PR the person confirmed and answers it, 201.
func TestOpeningAPRAnswersIt(t *testing.T) {
	dir, gh := prWorkspace(t)
	writeFile(t, gh, "created.json", ghPR)
	srv, _ := newServer(t, 4)
	rec := post(srv, "/git/pr", protocol.CreateGitPRRequest{Workspace: dir, Title: "Add b", Body: "b", Base: "main"})
	if rec.Code != http.StatusCreated {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}
	var out protocol.GitPRResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if out.PR == nil || out.PR.Number != 7 || out.Branch != "feature" {
		t.Errorf("created %+v", out)
	}
}

// Every refusal carries a code that says whose it is: gh's absence, a branch
// that cannot become a PR, a request that is malformed.
func TestThePRRoutesSayWhyWithACode(t *testing.T) {
	dir, gh := prWorkspace(t)
	srv, _ := newServer(t, 4)

	for _, c := range []struct {
		name   string
		body   any
		status int
		code   string
	}{
		{"malformed", "not an object", http.StatusBadRequest, protocol.CodeInvalidInput},
		{"no title", protocol.CreateGitPRRequest{Workspace: dir}, http.StatusBadRequest, protocol.CodeInvalidInput},
		{"the base itself", protocol.CreateGitPRRequest{Workspace: dir, Title: "x", Base: "feature"}, http.StatusConflict, protocol.CodePRRefused},
	} {
		rec := post(srv, "/git/pr", c.body)
		if rec.Code != c.status || !strings.Contains(rec.Body.String(), c.code) {
			t.Errorf("%s: %d %s, want %d %s", c.name, rec.Code, rec.Body.String(), c.status, c.code)
		}
	}

	writeFile(t, gh, "logged-out", "")
	for _, rec := range []interface{ Result() *http.Response }{
		get(srv, prPath(dir)),
		post(srv, "/git/pr", protocol.CreateGitPRRequest{Workspace: dir, Title: "x"}),
	} {
		res := rec.Result()
		var e protocol.Error
		_ = json.NewDecoder(res.Body).Decode(&e)
		if res.StatusCode != http.StatusServiceUnavailable || e.Code != protocol.CodeGhUnavailable || !strings.Contains(e.Message, "gh auth login") {
			t.Errorf("logged out: %d %+v", res.StatusCode, e)
		}
	}
}

// Both PR routes refuse a workspace that is missing, relative or not there,
// before git or gh runs.
func TestThePRRoutesRefuseAWorkspaceTheyCannotRead(t *testing.T) {
	_, gh := prWorkspace(t)
	srv, _ := newServer(t, 4)
	missing := filepath.Join(t.TempDir(), "nowhere")
	for _, rec := range []interface{ Result() *http.Response }{
		get(srv, "/git/pr"), get(srv, "/git/pr?workspace=projects/dcode"), get(srv, prPath(missing)),
		post(srv, "/git/pr", protocol.CreateGitPRRequest{Title: "x"}),
		post(srv, "/git/pr", protocol.CreateGitPRRequest{Workspace: "projects/dcode", Title: "x"}),
		post(srv, "/git/pr", protocol.CreateGitPRRequest{Workspace: missing, Title: "x"}),
	} {
		res := rec.Result()
		var e protocol.Error
		_ = json.NewDecoder(res.Body).Decode(&e)
		if res.StatusCode != http.StatusBadRequest || e.Code != protocol.CodeWorkspaceInvalid {
			t.Errorf("%d %+v, want 400 %s", res.StatusCode, e, protocol.CodeWorkspaceInvalid)
		}
	}
	if b, _ := os.ReadFile(filepath.Join(gh, "calls")); len(b) != 0 {
		t.Errorf("gh ran for a workspace that was refused:\n%s", b)
	}
}
