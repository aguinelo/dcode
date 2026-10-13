package client

import (
	"context"
	"net/http"
	"testing"

	"github.com/aguinelo/dcode/internal/protocol"
)

func TestListCommandsAsksForTheWorkspace(t *testing.T) {
	var asked string
	c := serveOnSocket(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/"+protocol.Version+"/commands" {
			http.NotFound(w, r)
			return
		}
		asked = r.URL.Query().Get("workspace")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"commands":[{"name":"review","description":"","source":"project",` +
			`"path":"review.md","body":"Review it."}],"overrides":[]}`))
	}))
	got, err := c.ListCommands(context.Background(), "/a b")
	if err != nil {
		t.Fatal(err)
	}
	if asked != "/a b" {
		t.Errorf("the daemon was asked about %q", asked)
	}
	if len(got.Commands) != 1 || got.Commands[0].Body != "Review it." {
		t.Errorf("the commands came back as %+v", got)
	}
}

// A refusal arrives as the daemon's error, code and reason intact.
func TestListCommandsCarriesTheDaemonsRefusal(t *testing.T) {
	c := serveOnSocket(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"code":"workspace_invalid","message":"a workspace is required"}`))
	}))
	_, err := c.ListCommands(context.Background(), "")
	if pe, ok := protocol.AsError(err); !ok || pe.Code != protocol.CodeWorkspaceInvalid {
		t.Errorf("got %v, want the daemon's workspace_invalid", err)
	}
}
