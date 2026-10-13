package session

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/aguinelo/dcode/internal/protocol"
)

func writeHistoryRecord(t *testing.T, dir, id string, evs ...protocol.Event) {
	t.Helper()
	var b strings.Builder
	for _, e := range evs {
		raw, _ := json.Marshal(e)
		b.Write(raw)
		b.WriteByte('\n')
	}
	if err := os.WriteFile(filepath.Join(dir, id+".jsonl"), []byte(b.String()), 0o600); err != nil {
		t.Fatal(err)
	}
}

func asked(seq uint64, id, text string) protocol.Event {
	raw, _ := json.Marshal(protocol.TurnStarted{TurnID: id, Text: text})
	return protocol.Event{Seq: seq, SessionID: "other", Type: protocol.EventTurnStarted, Payload: raw}
}

// A conversation that ended is read from its record, the whole chain of it,
// numbered as one sequence under its own id — the way a session continuing it
// would replay it, without loading anything.
//
// The window opened a recorded conversation and showed nothing, because the
// only route for events served loaded sessions: the conversation looked lost.
func TestARecordedConversationIsReadWholeAndInOneSequence(t *testing.T) {
	dir := t.TempDir()
	marker, _ := json.Marshal(protocol.SessionResumed{SourceID: "aaa", Turns: 1})
	writeHistoryRecord(t, dir, "aaa", asked(1, "t1", "primeira"))
	writeHistoryRecord(t, dir, "bbb",
		protocol.Event{Seq: 1, Type: protocol.EventSessionResumed, Payload: marker},
		asked(2, "t2", "segunda"))

	evs, err := History(dir, "bbb")
	if err != nil {
		t.Fatal(err)
	}
	var texts []string
	for i, e := range evs {
		if e.Seq != uint64(i+1) || e.SessionID != "bbb" {
			t.Errorf("event %d is numbered %d under %q, want %d under bbb", i, e.Seq, e.SessionID, i+1)
		}
		var d protocol.TurnStarted
		_ = json.Unmarshal(e.Payload, &d)
		texts = append(texts, d.Text)
	}
	if got := strings.Join(texts, ","); got != "primeira,segunda" {
		t.Errorf("the conversation reads %q, want primeira,segunda", got)
	}
}

// An id names a record inside the directory and nothing else. Read as a path,
// `../x` would answer any `.jsonl` the daemon's user can open.
func TestAnIdThatLeavesTheRecordDirectoryIsRefused(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "sessions")
	if err := os.Mkdir(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	writeHistoryRecord(t, root, "secret", asked(1, "t", "não devia sair"))

	for _, id := range []string{"", ".", "..", "../secret", "a/b", `a\b`, "/etc/passwd", "secret.jsonl", strings.Repeat("a", 200)} {
		evs, err := History(dir, id)
		if err == nil || len(evs) != 0 {
			t.Errorf("History(%q) = %d events, %v; want a refusal", id, len(evs), err)
			continue
		}
		if pe, ok := protocol.AsError(err); !ok || pe.Code != protocol.CodeInvalidInput {
			t.Errorf("History(%q) refused with %v, want %s", id, err, protocol.CodeInvalidInput)
		}
	}
}

// A conversation with no record says so, rather than reading as one that is
// empty.
func TestAMissingRecordIsNotAnEmptyConversation(t *testing.T) {
	_, err := History(t.TempDir(), "0199a1b2c3d00000000")
	if !errors.Is(err, ErrNoSuchSession) {
		t.Errorf("got %v, want ErrNoSuchSession", err)
	}
}

// A marker is data read from disk, and the id it names is held to the same
// rule as one that came over the wire.
func TestAMarkerThatLeavesTheRecordDirectoryIsNotFollowed(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "sessions")
	if err := os.Mkdir(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	writeHistoryRecord(t, root, "secret", asked(1, "t", "não devia sair"))
	marker, _ := json.Marshal(protocol.SessionResumed{SourceID: "../secret"})
	writeHistoryRecord(t, dir, "ccc",
		protocol.Event{Seq: 1, Type: protocol.EventSessionResumed, Payload: marker},
		asked(2, "t2", "minha"))

	evs, err := History(dir, "ccc")
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range evs {
		if strings.Contains(string(e.Payload), "não devia sair") {
			t.Error("a marker pointing outside the directory was followed")
		}
	}
}
