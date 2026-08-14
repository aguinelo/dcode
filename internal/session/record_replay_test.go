package session

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/aguinelo/dcode/internal/protocol"
)

func fill(t *testing.T, l *EventLog, n int) {
	t.Helper()
	for i := 0; i < n; i++ {
		if _, err := l.Append(protocol.EventMessageDelta, json.RawMessage(`{"text":"x"}`)); err != nil {
			t.Fatal(err)
		}
	}
}

// The event log IS the session: a client holds one number and everything else
// lives here. Retention was memory-only, so a client away long enough got
// events_expired and the session became unreadable — for a reason that has
// nothing to do with the session and everything to do with how long it took to
// come back.
func TestAClientAwayTooLongStillGetsTheSession(t *testing.T) {
	dir := t.TempDir()
	sp, err := NewRecord(dir, "s-1")
	if err != nil {
		t.Fatal(err)
	}
	l := NewEventLog("s-1", 10, nil)
	l.SetRecord(sp)

	fill(t, l, 50)

	got, err := l.Replay(1)
	if err != nil {
		t.Fatalf("replaying from the beginning was refused: %v", err)
	}
	if len(got) != 50 {
		t.Fatalf("got %d events, want all 50 — %d were trimmed out of memory and must come from disk", len(got), 40)
	}
	for i, ev := range got {
		if ev.Seq != uint64(i+1) {
			t.Fatalf("event %d has seq %d: the replay has a gap or is out of order", i, ev.Seq)
		}
	}
}

func TestReplayFromTheMiddleSpansDiskAndMemory(t *testing.T) {
	dir := t.TempDir()
	sp, _ := NewRecord(dir, "s-2")
	l := NewEventLog("s-2", 10, nil)
	l.SetRecord(sp)
	fill(t, l, 50)

	got, err := l.Replay(35)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 16 {
		t.Fatalf("got %d events from seq 35, want 16", len(got))
	}
	if got[0].Seq != 35 {
		t.Errorf("first seq = %d, want 35", got[0].Seq)
	}
}

// Without a spill the horizon is hard, and saying so is better than answering
// with a gap the client cannot detect.
func TestWithoutASpillTheOldBehaviourStands(t *testing.T) {
	l := NewEventLog("s-3", 10, nil)
	fill(t, l, 50)

	if _, err := l.Replay(1); err == nil {
		t.Fatal("replaying below the retention horizon succeeded with nothing keeping the events")
	}
}

// Nothing is dropped if it could not be kept. Memory growing is visible and
// recoverable; a silent hole in the session is neither.
// A record that cannot be written must not take the events with it. The
// session is still a session; what is lost is the ability to read it later.
func TestARecordThatCannotBeOpenedLeavesTheSessionUnrecorded(t *testing.T) {
	dir := t.TempDir()
	// A directory where the file should be: opening it fails.
	if err := os.MkdirAll(filepath.Join(dir, "s-4.jsonl"), 0o700); err != nil {
		t.Fatal(err)
	}
	rec, err := NewRecord(dir, "s-4")
	if err == nil {
		t.Fatal("opening a record over a directory reported success")
	}
	if rec != nil {
		t.Fatal("a failed open handed back something to write to")
	}

	// The session runs anyway. Refusing to work because a file could not be
	// opened would be the audit trail holding the product hostage, and the
	// daemon says so on its log rather than swallowing it.
	l := NewEventLog("s-4", 10, nil)
	l.SetRecord(rec)
	fill(t, l, 50)

	// And it behaves exactly like a session that was never being recorded:
	// retention is a hard horizon, and asking below it is REFUSED rather than
	// answered with a gap. A refusal is something a client can act on.
	if _, err := l.Replay(1); err == nil {
		t.Fatal("replaying below the horizon of an unrecorded session succeeded")
	}
	got, err := l.Replay(41)
	if err != nil {
		t.Fatalf("replaying what memory still holds failed: %v", err)
	}
	if len(got) != 10 {
		t.Errorf("got %d events, want the 10 retention kept", len(got))
	}
}

// A large payload — a diff — exceeds a scanner's default line limit, and
// hitting it would silently truncate the replay.
func TestALargePayloadSurvivesTheRoundTrip(t *testing.T) {
	dir := t.TempDir()
	sp, _ := NewRecord(dir, "s-5")
	big := make([]byte, 200_000)
	for i := range big {
		big[i] = 'x'
	}
	payload, err := json.Marshal(map[string]string{"diff": string(big)})
	if err != nil {
		t.Fatal(err)
	}
	if err := sp.Append([]protocol.Event{{Seq: 1, Type: protocol.EventToolCompleted, Payload: payload}}); err != nil {
		t.Fatal(err)
	}
	got, err := sp.Replay(1)
	if err != nil {
		t.Fatalf("a large event could not be read back: %v", err)
	}
	if len(got) != 1 || len(got[0].Payload) < 200_000 {
		t.Fatalf("the payload was truncated: %d events, %d bytes", len(got), len(got[0].Payload))
	}
}

