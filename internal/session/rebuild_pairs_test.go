package session

import (
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/aguinelo/dcode/internal/protocol"
	"github.com/aguinelo/dcode/internal/provider"
)

// An orchestrator's record, continued by an ordinary session, rebuilds into a
// conversation every dialect accepts: each call rides with its result, a call
// that never got one leaves with it, and what the orchestrator itself said
// between turns — children starting, finishing, asking — is not tool-shaped.
//
// The tools are not this session's (`delegate`, `children`, `ask`), and that is
// not a reason to drop them: a provider reads past calls without the tool
// declared, and a result kept without its call is the request it rejects.
func TestAnOrchestratorRecordRebuildsIntoPairsTheProviderAccepts(t *testing.T) {
	dir := t.TempDir()
	in := func(s string) json.RawMessage { return json.RawMessage(s) }
	recordFile(t, dir, "o1",
		event(t, 1, "o1", protocol.EventSessionCreated, protocol.Session{ID: "o1", Workspace: "/w"}),
		event(t, 2, "o1", protocol.EventTurnStarted, protocol.TurnStarted{TurnID: "t1", Text: "split the parser work"}),
		event(t, 3, "o1", protocol.EventMessageDelta, protocol.MessageDelta{TurnID: "t1", Text: "Delegating."}),
		event(t, 4, "o1", protocol.EventToolRequested, protocol.ToolRequested{TurnID: "t1", ToolCallID: "call_d1", Name: "delegate", Input: in(`{"task":"parser"}`)}),
		event(t, 5, "o1", protocol.EventToolRequested, protocol.ToolRequested{TurnID: "t1", ToolCallID: "call_c1", Name: "children", Input: in(`{}`)}),
		// Refused before it ran: the loop answers it in memory and records no
		// completion, so the replay has no result for it.
		event(t, 6, "o1", protocol.EventToolRequested, protocol.ToolRequested{TurnID: "t1", ToolCallID: "call_x1", Name: "merge", Input: in(`{}`)}),
		event(t, 7, "o1", protocol.EventChildStarted, protocol.ChildChanged{}),
		event(t, 8, "o1", protocol.EventToolCompleted, protocol.ToolCompleted{ToolCallID: "call_c1", OK: true, Output: "c1 working"}),
		event(t, 9, "o1", protocol.EventToolCompleted, protocol.ToolCompleted{ToolCallID: "call_d1", OK: true, Output: "child c1 started"}),
		event(t, 10, "o1", protocol.EventMessageDelta, protocol.MessageDelta{TurnID: "t1", Text: "c1 is on it."}),
		event(t, 11, "o1", protocol.EventTurnCompleted, protocol.TurnCompleted{TurnID: "t1"}),
		event(t, 12, "o1", protocol.EventChildCompleted, protocol.ChildCompleted{ChildID: "c1"}),
		// The report the harness starts the next turn with.
		event(t, 13, "o1", protocol.EventTurnStarted, protocol.TurnStarted{TurnID: "t2", Text: "child c1 finished: parser done"}),
		event(t, 14, "o1", protocol.EventToolRequested, protocol.ToolRequested{TurnID: "t2", ToolCallID: "call_a1", Name: "ask", Input: in(`{"question":"merge?"}`)}),
		event(t, 15, "o1", protocol.EventQuestionAsked, protocol.Question{}),
		event(t, 16, "o1", protocol.EventToolCompleted, protocol.ToolCompleted{ToolCallID: "call_a1", OK: true, Output: "yes"}),
		event(t, 17, "o1", protocol.EventMessageDelta, protocol.MessageDelta{TurnID: "t2", Text: "Merging."}),
		event(t, 18, "o1", protocol.EventTurnCompleted, protocol.TurnCompleted{TurnID: "t2"}),
	)

	got, err := Rebuild(filepath.Join(dir, "o1.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	if err := provider.CheckToolPairs(got); err != nil {
		t.Fatalf("the replay is a conversation the provider rejects: %v\n%+v", err, got)
	}
	calls := 0
	for _, m := range got {
		for _, c := range m.ToolCalls {
			calls++
			if c.ID == "call_x1" {
				t.Error("a call with no result was replayed, and the provider rejects it")
			}
		}
	}
	if calls != 3 {
		t.Errorf("replayed %d calls, want the 3 that were answered", calls)
	}
}
