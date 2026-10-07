package server

import (
	"net/http"
	"path/filepath"

	"github.com/aguinelo/dcode/internal/protocol"
)

// listModels answers which models a session can ask for: what a session in
// ?workspace= gets when it asks for none, and every profile it can name, each
// saying whether its family has measurements behind it. Without a workspace,
// the daemon answers for the configuration it started with.
//
// A relative workspace is refused here, before the daemon is asked: it names
// no project, and resolved against wherever the daemon runs it would answer for
// one nobody meant.
func (s *Server) listModels(w http.ResponseWriter, r *http.Request) {
	ws := r.URL.Query().Get("workspace")
	if ws != "" && !filepath.IsAbs(ws) {
		writeErr(w, protocol.Errorf(protocol.CodeWorkspaceInvalid,
			"workspace must be an absolute path, got %q", ws))
		return
	}
	if s.cfg.Models == nil {
		// Refused rather than answered empty: a menu with nothing in it reads
		// as "no model can be asked for", which is never what happened.
		writeErr(w, protocol.Errorf(protocol.CodeInternal,
			"this daemon cannot say which models a session can ask for"))
		return
	}
	out, err := s.cfg.Models(ws)
	if err != nil {
		writeErr(w, wrapErr(err))
		return
	}
	if out.Profiles == nil {
		// An array, not null, so a client iterates it without a special case.
		out.Profiles = []protocol.ModelChoice{}
	}
	writeJSON(w, http.StatusOK, out)
}
