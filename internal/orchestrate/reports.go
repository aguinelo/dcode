package orchestrate

import (
	"fmt"
	"strings"
	"time"

	"github.com/aguinelo/dcode/internal/protocol"
)

// The orchestrator's inbox: what its children's pipelines ended in, carried to
// it as one turn of its own. A report never enters the middle of a turn and
// never rewrites what the model already read: it is a new turn, appended at
// the end of the history, started by the harness with Origin children
// (ADR-03).
//
// Spec: docs/specs/architecture/orchestration/202610132000-orchestration.p.spec.md §4, §8, §10.

// afterFunc is time.AfterFunc: the debounce's clock.
var afterFunc = time.AfterFunc

// Report puts one report in the inbox — on disk first, so a restart keeps it
// — and starts the debounce that turns the inbox into a turn. The merge step
// reports through here too.
func (o *Orchestrator) Report(text string) {
	text = strings.TrimSpace(text)
	if text == "" {
		return
	}
	if err := o.apply(func(st *State) { st.PendingReports = append(st.PendingReports, text) }); err != nil {
		o.sayError(protocol.CodeInternal, "a report to the orchestrator could not be kept, so it is said here instead: "+text+": "+err.Error())
		return
	}
	o.scheduleReports()
}

// reportChild is a report about one child, named by id and title.
func (o *Orchestrator) reportChild(id, what string) {
	c := o.child(id)
	o.Report(fmt.Sprintf("Child %s (%q, %s): %s", id, c.Title, c.State, what))
}

// scheduleReports starts the debounce unless one is already running: reports
// that arrive inside it join the turn the first one starts.
func (o *Orchestrator) scheduleReports() {
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.closed || o.reportTimer != nil || len(o.state.PendingReports) == 0 {
		return
	}
	o.reportTimer = afterFunc(o.limits.ReportDebounce, o.flushReports)
}

// flushReports starts one orchestrator turn with every report in the inbox.
// An orchestrator in a turn — the person's, or one already carrying reports —
// keeps them until that turn ends, when Observe schedules them again.
func (o *Orchestrator) flushReports() {
	o.mu.Lock()
	o.reportTimer = nil
	pending := append([]string(nil), o.state.PendingReports...)
	orq, closed := o.state.Project.OrchestratorID, o.closed
	o.mu.Unlock()
	if closed || len(pending) == 0 {
		return
	}
	err := o.reg.spawner.Submit(orq, reportText(pending), protocol.OriginChildren)
	if isCode(err, protocol.CodeTurnAlreadyActive) {
		return // the turn's end brings them (Observe)
	}
	if err != nil {
		o.sayError(protocol.CodeInternal, fmt.Sprintf("%d report(s) could not start the orchestrator's turn, and wait in its inbox: %v", len(pending), err))
		return
	}
	if err := o.apply(func(st *State) {
		if len(st.PendingReports) >= len(pending) {
			st.PendingReports = append([]string(nil), st.PendingReports[len(pending):]...)
		}
	}); err != nil {
		o.sayError(protocol.CodeInternal, "the reports were delivered and the inbox could not be emptied on disk; a restart may deliver them again: "+err.Error())
	}
}

// reportText is the turn the harness writes for the orchestrator, in English:
// text a model reads is behaviour (LANGUAGE.md).
func reportText(reports []string) string {
	var b strings.Builder
	b.WriteString("Reports from your children, written by the harness. Review, gate and merge run on their own " +
		"after a child's turn ends; these are their results.\n\n")
	for _, r := range reports {
		fmt.Fprintf(&b, "- %s\n", r)
	}
	b.WriteString("\nRead them and decide what comes next: steer a child, close it, dispatch more, or wait. " +
		"Say why for each choice.")
	return b.String()
}

// apply changes the state without a line of its own: what follows a decision
// already in the log, or bookkeeping that is no decision (the inbox).
func (o *Orchestrator) apply(mutate func(*State)) error {
	o.mu.Lock()
	defer o.mu.Unlock()
	st := cloneState(o.state)
	mutate(&st)
	if err := o.store.Save(st); err != nil {
		return err
	}
	o.state = st
	return nil
}

// stop ends the inbox's timer: the registry is closing.
func (o *Orchestrator) stop() {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.closed = true
	if o.reportTimer != nil {
		o.reportTimer.Stop()
		o.reportTimer = nil
	}
}
