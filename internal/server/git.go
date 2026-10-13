package server

import (
	"errors"
	"net/http"
	"os"

	"github.com/aguinelo/dcode/internal/protocol"
	"github.com/aguinelo/dcode/internal/vcs"
)

// existingWorkspace is workspaceOf for a route that runs something there: the
// folder must also exist, or git would answer about wherever it fell back to.
func existingWorkspace(r *http.Request) (string, *protocol.Error) {
	ws, perr := workspaceOf(r)
	if perr != nil {
		return "", perr
	}
	info, err := os.Stat(ws)
	if err != nil {
		return "", protocol.Errorf(protocol.CodeWorkspaceInvalid, "workspace %s cannot be read: %v", ws, err)
	}
	if !info.IsDir() {
		return "", protocol.Errorf(protocol.CodeWorkspaceInvalid, "workspace %s is not a folder", ws)
	}
	return ws, nil
}

// gitErr says why git could not answer. Git missing and git failing are both
// the daemon's to say: neither is "not a repository".
func gitErr(err error) *protocol.Error {
	if errors.Is(err, vcs.ErrNotChanged) {
		return protocol.Errorf(protocol.CodeInvalidInput, "%v", err)
	}
	return protocol.Errorf(protocol.CodeInternal, "%v", err)
}

// wireStatus is how a change reads on the wire. Spelled out rather than passed
// through, so a status vcs learns is a decision here and not a new word on the
// wire nobody declared.
var wireStatus = map[string]string{
	vcs.Modified:   protocol.GitModified,
	vcs.Added:      protocol.GitAdded,
	vcs.Deleted:    protocol.GitDeleted,
	vcs.Untracked:  protocol.GitUntracked,
	vcs.Conflicted: protocol.GitConflicted,
}

func statusOnWire(s string) string {
	if w, ok := wireStatus[s]; ok {
		return w
	}
	return protocol.GitModified
}

// readGit answers where ?workspace= is in git now and what changed in it.
// Read only: nothing here stages, commits or switches.
func (s *Server) readGit(w http.ResponseWriter, r *http.Request) {
	ws, perr := existingWorkspace(r)
	if perr != nil {
		writeErr(w, perr)
		return
	}
	t, err := vcs.ReadTree(r.Context(), ws)
	if err != nil {
		writeErr(w, gitErr(err))
		return
	}
	out := protocol.GitResponse{Files: []protocol.GitFile{}}
	if !t.Absent {
		out.Repository = true
		out.Root, out.Branch, out.Detached, out.Head = t.Root, t.Branch, t.Detached, t.Head
		out.Worktree, out.MainCheckout, out.Truncated = t.Worktree, t.MainCheckout, t.Truncated
		for _, f := range t.Files {
			out.Files = append(out.Files, protocol.GitFile{
				Path: f.Path, Status: statusOnWire(f.Status), Added: f.Added, Removed: f.Removed, Binary: f.Binary,
			})
		}
	}
	writeJSON(w, http.StatusOK, out)
}

// readGitDiff answers one changed file's diff, ?path= as /git lists it.
func (s *Server) readGitDiff(w http.ResponseWriter, r *http.Request) {
	ws, perr := existingWorkspace(r)
	if perr != nil {
		writeErr(w, perr)
		return
	}
	path := r.URL.Query().Get("path")
	if path == "" {
		writeErr(w, protocol.Errorf(protocol.CodeInvalidInput, "a path is required, as GET /git lists it"))
		return
	}
	d, err := vcs.FileDiff(r.Context(), ws, path, protocol.GitDiffLimit)
	if err != nil {
		writeErr(w, gitErr(err))
		return
	}
	writeJSON(w, http.StatusOK, protocol.GitDiffResponse{
		Path: d.Path, Status: statusOnWire(d.Status), Diff: d.Text, Binary: d.Binary, Truncated: d.Truncated,
		Limit: protocol.GitDiffLimit,
	})
}
