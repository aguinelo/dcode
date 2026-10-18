package orchestrate

// The review of a child's work: what the reviewer is given to read, and the
// owns check that decides, before any model is asked, whether there is
// anything to review at all (RN-5).
//
// Spec: docs/specs/architecture/orchestration/202610132000-orchestration.p.spec.md §7.1, §7.2;
// .config.spec.md §2, §3.

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/aguinelo/dcode/internal/config"
	"github.com/aguinelo/dcode/internal/policy"
	"github.com/aguinelo/dcode/internal/protocol"
)

const (
	// The bounds and default of orchestrator.review_max_diff_bytes (.config §3).
	reviewMaxDiffDefault = 131072
	reviewMaxDiffMin     = 4096
	reviewMaxDiffMax     = 1 << 20
	// reviewGitDeadline bounds each git the review assembly runs.
	reviewGitDeadline = 2 * time.Minute
)

// ReviewConfig is the [orchestrator] keys the review reads.
type ReviewConfig struct {
	// Profiles is orchestrator.reviewer_profiles, the allow-list of RN-8.
	// Empty is the first of orchestrator.child_profiles, which the caller
	// passes to Profile.
	Profiles []string
	// MaxDiffBytes is orchestrator.review_max_diff_bytes: a diff past it is
	// cut whole and said, never truncated.
	MaxDiffBytes int
}

// ReviewFrom reads the review's keys. known says whether a models.toml
// profile exists; a listed name that does not is an error naming the key and
// the value, as a bound that does not hold is.
func ReviewFrom(r config.Resolved, known func(string) bool) (ReviewConfig, error) {
	c := ReviewConfig{MaxDiffBytes: reviewMaxDiffDefault}
	raw := r.String("orchestrator.reviewer_profiles", "")
	for _, p := range strings.Split(raw, ",") {
		if p = strings.TrimSpace(p); p == "" {
			continue
		}
		if known == nil || !known(p) {
			return ReviewConfig{}, fmt.Errorf("orchestrator.reviewer_profiles names %q, which is not a profile of models.toml (%q)", p, raw)
		}
		c.Profiles = append(c.Profiles, p)
	}
	if v := strings.TrimSpace(r.String("orchestrator.review_max_diff_bytes", "")); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < reviewMaxDiffMin || n > reviewMaxDiffMax {
			return ReviewConfig{}, fmt.Errorf("orchestrator.review_max_diff_bytes must be an integer from %d to %d, got %q",
				reviewMaxDiffMin, reviewMaxDiffMax, v)
		}
		c.MaxDiffBytes = n
	}
	return c, nil
}

// Profile is the profile a reviewer runs on: asked, when the list allows it,
// or the first allowed when nothing is asked. The list is Profiles, or
// fallback — the first child profile — when none is declared. A profile off
// the list is model_not_allowed, never quietly replaced by one on it.
func (c ReviewConfig) Profile(asked, fallback string) (string, error) {
	allowed := c.Profiles
	if len(allowed) == 0 && strings.TrimSpace(fallback) != "" {
		allowed = []string{strings.TrimSpace(fallback)}
	}
	if len(allowed) == 0 {
		return "", fmt.Errorf("orchestrate: no profile for the reviewer: orchestrator.reviewer_profiles and the child profiles are both empty")
	}
	asked = strings.TrimSpace(asked)
	if asked == "" {
		return allowed[0], nil
	}
	if !slices.Contains(allowed, asked) {
		return "", &protocol.Error{Code: protocol.CodeModelNotAllowed, Message: fmt.Sprintf(
			"the reviewer runs on %s (orchestrator.reviewer_profiles); %q is not on the list", strings.Join(allowed, ", "), asked)}
	}
	return asked, nil
}

// OwnsViolation is a child's change outside what it owns. It goes back to the
// child without a commit and without a review: no model is asked about a diff
// the child had no right to make.
type OwnsViolation struct {
	Paths []string
}

func (e *OwnsViolation) Error() string {
	return fmt.Sprintf("outside_owns: the child changed %s, outside what it owns; it goes back without a review",
		strings.Join(e.Paths, ", "))
}

// Return is the hand-back the pipeline records.
func (e *OwnsViolation) Return() protocol.Return {
	return protocol.Return{Why: protocol.CodeOutsideOwns, Paths: slices.Clone(e.Paths)}
}

