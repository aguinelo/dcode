package protocol

import (
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"testing"
	"time"
)

var update = flag.Bool("update", false, "rewrite the golden files")

// golden compares against a recorded shape, or rewrites it under -update.
//
// The point is not that the bytes are pretty. It is that the wire format of a
// contract marked `stable` cannot drift by accident: a renamed field, a tag
// that stops being omitempty, a type that becomes a pointer — each is a silent
// break for every client, and each is invisible to a test that only round-trips
// a value through the same code that produced it.
func golden(t *testing.T, name string, v any) {
	t.Helper()
	got, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	got = append(got, '\n')

	path := filepath.Join("testdata", name+".json")
	if *update {
		if err := os.MkdirAll("testdata", 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, got, 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("%s: %v\nRun `go test ./internal/protocol -update` to record it.", path, err)
	}
	if string(got) != string(want) {
		t.Errorf("%s no longer matches its recorded shape.\n\nwant:\n%s\ngot:\n%s\n\n"+
			"This type is part of a contract with third parties. If the change is intended, "+
			"re-record with -update and say so in a changelog entry.", path, want, got)
	}
}

// A fixed instant, because a clock in a golden file breaks the test on every
// run — the trap the .i spec names by hand.
var at = time.Date(2026, 8, 11, 12, 0, 0, 0, time.UTC)

func TestGoldenEvent(t *testing.T) {
	golden(t, "event", Event{
		Seq: 42, Type: EventToolCompleted, At: at,
		Payload: json.RawMessage(`{"tool":"read"}`),
	})
}

func TestGoldenSession(t *testing.T) {
	golden(t, "session", Session{
		ID: "s-1", Workspace: "/w", Model: "MiniMax-M3",
		State: SessionStateIdle, SandboxMode: "workspace-write",
		ContextWindow: 1000000, CreatedAt: at,
	})
}

func TestGoldenApprovalRequest(t *testing.T) {
	golden(t, "approval_request", ApprovalRequest{
		ApprovalID: "a-1", TurnID: "t-1", ToolCallID: "c-1",
		Tool: "bash", Command: "rm -rf build",
		BoundaryCrossed: "workspace-write", Reason: "matches a confirm rule",
		Rule: "rm -rf*", ExpiresAt: at,
	})
}

func TestGoldenTurnCompleted(t *testing.T) {
	golden(t, "turn_completed", TurnCompleted{
		TurnID: "t-1", Reason: StopDone,
		Usage: &Usage{InputTokens: 100, OutputTokens: 20, CacheReadTokens: 80},
		Completion: &Completion{
			Verification: "passed", Met: []string{"tests"},
			TouchedProtected: []string{"a_test.go"},
		},
	})
}

// The absent halves matter as much as the present ones: `omitempty` on Usage
// and Completion is what lets a client tell "no tokens" from "unknown", and
// dropping it would be a silent break.
func TestGoldenTurnCompletedMinimal(t *testing.T) {
	golden(t, "turn_completed_minimal", TurnCompleted{TurnID: "t-1", Reason: StopInterrupted})
}

func TestGoldenError(t *testing.T) {
	golden(t, "error", Error{Code: CodeWorkspaceInvalid, Message: "not a directory"})
}

func TestGoldenCreateSessionRequest(t *testing.T) {
	golden(t, "create_session_request", CreateSessionRequest{
		Workspace: "/w", Model: "MiniMax-M3", SandboxMode: "read-only",
	})
}
