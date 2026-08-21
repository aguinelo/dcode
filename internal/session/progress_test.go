package session

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/aguinelo/dcode/internal/protocol"
)

// Progress is the one event that is not a fact, and it still carries a Seq and
// still lands in the record.
//
// The alternative was to leave it out of the sequence, on the grounds that
// nobody replays a count. That would have put a GAP in the one property the
// record is built on — monotonic from 1, never reused and never gapped — and a
// record with a hole in it is a record whose replay cannot be trusted about
// anything else either. message.delta is already chatty and already in there;
// this follows it rather than inventing an exception for a second kind.
func TestProgressJoinsTheSequenceLikeAnyOtherEvent(t *testing.T) {
	dir := t.TempDir()
	rec, err := NewRecord(dir, "s1")
	if err != nil {
		t.Fatal(err)
	}

	l := NewEventLog("s1", 10, func() time.Time { return time.Unix(0, 0) })
	l.SetRecord(rec)

	first, err := l.Append(protocol.EventTurnStarted, protocol.TurnStarted{TurnID: "t1"})
	if err != nil {
		t.Fatal(err)
	}
	prog, err := l.Append(protocol.EventProgress, protocol.Progress{
		TurnID: "t1", Kind: protocol.ProgressRounds, Done: 1, Total: 100})
	if err != nil {
		t.Fatal(err)
	}
	after, err := l.Append(protocol.EventTurnCompleted, protocol.TurnCompleted{TurnID: "t1"})
	if err != nil {
		t.Fatal(err)
	}

	if prog.Seq != first.Seq+1 || after.Seq != prog.Seq+1 {
		t.Errorf("the sequence has a gap around progress: %d, %d, %d",
			first.Seq, prog.Seq, after.Seq)
	}
	if err := rec.Close(); err != nil {
		t.Fatal(err)
	}

	var body []byte
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		b, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			t.Fatal(err)
		}
		body = append(body, b...)
	}
	if !strings.Contains(string(body), string(protocol.EventProgress)) {
		t.Errorf("progress never reached the record:\n%s", body)
	}
}