// OutsideOwns is the paths no entry of owns claims. An entry claims a path
// when it is the path, a directory above it, or a policy.Glob that matches
// it. No owns claims nothing.
func OutsideOwns(owns, paths []string) []string {
	var out []string
	for _, p := range paths {
		if !owned(owns, p) {
			out = append(out, p)
		}
	}
	return out
}

func owned(owns []string, p string) bool {
	p = strings.TrimPrefix(p, "./")
	for _, o := range owns {
		o = strings.TrimSuffix(strings.TrimPrefix(strings.TrimSpace(o), "./"), "/")
		if o == "" || o == "." {
			continue // owning the root is owning nothing in particular; delegate refuses it
		}
		if p == o || strings.HasPrefix(p, o+"/") || policy.Glob(o, p) {
			return true
		}
	}
	return false
}

// Uncommitted is every path git status lists in a worktree — changed, added,
// deleted, untracked, and both sides of a rename — relative to it and sorted.
// It is what the harness checks against owns before it commits (.p §7.1):
// what the child wrote through bash is on this list too.
func Uncommitted(ctx context.Context, worktree string) ([]string, error) {
	out, err := reviewGit(ctx, worktree, "status", "--porcelain=v1", "-z", "--untracked-files=all", "--no-renames")
	if err != nil {
		return nil, err
	}
	var paths []string
	for _, entry := range strings.Split(out, "\x00") {
		if len(entry) > 3 {
			paths = append(paths, entry[3:])
		}
	}
	slices.Sort(paths)
	return slices.Compact(paths), nil
}

// ReviewInput is what the pipeline knows when a child's turn ends done.
type ReviewInput struct {
	ChildID, Worktree string
	// Base is the main commit the branch left from; Head the harness's last
	// commit. Both full SHAs.
	Base, Head string
	Task       string
	Owns       []string
	// MaxDiffBytes is ReviewConfig.MaxDiffBytes.
	MaxDiffBytes int
}

// ReviewRequest is a reviewer's whole brief, and what SpawnReviewer is given.
// ParentID and Profile are the caller's to fill.
type ReviewRequest struct {
	ParentID, ChildID, Worktree, Profile string
	Base, Commit, Task                   string
	Owns                                 []string
	// Files is every path Base..Head changes.
	Files []string
	// Specs are the spec paths the task names that the commit has;
	// MissingSpecs the ones it names and the commit does not, said rather
	// than dropped.
	Specs, MissingSpecs []string
	// Diff is Base..Head whole, or empty with DiffCut when its DiffBytes
	// pass the limit: then the reviewer reads the files instead.
	Diff      string
	DiffBytes int
	DiffCut   bool
}

// AssembleReview builds the reviewer's brief from the child's worktree. A
// change outside owns is an *OwnsViolation and no brief: the child gets its
// work back, and no reviewer is spawned.
func AssembleReview(ctx context.Context, in ReviewInput) (ReviewRequest, error) {
	if !exactCommit.MatchString(in.Base) || !exactCommit.MatchString(in.Head) {
		return ReviewRequest{}, &protocol.Error{Code: protocol.CodeInvalidInput, Message: fmt.Sprintf(
			"a review reads the diff between two exact commits, full SHAs; got %q..%q", in.Base, in.Head)}
	}
	if in.MaxDiffBytes <= 0 {
		return ReviewRequest{}, fmt.Errorf("orchestrate: a review needs orchestrator.review_max_diff_bytes; got %d", in.MaxDiffBytes)
	}
	names, err := reviewGit(ctx, in.Worktree, "diff", "--name-only", "-z", "--no-renames", in.Base, in.Head)
	if err != nil {
		return ReviewRequest{}, err
	}
	var files []string
	for _, f := range strings.Split(names, "\x00") {
		if f != "" {
			files = append(files, f)
		}
	}
	if len(files) == 0 {
		return ReviewRequest{}, fmt.Errorf("orchestrate: %s changes nothing from %s; there is nothing to review", short(in.Head), short(in.Base))
	}
	if outside := OutsideOwns(in.Owns, files); len(outside) > 0 {
		return ReviewRequest{}, &OwnsViolation{Paths: outside}
	}
	diff, err := reviewGit(ctx, in.Worktree, "diff", "--no-color", "--no-ext-diff", "--no-textconv", in.Base, in.Head)
	if err != nil {
		return ReviewRequest{}, err
	}
	req := ReviewRequest{ChildID: in.ChildID, Worktree: in.Worktree, Base: in.Base, Commit: in.Head, Task: in.Task,
		Owns: slices.Clone(in.Owns), Files: files, DiffBytes: len(diff)}
	if len(diff) > in.MaxDiffBytes {
		req.DiffCut = true
	} else {
		req.Diff = diff
	}
	req.Specs, req.MissingSpecs, err = namedSpecs(ctx, in.Worktree, in.Head, in.Task)
	if err != nil {
		return ReviewRequest{}, err
	}
	return req, nil
}

