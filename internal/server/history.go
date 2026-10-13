package server

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/aguinelo/dcode/internal/protocol"
	"github.com/aguinelo/dcode/internal/session"
)

// conversationHistory answers one page of a recorded conversation's events,
// read from its record, without loading or continuing it.
//
// A window opening a conversation that had ended showed an empty thread: the
// only route for events was the live stream, and a recorded conversation has
// no stream. Continuing it to see it would start a session nobody asked for.
//
// The id is validated before it touches the disk (session.ValidID), and the
// page is capped (protocol.ConversationEventsLimit): a limit above the cap is
// refused, and a page that does not reach the end says where to read on.
func (s *Server) conversationHistory(w http.ResponseWriter, r *http.Request) {
	if s.cfg.RecordDir == "" {
		writeErr(w, protocol.Errorf(protocol.CodeInvalidInput,
			"this daemon keeps no transcripts, so there is no recorded conversation to read"))
		return
	}
	id := r.PathValue("id")
	if err := session.ValidID(id); err != nil {
		writeErr(w, wrapErr(err))
		return
	}
	from, ok := positiveParam(w, r, "from", 1, 0)
	if !ok {
		return
	}
	limit, ok := positiveParam(w, r, "limit", protocol.ConversationEventsLimit, protocol.ConversationEventsLimit)
	if !ok {
		return
	}

	evs, err := session.History(s.cfg.RecordDir, id)
	switch {
	case errors.Is(err, session.ErrNoSuchSession):
		writeErr(w, protocol.Errorf(protocol.CodeSessionNotFound,
			"no recorded conversation with the id %q", id))
		return
	case err != nil:
		writeErr(w, protocol.Errorf(protocol.CodeInternal,
			"the record of %s could not be read: %v", id, err))
		return
	}

	out := protocol.ConversationEventsResponse{Events: []protocol.Event{}, Total: len(evs)}
	if start := from - 1; start < uint64(len(evs)) {
		end := min(start+limit, uint64(len(evs)))
		out.Events = evs[start:end]
		if end < uint64(len(evs)) {
			out.Next = end + 1
		}
	}
	writeJSON(w, http.StatusOK, out)
}

// positiveParam reads a query parameter that must be a whole number from 1 up
// to ceiling (none when ceiling is zero), answering def when it is absent. A
// value out of range is refused with what was sent, never clamped: a client
// handed less than it asked for, silently, would take it for the whole.
func positiveParam(w http.ResponseWriter, r *http.Request, name string, def, ceiling uint64) (uint64, bool) {
	v := r.URL.Query().Get(name)
	if v == "" {
		return def, true
	}
	n, err := strconv.ParseUint(v, 10, 64)
	if err != nil || n == 0 || (ceiling > 0 && n > ceiling) {
		limit := "a whole number from 1"
		if ceiling > 0 {
			limit += " to " + strconv.FormatUint(ceiling, 10)
		}
		writeErr(w, protocol.Errorf(protocol.CodeInvalidInput, "%s must be %s, got %q", name, limit, v))
		return 0, false
	}
	return n, true
}
