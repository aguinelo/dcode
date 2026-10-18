package orchestrate

// The forge: where a child's commit becomes main. On GitHub each child is a
// PR, squash-merged by gh with --match-head-commit on the measured commit;
// with no forge the merge is local, and what that loses is said (Q-G).
//
// Spec: docs/specs/architecture/orchestration/202610132000-orchestration.p.spec.md §4, §7.5, §7.6;
// .config.spec.md §3.

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

	"github.com/aguinelo/dcode/internal/config"
	"github.com/aguinelo/dcode/internal/protocol"
	"github.com/aguinelo/dcode/internal/vcs"
)

// The values of orchestrator.forge.
const (
	ForgeAuto   = "auto"
	ForgeGitHub = "github"
	ForgeLocal  = "local"
)

// The four words of MainHealth.Health (.p §5.2).
const (
	healthGreen   = "green"
	healthRed     = "red"
	healthPending = "pending"
	healthUnknown = "unknown"
)

const (
	// The bounds and default of orchestrator.ci_timeout (.config §3).
	ciTimeoutDefault = 45 * time.Minute
	ciTimeoutMin     = time.Minute
	ciTimeoutMax     = 6 * time.Hour
	// The bounds and default of orchestrator.sensitive_diff_lines.
	sensitiveLinesDefault = 800
	sensitiveLinesMin     = 50
	sensitiveLinesMax     = 100000
	// forgeDeadline bounds one call to the forge: a push, a gh, a fetch.
	forgeDeadline = 2 * time.Minute
)

// defaultSensitive is orchestrator.sensitive's default (Q-E, .config §3).
var defaultSensitive = []string{".github/**", "scripts/**", "Makefile", "go.mod", "go.sum",
	"**/package.json", "**/package-lock.json", ".dcode/**", "**/*.r.spec.md"}

// MergeConfig is the [orchestrator] keys of the merge.
type MergeConfig struct {
	// Forge is auto, github or local.
	Forge string
	// CITimeout bounds the wait for main's CI after a merge; pending past it
	// is unknown, which stops the queue as red does.
	CITimeout time.Duration
	// Sensitive are the policy.Glob patterns that mark a merge question.
	Sensitive []string
	// SensitiveDiffLines is the diff size past which a question is marked.
	SensitiveDiffLines int
}

// MergeFrom reads the merge's keys. A value that does not hold is an error
// naming the key and the value, never the default in silence.
func MergeFrom(r config.Resolved) (MergeConfig, error) {
	c := MergeConfig{Forge: ForgeAuto, CITimeout: ciTimeoutDefault, Sensitive: defaultSensitive,
		SensitiveDiffLines: sensitiveLinesDefault}
	if v := strings.TrimSpace(r.String("orchestrator.forge", "")); v != "" {
		if v != ForgeAuto && v != ForgeGitHub && v != ForgeLocal {
			return MergeConfig{}, fmt.Errorf("orchestrator.forge must be auto, github or local, got %q", v)
		}
		c.Forge = v
	}
	if v := strings.TrimSpace(r.String("orchestrator.ci_timeout", "")); v != "" {
		d, err := time.ParseDuration(v)
		if err != nil || d < ciTimeoutMin || d > ciTimeoutMax {
			return MergeConfig{}, fmt.Errorf("orchestrator.ci_timeout must be a duration from 1m to 6h, got %q", v)
		}
		c.CITimeout = d
	}
	if v := r.String("orchestrator.sensitive", ""); strings.TrimSpace(v) != "" {
		c.Sensitive = nil
		for _, g := range strings.Split(v, ",") {
			if g = strings.TrimSpace(g); g != "" {
				c.Sensitive = append(c.Sensitive, g)
			}
		}
	}
	if v := strings.TrimSpace(r.String("orchestrator.sensitive_diff_lines", "")); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < sensitiveLinesMin || n > sensitiveLinesMax {
			return MergeConfig{}, fmt.Errorf("orchestrator.sensitive_diff_lines must be an integer from %d to %d, got %q",
				sensitiveLinesMin, sensitiveLinesMax, v)
		}
		c.SensitiveDiffLines = n
	}
	return c, nil
}

