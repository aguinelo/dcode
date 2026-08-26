package vcs

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// repo builds a real repository. Reading git by shelling out to git means the
// only honest test is against a real one — a fake would test the fake.
func repo(t *testing.T) string {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("no git here")
	}
	dir := t.TempDir()
	for _, args := range [][]string{
		{"init", "-q", "-b", "main"},
		{"config", "user.email", "t@example.com"},
		{"config", "user.name", "T"},
		{"config", "commit.gpgsign", "false"},
	} {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Skipf("git %v: %v\n%s", args, err, out)
		}
	}
	return dir
}

func write(t *testing.T, dir, name, body string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
}

func git(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

// A directory that is not a repository is the ordinary case for a scratch
// directory, and a session has to open there exactly as it always did — but it
// comes back marked rather than empty.
//
// It used to be nil, which put nothing in the prefix. Ordinary is not the same
// as not worth saying: without a repository there is no diff, no review and no
// undo, and every working agreement a project file describes is describing
// machinery that is not there.
func TestSomewhereThatIsNotARepositoryIsMarkedAbsent(t *testing.T) {
	got := Read(context.Background(), t.TempDir())
	if got == nil {
		t.Fatal("a directory that is not a repository came back as no snapshot at all")
	}
	if !got.Absent {
		t.Errorf("got %+v, want Absent", got)
	}
	if got.Branch != "" || got.MainBranch != "" || len(got.Commits) != 0 {
		t.Errorf("an absent repository carried repository facts: %+v", got)
	}
}

// A real repository is never marked absent, which is the other half of the
// same claim and the one a typo would break silently.
func TestARealRepositoryIsNotMarkedAbsent(t *testing.T) {
	dir := repo(t)
	got := Read(context.Background(), dir)
	if got == nil {
		t.Fatal("no snapshot for a real repository")
	}
	if got.Absent {
		t.Errorf("a real repository was reported as absent: %+v", got)
	}
}

// The branch, the tree and the history — the three facts a rule about where
// work belongs depends on.
func TestReadingARepositoryAnswersTheThreeQuestions(t *testing.T) {
	dir := repo(t)
	write(t, dir, "a.txt", "one\n")
	git(t, dir, "add", "a.txt")
	git(t, dir, "commit", "-qm", "feat: the first thing")

	got := Read(context.Background(), dir)
	if got == nil {
		t.Fatal("a real repository read as nothing")
	}
	if got.Branch != "main" {
		t.Errorf("branch = %q, want main", got.Branch)
	}
	if got.Detached {
		t.Error("a checked-out branch was reported as detached")
	}
	if !got.Clean {
		t.Errorf("a committed tree is not clean: %q", got.Status)
	}
	if len(got.Commits) != 1 || !strings.Contains(got.Commits[0], "the first thing") {
		t.Errorf("commits = %v, want the one just made", got.Commits)
	}
}

// A dirty tree says what is dirty. "Something changed" is not actionable; the
// paths are the whole point.
func TestADirtyTreeReportsWhatChanged(t *testing.T) {
	dir := repo(t)
	write(t, dir, "a.txt", "one\n")
	git(t, dir, "add", "a.txt")
	git(t, dir, "commit", "-qm", "first")
	write(t, dir, "a.txt", "two\n")
	write(t, dir, "untracked.txt", "x\n")

	got := Read(context.Background(), dir)
	if got.Clean {
		t.Fatal("a modified tree was reported clean")
	}
	if !strings.Contains(got.Status, "a.txt") || !strings.Contains(got.Status, "untracked.txt") {
		t.Errorf("status does not name what changed:\n%s", got.Status)
	}
}

// A tree with hundreds of changes must not spend the context window on a list
// nobody reads, and nothing in this codebase cuts output without saying so.
func TestAVeryDirtyTreeIsCutAndSaysSo(t *testing.T) {
	dir := repo(t)
	write(t, dir, "a.txt", "one\n")
	git(t, dir, "add", "a.txt")
	git(t, dir, "commit", "-qm", "first")
	for i := 0; i < maxStatusLines+20; i++ {
		write(t, dir, "f"+itoa(i)+".txt", "x\n")
	}

	got := Read(context.Background(), dir)
	if !got.Truncated {
		t.Error("a status past the cap did not report being cut")
	}
	if n := strings.Count(got.Status, "\n") + 1; n > maxStatusLines {
		t.Errorf("status kept %d lines, cap is %d", n, maxStatusLines)
	}
}

// A detached head has no branch to name, and "HEAD" is what git answers when
// there is none. Treating that string as a name is how an agent reports working
// on a branch called HEAD.
func TestADetachedHeadIsReportedAsDetached(t *testing.T) {
	dir := repo(t)
	write(t, dir, "a.txt", "one\n")
	git(t, dir, "add", "a.txt")
	git(t, dir, "commit", "-qm", "first")

	cmd := exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		t.Fatal(err)
	}
	git(t, dir, "checkout", "-q", strings.TrimSpace(string(out)))

	got := Read(context.Background(), dir)
	if !got.Detached {
		t.Errorf("a detached head reported branch %q", got.Branch)
	}
	if got.Branch == "HEAD" {
		t.Error("the literal HEAD was taken for a branch name")
	}
}

