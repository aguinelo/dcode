package server

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"path/filepath"
	"sync"
	"time"

	"github.com/aguinelo/dcode/internal/loop/loopcommand"
	"github.com/aguinelo/dcode/internal/protocol"
	"github.com/aguinelo/dcode/internal/session"
)

// loopRegistry holds the `/loop` runs the daemon conducts, next to the
// sessions they open.
//
// A run lives as long as the daemon and no longer, and NOT as long as the
// client that asked for it: the daemon conducts it, so closing that client
// changes nothing. Ended runs stay, because their verdict is what a client
// shows after the end.
type loopRegistry struct {
	// transport is the daemon as one run sees it: every session it opens
	// belongs to that run.
	transport func(loopID string) loopcommand.Transport

	ctx  context.Context
	stop context.CancelFunc
	wg   sync.WaitGroup

	mu    sync.Mutex
	runs  map[string]*loopcommand.Runner
	order []string
	subs  map[chan protocol.LoopChange]struct{}
}

func newLoopRegistry(transport func(loopID string) loopcommand.Transport) *loopRegistry {
	ctx, stop := context.WithCancel(context.Background())
	return &loopRegistry{transport: transport, ctx: ctx, stop: stop,
		runs: map[string]*loopcommand.Runner{}, subs: map[chan protocol.LoopChange]struct{}{}}
}

// start refuses a second live run in a workspace, then starts one.
func (g *loopRegistry) start(req protocol.StartLoopRequest) (protocol.LoopRun, error) {
	if req.Workspace == "" || !filepath.IsAbs(req.Workspace) {
		return protocol.LoopRun{}, protocol.Errorf(protocol.CodeWorkspaceInvalid,
			"workspace must be an absolute path, got %q", req.Workspace)
	}
	ws := filepath.Clean(req.Workspace)
	req.Workspace = ws

	g.mu.Lock()
	for _, id := range g.order {
		if run := g.runs[id].Snapshot(); run.Workspace == ws && run.Live() {
			g.mu.Unlock()
			// Two loops in one tree measure each other's work.
			return protocol.LoopRun{}, protocol.Errorf(protocol.CodeLoopAlreadyRunning,
				"loop %s is already running in %s; stop it before starting another", id, ws)
		}
	}
	id := session.NewID(time.Now, randomUint32)
	r, err := loopcommand.NewRunner(id, req, g.transport(id), func(protocol.LoopRun) { g.changed(id) }, time.Now)
	if err != nil {
		g.mu.Unlock()
		return protocol.LoopRun{}, err
	}
	g.runs[id] = r
	g.order = append(g.order, id)
	g.mu.Unlock()

	g.wg.Add(1)
	go func() {
		defer g.wg.Done()
		r.Run(g.ctx)
	}()
	snap := r.Snapshot()
	g.changed(id)
	return snap, nil
}

func (g *loopRegistry) get(id string) (*loopcommand.Runner, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	r, ok := g.runs[id]
	if !ok {
		return nil, protocol.Errorf(protocol.CodeLoopNotFound, "no loop %s", id)
	}
	return r, nil
}

// list is every run, oldest first, or one workspace's.
func (g *loopRegistry) list(workspace string) []protocol.LoopRun {
	g.mu.Lock()
	defer g.mu.Unlock()
	out := []protocol.LoopRun{}
	for _, id := range g.order {
		run := g.runs[id].Snapshot()
		if workspace == "" || run.Workspace == filepath.Clean(workspace) {
			out = append(out, run)
		}
	}
	return out
}

// changed sends a run as it is NOW, not the copy that announced the change:
// two changes racing to the stream can arrive in either order, and the frame
// that arrives last must be the newest.
func (g *loopRegistry) changed(id string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	r, ok := g.runs[id]
	if !ok {
		return
	}
	run := r.Snapshot()
	for sub := range g.subs {
		select {
		case sub <- protocol.LoopChange{Kind: protocol.LoopChanged, Loop: &run}:
		default:
			// A subscriber that stopped reading is dropped, never waited for:
			// a loop must not hold up on a list.
			delete(g.subs, sub)
			close(sub)
		}
	}
}

// subscribe opens the stream with every run, then each change.
func (g *loopRegistry) subscribe(ctx context.Context) ([]protocol.LoopRun, <-chan protocol.LoopChange) {
	ch := make(chan protocol.LoopChange, 64)
	snapshot := g.list("")
	g.mu.Lock()
	g.subs[ch] = struct{}{}
	g.mu.Unlock()
	go func() {
		<-ctx.Done()
		g.mu.Lock()
		if _, ok := g.subs[ch]; ok {
			delete(g.subs, ch)
			close(ch)
		}
		g.mu.Unlock()
	}()
	return snapshot, ch
}

// close stops every run and waits for them: the daemon is going away.
func (g *loopRegistry) close() {
	g.stop()
	g.wg.Wait()
}

func randomUint32() uint32 {
	var b [4]byte
	if _, err := rand.Read(b[:]); err != nil {
		return 0
	}
	return binary.BigEndian.Uint32(b[:])
}

// serverLoops is the daemon as a loop sees it: the same sessions, specs and
// commits every client reaches over HTTP, without going through HTTP.
type serverLoops struct {
	s      *Server
	loopID string
}

func (sl serverLoops) ListSpecs(ctx context.Context, ws string, measure bool) ([]protocol.SpecFolder, error) {
	if sl.s.cfg.Specs == nil {
		return nil, nil
	}
	return sl.s.cfg.Specs(ctx, ws, measure)
}

