package server

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/aguinelo/dcode/internal/protocol"
)

// The routes are asserted on their own here, with the daemon's answer faked:
// what a workspace remembers and which skills it has are resolved by the app,
// and are tested there against real files.

func TestReadMemoryAnswersWhatTheDaemonResolves(t *testing.T) {
	srv, _ := newServer(t, 4)
	var asked []string
	srv.cfg.Memory = func(ws string) (protocol.MemoryResponse, error) {
		asked = append(asked, ws)
		return protocol.MemoryResponse{Path: ".dcode/memory.md", Exists: true, Enabled: true,
			Entries: []protocol.MemoryEntry{{Kind: "gotcha", Subject: "go generate first", Shown: true}}}, nil
	}
	rec := get(srv, "/memory?workspace=/w")
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}
	var out protocol.MemoryResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if !out.Exists || len(out.Entries) != 1 || out.Entries[0].Subject != "go generate first" {
		t.Errorf("the memory came back as %+v", out)
	}
	if len(asked) != 1 || asked[0] != "/w" {
		t.Errorf("the daemon was asked about %q, want [\"/w\"]", asked)
	}
	if !strings.Contains(rec.Body.String(), `"malformed":[]`) {
		t.Errorf("no malformed block came back as something other than an empty array: %s", rec.Body.String())
	}
}

func TestListSkillsAnswersWhatTheDaemonResolves(t *testing.T) {
	srv, _ := newServer(t, 4)
	var asked []string
	srv.cfg.Skills = func(ws string) (protocol.SkillsResponse, error) {
		asked = append(asked, ws)
		return protocol.SkillsResponse{Enabled: true, Skills: []protocol.SkillInfo{{
			Name: "release", WhenToUse: "cutting a release", Source: protocol.SkillSourceProject,
			Path: "release.md"}}}, nil
	}
	rec := get(srv, "/skills?workspace=/w")
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}
	var out protocol.SkillsResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if len(out.Skills) != 1 || out.Skills[0].Name != "release" || out.Skills[0].Source != "project" {
		t.Errorf("the skills came back as %+v", out)
	}
	if len(asked) != 1 || asked[0] != "/w" {
		t.Errorf("the daemon was asked about %q, want [\"/w\"]", asked)
	}
	if !strings.Contains(rec.Body.String(), `"notices":[]`) {
		t.Errorf("no notice came back as something other than an empty array: %s", rec.Body.String())
	}
}

// A workspace is what both routes are about, so none is refused, and a relative
// one names no project: both are refused before the daemon is asked.
func TestMemoryAndSkillsRefuseAMissingOrRelativeWorkspace(t *testing.T) {
	srv, _ := newServer(t, 4)
	asked := false
	srv.cfg.Memory = func(string) (protocol.MemoryResponse, error) { asked = true; return protocol.MemoryResponse{}, nil }
	srv.cfg.Skills = func(string) (protocol.SkillsResponse, error) { asked = true; return protocol.SkillsResponse{}, nil }
	for _, path := range []string{"/memory", "/memory?workspace=projects/dcode", "/skills", "/skills?workspace=projects/dcode"} {
		rec := get(srv, path)
		if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), protocol.CodeWorkspaceInvalid) {
			t.Errorf("%s gave %d %s, want 400 %s", path, rec.Code, rec.Body.String(), protocol.CodeWorkspaceInvalid)
		}
	}
	if asked {
		t.Error("the daemon was asked about a workspace that names no project")
	}
}

// What the daemon refuses reaches the client with its code and its reason.
func TestMemoryAndSkillsCarryTheDaemonsRefusal(t *testing.T) {
	srv, _ := newServer(t, 4)
	refusal := protocol.Errorf(protocol.CodeWorkspaceInvalid, "the configuration of /w cannot be read: nmae")
	srv.cfg.Memory = func(string) (protocol.MemoryResponse, error) { return protocol.MemoryResponse{}, refusal }
	srv.cfg.Skills = func(string) (protocol.SkillsResponse, error) {
		return protocol.SkillsResponse{}, errors.New("the skills directory cannot be read")
	}
	if rec := get(srv, "/memory?workspace=/w"); rec.Code != http.StatusBadRequest ||
		!strings.Contains(rec.Body.String(), "nmae") {
		t.Errorf("got %d %s, want 400 with the reason", rec.Code, rec.Body.String())
	}
	if rec := get(srv, "/skills?workspace=/w"); rec.Code != http.StatusInternalServerError ||
		!strings.Contains(rec.Body.String(), "cannot be read") {
		t.Errorf("an unclassified failure gave %d %s", rec.Code, rec.Body.String())
	}
}

// A daemon that cannot answer says so instead of answering empty, which would
// read as "nothing was learned here" or "no skill is installed".
func TestMemoryAndSkillsWithoutTheHookRefuse(t *testing.T) {
	srv, _ := newServer(t, 4)
	for path, why := range map[string]string{
		"/memory?workspace=/w": "cannot say what a workspace remembers",
		"/skills?workspace=/w": "cannot say which skills a session has",
	} {
		if rec := get(srv, path); rec.Code == http.StatusOK || !strings.Contains(rec.Body.String(), why) {
			t.Errorf("%s on a daemon with nothing to ask answered %d %s", path, rec.Code, rec.Body.String())
		}
	}
}
