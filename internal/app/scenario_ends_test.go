package app

import (
	"encoding/json"
	"testing"

	"github.com/aguinelo/dcode/internal/protocol"
)

// rn7Ends are the names a child's end may have (.r RN-7): nothing else ends one.
var rn7Ends = map[string]bool{
	protocol.StopDone: true, protocol.StopIncomplete: true, protocol.StopMaxIterations: true,
	protocol.StopError: true, protocol.StopInterrupted: true, "cancelled": true, "deadline": true,
	"budget": true, protocol.StopDaemonRestart: true,
}

// unnamedEnds is every end of a child's turn, in the log or in the list,
// that has no RN-7 name.
func (s *scenario) unnamedEnds() []string {
	s.t.Helper()
	var bad []string
	for _, d := range s.decisions() {
		if d.Kind != "child.completed" && d.Kind != "child.state" && d.Kind != "child.closed" {
			continue
		}
		var data struct {
			Reason string `json:"reason"`
		}
		if len(d.Data) > 0 && json.Unmarshal(d.Data, &data) == nil && data.Reason != "" && !rn7Ends[data.Reason] {
			bad = append(bad, d.Kind+" #"+tail(d.Child)+": "+data.Reason)
		}
	}
	for _, c := range s.children() {
		if c.Reason != "" && !rn7Ends[c.Reason] {
			bad = append(bad, "the list's "+c.Title+": "+c.Reason)
		}
	}
	return bad
}

// TestScenarioEveryEndOfAChildsTurnIsNamed: a child that ends done, one the
// person closes in the middle of a turn, and one a daemon restart cuts — each
// end, in the log and in the list, has one of RN-7's names.
func TestScenarioEveryEndOfAChildsTurnIsNamed(t *testing.T) {
	s := newScenario(t, scenarioOptions{forge: "local"})
	never := make(chan struct{})
	defer close(never)
	s.model.orchestrator(dispatch(item("docs", "docs"), item("closed", "closed"), item("cut", "cut")), say("Three."))
	s.model.child("docs", writes("docs/a.md", "docs\n"), say("Done."))
	s.model.child("closed", writes("closed/a.md", "half\n"), say("unreachable").heldUntil(never))
	s.model.child("cut", writes("cut/a.md", "half\n"), say("unreachable").heldUntil(never))
	s.model.reviewer("docs", approve("fine"))
	s.open()
	s.ask("Three themes.")

	s.waitFor("three children were not dispatched", func() bool { return len(s.children()) == 3 })
	s.waitQuestion(s.kid("docs").ID)
	s.waitFor("the held turns never reached the model", func() bool {
		return s.model.requests("child/closed") == 2 && s.model.requests("child/cut") == 2
	})
	closed := s.kid("closed")
	s.closeChild(closed.ID)
	s.waitState("closed", "closed")
	s.restart()
	cut := s.waitState("cut", "failed")

	if k := s.kid("docs"); k.Reason != protocol.StopDone {
		t.Errorf("the child that finished ended %q, want done", k.Reason)
	}
	if k := s.kid("closed"); k.Reason != "cancelled" {
		t.Errorf("the child the person closed ended %q, want cancelled", k.Reason)
	}
	if cut.Reason != protocol.StopDaemonRestart {
		t.Errorf("the child the restart cut ended %q, want daemon_restart", cut.Reason)
	}
	if bad := s.unnamedEnds(); len(bad) != 0 {
		t.Errorf("ends without an RN-7 name: %v\n%s", bad, s.logText())
	}
	s.inOrder(closed.ID, "dispatch", "child.closed")
	s.inOrder("", "child.closed", "daemon.restart", "child.state")
}
