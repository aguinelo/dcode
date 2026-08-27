package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/aguinelo/dcode/internal/protocol"
)

// serveOn starts a daemon-shaped server on a Unix socket and returns a client
// pointed at it. A real socket rather than httptest, because the socket is the
// transport the protocol actually specifies.
func serveOn(t *testing.T, h http.Handler) *Client {
	t.Helper()
	// os.MkdirTemp rather than t.TempDir: a Unix socket path is capped near 104
	// bytes on macOS, and the test name in a TempDir path can exhaust it.
	dir, err := os.MkdirTemp("", "dc")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })

	path := filepath.Join(dir, "d.sock")
	ln, err := net.Listen("unix", path)
	if err != nil {
		t.Fatal(err)
	}
	srv := &http.Server{Handler: h}
	go srv.Serve(ln)
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		srv.Shutdown(ctx)
	})
	return New(path)
}

func jsonHandler(t *testing.T, status int, body any) http.HandlerFunc {
	t.Helper()
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		if body != nil {
			json.NewEncoder(w).Encode(body)
		}
	}
}

func TestHealth(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	if err := serveOn(t, mux).Health(context.Background()); err != nil {
		t.Fatal(err)
	}

	down := serveOn(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "no", http.StatusServiceUnavailable)
	}))
	if err := down.Health(context.Background()); err == nil {
		t.Error("an unhealthy daemon must be reported")
	}

	// Nothing listening at all is the ordinary case before `dcode serve`.
	if err := New(filepath.Join(t.TempDir(), "absent.sock")).Health(context.Background()); err == nil {
		t.Error("a missing daemon must be reported")
	}
}

func TestTheRequestSurfaceMatchesTheProtocol(t *testing.T) {
	var (
		mu     sync.Mutex
		seen   []string
		bodies []string
	)
	record := func(w http.ResponseWriter, r *http.Request, reply any) {
		mu.Lock()
		seen = append(seen, r.Method+" "+r.URL.Path)
		buf := make([]byte, 4096)
		n, _ := r.Body.Read(buf)
		bodies = append(bodies, string(buf[:n]))
		mu.Unlock()
		if reply != nil {
			json.NewEncoder(w).Encode(reply)
		}
	}

	p := "/" + protocol.Version
	mux := http.NewServeMux()
	mux.HandleFunc("POST "+p+"/sessions", func(w http.ResponseWriter, r *http.Request) {
		record(w, r, protocol.Session{ID: "s1", Workspace: "/w"})
	})
	mux.HandleFunc("GET "+p+"/sessions", func(w http.ResponseWriter, r *http.Request) {
		record(w, r, []protocol.Session{{ID: "s1"}})
	})
	mux.HandleFunc("GET "+p+"/sessions/s1", func(w http.ResponseWriter, r *http.Request) {
		record(w, r, protocol.Session{ID: "s1"})
	})
	mux.HandleFunc("DELETE "+p+"/sessions/s1", func(w http.ResponseWriter, r *http.Request) {
		record(w, r, nil)
	})
	mux.HandleFunc("POST "+p+"/sessions/s1/turns", func(w http.ResponseWriter, r *http.Request) {
		record(w, r, nil)
	})
	mux.HandleFunc("POST "+p+"/sessions/s1/interrupt", func(w http.ResponseWriter, r *http.Request) {
		record(w, r, nil)
	})
	mux.HandleFunc("POST "+p+"/sessions/s1/approvals/a1", func(w http.ResponseWriter, r *http.Request) {
		record(w, r, nil)
	})

	c := serveOn(t, mux)
	ctx := context.Background()

	sess, err := c.CreateSession(ctx, protocol.CreateSessionRequest{Workspace: "/w", Model: "m"})
	if err != nil || sess.ID != "s1" {
		t.Fatalf("got %+v, %v", sess, err)
	}
	if list, err := c.ListSessions(ctx); err != nil || len(list) != 1 {
		t.Fatalf("got %v, %v", list, err)
	}
	if got, err := c.GetSession(ctx, "s1"); err != nil || got.ID != "s1" {
		t.Fatalf("got %+v, %v", got, err)
	}
	if err := c.Submit(ctx, "s1", "hello"); err != nil {
		t.Fatal(err)
	}
	if err := c.Interrupt(ctx, "s1"); err != nil {
		t.Fatal(err)
	}
	if err := c.Resolve(ctx, "s1", "a1", protocol.ApprovalDeny); err != nil {
		t.Fatal(err)
	}
	if err := c.DeleteSession(ctx, "s1"); err != nil {
		t.Fatal(err)
	}

	mu.Lock()
	defer mu.Unlock()
	want := []string{
		"POST " + p + "/sessions",
		"GET " + p + "/sessions",
		"GET " + p + "/sessions/s1",
		"POST " + p + "/sessions/s1/turns",
		"POST " + p + "/sessions/s1/interrupt",
		"POST " + p + "/sessions/s1/approvals/a1",
		"DELETE " + p + "/sessions/s1",
	}
	if len(seen) != len(want) {
		t.Fatalf("got %v", seen)
	}
	for i := range want {
		if seen[i] != want[i] {
			t.Errorf("call %d: got %q want %q", i, seen[i], want[i])
		}
	}
	if bodies[3] == "" || bodies[5] == "" {
		t.Errorf("a turn and an approval both carry a body: %q %q", bodies[3], bodies[5])
	}
}

