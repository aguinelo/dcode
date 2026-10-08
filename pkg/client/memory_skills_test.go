package client

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/aguinelo/dcode/internal/protocol"
)

func TestReadMemoryAsksForTheWorkspace(t *testing.T) {
	var asked string
	c := serveOnSocket(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/"+protocol.Version+"/memory" {
			http.NotFound(w, r)
			return
		}
		asked = r.URL.Query().Get("workspace")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"path":".dcode/memory.md","exists":true,"enabled":true,"max_entries":40,` +
			`"entries":[{"kind":"gotcha","subject":"go generate first","stale":false,"shown":true}],` +
			`"malformed":[{"line":"## diary: x","reason":"not a kind"}]}`))
	}))
	got, err := c.ReadMemory(context.Background(), "/w a")
	if err != nil {
		t.Fatal(err)
	}
	if asked != "/w a" {
		t.Errorf("the daemon was asked about %q", asked)
	}
	if !got.Exists || got.MaxEntries != 40 || len(got.Entries) != 1 || !got.Entries[0].Shown ||
		len(got.Malformed) != 1 || got.Malformed[0].Reason != "not a kind" {
		t.Errorf("the memory came back as %+v", got)
	}
}

func TestListSkillsAsksForTheWorkspace(t *testing.T) {
	var asked string
	c := serveOnSocket(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/"+protocol.Version+"/skills" {
			http.NotFound(w, r)
			return
		}
		asked = r.URL.Query().Get("workspace")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"enabled":true,"skills":[{"name":"release","when_to_use":"cutting a release",` +
			`"source":"project","path":"release.md","held":false}],"notices":[]}`))
	}))
	got, err := c.ListSkills(context.Background(), "/w")
	if err != nil {
		t.Fatal(err)
	}
	if asked != "/w" {
		t.Errorf("the daemon was asked about %q", asked)
	}
	if !got.Enabled || len(got.Skills) != 1 || got.Skills[0].Source != protocol.SkillSourceProject {
		t.Errorf("the skills came back as %+v", got)
	}
}

// A refusal arrives as the daemon's error, code and reason intact.
func TestMemoryAndSkillsCarryTheDaemonsRefusal(t *testing.T) {
	c := serveOnSocket(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"code":"workspace_invalid","message":"the workspace \"w\" must be an absolute path"}`))
	}))
	_, merr := c.ReadMemory(context.Background(), "w")
	_, serr := c.ListSkills(context.Background(), "w")
	for _, err := range []error{merr, serr} {
		pe, ok := protocol.AsError(err)
		if !ok || pe.Code != protocol.CodeWorkspaceInvalid || !strings.Contains(pe.Message, "absolute") {
			t.Errorf("the refusal came back as %v", err)
		}
	}
}
