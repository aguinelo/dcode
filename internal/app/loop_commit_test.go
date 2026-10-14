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

// qualifyingDaemon runs `/loop specs/x` on a folder that declares nothing,
// against a model that proposes one criterion and then holds the turn open
// until release closes. It returns the daemon's handler, the spec folder, and
// the release.
func qualifyingDaemon(t *testing.T) (http.Handler, string, chan struct{}) {
	t.Helper()
	ws := t.TempDir()
	spec := filepath.Join(ws, "specs", "x")
	if err := os.MkdirAll(spec, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(spec, "spec.md"), []byte("# x\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	release := make(chan struct{})
	model := &scriptedModel{turns: [][]string{
		{frameToolCall("c1", "done_propose",
			`{"criteria":[{"name":"red","command":"false","expects":"fail","why":"the work is not done"}]}`)},
		{frameText("Proposed.")},
	}}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		model.mu.Lock()
		round := model.seen
		model.mu.Unlock()
		if round == 1 {
			// The proposal is recorded; the turn has not ended.
			<-release
		}
		model.serve(w, r)
	}))
	t.Cleanup(func() {
		select {
		case <-release:
		default:
			close(release)
		}
		srv.Close()
	})

	base := baseOpts(t)
	base.BaseURL = srv.URL
	base.Policy = policy.PolicyNever
	requireSandbox(t, base)
	read := base
	read.SandboxMode = policy.ModeReadOnly
	requireSandbox(t, read)
	d := NewDaemon(DaemonOptions{
		SocketPath:     filepath.Join(t.TempDir(), "d.sock"),
		EventRetention: 10000,
		Base:           base,
	})
	h := d.server.Handler()

	body, _ := json.Marshal(protocol.StartLoopRequest{Workspace: ws, Args: "specs/x"})
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/"+protocol.Version+"/loops", strings.NewReader(string(body))))
	if rec.Code != http.StatusCreated {
		t.Fatalf("starting the loop gave %d %s", rec.Code, rec.Body.String())
	}
	return h, spec, release
}

func loopNow(t *testing.T, h http.Handler) protocol.LoopRun {
	t.Helper()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/"+protocol.Version+"/loops", nil))
	var list protocol.ListLoopsResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &list); err != nil || len(list.Loops) != 1 {
		t.Fatalf("listing gave %s (%v)", rec.Body.String(), err)
	}
	return list.Loops[0]
}

func waitLoopState(t *testing.T, h http.Handler, ok func(protocol.LoopRun) bool) protocol.LoopRun {
	t.Helper()
	deadline := time.Now().Add(20 * time.Second)
	for time.Now().Before(deadline) {
		if run := loopNow(t, h); ok(run) {
			return run
		}
		time.Sleep(20 * time.Millisecond)
	}
	run := loopNow(t, h)
	t.Fatalf("the loop never got there: %+v", run)
	return run
}

// The proposal is written when the qualifying TURN ends, not whenever the
// session happens to be idle.
//
// A session is idle before its first turn starts, and the client that ran the
// loop used to replay from the beginning — so the first event of a brand-new
// qualifying session found it idle, and committed a proposal nobody had made
// yet:
//
//	could not write the proposed definition of done: invalid_input:
//	nothing was proposed for 1a05dd01b225554ea98
//
// The daemon conducts the loop now, and the rule is its: with the proposal
// recorded and the turn still open, nothing is written.
func TestTheProposalIsNotCommittedBeforeTheTurnStarts(t *testing.T) {
	h, spec, _ := qualifyingDaemon(t)
	run := waitLoopState(t, h, func(r protocol.LoopRun) bool {
		return r.State == protocol.LoopQualifying && len(r.Specs) == 1 && r.Specs[0].SessionID != ""
	})
	// Long enough for the proposal to be recorded and the turn to sit open.
	time.Sleep(300 * time.Millisecond)
	if _, err := os.Stat(filepath.Join(spec, "done.toml")); err == nil {
		t.Fatal("the proposal was written before its turn had ended")
	}
	if now := loopNow(t, h); now.State != protocol.LoopQualifying || now.Specs[0].Proposal != "" {
		t.Errorf("the loop moved on before the turn ended: %+v (was %+v)", now, run)
	}
}

// And it IS written when the turn completes.
func TestTheProposalIsCommittedWhenTheTurnCompletes(t *testing.T) {
	h, spec, release := qualifyingDaemon(t)
	waitLoopState(t, h, func(r protocol.LoopRun) bool { return r.State == protocol.LoopQualifying })
	close(release)
	run := waitLoopState(t, h, func(r protocol.LoopRun) bool { return !r.Live() || r.State == protocol.LoopAwaitingReview })
	if run.State != protocol.LoopAwaitingReview {
		t.Fatalf("the turn ended and the proposal was never written: %+v", run)
	}
	if run.Specs[0].ProposalCriteria != 1 || !strings.HasSuffix(run.Specs[0].Proposal, "done.toml") {
		t.Errorf("the run does not say what was written: %+v", run.Specs[0])
	}
	if _, err := os.Stat(filepath.Join(spec, "done.toml")); err != nil {
		t.Errorf("no file was written: %v", err)
	}
}
