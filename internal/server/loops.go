package server

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
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

	// log receives what the registry drops or cannot do. Nil silences it,
	// which is what a test wants and what a daemon must not do.
	log func(string)

	mu    sync.Mutex
	runs  map[string]*loopcommand.Runner
	order []string
	// roots is each run's workspace with its links resolved, so two spellings
	// of one tree are one tree.
	roots map[string]string
	subs  map[chan protocol.LoopChange]struct{}
}

func newLoopRegistry(transport func(loopID string) loopcommand.Transport) *loopRegistry {
	ctx, stop := context.WithCancel(context.Background())
	return &loopRegistry{transport: transport, ctx: ctx, stop: stop,
		runs: map[string]*loopcommand.Runner{}, roots: map[string]string{},
		subs: map[chan protocol.LoopChange]struct{}{}}
}

// loopRoot checks a workspace the way a session's is checked — absolute, there,
// a folder — and resolves its links.
func loopRoot(ws string) (string, error) {
	if ws == "" || !filepath.IsAbs(ws) {
		return "", protocol.Errorf(protocol.CodeWorkspaceInvalid,
			"workspace must be an absolute path, got %q", ws)
	}
	info, err := os.Stat(ws)
	if err != nil {
		return "", protocol.Errorf(protocol.CodeWorkspaceInvalid, "workspace %s cannot be read: %v", ws, err)
	}
	if !info.IsDir() {
		return "", protocol.Errorf(protocol.CodeWorkspaceInvalid, "workspace %s is not a folder", ws)
	}
	root, err := filepath.EvalSymlinks(ws)
	if err != nil {
		return "", protocol.Errorf(protocol.CodeWorkspaceInvalid, "workspace %s cannot be resolved: %v", ws, err)
	}
	return root, nil
}

// overlaps reports two trees where one holds the other. Both end in exactly
// one separator before comparing, so the root "/" holds every tree rather
// than being compared as "//".
func overlaps(a, b string) bool {
	sep := string(filepath.Separator)
	a, b = strings.TrimSuffix(a, sep)+sep, strings.TrimSuffix(b, sep)+sep
	return strings.HasPrefix(a, b) || strings.HasPrefix(b, a)
}

// start refuses a second live run in, inside or around a live run's tree, then
// starts one.
func (g *loopRegistry) start(req protocol.StartLoopRequest) (protocol.LoopRun, error) {
	root, err := loopRoot(req.Workspace)
	if err != nil {
		return protocol.LoopRun{}, err
	}
	req.Workspace = filepath.Clean(req.Workspace)

	g.mu.Lock()
	defer g.mu.Unlock()
	if g.ctx.Err() != nil {
		return protocol.LoopRun{}, protocol.Errorf(protocol.CodeInternal,
			"the daemon is shutting down; no loop starts now")
	}
	for _, id := range g.order {
		if g.runs[id].Snapshot().Live() && overlaps(g.roots[id], root) {
			// Two loops in one tree measure each other's work.
			return protocol.LoopRun{}, protocol.Errorf(protocol.CodeLoopAlreadyRunning,
				"loop %s is already running in %s; stop it before starting another", id, g.roots[id])
		}
	}
	id, err := newLoopID()
	if err != nil {
		return protocol.LoopRun{}, err
	}
	r, err := loopcommand.NewRunner(id, req, g.transport(id), func(protocol.LoopRun) { g.changed(id) }, time.Now)
	if err != nil {
		return protocol.LoopRun{}, err
	}
	g.runs[id], g.roots[id] = r, root
	g.order = append(g.order, id)
	// Under the lock that checked the context: close cannot slip between the
	// check and the Add and wait for a group that has not counted this run.
	g.wg.Add(1)
	go func() {
		defer g.wg.Done()
		r.Run(g.ctx)
	}()
	snap := r.Snapshot()
	g.sendLocked(id)
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
	return g.listLocked(workspace)
}

func (g *loopRegistry) listLocked(workspace string) []protocol.LoopRun {
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
	g.sendLocked(id)
}

func (g *loopRegistry) sendLocked(id string) {
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
			// a loop must not hold up on a list. Said, because a client left
			// behind sees a list that stopped moving.
			delete(g.subs, sub)
			close(sub)
			if g.log != nil {
				g.log(fmt.Sprintf("a subscriber to the list of loops stopped reading and was dropped (at loop %s)", id))
			}
		}
	}
}

// subscribe opens the stream with every run, then each change. The snapshot
// and the registration are one critical section: a change between them would
// be in neither.
func (g *loopRegistry) subscribe(ctx context.Context) ([]protocol.LoopRun, <-chan protocol.LoopChange) {
	ch := make(chan protocol.LoopChange, 64)
	g.mu.Lock()
	snapshot := g.listLocked("")
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
	g.mu.Lock()
	g.stop()
	g.mu.Unlock()
	g.wg.Wait()
}

// randRead draws the entropy of a loop id; a variable so a test can fail it.
var randRead = rand.Read

// newLoopID is a time-ordered id, or the reason there is none. Never zeroes in
// place of the entropy: two loops in one millisecond would share a name.
func newLoopID() (string, error) {
	var b [4]byte
	if _, err := randRead(b[:]); err != nil {
		return "", protocol.Errorf(protocol.CodeInternal, "a loop id could not be drawn: %v", err)
	}
	return session.NewID(time.Now, func() uint32 { return binary.BigEndian.Uint32(b[:]) }), nil
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
	// A stopped loop submits nothing: the context is checked before the turn
	// exists, not only while it runs.
	if err := ctx.Err(); err != nil {
		return protocol.TurnCompleted{}, err
	}
	stopped := ctx
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	events, unsubscribe, err := sess.Log.Subscribe(ctx, sess.Log.LastSeq()+1)
	if err != nil {
		return protocol.TurnCompleted{}, err
	}
	defer unsubscribe()
	// Cancelled once the turn is under way, the turn is interrupted: the loop
	// saying stopped must not leave the session working behind it.
	defer func() {
		if stopped.Err() != nil {
			sess.Interrupt()
		}
	}()
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
