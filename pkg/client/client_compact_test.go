package client

import (
	"context"
	"net"
	"net/http"
	"strings"
	"testing"
)

// TestCompact exercises the Compact transport method end to end: a request is
// sent over a real Unix socket, the path is what routes the request, the
// response body is decoded into the bool the caller asked for, and a 4xx
// surfaces as an error.
func TestCompact(t *testing.T) {
	var gotMethod, gotPath string

	srv := &http.Server{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			gotMethod = r.Method
			gotPath = r.URL.Path
			if strings.HasSuffix(r.URL.Path, "/sessions/busy/compact") {
				w.WriteHeader(http.StatusConflict)
				_, _ = w.Write([]byte(`{"code":"turn_active","message":"a turn is running"}`))
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"compacted":true}`))
		}),
	}
	ln, err := net.Listen("unix", t.TempDir()+"/dcode.sock")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer ln.Close()
	go srv.Serve(ln)
	defer srv.Close()

	c := New(ln.Addr().String())

	compacted, err := c.Compact(context.Background(), "s-1")
	if err != nil {
		t.Fatalf("Compact: %v", err)
	}
	if !compacted {
		t.Error("got compacted = false, want true")
	}
	if gotMethod != "POST" {
		t.Errorf("method = %q, want POST", gotMethod)
	}
	if !strings.HasSuffix(gotPath, "/sessions/s-1/compact") {
		t.Errorf("path = %q, want suffix /sessions/s-1/compact", gotPath)
	}

	if _, err := c.Compact(context.Background(), "busy"); err == nil {
		t.Fatal("expected error on 409, got nil")
	}
}
