package client

import (
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"strings"
	"testing"
)

// TestSetMode exercises the SetMode transport method end to end: a request is
// sent over a real Unix socket, the body is what the server reads, the path is
// what routes the request, and a 4xx surfaces as an error.
func TestSetMode(t *testing.T) {
	var gotMethod, gotPath string
	var gotBody struct {
		Mode string `json:"mode"`
	}

	srv := &http.Server{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			gotMethod = r.Method
			gotPath = r.URL.Path
			raw, _ := io.ReadAll(r.Body)
			_ = json.Unmarshal(raw, &gotBody)
			if gotBody.Mode == "auto" {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"code":"bad","message":"nope"}`))
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

	if err := c.SetMode(context.Background(), "s-1", "auto"); err != nil {
		t.Fatalf("SetMode(auto): %v", err)
	}
	if gotMethod != "POST" {
		t.Errorf("method = %q, want POST", gotMethod)
	}
	if !strings.HasSuffix(gotPath, "/sessions/s-1/mode") {
		t.Errorf("path = %q, want suffix /sessions/s-1/mode", gotPath)
	}
	if gotBody.Mode != "auto" {
		t.Errorf("body.Mode = %q, want auto", gotBody.Mode)
	}

	if err := c.SetMode(context.Background(), "s-1", "weird"); err == nil {
		t.Fatal("expected error on 400, got nil")
	}
}