// specName is a spec path as a task writes it.
var specName = regexp.MustCompile(`[A-Za-z0-9_./-]*[A-Za-z0-9_-]\.spec\.md`)

// namedSpecs finds, in the commit, each spec the task names: the path as
// written, or the one file of that name when the task names it bare.
func namedSpecs(ctx context.Context, worktree, commit, task string) (found, missing []string, err error) {
	named := specName.FindAllString(task, -1)
	if len(named) == 0 {
		return nil, nil, nil
	}
	tree, err := reviewGit(ctx, worktree, "ls-tree", "-r", "-z", "--name-only", commit)
	if err != nil {
		return nil, nil, err
	}
	files := strings.Split(tree, "\x00")
	for _, n := range named {
		n = strings.TrimPrefix(n, "./")
		var hit []string
		for _, f := range files {
			if f == n || (!strings.Contains(n, "/") && strings.HasSuffix(f, "/"+n)) {
				hit = append(hit, f)
			}
		}
		if len(hit) == 1 && !slices.Contains(found, hit[0]) {
			found = append(found, hit[0])
		} else if len(hit) != 1 && !slices.Contains(missing, n) {
			missing = append(missing, n) // absent, or a bare name more than one file has
		}
	}
	return found, missing, nil
}

// Prompt is the reviewer's first turn, in English: text a model reads is
// behaviour (LANGUAGE.md). Empty for a request AssembleReview did not build.
func (r ReviewRequest) Prompt() string {
	if r.Commit == "" {
		return ""
	}
	var b strings.Builder
	fmt.Fprintf(&b, "Review commit %s of a child's work, against the task it was given and the specs that task names. "+
		"You only read: the tree is the child's worktree at that commit. Your review ends when you call `verdict`, "+
		"and a review that ends without it counts as no review.\n\n", r.Commit)
	fmt.Fprintf(&b, "## Task\n\n%s\n\n## What the child owns\n\n", strings.TrimSpace(r.Task))
	list(&b, r.Owns)
	if len(r.Specs)+len(r.MissingSpecs) > 0 {
		b.WriteString("\n## Specs the task names\n\nRead each one; the diff must keep what they say.\n\n")
		list(&b, r.Specs)
		if len(r.MissingSpecs) > 0 {
			fmt.Fprintf(&b, "\nNamed by the task and not found as one file in the commit: %s.\n", strings.Join(r.MissingSpecs, ", "))
		}
	}
	fmt.Fprintf(&b, "\n## Files changed from %s\n\n", r.Base)
	list(&b, r.Files)
	if r.DiffCut {
		fmt.Fprintf(&b, "\n## Diff cut\n\nThe diff is %d bytes, over orchestrator.review_max_diff_bytes, and none of it "+
			"is here: read each changed file above before you judge.\n", r.DiffBytes)
	} else {
		fmt.Fprintf(&b, "\n## Diff\n\n```diff\n%s\n```\n", strings.TrimSuffix(r.Diff, "\n"))
	}
	return b.String()
}

func list(b *strings.Builder, items []string) {
	if len(items) == 0 {
		b.WriteString("(none)\n")
	}
	for _, it := range items {
		fmt.Fprintf(b, "- %s\n", it)
	}
}

func reviewGit(ctx context.Context, dir string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, reviewGitDeadline)
	defer cancel()
	cmd := exec.CommandContext(ctx, "git", append([]string{"-c", "core.quotepath=off"}, args...)...)
	cmd.Dir = dir
	cmd.Env = gitEnv(cmd.Environ())
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return "", fmt.Errorf("orchestrate: git %s in %s: %s", strings.Join(args, " "), dir, msg)
	}
	return stdout.String(), nil
}
