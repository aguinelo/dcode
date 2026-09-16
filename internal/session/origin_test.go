package session

import (
	"path/filepath"
	"testing"

	"github.com/aguinelo/dcode/internal/protocol"
)

// Origin reads the whole bundle a record opened with, not only the model
// name — a resume needs the family, transport and base URL to reconnect to
// the same endpoint, not just a string that means nothing without them.
func TestOriginReadsTheWholeBundleTheRecordOpenedWith(t *testing.T) {
	dir := t.TempDir()
	recordFile(t, dir, "old",
		event(t, 1, "old", protocol.EventSessionCreated, protocol.Session{
			ID: "old", Workspace: "/w", Model: "qwen3.5-9b",
			Family: "generic", Transport: "openai", BaseURL: "http://192.168.0.149:1234/v1",
			ContextWindow: 32000,
		}),
		event(t, 2, "old", protocol.EventTurnStarted, protocol.TurnStarted{TurnID: "t1", Text: "hi"}),
	)

	got, err := Origin(filepath.Join(dir, "old.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	if got.Model != "qwen3.5-9b" || got.Family != "generic" || got.Transport != "openai" ||
		got.BaseURL != "http://192.168.0.149:1234/v1" || got.ContextWindow != 32000 {
		t.Errorf("got %+v", got)
	}
}

// A path that is not a record answers the same way Browse and Rebuild do for
// one — an error naming the problem, not a zero-value bundle a caller could
// mistake for "no model set".
func TestOriginOfAMissingRecordIsAnError(t *testing.T) {
	if _, err := Origin(filepath.Join(t.TempDir(), "never-existed.jsonl")); err == nil {
		t.Fatal("reading a missing record reported success")
	}
}

// A file with no session.created — corrupt, or truncated before its first
// line landed — is the same failure as a missing one: nothing this resume
// could safely restore from.
func TestOriginOfAFileWithNoSessionCreatedIsAnError(t *testing.T) {
	dir := t.TempDir()
	recordFile(t, dir, "headless",
		event(t, 2, "headless", protocol.EventTurnStarted, protocol.TurnStarted{TurnID: "t1", Text: "hi"}),
	)

	if _, err := Origin(filepath.Join(dir, "headless.jsonl")); err == nil {
		t.Fatal("a record with no session.created reported success")
	}
}
