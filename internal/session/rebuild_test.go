package session

import (
	"encoding/json"
	"path/filepath"
	"slices"
	"testing"

	ce "github.com/aguinelo/dcode/internal/contextengine"
	"github.com/aguinelo/dcode/internal/protocol"
)

// The record is the only copy of what happened, so continuing means reading it
// back into the shape the model is sent.
func TestAConversationIsRebuiltInOrder(t *testing.T) {
	dir := t.TempDir()
	recordFile(t, dir, "s1",
		event(t, 1, "s1", protocol.EventSessionCreated, protocol.Session{ID: "s1", Workspace: "/w"}),
		event(t, 2, "s1", protocol.EventTurnStarted, protocol.TurnStarted{TurnID: "t1", Text: "what does Rows do?"}),
		event(t, 3, "s1", protocol.EventMessageDelta, protocol.MessageDelta{TurnID: "t1", Text: "Let me "}),
		event(t, 4, "s1", protocol.EventMessageDelta, protocol.MessageDelta{TurnID: "t1", Text: "read it."}),
		event(t, 5, "s1", protocol.EventToolRequested, protocol.ToolRequested{TurnID: "t1", ToolCallID: "c1", Name: "read", Input: json.RawMessage(`{"path":"stats.go"}`)}),
		event(t, 6, "s1", protocol.EventToolCompleted, protocol.ToolCompleted{ToolCallID: "c1", OK: true, Output: "func Rows() int { return s.count - 1 }"}),
		event(t, 7, "s1", protocol.EventMessageDelta, protocol.MessageDelta{TurnID: "t1", Text: "It returns count minus one."}),
		event(t, 8, "s1", protocol.EventTurnCompleted, protocol.TurnCompleted{TurnID: "t1"}),
	)

	got, err := Rebuild(filepath.Join(dir, "s1.jsonl"))
	if err != nil {
		t.Fatal(err)
	}

	want := []ce.Role{ce.RoleUser, ce.RoleAssistant, ce.RoleTool, ce.RoleAssistant}
	if len(got) != len(want) {
		t.Fatalf("rebuilt %d messages, want %d: %+v", len(got), len(want), got)
	}
	for i, r := range want {
		if got[i].Role != r {
			t.Errorf("message %d is %q, want %q", i, got[i].Role, r)
		}
	}

	if got[0].Text != "what does Rows do?" {
		t.Errorf("the question is %q", got[0].Text)
	}
	// The assistant message carries its text AND the call it made, because
	// that is one message on the wire and splitting it loses which text went
	// with which call.
	if got[1].Text != "Let me read it." {
		t.Errorf("the fragments were not joined: %q", got[1].Text)
	}
	if len(got[1].ToolCalls) != 1 || got[1].ToolCalls[0].ID != "c1" {
		t.Errorf("the call did not ride with its message: %+v", got[1])
	}
	if got[2].ToolResult == nil || got[2].ToolResult.ToolCallID != "c1" {
		t.Errorf("the result is not tied to its call: %+v", got[2])
	}
	if got[3].Text != "It returns count minus one." {
		t.Errorf("the answer after the tool is %q", got[3].Text)
	}
}