func (sl serverLoops) Open(_ context.Context, req protocol.CreateSessionRequest) (protocol.Session, error) {
	sess, err := sl.s.openSession(req, sl.loopID)
	if err != nil {
		return protocol.Session{}, err
	}
	return sess.Describe(), nil
}

// errSessionGone is a session that closed while a loop waited on its turn.
var errSessionGone = errors.New("the session closed before its turn ended")

// RunTurn listens before it submits, so the end of the turn cannot slip
// between the two.
func (sl serverLoops) RunTurn(ctx context.Context, id, text string) (protocol.TurnCompleted, error) {
	sess, err := sl.s.cfg.Manager.Get(id)
	if err != nil {
		return protocol.TurnCompleted{}, err
	}
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	events, unsubscribe, err := sess.Log.Subscribe(ctx, sess.Log.LastSeq()+1)
	if err != nil {
		return protocol.TurnCompleted{}, err
	}
	defer unsubscribe()
	if err := sess.Submit(text); err != nil {
		return protocol.TurnCompleted{}, err
	}
	for {
		select {
		case ev, open := <-events:
			if !open {
				if ctx.Err() != nil {
					return protocol.TurnCompleted{}, ctx.Err()
				}
				return protocol.TurnCompleted{}, errSessionGone
			}
			if ev.Type != protocol.EventTurnCompleted {
				continue
			}
			var done protocol.TurnCompleted
			if err := json.Unmarshal(ev.Payload, &done); err != nil {
				return protocol.TurnCompleted{}, fmt.Errorf("the end of the turn could not be read: %w", err)
			}
			return done, nil
		case <-ctx.Done():
			return protocol.TurnCompleted{}, ctx.Err()
		}
	}
}

func (sl serverLoops) CommitDone(ctx context.Context, id string) (protocol.CommitDoneResponse, error) {
	if sl.s.cfg.CommitDone == nil {
		return protocol.CommitDoneResponse{}, protocol.Errorf(protocol.CodeInvalidInput,
			"this daemon cannot write a proposed definition of done")
	}
	return sl.s.cfg.CommitDone(ctx, id)
}

func (sl serverLoops) Interrupt(id string) {
	if sess, err := sl.s.cfg.Manager.Get(id); err == nil {
		sess.Interrupt()
	}
}

// startLoop answers POST /loops.
func (s *Server) startLoop(w http.ResponseWriter, r *http.Request) {
	var req protocol.StartLoopRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, protocol.Errorf(protocol.CodeInvalidInput, "malformed request: %v", err))
		return
	}
	run, err := s.loops.start(req)
	if err != nil {
		writeErr(w, wrapErr(err))
		return
	}
	writeJSON(w, http.StatusCreated, run)
}

// listLoops answers GET /loops, oldest first; ?workspace= keeps one project's.
func (s *Server) listLoops(w http.ResponseWriter, r *http.Request) {
	ws := r.URL.Query().Get("workspace")
	if ws != "" && !filepath.IsAbs(ws) {
		writeErr(w, protocol.Errorf(protocol.CodeWorkspaceInvalid,
			"workspace must be an absolute path, got %q", ws))
		return
	}
	writeJSON(w, http.StatusOK, protocol.ListLoopsResponse{Loops: s.loops.list(ws)})
}

// continueLoop takes a run out of awaiting_review.
func (s *Server) continueLoop(w http.ResponseWriter, r *http.Request) {
	run, err := s.loops.get(r.PathValue("id"))
	if err != nil {
		writeErr(w, wrapErr(err))
		return
	}
	out, err := run.Continue()
	if err != nil {
		writeErr(w, wrapErr(err))
		return
	}
	writeJSON(w, http.StatusOK, out)
}

// stopLoop stops a run now or after the folder it is in.
func (s *Server) stopLoop(w http.ResponseWriter, r *http.Request) {
	run, err := s.loops.get(r.PathValue("id"))
	if err != nil {
		writeErr(w, wrapErr(err))
		return
	}
	var req protocol.StopLoopRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, protocol.Errorf(protocol.CodeInvalidInput, "malformed request: %v", err))
		return
	}
	out, err := run.Stop(req.When)
	if err != nil {
		writeErr(w, wrapErr(err))
		return
	}
	writeJSON(w, http.StatusOK, out)
}

// loopEvents streams the runs: every one first, then each change. Without a
// from: reconnecting gets another snapshot, as the list of conversations does.
func (s *Server) loopEvents(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeErr(w, protocol.Errorf(protocol.CodeInternal, "streaming is not supported here"))
		return
	}
	ctx := r.Context()
	snapshot, changes := s.loops.subscribe(ctx)

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)
	if !s.writeLoopChange(w, flusher, protocol.LoopChange{Kind: protocol.LoopSnapshot, Loops: snapshot}) {
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
			if !s.writeLoopChange(w, flusher, ch) {
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

func (s *Server) writeLoopChange(w http.ResponseWriter, flusher http.Flusher, ch protocol.LoopChange) bool {
	payload, err := json.Marshal(ch)
	if err != nil {
		if s.cfg.Log != nil {
			s.cfg.Log(fmt.Sprintf("the list of loops could not be sent: %v", err))
		}
		return false
	}
	fmt.Fprintf(w, "data: %s\n\n", payload)
	flusher.Flush()
	return true
}
