package session

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/aguinelo/dcode/internal/policy"
	"github.com/aguinelo/dcode/internal/protocol"
)

// Spec: orchestration .i step 5 — the orchestrator's record is what it comes
// back from after a restart, so pruning old history never takes it, however
// old it is. A plain record of the same age goes.
func TestPruneSparesAnOrchestratorsRecord(t *testing.T) {
	dir := t.TempDir()
	orq := filepath.Join(dir, "orq.jsonl")
	body := `{"seq":1,"type":"session.created","payload":{"id":"orq","workspace":"/w","role":"orchestrator"}}` + "\n"
	if err := os.WriteFile(orq, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	old := time.Now().Add(-400 * 24 * time.Hour)
	if err := os.Chtimes(orq, old, old); err != nil {
		t.Fatal(err)
	}
	writeRecord(t, dir, "plain.jsonl", 10, 400*24*time.Hour)

	n, err := Prune(dir, PruneBudget{MaxAge: 24 * time.Hour, MaxBytes: 1})
	if err != nil {
		t.Fatal(err)
	}
	if got := kept(t, dir); n != 1 || len(got) != 1 || got[0] != "orq.jsonl" {
		t.Errorf("removed %d, kept %v; want the orchestrator's record alone", n, got)
	}
}

// Spec: orchestration .p §10 — the orchestrator runs in plan mode and never
// writes to the tree, so a switch out of plan is refused and the mode stays.
func TestAnOrchestratorDoesNotLeavePlanMode(t *testing.T) {
	s := modeSession(t, policy.ModeReadOnly, policy.PolicyNever)
	s.Role = protocol.RoleOrchestrator
	before := s.Log.LastSeq()
	for _, mode := range []string{protocol.ModeAssist, protocol.ModeAuto} {
		err := s.SetMode(mode)
		if pe, ok := protocol.AsError(err); !ok || pe.Code != protocol.CodeInvalidInput {
			t.Errorf("SetMode(%s) on an orchestrator = %v, want invalid_input", mode, err)
		}
	}
	if got := s.Describe().Mode; got != protocol.ModePlan {
		t.Errorf("mode = %q after refused switches, want plan", got)
	}
	if s.Log.LastSeq() != before {
		t.Error("a refused switch announced something")
	}
}
