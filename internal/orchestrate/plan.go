package orchestrate

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"github.com/aguinelo/dcode/internal/protocol"
)

// The orchestrator's plan, followed by its children: a delegate item names
// the plan step it fulfils, and the harness sets that step's status from the
// children's states, so the plan the person reads says where the work is
// without the orchestrator taking a turn to say it.
//
// The plan is the orchestrator's own — its plan tool writes it, and the steps
// with no child keep the status it gave them. What the harness sends is that
// plan with the statuses of the steps that have children set from them, on
// the orchestrator's log, every time a child moves. Not a decision: a
// projection of the children, whose every move is already in the log.
//
// Spec: docs/specs/architecture/orchestration/202610132000-orchestration.p.spec.md §10.

// planOf is plan with the status of each step that has children set from
// them: blocked, saying who, when one failed; done when every one is merged;
// active while any is at work, waiting, under review, gated or waiting for its
// merge. A closed child no longer holds its step. plan itself is not changed.
func planOf(plan []protocol.PlanItem, children []protocol.Child) []protocol.PlanItem {
	out := make([]protocol.PlanItem, len(plan))
	copy(out, plan)
	for i := range out {
		var held, merged int
		var failed []string
		for _, c := range children {
			if c.PlanStep != out[i].ID || c.PlanStep == 0 || c.State == stateClosed {
				continue
			}
			held++
			switch c.State {
			case stateMerged:
				merged++
			case stateFailed:
				failed = append(failed, failure(c))
			}
		}
		switch {
		case held == 0:
		case len(failed) > 0:
			out[i].Status, out[i].Blocked = protocol.PlanBlocked, strings.Join(failed, "; ")
		case merged == held:
			out[i].Status, out[i].Blocked = protocol.PlanDone, ""
		default:
			out[i].Status, out[i].Blocked = protocol.PlanActive, ""
		}
	}
	return out
}

// failure says a failed child in the plan's blocked reason.
func failure(c protocol.Child) string {
	if c.Reason == "" {
		return fmt.Sprintf("child %q failed", c.Title)
	}
	return fmt.Sprintf("child %q failed (%s)", c.Title, c.Reason)
}

// planRevised reads a plan.updated of the orchestrator's session. The plan
// the harness sent comes back this way too, and is not the orchestrator's:
// it changes nothing. Any other is the orchestrator's plan tool, kept, and
// sent again with the steps that have children set from them.
func (o *Orchestrator) planRevised(ev protocol.Event) {
	var d protocol.PlanUpdated
	if err := json.Unmarshal(ev.Payload, &d); err != nil {
		o.sayError(protocol.CodeInternal, fmt.Sprintf("the orchestrator's plan does not read, so its steps do not follow its children: %v", err))
		return
	}
	o.planMu.Lock()
	defer o.planMu.Unlock()
	if reflect.DeepEqual(d.Items, o.planSent) {
		return
	}
	o.mu.Lock()
	st := cloneState(o.state)
	st.Plan = append([]protocol.PlanItem(nil), d.Items...)
	err := o.store.Save(st)
	if err == nil {
		o.state = st
	}
	o.mu.Unlock()
	if err != nil {
		o.sayError(protocol.CodeInternal, "the orchestrator's plan could not be saved, so a restart loses which steps follow which children: "+err.Error())
	}
	// What the wire shows now is what the plan tool sent.
	o.planSent = append([]protocol.PlanItem(nil), d.Items...)
	o.sendPlan(d.Items)
}

// advancePlan sends the plan again when a child's move changes a step.
func (o *Orchestrator) advancePlan() {
	o.planMu.Lock()
	defer o.planMu.Unlock()
	st := o.snapshot()
	o.sendPlan(st.Plan)
}

// sendPlan sends plan with its steps set from the children, unless the wire
// already shows it. Called with planMu held.
func (o *Orchestrator) sendPlan(plan []protocol.PlanItem) {
	if len(plan) == 0 {
		return
	}
	next := planOf(plan, o.snapshot().Children)
	if reflect.DeepEqual(next, o.planSent) {
		return
	}
	o.planSent = next
	if o.reg != nil && o.reg.emit != nil {
		o.reg.emit(o.orchestratorID(), protocol.EventPlanUpdated, protocol.PlanUpdated{Items: next})
	}
}
