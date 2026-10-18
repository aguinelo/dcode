package evals

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	ce "github.com/aguinelo/dcode/internal/contextengine"
)

func delegateCall(items string) ce.ToolCall {
	return ce.ToolCall{Name: "delegate", Input: json.RawMessage(`{"items":` + items + `}`)}
}

// The judge of orchestrator-splits-by-theme passes the batch the contract asks
// for and nothing short of it: too few items, an item that owns nothing, owns
// that cross, or no delegate at all.
func TestDisjointBatchJudgesTheBatchTheContractAsks(t *testing.T) {
	three := `[{"title":"slug","owns":["internal/slug"]},{"title":"money","owns":["internal/money"]},{"title":"retry","owns":["internal/retry"]}]`
	for _, tc := range []struct {
		name  string
		calls []ce.ToolCall
		want  bool
	}{
		{"three disjoint", []ce.ToolCall{delegateCall(three)}, true},
		{"two items", []ce.ToolCall{delegateCall(`[{"title":"a","owns":["a"]},{"title":"b","owns":["b"]}]`)}, false},
		{"one owns nothing", []ce.ToolCall{delegateCall(`[{"title":"a","owns":["a"]},{"title":"b","owns":[]},{"title":"c","owns":["c"]}]`)}, false},
		{"owns cross", []ce.ToolCall{delegateCall(`[{"title":"a","owns":["internal"]},{"title":"b","owns":["internal/b"]},{"title":"c","owns":["c"]}]`)}, false},
		{"no delegate", []ce.ToolCall{{Name: "read", Input: json.RawMessage(`{"path":"go.mod"}`)}}, false},
		{"a later batch", []ce.ToolCall{delegateCall(`[{"title":"a","owns":["a"]}]`), delegateCall(three)}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := DisjointBatch(3)(Transcript{Calls: tc.calls}); got != tc.want {
				t.Errorf("DisjointBatch(3) = %v, want %v", got, tc.want)
			}
		})
	}
}

// The scenario is measured as the product builds an orchestrator: read-only,
// with the role section, and a delegate that answers rather than refuses.
func TestTheOrchestratorScenarioIsTheOrchestratorTheProductBuilds(t *testing.T) {
	f, err := LoadFixture(FixtureRoot, "orchestrator-splits-by-theme")
	if err != nil {
		t.Fatal(err)
	}
	p, err := f.Prompt("minimax-m3")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(p, "You are this project's orchestrator") || !strings.Contains(p, "Read-only, by your role") {
		t.Errorf("the scenario's prompt is not the orchestrator's:\n%s", p)
	}
	w, err := NewWorkspace(t.TempDir(), f.Files, f.ToolNames())
	if err != nil {
		t.Fatal(err)
	}
	w.Mode = "read-only"
	out, isErr := w.Execute(context.Background(), "delegate",
		json.RawMessage(`{"items":[{"title":"slug","task":"t","owns":["internal/slug"],"why":"w"}]}`))
	if isErr || !strings.Contains(out, "Dispatched 1 child") {
		t.Errorf("delegate answered %q (error %v)", out, isErr)
	}
	if out, isErr := w.Execute(context.Background(), "write", json.RawMessage(`{"path":"x.go","content":"package x"}`)); !isErr {
		t.Errorf("a write under the orchestrator's read-only went through: %q", out)
	}
	if out, _ := w.Execute(context.Background(), "children", json.RawMessage(`{}`)); !strings.Contains(out, "internal/slug") {
		t.Errorf("children does not list the dispatched child: %q", out)
	}
}

func TestAScenarioRoleThatIsNoRoleIsRefused(t *testing.T) {
	if err := validRole("conductor"); err == nil {
		t.Error("a role that is no role was accepted")
	}
	for _, r := range []string{"", "orchestrator", "child", "reviewer"} {
		if err := validRole(r); err != nil {
			t.Errorf("%q: %v", r, err)
		}
	}
}
