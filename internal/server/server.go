// Package server exposes sessions over HTTP with SSE, on a Unix domain socket.
//
// HTTP rather than gRPC: no codegen step, reachable from a future web surface,
// and debuggable with `curl --unix-socket`. The bottleneck is the model, not
// serialisation, so optimising the wire would be optimising the wrong place.
//
// The socket is the trust boundary. There is no authentication in the protocol
// because there is no network surface; exposing it over TCP would be a contract
// change, not a configuration one.
//
// Spec: docs/specs/architecture/client-server-protocol/202608072240-*.
package server

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	ce "github.com/aguinelo/dcode/internal/contextengine"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/aguinelo/dcode/internal/policy"
	"github.com/aguinelo/dcode/internal/protocol"
	"github.com/aguinelo/dcode/internal/session"
	"github.com/aguinelo/dcode/internal/version"
)

// Builder creates a session for a request. Injected so the server does not
// depend on the wiring package, which would be a cycle.
type Builder func(req protocol.CreateSessionRequest) (*session.Session, error)

// Config tunes the server.
type Config struct {
	SocketPath  string
	Manager     *session.Manager
	Build       Builder
	PingEvery   time.Duration
	MaxSessions int
	// DefaultMode and DefaultPolicy are what a session gets when the request
	// does not say. They are here so the boundary warning can be raised once,
	// at boot, rather than on every session that inherits them.
	DefaultMode   policy.SandboxMode
	DefaultPolicy policy.ApprovalPolicy
	// Log receives operational notices. Nil silences them, which is what a
	// test wants and what a daemon must not do.
	Log func(string)
}

// Server serves the protocol.
type Server struct {
	cfg      Config
	mux      *http.ServeMux
	listener net.Listener
	http     *http.Server
}

// New builds a server.
func New(cfg Config) *Server {
	if cfg.PingEvery <= 0 {
		cfg.PingEvery = 20 * time.Second
	}
	// The combination that leaves nothing between the agent and the machine.
	// Warned rather than refused: someone running a throwaway container wants
	// exactly this. What it must not do is happen quietly.
	if w := policy.BoundaryWarning(cfg.DefaultMode, cfg.DefaultPolicy); w != "" && cfg.Log != nil {
		cfg.Log("warn: " + w)
	}

	s := &Server{cfg: cfg, mux: http.NewServeMux()}
	s.routes()
	s.http = &http.Server{Handler: s.mux, ReadHeaderTimeout: 10 * time.Second}
	return s
}

// Handler exposes the routes, for tests that do not want a socket.
func (s *Server) Handler() http.Handler { return s.mux }

// Listen binds the Unix socket with owner-only permissions.
//
// A stale socket from a crashed daemon is removed only after a connection
// attempt fails: deleting it blindly would evict a healthy daemon that happens
// to be running.
func (s *Server) Listen() error {
	path := s.cfg.SocketPath
	if path == "" {
		return fmt.Errorf("server: socket path is required")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}

	if _, err := os.Stat(path); err == nil {
		if conn, derr := net.DialTimeout("unix", path, 300*time.Millisecond); derr == nil {
			conn.Close()
			return fmt.Errorf("server: another dcode daemon is already listening on %s", path)
		}
		if rerr := os.Remove(path); rerr != nil {
			return fmt.Errorf("server: could not remove the stale socket %s: %w", path, rerr)
		}
	}

	ln, err := net.Listen("unix", path)
	if err != nil {
		return err
	}
	// 0700 is the whole access-control story, which is why it is not optional.
	if err := os.Chmod(path, 0o700); err != nil {
		ln.Close()
		return err
	}
	s.listener = ln
	return nil
}