// Forge is where a measured commit becomes main.
type Forge interface {
	// Name is github or local.
	Name() string
	// Tips is main and branch as the forge has them now, fetched first when
	// the forge is remote.
	Tips(ctx context.Context, branch string) (main, head string, err error)
	// OpenPR pushes branch, never forced, and opens its PR. Local opens
	// nothing and pushes nothing.
	OpenPR(ctx context.Context, branch, title, body string) (protocol.ChildPR, error)
	// Merge squashes c into main, on the condition that what merges is
	// measured, and returns the commit on main.
	Merge(ctx context.Context, c protocol.Child, measured string) (string, error)
	// MainHealth is one reading of main's health at commit.
	MainHealth(ctx context.Context, commit string) (protocol.MainHealth, error)
	// MainTip is main's tip as the forge has it now: origin's after a fetch
	// on github, the local branch on local. A fetch that fails is an error,
	// never the local main in its place.
	MainTip(ctx context.Context) (string, error)
	// UpdatePR pushes branch again, never forced, and refreshes pr's body;
	// a child measured again keeps the PR it has. Local has none.
	UpdatePR(ctx context.Context, pr protocol.ChildPR, branch, body string) (protocol.ChildPR, error)
}

// ForgeSetup is what NewForge needs. Integration and Gate are the local
// forge's: where it squashes, and what says main's health without a CI.
type ForgeSetup struct {
	Declared, Root, Main string
	Integration          *Integration
	Gate                 Gate
	Now                  func() time.Time
}

// NewForge is the forge orchestrator.forge declares. auto is github when the
// repository has an origin and gh answers, local otherwise; github declared
// without a gh that answers is an error, never a quiet fall to local.
func NewForge(ctx context.Context, s ForgeSetup) (Forge, error) {
	if s.Root == "" || s.Main == "" {
		return nil, errors.New("orchestrate: a forge needs the project's root and its main branch")
	}
	if s.Now == nil {
		s.Now = time.Now
	}
	switch s.Declared {
	case ForgeGitHub:
		if err := ghAnswers(ctx, s.Root); err != nil {
			return nil, fmt.Errorf("orchestrator.forge is github and %w", err)
		}
		return &githubForge{root: s.Root, main: s.Main, now: s.Now}, nil
	case ForgeAuto, "":
		if hasOrigin(ctx, s.Root) && ghAnswers(ctx, s.Root) == nil {
			return &githubForge{root: s.Root, main: s.Main, now: s.Now}, nil
		}
		return newLocal(s)
	case ForgeLocal:
		return newLocal(s)
	}
	return nil, fmt.Errorf("orchestrator.forge must be auto, github or local, got %q", s.Declared)
}

func hasOrigin(ctx context.Context, root string) bool {
	_, err := forgeGit(ctx, root, "remote", "get-url", "origin")
	return err == nil
}

// ghAnswers says why gh cannot speak for the repository, or nil when it can.
func ghAnswers(ctx context.Context, root string) error {
	if _, err := exec.LookPath("gh"); err != nil {
		return vcs.ErrNoGh
	}
	if _, stderr, err := ghRun(ctx, root, nil, "auth", "status"); err != nil {
		return fmt.Errorf("%w (%s)", vcs.ErrGhLoggedOut, firstOf(stderr, err))
	}
	return nil
}

// githubForge is a project whose origin is on GitHub, spoken to through gh.
type githubForge struct {
	root, main string
	now        func() time.Time
}

func (g *githubForge) Name() string { return ForgeGitHub }

// MainTip fetches main from origin and reads origin's: a merge on GitHub
// moves origin's main, never the local one.
func (g *githubForge) MainTip(ctx context.Context) (string, error) {
	if _, err := forgeGit(ctx, g.root, "fetch", "-q", "origin", g.main); err != nil {
		return "", fmt.Errorf("orchestrate: fetch origin %s, which main's tip is read from with forge github: %w", g.main, err)
	}
	return forgeGit(ctx, g.root, "rev-parse", "--verify", "refs/remotes/origin/"+g.main)
}

