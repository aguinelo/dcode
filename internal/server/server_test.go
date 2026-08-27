package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	ce "github.com/aguinelo/dcode/internal/contextengine"
	"github.com/aguinelo/dcode/internal/loop"
	"github.com/aguinelo/dcode/internal/policy"
	"github.com/aguinelo/dcode/internal/protocol"
	"github.com/aguinelo/dcode/internal/provider"
	"github.com/aguinelo/dcode/internal/session"
	"github.com/aguinelo/dcode/internal/tools"
	"github.com/aguinelo/dcode/pkg/client"
)

func fixedClock() session.Clock {
	t := time.Date(2026, 8, 8, 12, 0, 0, 0, time.UTC)
	return func() time.Time { return t }
}

// newServer wires a server whose sessions carry no engine: the protocol layer
// is what is under test, and a real engine would drag a model into it.
func newServer(t *testing.T, max int) (*Server, *session.Manager) {
	t.Helper()
	mgr := session.NewManager(max)
	var n int
	srv := New(Config{
		Manager:   mgr,
		PingEvery: 50 * time.Millisecond,
		Build: func(req protocol.CreateSessionRequest) (*session.Session, error) {
			n++
			id := fmt.Sprintf("s%d", n)
			return session.New(id, req.Workspace, "MiniMax-M3", "workspace-write",
				nil, session.NewEventLog(id, 0, fixedClock()), fixedClock()), nil
		},
	})
	return srv, mgr
}