// A repository with no commit yet is a repository. Reading it must not fail,
// and the absence of history is not an absence of the repository.
func TestARepositoryWithNoCommitsIsStillARepository(t *testing.T) {
	dir := repo(t)
	got := Read(context.Background(), dir)
	if got == nil {
		t.Fatal("an empty repository read as no repository")
	}
	if len(got.Commits) != 0 {
		t.Errorf("commits = %v, want none", got.Commits)
	}
}

// Work is cut from somewhere, and "am I on the right branch" is answered
// against it.
func TestTheMainBranchIsFoundWithoutARemote(t *testing.T) {
	dir := repo(t)
	write(t, dir, "a.txt", "one\n")
	git(t, dir, "add", "a.txt")
	git(t, dir, "commit", "-qm", "first")

	if got := Read(context.Background(), dir); got.MainBranch != "main" {
		t.Errorf("main branch = %q, want main", got.MainBranch)
	}

	// And a repository whose default is not main is not told it is: naming a
	// branch that does not exist reads as an answer.
	other := repo(t)
	git(t, other, "checkout", "-q", "-b", "trunk")
	write(t, other, "a.txt", "one\n")
	git(t, other, "add", "a.txt")
	git(t, other, "commit", "-qm", "first")
	// No branch called main was ever born here: the first commit landed on
	// trunk, so there is nothing to delete and nothing to find.

	if got := Read(context.Background(), other); got.MainBranch != "" {
		t.Errorf("main branch = %q, want none — neither main nor master exists", got.MainBranch)
	}
}

// The read runs before the first frame, so it is bounded. A cancelled context
// gives back nothing rather than hanging the session start.
func TestACancelledReadDoesNotHang(t *testing.T) {
	dir := repo(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if got := Read(ctx, dir); got != nil {
		t.Errorf("a cancelled read produced %+v", got)
	}
}

// With a remote, the repository is asked rather than guessed: origin/HEAD is
// what the remote says its default is, and a project whose default is not
// "main" is the case guessing gets wrong.
func TestTheRemoteDecidesTheMainBranch(t *testing.T) {
	origin := repo(t)
	git(t, origin, "checkout", "-q", "-b", "trunk")
	write(t, origin, "a.txt", "one\n")
	git(t, origin, "add", "a.txt")
	git(t, origin, "commit", "-qm", "first")

	dir := t.TempDir()
	clone := exec.Command("git", "clone", "-q", origin, dir)
	if out, err := clone.CombinedOutput(); err != nil {
		t.Skipf("clone: %v\n%s", err, out)
	}
	git(t, dir, "config", "user.email", "t@example.com")
	git(t, dir, "config", "user.name", "T")

	got := Read(context.Background(), dir)
	if got == nil {
		t.Fatal("a clone read as no repository")
	}
	if got.MainBranch != "trunk" {
		t.Errorf("main branch = %q, want trunk — the remote's default, not a guess", got.MainBranch)
	}
}

// Which commits the repository still has, in one call.
func TestKnownAnswersForEveryCommitAtOnce(t *testing.T) {
	dir := repo(t)
	write(t, dir, "a.txt", "one\n")
	git(t, dir, "add", "a.txt")
	git(t, dir, "commit", "-qm", "first")

	cmd := exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		t.Fatal(err)
	}
	real := strings.TrimSpace(string(out))

	got := Known(context.Background(), dir, []string{real, "0000000000000000000000000000000000000000"})
	if !got[real] {
		t.Errorf("a commit that is here was reported missing: %v", got)
	}
	if got["0000000000000000000000000000000000000000"] {
		t.Error("a commit that is not here was reported present")
	}
}

// Nothing asked is nothing answered, and the caller must not read that as
// "they are all gone".
func TestKnownWithNothingToAskAnswersNothing(t *testing.T) {
	if got := Known(context.Background(), repo(t), nil); got != nil {
		t.Errorf("got %v, want nil", got)
	}
}

// Somewhere that is not a repository cannot answer, and nil is how it says so —
// distinct from an answer that every commit is missing. Marking a memory stale
// because git was absent would be the heuristic deciding on no evidence.
func TestKnownSomewhereThatCannotAnswerReturnsNil(t *testing.T) {
	if got := Known(context.Background(), t.TempDir(), []string{"abc1234"}); got != nil {
		t.Errorf("got %v, want nil — nothing could be asked", got)
	}
}

// Every commit missing is indistinguishable from a repository that answered
// nothing useful, and the safe reading is that we did not look.
func TestKnownWithEveryCommitMissingAnswersNothing(t *testing.T) {
	dir := repo(t)
	write(t, dir, "a.txt", "one\n")
	git(t, dir, "add", "a.txt")
	git(t, dir, "commit", "-qm", "first")

	got := Known(context.Background(), dir, []string{
		"0000000000000000000000000000000000000000",
		"1111111111111111111111111111111111111111",
	})
	if got != nil {
		t.Errorf("got %v, want nil", got)
	}
}
