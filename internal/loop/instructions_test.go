package loop

import (
	"testing"

	ce "github.com/aguinelo/dcode/internal/contextengine"
)

// A new prompt waits for the next turn.
//
// The prefix is byte-identical for the life of a session by design, and the
// assembly of every round reads it without the lock. SetMode arrives from the
// HTTP handler, so applying a prompt where it arrives would be a data race —
// and worse than the race, it would move the boundary under a conversation
// halfway through, so the model would read one thing in round two and another
// in round three of the same answer.
//
// The same rule the mode itself follows: a call already in flight finishes
// under whatever was in force when it started.
func TestANewPromptWaitsForTheNextTurn(t *testing.T) {
	e := New(Config{}, ce.Session{Instructions: "before"})

	e.SetInstructions("after")
	if got := e.Session().Instructions; got != "before" {
		t.Errorf("the prompt changed on arrival (%q); a live turn would have seen the boundary move under it", got)
	}

	e.takePendingPrompt()
	if got := e.Session().Instructions; got != "after" {
		t.Errorf("the prompt is %q after the turn began, want %q", got, "after")
	}

	// And it is taken once. A pending prompt left behind would reapply itself
	// at the top of every later turn, quietly undoing a switch made since.
	e.session.Instructions = "changed since"
	e.takePendingPrompt()
	if got := e.Session().Instructions; got != "changed since" {
		t.Errorf("the pending prompt was applied twice: %q", got)
	}
}

// An empty prompt is ignored.
//
// A rebuild that failed hands back "" and an error. The session reports the
// error and keeps going, so this must leave the prompt it had: a session
// running with NO doctrine is the one outcome worse than a session whose
// doctrine is out of date, and contextengine refuses to assemble one at all.
func TestAnEmptyPromptIsIgnored(t *testing.T) {
	e := New(Config{}, ce.Session{Instructions: "before"})
	e.SetInstructions("")
	e.takePendingPrompt()
	if got := e.Session().Instructions; got != "before" {
		t.Errorf("the prompt is %q; an empty rebuild must leave what was there", got)
	}
}
