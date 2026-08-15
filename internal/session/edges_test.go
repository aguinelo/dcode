package session

import (
	"math"
	"strings"
	"testing"
	"time"

	"github.com/aguinelo/dcode/internal/protocol"
)

// A payload that cannot be encoded is an error to the caller, not an event with
// a hole in it. A session whose log holds an entry nobody can decode is worse
// than one missing the entry: the gap is invisible.
func TestAPayloadThatCannotBeEncodedIsRefused(t *testing.T) {
	l := NewEventLog("s1", 4, func() time.Time { return time.Unix(0, 0) })

	if _, err := l.Append(protocol.EventMessageDelta, math.Inf(1)); err == nil {
		t.Fatal("an unencodable payload was appended")
	}
	// And the sequence did not advance: a failed append must not leave a hole
	// a client would read as a lost event.
	ev, err := l.Append(protocol.EventMessageDelta, protocol.MessageDelta{Text: "x"})
	if err != nil {
		t.Fatal(err)
	}
	if ev.Seq != 1 {
		t.Errorf("seq = %d after a refused append, want 1", ev.Seq)
	}
}

// Asking for a session that is not there is a refusal naming what was asked
// for, so a client can tell "wrong id" from "empty answer".
func TestAskingForASessionThatIsNotThereNamesIt(t *testing.T) {
	m := NewManager(4)
	_, err := m.Get("never-existed")
	if err == nil {
		t.Fatal("an unknown session was returned")
	}
	if !strings.Contains(err.Error(), "never-existed") {
		t.Errorf("err = %v, want it to name the session asked for", err)
	}
}

// Answering an approval twice is a conflict, not a silent overwrite: two
// clients must not both believe they decided. And a decision that is not a
// decision is refused before it reaches anything.
func TestAnApprovalIsAnsweredOnceAndOnlyWithARealDecision(t *testing.T) {
	s := New("s1", "/w", "m", "read-only", nil,
		NewEventLog("s1", 10, nil), func() time.Time { return time.Unix(0, 0) })

	if err := s.Resolve("a1", protocol.ApprovalDecision("maybe")); err == nil {
		t.Error("an unknown decision was accepted")
	}

	// Nothing pending under that id, and nothing lapsed either: the answer says
	// it was already handled rather than inventing a state.
	err := s.Resolve("a1", protocol.ApprovalAllow)
	if err == nil {
		t.Fatal("answering an approval nobody is waiting on reported success")
	}
	if !strings.Contains(err.Error(), "a1") {
		t.Errorf("err = %v, want it to name the approval", err)
	}
}

// Retention is about memory, not about the record: Append writes every event as
// it arrives, so trimming must not write them again. The same sequence number
// twice in the file is a replay that hands a client the same event and no way
// to tell.
func TestTrimmingDoesNotWriteWhatTheRecordAlreadyHas(t *testing.T) {
	dir := t.TempDir()
	rec, err := NewRecord(dir, "s1")
	if err != nil {
		t.Fatal(err)
	}
	l := NewEventLog("s1", 3, func() time.Time { return time.Unix(0, 0) })
	l.SetRecord(rec)

	for i := 0; i < 12; i++ {
		if _, err := l.Append(protocol.EventMessageDelta, protocol.MessageDelta{Text: "x"}); err != nil {
			t.Fatal(err)
		}
	}
	l.Close()

	got, err := rec.Replay(1)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 12 {
		t.Fatalf("the record holds %d events, want 12", len(got))
	}
	seen := map[uint64]bool{}
	for _, ev := range got {
		if seen[ev.Seq] {
			t.Fatalf("sequence %d appears twice; trimming wrote what Append already had", ev.Seq)
		}
		seen[ev.Seq] = true
	}
}