// UpdatePR pushes the branch again — a plain push, never forced: the branch
// only moved forward — and refreshes the PR's body, whose evidence names the
// commit measured.
func (g *githubForge) UpdatePR(ctx context.Context, pr protocol.ChildPR, branch, body string) (protocol.ChildPR, error) {
	if len(body) > vcs.PRBodyLimit {
		return pr, fmt.Errorf("orchestrate: the PR body is %d bytes, over GitHub's %d", len(body), vcs.PRBodyLimit)
	}
	if _, err := forgeGit(ctx, g.root, "push", "-q", "origin", "refs/heads/"+branch+":refs/heads/"+branch); err != nil {
		return pr, err
	}
	if _, stderr, err := ghRun(ctx, g.root, strings.NewReader(body), "pr", "edit", strconv.Itoa(pr.Number), "--body-file", "-"); err != nil {
		return pr, fmt.Errorf("orchestrate: gh pr edit #%d: %s", pr.Number, firstOf(stderr, err))
	}
	pr.CI = vcs.ChecksPending
	return pr, nil
}

func (g *githubForge) Tips(ctx context.Context, branch string) (string, string, error) {
	if _, err := forgeGit(ctx, g.root, "fetch", "-q", "origin", g.main, branch); err != nil {
		return "", "", err
	}
	main, err := forgeGit(ctx, g.root, "rev-parse", "--verify", "refs/remotes/origin/"+g.main)
	if err != nil {
		return "", "", err
	}
	head, err := forgeGit(ctx, g.root, "rev-parse", "--verify", "refs/remotes/origin/"+branch)
	if err != nil {
		return "", "", err
	}
	return main, head, nil
}

// OpenPR pushes the branch with a plain push — never forced: a branch the
// remote refuses is said, never overwritten — and opens the PR against main.
func (g *githubForge) OpenPR(ctx context.Context, branch, title, body string) (protocol.ChildPR, error) {
	if len(body) > vcs.PRBodyLimit {
		return protocol.ChildPR{}, fmt.Errorf("orchestrate: the PR body is %d bytes, over GitHub's %d", len(body), vcs.PRBodyLimit)
	}
	if _, err := forgeGit(ctx, g.root, "push", "-q", "-u", "origin", "refs/heads/"+branch+":refs/heads/"+branch); err != nil {
		return protocol.ChildPR{}, err
	}
	out, stderr, err := ghRun(ctx, g.root, strings.NewReader(body), "pr", "create", "--base", g.main, "--head", branch,
		"--title", title, "--body-file", "-")
	if err != nil {
		return protocol.ChildPR{}, fmt.Errorf("orchestrate: gh pr create for %s: %s", branch, firstOf(stderr, err))
	}
	url := lastLine(string(out))
	_, num, found := strings.Cut(url, "/pull/")
	n, cerr := strconv.Atoi(num)
	if !found || cerr != nil || n <= 0 {
		return protocol.ChildPR{}, fmt.Errorf("orchestrate: gh pr create wrote %q, which is not a PR's address", url)
	}
	return protocol.ChildPR{Number: n, URL: url, CI: vcs.ChecksPending}, nil
}