// Serve blocks until the context is cancelled, then shuts down and removes the
// socket so the next start does not have to reason about a stale one.
func (s *Server) Serve(ctx context.Context) error {
	if s.listener == nil {
		if err := s.Listen(); err != nil {
			return err
		}
	}
	go func() {
		<-ctx.Done()
		shutCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = s.http.Shutdown(shutCtx)
		s.cfg.Manager.CloseAll()
		_ = os.Remove(s.cfg.SocketPath)
	}()

	err := s.http.Serve(s.listener)
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}

// Addr returns the socket path.
func (s *Server) Addr() string { return s.cfg.SocketPath }

func (s *Server) routes() {
	p := "/" + protocol.Version
	s.mux.HandleFunc("GET /health", s.health)
	s.mux.HandleFunc("GET /version", s.version)
	s.mux.HandleFunc("POST "+p+"/sessions", s.createSession)
	s.mux.HandleFunc("GET "+p+"/sessions", s.listSessions)
	s.mux.HandleFunc("GET "+p+"/sessions/{id}", s.getSession)
	s.mux.HandleFunc("DELETE "+p+"/sessions/{id}", s.deleteSession)
	s.mux.HandleFunc("GET "+p+"/sessions/{id}/events", s.events)
	s.mux.HandleFunc("POST "+p+"/sessions/{id}/turns", s.submitTurn)
	s.mux.HandleFunc("POST "+p+"/sessions/{id}/interrupt", s.interrupt)
	s.mux.HandleFunc("POST "+p+"/sessions/{id}/undo", s.undo)
	s.mux.HandleFunc("POST "+p+"/sessions/{id}/approvals/{approvalID}", s.resolveApproval)
}

func (s *Server) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) version(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"version":  version.Short(),
		"protocol": protocol.Version,
	})
}

func (s *Server) createSession(w http.ResponseWriter, r *http.Request) {
	var req protocol.CreateSessionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, protocol.Errorf(protocol.CodeWorkspaceInvalid, "malformed request: %v", err))
		return
	}
	if req.Workspace == "" || !filepath.IsAbs(req.Workspace) {
		writeErr(w, protocol.Errorf(protocol.CodeWorkspaceInvalid,
			"workspace must be an absolute path, got %q", req.Workspace))
		return
	}

	sess, err := s.cfg.Build(req)
	if err != nil {
		writeErr(w, wrapErr(err))
		return
	}
	if err := s.cfg.Manager.Add(sess); err != nil {
		sess.Close()
		writeErr(w, wrapErr(err))
		return
	}

	desc := sess.Describe()
	sess.Emit(protocol.EventSessionCreated, desc)
	writeJSON(w, http.StatusCreated, desc)
}

func (s *Server) listSessions(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, s.cfg.Manager.List())
}

func (s *Server) getSession(w http.ResponseWriter, r *http.Request) {
	sess, err := s.cfg.Manager.Get(r.PathValue("id"))
	if err != nil {
		writeErr(w, wrapErr(err))
		return
	}
	writeJSON(w, http.StatusOK, sess.Describe())
}

func (s *Server) deleteSession(w http.ResponseWriter, r *http.Request) {
	if err := s.cfg.Manager.Remove(r.PathValue("id")); err != nil {
		writeErr(w, wrapErr(err))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) submitTurn(w http.ResponseWriter, r *http.Request) {
	sess, err := s.cfg.Manager.Get(r.PathValue("id"))
	if err != nil {
		writeErr(w, wrapErr(err))
		return
	}
	var req protocol.SubmitTurnRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, protocol.Errorf(protocol.CodeInternal, "malformed request: %v", err))
		return
	}
	images, err := decodeImages(req.Images)
	if err != nil {
		writeErr(w, protocol.Errorf(protocol.CodeInternal, "%v", err))
		return
	}
	if err := sess.Submit(req.Text, images...); err != nil {
		writeErr(w, wrapErr(err))
		return
	}
	w.WriteHeader(http.StatusAccepted)
}

