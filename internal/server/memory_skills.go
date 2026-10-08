package server

import (
	"net/http"
	"path/filepath"

	"github.com/aguinelo/dcode/internal/protocol"
)

// workspaceOf is the ?workspace= a route about one workspace needs.
//
// None, or a relative one, is refused here, before the daemon is asked: what a
// workspace remembers and which skills it has belong to a project, and a
// relative path resolved against wherever the daemon runs would answer for one
// nobody meant.
func workspaceOf(r *http.Request) (string, *protocol.Error) {
	ws := r.URL.Query().Get("workspace")
	if ws == "" {
		return "", protocol.Errorf(protocol.CodeWorkspaceInvalid, "a workspace is required")
	}
	if !filepath.IsAbs(ws) {
		return "", protocol.Errorf(protocol.CodeWorkspaceInvalid,
			"workspace must be an absolute path, got %q", ws)
	}
	return ws, nil
}

// readMemory answers what a session in ?workspace= reads as memory: every
// entry of its memory file, whether a session shows it, and the blocks that are
// not memories, with why.
func (s *Server) readMemory(w http.ResponseWriter, r *http.Request) {
	ws, perr := workspaceOf(r)
	if perr != nil {
		writeErr(w, perr)
		return
	}
	if s.cfg.Memory == nil {
		// Refused rather than answered empty: an empty memory reads as
		// "nothing was learned here", which is not what happened.
		writeErr(w, protocol.Errorf(protocol.CodeInternal,
			"this daemon cannot say what a workspace remembers"))
		return
	}
	out, err := s.cfg.Memory(ws)
	if err != nil {
		writeErr(w, wrapErr(err))
		return
	}
	// Arrays, not null, so a client iterates them without a special case.
	if out.Entries == nil {
		out.Entries = []protocol.MemoryEntry{}
	}
	if out.Malformed == nil {
		out.Malformed = []protocol.MemoryMalformed{}
	}
	writeJSON(w, http.StatusOK, out)
}

// listSkills answers which skills a session in ?workspace= has available, the
// user's and the project's, and what was said while loading them.
func (s *Server) listSkills(w http.ResponseWriter, r *http.Request) {
	ws, perr := workspaceOf(r)
	if perr != nil {
		writeErr(w, perr)
		return
	}
	if s.cfg.Skills == nil {
		// Refused rather than answered empty: no skill reads as "none is
		// installed", which is not what happened.
		writeErr(w, protocol.Errorf(protocol.CodeInternal,
			"this daemon cannot say which skills a session has"))
		return
	}
	out, err := s.cfg.Skills(ws)
	if err != nil {
		writeErr(w, wrapErr(err))
		return
	}
	if out.Skills == nil {
		out.Skills = []protocol.SkillInfo{}
	}
	if out.Notices == nil {
		out.Notices = []protocol.SkillNotice{}
	}
	writeJSON(w, http.StatusOK, out)
}
