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

// A record holds its own events and a marker, never a copy of what it
// continues — and reading one still yields the whole conversation.
//
// Continuing used to copy the previous record into the new one, so a session
// that continued a session that continued a session held three copies of the
// first. On this machine that reached 3.6 MB, and it grows with every `-c`:
// each continuation copies every copy before it.
func TestARecordDoesNotCopyWhatItContinues(t *testing.T) {
	dir := t.TempDir()

	// A first conversation with a real turn in it.
	write := func(id string, evs ...protocol.Event) {
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
	turn := func(seq uint64, id, text string) []protocol.Event {
		started, _ := json.Marshal(protocol.TurnStarted{TurnID: id, Text: text})
		done, _ := json.Marshal(protocol.TurnCompleted{TurnID: id, Reason: protocol.StopDone})
		return []protocol.Event{
			{Seq: seq, Type: protocol.EventTurnStarted, Payload: started},
			{Seq: seq + 1, Type: protocol.EventTurnCompleted, Payload: done},
		}
	}
	marker := func(seq uint64, source string) protocol.Event {
		raw, _ := json.Marshal(protocol.SessionResumed{SourceID: source, Turns: 1})
		return protocol.Event{Seq: seq, Type: protocol.EventSessionResumed, Payload: raw}
	}

	write("aaa", turn(1, "t1", "primeira")...)
	write("bbb", append([]protocol.Event{marker(1, "aaa")}, turn(2, "t2", "segunda")...)...)
	write("ccc", append([]protocol.Event{marker(1, "bbb")}, turn(2, "t3", "terceira")...)...)

	evs, turns, err := Carry(filepath.Join(dir, "ccc.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	if turns != 3 {
		t.Errorf("the chain counted %d turns, want 3", turns)
	}

	// Oldest first, and every question present exactly once. A copy would say
	// "primeira" twice from the third record.
	var asked []string
	for _, e := range evs {
		if e.Type != protocol.EventTurnStarted {
			continue
		}
		var d protocol.TurnStarted
		_ = json.Unmarshal(e.Payload, &d)
		asked = append(asked, d.Text)
	}
	if got, want := strings.Join(asked, ","), "primeira,segunda,terceira"; got != want {
		t.Errorf("the chain reads %q, want %q", got, want)
	}

	// And the marker itself does not survive: the session being built emits its
	// own, naming the record it was asked to continue.
	for _, e := range evs {
		if e.Type == protocol.EventSessionResumed {
			t.Error("a marker from further up the chain reached the replay")
		}
	}
}

// A record naming itself, or two naming each other, is a corrupt pair rather
// than an impossible one: the id is a timestamp and a random suffix, and
// nothing enforces the arrow points backwards.
func TestAChainThatLoopsIsReadOnce(t *testing.T) {
	dir := t.TempDir()
	self, _ := json.Marshal(protocol.SessionResumed{SourceID: "aaa"})
	started, _ := json.Marshal(protocol.TurnStarted{TurnID: "t", Text: "só uma"})
	var b strings.Builder
	for _, e := range []protocol.Event{
		{Seq: 1, Type: protocol.EventSessionResumed, Payload: self},
		{Seq: 2, Type: protocol.EventTurnStarted, Payload: started},
	} {
		raw, _ := json.Marshal(e)
		b.Write(raw)
		b.WriteByte('\n')
	}
	if err := os.WriteFile(filepath.Join(dir, "aaa.jsonl"), []byte(b.String()), 0o600); err != nil {
		t.Fatal(err)
	}

	done := make(chan struct{})
	go func() {
		defer close(done)
		evs, _, err := Carry(filepath.Join(dir, "aaa.jsonl"))
		if err != nil {
			t.Error(err)
		}
		if len(evs) != 1 {
			t.Errorf("a self-continuing record read %d events, want 1", len(evs))
		}
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("reading a looping chain did not terminate")
	}
}