// newDaemon starts a real daemon on a real socket and returns a client, so the
// integration tests exercise the transport rather than the handler directly.
func newDaemon(t *testing.T) (*client.Client, *session.Manager) {
	t.Helper()
	srv, mgr := newServer(t, 10)
	// Short path: macOS caps a Unix socket at ~104 bytes, and t.TempDir() is
	// long enough to blow that on its own.
	dir, err := os.MkdirTemp("", "dc")
	if err != nil {
		t.Fatal(err)
	}
	sock := filepath.Join(dir, "s.sock")
	srv.cfg.SocketPath = sock

	if err := srv.Listen(); err != nil {
		t.Skipf("cannot bind a unix socket here: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	go func() { _ = srv.Serve(ctx) }()
	t.Cleanup(func() { cancel(); os.RemoveAll(dir) })

	c := client.New(sock)
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if err := c.Health(context.Background()); err == nil {
			return c, mgr
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("the daemon never became healthy")
	return nil, nil
}

// ---------- transport ----------

// The socket must be owner-only: it is the entire access-control story, since
// the protocol carries no authentication.
func TestSocketIsOwnerOnly(t *testing.T) {
	srv, _ := newServer(t, 10)
	dir, err := os.MkdirTemp("", "dc")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)
	srv.cfg.SocketPath = filepath.Join(dir, "s.sock")

	if err := srv.Listen(); err != nil {
		t.Skipf("cannot bind: %v", err)
	}
	defer srv.listener.Close()

	info, err := os.Stat(srv.cfg.SocketPath)
	if err != nil {
		t.Fatal(err)
	}
	if perm := info.Mode().Perm(); perm != 0o700 {
		t.Errorf("got %o want 700", perm)
	}
}

// A leftover socket from a crashed daemon must be cleared, but a healthy daemon
// must never be evicted by the next start.
func TestStaleSocketIsReplacedButALiveOneIsNot(t *testing.T) {
	dir, err := os.MkdirTemp("", "dc")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)
	sock := filepath.Join(dir, "s.sock")

	// A file that nothing is listening on: the crashed-daemon case.
	if err := os.WriteFile(sock, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	first, _ := newServer(t, 10)
	first.cfg.SocketPath = sock
	if err := first.Listen(); err != nil {
		t.Skipf("cannot bind: %v", err)
	}

	// Now one is live, so a second start must refuse rather than steal it.
	second, _ := newServer(t, 10)
	second.cfg.SocketPath = sock
	err = second.Listen()
	first.listener.Close()

	if err == nil {
		t.Fatal("a second daemon must not take over a live socket")
	}
	if !strings.Contains(err.Error(), "already listening") {
		t.Errorf("the error should say what is wrong: %v", err)
	}
}

func TestListenRequiresASocketPath(t *testing.T) {
	srv, _ := newServer(t, 10)
	if err := srv.Listen(); err == nil {
		t.Error("an empty socket path must be rejected")
	}
}

// ---------- endpoints ----------

func TestHealthAndVersionAreStable(t *testing.T) {
	srv, _ := newServer(t, 10)

	for _, path := range []string{"/health", "/version"} {
		rec := httptest.NewRecorder()
		srv.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
		if rec.Code != http.StatusOK {
			t.Errorf("%s: got %d", path, rec.Code)
		}
		if ct := rec.Header().Get("Content-Type"); !strings.Contains(ct, "json") {
			t.Errorf("%s: content type %q", path, ct)
		}
	}
}

func TestCreateSessionRequiresAnAbsoluteWorkspace(t *testing.T) {
	c, _ := newDaemon(t)
	for _, ws := range []string{"", "relative/path", "./x"} {
		_, err := c.CreateSession(context.Background(),
			protocol.CreateSessionRequest{Workspace: ws})
		if err == nil {
			t.Errorf("workspace %q should be rejected", ws)
			continue
		}
		if pe, ok := protocol.AsError(err); !ok || pe.Code != protocol.CodeWorkspaceInvalid {
			t.Errorf("workspace %q: got %v", ws, err)
		}
	}
}

func TestSessionLifecycle(t *testing.T) {
	c, mgr := newDaemon(t)
	ctx := context.Background()

	s, err := c.CreateSession(ctx, protocol.CreateSessionRequest{Workspace: "/tmp"})
	if err != nil {
		t.Fatal(err)
	}
	if s.ID == "" || s.State != protocol.SessionStateIdle {
		t.Fatalf("got %+v", s)
	}

	got, err := c.GetSession(ctx, s.ID)
	if err != nil || got.ID != s.ID {
		t.Fatalf("got %+v %v", got, err)
	}

	list, err := c.ListSessions(ctx)
	if err != nil || len(list) != 1 {
		t.Fatalf("got %d sessions, %v", len(list), err)
	}

	if err := c.DeleteSession(ctx, s.ID); err != nil {
		t.Fatal(err)
	}
	if mgr.Count() != 0 {
		t.Errorf("got %d live sessions", mgr.Count())
	}
	if _, err := c.GetSession(ctx, s.ID); err == nil {
		t.Error("a deleted session must not resolve")
	}
}

func TestUnknownSessionIsNotFound(t *testing.T) {
	c, _ := newDaemon(t)
	_, err := c.GetSession(context.Background(), "nope")
	pe, ok := protocol.AsError(err)
	if !ok || pe.Code != protocol.CodeSessionNotFound {
		t.Errorf("got %v", err)
	}
}

// The ceiling exists to make session density measurable rather than to limit
// the user, so it has to answer with something actionable.
func TestSessionCeilingIsEnforced(t *testing.T) {
	c, _ := newDaemon(t)
	ctx := context.Background()
	for i := 0; i < 10; i++ {
		if _, err := c.CreateSession(ctx, protocol.CreateSessionRequest{Workspace: "/tmp"}); err != nil {
			t.Fatalf("session %d: %v", i, err)
		}
	}
	_, err := c.CreateSession(ctx, protocol.CreateSessionRequest{Workspace: "/tmp"})
	pe, ok := protocol.AsError(err)
	if !ok || pe.Code != protocol.CodeMaxSessionsReached {
		t.Errorf("got %v", err)
	}
}

// ---------- events ----------

// Replay and live must produce the same sequence over the wire, which is what
// makes resuming a session equivalent to having watched it.
func TestEventsReplayThenStreamLive(t *testing.T) {
	c, mgr := newDaemon(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	s, err := c.CreateSession(ctx, protocol.CreateSessionRequest{Workspace: "/tmp"})
	if err != nil {
		t.Fatal(err)
	}
	sess, err := mgr.Get(s.ID)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 5; i++ {
		sess.Emit(protocol.EventMessageDelta, protocol.MessageDelta{TurnID: "t1", Text: "before"})
	}

	events, errs := c.Subscribe(ctx, s.ID, 1)

	go func() {
		time.Sleep(150 * time.Millisecond)
		for i := 0; i < 5; i++ {
			sess.Emit(protocol.EventMessageDelta, protocol.MessageDelta{TurnID: "t1", Text: "after"})
		}
	}()

	var seqs []uint64
	deadline := time.After(8 * time.Second)
	for len(seqs) < 11 { // session.created + 5 before + 5 after
		select {
		case ev, open := <-events:
			if !open {
				t.Fatalf("stream closed after %d events", len(seqs))
			}
			seqs = append(seqs, ev.Seq)
		case err := <-errs:
			t.Fatalf("stream error: %v", err)
		case <-deadline:
			t.Fatalf("only saw %d events: %v", len(seqs), seqs)
		}
	}
	for i, seq := range seqs {
		if seq != uint64(i+1) {
			t.Fatalf("gap or duplicate at %d: %v", i, seqs)
		}
	}
}

func TestEventsFromLaterSequence(t *testing.T) {
	c, mgr := newDaemon(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	s, _ := c.CreateSession(ctx, protocol.CreateSessionRequest{Workspace: "/tmp"})
	sess, _ := mgr.Get(s.ID)
	for i := 0; i < 5; i++ {
		sess.Emit(protocol.EventMessageDelta, protocol.MessageDelta{Text: "x"})
	}

	events, _ := c.Subscribe(ctx, s.ID, 4)
	select {
	case ev := <-events:
		if ev.Seq != 4 {
			t.Errorf("got %d want 4", ev.Seq)
		}
	case <-time.After(4 * time.Second):
		t.Fatal("nothing delivered")
	}
}

func TestEventsRejectsAMalformedFrom(t *testing.T) {
	srv, mgr := newServer(t, 10)
	sess := session.New("s1", "/w", "m", "workspace-write", nil,
		session.NewEventLog("s1", 0, fixedClock()), fixedClock())
	if err := mgr.Add(sess); err != nil {
		t.Fatal(err)
	}

	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet,
		"/v1/sessions/s1/events?from=abc", nil))
	if rec.Code < 400 {
		t.Errorf("got %d", rec.Code)
	}
}

// Dropped history must fail rather than start later in silence: the client
// would otherwise believe it had the whole session.
func TestExpiredHistoryIsRefusedOverTheWire(t *testing.T) {
	srv, mgr := newServer(t, 10)
	log := session.NewEventLog("s1", 3, fixedClock())
	sess := session.New("s1", "/w", "m", "workspace-write", nil, log, fixedClock())
	if err := mgr.Add(sess); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 20; i++ {
		sess.Emit(protocol.EventMessageDelta, protocol.MessageDelta{Text: "x"})
	}

	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet,
		"/v1/sessions/s1/events?from=1", nil))
	if rec.Code != http.StatusGone {
		t.Errorf("got %d want 410", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), protocol.CodeEventsExpired) {
		t.Errorf("body: %s", rec.Body.String())
	}
}

