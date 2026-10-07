package server

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/aguinelo/dcode/internal/protocol"
)

// The route is asserted on its own here, with the daemon's answer faked: which
// models a session can ask for is resolved by the app, and is tested there
// against real configuration.

func TestListModelsAnswersWhatTheDaemonResolves(t *testing.T) {
	srv, _ := newServer(t, 4)
	var asked []string
	srv.cfg.Models = func(ws string) (protocol.ModelsResponse, error) {
		asked = append(asked, ws)
		return protocol.ModelsResponse{
			Default: protocol.ModelChoice{Name: "MiniMax-M3", Model: "MiniMax-M3",
				Family: "minimax-m3", Transport: "openai", Measured: true},
			Profiles: []protocol.ModelChoice{{Name: "local", Model: "qwen3.5-9b",
				Family: "generic", Transport: "openai", Notice: "nobody measured this one"}},
		}, nil
	}

	rec := get(srv, "/models?workspace=/w")
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}
	var out protocol.ModelsResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if out.Default.Name != "MiniMax-M3" || !out.Default.Measured {
		t.Errorf("the default came back as %+v", out.Default)
	}
	if len(out.Profiles) != 1 || out.Profiles[0].Name != "local" ||
		out.Profiles[0].Measured || out.Profiles[0].Notice == "" {
		t.Errorf("the profiles came back as %+v", out.Profiles)
	}

	// No workspace asks the daemon about the configuration it started with.
	if rec := get(srv, "/models"); rec.Code != http.StatusOK {
		t.Fatalf("status %d without a workspace: %s", rec.Code, rec.Body.String())
	}
	if len(asked) != 2 || asked[0] != "/w" || asked[1] != "" {
		t.Errorf("the daemon was asked about %q, want [\"/w\" \"\"]", asked)
	}
}

// A relative workspace names no project, so it is refused before the daemon is
// asked: resolved against wherever the daemon runs, it would answer for a
// project nobody meant.
func TestListModelsRefusesARelativeWorkspace(t *testing.T) {
	srv, _ := newServer(t, 4)
	asked := false
	srv.cfg.Models = func(string) (protocol.ModelsResponse, error) {
		asked = true
		return protocol.ModelsResponse{}, nil
	}
	rec := get(srv, "/models?workspace=projects/dcode")
	if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), protocol.CodeWorkspaceInvalid) {
		t.Errorf("a relative workspace gave %d %s, want 400 %s",
			rec.Code, rec.Body.String(), protocol.CodeWorkspaceInvalid)
	}
	if asked {
		t.Error("the daemon was asked about a relative workspace")
	}
}

// What the daemon refuses reaches the client with its code and its reason,
// never as a menu.
func TestListModelsCarriesTheDaemonsRefusal(t *testing.T) {
	srv, _ := newServer(t, 4)
	srv.cfg.Models = func(string) (protocol.ModelsResponse, error) {
		return protocol.ModelsResponse{}, protocol.Errorf(protocol.CodeWorkspaceInvalid,
			`the configuration of /w cannot be read: profile "x" has no model`)
	}
	if rec := get(srv, "/models?workspace=/w"); rec.Code != http.StatusBadRequest ||
		!strings.Contains(rec.Body.String(), "has no model") {
		t.Errorf("got %d %s, want 400 with the reason", rec.Code, rec.Body.String())
	}

	srv.cfg.Models = func(string) (protocol.ModelsResponse, error) {
		return protocol.ModelsResponse{}, errors.New("the configuration root cannot be found")
	}
	if rec := get(srv, "/models"); rec.Code != http.StatusInternalServerError ||
		!strings.Contains(rec.Body.String(), "cannot be found") {
		t.Errorf("an unclassified failure gave %d %s", rec.Code, rec.Body.String())
	}
}

// No profiles is an empty list — an array, not null, so a client iterates it
// without a special case. A daemon that cannot answer at all says so instead
// of answering a menu with nothing in it, which would read as "no model can be
// asked for".
func TestAnEmptyMenuIsAnAnswerAndNeverAFailure(t *testing.T) {
	srv, _ := newServer(t, 4)
	if rec := get(srv, "/models"); rec.Code == http.StatusOK ||
		!strings.Contains(rec.Body.String(), "cannot say which models") {
		t.Errorf("a daemon with nothing to ask answered %d %s", rec.Code, rec.Body.String())
	}

	srv.cfg.Models = func(string) (protocol.ModelsResponse, error) {
		return protocol.ModelsResponse{Default: protocol.ModelChoice{Name: "MiniMax-M3"}}, nil
	}
	if rec := get(srv, "/models"); rec.Code != http.StatusOK ||
		!strings.Contains(rec.Body.String(), `"profiles":[]`) {
		t.Errorf("no profiles came back as %d %s, want an empty array", rec.Code, rec.Body.String())
	}
}
