package vcs

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// fakeGh puts the stand-in GitHub CLI first on PATH and returns the folder it
// answers from and writes its calls to. gh talks to GitHub; a test cannot, so
// this is the one place a fake is the honest choice — git stays real.
func fakeGh(t *testing.T) string {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("no git here")
	}
	bin, err := filepath.Abs(filepath.Join("testdata", "fakegh"))
	if err != nil {
		t.Fatal(err)
	}
	d := t.TempDir()
	t.Setenv("FAKE_GH", d)
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	put(t, d, "repo.json", `{"defaultBranchRef":{"name":"main"}}`)
	return d
}

func put(t *testing.T, dir, name, body string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
}

// prRepo builds a repository whose origin is a bare one in another folder, with
// main pushed and the clone knowing it as the default, and a branch `feature`
// one commit ahead of it, checked out and not pushed.
func prRepo(t *testing.T) (dir, remote string) {
	t.Helper()
	dir = repo(t)
	remote = t.TempDir()
	git(t, remote, "init", "-q", "--bare", "-b", "main")
	write(t, dir, "a.txt", "one\n")
	commitAll(t, dir, "first")
	git(t, dir, "remote", "add", "origin", remote)
	git(t, dir, "push", "-q", "origin", "main")
	git(t, dir, "remote", "set-head", "origin", "main")
	git(t, dir, "switch", "-q", "-c", "feature")
	write(t, dir, "b.txt", "two\n")
	commitAll(t, dir, "second")
	return dir, remote
}

// remoteHas says whether the bare remote holds a branch, and at which commit.
func remoteHas(t *testing.T, remote, branch string) string {
	t.Helper()
	out, err := exec.Command("git", "--git-dir", remote, "rev-parse", "--verify", "--quiet", "refs/heads/"+branch).Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func calls(t *testing.T, d string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(d, "calls"))
	if err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}
	return string(b)
}

const openPR = `{"number":7,"url":"https://github.com/o/r/pull/7","title":"Add b","state":"OPEN","isDraft":true,` +
	`"baseRefName":"main","headRefName":"feature","statusCheckRollup":[` +
	`{"__typename":"CheckRun","name":"test","status":"COMPLETED","conclusion":"SUCCESS"},` +
	`{"__typename":"StatusContext","context":"ci/lint","state":"PENDING"}]}`

// The branch checked out now is the one asked about, and its PR comes back with
// the number, the address, the state and the checks said as one word and as
// counts.
func TestThePullRequestOfTheBranchIsReadWithItsChecks(t *testing.T) {
	gh := fakeGh(t)
	dir, _ := prRepo(t)
	put(t, gh, "pr.json", openPR)
	got, err := ReadPR(context.Background(), dir)
	if err != nil {
		t.Fatal(err)
	}
	if got.Branch != "feature" || got.Base != "main" || got.PR == nil {
		t.Fatalf("read %+v", got)
	}
	pr := *got.PR
	want := PullRequest{Number: 7, URL: "https://github.com/o/r/pull/7", Title: "Add b", State: PROpen, Draft: true,
		Base: "main", Head: "feature", Checks: Checks{State: ChecksPending, Passed: 1, Pending: 1}}
	if pr != want {
		t.Errorf("pr %+v, want %+v", pr, want)
	}
	if !strings.Contains(calls(t, gh), "pr\nview\nfeature\n") {
		t.Errorf("gh was not asked about the branch checked out now:\n%s", calls(t, gh))
	}
}

// A branch with no PR is an answer, not a failure: the base a new one would
// target comes back, and no PR.
func TestABranchWithNoPullRequestSaysSo(t *testing.T) {
	gh := fakeGh(t)
	dir, _ := prRepo(t)
	got, err := ReadPR(context.Background(), dir)
	if err != nil {
		t.Fatal(err)
	}
	if got.Branch != "feature" || got.Base != "main" || got.PR != nil {
		t.Errorf("read %+v", got)
	}
	if strings.Contains(calls(t, gh), "repo\nview") {
		t.Errorf("the clone says its default branch, and gh was asked anyway:\n%s", calls(t, gh))
	}
}