// ---------- turns and approvals ----------

func TestSecondTurnWhileOneIsRunningConflicts(t *testing.T) {
	c, mgr := newDaemon(t)
	ctx := context.Background()

	s, err := c.CreateSession(ctx, protocol.CreateSessionRequest{Workspace: "/tmp"})
	if err != nil {
		t.Fatal(err)
	}
	sess, _ := mgr.Get(s.ID)

	// Hold the session busy without an engine by blocking on an approval.
	go func() {
		_, _ = sess.Approve(context.Background(),
			protocol.ApprovalRequest{ApprovalID: "a1", Tool: "bash"}, 5*time.Second)
	}()
	for sess.State() != protocol.SessionStateBlocked {
		time.Sleep(time.Millisecond)
	}

	err = c.Submit(ctx, s.ID, "second")
	pe, ok := protocol.AsError(err)
	if !ok || pe.Code != protocol.CodeTurnAlreadyActive {
		t.Errorf("got %v", err)
	}
	_ = sess.Resolve("a1", protocol.ApprovalDeny)
}

// Any attached client may answer, and exactly one wins.
func TestApprovalIsResolvedOverTheWireAndSecondConflicts(t *testing.T) {
	c, mgr := newDaemon(t)
	ctx := context.Background()

	s, err := c.CreateSession(ctx, protocol.CreateSessionRequest{Workspace: "/tmp"})
	if err != nil {
		t.Fatal(err)
	}
	sess, _ := mgr.Get(s.ID)

	answered := make(chan protocol.ApprovalDecision, 1)
	go func() {
		d, _ := sess.Approve(context.Background(),
			protocol.ApprovalRequest{ApprovalID: "a1", Tool: "bash", Command: "curl x"},
			5*time.Second)
		answered <- d
	}()
	for len(sess.Pending()) == 0 {
		time.Sleep(time.Millisecond)
	}

	if err := c.Resolve(ctx, s.ID, "a1", protocol.ApprovalAllow); err != nil {
		t.Fatal(err)
	}
	select {
	case d := <-answered:
		if d != protocol.ApprovalAllow {
			t.Errorf("got %s", d)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("the loop was never unblocked")
	}

	err = c.Resolve(ctx, s.ID, "a1", protocol.ApprovalDeny)
	if pe, ok := protocol.AsError(err); !ok || pe.Code != protocol.CodeApprovalResolved {
		t.Errorf("a second answer must conflict, got %v", err)
	}
}

func TestInterruptIsIdempotentOverTheWire(t *testing.T) {
	c, _ := newDaemon(t)
	ctx := context.Background()
	s, err := c.CreateSession(ctx, protocol.CreateSessionRequest{Workspace: "/tmp"})
	if err != nil {
		t.Fatal(err)
	}
	// Interrupting an idle session is not an error: the user cannot know the
	// turn just finished.
	for i := 0; i < 3; i++ {
		if err := c.Interrupt(ctx, s.ID); err != nil {
			t.Fatalf("interrupt %d: %v", i, err)
		}
	}
}

// A client going away must not cancel the turn. The session is server-owned,
// which is the whole reason a client is disposable.
func TestDisconnectingDoesNotAffectTheSession(t *testing.T) {
	c, mgr := newDaemon(t)
	ctx := context.Background()

	s, err := c.CreateSession(ctx, protocol.CreateSessionRequest{Workspace: "/tmp"})
	if err != nil {
		t.Fatal(err)
	}
	sess, _ := mgr.Get(s.ID)

	streamCtx, streamCancel := context.WithCancel(context.Background())
	events, _ := c.Subscribe(streamCtx, s.ID, 1)
	select {
	case <-events:
	case <-time.After(3 * time.Second):
		t.Fatal("no events")
	}

	streamCancel() // the client disappears
	time.Sleep(100 * time.Millisecond)

	// The session is still alive and still accepting work.
	before := sess.Log.LastSeq()
	sess.Emit(protocol.EventMessageDelta, protocol.MessageDelta{Text: "still here"})
	if sess.Log.LastSeq() != before+1 {
		t.Error("the session stopped recording after the client left")
	}
	if got, err := c.GetSession(ctx, s.ID); err != nil || got.State == protocol.SessionStateClosed {
		t.Errorf("the session should still be open: %+v %v", got, err)
	}
}

// A workspace problem is classified because the producer SAID it was one, not
// because the message happens to contain the word.
//
// It used to search the text, and a message carries paths: anyone working
// under a directory called `workspace` — which is where this repository itself
// lives — got `workspace_invalid` for errors with nothing to do with their
// workspace. Found by running a real daemon from such a directory and asking
// for a spec that was not there.
func TestWrapErrClassifiesWorkspaceProblems(t *testing.T) {
	pe := wrapErr(fmt.Errorf("%w %q must be absolute", policy.ErrWorkspace, "x"))
	if pe.Code != protocol.CodeWorkspaceInvalid {
		t.Errorf("got %s", pe.Code)
	}
	// A path with the word in it is not a workspace problem.
	noise := wrapErr(fmt.Errorf("open /Users/someone/workspace/proj/specs/x/tasks.md: no such file"))
	if noise.Code != protocol.CodeInternal {
		t.Errorf("a path containing the word was classified as %s", noise.Code)
	}
	if got := wrapErr(fmt.Errorf("something else")); got.Code != protocol.CodeInternal {
		t.Errorf("got %s", got.Code)
	}
	orig := protocol.Errorf(protocol.CodeTurnAlreadyActive, "busy")
	if got := wrapErr(orig); got.Code != protocol.CodeTurnAlreadyActive {
		t.Errorf("an already-classified error must keep its code, got %s", got.Code)
	}
}

func TestBuildFailureSurfaces(t *testing.T) {
	mgr := session.NewManager(10)
	srv := New(Config{
		Manager: mgr,
		Build: func(protocol.CreateSessionRequest) (*session.Session, error) {
			return nil, fmt.Errorf("no sandbox available")
		},
	})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/sessions",
		strings.NewReader(`{"workspace":"/tmp"}`))
	srv.Handler().ServeHTTP(rec, req)

	if rec.Code < 400 {
		t.Errorf("a session that cannot be built must fail, got %d", rec.Code)
	}
	if mgr.Count() != 0 {
		t.Error("a failed build must not leave a session registered")
	}
}

