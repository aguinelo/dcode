package session

import (
	"context"
	"testing"
	"time"

	"github.com/aguinelo/dcode/internal/protocol"
)

// take reads n events off a subscription, failing on a stall.
func take(t *testing.T, ch <-chan protocol.Event, n int) []protocol.Event {
	t.Helper()
	out := make([]protocol.Event, 0, n)
	for len(out) < n {
		select {
		case ev, open := <-ch:
			if !open {
				t.Fatalf("the stream closed after %d events, want %d", len(out), n)
			}
			out = append(out, ev)
		case <-time.After(2 * time.Second):
			t.Fatalf("got %d events, want %d", len(out), n)
		}
	}
	return out
}

func wantSeqs(t *testing.T, evs []protocol.Event, from uint64) {
	t.Helper()
	for i, ev := range evs {
		if ev.Seq != from+uint64(i) {
			t.Fatalf("event %d has seq %d, want %d: a gap or a duplicate", i, ev.Seq, from+uint64(i))
		}
	}
}

// A reopened session holds nothing in memory: its log starts after the
// record. The event stream is what every client follows, and it refused from=1
// with events_expired — the reopened orchestrator and its children each put
// "the events stopped arriving" on the desktop after a restart, while Replay,
// which no route uses for the stream, read the record.
func TestAReopenedSessionIsFollowedFromTheStart(t *testing.T) {
	dir := t.TempDir()
	recordSession(t, dir, false)
	s, _, err := reopenBare(NewManager(4), dir)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ch, stop, err := s.Log.Subscribe(ctx, 1)
	if err != nil {
		t.Fatalf("following a reopened session from 1 was refused: %v", err)
	}
	defer stop()

	if _, err := s.Log.Append(protocol.EventMessageDelta, protocol.MessageDelta{Text: "after"}); err != nil {
		t.Fatal(err)
	}
	evs := take(t, ch, 7)
	wantSeqs(t, evs, 1)
	if evs[0].Type != protocol.EventSessionCreated {
		t.Errorf("first event is %s, want the recorded session.created", evs[0].Type)
	}
}

// Retention trims memory, not the session: a recorded one is followed from
// any point, the trimmed part from disk and the rest from memory, once each.
func TestATrimmedRecordedSessionIsFollowedAcrossDiskAndMemory(t *testing.T) {
	rec, err := NewRecord(t.TempDir(), "s-sub")
	if err != nil {
		t.Fatal(err)
	}
	l := NewEventLog("s-sub", 10, nil)
	l.SetRecord(rec)
	defer l.Close()
	fill(t, l, 50)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ch, stop, err := l.Subscribe(ctx, 5)
	if err != nil {
		t.Fatalf("following from below the horizon was refused with a record behind it: %v", err)
	}
	defer stop()
	fill(t, l, 1)
	wantSeqs(t, take(t, ch, 47), 5)
}
