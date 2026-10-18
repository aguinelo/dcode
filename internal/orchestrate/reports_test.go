package orchestrate

import (
	"strings"
	"testing"
	"time"

	"github.com/aguinelo/dcode/internal/protocol"
)

// Spec: orchestration .i step 7 — two children that finish inside the
// debounce become ONE orchestrator turn with both reports, started with
// Origin children.
func TestTwoChildrenInsideTheDebounceBecomeOneTurnWithBothReports(t *testing.T) {
	p := openWith(t, map[string]string{"orchestrator.report_debounce": "300ms"})
	a := p.one(t, item("a", "a"))
	b := p.one(t, item("b", "b"))
	p.end(t, a.ID, protocol.StopError, "It broke.")
	p.end(t, b.ID, protocol.StopMaxIterations, "Out of steps.")
	if n := len(p.reportTurns()); n != 0 {
		t.Fatalf("a report turn started inside the debounce: %d", n)
	}
	waitFor(t, "the report turn", func() bool { return len(p.reportTurns()) > 0 })
	time.Sleep(400 * time.Millisecond)
	turns := p.reportTurns()
	if len(turns) != 1 {
		t.Fatalf("%d report turns, want one: %q", len(turns), turns)
	}
	orq := p.o.orchestratorID()
	if !strings.HasPrefix(turns[0], orq+": ") || !strings.Contains(turns[0], a.ID) || !strings.Contains(turns[0], b.ID) {
		t.Errorf("the report turn = %q", turns[0])
	}
	if pending := p.o.snapshot().PendingReports; len(pending) != 0 {
		t.Errorf("the inbox still holds %q", pending)
	}
}

// Spec: orchestration .i step 7 — a report during a turn of the person's
// enters only at the start of the next turn, never in the middle.
func TestAReportDuringATurnWaitsForThatTurnToEnd(t *testing.T) {
	p := openWith(t, map[string]string{"orchestrator.report_debounce": "0s"})
	orq := p.o.orchestratorID()
	c := p.one(t, item("a", "a"))
	p.sp.mu.Lock()
	p.sp.busy = map[string]bool{orq: true}
	p.sp.mu.Unlock()

	p.end(t, c.ID, protocol.StopError, "It broke.")
	time.Sleep(150 * time.Millisecond)
	if n := len(p.reportTurns()); n != 0 {
		t.Fatalf("a report entered a running turn: %d", n)
	}
	if len(p.o.snapshot().PendingReports) != 1 {
		t.Fatalf("the report is not waiting in the inbox: %q", p.o.snapshot().PendingReports)
	}

	p.sp.mu.Lock()
	p.sp.busy = nil
	p.sp.mu.Unlock()
	p.reg.Observe(orq, event(t, protocol.EventTurnCompleted, protocol.TurnCompleted{TurnID: "person", Reason: protocol.StopDone}))
	waitFor(t, "the report turn after the person's", func() bool { return len(p.reportTurns()) == 1 })
}

// Spec: orchestration .p §8 — a report pending when the daemon stops is
// delivered after the orchestrator is reopened.
func TestAReportPendingAtARestartIsDeliveredAfterTheReopen(t *testing.T) {
	p := openWith(t, map[string]string{"orchestrator.report_debounce": "0s"})
	orq := p.o.orchestratorID()
	c := p.one(t, item("a", "a"))
	p.sp.mu.Lock()
	p.sp.busy = map[string]bool{orq: true}
	p.sp.mu.Unlock()
	p.end(t, c.ID, protocol.StopError, "It broke.")
	time.Sleep(50 * time.Millisecond)
	p.reg.Close()

	p.sp.mu.Lock()
	p.sp.busy = nil
	p.sp.mu.Unlock()
	again := p.registry(t, map[string]string{"orchestrator.report_debounce": "0s"})
	if errs := again.Reload(t.Context()); len(errs) != 0 {
		t.Fatalf("Reload: %v", errs)
	}
	waitFor(t, "the pending report after the restart", func() bool { return len(p.reportTurns()) == 1 })
	if turns := p.reportTurns(); !strings.Contains(turns[0], c.ID) {
		t.Errorf("the report turn = %q", turns[0])
	}
}

// Spec: orchestration .p §10 — the person's answer to a question is a report
// to the orchestrator too.
func TestAnAnsweredQuestionIsReported(t *testing.T) {
	p := openWith(t, map[string]string{"orchestrator.report_debounce": "1m"})
	p.o.reachCeiling("the ceiling, for the test")
	qs := p.o.Questions()
	if len(qs) != 1 {
		t.Fatalf("questions = %+v", qs)
	}
	if _, err := p.o.Answer(qs[0].ID, "continue", ""); err != nil {
		t.Fatal(err)
	}
	if pending := strings.Join(p.o.snapshot().PendingReports, "\n"); !strings.Contains(pending, `answered question q1 (ceiling) with "continue"`) {
		t.Errorf("reports = %q", pending)
	}
}

// Spec: orchestration .p §10 — the report is the harness's text, in English,
// a new turn: it names what to do next, and never a step the harness runs.
func TestTheReportTurnIsTheHarnessesText(t *testing.T) {
	text := reportText([]string{"Child x: it failed."})
	for _, want := range []string{"Reports from your children", "- Child x: it failed.", "Say why"} {
		if !strings.Contains(text, want) {
			t.Errorf("report text lacks %q:\n%s", want, text)
		}
	}
}