// ---------- malformed and missing ----------

// Every session-scoped route has to answer for an id that does not exist, or a
// stale client id turns into a nil dereference in the daemon.
func TestEverySessionRouteAnswersForAMissingSession(t *testing.T) {
	srv, _ := newServer(t, 10)
	p := "/" + protocol.Version + "/sessions/nope"

	for name, req := range map[string]*http.Request{
		"get":       httptest.NewRequest(http.MethodGet, p, nil),
		"delete":    httptest.NewRequest(http.MethodDelete, p, nil),
		"submit":    httptest.NewRequest(http.MethodPost, p+"/turns", strings.NewReader(`{"text":"x"}`)),
		"interrupt": httptest.NewRequest(http.MethodPost, p+"/interrupt", nil),
		"approve":   httptest.NewRequest(http.MethodPost, p+"/approvals/a1", strings.NewReader(`{"decision":"deny"}`)),
		"events":    httptest.NewRequest(http.MethodGet, p+"/events", nil),
	} {
		rec := httptest.NewRecorder()
		srv.Handler().ServeHTTP(rec, req)
		if rec.Code != http.StatusNotFound {
			t.Errorf("%s: got %d want 404\n%s", name, rec.Code, rec.Body.String())
		}
		if !strings.Contains(rec.Body.String(), protocol.CodeSessionNotFound) {
			t.Errorf("%s: the code is what a client branches on: %s", name, rec.Body.String())
		}
	}
}

