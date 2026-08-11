package session

import (
	"encoding/json"
	"os"
	"path/filepath"
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
	sp, err := NewSpill(dir, "s-1")
	if err != nil {
		t.Fatal(err)
	}
	l := NewEventLog("s-1", 10, nil)
	l.SetSpill(sp)

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
	sp, _ := NewSpill(dir, "s-2")
	l := NewEventLog("s-2", 10, nil)
	l.SetSpill(sp)
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
func TestAFailedSpillKeepsTheEventsInMemory(t *testing.T) {
	dir := t.TempDir()
	sp, _ := NewSpill(dir, "s-4")
	// A directory where the file should be: every write fails.
	if err := os.MkdirAll(filepath.Join(dir, "s-4.events.jsonl"), 0o700); err != nil {
		t.Fatal(err)
	}
	l := NewEventLog("s-4", 10, nil)
	l.SetSpill(sp)
	fill(t, l, 50)

	got, err := l.Replay(1)
	if err != nil {
		t.Fatalf("the events were dropped although they could not be kept: %v", err)
	}
	if len(got) != 50 {
		t.Fatalf("got %d, want all 50 still in memory", len(got))
	}
}

// A large payload — a diff — exceeds a scanner's default line limit, and
// hitting it would silently truncate the replay.
func TestALargePayloadSurvivesTheRoundTrip(t *testing.T) {
	dir := t.TempDir()
	sp, _ := NewSpill(dir, "s-5")
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

func TestNoDirectoryMeansNoSpill(t *testing.T) {
	sp, err := NewSpill("", "s-6")
	if err != nil {
		t.Fatal(err)
	}
	if sp != nil {
		t.Fatal("an empty directory produced a spill file")
	}
	// And every method tolerates it, so the caller needs no nil check.
	if err := sp.Append([]protocol.Event{{Seq: 1}}); err != nil {
		t.Errorf("Append on a nil spill: %v", err)
	}
	if got, err := sp.Replay(1); err != nil || got != nil {
		t.Errorf("Replay on a nil spill: %v, %v", got, err)
	}
	if err := sp.Remove(); err != nil {
		t.Errorf("Remove on a nil spill: %v", err)
	}
}

// The spill exists so a client that was away can still read what it missed.
// Once the session is gone, nobody can ask for those events — the id is not
// resolvable and there is no route to it — so the file is garbage with a
// session id in its name.
//
// Nothing removed it. One file per session, kept forever, in a directory the
// user configured and never looks at, growing for exactly as long as the daemon
// is useful. The method to delete it existed and had no caller.
func TestClosingASessionTakesItsSpillWithIt(t *testing.T) {
	dir := t.TempDir()
	sp, err := NewSpill(dir, "s1")
	if err != nil {
		t.Fatal(err)
	}
	l := NewEventLog("s1", 2, func() time.Time { return time.Unix(0, 0) })
	l.SetSpill(sp)

	// Enough events that retention trims and something reaches the file.
	for i := 0; i < 6; i++ {
		if _, err := l.Append(protocol.EventMessageDelta, protocol.MessageDelta{Text: "x"}); err != nil {
			t.Fatal(err)
		}
	}
	before, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(before) != 1 {
		t.Fatalf("expected the spill to exist before closing, found %d entries", len(before))
	}

	l.Close()

	after, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(after) != 0 {
		t.Errorf("the spill outlived the session: %v — one file per session, kept "+
			"forever, in a directory nobody looks at", names(after))
	}
}

// Closing twice is what a shutdown racing a delete does, and a second close
// must not turn into an error nobody can act on.
func TestClosingTwiceIsNotAnError(t *testing.T) {
	dir := t.TempDir()
	sp, err := NewSpill(dir, "s1")
	if err != nil {
		t.Fatal(err)
	}
	l := NewEventLog("s1", 2, func() time.Time { return time.Unix(0, 0) })
	l.SetSpill(sp)
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
func TestRemovingASessionThroughTheManagerLeavesNoSpill(t *testing.T) {
	dir := t.TempDir()
	m := NewManager(4)
	log := NewEventLog("s1", 2, func() time.Time { return time.Unix(0, 0) })
	sp, err := NewSpill(dir, "s1")
	if err != nil {
		t.Fatal(err)
	}
	log.SetSpill(sp)
	s := New("s1", "/w", "MiniMax-M3", "workspace-write", nil, log, func() time.Time { return time.Unix(0, 0) })
	if err := m.Add(s); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 6; i++ {
		s.Emit(protocol.EventMessageDelta, protocol.MessageDelta{Text: "x"})
	}
	if entries, _ := os.ReadDir(dir); len(entries) == 0 {
		t.Fatal("nothing spilled; the assertion below would prove nothing")
	}

	if err := m.Remove(s.ID); err != nil {
		t.Fatal(err)
	}
	if entries, _ := os.ReadDir(dir); len(entries) != 0 {
		t.Errorf("deleting the session left %v behind", names(entries))
	}
}
