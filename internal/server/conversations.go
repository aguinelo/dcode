package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
	"time"

	"github.com/aguinelo/dcode/internal/protocol"
)

// ConversationIndex is the list of conversations a daemon keeps: the live ones
// and the recorded ones, with a stream of what changes.
type ConversationIndex interface {
	List(workspace string) ([]protocol.Conversation, error)
	Subscribe(ctx context.Context) ([]protocol.Conversation, <-chan protocol.ConversationChange, error)
}

// listConversations answers every conversation, live and recorded, newest
// activity first; ?workspace= keeps one project's.
func (s *Server) listConversations(w http.ResponseWriter, r *http.Request) {
	ws := r.URL.Query().Get("workspace")
	if ws != "" && !filepath.IsAbs(ws) {
		writeErr(w, protocol.Errorf(protocol.CodeWorkspaceInvalid,
			"workspace must be an absolute path, got %q", ws))
		return
	}
	list := []protocol.Conversation{}
	if s.cfg.Conversations != nil {
		found, err := s.cfg.Conversations.List(ws)
		if err != nil {
			writeErr(w, wrapErr(err))
			return
		}
		if found != nil {
			list = found
		}
	}
	writeJSON(w, http.StatusOK, protocol.ListConversationsResponse{Conversations: list})
}

// conversationEvents streams the list: the whole of it first, then each
// conversation that changes in something a list shows. A client that falls
// too far behind is dropped and reconnects to a fresh snapshot, which it
// applies like the first — so the stream needs no sequence to resume from.
func (s *Server) conversationEvents(w http.ResponseWriter, r *http.Request) {
	if s.cfg.Conversations == nil {
		writeErr(w, protocol.Errorf(protocol.CodeInternal, "this daemon keeps no list of conversations"))
		return
	}
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeErr(w, protocol.Errorf(protocol.CodeInternal, "streaming is not supported here"))
		return
	}

	ctx := r.Context()
	snapshot, changes, err := s.cfg.Conversations.Subscribe(ctx)
	if err != nil {
		writeErr(w, wrapErr(err))
		return
	}
	if snapshot == nil {
		snapshot = []protocol.Conversation{}
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)
	if !s.writeChange(w, flusher, protocol.ConversationChange{Kind: protocol.ConversationSnapshot, Conversations: snapshot}) {
		return
	}

	ping := time.NewTicker(s.cfg.PingEvery)
	defer ping.Stop()
	for {
		select {
		case ch, open := <-changes:
			if !open {
				return
			}
			if !s.writeChange(w, flusher, ch) {
				return
			}
		case <-ping.C:
			fmt.Fprint(w, ": ping\n\n")
			flusher.Flush()
		case <-ctx.Done():
			return
		}
	}
}

// writeChange sends one frame, and reports whether the stream can go on. A
// frame that cannot be encoded ends the stream and is said: a client would
// rather reconnect to a snapshot than read a list with a hole in it.
func (s *Server) writeChange(w http.ResponseWriter, flusher http.Flusher, ch protocol.ConversationChange) bool {
	payload, err := json.Marshal(ch)
	if err != nil {
		if s.cfg.Log != nil {
			s.cfg.Log(fmt.Sprintf("the list of conversations could not be sent: %v", err))
		}
		return false
	}
	fmt.Fprintf(w, "data: %s\n\n", payload)
	flusher.Flush()
	return true
}