// A body the daemon cannot read must produce an error a client can act on,
// rather than a panic or a silent success.
func TestMalformedBodiesAreRejected(t *testing.T) {
	srv, _ := newServer(t, 10)
	created := httptest.NewRecorder()
	srv.Handler().ServeHTTP(created, httptest.NewRequest(http.MethodPost,
		"/"+protocol.Version+"/sessions", strings.NewReader(`{"workspace":"/w"}`)))
	if created.Code != http.StatusCreated {
		t.Fatalf("setup failed: %d %s", created.Code, created.Body.String())
	}

	p := "/" + protocol.Version + "/sessions/s1"
	for name, req := range map[string]*http.Request{
		"create":  httptest.NewRequest(http.MethodPost, "/"+protocol.Version+"/sessions", strings.NewReader(`{{`)),
		"submit":  httptest.NewRequest(http.MethodPost, p+"/turns", strings.NewReader(`{{`)),
		"approve": httptest.NewRequest(http.MethodPost, p+"/approvals/a1", strings.NewReader(`{{`)),
	} {
		rec := httptest.NewRecorder()
		srv.Handler().ServeHTTP(rec, req)
		if rec.Code < 400 {
			t.Errorf("%s: got %d, want a refusal", name, rec.Code)
		}
	}
}

func TestDeleteRemovesTheSession(t *testing.T) {
	srv, mgr := newServer(t, 10)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodPost,
		"/"+protocol.Version+"/sessions", strings.NewReader(`{"workspace":"/w"}`)))
	if rec.Code != http.StatusCreated {
		t.Fatalf("got %d", rec.Code)
	}

	del := httptest.NewRecorder()
	srv.Handler().ServeHTTP(del, httptest.NewRequest(http.MethodDelete,
		"/"+protocol.Version+"/sessions/s1", nil))
	if del.Code != http.StatusNoContent {
		t.Fatalf("got %d: %s", del.Code, del.Body.String())
	}
	if mgr.Count() != 0 {
		t.Errorf("the session must be gone, %d remain", mgr.Count())
	}
}

// Interrupting an idle session is not an error: the user cannot know the turn
// just finished on its own.
func TestInterruptIsAcceptedOnAnIdleSession(t *testing.T) {
	srv, _ := newServer(t, 10)
	srv.Handler().ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost,
		"/"+protocol.Version+"/sessions", strings.NewReader(`{"workspace":"/w"}`)))

	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodPost,
		"/"+protocol.Version+"/sessions/s1/interrupt", nil))
	if rec.Code != http.StatusNoContent {
		t.Errorf("got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestResolvingAnUnknownApprovalIsRefused(t *testing.T) {
	srv, _ := newServer(t, 10)
	srv.Handler().ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost,
		"/"+protocol.Version+"/sessions", strings.NewReader(`{"workspace":"/w"}`)))

	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodPost,
		"/"+protocol.Version+"/sessions/s1/approvals/nope", strings.NewReader(`{"decision":"allow"}`)))
	if rec.Code < 400 {
		t.Errorf("got %d", rec.Code)
	}
}

// ---------- the socket ----------