// decodeImages turns the wire form back into bytes.
func decodeImages(in []protocol.TurnImage) ([]ce.Image, error) {
	var out []ce.Image
	for _, img := range in {
		body, err := base64.StdEncoding.DecodeString(img.Data)
		if err != nil {
			return nil, fmt.Errorf("an image could not be decoded: %v", err)
		}
		out = append(out, ce.Image{MediaType: img.MediaType, Data: body})
	}
	return out, nil
}

func (s *Server) interrupt(w http.ResponseWriter, r *http.Request) {
	sess, err := s.cfg.Manager.Get(r.PathValue("id"))
	if err != nil {
		writeErr(w, wrapErr(err))
		return
	}
	// Idempotent: the user cannot know the turn just finished on its own.
	sess.Interrupt()
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) undo(w http.ResponseWriter, r *http.Request) {
	sess, err := s.cfg.Manager.Get(r.PathValue("id"))
	if err != nil {
		writeErr(w, wrapErr(err))
		return
	}
	out, err := sess.Undo()
	if err != nil {
		writeErr(w, wrapErr(err))
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) resolveApproval(w http.ResponseWriter, r *http.Request) {
	sess, err := s.cfg.Manager.Get(r.PathValue("id"))
	if err != nil {
		writeErr(w, wrapErr(err))
		return
	}
	var req protocol.ResolveApprovalRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, protocol.Errorf(protocol.CodeInternal, "malformed request: %v", err))
		return
	}
	if err := sess.Resolve(r.PathValue("approvalID"), req.Decision); err != nil {
		writeErr(w, wrapErr(err))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// events streams the session log as SSE.
//
// Replay position is always explicit through `from`, and Last-Event-ID is
// deliberately not honoured: one mechanism means reconnecting behaves exactly
// like connecting for the first time.
func (s *Server) events(w http.ResponseWriter, r *http.Request) {
	sess, err := s.cfg.Manager.Get(r.PathValue("id"))
	if err != nil {
		writeErr(w, wrapErr(err))
		return
	}

	from := uint64(1)
	if v := r.URL.Query().Get("from"); v != "" {
		n, perr := strconv.ParseUint(v, 10, 64)
		if perr != nil {
			writeErr(w, protocol.Errorf(protocol.CodeInternal, "invalid from: %q", v))
			return
		}
		from = n
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		writeErr(w, protocol.Errorf(protocol.CodeInternal, "streaming is not supported here"))
		return
	}

	ctx := r.Context()
	ch, stop, err := sess.Log.Subscribe(ctx, from)
	if err != nil {
		writeErr(w, wrapErr(err))
		return
	}
	defer stop()

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)
	flusher.Flush()

	ping := time.NewTicker(s.cfg.PingEvery)
	defer ping.Stop()

	for {
		select {
		case ev, open := <-ch:
			if !open {
				return
			}
			payload, merr := json.Marshal(ev)
			if merr != nil {
				continue
			}
			fmt.Fprintf(w, "id: %d\ndata: %s\n\n", ev.Seq, payload)
			flusher.Flush()
		case <-ping.C:
			// A comment keeps idle proxies and connections from cutting the
			// stream during a long turn.
			fmt.Fprint(w, ": ping\n\n")
			flusher.Flush()
		case <-ctx.Done():
			// The client went away. The turn keeps running: the session is
			// server-owned and a disconnect is not a cancellation.
			return
		}
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, e *protocol.Error) {
	writeJSON(w, e.Status(), e)
}

// wrapErr turns any error into a protocol error, preserving one that already
// carries a code so the status stays right.
func wrapErr(err error) *protocol.Error {
	if pe, ok := protocol.AsError(err); ok {
		return pe
	}
	msg := err.Error()
	// A workspace problem is the user's to fix and deserves its own status
	// rather than a generic 500.
	if strings.Contains(msg, "workspace") {
		return protocol.Errorf(protocol.CodeWorkspaceInvalid, "%s", msg)
	}
	return protocol.Errorf(protocol.CodeInternal, "%s", msg)
}
