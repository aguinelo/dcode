package client

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"github.com/aguinelo/dcode/internal/protocol"
)

func loopDaemon(t *testing.T, h http.HandlerFunc) *Client {
	t.Helper()
	// Short: macOS caps a Unix socket path near 104 bytes, and t.TempDir()
	// under a long test name passes that on its own.
	dir, err := os.MkdirTemp("", "dc")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })
	ln, err := net.Listen("unix", dir+"/d.sock")
	if err != nil {
		t.Skipf("cannot bind a unix socket here: %v", err)
	}
	srv := &http.Server{Handler: h}
	go srv.Serve(ln)
	t.Cleanup(func() { srv.Close() })
	return New(ln.Addr().String())
}

// A dropped stream is reconnected, and the new connection opens with a new
// snapshot; a refusal the daemon will give again ends the watch, said.
func TestWatchLoopsReconnectsAndEndsOnARefusal(t *testing.T) {
	var calls atomic.Int32
	c := loopDaemon(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/"+protocol.Version+"/loops/events" {
			http.NotFound(w, r)
			return
		}
		switch calls.Add(1) {
		case 1, 2:
			w.Header().Set("Content-Type", "text/event-stream")
			fmt.Fprint(w, ": ping\n\ndata: not json\n\n")
			fmt.Fprintf(w, "data: {\"kind\":%q,\"loops\":[{\"id\":\"L1\",\"state\":\"working\"}]}\n\n", protocol.LoopSnapshot)
		default:
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprint(w, `{"code":"internal","message":"no loops here"}`)
		}
	})
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	changes, errs := c.WatchLoops(ctx)
	for i := 0; i < 2; i++ {
		ch := <-changes
		if ch.Kind != protocol.LoopSnapshot || len(ch.Loops) != 1 || ch.Loops[0].ID != "L1" {
			t.Fatalf("frame %d is %+v", i, ch)
		}
	}
	err := <-errs
	if pe, ok := protocol.AsError(err); !ok || pe.Code != protocol.CodeInternal {
		t.Errorf("the watch ended with %v, want the daemon's refusal", err)
	}
}

// Cancelling the watch ends it without an error: the caller asked.
func TestWatchLoopsEndsWhenCancelled(t *testing.T) {
	c := loopDaemon(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.(http.Flusher).Flush()
		<-r.Context().Done()
	})
	ctx, cancel := context.WithCancel(context.Background())
	changes, errs := c.WatchLoops(ctx)
	time.Sleep(20 * time.Millisecond)
	cancel()
	if _, open := <-changes; open {
		t.Error("a change arrived on a cancelled watch")
	}
	if err, open := <-errs; open {
		t.Errorf("a cancelled watch reported %v", err)
	}
}