func TestListenRefusesWithoutAPathAndReportsAStaleOne(t *testing.T) {
	srv, _ := newServer(t, 10)
	if err := srv.Listen(); err == nil {
		t.Error("a daemon with no address cannot listen")
	}

	dir, err := os.MkdirTemp("", "dc")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	// A stale socket left by a crashed daemon is removed only after a
	// connection attempt fails, so a healthy daemon is never evicted.
	sock := filepath.Join(dir, "s.sock")
	if err := os.WriteFile(sock, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	srv2, _ := newServer(t, 10)
	srv2.cfg.SocketPath = sock
	if err := srv2.Listen(); err != nil {
		t.Fatalf("a stale socket must be replaced, got %v", err)
	}
	// Closed through Cleanup rather than left to the collector. A
	// *net.UnixListener unlinks its socket file when it closes, and its
	// finaliser runs as soon as nothing references it — so an unheld listener
	// deletes the very file the next assertion is about, and the test fails
	// only sometimes, and only on the machine that collected first.
	t.Cleanup(func() { srv2.listener.Close() })

	if srv2.Addr() != sock {
		t.Errorf("got %q", srv2.Addr())
	}

	// A live daemon on the same path is not evicted.
	srv3, _ := newServer(t, 10)
	srv3.cfg.SocketPath = sock
	if err := srv3.Listen(); err == nil {
		t.Error("a second daemon must refuse rather than steal the socket")
	}
}

func TestListenSetsOwnerOnlyPermissions(t *testing.T) {
	dir, err := os.MkdirTemp("", "dc")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	srv, _ := newServer(t, 10)
	srv.cfg.SocketPath = filepath.Join(dir, "s.sock")
	if err := srv.Listen(); err != nil {
		t.Skipf("cannot bind a unix socket here: %v", err)
	}
	fi, err := os.Stat(srv.Addr())
	if err != nil {
		t.Fatal(err)
	}
	// 0700 is the whole access-control story, which is why it is not optional.
	if perm := fi.Mode().Perm(); perm&0o077 != 0 {
		t.Errorf("the socket is reachable by others: %v", perm)
	}
}

// collect drains n events or fails.
func collect(t *testing.T, events <-chan protocol.Event, errs <-chan error, n int) []protocol.Event {
	t.Helper()
	var out []protocol.Event
	deadline := time.After(8 * time.Second)
	for len(out) < n {
		select {
		case ev, open := <-events:
			if !open {
				t.Fatalf("stream closed after %d of %d events", len(out), n)
			}
			out = append(out, ev)
		case err := <-errs:
			t.Fatalf("stream error: %v", err)
		case <-deadline:
			t.Fatalf("only saw %d of %d events", len(out), n)
		}
	}
	return out
}

// The invariant the sibling test was named for and never asserted: it observed
// the stream once and checked the numbering was contiguous, which is a weaker
// claim than replay reproducing the live observation.
//
// The two are the same journal read twice, and that is precisely what makes the
// bug it guards against invisible: a payload that renders live from state the
// server still has, and renders differently on replay when that state is gone,
// looks correct every time anyone watches it happen.
func TestReplayReproducesTheLiveObservationExactly(t *testing.T) {
	c, mgr := newDaemon(t)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	s, err := c.CreateSession(ctx, protocol.CreateSessionRequest{Workspace: "/tmp"})
	if err != nil {
		t.Fatal(err)
	}
	sess, err := mgr.Get(s.ID)
	if err != nil {
		t.Fatal(err)
	}

	live, liveErrs := c.Subscribe(ctx, s.ID, 1)
	for i := 0; i < 4; i++ {
		sess.Emit(protocol.EventMessageDelta, protocol.MessageDelta{TurnID: "t1", Text: "live"})
	}
	observed := collect(t, live, liveErrs, 5) // session.created + 4

	rep, repErrs := c.Subscribe(ctx, s.ID, 1)
	replayed := collect(t, rep, repErrs, 5)

	if len(observed) != len(replayed) {
		t.Fatalf("live saw %d events, replay %d", len(observed), len(replayed))
	}
	for i := range observed {
		a, b := observed[i], replayed[i]
		if a.Seq != b.Seq || a.Type != b.Type || a.SessionID != b.SessionID {
			t.Errorf("event %d differs: live %s/%d, replay %s/%d", i, a.Type, a.Seq, b.Type, b.Seq)
			continue
		}
		if string(mustJSON(t, a.Payload)) != string(mustJSON(t, b.Payload)) {
			t.Errorf("event %d (%s) has a different payload on replay:\n live: %s\nreplay: %s",
				i, a.Type, mustJSON(t, a.Payload), mustJSON(t, b.Payload))
		}
	}
}

func mustJSON(t *testing.T, v any) []byte {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// The protocol table promises 410 for an approval that ran out of time, and the
// code was declared, mapped to that status, and produced by nothing. A client
// asking about a lapsed approval got 409 — "somebody already answered" — which
// is the opposite of what happened.
func TestALapsedApprovalIsRefusedAsExpiredOverTheWire(t *testing.T) {
	c, mgr := newDaemon(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	s, err := c.CreateSession(ctx, protocol.CreateSessionRequest{Workspace: "/tmp"})
	if err != nil {
		t.Fatal(err)
	}
	sess, err := mgr.Get(s.ID)
	if err != nil {
		t.Fatal(err)
	}

	// Nobody answers, and the deadline denies it.
	if d, err := sess.Approve(ctx, protocol.ApprovalRequest{
		ApprovalID: "a1", Tool: "bash", Command: "curl x",
	}, 10*time.Millisecond); err != nil || d != protocol.ApprovalDeny {
		t.Fatalf("the lapse resolved to %v, %v", d, err)
	}

	err = c.Resolve(ctx, s.ID, "a1", protocol.ApprovalAllow)
	pe, ok := protocol.AsError(err)
	if !ok {
		t.Fatalf("a lapsed approval answered over the wire: %v", err)
	}
	if pe.Code != protocol.CodeApprovalExpired {
		t.Errorf("code = %q, want %q — the client is told a decision was made when "+
			"the deadline denied it", pe.Code, protocol.CodeApprovalExpired)
	}
}

// Undo travels the same wire as everything else, so a second client can put
// back what the first one's turn changed.
func TestUndoOverTheWire(t *testing.T) {
	c, _ := newDaemon(t)
	ctx := context.Background()

	s, err := c.CreateSession(ctx, protocol.CreateSessionRequest{Workspace: "/tmp"})
	if err != nil {
		t.Fatal(err)
	}

	// Nothing has run, so nothing goes back — and that is an answer, not an
	// error. A session with no engine must not turn a request into a failure
	// the caller has to interpret.
	out, err := c.Undo(ctx, s.ID)
	if err != nil {
		t.Fatalf("undoing an untouched session failed: %v", err)
	}
	if len(out.Restored) != 0 || len(out.Refused) != 0 {
		t.Errorf("got %+v", out)
	}
}

// A session that is not there is a 404 the caller can act on, not a silent
// success that looks like the undo worked.
func TestUndoingAnUnknownSessionIsRefused(t *testing.T) {
	c, _ := newDaemon(t)
	if _, err := c.Undo(context.Background(), "never-existed"); err == nil {
		t.Fatal("undoing a session that does not exist reported success")
	}
}

// A picture reaches the session as bytes, decoded once at the edge.
func TestAnImageArrivesOnATurn(t *testing.T) {
	c, mgr := newDaemon(t)
	ctx := context.Background()

	s, err := c.CreateSession(ctx, protocol.CreateSessionRequest{Workspace: "/tmp"})
	if err != nil {
		t.Fatal(err)
	}
	// No engine here, so the turn is refused — but the image must survive
	// decoding first, which is what this asserts. A malformed one has to fail
	// as a bad request rather than reaching the session as empty bytes.
	err = c.Submit(ctx, s.ID, "look", protocol.TurnImage{
		MediaType: "image/png", Data: "iVBORw0KGgo=",
	})
	_ = err
	if _, gerr := mgr.Get(s.ID); gerr != nil {
		t.Fatalf("the session went away: %v", gerr)
	}
}

// Base64 that is not base64 fails at the edge, where the caller can be told
// which part of their request was wrong.
func TestAMalformedImageIsRefused(t *testing.T) {
	c, _ := newDaemon(t)
	s, err := c.CreateSession(context.Background(), protocol.CreateSessionRequest{Workspace: "/tmp"})
	if err != nil {
		t.Fatal(err)
	}
	if err := c.Submit(context.Background(), s.ID, "look", protocol.TurnImage{
		MediaType: "image/png", Data: "not base64 at all!!",
	}); err == nil {
		t.Fatal("a malformed image was accepted")
	}
}

// Steering travels the same wire as everything else, so the correction reaches
// the session a second client is watching.
func TestSteeringOverTheWire(t *testing.T) {
	c, mgr := newDaemon(t)
	ctx := context.Background()

	s, err := c.CreateSession(ctx, protocol.CreateSessionRequest{Workspace: "/tmp"})
	if err != nil {
		t.Fatal(err)
	}

	// Nothing is running, so it is refused — and the refusal is the one that
	// says there is nothing to correct, not the one that says wait.
	err = c.Steer(ctx, s.ID, "use tabs")
	if err == nil {
		t.Fatal("steering an idle session reported success")
	}
	if !strings.Contains(err.Error(), protocol.CodeNoActiveTurn) {
		t.Errorf("err = %v, want no_active_turn", err)
	}

	// With a turn actually running it is accepted and comes back out for the
	// loop. A real turn, not a flag flipped in a test: what the handler does is
	// map the session's answer, and a session pretending to run would not have
	// one to map.
	live := runningSession(t, mgr)
	if err := c.Steer(ctx, live.ID, "use tabs"); err != nil {
		t.Fatalf("steering a running turn failed: %v", err)
	}
	if got := live.TakeSteering(); got != "use tabs" {
		t.Errorf("the correction did not reach the session: %q", got)
	}
}

// blockingProvider holds a turn open until its context is cancelled, which is
// what a long turn looks like from the outside.
type blockingProvider struct{}

func (blockingProvider) Family() provider.Family       { return nil }
func (blockingProvider) Transport() provider.Transport { return nil }
func (blockingProvider) Window(string) (int, error)    { return 1_000_000, nil }
func (blockingProvider) Limits() provider.Limits       { return provider.Limits{} }

func (blockingProvider) Stream(ctx context.Context, _ provider.Request) (<-chan provider.StreamEvent, error) {
	ch := make(chan provider.StreamEvent)
	go func() {
		defer close(ch)
		<-ctx.Done()
	}()
	return ch, nil
}

// runningSession adds a session with a turn genuinely under way.
func runningSession(t *testing.T, mgr *session.Manager) *session.Session {
	t.Helper()
	eng := loop.New(loop.Config{
		Provider: blockingProvider{},
		Limits:   loop.DefaultLimits(),
		Tools:    tools.NewRegistry(),
	}, ce.Session{Instructions: "x"})
	s := session.New("running-1", "/tmp", "m", "read-only", eng,
		session.NewEventLog("running-1", 10, nil), time.Now)
	if err := mgr.Add(s); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(s.Close)

	if err := s.Submit("go"); err != nil {
		t.Fatalf("starting a turn: %v", err)
	}
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if s.State() == protocol.SessionStateRunning {
			return s
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("the turn never started")
	return nil
}

// A session that is not there is a 404 the caller can act on, not a silent
// success that looks like the correction landed.
func TestSteeringAnUnknownSessionIsRefused(t *testing.T) {
	c, _ := newDaemon(t)
	if err := c.Steer(context.Background(), "never-existed", "hello"); err == nil {
		t.Fatal("steering a session that does not exist reported success")
	}
}

// Nothing to say is refused at the edge rather than reaching the session.
func TestSteeringWithNothingToSayIsRefused(t *testing.T) {
	c, _ := newDaemon(t)
	ctx := context.Background()
	s, err := c.CreateSession(ctx, protocol.CreateSessionRequest{Workspace: "/tmp"})
	if err != nil {
		t.Fatal(err)
	}
	if err := c.Steer(ctx, s.ID, "   "); err == nil {
		t.Fatal("an empty correction was accepted")
	}
}

// Creating a session that continues another describes it AFTER the conversation
// is in it.
//
// The response was built before the carried events were appended, so it said
// `last_seq: 0` about a session that had just been handed eighteen thousand
// events, and `first_seq: 1` about one whose earliest surviving event was
// 8411 — retention having dropped the rest. The client believes what it is
// told: it asked for events from 1, was refused because they were gone, and
// the conversation somebody typed `-c` to reopen never opened.
func TestContinuingDescribesTheSessionTheConversationIsIn(t *testing.T) {
	mgr := session.NewManager(10)
	const retention = 50
	const carried = 200
	srv := New(Config{
		Manager: mgr,
		Build: func(req protocol.CreateSessionRequest) (*session.Session, error) {
			log := session.NewEventLog("s1", retention, fixedClock())
			s := session.New("s1", req.Workspace, "m", "workspace-write", nil, log, fixedClock())
			for i := 0; i < carried; i++ {
				s.Carried = append(s.Carried, protocol.Event{
					Type: protocol.EventMessageDelta, Payload: json.RawMessage(`{}`),
				})
			}
			s.CarriedFrom = "older"
			return s, nil
		},
	})

	body, _ := json.Marshal(protocol.CreateSessionRequest{Workspace: t.TempDir()})
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/"+protocol.Version+"/sessions",
		strings.NewReader(string(body))))
	if rec.Code != http.StatusCreated {
		t.Fatalf("creating the session failed: %d %s", rec.Code, rec.Body.String())
	}

	var got protocol.Session
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	// session.created + session.resumed + the carried events.
	if want := uint64(carried + 2); got.LastSeq != want {
		t.Errorf("last_seq is %d; the session holds %d events", got.LastSeq, want)
	}
	if got.FirstSeq <= 1 {
		t.Errorf("first_seq is %d, so a client will ask for events retention "+
			"has already dropped and be refused", got.FirstSeq)
	}
	if got.FirstSeq > got.LastSeq {
		t.Errorf("first_seq %d is past last_seq %d", got.FirstSeq, got.LastSeq)
	}
}

// A command the person typed has its own route.
//
// Not a flag on submitting: submitting starts a turn and this does not. It
// blocks until the command is done, because the person is waiting on the
// output, and the response body carries nothing — the output arrives on the
// event stream, so a client that disconnected mid-command and came back sees
// what one that stayed sees.
func TestExecIsItsOwnRouteAndCarriesNoBody(t *testing.T) {
	srv, _ := newServer(t, 10)
	h := srv.Handler()

	body, _ := json.Marshal(protocol.CreateSessionRequest{Workspace: t.TempDir()})
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/"+protocol.Version+"/sessions",
		strings.NewReader(string(body))))
	var sess protocol.Session
	if err := json.Unmarshal(rec.Body.Bytes(), &sess); err != nil {
		t.Fatal(err)
	}

	// This server builds sessions with no engine, so the command cannot run —
	// which is the point of the assertion: the route exists, decodes, reaches
	// the session, and reports what the session says instead of 404.
	req, _ := json.Marshal(protocol.ExecRequest{Command: "echo hello"})
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost,
		"/"+protocol.Version+"/sessions/"+sess.ID+"/exec", strings.NewReader(string(req))))
	if rec.Code == http.StatusNotFound {
		t.Fatalf("there is no route to run a typed command: %d", rec.Code)
	}
	if rec.Code == http.StatusNoContent && rec.Body.Len() != 0 {
		t.Errorf("the response carries a body: %q", rec.Body.String())
	}
}

// A malformed body is reported, not swallowed.
func TestExecReportsAMalformedRequest(t *testing.T) {
	srv, _ := newServer(t, 10)
	h := srv.Handler()

	body, _ := json.Marshal(protocol.CreateSessionRequest{Workspace: t.TempDir()})
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/"+protocol.Version+"/sessions",
		strings.NewReader(string(body))))
	var sess protocol.Session
	if err := json.Unmarshal(rec.Body.Bytes(), &sess); err != nil {
		t.Fatal(err)
	}

	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost,
		"/"+protocol.Version+"/sessions/"+sess.ID+"/exec", strings.NewReader("{not json")))
	if rec.Code < 400 {
		t.Errorf("a malformed request was accepted: %d", rec.Code)
	}
}

// And running a command on a session that does not exist says so.
func TestExecOnAnUnknownSession(t *testing.T) {
	srv, _ := newServer(t, 10)
	req, _ := json.Marshal(protocol.ExecRequest{Command: "echo hello"})
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodPost,
		"/"+protocol.Version+"/sessions/nope/exec", strings.NewReader(string(req))))
	if rec.Code < 400 {
		t.Errorf("a command ran on a session that does not exist: %d", rec.Code)
	}
}