// A clone that never learned the remote's default branch asks GitHub for it.
func TestTheDefaultBranchComesFromGitHubWhenTheCloneDoesNotSay(t *testing.T) {
	gh := fakeGh(t)
	dir, _ := prRepo(t)
	git(t, dir, "remote", "set-head", "origin", "--delete")
	put(t, gh, "repo.json", `{"defaultBranchRef":{"name":"trunk"}}`)
	got, err := ReadPR(context.Background(), dir)
	if err != nil {
		t.Fatal(err)
	}
	if got.Base != "trunk" {
		t.Errorf("base %q, want the one gh names", got.Base)
	}
}

// The summary never says pass over a red check: one failure makes it fail, a
// check still running makes it pending, and a PR with no check says none.
func TestOneFailingCheckMakesTheSummaryFail(t *testing.T) {
	run := func(status, conclusion string) rollupItem {
		return rollupItem{Typename: "CheckRun", Status: status, Conclusion: conclusion}
	}
	status := func(state string) rollupItem { return rollupItem{Typename: "StatusContext", State: state} }
	for _, c := range []struct {
		name  string
		items []rollupItem
		want  Checks
	}{
		{"none", nil, Checks{State: ChecksNone}},
		{"green", []rollupItem{run("COMPLETED", "SUCCESS"), status("SUCCESS")}, Checks{State: ChecksPass, Passed: 2}},
		{"running", []rollupItem{run("COMPLETED", "SUCCESS"), run("IN_PROGRESS", "")}, Checks{State: ChecksPending, Passed: 1, Pending: 1}},
		{"red beats running", []rollupItem{run("QUEUED", ""), run("COMPLETED", "FAILURE")}, Checks{State: ChecksFail, Failed: 1, Pending: 1}},
		{"cancelled is red", []rollupItem{run("COMPLETED", "CANCELLED"), run("COMPLETED", "SUCCESS")}, Checks{State: ChecksFail, Passed: 1, Failed: 1}},
		{"timed out is red", []rollupItem{run("COMPLETED", "TIMED_OUT")}, Checks{State: ChecksFail, Failed: 1}},
		{"status error is red", []rollupItem{status("ERROR")}, Checks{State: ChecksFail, Failed: 1}},
		{"expected is waiting", []rollupItem{status("EXPECTED")}, Checks{State: ChecksPending, Pending: 1}},
		{"skipped is not green", []rollupItem{run("COMPLETED", "SKIPPED"), run("COMPLETED", "NEUTRAL")}, Checks{State: ChecksNone, Skipped: 2}},
		{"an unknown word is not green", []rollupItem{run("COMPLETED", "SOMETHING_NEW")}, Checks{State: ChecksFail, Failed: 1}},
	} {
		if got := summarize(c.items); got != c.want {
			t.Errorf("%s: %+v, want %+v", c.name, got, c.want)
		}
	}
}

// Opening a PR pushes the branch with an upstream, never by force, and only
// then asks gh to open it, with the title, body and base the person confirmed.
// Nothing merges and nothing turns auto-merge on.
func TestCreatingAPRPushesTheBranchThenOpensIt(t *testing.T) {
	gh := fakeGh(t)
	dir, remote := prRepo(t)
	put(t, gh, "created.json", strings.Replace(openPR, `"isDraft":true`, `"isDraft":false`, 1))
	got, err := CreatePR(context.Background(), dir, NewPR{Title: "  Add b  ", Body: "It adds b.\n\n- one\n", Base: "main"})
	if err != nil {
		t.Fatal(err)
	}
	if got.PR == nil || got.PR.Number != 7 || got.Branch != "feature" || got.Base != "main" {
		t.Fatalf("created %+v", got)
	}
	head, _ := exec.Command("git", "-C", dir, "rev-parse", "HEAD").Output()
	if remoteHas(t, remote, "feature") != strings.TrimSpace(string(head)) {
		t.Errorf("the branch was not pushed as it is")
	}
	upstream, _ := exec.Command("git", "-C", dir, "rev-parse", "--abbrev-ref", "feature@{upstream}").Output()
	if strings.TrimSpace(string(upstream)) != "origin/feature" {
		t.Errorf("the branch tracks %q, want origin/feature", upstream)
	}
	log := calls(t, gh)
	if !strings.Contains(log, "pr\ncreate\n--base=main\n--head=feature\n--title=Add b\n--body-file=-\n\n") {
		t.Errorf("gh was not asked to open the PR as confirmed:\n%s", log)
	}
	if body, _ := os.ReadFile(filepath.Join(gh, "body")); string(body) != "It adds b.\n\n- one\n" {
		t.Errorf("the body reached gh as %q", body)
	}
	for _, never := range []string{"merge", "--auto", "--force", "--fill"} {
		if strings.Contains(log, never) {
			t.Errorf("gh was called with %q:\n%s", never, log)
		}
	}
}

