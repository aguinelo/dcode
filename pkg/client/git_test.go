package client

import (
	"context"
	"encoding/json"
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

func TestReadGitPRAsksForTheWorkspace(t *testing.T) {
	var asked string
	c := serveOnSocket(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/"+protocol.Version+"/git/pr" {
			http.NotFound(w, r)
			return
		}
		asked = r.URL.Query().Get("workspace")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"branch":"feature","base":"main","pr":{"number":7,"url":"u","title":"t","state":"open",` +
			`"draft":false,"base":"main","head":"feature","checks":{"state":"pass","passed":3,"failed":0,"pending":0,"skipped":0}}}`))
	}))
	got, err := c.ReadGitPR(context.Background(), "/w a")
	if err != nil {
		t.Fatal(err)
	}
	if asked != "/w a" || got.PR == nil || got.PR.Number != 7 || got.PR.Checks.Passed != 3 {
		t.Errorf("asked %q, got %+v", asked, got)
	}
}

func TestCreateGitPRPostsWhatWasConfirmed(t *testing.T) {
	var sent protocol.CreateGitPRRequest
	c := serveOnSocket(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/"+protocol.Version+"/git/pr" {
			http.NotFound(w, r)
			return
		}
		_ = json.NewDecoder(r.Body).Decode(&sent)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"branch":"feature","base":"main","pr":{"number":8,"url":"u","title":"t","state":"open",` +
			`"draft":true,"base":"main","head":"feature","checks":{"state":"none","passed":0,"failed":0,"pending":0,"skipped":0}}}`))
	}))
	req := protocol.CreateGitPRRequest{Workspace: "/w", Title: "t", Body: "b", Base: "main", Draft: true}
	got, err := c.CreateGitPR(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if sent != req || got.PR == nil || got.PR.Number != 8 {
		t.Errorf("sent %+v, got %+v", sent, got)
	}
}