// Merge asks GitHub for the squash with --match-head-commit: the server
// merges the commit measured or nothing. The harness's branch is deleted on
// origin after; failing that is said beside the merge, which stands.
func (g *githubForge) Merge(ctx context.Context, c protocol.Child, measured string) (string, error) {
	if c.PR == nil || c.PR.Number <= 0 {
		return "", fmt.Errorf("orchestrate: %s has no PR to merge", c.ID)
	}
	n := strconv.Itoa(c.PR.Number)
	if _, stderr, err := ghRun(ctx, g.root, nil, "pr", "merge", n, "--squash", "--match-head-commit", measured); err != nil {
		return "", &forgeRefusal{reason: fmt.Sprintf("GitHub refused to merge #%s at %s: %s", n, short(measured), firstOf(stderr, err))}
	}
	out, stderr, err := ghRun(ctx, g.root, nil, "pr", "view", n, "--json", "state,mergeCommit")
	if err != nil {
		return "", fmt.Errorf("orchestrate: #%s merged, and reading where it landed failed: %s", n, firstOf(stderr, err))
	}
	var v struct {
		State       string `json:"state"`
		MergeCommit struct {
			OID string `json:"oid"`
		} `json:"mergeCommit"`
	}
	if err := json.Unmarshal(out, &v); err != nil || v.State != "MERGED" || !exactCommit.MatchString(v.MergeCommit.OID) {
		return "", fmt.Errorf("orchestrate: gh says #%s is %q at %q after the merge, not merged at a commit", n, v.State, v.MergeCommit.OID)
	}
	if _, err := forgeGit(ctx, g.root, "push", "-q", "origin", "--delete", c.Branch); err != nil {
		return v.MergeCommit.OID, &branchLeft{branch: c.Branch, err: err}
	}
	return v.MergeCommit.OID, nil
}

// MainHealth reads the CI runs GitHub has for commit. A failure beats a run
// still going, which beats a pass; no run yet is pending, and a conclusion
// GitHub may add later counts as a failure, never as a pass.
func (g *githubForge) MainHealth(ctx context.Context, commit string) (protocol.MainHealth, error) {
	h := protocol.MainHealth{Commit: commit, Health: healthUnknown, Checked: g.now().UTC()}
	out, stderr, err := ghRun(ctx, g.root, nil, "run", "list", "--commit", commit, "--json", "status,conclusion", "--limit", "100")
	if err != nil {
		return h, fmt.Errorf("orchestrate: read CI of %s: %s", short(commit), firstOf(stderr, err))
	}
	var runs []struct{ Status, Conclusion string }
	if err := json.Unmarshal(out, &runs); err != nil {
		return h, fmt.Errorf("orchestrate: gh run list wrote what is not a list of runs: %v", err)
	}
	failed, pending := false, len(runs) == 0
	for _, r := range runs {
		switch {
		case r.Status != "completed":
			pending = true
		case r.Conclusion == "success" || r.Conclusion == "skipped" || r.Conclusion == "neutral":
		default:
			failed = true
		}
	}
	switch {
	case failed:
		h.Health = healthRed
	case pending:
		h.Health = healthPending
	default:
		h.Health = healthGreen
	}
	return h, nil
}

// forgeRefusal is the forge saying no: merge.refused, not a failure.
type forgeRefusal struct{ reason string }

func (r *forgeRefusal) Error() string { return r.reason }

// branchLeft is a merge that stands with the harness's branch still on origin.
type branchLeft struct {
	branch string
	err    error
}

func (b *branchLeft) Error() string {
	return fmt.Sprintf("the merge stands, and the branch %s is still on origin: %v", b.branch, b.err)
}

// ghRun runs gh in dir with nothing on its input but in, and nobody to ask.
func ghRun(ctx context.Context, dir string, in *strings.Reader, args ...string) ([]byte, string, error) {
	ctx, cancel := context.WithTimeout(ctx, forgeDeadline)
	defer cancel()
	cmd := exec.CommandContext(ctx, "gh", args...)
	cmd.Dir = dir
	cmd.Env = append(gitEnv(cmd.Environ()), "GH_PROMPT_DISABLED=1", "GH_NO_UPDATE_NOTIFIER=1", "NO_COLOR=1",
		"GH_PAGER=cat", "GIT_TERMINAL_PROMPT=0")
	if in != nil {
		cmd.Stdin = in
	}
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	return out, stderr.String(), err
}

func forgeGit(ctx context.Context, dir string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, forgeDeadline)
	defer cancel()
	return gateGit(ctx, dir, args...)
}

func firstOf(text string, err error) string {
	for _, l := range strings.Split(text, "\n") {
		if l = strings.TrimSpace(l); l != "" {
			return l
		}
	}
	return err.Error()
}

func lastLine(text string) string {
	lines := strings.Split(strings.TrimSpace(text), "\n")
	return strings.TrimSpace(lines[len(lines)-1])
}
