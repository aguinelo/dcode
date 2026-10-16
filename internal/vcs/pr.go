package vcs

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

// A branch's pull request, read and opened through the GitHub CLI the person
// already has. gh holds the login; the daemon never sees a token. Reading asks
// gh what GitHub knows; opening pushes the branch and asks gh to open it, and
// is only ever done for a request a person confirmed. Nothing here merges, and
// nothing pushes by force.

// ErrNoGh is the answer when gh is not there to ask.
var ErrNoGh = errors.New("gh is not installed: it is not on the daemon's PATH")

// ErrGhLoggedOut is gh present without a login it can use.
var ErrGhLoggedOut = errors.New("gh is not logged in: run `gh auth login` where the daemon runs")

// ErrInvalidPR is a request to open a PR that is malformed in itself.
var ErrInvalidPR = errors.New("not a pull request that can be asked for")

// Refusal is a branch that cannot become a PR as things stand, and why: no
// branch, no remote, the base itself, nothing to propose, a PR already open,
// a push the remote refused.
type Refusal struct{ Reason string }

func (r *Refusal) Error() string { return r.Reason }

func refuse(format string, args ...any) error { return &Refusal{Reason: fmt.Sprintf(format, args...)} }

// How a PR's checks add up, as Checks.State says it.
const (
	ChecksPass    = "pass"
	ChecksFail    = "fail"
	ChecksPending = "pending"
	ChecksNone    = "none"
)

// How a PR stands, as PullRequest.State says it.
const (
	PROpen   = "open"
	PRClosed = "closed"
	PRMerged = "merged"
)

// PRTitleLimit is the longest title asked for, in characters: GitHub's own.
const PRTitleLimit = 256

// PRBodyLimit is the longest body asked for, in bytes: GitHub's own cap.
const PRBodyLimit = 65536

// prReadDeadline bounds one read: gh asks GitHub, over the network.
const prReadDeadline = 20 * time.Second

// prCreateDeadline bounds opening one: a push and two calls to GitHub.
const prCreateDeadline = 2 * time.Minute

// Checks is a PR's checks, counted and summed up in one word.
type Checks struct {
	State                            string
	Passed, Failed, Pending, Skipped int
}

// PullRequest is one PR as GitHub has it.
type PullRequest struct {
	Number     int
	URL        string
	Title      string
	State      string
	Draft      bool
	Base, Head string
	Checks     Checks
}

// BranchPR is the branch checked out now, the base a new PR would target, and
// the PR the branch has, if any.
type BranchPR struct {
	Branch string
	Base   string
	PR     *PullRequest
}

// NewPR is what a person confirmed: an empty Base is the repository's default.
type NewPR struct {
	Title, Body, Base string
	Draft             bool
}

// ReadPR answers the PR of the branch dir has checked out now.
func ReadPR(ctx context.Context, dir string) (*BranchPR, error) {
	if err := toolsThere(); err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(ctx, prReadDeadline)
	defer cancel()
	root, branch, err := prBranch(ctx, dir)
	if err != nil {
		return nil, ranOut(ctx, prReadDeadline, err)
	}
	out := &BranchPR{Branch: branch}
	if out.Base, err = defaultBase(ctx, root); err != nil {
		return nil, ranOut(ctx, prReadDeadline, err)
	}
	if out.PR, err = viewPR(ctx, root, branch); err != nil {
		return nil, ranOut(ctx, prReadDeadline, err)
	}
	return out, nil
}

// CreatePR pushes the branch dir has checked out and opens a PR for it, as req
// says. Everything that can refuse refuses before the push.
func CreatePR(ctx context.Context, dir string, req NewPR) (*BranchPR, error) {
	req.Title = strings.TrimSpace(req.Title)
	if err := req.check(); err != nil {
		return nil, err
	}
	if err := toolsThere(); err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(ctx, prCreateDeadline)
	defer cancel()
	out, err := createPR(ctx, dir, req)
	if err != nil {
		return nil, ranOut(ctx, prCreateDeadline, err)
	}
	return out, nil
}

