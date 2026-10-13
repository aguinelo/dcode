package server

import (
	"net/http"

	"github.com/aguinelo/dcode/internal/protocol"
)

// listCommands answers which user-defined slash commands a client in
// ?workspace= offers, the user's and the project's, and which of the user's
// the project's took by name.
func (s *Server) listCommands(w http.ResponseWriter, r *http.Request) {
	ws, perr := workspaceOf(r)
	if perr != nil {
		writeErr(w, perr)
		return
	}
	if s.cfg.Commands == nil {
		// Refused rather than answered empty: no command reads as "none was
		// written", which is not what happened.
		writeErr(w, protocol.Errorf(protocol.CodeInternal,
			"this daemon cannot say which commands a workspace has"))
		return
	}
	out, err := s.cfg.Commands(ws)
	if err != nil {
		writeErr(w, wrapErr(err))
		return
	}
	// Arrays, not null, so a client iterates them without a special case.
	if out.Commands == nil {
		out.Commands = []protocol.CommandInfo{}
	}
	if out.Overrides == nil {
		out.Overrides = []string{}
	}
	writeJSON(w, http.StatusOK, out)
}
