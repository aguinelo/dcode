package vcs

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func commitAll(t *testing.T, dir, msg string) {
	t.Helper()
	git(t, dir, "add", "-A")
	git(t, dir, "commit", "-q", "-m", msg)
}

func fileNamed(tr *Tree, path string) *TreeFile {
	for i := range tr.Files {
		if tr.Files[i].Path == path {
			return &tr.Files[i]
		}
	}
	return nil
}

// The tree as it is now, against the last commit: each file git lists as
// changed, how, and by how many lines — staged or not, since what a person
// wants to see is what the folder holds that the commit does not.
func TestTheTreeListsEachChangedFileWithItsLines(t *testing.T) {
	dir := repo(t)
	write(t, dir, "kept.txt", "one\ntwo\nthree\n")
	write(t, dir, "gone.txt", "bye\n")
	commitAll(t, dir, "first")

	write(t, dir, "kept.txt", "one\nTWO\nthree\nfour\n")
	if err := os.Remove(filepath.Join(dir, "gone.txt")); err != nil {
		t.Fatal(err)
	}
	write(t, dir, "staged.txt", "a\nb\n")
	git(t, dir, "add", "staged.txt")
	if err := os.MkdirAll(filepath.Join(dir, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	write(t, dir, "sub/new.txt", "x\ny\nz\n")

	tr, err := ReadTree(context.Background(), dir)
	if err != nil {
		t.Fatal(err)
	}
	if tr.Absent {
		t.Fatal("a repository read as absent")
	}
	want := map[string]TreeFile{
		"kept.txt":    {Path: "kept.txt", Status: "modified", Added: 2, Removed: 1},
		"gone.txt":    {Path: "gone.txt", Status: "deleted", Removed: 1},
		"staged.txt":  {Path: "staged.txt", Status: "added", Added: 2},
		"sub/new.txt": {Path: "sub/new.txt", Status: "untracked", Added: 3},
	}
	if len(tr.Files) != len(want) {
		t.Fatalf("got %d files, want %d: %+v", len(tr.Files), len(want), tr.Files)
	}
	for path, w := range want {
		got := fileNamed(tr, path)
		if got == nil {
			t.Errorf("%s is not listed: %+v", path, tr.Files)
			continue
		}
		if *got != w {
			t.Errorf("%s: got %+v, want %+v", path, *got, w)
		}
	}
	for i := 1; i < len(tr.Files); i++ {
		if tr.Files[i-1].Path > tr.Files[i].Path {
			t.Errorf("files are not in path order: %+v", tr.Files)
		}
	}
}

// Where the folder is: the top of its work tree, the branch now — not the one
// the session started on — and the commit it sits on.
func TestTheTreeSaysWhereItIs(t *testing.T) {
	dir := repo(t)
	write(t, dir, "a.txt", "a\n")
	commitAll(t, dir, "first")
	git(t, dir, "switch", "-q", "-c", "feature")
	if err := os.MkdirAll(filepath.Join(dir, "inner"), 0o755); err != nil {
		t.Fatal(err)
	}

	tr, err := ReadTree(context.Background(), filepath.Join(dir, "inner"))
	if err != nil {
		t.Fatal(err)
	}
	if !samePath(t, tr.Root, dir) {
		t.Errorf("root %q, want %q", tr.Root, dir)
	}
	if tr.Branch != "feature" || tr.Detached {
		t.Errorf("branch %q detached=%v, want feature", tr.Branch, tr.Detached)
	}
	if len(tr.Head) < 7 {
		t.Errorf("head %q is not a commit", tr.Head)
	}
	if tr.Worktree || tr.MainCheckout != "" {
		t.Errorf("the main checkout read as a linked worktree: %+v", tr)
	}
	if len(tr.Files) != 0 {
		t.Errorf("a clean tree listed files: %+v", tr.Files)
	}
}

// A linked worktree says so, and names the checkout it belongs to: two folders
// of one repository look the same from inside, and which one a session writes
// in is the thing a person most needs to tell apart.
func TestALinkedWorktreeNamesItsMainCheckout(t *testing.T) {
	dir := repo(t)
	write(t, dir, "a.txt", "a\n")
	commitAll(t, dir, "first")
	linked := filepath.Join(t.TempDir(), "linked")
	git(t, dir, "worktree", "add", "-q", "-b", "side", linked)

	tr, err := ReadTree(context.Background(), linked)
	if err != nil {
		t.Fatal(err)
	}
	if !tr.Worktree {
		t.Fatalf("a linked worktree did not say so: %+v", tr)
	}
	if !samePath(t, tr.MainCheckout, dir) {
		t.Errorf("main checkout %q, want %q", tr.MainCheckout, dir)
	}
	if !samePath(t, tr.Root, linked) || tr.Branch != "side" {
		t.Errorf("root %q branch %q, want %q side", tr.Root, tr.Branch, linked)
	}
}

// A detached head is said, never reported as a branch called HEAD.
func TestADetachedHeadIsNotABranch(t *testing.T) {
	dir := repo(t)
	write(t, dir, "a.txt", "a\n")
	commitAll(t, dir, "first")
	git(t, dir, "switch", "-q", "--detach")

	tr, err := ReadTree(context.Background(), dir)
	if err != nil {
		t.Fatal(err)
	}
	if !tr.Detached || tr.Branch != "" {
		t.Errorf("got branch %q detached=%v, want detached and no branch", tr.Branch, tr.Detached)
	}
}

// A repository with no commit yet has nothing to diff against but the empty
// tree, and every file in it is new — not an error, and not "no changes".
func TestARepositoryWithNoCommitListsWhatItHolds(t *testing.T) {
	dir := repo(t)
	write(t, dir, "a.txt", "a\nb\n")

	tr, err := ReadTree(context.Background(), dir)
	if err != nil {
		t.Fatal(err)
	}
	if tr.Branch != "main" || tr.Head != "" {
		t.Errorf("branch %q head %q, want main and no head", tr.Branch, tr.Head)
	}
	if f := fileNamed(tr, "a.txt"); f == nil || f.Status != "untracked" || f.Added != 2 {
		t.Errorf("a.txt came back as %+v", tr.Files)
	}
}

// A folder outside any repository is said as such, the ordinary case, and not
// as a tree with nothing changed in it.
func TestAFolderOutsideARepositoryIsAbsent(t *testing.T) {
	tr, err := ReadTree(context.Background(), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if !tr.Absent {
		t.Errorf("a plain folder read as a repository: %+v", tr)
	}
}

// Without git there is no answer at all, and saying "not a repository" would
// be a claim about something nobody looked at.
func TestWithoutGitTheTreeIsAnErrorNotAnAbsence(t *testing.T) {
	dir := repo(t)
	t.Setenv("PATH", t.TempDir())
	_, err := ReadTree(context.Background(), dir)
	if !errors.Is(err, ErrNoGit) {
		t.Fatalf("got %v, want ErrNoGit", err)
	}
	if _, err := FileDiff(context.Background(), dir, "a.txt", 1024); !errors.Is(err, ErrNoGit) {
		t.Fatalf("diff: got %v, want ErrNoGit", err)
	}
}

// A binary file is listed without line counts, and marked, rather than
// counted as zero lines changed.
func TestABinaryFileIsMarkedAndNotCounted(t *testing.T) {
	dir := repo(t)
	write(t, dir, "a.txt", "a\n")
	commitAll(t, dir, "first")
	write(t, dir, "blob.bin", "\x00\x01\x02")

	tr, err := ReadTree(context.Background(), dir)
	if err != nil {
		t.Fatal(err)
	}
	if f := fileNamed(tr, "blob.bin"); f == nil || !f.Binary || f.Added != 0 {
		t.Errorf("blob.bin came back as %+v", tr.Files)
	}
}

// One file's diff, unified, as git writes it — for a tracked file against the
// last commit and for a new one against nothing.
func TestTheDiffOfOneFileIsUnified(t *testing.T) {
	dir := repo(t)
	write(t, dir, "a.txt", "one\ntwo\n")
	commitAll(t, dir, "first")
	write(t, dir, "a.txt", "one\nTWO\n")
	write(t, dir, "new.txt", "fresh\n")

	d, err := FileDiff(context.Background(), dir, "a.txt", 1<<16)
	if err != nil {
		t.Fatal(err)
	}
	if d.Status != "modified" || !strings.Contains(d.Text, "-two\n") || !strings.Contains(d.Text, "+TWO\n") || d.Truncated {
		t.Errorf("a.txt diff: %+v", d)
	}
	d, err = FileDiff(context.Background(), dir, "new.txt", 1<<16)
	if err != nil {
		t.Fatal(err)
	}
	if d.Status != "untracked" || !strings.Contains(d.Text, "+fresh\n") {
		t.Errorf("new.txt diff: %+v", d)
	}
}

// A diff past the cap is cut and says it was, never handed over as if whole.
func TestALongDiffIsCutAndSaysSo(t *testing.T) {
	dir := repo(t)
	write(t, dir, "a.txt", "a\n")
	commitAll(t, dir, "first")
	write(t, dir, "a.txt", strings.Repeat("a long line of text\n", 500))

	d, err := FileDiff(context.Background(), dir, "a.txt", 1024)
	if err != nil {
		t.Fatal(err)
	}
	if !d.Truncated || len(d.Text) > 1024 {
		t.Errorf("got %d bytes truncated=%v, want at most 1024 and truncated", len(d.Text), d.Truncated)
	}
	if !strings.HasSuffix(d.Text, "\n") {
		t.Errorf("the cut fell mid-line: %q", d.Text[len(d.Text)-20:])
	}
}

// Only a file git lists as changed has a diff to give: a path outside that
// list — a clean file, one that does not exist, one outside the repository —
// is refused, so the route never becomes a way to read any file by name.
func TestOnlyAChangedFileHasADiff(t *testing.T) {
	dir := repo(t)
	write(t, dir, "a.txt", "a\n")
	commitAll(t, dir, "first")
	write(t, dir, "b.txt", "b\n")

	for _, path := range []string{"a.txt", "missing.txt", "../outside.txt", "/etc/hosts", ":(glob)*"} {
		if _, err := FileDiff(context.Background(), dir, path, 1024); !errors.Is(err, ErrNotChanged) {
			t.Errorf("%q: got %v, want ErrNotChanged", path, err)
		}
	}
}

func samePath(t *testing.T, a, b string) bool {
	t.Helper()
	ra, err := filepath.EvalSymlinks(a)
	if err != nil {
		return false
	}
	rb, err := filepath.EvalSymlinks(b)
	if err != nil {
		return false
	}
	return ra == rb
}
