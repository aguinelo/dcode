package server

import (
	"bufio"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/aguinelo/dcode/internal/loop/loopcommand"
	"github.com/aguinelo/dcode/internal/protocol"
)

// heldDaemon is the daemon as a loop sees it, with turns that end only when
// the test says so: the routes are under test, and the sequence is the
// runner's, tested in its own package.
type heldDaemon struct {
	mu      sync.Mutex
	found   []protocol.SpecFolder
	release chan struct{}
	opened  []string // loop id of each session opened
	n       int
}

func (h *heldDaemon) ListSpecs(context.Context, string, bool) ([]protocol.SpecFolder, error) {
	return h.found, nil
}
func (h *heldDaemon) Open(_ context.Context, req protocol.CreateSessionRequest) (protocol.Session, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.n++
	return protocol.Session{ID: "s" + string(rune('0'+h.n))}, nil
}
func (h *heldDaemon) RunTurn(ctx context.Context, _, _ string) (protocol.TurnCompleted, error) {
	select {
	case <-h.release:
		return protocol.TurnCompleted{Reason: "done"}, nil
	case <-ctx.Done():
		return protocol.TurnCompleted{}, ctx.Err()
	}
}
func (h *heldDaemon) CommitDone(context.Context, string) (protocol.CommitDoneResponse, error) {
	return protocol.CommitDoneResponse{Path: "/w/specs/x/done.toml"}, nil
}
func (h *heldDaemon) Interrupt(string) {}

func loopServer(t *testing.T, h *heldDaemon) *Server {
	t.Helper()
	srv, _ := newServer(t, 4)
	srv.loops = newLoopRegistry(func(id string) loopcommand.Transport {
		h.mu.Lock()
		h.opened = append(h.opened, id)
		h.mu.Unlock()
		return h
	})
	t.Cleanup(func() {
		srv.loops.close()
	})
	return srv
}

func post(srv *Server, path string, body any) *httptest.ResponseRecorder {
	raw, _ := json.Marshal(body)
	rec := httptest.NewRecorder()
	srv.mux.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/"+protocol.Version+path, strings.NewReader(string(raw))))
	return rec
}

func decodeRun(t *testing.T, rec *httptest.ResponseRecorder) protocol.LoopRun {
	t.Helper()
	var run protocol.LoopRun
	if err := json.Unmarshal(rec.Body.Bytes(), &run); err != nil {
		t.Fatalf("%v: %s", err, rec.Body.String())
	}
	return run
}