// A failed call is part of the history the model reasoned from. Dropping it
// would continue a conversation the model never had.
func TestAFailedCallSurvivesTheRebuild(t *testing.T) {
	dir := t.TempDir()
	recordFile(t, dir, "s2",
		event(t, 1, "s2", protocol.EventSessionCreated, protocol.Session{ID: "s2", Workspace: "/w"}),
		event(t, 2, "s2", protocol.EventTurnStarted, protocol.TurnStarted{TurnID: "t1", Text: "edit it"}),
		event(t, 3, "s2", protocol.EventToolRequested, protocol.ToolRequested{TurnID: "t1", ToolCallID: "c1", Name: "edit"}),
		event(t, 4, "s2", protocol.EventToolCompleted, protocol.ToolCompleted{ToolCallID: "c1", OK: false, Output: "old_string appears 3 times"}),
	)

	got, err := Rebuild(filepath.Join(dir, "s2.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	for _, m := range got {
		if m.ToolResult != nil {
			if !m.ToolResult.IsError {
				t.Error("a failed call was rebuilt as a success")
			}
			return
		}
	}
	t.Fatal("the failed call vanished")
}

// Reminders were appended by the harness, not typed. Rebuilding them as the
// user's words would put the product's voice in the person's mouth.
func TestRemindersAreNotRebuiltAsQuestions(t *testing.T) {
	dir := t.TempDir()
	recordFile(t, dir, "s3",
		event(t, 1, "s3", protocol.EventSessionCreated, protocol.Session{ID: "s3", Workspace: "/w"}),
		event(t, 2, "s3", protocol.EventTurnStarted, protocol.TurnStarted{TurnID: "t1", Text: "go"}),
	)
	got, err := Rebuild(filepath.Join(dir, "s3.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Text != "go" {
		t.Errorf("rebuilt %+v", got)
	}
}

// Nothing to continue is not an error: a record with no turn is a session that
// was opened and left.
func TestRebuildingAnEmptySessionIsQuiet(t *testing.T) {
	dir := t.TempDir()
	recordFile(t, dir, "s4",
		event(t, 1, "s4", protocol.EventSessionCreated, protocol.Session{ID: "s4", Workspace: "/w"}))

	got, err := Rebuild(filepath.Join(dir, "s4.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Errorf("rebuilt %+v from a session with no turns", got)
	}
}

// An interrupted turn leaves a call with no result. Sending that to a model is
// a malformed conversation — most providers reject it outright — so the
// dangling call goes with the message it rode in on.
func TestACallWithNoResultIsDropped(t *testing.T) {
	dir := t.TempDir()
	recordFile(t, dir, "s5",
		event(t, 1, "s5", protocol.EventSessionCreated, protocol.Session{ID: "s5", Workspace: "/w"}),
		event(t, 2, "s5", protocol.EventTurnStarted, protocol.TurnStarted{TurnID: "t1", Text: "start"}),
		event(t, 3, "s5", protocol.EventMessageDelta, protocol.MessageDelta{TurnID: "t1", Text: "working"}),
		event(t, 4, "s5", protocol.EventToolRequested, protocol.ToolRequested{TurnID: "t1", ToolCallID: "c1", Name: "read"}),
	)

	got, err := Rebuild(filepath.Join(dir, "s5.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	for _, m := range got {
		for _, c := range m.ToolCalls {
			if c.ID == "c1" {
				t.Error("a call with no result survived; the model would be sent a conversation it cannot answer")
			}
		}
	}
	// What the model said still counts.
	if len(got) != 2 || got[1].Text != "working" {
		t.Errorf("rebuilt %+v", got)
	}
}

// A batch is several calls in one assistant message, and the loop runs
// independent ones together, so they finish in whatever order they finish.
// The rebuild once let the first result close the message: the calls answered
// after it were lost and their results kept, and a result answering no call is
// a request every provider rejects. A resumed session with a parallel batch in
// its record failed on the first turn after the resume.
func TestABatchOfCallsIsRebuiltWhole(t *testing.T) {
	for _, tc := range []struct {
		name     string
		finished []string
	}{
		{"finished in the order they were made", []string{"c1", "c2"}},
		{"finished the other way round", []string{"c2", "c1"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			evs := []protocol.Event{
				event(t, 1, "s6", protocol.EventSessionCreated, protocol.Session{ID: "s6", Workspace: "/w"}),
				event(t, 2, "s6", protocol.EventTurnStarted, protocol.TurnStarted{TurnID: "t1", Text: "compare them"}),
				event(t, 3, "s6", protocol.EventMessageDelta, protocol.MessageDelta{TurnID: "t1", Text: "Reading both."}),
				event(t, 4, "s6", protocol.EventToolRequested, protocol.ToolRequested{TurnID: "t1", ToolCallID: "c1", Name: "read", Input: json.RawMessage(`{"path":"a.go"}`)}),
				event(t, 5, "s6", protocol.EventToolRequested, protocol.ToolRequested{TurnID: "t1", ToolCallID: "c2", Name: "read", Input: json.RawMessage(`{"path":"b.go"}`)}),
			}
			for i, id := range tc.finished {
				evs = append(evs, event(t, uint64(6+i), "s6", protocol.EventToolCompleted, protocol.ToolCompleted{ToolCallID: id, OK: true, Output: "contents of " + id}))
			}
			dir := t.TempDir()
			recordFile(t, dir, "s6", evs...)

			got, err := Rebuild(filepath.Join(dir, "s6.jsonl"))
			if err != nil {
				t.Fatal(err)
			}

			// The question, the one message that made both calls, and a result
			// for each.
			if len(got) != 4 {
				t.Fatalf("rebuilt %d messages, want 4: %+v", len(got), got)
			}
			made := got[1]
			if made.Role != ce.RoleAssistant || made.Text != "Reading both." {
				t.Errorf("message 1 is %q %q, want the assistant's text riding with its calls", made.Role, made.Text)
			}
			if ids := callIDs(made.ToolCalls); !slices.Equal(ids, []string{"c1", "c2"}) {
				t.Errorf("the assistant message carries calls %v, want both it made: [c1 c2]", ids)
			}
			results := map[string]int{}
			for _, m := range got[2:] {
				if m.Role != ce.RoleTool || m.ToolResult == nil {
					t.Errorf("after the calls came %+v, want their results", m)
					continue
				}
				id := m.ToolResult.ToolCallID
				if !slices.Contains(callIDs(made.ToolCalls), id) {
					t.Errorf("the result for %s answers no call in the message before it; a provider rejects that request", id)
				}
				results[id]++
			}
			for _, id := range []string{"c1", "c2"} {
				if results[id] != 1 {
					t.Errorf("call %s has %d results, want exactly one", id, results[id])
				}
			}
		})
	}
}

// The live session appends a batch's results at the index the model emitted
// each call at, never in the order they finished (RN-3 of the agent loop); the
// record holds them in the order they finished. Rebuilding in that order would
// continue a conversation the model was never sent, and make the rebuilt
// history depend on which read happened to win a race.
func TestABatchIsRebuiltInTheOrderItsCallsWereMade(t *testing.T) {
	dir := t.TempDir()
	recordFile(t, dir, "s7",
		event(t, 1, "s7", protocol.EventSessionCreated, protocol.Session{ID: "s7", Workspace: "/w"}),
		event(t, 2, "s7", protocol.EventTurnStarted, protocol.TurnStarted{TurnID: "t1", Text: "read all three"}),
		event(t, 3, "s7", protocol.EventToolRequested, protocol.ToolRequested{TurnID: "t1", ToolCallID: "c1", Name: "read"}),
		event(t, 4, "s7", protocol.EventToolRequested, protocol.ToolRequested{TurnID: "t1", ToolCallID: "c2", Name: "read"}),
		event(t, 5, "s7", protocol.EventToolRequested, protocol.ToolRequested{TurnID: "t1", ToolCallID: "c3", Name: "read"}),
		event(t, 6, "s7", protocol.EventToolCompleted, protocol.ToolCompleted{ToolCallID: "c3", OK: true}),
		event(t, 7, "s7", protocol.EventToolCompleted, protocol.ToolCompleted{ToolCallID: "c1", OK: true}),
		event(t, 8, "s7", protocol.EventToolCompleted, protocol.ToolCompleted{ToolCallID: "c2", OK: true}),
	)

	got, err := Rebuild(filepath.Join(dir, "s7.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	var order []string
	for _, m := range got {
		if m.ToolResult != nil {
			order = append(order, m.ToolResult.ToolCallID)
		}
	}
	if want := []string{"c1", "c2", "c3"}; !slices.Equal(order, want) {
		t.Errorf("the results came back as %v, want the order the calls were made in, %v, not the order they finished", order, want)
	}
}

// An interrupt can land in the middle of a batch: the calls already running
// finish, and the rest never do. What finished is history the model reasoned
// from and stays; what did not is dropped alone, not with its siblings.
func TestABatchDropsOnlyTheCallThatGotNoResult(t *testing.T) {
	dir := t.TempDir()
	recordFile(t, dir, "s8",
		event(t, 1, "s8", protocol.EventSessionCreated, protocol.Session{ID: "s8", Workspace: "/w"}),
		event(t, 2, "s8", protocol.EventTurnStarted, protocol.TurnStarted{TurnID: "t1", Text: "read both"}),
		event(t, 3, "s8", protocol.EventToolRequested, protocol.ToolRequested{TurnID: "t1", ToolCallID: "c1", Name: "read"}),
		event(t, 4, "s8", protocol.EventToolRequested, protocol.ToolRequested{TurnID: "t1", ToolCallID: "c2", Name: "read"}),
		event(t, 5, "s8", protocol.EventToolCompleted, protocol.ToolCompleted{ToolCallID: "c2", OK: true}),
		event(t, 6, "s8", protocol.EventTurnStarted, protocol.TurnStarted{TurnID: "t2", Text: "go on"}),
	)

	got, err := Rebuild(filepath.Join(dir, "s8.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 4 {
		t.Fatalf("rebuilt %d messages, want the question, the call that finished, its result and the next question: %+v", len(got), got)
	}
	if ids := callIDs(got[1].ToolCalls); !slices.Equal(ids, []string{"c2"}) {
		t.Errorf("the assistant message carries calls %v, want only the one that finished: [c2]", ids)
	}
	if got[2].ToolResult == nil || got[2].ToolResult.ToolCallID != "c2" {
		t.Errorf("message 2 is %+v, want the result of c2", got[2])
	}
	if got[3].Role != ce.RoleUser || got[3].Text != "go on" {
		t.Errorf("message 3 is %q %q, want the next question", got[3].Role, got[3].Text)
	}
}

func callIDs(calls []ce.ToolCall) []string {
	var ids []string
	for _, c := range calls {
		ids = append(ids, c.ID)
	}
	return ids
}
