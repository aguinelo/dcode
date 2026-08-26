// Package vcs reads the state of the repository the agent is working in.
//
// Reading only. Nothing here commits, stages, branches or pushes: git is the
// user's, and an agent that moves it without being asked is an agent nobody
// leaves alone with a repository. What this produces is a fact for the prompt,
// and `bash` is how git is actually run.
package vcs

import (
	"bufio"
	"context"
	"os/exec"
	"strings"
	"time"

	"github.com/aguinelo/dcode/internal/behavior"
)

// deadline bounds the whole read. It runs before the first frame, and a slow
// probe is felt as a slow start — a repository on a cold network filesystem
// must not cost the session its startup.
const deadline = 2 * time.Second

// maxStatusLines is how much of a dirty tree reaches the prompt.
//
// A repository mid-merge with four hundred modified files would otherwise spend
// the context window on a list nobody reads, and the fact that matters —
// "this tree is dirty, here is the shape of it" — survives the cut.
const maxStatusLines = 40

// commitCount is how far back the log goes. Enough to see what the recent work
// was about; not a history lesson.
const commitCount = 5

// Read takes the snapshot.
//
// A directory that is not a repository comes back marked Absent rather than
// nil. It used to be nil, and nil put nothing in the prefix at all — the
// ordinary case, handled by saying nothing about it.
//
// Ordinary it is. Worth saying it is too: without a repository there is no
// diff, no review and no undo, and every working agreement a project file
// describes — a commit per task, a pull request per spec, a floor for merging
// — is describing machinery that is not there. An agent spent a day in exactly
// that state and nothing told it.
//
// Nil is kept for the case it always should have meant: the snapshot was not
// taken. An error is still reserved for git being present and failing, which
// is worth neither stopping the session nor hiding.
func Read(ctx context.Context, dir string) *behavior.Repo {
	ctx, cancel := context.WithTimeout(ctx, deadline)
	defer cancel()

	// Without git installed there is no answer, and "there is no repository"
	// would be a claim about something never looked at. That is the exact
	// defect this change exists to remove, and producing it here while
	// removing it there would be the funniest possible way to ship it.
	if _, err := exec.LookPath("git"); err != nil {
		return nil
	}

	if _, err := run(ctx, dir, "rev-parse", "--is-inside-work-tree"); err != nil {
		// A cancelled or timed-out probe did not find out either. git answered
		// nothing, and "there is no repository" would again be a claim about
		// something never looked at — the third place in this one function
		// where the two have to be kept apart.
		if ctx.Err() != nil {
			return nil
		}
		return &behavior.Repo{Absent: true}
	}

	r := &behavior.Repo{}

	// --abbrev-ref gives the name, or the literal "HEAD" when there is none to
	// give. Treating that string as a branch name is how an agent reports
	// working on a branch called HEAD.
	if head, err := run(ctx, dir, "rev-parse", "--abbrev-ref", "HEAD"); err == nil {
		if head == "HEAD" {
			r.Detached = true
		} else {
			r.Branch = head
		}
	}

	r.MainBranch = mainBranch(ctx, dir)

	if out, err := run(ctx, dir, "status", "--porcelain"); err == nil {
		if strings.TrimSpace(out) == "" {
			r.Clean = true
		} else {
			r.Status, r.Truncated = clampLines(out, maxStatusLines)
		}
	}

	if out, err := run(ctx, dir, "log", "--oneline", "-n", itoa(commitCount)); err == nil {
		for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
			if line != "" {
				r.Commits = append(r.Commits, line)
			}
		}
	}
	return r
}

// mainBranch is what work is normally cut from.
//
// Asked of the repository rather than guessed: origin/HEAD is what the remote
// says its default is. Only when there is no remote to ask does this fall back
// to looking for the conventional names, and it checks that they exist rather
// than naming one that does not — a wrong main branch is worse than none,
// because it reads as an answer.
func mainBranch(ctx context.Context, dir string) string {
	if out, err := run(ctx, dir, "rev-parse", "--abbrev-ref", "origin/HEAD"); err == nil {
		if _, name, ok := strings.Cut(out, "/"); ok && name != "" {
			return name
		}
	}
	for _, name := range []string{"main", "master"} {
		if _, err := run(ctx, dir, "rev-parse", "--verify", "--quiet", "refs/heads/"+name); err == nil {
			return name
		}
	}
	return ""
}

// clampLines keeps the first n lines and reports whether it cut.
func clampLines(s string, n int) (string, bool) {
	sc := bufio.NewScanner(strings.NewReader(s))
	var kept []string
	cut := false
	for sc.Scan() {
		if len(kept) == n {
			cut = true
			break
		}
		kept = append(kept, sc.Text())
	}
	return strings.Join(kept, "\n"), cut
}

func run(ctx context.Context, dir string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir
	// The environment git reads can change what it answers — a pager, a
	// different config, a locale that translates the porcelain. None of that
	// belongs in a snapshot that has to mean the same thing everywhere.
	cmd.Env = append(cmd.Environ(), "GIT_OPTIONAL_LOCKS=0", "LC_ALL=C", "GIT_PAGER=cat")
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}

// Known reports which of these commits the repository still has.
//
// One process for the whole list: `git cat-file --batch-check` takes them on
// stdin and answers each in order, so checking forty memories costs one call
// rather than forty. Forty processes at session start would be felt as a slow
// start, and the answer is not worth that.
//
// Nil when nothing could be asked — no git, no repository, a failed call. The
// caller must read that as "we did not look" rather than "we looked and they are
// gone": marking a memory stale because git was missing would be the heuristic
// deciding on no evidence at all.
func Known(ctx context.Context, dir string, shas []string) map[string]bool {
	if len(shas) == 0 {
		return nil
	}
	ctx, cancel := context.WithTimeout(ctx, deadline)
	defer cancel()

	cmd := exec.CommandContext(ctx, "git", "cat-file", "--batch-check")
	cmd.Dir = dir
	cmd.Env = append(cmd.Environ(), "GIT_OPTIONAL_LOCKS=0", "LC_ALL=C", "GIT_PAGER=cat")
	cmd.Stdin = strings.NewReader(strings.Join(shas, "\n") + "\n")
	out, err := cmd.Output()
	if err != nil {
		return nil
	}

	// A line is either "<sha> commit <size>" or "<name> missing".
	known := map[string]bool{}
	for i, line := range strings.Split(strings.TrimRight(string(out), "\n"), "\n") {
		if i >= len(shas) {
			break
		}
		if fields := strings.Fields(line); len(fields) >= 2 && fields[1] != "missing" {
			known[shas[i]] = true
		}
	}
	if len(known) == 0 {
		// Every one missing is indistinguishable from a repository that answered
		// nothing useful, and the safe reading of that is that we did not look.
		return nil
	}
	return known
}
