package vcs

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

// What a client sees of a working tree, asked for when it is looked at rather
// than once at session start. Read lets the prompt carry a snapshot; this is
// the tree as it is now, file by file, for a person to look at.

// ErrNoGit is the answer when git is not there to ask. Never "not a
// repository": that would be a claim about a folder nobody looked at.
var ErrNoGit = errors.New("git is not installed: it is not on the daemon's PATH")

// ErrNotChanged is a diff asked for a path git does not list as changed. Only
// those have a diff here, so a diff is never a way to read any file by name.
var ErrNotChanged = errors.New("not among the files git lists as changed")

// How a file differs from the last commit.
const (
	Modified   = "modified"
	Added      = "added"
	Deleted    = "deleted"
	Untracked  = "untracked"
	Conflicted = "conflicted"
)

// treeDeadline bounds one read. Longer than the snapshot's: this one is asked
// by a person looking at a panel, and a large repository's status can take
// a few seconds where a session start cannot.
const treeDeadline = 5 * time.Second

// maxTreeFiles is how many changed files a tree carries. A checkout with
// thousands of generated files is said as cut rather than shipped whole.
const maxTreeFiles = 1000

// emptyTree is git's id for the tree with nothing in it: what a repository
// with no commit yet is diffed against.
const emptyTree = "4b825dc642cb6eb9a060e54bf8d69288fbee4904"

// TreeFile is one changed file. Path is relative to the top of the work tree.
type TreeFile struct {
	Path    string
	Status  string
	Added   int
	Removed int
	Binary  bool
}

// Tree is a working tree as git answers it now.
type Tree struct {
	// Absent: the folder is in no repository, and nothing else is set.
	Absent       bool
	Root         string
	Branch       string
	Detached     bool
	Head         string
	Worktree     bool
	MainCheckout string
	Files        []TreeFile
	Truncated    bool
}

// FileChange is one file's unified diff.
type FileChange struct {
	Path      string
	Status    string
	Text      string
	Binary    bool
	Truncated bool
}

// ReadTree reads where dir is in git and what changed there since the last
// commit, staged or not.
func ReadTree(ctx context.Context, dir string) (*Tree, error) {
	if _, err := exec.LookPath("git"); err != nil {
		return nil, ErrNoGit
	}
	ctx, cancel := context.WithTimeout(ctx, treeDeadline)
	defer cancel()

	top, err := gitOut(ctx, dir, "rev-parse", "--show-toplevel")
	if err != nil {
		var ge *gitError
		if errors.As(err, &ge) && strings.Contains(ge.stderr, "not a git repository") {
			return &Tree{Absent: true}, nil
		}
		return nil, err
	}
	t := &Tree{Root: strings.TrimSpace(top)}

	if head, err := gitOut(ctx, t.Root, "rev-parse", "--verify", "--quiet", "--short", "HEAD"); err == nil {
		t.Head = strings.TrimSpace(head)
	}
	// symbolic-ref answers a branch even before the first commit, and fails
	// only when HEAD points at a commit rather than a branch.
	if branch, err := gitOut(ctx, t.Root, "symbolic-ref", "--quiet", "--short", "HEAD"); err == nil {
		t.Branch = strings.TrimSpace(branch)
	} else if ctx.Err() == nil {
		t.Detached = true
	}
	if err := t.readWorktree(ctx); err != nil {
		return nil, err
	}
	if err := t.readFiles(ctx); err != nil {
		return nil, err
	}
	if ctx.Err() != nil {
		return nil, fmt.Errorf("git did not answer within %s", treeDeadline)
	}
	return t, nil
}

