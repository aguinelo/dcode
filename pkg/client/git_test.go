package client

import (
	"context"
	"net/http"
	"testing"

	"github.com/aguinelo/dcode/internal/protocol"
)

func TestReadGitAsksForTheWorkspace(t *testing.T) {
	var asked string
	c := serveOnSocket(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/"+protocol.Version+"/git" {
			http.NotFound(w, r)
			return
		}
		asked = r.URL.Query().Get("workspace")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"repository":true,"branch":"main","detached":false,"worktree":false,` +
			`"files":[{"path":"a.txt","status":"modified","added":2,"removed":1,"binary":false}],"truncated":false}`))
	}))
	got, err := c.ReadGit(context.Background(), "/w a")
	if err != nil {
		t.Fatal(err)
	}
	if asked != "/w a" {
		t.Errorf("the daemon was asked about %q", asked)
	}
	if !got.Repository || got.Branch != "main" || len(got.Files) != 1 || got.Files[0].Added != 2 {
		t.Errorf("the tree came back as %+v", got)
	}
}

func TestReadGitDiffAsksForTheFile(t *testing.T) {
	var ws, path string
	c := serveOnSocket(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/"+protocol.Version+"/git/diff" {
			http.NotFound(w, r)
			return
		}
		ws, path = r.URL.Query().Get("workspace"), r.URL.Query().Get("path")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"path":"a b.txt","status":"modified","diff":"+x\n","binary":false,"truncated":false,"limit":262144}`))
	}))
	got, err := c.ReadGitDiff(context.Background(), "/w", "a b.txt")
	if err != nil {
		t.Fatal(err)
	}
	if ws != "/w" || path != "a b.txt" {
		t.Errorf("the daemon was asked about %q %q", ws, path)
	}
	if got.Diff != "+x\n" || got.Limit != 262144 {
		t.Errorf("the diff came back as %+v", got)
	}
}
