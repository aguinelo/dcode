package server

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/aguinelo/dcode/internal/protocol"
)

// The route on its own, with the daemon's answer faked: which commands a
// workspace has is resolved by the app, and tested there against real files.

func TestListCommandsAnswersWhatTheDaemonResolves(t *testing.T) {
	srv, _ := newServer(t, 4)
	var asked []string
	srv.cfg.Commands = func(ws string) (protocol.CommandsResponse, error) {
		asked = append(asked, ws)
		return protocol.CommandsResponse{Commands: []protocol.CommandInfo{{
			Name: "review", Description: "review the staged change", Source: protocol.CommandSourceProject,
			Path: "review.md", Body: "Review the staged change."}}}, nil
	}
	rec := get(srv, "/commands?workspace=/w")
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}
	var out protocol.CommandsResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if len(out.Commands) != 1 || out.Commands[0].Name != "review" || out.Commands[0].Body != "Review the staged change." {
		t.Errorf("the commands came back as %+v", out)
	}
	if len(asked) != 1 || asked[0] != "/w" {
		t.Errorf("the daemon was asked about %q, want [\"/w\"]", asked)
	}
	if !strings.Contains(rec.Body.String(), `"overrides":[]`) {
		t.Errorf("no override came back as something other than an empty array: %s", rec.Body.String())
	}
}

// No command at all is an empty list, not null: a client iterates it without
// a special case.
func TestNoCommandIsAnEmptyList(t *testing.T) {
	srv, _ := newServer(t, 4)
	srv.cfg.Commands = func(string) (protocol.CommandsResponse, error) { return protocol.CommandsResponse{}, nil }
	rec := get(srv, "/commands?workspace=/w")
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"commands":[]`) {
		t.Errorf("got %d %s, want 200 with an empty list", rec.Code, rec.Body.String())
	}
}

// A workspace is what the route is about: none, or a relative one, is refused
// before the daemon is asked.
func TestCommandsRefuseAMissingOrRelativeWorkspace(t *testing.T) {
	srv, _ := newServer(t, 4)
	asked := false
	srv.cfg.Commands = func(string) (protocol.CommandsResponse, error) { asked = true; return protocol.CommandsResponse{}, nil }
	for _, path := range []string{"/commands", "/commands?workspace=projects/dcode"} {
		rec := get(srv, path)
		if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), protocol.CodeWorkspaceInvalid) {
			t.Errorf("%s: got %d %s, want 400 %s", path, rec.Code, rec.Body.String(), protocol.CodeWorkspaceInvalid)
		}
	}
	if asked {
		t.Error("the daemon was asked about a workspace the route should have refused")
	}
}

// A daemon that cannot say which commands a workspace has refuses, saying so:
// an empty list would read as "none was written", which is not what happened.
func TestCommandsWithoutTheHookRefuse(t *testing.T) {
	srv, _ := newServer(t, 4)
	rec := get(srv, "/commands?workspace=/w")
	if rec.Code == http.StatusOK || !strings.Contains(rec.Body.String(), "cannot say which commands") {
		t.Errorf("got %d %s, want a refusal saying why", rec.Code, rec.Body.String())
	}
}
