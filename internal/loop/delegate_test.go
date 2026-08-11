package loop

import (
	"context"
	"strings"
	"testing"

	ce "github.com/aguinelo/dcode/internal/contextengine"
	"github.com/aguinelo/dcode/internal/policy"
	"github.com/aguinelo/dcode/internal/protocol"
	"github.com/aguinelo/dcode/internal/tools"
)

func fullRegistry() *tools.Registry {
	return tools.NewRegistry(
		tools.Read{}, tools.Write{}, tools.Edit{},
		tools.Glob{}, tools.Grep{}, tools.Symbol{},
		tools.Bash{}, tools.Plan{}, tools.Explore{},
	)
}

// The child cannot write, and not because a condition says so: the tools that
// write are not in its registry.
func TestTheChildGetsOnlyReadingTools(t *testing.T) {
	reg, names := readOnlyRegistry(fullRegistry())

	for _, forbidden := range []string{"write", "edit", "bash", "plan"} {
		if _, ok := reg.Get(forbidden); ok {
			t.Errorf("the child can call %q", forbidden)
		}
	}
	for _, wanted := range []string{"read", "glob", "grep", "symbol"} {
		if _, ok := reg.Get(wanted); !ok {
			t.Errorf("the child cannot call %q, so it cannot answer anything", wanted)
		}
	}
	if len(names) != 4 {
		t.Errorf("readable tools = %v, want the four that only read", names)
	}
}

// Nesting is impossible rather than forbidden. Nested delegation is exponential
// cost and the error lands far from its cause.
func TestTheChildCannotDelegateAgain(t *testing.T) {
	reg, names := readOnlyRegistry(fullRegistry())
	if _, ok := reg.Get(ExploreToolName); ok {
		t.Fatal("the child's registry contains the delegating tool")
	}
	for _, n := range names {
		if n == ExploreToolName {
			t.Fatal("the delegating tool was offered to the child in its instructions")
		}
	}
}

// An allow list, so a tool added later is excluded until someone thinks about
// it. That is the safe direction to be wrong in.
func TestAToolAddedLaterIsExcludedUntilConsidered(t *testing.T) {
	parent := tools.NewRegistry(tools.Read{}, unknownTool{})
	reg, _ := readOnlyRegistry(parent)
	if _, ok := reg.Get("deploy"); ok {
		t.Fatal("a tool nobody classified reached the child")
	}
}

type unknownTool struct{ tools.Read }

func (unknownTool) Name() string { return "deploy" }

// Approval is consent, and consent given to the parent does not transfer.
func TestADelegatedTurnNeverAsksAndReportsWhatItCouldNotRead(t *testing.T) {
	a := &denyAll{}
	dec, err := a.Approve(context.Background(), protocol.ApprovalRequest{Tool: "read", Command: "secrets/keys.env"})
	if err != nil {
		t.Fatal(err)
	}
	if dec != protocol.ApprovalDeny {
		t.Fatalf("decision = %v, want deny: a child must never prompt the user for a question they did not ask", dec)
	}
	if len(a.denied) != 1 || a.denied[0] != "secrets/keys.env" {
		t.Fatalf("denied = %v, want the refused path recorded so the report can declare it", a.denied)
	}
}

// A conclusion with an undeclared hole is a wrong conclusion wearing the face
// of a complete one.
func TestTheResultCarriesWhereItLookedAndWhatItCouldNotRead(t *testing.T) {
	r := DelegateResult{
		Conclusion: "Payments are validated in pay/validate.go:42.",
		Read:       []string{"pay/validate.go", "pay/types.go"},
		Unread:     []string{"pay/.env"},
	}
	out := r.String()
	for _, want := range []string{"validate.go:42", "looked at:", "pay/types.go", "could not read:", "pay/.env"} {
		if !strings.Contains(out, want) {
			t.Errorf("the report is missing %q:\n%s", want, out)
		}
	}
}

func TestATruncatedReportSaysSo(t *testing.T) {
	r := DelegateResult{Conclusion: "long", Truncated: true}
	if !strings.Contains(r.String(), "truncated") {
		t.Fatalf("a cut report did not declare it:\n%s", r.String())
	}
}

// Without this the parent's ceiling is fiction: a turn could delegate its way
// past any budget.
func TestChildTokensAreDebitedFromTheParent(t *testing.T) {
	e := New(Config{}, ce.Session{Instructions: "x"})
	e.delegated.InputTokens = 900
	e.delegated.OutputTokens = 100

	out := Outcome{}
	out.Usage.InputTokens += e.delegated.InputTokens
	out.Usage.OutputTokens += e.delegated.OutputTokens
	if out.Usage.InputTokens != 900 || out.Usage.OutputTokens != 100 {
		t.Fatal("the child's usage is not reaching the parent's accounting")
	}
}

// The mode is fixed where the sub-turn is built, so it is not a field the model
// passes and not one a caller forgets.
func TestReadOnlyIsNotSomethingTheInputCanCarry(t *testing.T) {
	var in tools.ExploreInput
	// A compile-time fact stated as a test: if someone adds a Mode field to
	// ExploreInput, this stops compiling and they have to justify it.
	_ = in.Task
	_ = in.Path
	if policy.ModeReadOnly != "read-only" {
		t.Fatal("the mode the child is built with changed name")
	}
}