// readWorktree tells a linked worktree from the main checkout: in a linked
// one, the git directory is not the common one.
func (t *Tree) readWorktree(ctx context.Context) error {
	gitDir, err := gitOut(ctx, t.Root, "rev-parse", "--absolute-git-dir")
	if err != nil {
		return err
	}
	common, err := gitOut(ctx, t.Root, "rev-parse", "--git-common-dir")
	if err != nil {
		return err
	}
	gitDir, common = strings.TrimSpace(gitDir), strings.TrimSpace(common)
	if !filepath.IsAbs(common) {
		common = filepath.Join(t.Root, common)
	}
	if resolved(gitDir) == resolved(common) {
		return nil
	}
	t.Worktree = true
	common = filepath.Clean(common)
	if filepath.Base(common) == ".git" {
		t.MainCheckout = filepath.Dir(common)
	} else {
		t.MainCheckout = common // a bare repository has no working tree of its own
	}
	return nil
}

func resolved(p string) string {
	if r, err := filepath.EvalSymlinks(p); err == nil {
		return r
	}
	return filepath.Clean(p)
}

// base is what the tree is compared against: the last commit, or nothing.
func (t *Tree) base() string {
	if t.Head == "" {
		return emptyTree
	}
	return "HEAD"
}

func (t *Tree) readFiles(ctx context.Context) error {
	status, err := gitOut(ctx, t.Root, "status", "--porcelain=v1", "-z", "--untracked-files=all", "--no-renames")
	if err != nil {
		return err
	}
	byPath := map[string]*TreeFile{}
	for _, rec := range strings.Split(status, "\x00") {
		if len(rec) < 4 {
			continue
		}
		f := TreeFile{Path: rec[3:], Status: statusOf(rec[0], rec[1])}
		t.Files = append(t.Files, f)
	}
	sort.Slice(t.Files, func(i, j int) bool { return t.Files[i].Path < t.Files[j].Path })
	if len(t.Files) > maxTreeFiles {
		t.Files, t.Truncated = t.Files[:maxTreeFiles], true
	}
	for i := range t.Files {
		byPath[t.Files[i].Path] = &t.Files[i]
	}

	numstat, err := gitOut(ctx, t.Root, "diff", "--numstat", "-z", "--no-renames", "--no-ext-diff", "--no-textconv", t.base(), "--")
	if err != nil {
		return err
	}
	for _, rec := range strings.Split(numstat, "\x00") {
		parts := strings.SplitN(rec, "\t", 3)
		if len(parts) != 3 {
			continue
		}
		f := byPath[parts[2]]
		if f == nil {
			continue
		}
		if parts[0] == "-" {
			f.Binary = true
			continue
		}
		f.Added, _ = strconv.Atoi(parts[0])
		f.Removed, _ = strconv.Atoi(parts[1])
	}
	for i := range t.Files {
		if f := &t.Files[i]; f.Status == Untracked {
			f.Added, f.Binary = countLines(filepath.Join(t.Root, filepath.FromSlash(f.Path)))
		}
	}
	return nil
}

// statusOf reads porcelain's two columns, the index's and the tree's.
func statusOf(x, y byte) string {
	switch {
	case x == '?':
		return Untracked
	case x == 'U' || y == 'U' || (x == 'A' && y == 'A') || (x == 'D' && y == 'D'):
		return Conflicted
	case x == 'D' || y == 'D':
		return Deleted
	case x == 'A':
		return Added
	default:
		return Modified
	}
}

// countLines counts a new file's lines as git would add them. A file that is
// not a regular one — a link, a socket — is counted as none, and a file with a
// NUL in its head as binary, the way git decides.
func countLines(path string) (lines int, binary bool) {
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() || info.Size() == 0 {
		return 0, false
	}
	f, err := os.Open(path)
	if err != nil {
		return 0, false
	}
	defer f.Close()
	r := bufio.NewReader(f)
	head, _ := r.Peek(8000)
	if bytes.IndexByte(head, 0) >= 0 {
		return 0, true
	}
	var last byte
	buf := make([]byte, 32<<10)
	for {
		n, err := r.Read(buf)
		if n > 0 {
			lines += bytes.Count(buf[:n], []byte{'\n'})
			last = buf[n-1]
		}
		if err != nil {
			break
		}
	}
	if last != '\n' {
		lines++
	}
	return lines, false
}

