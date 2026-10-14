package app

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/aguinelo/dcode/internal/policy"
	"github.com/aguinelo/dcode/internal/protocol"
)

// The daemon runs a whole `/loop` on a folder: it opens the session, submits
// the turn, waits for it to end and keeps where the folder was left — with a
// real session, a real engine and a scripted model, reached through the
// daemon's own routes.
func TestTheDaemonRunsALoopOnAFolder(t *testing.T) {
	ws := t.TempDir()
	spec := filepath.Join(ws, "specs", "a")
	if err := os.MkdirAll(spec, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(spec, "done.toml"), []byte("[ok]\ncommand = \"true\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	model := &scriptedModel{turns: [][]string{{frameText("Nothing to change.")}}}
	srv := httptest.NewServer(http.HandlerFunc(model.serve))
	defer srv.Close()

	base := baseOpts(t)
	base.BaseURL = srv.URL
	base.Policy = policy.PolicyNever
	requireSandbox(t, base)
	d := NewDaemon(DaemonOptions{
		SocketPath:     filepath.Join(t.TempDir(), "d.sock"),
		EventRetention: 10000,
		Base:           base,
	})
	h := d.server.Handler()

	body, _ := json.Marshal(protocol.StartLoopRequest{Workspace: ws, Args: "specs/a"})
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/"+protocol.Version+"/loops", strings.NewReader(string(body))))
	if rec.Code != http.StatusCreated {
		t.Fatalf("starting the loop gave %d %s", rec.Code, rec.Body.String())
	}

	var run protocol.LoopRun
	deadline := time.Now().Add(20 * time.Second)
	for time.Now().Before(deadline) {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/"+protocol.Version+"/loops?workspace="+ws, nil))
		var list protocol.ListLoopsResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &list); err != nil || len(list.Loops) != 1 {
			t.Fatalf("listing gave %s (%v)", rec.Body.String(), err)
		}
		run = list.Loops[0]
		if !run.Live() {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if run.State != protocol.LoopFinished {
		t.Fatalf("the loop ended %s (%s), want finished", run.State, run.Error)
	}
	if len(run.Specs) != 1 || run.Specs[0].Path != "specs/a" || run.Specs[0].SessionID == "" {
		t.Fatalf("specs %+v", run.Specs)
	}
	if run.Verdict == nil || run.Verdict.Worked != 1 {
		t.Errorf("verdict %+v, want one folder worked", run.Verdict)
	}

	// The session the loop opened is the daemon's, and says it is the loop's.
	sess, err := d.manager.Get(run.Specs[0].SessionID)
	if err != nil {
		t.Fatal(err)
	}
	if got := sess.Describe(); got.LoopID != run.ID || got.DoneCriteria != 1 {
		t.Errorf("session %+v, want loop %s and one criterion", got, run.ID)
	}
	model.mu.Lock()
	defer model.mu.Unlock()
	if model.seen == 0 || !strings.Contains(model.bodies[0], "specs/a") {
		t.Error("the turn the loop submitted never reached the model, or did not name the folder")
	}
}
