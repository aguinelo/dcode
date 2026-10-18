package orchestrate

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/aguinelo/dcode/internal/protocol"
)

// lastPlan is the last plan.updated the registry sent on the orchestrator's log.
func (p *project) lastPlan(t *testing.T) []protocol.PlanItem {
	t.Helper()
	p.w.mu.Lock()
	defer p.w.mu.Unlock()
	d, ok := p.w.last[protocol.EventPlanUpdated].(protocol.PlanUpdated)
	if !ok {
		t.Fatal("no plan.updated was sent on the orchestrator's log")
	}
	return d.Items
}

func statusOf(items []protocol.PlanItem, id int) protocol.PlanItem {
	for _, it := range items {
		if it.ID == id {
			return it
		}
	}
	return protocol.PlanItem{}
}

// Spec: orchestration .p §10 — a plan step's status follows the children that
// fulfil it: in progress while one works, waits, is reviewed, gated or waits
// for its merge; done once every one is merged; blocked, saying who, when one
// failed. A closed child no longer holds its step, a step with no child keeps
// the orchestrator's status, and a child with no step changes nothing.
func TestAPlanStepFollowsTheStateOfItsChildren(t *testing.T) {
	kid := func(id, state string, step int) protocol.Child {
		return protocol.Child{ID: id, Title: "t-" + id, State: state, PlanStep: step, Reason: "incomplete"}
	}
	plan := []protocol.PlanItem{
		{ID: 1, Text: "index", Status: protocol.PlanPending},
		{ID: 2, Text: "docs", Status: protocol.PlanPending},
		{ID: 3, Text: "notes", Status: protocol.PlanActive},
	}
	for _, c := range []struct {
		name     string
		children []protocol.Child
		want     string // the status of step 1
		blocked  string // a fragment of its reason
	}{
		{"dispatched", []protocol.Child{kid("a", stateDispatched, 1)}, protocol.PlanActive, ""},
		{"working", []protocol.Child{kid("a", stateWorking, 1)}, protocol.PlanActive, ""},
		{"awaiting", []protocol.Child{kid("a", stateAwaiting, 1)}, protocol.PlanActive, ""},
		{"in review", []protocol.Child{kid("a", stateInReview, 1)}, protocol.PlanActive, ""},
		{"awaiting merge", []protocol.Child{kid("a", statePR, 1)}, protocol.PlanActive, ""},
		{"merged", []protocol.Child{kid("a", stateMerged, 1)}, protocol.PlanDone, ""},
		{"one merged, one working", []protocol.Child{kid("a", stateMerged, 1), kid("b", stateWorking, 1)}, protocol.PlanActive, ""},
		{"both merged", []protocol.Child{kid("a", stateMerged, 1), kid("b", stateMerged, 1)}, protocol.PlanDone, ""},
		{"failed", []protocol.Child{kid("a", stateFailed, 1), kid("b", stateWorking, 1)}, protocol.PlanBlocked, `"t-a" failed (incomplete)`},
		{"closed", []protocol.Child{kid("a", stateClosed, 1)}, protocol.PlanPending, ""},
		{"no step", []protocol.Child{kid("a", stateMerged, 0)}, protocol.PlanPending, ""},
	} {
		t.Run(c.name, func(t *testing.T) {
			got := planOf(plan, c.children)
			one := statusOf(got, 1)
			if one.Status != c.want || !strings.Contains(one.Blocked, c.blocked) || (c.blocked == "" && one.Blocked != "") {
				t.Errorf("step 1 = %s %q, want %s with %q", one.Status, one.Blocked, c.want, c.blocked)
			}
			if two, three := statusOf(got, 2), statusOf(got, 3); two.Status != protocol.PlanPending || three.Status != protocol.PlanActive {
				t.Errorf("steps with no child moved: 2 is %s, 3 is %s", two.Status, three.Status)
			}
			if plan[0].Status != protocol.PlanPending {
				t.Fatal("planOf changed the plan it was given")
			}
		})
	}
}

// Spec: orchestration .p §10 — the step a delegate item names is recorded in
// the dispatch decision and on the Child, and the harness sends the plan with
// that step advanced on the orchestrator's log as the child moves; the plan it
// sent, read back, sends nothing more.
func TestTheHarnessAdvancesThePlanStepFromTheChild(t *testing.T) {
	p := openWith(t, nil)
	orq := p.o.orchestratorID()
	p.reg.Observe(orq, event(t, protocol.EventPlanUpdated, protocol.PlanUpdated{Items: []protocol.PlanItem{
		{ID: 1, Text: "index", Status: protocol.PlanActive},
		{ID: 2, Text: "docs", Status: protocol.PlanPending},
	}}))
	it := item("index", "internal/index")
	it.PlanStep = 2
	c := p.one(t, it)
	if c.PlanStep != 2 || p.o.child(c.ID).PlanStep != 2 {
		t.Fatalf("the child holds plan step %d, want 2", p.o.child(c.ID).PlanStep)
	}
	var data map[string]any
	lines := p.lines(t, "dispatch")
	if err := json.Unmarshal(lines[len(lines)-1].Data, &data); err != nil || data["plan_step"] != float64(2) {
		t.Errorf("the dispatch decision says plan_step %v (%v)", data["plan_step"], err)
	}
	if got := statusOf(p.lastPlan(t), 2); got.Status != protocol.PlanActive {
		t.Errorf("with its child working, step 2 is %s", got.Status)
	}
	if got := statusOf(p.lastPlan(t), 1); got.Status != protocol.PlanActive {
		t.Errorf("step 1, with no child, is %s; the orchestrator said active", got.Status)
	}
	other := p.one(t, item("notes", "docs/notes"))
	if other.PlanStep != 0 {
		t.Errorf("a child with no step holds step %d", other.PlanStep)
	}

	p.end(t, c.ID, protocol.StopMaxIterations, "ran out")
	got := statusOf(p.lastPlan(t), 2)
	if got.Status != protocol.PlanBlocked || !strings.Contains(got.Blocked, `"index" failed (max_iterations)`) {
		t.Fatalf("with its child failed, step 2 is %s %q", got.Status, got.Blocked)
	}

	sent := p.w.count(protocol.EventPlanUpdated)
	p.reg.Observe(orq, event(t, protocol.EventPlanUpdated, protocol.PlanUpdated{Items: p.lastPlan(t)}))
	if n := p.w.count(protocol.EventPlanUpdated); n != sent {
		t.Errorf("the plan the harness sent, read back, sent %d more", n-sent)
	}
}

// Spec: orchestration .p §10 — a plan step below zero is no step, and the
// batch is refused whole.
func TestADelegateItemWithANegativePlanStepIsRefused(t *testing.T) {
	p := openWith(t, nil)
	it := item("index", "internal/index")
	it.PlanStep = -1
	_, err := p.o.Delegate([]Item{it})
	wantCode(t, err, protocol.CodeInvalidInput)
	p.noWorktree(t)
}
