package session

import (
	"strings"
	"testing"
	"time"

	"github.com/aguinelo/dcode/internal/protocol"
)

func running(t *testing.T) *Session {
	t.Helper()
	s := New("s1", "/w", "m", "read-only", nil,
		NewEventLog("s1", 10, nil), func() time.Time { return time.Unix(0, 0) })
	s.mu.Lock()
	s.state = protocol.SessionStateRunning
	s.mu.Unlock()
	return s
}

// A correction reaches the running turn and comes back out in the order it was
// said. Two reordered is one instruction answering the wrong question.
func TestCorrectionsQueueAndComeBackInOrder(t *testing.T) {
	s := running(t)

	for _, text := range []string{"first", "second"} {
		if err := s.Steer(text); err != nil {
			t.Fatalf("steering %q: %v", text, err)
		}
	}
	if got := s.TakeSteering(); got != "first" {
		t.Errorf("got %q, want first", got)
	}
	if got := s.TakeSteering(); got != "second" {
		t.Errorf("got %q, want second", got)
	}
	if got := s.TakeSteering(); got != "" {
		t.Errorf("got %q from an empty queue", got)
	}
}

// Correcting a turn that is not running is refused, not quietly turned into a
// message. The person believed it was urgent; letting them think it landed when
// it did not is worse than saying so.
func TestCorrectingWhenNothingRunsIsRefused(t *testing.T) {
	s := New("s1", "/w", "m", "read-only", nil,
		NewEventLog("s1", 10, nil), func() time.Time { return time.Unix(0, 0) })

	err := s.Steer("too late")
	if err == nil {
		t.Fatal("steering an idle session reported success")
	}
	var perr *protocol.Error
	if !asProtocol(err, &perr) || perr.Code != protocol.CodeNoActiveTurn {
		t.Fatalf("err = %v, want no_active_turn", err)
	}
	// And it is the mirror of the other refusal, not the same one: one says
	// wait, this one says there is nothing to correct.
	if perr.Code == protocol.CodeTurnAlreadyActive {
		t.Error("the two refusals collapsed into one the caller cannot tell apart")
	}
}

// A closed session refuses too, and says which of the two it is.
func TestCorrectingAClosedSessionIsRefused(t *testing.T) {
	s := running(t)
	s.Close()
	if err := s.Steer("anything"); err == nil {
		t.Fatal("steering a closed session reported success")
	} else if !strings.Contains(err.Error(), "closed") {
		t.Errorf("err = %v, want it to say the session is closed", err)
	}
}

// Nothing to say is not a correction.
func TestAnEmptyCorrectionIsRefused(t *testing.T) {
	s := running(t)
	for _, text := range []string{"", "   ", "\n\t"} {
		if err := s.Steer(text); err == nil {
			t.Errorf("steering %q reported success", text)
		}
	}
}

// What nobody delivered is forgotten with the turn it was meant for. Carried
// forward it would deliver "no, do it the other way" about work that already
// finished.
func TestUndeliveredCorrectionsDoNotOutliveTheirTurn(t *testing.T) {
	s := running(t)
	if err := s.Steer("never reached the loop"); err != nil {
		t.Fatal(err)
	}

	left := s.dropSteering()
	if len(left) != 1 {
		t.Fatalf("dropped %v, want the one nobody delivered", left)
	}
	if got := s.TakeSteering(); got != "" {
		t.Errorf("got %q after dropping", got)
	}
	// And dropping again finds nothing rather than repeating the report.
	if again := s.dropSteering(); len(again) != 0 {
		t.Errorf("dropped %v twice", again)
	}
}

func asProtocol(err error, out **protocol.Error) bool {
	p, ok := err.(*protocol.Error)
	if ok {
		*out = p
	}
	return ok
}
