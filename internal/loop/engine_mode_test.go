package loop

import (
	"path/filepath"
	"sync"
	"testing"

	"github.com/aguinelo/dcode/internal/policy"
	"github.com/aguinelo/dcode/internal/provider"
)

func writeInside(ws string) policy.Request {
	return policy.Request{
		Tool:  "write",
		Paths: []policy.Access{{Path: filepath.Join(ws, "pay.go"), Write: true}},
	}
}

// TestSetModeChangesTheVerdict asks the decision, not the field.
//
// A test that sets cfg.Mode and then asserts cfg.Mode is the shape that cannot
// fail: it passes with every reader of the field still holding a stale copy.
// What makes SetMode true is that the next thing decided is decided differently.
func TestSetModeChangesTheVerdict(t *testing.T) {
	e, ws := delegateEngine(t, nil)

	if got := e.evaluate(writeInside(ws)).Decision; got != policy.DecisionAllow {
		t.Fatalf("workspace-write must allow a write inside the workspace, got %q", got)
	}
	e.SetMode(policy.ModeReadOnly, policy.PolicyNever)
	if got := e.evaluate(writeInside(ws)).Decision; got != policy.DecisionDeny {
		t.Errorf("after SetMode(read-only, never): decision = %q, want deny", got)
	}
	e.SetMode(policy.ModeWorkspaceWrite, policy.PolicyOnRequest)
	if got := e.evaluate(writeInside(ws)).Decision; got != policy.DecisionAllow {
		t.Errorf("the way back must work too: decision = %q, want allow", got)
	}
}

// TestSetModeIsSeenByDelegation covers the second reader of the same fields.
//
// evaluate is not the only place the mode is consulted: childConfig refuses a
// writing child under read-only and inherits the mode when it does not. A fix
// that reaches only evaluate leaves this path answering from before the switch.
func TestSetModeIsSeenByDelegation(t *testing.T) {
	e, _ := delegateEngine(t, nil)

	cfg, err := e.childConfig(DelegateLimits{MaxIterations: 3}, []string{"pay.go"})
	if err != nil {
		t.Fatalf("a writing child under workspace-write must be allowed: %v", err)
	}
	if cfg.Mode != policy.ModeWorkspaceWrite {
		t.Errorf("child inherited mode %q, want workspace-write", cfg.Mode)
	}
	e.SetMode(policy.ModeReadOnly, policy.PolicyNever)
	if _, err := e.childConfig(DelegateLimits{MaxIterations: 3}, []string{"pay.go"}); err == nil {
		t.Error("after SetMode(read-only): a writing child must be refused, got no error")
	}
}

// TestSetModeUnderConcurrentReads is the race guard.
//
// SetMode is called from the HTTP handler while a turn runs — its own doc says
// live turns are not interrupted, and that is precisely the window. Both
// readers run here because guarding only one of them is what happened the first
// time. Meaningful under -race, which is how the gate runs it.
func TestSetModeUnderConcurrentReads(t *testing.T) {
	e, ws := delegateEngine(t, [][]provider.StreamEvent{})

	var wg sync.WaitGroup
	wg.Add(3)
	go func() {
		defer wg.Done()
		for i := range 500 {
			if i%2 == 0 {
				e.SetMode(policy.ModeFullAccess, policy.PolicyOnRequest)
				continue
			}
			e.SetMode(policy.ModeReadOnly, policy.PolicyNever)
		}
	}()
	go func() {
		defer wg.Done()
		for range 500 {
			_ = e.evaluate(writeInside(ws))
		}
	}()
	go func() {
		defer wg.Done()
		for range 500 {
			_, _ = e.childConfig(DelegateLimits{MaxIterations: 3}, []string{"pay.go"})
		}
	}()
	wg.Wait()
}