// FileDiff gives one changed file's unified diff, at most max bytes of it, cut
// at a line. The path is relative to the top of the work tree, as ReadTree
// lists it, and only a path it lists is answered.
func FileDiff(ctx context.Context, dir, path string, max int) (*FileChange, error) {
	t, err := ReadTree(ctx, dir)
	if err != nil {
		return nil, err
	}
	if t.Absent {
		return nil, fmt.Errorf("the folder is in no git repository, so %q is %w", path, ErrNotChanged)
	}
	var file *TreeFile
	for i := range t.Files {
		if t.Files[i].Path == path {
			file = &t.Files[i]
			break
		}
	}
	if file == nil {
		return nil, fmt.Errorf("%q is %w", path, ErrNotChanged)
	}
	ctx, cancel := context.WithTimeout(ctx, treeDeadline)
	defer cancel()

	args := []string{"-c", "core.quotePath=false", "diff", "--no-color", "--no-ext-diff", "--no-textconv", "--no-renames"}
	if file.Status == Untracked {
		args = append(args, "--no-index", "--", os.DevNull, file.Path)
	} else {
		args = append([]string{"--literal-pathspecs"}, append(args, t.base(), "--", file.Path)...)
	}
	text, truncated, err := gitCapped(ctx, t.Root, max, args...)
	if err != nil {
		return nil, err
	}
	return &FileChange{Path: file.Path, Status: file.Status, Text: text, Binary: file.Binary, Truncated: truncated}, nil
}

// gitCapped runs git and keeps at most max bytes of what it writes, cut after
// the last whole line. `diff --no-index` exits 1 when the files differ, which
// is the answer and not a failure.
func gitCapped(ctx context.Context, dir string, max int, args ...string) (string, bool, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	cmd := gitCmd(ctx, dir, args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.StdoutPipe()
	if err != nil {
		return "", false, err
	}
	if err := cmd.Start(); err != nil {
		return "", false, err
	}
	got, readErr := io.ReadAll(io.LimitReader(out, int64(max)+1))
	truncated := len(got) > max
	if truncated {
		got = got[:max]
		if i := bytes.LastIndexByte(got, '\n'); i >= 0 {
			got = got[:i+1]
		}
		cancel() // the rest is not wanted; stop git rather than drain it
	}
	waitErr := cmd.Wait()
	if truncated {
		return string(got), true, nil
	}
	if readErr != nil {
		return "", false, readErr
	}
	var exit *exec.ExitError
	if waitErr != nil && !(errors.As(waitErr, &exit) && exit.ExitCode() == 1) {
		return "", false, &gitError{args: args, err: waitErr, stderr: strings.TrimSpace(stderr.String())}
	}
	return string(got), false, nil
}

// gitError is git failing, with what it said about why.
type gitError struct {
	args   []string
	err    error
	stderr string
}

func (e *gitError) Error() string {
	if e.stderr != "" {
		return fmt.Sprintf("git %s: %s", strings.Join(e.args, " "), e.stderr)
	}
	return fmt.Sprintf("git %s: %v", strings.Join(e.args, " "), e.err)
}

func (e *gitError) Unwrap() error { return e.err }

func gitCmd(ctx context.Context, dir string, args ...string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir
	cmd.Env = append(cmd.Environ(), "GIT_OPTIONAL_LOCKS=0", "LC_ALL=C", "GIT_PAGER=cat")
	return cmd
}

// gitOut runs git and returns what it wrote, untrimmed: -z output ends in NUL.
func gitOut(ctx context.Context, dir string, args ...string) (string, error) {
	cmd := gitCmd(ctx, dir, args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		if ctx.Err() != nil {
			return "", fmt.Errorf("git did not answer within %s", treeDeadline)
		}
		return "", &gitError{args: args, err: err, stderr: strings.TrimSpace(stderr.String())}
	}
	return string(out), nil
}
