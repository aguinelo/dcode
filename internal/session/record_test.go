package session

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/aguinelo/dcode/internal/protocol"
)

func readLines(t *testing.T, path string) []protocol.Event {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var out []protocol.Event
	for _, line := range strings.Split(strings.TrimSpace(string(b)), "\n") {
		if line == "" {
			continue
		}
		var ev protocol.Event
		if err := json.Unmarshal([]byte(line), &ev); err != nil {
			t.Fatalf("line is not an event: %v\n%s", err, line)
		}
		out = append(out, ev)
	}
	return out
}

// The record is the session, not the overflow.
//
// The file existed and held only what retention pushed out of memory, so a
// session that fit in memory — which is nearly all of them — left nothing at
// all. And Close deleted it, on the reasoning that once the session is gone
// nobody can ask for a replay. True for replay; the point of a record is the
// asking that happens afterwards.
func TestEveryEventReachesTheRecord(t *testing.T) {
	dir := t.TempDir()
	rec, err := NewRecord(dir, "s1")
	if err != nil {
		t.Fatal(err)
	}
	log := NewEventLog("s1", 10000, fixedClock())
	log.SetRecord(rec)

	for i := 0; i < 5; i++ {
		if _, err := log.Append(protocol.EventToolRequested, map[string]int{"n": i}); err != nil {
			t.Fatal(err)
		}
	}
	log.Close()

	got := readLines(t, filepath.Join(dir, "s1.jsonl"))
	if len(got) != 5 {
		t.Fatalf("recorded %d events, want 5 — retention never trimmed, and that is the normal case", len(got))
	}
	for i, ev := range got {
		if ev.Seq != uint64(i+1) {
			t.Errorf("event %d has seq %d", i, ev.Seq)
		}
	}
}

// Closing keeps it. A record deleted when the session ends is a record of
// nothing anyone can read.
func TestTheRecordSurvivesTheSession(t *testing.T) {
	dir := t.TempDir()
	rec, err := NewRecord(dir, "s2")
	if err != nil {
		t.Fatal(err)
	}
	log := NewEventLog("s2", 2, fixedClock())
	log.SetRecord(rec)
	for i := 0; i < 6; i++ {
		if _, err := log.Append(protocol.EventToolCompleted, map[string]int{"n": i}); err != nil {
			t.Fatal(err)
		}
	}
	log.Close()

	if got := readLines(t, filepath.Join(dir, "s2.jsonl")); len(got) != 6 {
		t.Fatalf("recorded %d of 6 events after a close", len(got))
	}
}

// Trimming must not write a second copy. The record already has them, and a
// duplicated sequence number is a replay that returns the same event twice.
func TestTrimmingDoesNotRecordTwice(t *testing.T) {
	dir := t.TempDir()
	rec, err := NewRecord(dir, "s3")
	if err != nil {
		t.Fatal(err)
	}
	log := NewEventLog("s3", 2, fixedClock())
	log.SetRecord(rec)
	for i := 0; i < 5; i++ {
		if _, err := log.Append(protocol.EventMessageDelta, map[string]int{"n": i}); err != nil {
			t.Fatal(err)
		}
	}
	log.Close()

	got := readLines(t, filepath.Join(dir, "s3.jsonl"))
	seen := map[uint64]int{}
	for _, ev := range got {
		seen[ev.Seq]++
	}
	for seq, n := range seen {
		if n > 1 {
			t.Errorf("seq %d recorded %d times", seq, n)
		}
	}
	if len(got) != 5 {
		t.Errorf("recorded %d events, want 5", len(got))
	}
}

// Replay still reads what memory no longer holds, which is what the file did
// before and must keep doing.
func TestReplayReadsBelowWhatMemoryKept(t *testing.T) {
	dir := t.TempDir()
	rec, err := NewRecord(dir, "s4")
	if err != nil {
		t.Fatal(err)
	}
	log := NewEventLog("s4", 2, fixedClock())
	log.SetRecord(rec)
	for i := 0; i < 6; i++ {
		if _, err := log.Append(protocol.EventToolRequested, map[string]int{"n": i}); err != nil {
			t.Fatal(err)
		}
	}

	got, err := log.Replay(1)
	if err != nil {
		t.Fatalf("replay from the beginning failed: %v", err)
	}
	if len(got) != 6 {
		t.Errorf("replayed %d events, want all 6", len(got))
	}
}

// Anything but a text delta is flushed as it happens. A crash mid-turn must
// not cost the tool calls and approvals that led up to it, and those are the
// low-frequency events — flushing them costs almost nothing.
func TestTheRecordIsDurableAtToolGranularity(t *testing.T) {
	dir := t.TempDir()
	rec, err := NewRecord(dir, "s5")
	if err != nil {
		t.Fatal(err)
	}
	log := NewEventLog("s5", 10000, fixedClock())
	log.SetRecord(rec)

	if _, err := log.Append(protocol.EventToolRequested, map[string]string{"tool": "read"}); err != nil {
		t.Fatal(err)
	}
	// No Close: this is the crash case.
	if got := readLines(t, filepath.Join(dir, "s5.jsonl")); len(got) != 1 {
		t.Fatalf("a tool call was still in the buffer when the process died: %d", len(got))
	}
}