func createPR(ctx context.Context, dir string, req NewPR) (*BranchPR, error) {
	if req.Base != "" {
		if _, err := gitOut(ctx, dir, "check-ref-format", "--branch", req.Base); err != nil {
			return nil, fmt.Errorf("%w: %q is not a branch name", ErrInvalidPR, req.Base)
		}
	}
	if _, stderr, err := gh(ctx, dir, nil, "auth", "status"); err != nil {
		return nil, fmt.Errorf("%w (%s)", ErrGhLoggedOut, firstLine(stderr, err))
	}
	root, branch, err := prBranch(ctx, dir)
	if err != nil {
		return nil, err
	}
	base := req.Base
	if base == "" {
		if base, err = defaultBase(ctx, root); err != nil {
			return nil, err
		}
	}
	if base == branch {
		return nil, refuse("the branch %s is the base itself: a PR needs a branch of its own", branch)
	}
	if existing, err := viewPR(ctx, root, branch); err != nil {
		return nil, err
	} else if existing != nil && existing.State == PROpen {
		return nil, refuse("the branch %s already has PR #%d open: %s", branch, existing.Number, existing.URL)
	}
	if err := somethingToPropose(ctx, root, branch, base); err != nil {
		return nil, err
	}
	push := gitCmd(ctx, root, "push", "-u", "origin", branch)
	push.Env = append(push.Env, "GIT_TERMINAL_PROMPT=0")
	var pushErr bytes.Buffer
	push.Stderr = &pushErr
	if err := push.Run(); err != nil {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		return nil, refuse("git push -u origin %s was refused: %s", branch, gitSaid(pushErr.String(), err))
	}
	args := []string{"pr", "create", "--base=" + base, "--head=" + branch, "--title=" + req.Title, "--body-file=-"}
	if req.Draft {
		args = append(args, "--draft")
	}
	if _, stderr, err := gh(ctx, root, strings.NewReader(req.Body), args...); err != nil {
		return nil, fmt.Errorf("the branch was pushed, and gh pr create failed: %s", firstLine(stderr, err))
	}
	pr, err := viewPR(ctx, root, branch)
	if err != nil {
		return nil, err
	}
	if pr == nil {
		return nil, fmt.Errorf("gh pr create answered, and gh pr view finds no PR for %s", branch)
	}
	return &BranchPR{Branch: branch, Base: base, PR: pr}, nil
}

func (req NewPR) check() error {
	switch {
	case req.Title == "":
		return fmt.Errorf("%w: a title is required", ErrInvalidPR)
	case strings.ContainsAny(req.Title, "\r\n"):
		return fmt.Errorf("%w: a title is one line", ErrInvalidPR)
	case utf8.RuneCountInString(req.Title) > PRTitleLimit:
		return fmt.Errorf("%w: the title is longer than %d characters", ErrInvalidPR, PRTitleLimit)
	case len(req.Body) > PRBodyLimit:
		return fmt.Errorf("%w: the body is longer than %d bytes", ErrInvalidPR, PRBodyLimit)
	case strings.HasPrefix(req.Base, "-"):
		return fmt.Errorf("%w: %q is not a branch name", ErrInvalidPR, req.Base)
	}
	return nil
}

func toolsThere() error {
	if _, err := exec.LookPath("git"); err != nil {
		return ErrNoGit
	}
	if _, err := exec.LookPath("gh"); err != nil {
		return ErrNoGh
	}
	return nil
}

// ranOut says a deadline ran out rather than whatever the cancelled command
// happened to print.
func ranOut(ctx context.Context, d time.Duration, err error) error {
	if ctx.Err() != nil {
		return fmt.Errorf("git and gh did not answer within %s", d)
	}
	return err
}

