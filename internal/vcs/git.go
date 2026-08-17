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

// Read takes the snapshot, or returns nil when there is nothing to snapshot.
//
// Nil rather than an error for a directory that is not a repository: that is
// the ordinary case for a scratch directory, and a session must open there
// exactly as it always did. An error is reserved for git being present and
// failing, which is worth neither stopping the session nor hiding.
func Read(ctx context.Context, dir string) *behavior.Repo {
	ctx, cancel := context.WithTimeout(ctx, deadline)
	defer cancel()

	if _, err := run(ctx, dir, "rev-parse", "--is-inside-work-tree"); err != nil {
		return nil
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
