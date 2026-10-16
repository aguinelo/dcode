package server

import (
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"path/filepath"

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
	return existingFolder(ws)
}

// existingFolder is existingWorkspace for a workspace named in a body.
func existingFolder(ws string) (string, *protocol.Error) {
	if ws == "" {
		return "", protocol.Errorf(protocol.CodeWorkspaceInvalid, "a workspace is required")
	}
	if !filepath.IsAbs(ws) {
		return "", protocol.Errorf(protocol.CodeWorkspaceInvalid, "workspace must be an absolute path, got %q", ws)
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

// prErr says whose a PR route's failure is: gh's absence, the branch's, the
// request's, or the daemon's.
func prErr(err error) *protocol.Error {
	var refusal *vcs.Refusal
	switch {
	case errors.Is(err, vcs.ErrNoGh), errors.Is(err, vcs.ErrGhLoggedOut):
		return protocol.Errorf(protocol.CodeGhUnavailable, "%v", err)
	case errors.Is(err, vcs.ErrInvalidPR):
		return protocol.Errorf(protocol.CodeInvalidInput, "%v", err)
	case errors.As(err, &refusal):
		return protocol.Errorf(protocol.CodePRRefused, "%v", err)
	}
	return protocol.Errorf(protocol.CodeInternal, "%v", err)
}

// wirePRState and wireChecks are how a PR and its checks read on the wire,
// spelled out for the reason wireStatus is.
var wirePRState = map[string]string{
	vcs.PROpen: protocol.GitPROpen, vcs.PRClosed: protocol.GitPRClosed, vcs.PRMerged: protocol.GitPRMerged,
}

var wireChecks = map[string]string{
	vcs.ChecksPass: protocol.GitChecksPass, vcs.ChecksFail: protocol.GitChecksFail,
	vcs.ChecksPending: protocol.GitChecksPending, vcs.ChecksNone: protocol.GitChecksNone,
}

func prOnWire(b *vcs.BranchPR) protocol.GitPRResponse {
	out := protocol.GitPRResponse{Branch: b.Branch, Base: b.Base}
	if p := b.PR; p != nil {
		c := p.Checks
		checks, ok := wireChecks[c.State]
		if !ok {
			checks = protocol.GitChecksFail // a summary the wire has no word for is never read as green
		}
		out.PR = &protocol.GitPR{Number: p.Number, URL: p.URL, Title: p.Title, State: wirePRState[p.State], Draft: p.Draft,
			Base: p.Base, Head: p.Head, Checks: protocol.GitChecks{State: checks, Passed: c.Passed,
				Failed: c.Failed, Pending: c.Pending, Skipped: c.Skipped}}
	}
	return out
}

// readGitPR answers the PR of the branch ?workspace= has checked out now.
// Read only: it asks GitHub through gh and changes nothing.
func (s *Server) readGitPR(w http.ResponseWriter, r *http.Request) {
	ws, perr := existingWorkspace(r)
	if perr != nil {
		writeErr(w, perr)
		return
	}
	b, err := vcs.ReadPR(r.Context(), ws)
	if err != nil {
		writeErr(w, prErr(err))
		return
	}
	writeJSON(w, http.StatusOK, prOnWire(b))
}

// createGitPR pushes the workspace's branch and opens the PR a person
// confirmed. It never merges, never pushes by force, never turns auto-merge on.
func (s *Server) createGitPR(w http.ResponseWriter, r *http.Request) {
	var req protocol.CreateGitPRRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, protocol.Errorf(protocol.CodeInvalidInput, "malformed request: %v", err))
		return
	}
	ws, perr := existingFolder(req.Workspace)
	if perr != nil {
		writeErr(w, perr)
		return
	}
	b, err := vcs.CreatePR(r.Context(), ws, vcs.NewPR{Title: req.Title, Body: req.Body, Base: req.Base, Draft: req.Draft})
	if err != nil {
		writeErr(w, prErr(err))
		return
	}
	writeJSON(w, http.StatusCreated, prOnWire(b))
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