func TestNoDirectoryMeansNoRecord(t *testing.T) {
	sp, err := NewRecord("", "s-6")
	if err != nil {
		t.Fatal(err)
	}
	if sp != nil {
		t.Fatal("an empty directory produced a file; that is how recording is turned off")
	}
	// And every method tolerates it, so the caller needs no nil check.
	if err := sp.Append([]protocol.Event{{Seq: 1}}); err != nil {
		t.Errorf("Append on a nil record: %v", err)
	}
	if got, err := sp.Replay(1); err != nil || got != nil {
		t.Errorf("Replay on a nil record: %v, %v", got, err)
	}
	if err := sp.Close(); err != nil {
		t.Errorf("Close on a nil record: %v", err)
	}
}

// The spill exists so a client that was away can still read what it missed.
// The record outlives the session, and that is the whole point.
//
// This asserted the opposite, and the reasoning was sound for what the file
// then was: once the session is gone nobody can ask for a replay, so the file
// was garbage with a session id in its name, one per session, in a directory
// the user configured and never looks at.
//
// Every word of that holds for a spill and none of it for a record. The asking
// that matters happens AFTER the session ends — what did it do, why did it do
// that, what would we change. Deleting at close destroyed the evidence at the
// exact moment it became evidence.
//
// The growth the old test worried about is real and is answered by pruning,
// which is a policy about age and size, not by throwing everything away.
func TestTheRecordOutlivesTheSession(t *testing.T) {
	dir := t.TempDir()
	rec, err := NewRecord(dir, "s1")
	if err != nil {
		t.Fatal(err)
	}
	l := NewEventLog("s1", 2, func() time.Time { return time.Unix(0, 0) })
	l.SetRecord(rec)

	for i := 0; i < 6; i++ {
		if _, err := l.Append(protocol.EventToolCompleted, protocol.MessageDelta{Text: "x"}); err != nil {
			t.Fatal(err)
		}
	}
	l.Close()

	after, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(after) != 1 {
		t.Fatalf("the record did not survive the session: %v", names(after))
	}
	b, err := os.ReadFile(filepath.Join(dir, "s1.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	if n := strings.Count(strings.TrimSpace(string(b)), "\n") + 1; n != 6 {
		t.Errorf("the record holds %d of 6 events", n)
	}
}

// Closing twice is what a shutdown racing a delete does, and a second close
// must not turn into an error nobody can act on.
func TestClosingTwiceIsNotAnError(t *testing.T) {
	dir := t.TempDir()
	sp, err := NewRecord(dir, "s1")
	if err != nil {
		t.Fatal(err)
	}
	l := NewEventLog("s1", 2, func() time.Time { return time.Unix(0, 0) })
	l.SetRecord(sp)
	if _, err := l.Append(protocol.EventMessageDelta, protocol.MessageDelta{Text: "x"}); err != nil {
		t.Fatal(err)
	}
	l.Close()
	l.Close()
}

// A log with no spill closes exactly as it did before. The spill is optional
// and off by default, and the close path must not start depending on it.
func TestClosingALogWithoutASpillIsUnchanged(t *testing.T) {
	l := NewEventLog("s1", 2, func() time.Time { return time.Unix(0, 0) })
	if _, err := l.Append(protocol.EventMessageDelta, protocol.MessageDelta{Text: "x"}); err != nil {
		t.Fatal(err)
	}
	l.Close()
}

func names(entries []os.DirEntry) []string {
	var out []string
	for _, e := range entries {
		out = append(out, e.Name())
	}
	return out
}

// End to end through the manager, which is the path a client's DELETE takes.
// The unit above proves the log removes it; this proves the removal is actually
// reached when a session is deleted rather than sitting behind a Close nobody
// calls.
func TestRemovingASessionThroughTheManagerKeepsItsRecord(t *testing.T) {
	dir := t.TempDir()
	m := NewManager(4)
	log := NewEventLog("s1", 2, func() time.Time { return time.Unix(0, 0) })
	rec, err := NewRecord(dir, "s1")
	if err != nil {
		t.Fatal(err)
	}
	log.SetRecord(rec)
	s := New("s1", "/w", "MiniMax-M3", "workspace-write", nil, log, func() time.Time { return time.Unix(0, 0) })
	if err := m.Add(s); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 6; i++ {
		s.Emit(protocol.EventToolCompleted, protocol.MessageDelta{Text: "x"})
	}

	if err := m.Remove(s.ID); err != nil {
		t.Fatal(err)
	}
	entries, _ := os.ReadDir(dir)
	if len(entries) != 1 {
		t.Errorf("removing the session took its record with it: %v", names(entries))
	}
}
