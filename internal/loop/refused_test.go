package loop

import (
	"context"
	"strings"
	"testing"

	"github.com/aguinelo/dcode/internal/policy"
	"github.com/aguinelo/dcode/internal/protocol"
	"github.com/aguinelo/dcode/internal/provider"
)

// A refused write says it was a write.
//
// Consent given to the parent does not transfer to the child (ADR-02), so a
// delegated turn never asks: what would be a question is a refusal, reported
// rather than swallowed. That was already true, and the report called every
// refusal "could not read" — which was the only kind there was.
//
// A writing child makes the name wrong. A refused write reported as an unread
// path tells the parent the child did not look, when what happened is that it
// looked, decided, and was stopped.
func TestARefusedWriteIsReportedAsAWrite(t *testing.T) {
	d := &denyAll{}
	if _, err := d.Approve(context.Background(), protocol.ApprovalRequest{
		Tool:            "write",
		BoundaryCrossed: string(policy.BoundaryPathRuleWrite),
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := d.Approve(context.Background(), protocol.ApprovalRequest{
		Tool:            "read",
		BoundaryCrossed: string(policy.BoundaryPathRuleRead),
	}); err != nil {
		t.Fatal(err)
	}

	if got := d.unwritten; len(got) != 1 || got[0] != "write" {
		t.Errorf("a refused write must be reported as one: %v", got)
	}
	if got := d.unread; len(got) != 1 || got[0] != "read" {
		t.Errorf("a refused read must stay a read: %v", got)
	}
}

// The report says which, because the two mean different things to whoever reads
// it: an unread path is a hole in the answer, and an unwritten one is work that
// did not happen.
func TestTheReportSeparatesWhatItCouldNotWriteFromWhatItCouldNotRead(t *testing.T) {
	r := DelegateResult{
		Conclusion: "catalogued",
		Unread:     []string{".env"},
		Unwritten:  []string{"vendor/x.go"},
	}
	out := r.String()
	if !strings.Contains(out, "could not read: .env") {
		t.Errorf("missing the unread report:\n%s", out)
	}
	if !strings.Contains(out, "could not write: vendor/x.go") {
		t.Errorf("missing the unwritten report:\n%s", out)
	}
}

// The child's tokens come out of the parent's budget, with the child writing
// exactly as when it only read. Without it the parent's ceiling is fiction: a
// turn could delegate its way past any budget.
func TestAWritingChildIsPaidForByTheParent(t *testing.T) {
	e, _ := delegateEngine(t, [][]provider.StreamEvent{
		{call("c1", "write", `{"path":"ARCHITECTURE.md","content":"# arch"}`), spent(300, 20)},
		{text("catalogued"), spent(200, 20)},
	})
	e.cfg.Mode = policy.ModeWorkspaceWrite
	e.cfg.State.BeginTurn()

	if _, err := e.Delegate(context.Background(), "catalogue it", "",
		DelegateLimits{MaxIterations: 3}, []string{"ARCHITECTURE.md"}); err != nil {
		t.Fatal(err)
	}
	if e.delegated.InputTokens != 500 || e.delegated.OutputTokens != 40 {
		t.Errorf("the parent was charged %d in / %d out, want 500/40 — a child that writes costs the same as one that reads",
			e.delegated.InputTokens, e.delegated.OutputTokens)
	}
}

// The ceiling on how much runs at once belongs to the session. A child inherits
// it and no request from the model widens it — a child may drop capability and
// never add it, and concurrency is capability.
func TestAChildDoesNotWidenTheSessionsConcurrency(t *testing.T) {
	e, _ := delegateEngine(t, [][]provider.StreamEvent{{text("done"), done()}})
	e.cfg.Mode = policy.ModeWorkspaceWrite
	e.cfg.Parallel = 2

	cfg, err := e.childConfig(DelegateLimits{MaxIterations: 3}, []string{"docs/x.md"})
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Parallel != e.cfg.Parallel {
		t.Errorf("child parallel = %d, session = %d", cfg.Parallel, e.cfg.Parallel)
	}
}

// The definition of done is the parent's, and it runs once over the whole tree
// after the children finish. A child has no criteria at all: it would be
// checking a tree that is still going to change, and green over an intermediate
// tree is worse than no green — it is the one nobody checks again.
func TestAChildCarriesNoDefinitionOfDone(t *testing.T) {
	e, _ := delegateEngine(t, [][]provider.StreamEvent{{text("done"), done()}})
	e.cfg.Mode = policy.ModeWorkspaceWrite

	cfg, err := e.childConfig(DelegateLimits{MaxIterations: 3}, []string{"docs/x.md"})
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.Done.Criteria) != 0 {
		t.Errorf("a child must carry no criteria, got %d", len(cfg.Done.Criteria))
	}
	if !strings.Contains(delegateInstructions(cfg.Tools.Names(), []string{"docs/x.md"}), "not your job") {
		t.Error("the child is not told that checking the tree is not its job")
	}
}