// prBranch is the top of dir's work tree and the branch checked out there,
// refused when there is none or no origin to push it to.
func prBranch(ctx context.Context, dir string) (root, branch string, err error) {
	top, err := gitOut(ctx, dir, "rev-parse", "--show-toplevel")
	if err != nil {
		var ge *gitError
		if errors.As(err, &ge) && strings.Contains(ge.stderr, "not a git repository") {
			return "", "", refuse("the folder is in no git repository")
		}
		return "", "", err
	}
	root = strings.TrimSpace(top)
	b, err := gitOut(ctx, root, "symbolic-ref", "--quiet", "--short", "HEAD")
	if err != nil {
		if ctx.Err() != nil {
			return "", "", ctx.Err()
		}
		return "", "", refuse("HEAD is detached: a PR is of a branch, and none is checked out")
	}
	if _, err := gitOut(ctx, root, "remote", "get-url", "origin"); err != nil {
		if ctx.Err() != nil {
			return "", "", ctx.Err()
		}
		return "", "", refuse("the repository has no remote named origin to push to")
	}
	return root, strings.TrimSpace(b), nil
}

// defaultBase is the branch a new PR targets: the remote's default as the
// clone knows it, or as GitHub says when the clone does not.
func defaultBase(ctx context.Context, root string) (string, error) {
	if ref, err := gitOut(ctx, root, "symbolic-ref", "--quiet", "--short", "refs/remotes/origin/HEAD"); err == nil {
		if b := strings.TrimPrefix(strings.TrimSpace(ref), "origin/"); b != "" {
			return b, nil
		}
	}
	out, stderr, err := gh(ctx, root, nil, "repo", "view", "--json", "defaultBranchRef")
	if err != nil {
		return "", ghFailure("gh repo view", stderr, err)
	}
	var repo struct {
		DefaultBranchRef struct{ Name string } `json:"defaultBranchRef"`
	}
	if err := json.Unmarshal(out, &repo); err != nil || repo.DefaultBranchRef.Name == "" {
		return "", fmt.Errorf("gh repo view did not name a default branch: %q", firstLine(string(out), err))
	}
	return repo.DefaultBranchRef.Name, nil
}

// somethingToPropose refuses a branch with no commit its base lacks, and a
// base the clone has never seen.
func somethingToPropose(ctx context.Context, root, branch, base string) error {
	var ref string
	for _, r := range []string{"refs/remotes/origin/" + base, "refs/heads/" + base} {
		if _, err := gitOut(ctx, root, "rev-parse", "--verify", "--quiet", r); err == nil {
			ref = r
			break
		}
	}
	if ref == "" {
		return refuse("the base %s is not known in this clone: run git fetch origin %s first", base, base)
	}
	n, err := gitOut(ctx, root, "rev-list", "--count", ref+"..HEAD")
	if err != nil {
		return err
	}
	if c, _ := strconv.Atoi(strings.TrimSpace(n)); c == 0 {
		return refuse("the branch %s has no commit that %s does not have: there is nothing to propose", branch, base)
	}
	return nil
}

// ghView is what `gh pr view --json` writes.
type ghView struct {
	Number      int          `json:"number"`
	URL         string       `json:"url"`
	Title       string       `json:"title"`
	State       string       `json:"state"`
	IsDraft     bool         `json:"isDraft"`
	BaseRefName string       `json:"baseRefName"`
	HeadRefName string       `json:"headRefName"`
	Rollup      []rollupItem `json:"statusCheckRollup"`
}

// rollupItem is one check: a CheckRun (status, conclusion) or a commit status
// (state), as GitHub's rollup lists both.
type rollupItem struct {
	Typename   string `json:"__typename"`
	Status     string `json:"status"`
	Conclusion string `json:"conclusion"`
	State      string `json:"state"`
}

// viewPR is the PR gh finds for branch, or nil when it finds none.
func viewPR(ctx context.Context, root, branch string) (*PullRequest, error) {
	out, stderr, err := gh(ctx, root, nil, "pr", "view", branch, "--json",
		"number,url,title,state,isDraft,baseRefName,headRefName,statusCheckRollup")
	if err != nil {
		if strings.Contains(stderr, "no pull requests found") {
			return nil, nil
		}
		return nil, ghFailure("gh pr view", stderr, err)
	}
	var v ghView
	if err := json.Unmarshal(out, &v); err != nil {
		return nil, fmt.Errorf("gh pr view wrote what is not a PR: %v", err)
	}
	state, ok := map[string]string{"OPEN": PROpen, "CLOSED": PRClosed, "MERGED": PRMerged}[v.State]
	if !ok {
		return nil, fmt.Errorf("gh pr view said the PR is %q, which is none of open, closed or merged", v.State)
	}
	return &PullRequest{Number: v.Number, URL: v.URL, Title: v.Title, State: state,
		Draft: v.IsDraft, Base: v.BaseRefName, Head: v.HeadRefName, Checks: summarize(v.Rollup)}, nil
}

