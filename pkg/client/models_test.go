package client

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/aguinelo/dcode/internal/protocol"
)

func TestListModelsAsksForTheWorkspace(t *testing.T) {
	var asked []string
	c := serveOnSocket(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/"+protocol.Version+"/models" {
			http.NotFound(w, r)
			return
		}
		q := r.URL.Query()
		if q.Has("workspace") {
			asked = append(asked, q.Get("workspace"))
		} else {
			asked = append(asked, "(none)")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"default":{"name":"MiniMax-M3","model":"MiniMax-M3","family":"minimax-m3",` +
			`"transport":"openai","window":1000000,"measured":true},"profiles":[{"name":"local",` +
			`"model":"qwen3.5-9b","family":"generic","transport":"openai","base_url":"http://127.0.0.1:1234/v1",` +
			`"window":32000,"measured":false,"notice":"nobody measured this one"}]}`))
	}))

	got, err := c.ListModels(context.Background(), "/w a")
	if err != nil {
		t.Fatal(err)
	}
	if got.Default.Name != "MiniMax-M3" || !got.Default.Measured || got.Default.Window != 1000000 {
		t.Errorf("the default came back as %+v", got.Default)
	}
	if len(got.Profiles) != 1 || got.Profiles[0].Name != "local" || got.Profiles[0].Measured ||
		got.Profiles[0].BaseURL != "http://127.0.0.1:1234/v1" || got.Profiles[0].Notice == "" {
		t.Errorf("the profiles came back as %+v", got.Profiles)
	}

	// No workspace sends none, and the daemon answers for its own configuration.
	if _, err := c.ListModels(context.Background(), ""); err != nil {
		t.Fatal(err)
	}
	if len(asked) != 2 || asked[0] != "/w a" || asked[1] != "(none)" {
		t.Errorf("the daemon was asked about %q, want [\"/w a\" \"(none)\"]", asked)
	}
}

// A refusal arrives as the daemon's error, code and reason intact, so a client
// can tell a workspace it should fix from a daemon that failed.
func TestListModelsCarriesTheDaemonsRefusal(t *testing.T) {
	c := serveOnSocket(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"code":"workspace_invalid","message":"the configuration of /w cannot be read: profile \"x\" has no model"}`))
	}))
	got, err := c.ListModels(context.Background(), "/w")
	pe, ok := protocol.AsError(err)
	if !ok || pe.Code != protocol.CodeWorkspaceInvalid || !strings.Contains(pe.Message, "has no model") {
		t.Fatalf("the refusal came back as %v", err)
	}
	if got.Default.Name != "" || got.Profiles != nil {
		t.Errorf("a refusal came back with a menu: %+v", got)
	}
}