// Callers branch on the code, not on a status number or a message: that is what
// makes the protocol usable from a client that was written against a different
// version of the daemon.
func TestErrorsCarryTheProtocolCode(t *testing.T) {
	c := serveOn(t, jsonHandler(t, http.StatusNotFound,
		protocol.Error{Code: protocol.CodeSessionNotFound, Message: "gone"}))

	err := c.Submit(context.Background(), "s1", "hi")
	pe, ok := protocol.AsError(err)
	if !ok || pe.Code != protocol.CodeSessionNotFound {
		t.Fatalf("got %v", err)
	}
}

// A daemon that answered with something unreadable still has to produce a
// usable error rather than a decode failure.
func TestAnUnreadableErrorBodyBecomesAnInternalError(t *testing.T) {
	c := serveOn(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		w.Write([]byte("<html>proxy error</html>"))
	}))
	err := c.Submit(context.Background(), "s1", "hi")
	pe, ok := protocol.AsError(err)
	if !ok || pe.Code != protocol.CodeInternal {
		t.Fatalf("got %v", err)
	}
}

func TestARequestToANonexistentDaemonFails(t *testing.T) {
	c := New(filepath.Join(t.TempDir(), "absent.sock"))
	if err := c.Submit(context.Background(), "s1", "hi"); err == nil {
		t.Error("an unreachable daemon must be reported")
	}
	if _, err := c.ListSessions(context.Background()); err == nil {
		t.Error("an unreachable daemon must be reported")
	}
}

// ---------- the event stream ----------

func sse(w http.ResponseWriter, evs []protocol.Event) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, ": ping\n\n")
	for _, ev := range evs {
		raw, _ := json.Marshal(ev)
		fmt.Fprintf(w, "id: %d\ndata: %s\n\n", ev.Seq, raw)
	}
	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}
}

func TestSubscribeDeliversEventsAndSkipsTheFraming(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /"+protocol.Version+"/sessions/s1/events",
		func(w http.ResponseWriter, r *http.Request) {
			sse(w, []protocol.Event{
				{Seq: 1, Type: protocol.EventTurnStarted},
				{Seq: 2, Type: protocol.EventTurnCompleted},
			})
		})
	c := serveOn(t, mux)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	events, _ := c.Subscribe(ctx, "s1", 1)

	for _, want := range []uint64{1, 2} {
		select {
		case ev := <-events:
			if ev.Seq != want {
				t.Fatalf("got seq %d want %d", ev.Seq, want)
			}
		case <-ctx.Done():
			t.Fatal("timed out waiting for an event")
		}
	}
}

// The read position is the only state a client keeps, which is what lets it die
// and rejoin without the session noticing.
func TestSubscribeResumesAfterTheLastSequenceSeen(t *testing.T) {
	var (
		mu    sync.Mutex
		froms []uint64
	)
	mux := http.NewServeMux()
	mux.HandleFunc("GET /"+protocol.Version+"/sessions/s1/events",
		func(w http.ResponseWriter, r *http.Request) {
			from, _ := strconv.ParseUint(r.URL.Query().Get("from"), 10, 64)
			mu.Lock()
			froms = append(froms, from)
			round := len(froms)
			mu.Unlock()

			if round == 1 {
				sse(w, []protocol.Event{{Seq: 1}, {Seq: 2}})
				return
			}
			sse(w, []protocol.Event{{Seq: uint64(round + 1)}})
		})
	c := serveOn(t, mux)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	events, _ := c.Subscribe(ctx, "s1", 1)

	// Three events across at least two connections: the second must resume at
	// 3, not restart at 1.
	for i := 0; i < 3; i++ {
		select {
		case <-events:
		case <-ctx.Done():
			t.Fatal("timed out")
		}
	}
	mu.Lock()
	defer mu.Unlock()
	if len(froms) < 2 {
		t.Fatalf("expected a reconnect, got %v", froms)
	}
	if froms[0] != 1 || froms[1] != 3 {
		t.Errorf("reconnect must resume at the next sequence, got %v", froms)
	}
}