// summarize counts the checks and sums them up. A failure beats a check still
// running, which beats a pass: the summary is never green over a red check. A
// word GitHub may add later counts as a failure, never as a pass.
func summarize(items []rollupItem) Checks {
	var c Checks
	for _, it := range items {
		word := it.Conclusion
		if it.Typename == "StatusContext" {
			word = it.State
		} else if it.Status != "COMPLETED" {
			word = "PENDING"
		}
		switch word {
		case "SUCCESS":
			c.Passed++
		case "PENDING", "EXPECTED":
			c.Pending++
		case "SKIPPED", "NEUTRAL":
			c.Skipped++
		default:
			c.Failed++
		}
	}
	switch {
	case c.Failed > 0:
		c.State = ChecksFail
	case c.Pending > 0:
		c.State = ChecksPending
	case c.Passed > 0:
		c.State = ChecksPass
	default:
		c.State = ChecksNone
	}
	return c
}

// gh runs the GitHub CLI in dir: never through a shell, with nothing on its
// input but in, and with nobody to ask.
func gh(ctx context.Context, dir string, in *strings.Reader, args ...string) (stdout []byte, stderr string, err error) {
	cmd := exec.CommandContext(ctx, "gh", args...)
	cmd.Dir = dir
	cmd.Env = ghEnv(cmd.Environ())
	if in != nil {
		cmd.Stdin = in
	}
	var errBuf bytes.Buffer
	cmd.Stderr = &errBuf
	out, err := cmd.Output()
	return out, errBuf.String(), err
}

// ghEnv is the daemon's environment as gh gets it. What would point gh at
// another repository than the folder's, or make it wait for a person, is
// taken out; the login (GH_TOKEN, GH_HOST, the config folder) stays.
func ghEnv(env []string) []string {
	drop := map[string]bool{"GH_REPO": true, "GIT_DIR": true, "GIT_WORK_TREE": true, "GIT_INDEX_FILE": true,
		"GH_FORCE_TTY": true, "GH_PAGER": true, "PAGER": true, "GH_BROWSER": true, "BROWSER": true}
	out := env[:0:0]
	for _, kv := range env {
		if k, _, _ := strings.Cut(kv, "="); !drop[k] {
			out = append(out, kv)
		}
	}
	return append(out, "GH_PROMPT_DISABLED=1", "GH_NO_UPDATE_NOTIFIER=1", "GH_SPINNER_DISABLED=1",
		"NO_COLOR=1", "CLICOLOR=0", "GH_PAGER=cat", "GIT_TERMINAL_PROMPT=0")
}

// ghFailure says why gh failed. Exit 4 is gh's "authentication required".
func ghFailure(what, stderr string, err error) error {
	var exit *exec.ExitError
	if errors.As(err, &exit) && exit.ExitCode() == 4 {
		return fmt.Errorf("%w (%s)", ErrGhLoggedOut, firstLine(stderr, err))
	}
	return fmt.Errorf("%s failed: %s", what, firstLine(stderr, err))
}

// gitSaid is what git wrote about a failed push, its hints left out and the
// lines joined, cut at a length a refusal can carry.
func gitSaid(text string, err error) string {
	var said []string
	for _, l := range strings.Split(text, "\n") {
		if l = strings.TrimSpace(l); l != "" && !strings.HasPrefix(l, "hint:") {
			said = append(said, l)
		}
	}
	s := strings.Join(said, "; ")
	if s == "" {
		return firstLine("", err)
	}
	if len(s) > 600 {
		s = s[:600] + "…"
	}
	return s
}

// firstLine is the first non-empty line of what a command said, or the error
// when it said nothing.
func firstLine(text string, err error) string {
	for _, l := range strings.Split(text, "\n") {
		if l = strings.TrimSpace(l); l != "" {
			return l
		}
	}
	if err != nil {
		return err.Error()
	}
	return "it said nothing"
}
