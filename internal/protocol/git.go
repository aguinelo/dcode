package protocol

// How a file differs from the last commit, as GitFile.Status says it.
const (
	GitModified   = "modified"
	GitAdded      = "added"
	GitDeleted    = "deleted"
	GitUntracked  = "untracked"
	GitConflicted = "conflicted"
)

// GitDiffLimit is how many bytes of one file's diff GET /git/diff hands over.
// Past it the diff is cut at a line and says so: a generated file rewritten
// whole would otherwise travel as megabytes nobody reads in a side panel.
const GitDiffLimit = 256 << 10

// GitFile is one file git lists as changed, against the last commit.
type GitFile struct {
	// Path is relative to the top of the work tree, with forward slashes.
	Path string `json:"path"`
	// Status is one of the Git* constants.
	Status  string `json:"status"`
	Added   int    `json:"added"`
	Removed int    `json:"removed"`
	// Binary says git counts no lines for it; Added and Removed are then zero
	// because there are no lines, not because nothing changed.
	Binary bool `json:"binary"`
}

// GitResponse answers GET /git: where a workspace is in git now, and what
// changed in it.
//
// Read when asked, never kept: the branch is the one checked out now, which
// may not be the one a session started on (Session.Branch is that). The files
// are the folder's, every change since the last commit, staged or not — not
// only what one conversation wrote.
type GitResponse struct {
	// Repository is false for a folder outside any repository: the ordinary
	// case for a scratch folder, said rather than answered as a clean tree.
	Repository bool `json:"repository"`
	// Root is the top of the work tree the workspace is in.
	Root string `json:"root,omitempty"`
	// Branch is empty when Detached.
	Branch   string `json:"branch,omitempty"`
	Detached bool   `json:"detached"`
	// Head is the abbreviated commit checked out; empty before the first one.
	Head string `json:"head,omitempty"`
	// Worktree says Root is a linked worktree (`git worktree add`), and
	// MainCheckout is then the repository's own working tree — or its git
	// directory, for a bare repository.
	Worktree     bool   `json:"worktree"`
	MainCheckout string `json:"main_checkout,omitempty"`
	// Files is in path order; none is an empty list, never null.
	Files []GitFile `json:"files"`
	// Truncated says there were more changed files than the list carries.
	Truncated bool `json:"truncated"`
}

// GitDiffResponse answers GET /git/diff: one changed file's unified diff
// against the last commit, or against nothing for a file git does not track.
type GitDiffResponse struct {
	Path   string `json:"path"`
	Status string `json:"status"`
	// Diff is as git writes it, without colour, cut at a line past Limit.
	Diff string `json:"diff"`
	// Binary says git has no lines to show for it, and Diff only says so.
	Binary    bool `json:"binary"`
	Truncated bool `json:"truncated"`
	// Limit is the cap the diff was cut at, in bytes (GitDiffLimit).
	Limit int `json:"limit"`
}
