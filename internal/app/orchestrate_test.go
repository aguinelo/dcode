package app

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/aguinelo/dcode/internal/protocol"
	"github.com/aguinelo/dcode/internal/session"
)

// projectRepo is a repository with one commit on main, its root with links
// resolved.
func projectRepo(t *testing.T) string {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("no git here")
	}
	ws, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	gitIn(t, ws, "init", "-q", "-b", "main")
	gitIn(t, ws, "config", "user.email", "t@example.com")
	gitIn(t, ws, "config", "user.name", "T")
	gitIn(t, ws, "config", "commit.gpgsign", "false")
	writeIn(t, ws, "a.txt", "one\n")
	gitIn(t, ws, "add", "a.txt")
	gitIn(t, ws, "commit", "-qm", "init")
	return ws
}

func orchestratingDaemon(t *testing.T, records, orch string) *Daemon {
	t.Helper()
	d := NewDaemon(DaemonOptions{
		SocketPath:     filepath.Join(t.TempDir(), "d.sock"),
		EventRetention: 10000,
		RecordDir:      records,
		Base:           baseOpts(t),
		Orchestrator:   OrchestratorOptions{Enabled: true, Dir: orch},
		Log:            func(string) {},
	})
	t.Cleanup(d.manager.CloseAll)
	// The daemon's other tests skip where no session can be built at all — no
	// sandbox on this machine — and so does this one, before it measures.
	probe, err := d.build(protocol.CreateSessionRequest{Workspace: t.TempDir()})
	if err != nil {
		t.Skipf("a session cannot be built here: %v", err)
	}
	probe.Close()
	return d
}

func route(t *testing.T, d *Daemon, method, path string, body any) (int, []byte) {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			t.Fatal(err)
		}
	}
	w := httptest.NewRecorder()
	d.server.Handler().ServeHTTP(w, httptest.NewRequest(method, "/"+protocol.Version+path, &buf))
	return w.Code, w.Body.Bytes()
}

// openProject posts the workspace and wants the status given.
func openProject(t *testing.T, d *Daemon, ws string, want int) protocol.Project {
	t.Helper()
	code, body := route(t, d, http.MethodPost, "/projects", protocol.OpenProjectRequest{Workspace: ws})
	if code != want {
		t.Fatalf("POST /projects %s = %d %s, want %d", ws, code, body, want)
	}
	var p protocol.Project
	if err := json.Unmarshal(body, &p); err != nil {
		t.Fatal(err)
	}
	return p
}

func orchestratorOf(t *testing.T, d *Daemon, id string) protocol.Session {
	t.Helper()
	code, body := route(t, d, http.MethodGet, "/projects/"+id+"/orchestrator", nil)
	if code != http.StatusOK {
		t.Fatalf("GET orchestrator = %d %s", code, body)
	}
	var s protocol.Session
	if err := json.Unmarshal(body, &s); err != nil {
		t.Fatal(err)
	}
	return s
}

// Spec: orchestration .i step 5 — two POST /v1/projects, one by the root and
// one by another worktree of the same tree, give the same Project; the first
// creates, the second finds. The orchestrator is a plan-mode session at the
// root, under the project.
func TestTwoOpensOfOneTreeAreOneProject(t *testing.T) {
	ws := projectRepo(t)
	other := filepath.Join(t.TempDir(), "wt")
	gitIn(t, ws, "worktree", "add", "-q", "-b", "side", other)
	d := orchestratingDaemon(t, t.TempDir(), t.TempDir())

	first := openProject(t, d, ws, http.StatusCreated)
	second := openProject(t, d, other, http.StatusOK)
	if first != second {
		t.Errorf("two opens of one tree gave\n%+v\n%+v", first, second)
	}
	orq := orchestratorOf(t, d, first.ID)
	if orq.ID != first.OrchestratorID || orq.Role != protocol.RoleOrchestrator ||
		orq.Mode != protocol.ModePlan || orq.ProjectID != first.ID || orq.Workspace != ws {
		t.Errorf("orchestrator = %+v", orq)
	}
	if n := d.manager.Count(); n != 1 {
		t.Errorf("%d sessions are open for one project", n)
	}
	if code, body := route(t, d, http.MethodDelete, "/sessions/"+orq.ID, nil); code != http.StatusBadRequest {
		t.Errorf("DELETE of the orchestrator = %d %s, want invalid_input", code, body)
	}
	if code, _ := route(t, d, http.MethodGet, "/projects/ffffffffffffffffffffffff", nil); code != http.StatusNotFound {
		t.Errorf("an unknown project answered %d, want 404", code)
	}
}

