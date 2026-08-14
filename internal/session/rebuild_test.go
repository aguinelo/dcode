package session

import (
	"encoding/json"
	"path/filepath"
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