// Retrying cannot recover a session that no longer exists, or history that has
// been dropped, so the client stops and says so.
func TestSubscribeGivesUpOnAnUnrecoverableError(t *testing.T) {
	for _, code := range []string{protocol.CodeSessionNotFound, protocol.CodeEventsExpired} {
		mux := http.NewServeMux()
		mux.HandleFunc("GET /"+protocol.Version+"/sessions/s1/events",
			func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusNotFound)
				json.NewEncoder(w).Encode(protocol.Error{Code: code, Message: "gone"})
			})
		c := serveOn(t, mux)

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		_, errs := c.Subscribe(ctx, "s1", 1)
		select {
		case err := <-errs:
			pe, ok := protocol.AsError(err)
			if !ok || pe.Code != code {
				t.Errorf("got %v", err)
			}
		case <-ctx.Done():
			t.Errorf("%s: the client must give up rather than retry forever", code)
		}
		cancel()
	}
}

// A recoverable failure is retried, and the channels close only when the caller
// cancels.
func TestSubscribeRetriesAndClosesOnCancel(t *testing.T) {
	var attempts int
	var mu sync.Mutex
	mux := http.NewServeMux()
	mux.HandleFunc("GET /"+protocol.Version+"/sessions/s1/events",
		func(w http.ResponseWriter, r *http.Request) {
			mu.Lock()
			attempts++
			mu.Unlock()
			http.Error(w, "later", http.StatusServiceUnavailable)
		})
	c := serveOn(t, mux)

	ctx, cancel := context.WithCancel(context.Background())
	events, errs := c.Subscribe(ctx, "s1", 1)

	deadline := time.After(3 * time.Second)
	for {
		mu.Lock()
		n := attempts
		mu.Unlock()
		if n >= 2 {
			break
		}
		select {
		case <-deadline:
			t.Fatalf("expected a retry, got %d attempts", n)
		case <-time.After(20 * time.Millisecond):
		}
	}

	cancel()
	for range events {
	}
	for range errs {
	}
}

func TestSubscribeIgnoresUndecodableData(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /"+protocol.Version+"/sessions/s1/events",
		func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/event-stream")
			fmt.Fprint(w, "data: {not json}\n\n")
			raw, _ := json.Marshal(protocol.Event{Seq: 7, Type: protocol.EventTurnStarted})
			fmt.Fprintf(w, "data: %s\n\n", raw)
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}
		})
	c := serveOn(t, mux)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	events, _ := c.Subscribe(ctx, "s1", 1)
	select {
	case ev := <-events:
		if ev.Seq != 7 {
			t.Errorf("garbage must be skipped, not delivered: %+v", ev)
		}
	case <-ctx.Done():
		t.Fatal("timed out")
	}
}

// The two calls the loop makes on its own: what specs are there, and write
// what was proposed.
func TestListSpecsAndCommitDone(t *testing.T) {
	var seen []string
	c := serveOn(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = append(seen, r.Method+" "+r.URL.Path+"?"+r.URL.RawQuery)
		w.Header().Set("content-type", "application/json")
		switch {
		case r.URL.Path == "/"+protocol.Version+"/specs":
			json.NewEncoder(w).Encode(protocol.ListSpecsResponse{Specs: []protocol.SpecFolder{
				{Path: "specs/a", Criteria: 2, Unmet: 1, Pending: true},
			}})
		default:
			json.NewEncoder(w).Encode(protocol.CommitDoneResponse{
				Path: "specs/a/done.toml", Criteria: 2, Summary: "two of them",
			})
		}
	}))

	specs, err := c.ListSpecs(context.Background(), "/w")
	if err != nil {
		t.Fatal(err)
	}
	if len(specs) != 1 || !specs[0].Pending {
		t.Fatalf("got %+v", specs)
	}

	out, err := c.CommitDone(context.Background(), "s 1")
	if err != nil {
		t.Fatal(err)
	}
	if out.Criteria != 2 || out.Path == "" {
		t.Fatalf("got %+v", out)
	}

	// The workspace travels as a query value and the id as a path segment,
	// both escaped: a workspace with a space in it is ordinary on a Mac.
	if len(seen) != 2 || !strings.Contains(seen[0], "workspace=%2Fw") {
		t.Errorf("requests were %v", seen)
	}
	// The id arrives intact. It was escaped on the wire and decoded here,
	// which is the round trip that matters: an id with a space in it must
	// reach the handler as the same id, not as two path segments.
	if !strings.Contains(seen[1], "/sessions/s 1/done") {
		t.Errorf("the session id did not survive the round trip: %v", seen)
	}
}
