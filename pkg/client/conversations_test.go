package client

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/aguinelo/dcode/internal/protocol"
)

func serveOnSocket(t *testing.T, h http.Handler) *Client {
	t.Helper()
	srv := &http.Server{Handler: h}
	// Short on purpose: a unix socket path is capped near 104 bytes on macOS,
	// and t.TempDir() carries the test's name.
	dir, err := os.MkdirTemp("", "dc")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })
	ln, err := net.Listen("unix", dir+"/d.sock")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	go srv.Serve(ln)
	t.Cleanup(func() { srv.Close() })
	return New(ln.Addr().String())
}

func TestListConversationsAsksForTheWorkspace(t *testing.T) {
	var query string
	c := serveOnSocket(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		query = r.URL.Query().Get("workspace")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"conversations":[{"id":"a","title":"oi","state":"recorded","live":false,"turns":2,"workspace":"/w a","started":"2026-09-30T12:00:00Z","last_activity":"2026-09-30T12:01:00Z"}]}`))
	}))
	list, err := c.ListConversations(context.Background(), "/w a")
	if err != nil {
		t.Fatal(err)
	}
	if query != "/w a" {
		t.Errorf("the workspace arrived as %q", query)
	}
	if len(list) != 1 || list[0].ID != "a" || list[0].Turns != 2 || list[0].State != protocol.ConversationRecorded {
		t.Errorf("got %+v", list)
	}
}

// A dropped connection is reconnected, and the new one opens with a snapshot
// the client applies like the first: the list has no position to resume from.
func TestWatchConversationsReconnectsToAFreshSnapshot(t *testing.T) {
	var mu sync.Mutex
	connections := 0
	c := serveOnSocket(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		connections++
		n := connections
		mu.Unlock()
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprintf(w, ": ping\n\n")
		fmt.Fprintf(w, "data: {\"kind\":\"snapshot\",\"conversations\":[{\"id\":\"s%d\"}]}\n\n", n)
		fmt.Fprintf(w, "data: not json\n\n")
		if n == 1 {
			fmt.Fprintf(w, "data: {\"kind\":\"changed\",\"conversation\":{\"id\":\"a\"}}\n\n")
		}
		// The handler returns: the connection ends, as a drop would.
	}))
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	changes, _ := c.WatchConversations(ctx)

	want := []string{"snapshot:s1", "changed:a", "snapshot:s2"}
	for _, w := range want {
		select {
		case ch := <-changes:
			got := ch.Kind + ":"
			if ch.Kind == protocol.ConversationSnapshot && len(ch.Conversations) == 1 {
				got += ch.Conversations[0].ID
			} else if ch.Conversation != nil {
				got += ch.Conversation.ID
			}
			if got != w {
				t.Fatalf("got %s, want %s", got, w)
			}
		case <-time.After(5 * time.Second):
			t.Fatalf("waited for %s", w)
		}
	}
	cancel()
	for range changes {
		// Drained until the watch closes it, which is what cancelling promises.
	}
}

// A refusal the daemon will give again ends the watch, said on the error
// channel rather than retried forever.
func TestWatchConversationsEndsOnARefusal(t *testing.T) {
	c := serveOnSocket(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"code":"internal","message":"this daemon keeps no list of conversations"}`))
	}))
	changes, errs := c.WatchConversations(context.Background())
	select {
	case err := <-errs:
		if err == nil || !strings.Contains(err.Error(), "no list") {
			t.Errorf("the refusal came as %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("the watch kept retrying a refusal")
	}
	if _, open := <-changes; open {
		t.Error("the watch sent changes after it was refused")
	}
}