// A draft is asked for as one.
func TestADraftIsOpenedAsADraft(t *testing.T) {
	gh := fakeGh(t)
	dir, _ := prRepo(t)
	put(t, gh, "created.json", openPR)
	if _, err := CreatePR(context.Background(), dir, NewPR{Title: "Add b", Draft: true}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(calls(t, gh), "--body-file=-\n--draft\n") {
		t.Errorf("a draft was not asked for:\n%s", calls(t, gh))
	}
}

// A push the remote refuses is said with what git said, nothing is opened, and
// the remote keeps what it had: there is no force to fall back on.
func TestARejectedPushIsSaidAndNeverForced(t *testing.T) {
	gh := fakeGh(t)
	dir, remote := prRepo(t)
	other := t.TempDir()
	git(t, other, "clone", "-q", remote, ".")
	git(t, other, "config", "user.email", "t@example.com")
	git(t, other, "config", "user.name", "T")
	git(t, other, "config", "commit.gpgsign", "false")
	git(t, other, "switch", "-q", "-c", "feature")
	write(t, other, "c.txt", "elsewhere\n")
	commitAll(t, other, "theirs")
	git(t, other, "push", "-q", "origin", "feature")
	theirs := remoteHas(t, remote, "feature")

	_, err := CreatePR(context.Background(), dir, NewPR{Title: "Add b"})
	var refusal *Refusal
	if !errors.As(err, &refusal) || !strings.Contains(err.Error(), "git push") {
		t.Fatalf("got %v, want a refusal naming the push", err)
	}
	if remoteHas(t, remote, "feature") != theirs {
		t.Errorf("the remote's branch was overwritten")
	}
	if strings.Contains(calls(t, gh), "pr\ncreate") {
		t.Errorf("a PR was opened after a refused push")
	}
}

// What cannot become a PR is refused with the reason before anything is
// pushed: no branch, no remote, the base itself, nothing the base lacks, or a
// branch that already has an open PR.
func TestAPRThatCannotBeOpenedIsRefusedBeforeAnyPush(t *testing.T) {
	for _, c := range []struct {
		name, want string
		arrange    func(t *testing.T, dir, gh string)
	}{
		{"detached", "detached", func(t *testing.T, dir, _ string) { git(t, dir, "switch", "-q", "--detach") }},
		{"no remote", "origin", func(t *testing.T, dir, _ string) { git(t, dir, "remote", "remove", "origin") }},
		{"the base itself", "is the base", func(t *testing.T, dir, _ string) { git(t, dir, "switch", "-q", "main") }},
		{"nothing to propose", "no commit", func(t *testing.T, dir, _ string) {
			git(t, dir, "switch", "-q", "-c", "empty", "main")
		}},
		{"already open", "already has", func(t *testing.T, _, gh string) { put(t, gh, "pr.json", openPR) }},
		{"unknown base", "fetch", func(t *testing.T, dir, gh string) {
			put(t, gh, "repo.json", `{"defaultBranchRef":{"name":"trunk"}}`)
			git(t, dir, "remote", "set-head", "origin", "--delete")
		}},
	} {
		t.Run(c.name, func(t *testing.T) {
			gh := fakeGh(t)
			dir, remote := prRepo(t)
			c.arrange(t, dir, gh)
			_, err := CreatePR(context.Background(), dir, NewPR{Title: "Add b"})
			var refusal *Refusal
			if !errors.As(err, &refusal) || !strings.Contains(err.Error(), c.want) {
				t.Fatalf("got %v, want a refusal saying %q", err, c.want)
			}
			for _, b := range []string{"feature", "empty"} {
				if remoteHas(t, remote, b) != "" {
					t.Errorf("%s was pushed before the refusal", b)
				}
			}
			if strings.Contains(calls(t, gh), "pr\ncreate") {
				t.Errorf("a PR was opened:\n%s", calls(t, gh))
			}
		})
	}
}

// Without gh, or with gh not logged in, there is no PR to read or open, and
// that is what is said — never "no PR".
func TestWithoutGhThePRIsNotReadOrOpened(t *testing.T) {
	gh := fakeGh(t)
	dir, _ := prRepo(t)
	put(t, gh, "logged-out", "")
	if _, err := ReadPR(context.Background(), dir); !errors.Is(err, ErrGhLoggedOut) {
		t.Errorf("read logged out: %v, want ErrGhLoggedOut", err)
	}
	if _, err := CreatePR(context.Background(), dir, NewPR{Title: "Add b"}); !errors.Is(err, ErrGhLoggedOut) {
		t.Errorf("create logged out: %v, want ErrGhLoggedOut", err)
	}

	gitPath, err := exec.LookPath("git")
	if err != nil {
		t.Skip("no git here")
	}
	only := t.TempDir()
	if err := os.Symlink(gitPath, filepath.Join(only, "git")); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", only)
	if _, err := ReadPR(context.Background(), dir); !errors.Is(err, ErrNoGh) {
		t.Errorf("read without gh: %v, want ErrNoGh", err)
	}
	if _, err := CreatePR(context.Background(), dir, NewPR{Title: "Add b"}); !errors.Is(err, ErrNoGh) {
		t.Errorf("create without gh: %v, want ErrNoGh", err)
	}
}

// A request is checked before gh or a push runs: a title that is empty, more
// than one line or too long, a body past the cap, a base that is no branch name.
func TestAPRRequestIsValidatedBeforeAnythingRuns(t *testing.T) {
	gh := fakeGh(t)
	dir, remote := prRepo(t)
	for _, req := range []NewPR{
		{Title: "   "},
		{Title: "two\nlines"},
		{Title: strings.Repeat("x", PRTitleLimit+1)},
		{Title: "ok", Body: strings.Repeat("x", PRBodyLimit+1)},
		{Title: "ok", Base: "-x"},
		{Title: "ok", Base: "a..b"},
	} {
		if _, err := CreatePR(context.Background(), dir, req); !errors.Is(err, ErrInvalidPR) {
			t.Errorf("%.40q: %v, want ErrInvalidPR", req.Title+"|"+req.Base, err)
		}
	}
	if calls(t, gh) != "" || remoteHas(t, remote, "feature") != "" {
		t.Errorf("something ran for a request that was refused:\n%s", calls(t, gh))
	}
}

// gh runs with nobody to ask: no terminal on its input, prompts off, and the
// repository it is pointed at is the folder's, never one the daemon's
// environment names.
func TestGhRunsWithoutAPromptOrAnInheritedRepository(t *testing.T) {
	gh := fakeGh(t)
	dir, _ := prRepo(t)
	t.Setenv("GH_REPO", "someone/else")
	if _, err := ReadPR(context.Background(), dir); err != nil {
		t.Fatal(err)
	}
	env, err := os.ReadFile(filepath.Join(gh, "env"))
	if err != nil {
		t.Fatal(err)
	}
	for _, line := range strings.Split(strings.TrimSpace(string(env)), "\n") {
		if line != "prompt=1 repo=unset tty=no" {
			t.Errorf("gh ran with %q", line)
		}
	}
}
