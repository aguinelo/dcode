package app

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/aguinelo/dcode/internal/orchestrate"
	"github.com/aguinelo/dcode/internal/protocol"
	"github.com/aguinelo/dcode/internal/tools"
)

// Spec: orchestration .p §7.2, §10 — a reviewer session has the reviewer's
// registry alone, whatever the session would otherwise carry; a child has
// red_test, which it uses only in its red phase.
func TestTheReviewerSessionHasOnlyTheReviewersTools(t *testing.T) {
	ws := projectRepo(t)
	opts := baseOpts(t)
	opts.Workspace = ws
	requireSandbox(t, opts)
	reviewerMode(func(_ context.Context, _ tools.VerdictInput) (string, error) { return "ok", nil })(&opts)
	rv, err := New(opts, &ConsoleEmitter{W: io.Discard}, DenyAll{})
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(rv.Registry.Names(), " "); got != "glob grep read symbol verdict" {
		t.Errorf("the reviewer's tools are %q", got)
	}

	child := baseOpts(t)
	child.Workspace = ws
	(&Daemon{}).childMode("c", []string{"docs"})(&child)
	c, err := New(child, &ConsoleEmitter{W: io.Discard}, DenyAll{})
	if err != nil {
		t.Fatal(err)
	}
	red, ok := c.Registry.Get("red_test")
	if !ok {
		t.Fatal("the child has no red_test")
	}
	res, _ := red.Execute(t.Context(), json.RawMessage(`{"command":"go test ./x"}`), nil)
	if !res.IsError || !strings.Contains(res.Output, "conducts no projects") {
		t.Errorf("red_test outside a project = %+v", res)
	}
}

// recordingModel answers the orchestrator from a script and keeps every
// request body it was sent.
type recordingModel struct {
	mu     sync.Mutex
	script [][]string
	bodies [][]byte
}

func (m *recordingModel) serve(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	m.mu.Lock()
	n := len(m.bodies)
	m.bodies = append(m.bodies, body)
	turn := []string{frameText("nothing more")}
	if n < len(m.script) {
		turn = m.script[n]
	}
	m.mu.Unlock()
	w.Header().Set("Content-Type", "text/event-stream")
	for _, f := range turn {
		_, _ = w.Write([]byte("data: " + f + "\n\n"))
	}
	_, _ = w.Write([]byte("data: [DONE]\n\n"))
}

func (m *recordingModel) sent() [][]byte {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([][]byte(nil), m.bodies...)
}

// Spec: orchestration .i step 7 — the report starts its own turn, with Origin
// children, and the context before it is byte for byte the one the model read
// before (ADR-03): the report is appended, never inserted or rewritten.
func TestTheReportIsAppendedAndThePrefixIsByteForByteTheSame(t *testing.T) {
	t.Setenv("GIT_CONFIG_GLOBAL", "/dev/null")
	ws := projectRepo(t)
	m := &recordingModel{script: [][]string{{frameText("Understood.")}, {frameText("Noted the report.")}}}
	srv := httptest.NewServer(http.HandlerFunc(m.serve))
	t.Cleanup(srv.Close)
	base := baseOpts(t)
	base.BaseURL = srv.URL
	requireSandbox(t, base)
	d := NewDaemon(DaemonOptions{
		SocketPath: filepath.Join(t.TempDir(), "d.sock"), EventRetention: 10000, RecordDir: t.TempDir(),
		ApprovalTimeout: time.Minute, Base: base, Log: func(string) {},
		Orchestrator: OrchestratorOptions{Enabled: true, Dir: t.TempDir()},
	})
	t.Cleanup(d.manager.CloseAll)
	t.Cleanup(d.closeProjects)
	p := openProject(t, d, ws, http.StatusCreated)
	orq, err := d.manager.Get(p.OrchestratorID)
	if err != nil {
		t.Fatal(err)
	}
	if err := orq.Submit("Plan the work."); err != nil {
		t.Fatal(err)
	}
	waitUntil(t, "the person's turn did not end", func() bool {
		return len(m.sent()) == 1 && orq.State() == protocol.SessionStateIdle
	})
	o, _ := d.projects.reg.Get(p.ID)
	o.Report("Child c1 (\"docs\", failed): its turn ended error.")
	waitUntil(t, "the report did not start a turn", func() bool {
		return len(m.sent()) == 2 && orq.State() == protocol.SessionStateIdle
	})

	bodies := m.sent()
	first, second := messagesOf(t, bodies[0]), messagesOf(t, bodies[1])
	if len(second) <= len(first) {
		t.Fatalf("the report turn has %d messages, the first %d", len(second), len(first))
	}
	for i := range first {
		if !bytes.Equal(first[i], second[i]) {
			t.Fatalf("message %d changed under the report:\nbefore %s\nafter  %s", i, first[i], second[i])
		}
	}
	if last := string(second[len(second)-1]); !strings.Contains(last, "Reports from your children") || !strings.Contains(last, "c1") {
		t.Errorf("the report is not the last message: %s", last)
	}

	events, err := orq.Log.Replay(0)
	if err != nil {
		t.Fatal(err)
	}
	var origins []string
	for _, ev := range events {
		if ev.Type == protocol.EventTurnStarted {
			var ts protocol.TurnStarted
			_ = json.Unmarshal(ev.Payload, &ts)
			origins = append(origins, ts.Origin)
		}
	}
	if len(origins) != 2 || origins[0] != "" || origins[1] != protocol.OriginChildren {
		t.Errorf("turn origins = %q, want the person's then children", origins)
	}
}

// messagesOf is the messages of a chat completion request, each as sent.
func messagesOf(t *testing.T, body []byte) []json.RawMessage {
	t.Helper()
	var req struct {
		Messages []json.RawMessage `json:"messages"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		t.Fatalf("request body: %v\n%s", err, body)
	}
	return req.Messages
}

// Spec: orchestration .p §7.2 — SpawnReviewer refuses a request it cannot
// build a review from.
func TestSpawnReviewerRefusesWhatItCannotBuild(t *testing.T) {
	ws := projectRepo(t)
	d := modelDaemon(t, &routedModel{}, time.Minute, t.TempDir(), t.TempDir())
	p := openProject(t, d, ws, http.StatusCreated)
	submit := func(_ context.Context, _ tools.VerdictInput) (string, error) { return "", nil }
	for name, req := range map[string]orchestrate.ReviewRequest{
		"bad id":    {ID: "../x", ParentID: p.OrchestratorID, Worktree: ws, Submit: submit},
		"no review": {ID: "r1", ParentID: p.OrchestratorID, Worktree: ws},
		"no parent": {ID: "r2", ParentID: "nobody", Worktree: ws, Submit: submit},
	} {
		if _, err := d.SpawnReviewer(req); err == nil {
			t.Errorf("%s: the reviewer was built", name)
		}
	}
	s, err := d.SpawnReviewer(orchestrate.ReviewRequest{ID: "20261017000000000-00000001", ParentID: p.OrchestratorID, Worktree: ws, Submit: submit})
	if err != nil {
		t.Fatal(err)
	}
	if s.Role != protocol.RoleReviewer || s.ParentID != p.OrchestratorID || s.Mode != protocol.ModePlan {
		t.Errorf("reviewer session = %+v", s)
	}
	if d.knownProfile(ws, "nowhere") {
		t.Error("a profile models.toml does not have is known")
	}
}
