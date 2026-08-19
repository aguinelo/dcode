package loop

import (
	"context"
	"strings"
	"testing"

	"github.com/aguinelo/dcode/internal/tools"

	"github.com/aguinelo/dcode/internal/policy"
	"github.com/aguinelo/dcode/internal/provider"
)

// A child may drop capability and never add it.
//
// This is the rule that keeps `owns` from being the mode field the read-only
// construction deliberately refuses. The model asks; the harness intersects.
func TestAWritingChildIsRefusedWhenTheParentCannotWrite(t *testing.T) {
	e, _ := delegateEngine(t, [][]provider.StreamEvent{{text("done"), done()}})
	e.cfg.Mode = policy.ModeReadOnly

	_, err := e.Delegate(context.Background(), "catalogue it", "",
		DelegateLimits{MaxIterations: 3}, []string{"ARCHITECTURE.md"})
	if err == nil {
		t.Fatal("a read-only parent produced a writing child")
	}
	if !strings.Contains(err.Error(), "read-only") {
		t.Errorf("the refusal must say why: %v", err)
	}
}

// Without owns nothing changes at all: the child is the read-only one that
// already existed, built in ModeReadOnly with a registry that cannot write.
func TestAChildWithoutOwnsIsStillReadOnly(t *testing.T) {
	e, _ := delegateEngine(t, [][]provider.StreamEvent{{text("done"), done()}})

	cfg, err := e.childConfig(DelegateLimits{MaxIterations: 3}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Mode != policy.ModeReadOnly {
		t.Errorf("mode = %s, want read-only", cfg.Mode)
	}
	for _, name := range cfg.Tools.Names() {
		if name == "write" || name == "edit" || name == "bash" {
			t.Errorf("a read-only child must not carry %q", name)
		}
	}
}

// With owns the child inherits the parent's mode — inherited, never passed —
// and its containment is narrowed to what it declared.
func TestAWritingChildInheritsTheModeAndIsNarrowedToWhatItOwns(t *testing.T) {
	e, ws := delegateEngine(t, [][]provider.StreamEvent{{text("done"), done()}})
	e.cfg.Mode = policy.ModeWorkspaceWrite

	cfg, err := e.childConfig(DelegateLimits{MaxIterations: 3}, []string{"docs/ARCHITECTURE.md"})
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Mode != policy.ModeWorkspaceWrite {
		t.Errorf("mode = %s, want the parent's", cfg.Mode)
	}
	owned, err := cfg.State.Resolver.Resolve(ws+"/docs/ARCHITECTURE.md", true)
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.State.Resolver.InWorkspace(owned) {
		t.Error("the child cannot write what it owns")
	}

	other, err := cfg.State.Resolver.Resolve(ws+"/src/main.go", true)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.State.Resolver.InWorkspace(other) {
		t.Error("the child can write what it does not own")
	}
}

// A writing child can write, and still cannot delegate or run a command.
//
// Not delegating keeps nesting impossible by absence, as it already was. Not
// running commands is the safe direction to be wrong in while the open question
// in the research spec has no answer: a shell command is opaque, so nothing can
// be proven about what it would touch.
func TestAWritingChildCarriesWritingToolsAndNothingOpaque(t *testing.T) {
	e, _ := delegateEngine(t, [][]provider.StreamEvent{{text("done"), done()}})
	e.cfg.Mode = policy.ModeWorkspaceWrite

	cfg, err := e.childConfig(DelegateLimits{MaxIterations: 3}, []string{"docs/x.md"})
	if err != nil {
		t.Fatal(err)
	}
	names := strings.Join(cfg.Tools.Names(), ",")
	// Only what the parent carries: a child may drop capability and never add
	// it, so asserting a tool the parent does not have would be asserting the
	// opposite of the rule.
	if !strings.Contains(names, "write") {
		t.Errorf("a writing child needs write: %s", names)
	}
	for _, never := range []string{ExploreToolName, "bash"} {
		if strings.Contains(names, never) {
			t.Errorf("a child must never carry %q: %s", never, names)
		}
	}
}

// childRegistry2 and delegateInstructions2 keep the tests that predate `owns`
// reading as they did: the read-only child is still the default, and saying so
// once here beats threading a false argument through every call.
func childRegistry2(parent *tools.Registry) (*tools.Registry, []string) {
	return childRegistry(parent, false)
}

func delegateInstructions2(names []string) string {
	return delegateInstructions(names, nil)
}