// Spec: orchestration .i step 5 — a daemon that restarts gives back the same
// orchestrator, with its history; the turn it was in ends daemon_restart, and
// the decision log says the restart.
func TestTheOrchestratorSurvivesADaemonRestart(t *testing.T) {
	ws := projectRepo(t)
	records, orch := t.TempDir(), t.TempDir()
	first := orchestratingDaemon(t, records, orch)
	p := openProject(t, first, ws, http.StatusCreated)
	sess, err := first.manager.Get(p.OrchestratorID)
	if err != nil {
		t.Fatal(err)
	}
	sess.Emit(protocol.EventTurnStarted, protocol.TurnStarted{TurnID: "t1", Text: "plan the index"})
	before := sess.Log.LastSeq()
	first.manager.CloseAll()
	first.closeProjects()

	again := orchestratingDaemon(t, records, orch)
	if errs := again.reloadProjects(context.Background()); len(errs) != 0 {
		t.Fatalf("reload: %v", errs)
	}
	orq := orchestratorOf(t, again, p.ID)
	if orq.ID != p.OrchestratorID || orq.Role != protocol.RoleOrchestrator || orq.Mode != protocol.ModePlan {
		t.Fatalf("after the restart the orchestrator is %+v, want %s in plan", orq, p.OrchestratorID)
	}
	if orq.LastSeq != before+1 {
		t.Errorf("last seq %d, want %d: the open turn's end and nothing else", orq.LastSeq, before+1)
	}
	history, err := session.Rebuild(filepath.Join(records, p.OrchestratorID+".jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	if len(history) == 0 || !strings.Contains(history[0].Text, "plan the index") {
		t.Errorf("the history did not come back: %+v", history)
	}
	rec, err := os.ReadFile(filepath.Join(records, p.OrchestratorID+".jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(rec), `"reason":"daemon_restart"`) {
		t.Errorf("the open turn did not end daemon_restart:\n%s", rec)
	}
	log, err := os.ReadFile(filepath.Join(orch, p.ID, "decisions.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(log), `"kind":"daemon.restart"`) {
		t.Errorf("the restart is not in the decision log:\n%s", log)
	}
}

// Spec: orchestration .i step 5 — with orchestrator.enabled false, or session
// recording off, the routes answer orchestrator_unavailable naming which.
func TestTheOrchestratorIsUnavailableAndSaysWhy(t *testing.T) {
	ws := projectRepo(t)
	for name, opts := range map[string]DaemonOptions{
		"orchestrator.enabled": {RecordDir: t.TempDir()},
		"record.enabled":       {Orchestrator: OrchestratorOptions{Enabled: true, Dir: t.TempDir()}},
	} {
		opts.SocketPath, opts.Base = filepath.Join(t.TempDir(), "d.sock"), baseOpts(t)
		d := NewDaemon(opts)
		code, body := route(t, d, http.MethodPost, "/projects", protocol.OpenProjectRequest{Workspace: ws})
		if code != http.StatusServiceUnavailable || !strings.Contains(string(body), protocol.CodeOrchestratorUnavailable) ||
			!strings.Contains(string(body), name) {
			t.Errorf("%s off: POST /projects = %d %s", name, code, body)
		}
		if code, body := route(t, d, http.MethodGet, "/projects", nil); code != http.StatusOK || !strings.Contains(string(body), `"projects":[]`) {
			t.Errorf("%s off: GET /projects = %d %s, want an empty list", name, code, body)
		}
	}
}