func waitLoop(t *testing.T, srv *Server, id, state string) protocol.LoopRun {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		r, err := srv.loops.get(id)
		if err != nil {
			t.Fatal(err)
		}
		if run := r.Snapshot(); run.State == state {
			return run
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("loop %s never reached %s", id, state)
	return protocol.LoopRun{}
}

// A loop is started, listed, refused a second time in the same workspace,
// stopped, and then a new one starts.
func TestALoopIsStartedListedAndStopped(t *testing.T) {
	h := &heldDaemon{found: []protocol.SpecFolder{{Path: "specs/a", Criteria: 1}}, release: make(chan struct{})}
	srv := loopServer(t, h)

	rec := post(srv, "/loops", protocol.StartLoopRequest{Workspace: "/w", Args: "specs/a"})
	if rec.Code != http.StatusCreated {
		t.Fatalf("start gave %d %s", rec.Code, rec.Body.String())
	}
	run := decodeRun(t, rec)
	if run.ID == "" || run.Workspace != "/w" {
		t.Fatalf("started %+v", run)
	}
	waitLoop(t, srv, run.ID, protocol.LoopWorking)

	// One loop per workspace while it lives.
	rec = post(srv, "/loops", protocol.StartLoopRequest{Workspace: "/w", Args: "specs/a"})
	if rec.Code != http.StatusConflict || !strings.Contains(rec.Body.String(), protocol.CodeLoopAlreadyRunning) {
		t.Errorf("a second loop gave %d %s, want loop_already_running", rec.Code, rec.Body.String())
	}
	// Another workspace is another tree.
	if rec := post(srv, "/loops", protocol.StartLoopRequest{Workspace: "/v", Args: "specs/a"}); rec.Code != http.StatusCreated {
		t.Errorf("a loop elsewhere gave %d %s", rec.Code, rec.Body.String())
	}

	var list protocol.ListLoopsResponse
	if err := json.Unmarshal(get(srv, "/loops?workspace=/w").Body.Bytes(), &list); err != nil {
		t.Fatal(err)
	}
	if len(list.Loops) != 1 || list.Loops[0].ID != run.ID {
		t.Errorf("listed %+v, want only the run in /w", list.Loops)
	}
	if rec := get(srv, "/loops?workspace=rel"); rec.Code != http.StatusBadRequest {
		t.Errorf("a relative workspace gave %d", rec.Code)
	}

	rec = post(srv, "/loops/"+run.ID+"/stop", protocol.StopLoopRequest{When: protocol.StopNow})
	if rec.Code != http.StatusOK {
		t.Fatalf("stop gave %d %s", rec.Code, rec.Body.String())
	}
	waitLoop(t, srv, run.ID, protocol.LoopStopped)
	if rec := post(srv, "/loops", protocol.StartLoopRequest{Workspace: "/w", Args: "specs/a"}); rec.Code != http.StatusCreated {
		t.Errorf("a loop after the last one stopped gave %d %s", rec.Code, rec.Body.String())
	}
}

// What the routes refuse, they refuse with a code that says what happened.
func TestTheLoopRoutesRefuseWithACodeThatSaysWhy(t *testing.T) {
	h := &heldDaemon{found: []protocol.SpecFolder{{Path: "specs/x"}}, release: make(chan struct{})}
	srv := loopServer(t, h)

	cases := []struct {
		name string
		rec  *httptest.ResponseRecorder
		code string
	}{
		{"relative workspace", post(srv, "/loops", protocol.StartLoopRequest{Workspace: "w", Args: "specs/x"}), protocol.CodeWorkspaceInvalid},
		{"no argument", post(srv, "/loops", protocol.StartLoopRequest{Workspace: "/w"}), protocol.CodeInvalidInput},
		{"unknown flag", post(srv, "/loops", protocol.StartLoopRequest{Workspace: "/w", Args: "specs/x --nope"}), protocol.CodeInvalidInput},
		{"unknown loop", post(srv, "/loops/nope/continue", nil), protocol.CodeLoopNotFound},
		{"unknown loop stop", post(srv, "/loops/nope/stop", protocol.StopLoopRequest{When: "now"}), protocol.CodeLoopNotFound},
	}
	for _, c := range cases {
		var e protocol.Error
		_ = json.Unmarshal(c.rec.Body.Bytes(), &e)
		if e.Code != c.code || c.rec.Code != protocol.HTTPStatus(c.code) {
			t.Errorf("%s: %d %s, want %s", c.name, c.rec.Code, c.rec.Body.String(), c.code)
		}
	}

	// A folder that declares nothing is qualified, and the run waits.
	run := decodeRun(t, post(srv, "/loops", protocol.StartLoopRequest{Workspace: "/w", Args: "specs/x"}))
	if rec := post(srv, "/loops/"+run.ID+"/continue", nil); rec.Code != http.StatusConflict ||
		!strings.Contains(rec.Body.String(), protocol.CodeLoopNotAwaitingReview) {
		t.Errorf("continuing a run still qualifying gave %d %s", rec.Code, rec.Body.String())
	}
	if rec := post(srv, "/loops/"+run.ID+"/stop", protocol.StopLoopRequest{When: "later"}); rec.Code != http.StatusBadRequest {
		t.Errorf("stop later gave %d %s", rec.Code, rec.Body.String())
	}
	releaseOne(t, h)
	waitLoop(t, srv, run.ID, protocol.LoopAwaitingReview)
	rec := post(srv, "/loops/"+run.ID+"/continue", nil)
	if rec.Code != http.StatusOK || decodeRun(t, rec).State != protocol.LoopWorking {
		t.Errorf("continue gave %d %s", rec.Code, rec.Body.String())
	}
	if rec := post(srv, "/loops/"+run.ID+"/continue", nil); rec.Code != http.StatusConflict {
		t.Errorf("a second continue gave %d, want the first to win", rec.Code)
	}
	close(h.release)
	waitLoop(t, srv, run.ID, protocol.LoopFinished)
}

// The stream opens with every run and then carries each change, ending in
// the run's end.
func TestTheLoopStreamStartsWithEveryRunAndFollowsThem(t *testing.T) {
	h := &heldDaemon{found: []protocol.SpecFolder{{Path: "specs/a", Criteria: 1}}, release: make(chan struct{})}
	srv := loopServer(t, h)
	first := decodeRun(t, post(srv, "/loops", protocol.StartLoopRequest{Workspace: "/w", Args: "specs/a"}))

	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, ts.URL+"/"+protocol.Version+"/loops/events", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	frames := make(chan protocol.LoopChange, 64)
	go func() {
		sc := bufio.NewScanner(resp.Body)
		for sc.Scan() {
			line := sc.Text()
			if !strings.HasPrefix(line, "data: ") {
				continue
			}
			var ch protocol.LoopChange
			if json.Unmarshal([]byte(strings.TrimPrefix(line, "data: ")), &ch) == nil {
				frames <- ch
			}
		}
		close(frames)
	}()

	snap := within(t, frames)
	if snap.Kind != protocol.LoopSnapshot || len(snap.Loops) != 1 || snap.Loops[0].ID != first.ID {
		t.Fatalf("the stream opened with %+v", snap)
	}
	close(h.release)
	deadline := time.After(5 * time.Second)
	for {
		select {
		case ch, open := <-frames:
			if !open {
				t.Fatal("the stream ended before the run did")
			}
			if ch.Kind == protocol.LoopChanged && ch.Loop != nil && ch.Loop.State == protocol.LoopFinished {
				if ch.Loop.Verdict == nil || ch.Loop.Verdict.Worked != 1 {
					t.Errorf("the run ended without its verdict: %+v", ch.Loop)
				}
				return
			}
		case <-deadline:
			t.Fatal("the stream never said the run finished")
		}
	}
}

// A session a loop opens says which loop it belongs to.
func TestASessionALoopOpensCarriesTheLoop(t *testing.T) {
	srv, mgr := newServer(t, 4)
	sl := serverLoops{s: srv, loopID: "L1"}
	got, err := sl.Open(context.Background(), protocol.CreateSessionRequest{Workspace: "/w"})
	if err != nil {
		t.Fatal(err)
	}
	if got.LoopID != "L1" {
		t.Errorf("the session says loop %q, want L1", got.LoopID)
	}
	sess, err := mgr.Get(got.ID)
	if err != nil {
		t.Fatal(err)
	}
	events, err := sess.Log.Replay(1)
	if err != nil || len(events) == 0 || !strings.Contains(string(events[0].Payload), `"loop_id":"L1"`) {
		t.Errorf("session.created does not carry the loop: %v %v", events, err)
	}
	// A session a client opened belongs to no loop.
	plain := decodeSession(t, post(srv, "/sessions", protocol.CreateSessionRequest{Workspace: "/w"}))
	if plain.LoopID != "" {
		t.Errorf("a session a client opened says loop %q", plain.LoopID)
	}
}

// The daemon side of a loop says what it cannot do instead of pretending.
func TestTheDaemonSideOfALoopSaysWhatItCannotDo(t *testing.T) {
	srv, mgr := newServer(t, 4)
	sl := serverLoops{s: srv, loopID: "L1"}
	if specs, err := sl.ListSpecs(context.Background(), "/w", false); err != nil || specs != nil {
		t.Errorf("a daemon that cannot look answered %v %v", specs, err)
	}
	if _, err := sl.CommitDone(context.Background(), "s1"); err == nil {
		t.Error("a daemon that cannot write a proposal said it did")
	}
	if _, err := sl.RunTurn(context.Background(), "nope", "x"); err == nil {
		t.Error("a turn on a session that does not exist ran")
	}
	s, err := sl.Open(context.Background(), protocol.CreateSessionRequest{Workspace: "/w"})
	if err != nil {
		t.Fatal(err)
	}
	// These sessions carry no engine, so the turn is refused, and said.
	if _, err := sl.RunTurn(context.Background(), s.ID, "x"); err == nil || !strings.Contains(err.Error(), "no engine") {
		t.Errorf("a turn that could not start gave %v", err)
	}
	sl.Interrupt(s.ID)
	sl.Interrupt("nope")
	if _, err := mgr.Get(s.ID); err != nil {
		t.Error(err)
	}
}

func decodeSession(t *testing.T, rec *httptest.ResponseRecorder) protocol.Session {
	t.Helper()
	var s protocol.Session
	if err := json.Unmarshal(rec.Body.Bytes(), &s); err != nil {
		t.Fatalf("%v: %s", err, rec.Body.String())
	}
	return s
}

// The Go client reaches every loop route over a real socket: start, list,
// watch, continue and stop.
func TestTheClientDrivesALoop(t *testing.T) {
	h := &heldDaemon{found: []protocol.SpecFolder{{Path: "specs/x"}}, release: make(chan struct{})}
	srv := loopServer(t, h)
	c := serve(t, srv)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	changes, errs := c.WatchLoops(ctx)
	if first := within(t, changes); first.Kind != protocol.LoopSnapshot || len(first.Loops) != 0 {
		t.Fatalf("the watch opened with %+v", first)
	}
	run, err := c.StartLoop(ctx, protocol.StartLoopRequest{Workspace: "/w", Args: "specs/x"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.StartLoop(ctx, protocol.StartLoopRequest{Workspace: "/w", Args: "specs/x"}); !hasCode(err, protocol.CodeLoopAlreadyRunning) {
		t.Errorf("a second loop gave %v", err)
	}
	releaseOne(t, h)
	waitLoop(t, srv, run.ID, protocol.LoopAwaitingReview)
	list, err := c.ListLoops(ctx, "/w")
	if err != nil || len(list) != 1 || list[0].State != protocol.LoopAwaitingReview {
		t.Fatalf("listed %+v %v", list, err)
	}
	if all, err := c.ListLoops(ctx, ""); err != nil || len(all) != 1 {
		t.Errorf("listing everything gave %+v %v", all, err)
	}
	if got, err := c.ContinueLoop(ctx, run.ID); err != nil || got.State != protocol.LoopWorking {
		t.Fatalf("continue gave %+v %v", got, err)
	}
	if _, err := c.ContinueLoop(ctx, run.ID); !hasCode(err, protocol.CodeLoopNotAwaitingReview) {
		t.Errorf("a second continue gave %v", err)
	}
	if _, err := c.StopLoop(ctx, run.ID, protocol.StopNow); err != nil {
		t.Fatal(err)
	}
	for {
		select {
		case ch := <-changes:
			if ch.Loop != nil && ch.Loop.ID == run.ID && ch.Loop.State == protocol.LoopStopped {
				return
			}
		case err := <-errs:
			t.Fatalf("the watch ended: %v", err)
		case <-time.After(5 * time.Second):
			t.Fatal("the watch never saw the run stop")
		}
	}
}

func hasCode(err error, code string) bool {
	pe, ok := protocol.AsError(err)
	return ok && pe.Code == code
}

// releaseOne ends one held turn, and fails rather than hangs when none runs.
func releaseOne(t *testing.T, h *heldDaemon) {
	t.Helper()
	select {
	case h.release <- struct{}{}:
	case <-time.After(5 * time.Second):
		t.Fatal("no turn was running to end")
	}
}

// within reads one frame, and fails rather than hangs.
func within[T any](t *testing.T, ch <-chan T) T {
	t.Helper()
	select {
	case v := <-ch:
		return v
	case <-time.After(5 * time.Second):
		t.Fatal("nothing arrived")
	}
	var zero T
	return zero
}
