package session

import (
	"strings"

	"github.com/aguinelo/dcode/internal/protocol"
)

// Steer hands the running turn something the person said without ending it.
//
// The two things a person could do when a turn went wrong were let it finish
// wrong and kill it. Killing loses everything the turn learned, which is why
// people watch bad turns to the end.
//
// Refused when nothing is running, and deliberately: a correction to a turn
// that already finished is a new turn, and quietly turning it into one would
// mean a message the person believed was urgent waiting for them to notice it
// never took effect. The caller submits it instead, which is a different thing
// and should look like one.
func (s *Session) Steer(text string) error {
	if strings.TrimSpace(text) == "" {
		return protocol.Errorf(protocol.CodeInvalidInput, "nothing to say")
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.state == protocol.SessionStateClosed {
		return protocol.Errorf(protocol.CodeSessionNotFound, "session %s is closed", s.ID)
	}
	if s.state != protocol.SessionStateRunning {
		return protocol.Errorf(protocol.CodeNoActiveTurn,
			"no turn is running; send it as a message instead")
	}
	s.steering = append(s.steering, text)
	return nil
}

// TakeSteering hands the engine the oldest correction, or "".
//
// Exported because the engine is wired in package app, one layer above: the
// session is created after the engine it owns, so the queue is reached through
// a closure bound late rather than through a field set at construction.
//
// One at a time so the engine drains in order and so a correction added while
// the engine is draining is not lost: the loop calls until this is empty.
func (s *Session) TakeSteering() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.steering) == 0 {
		return ""
	}
	out := s.steering[0]
	s.steering = s.steering[1:]
	return out
}

// dropSteering forgets what nobody delivered.
//
// A turn ends, and anything still queued was meant for that turn. Carrying it
// into the next one would deliver a correction against work that has already
// finished — the model would read "no, do it the other way" about something it
// is no longer doing.
func (s *Session) dropSteering() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	left := s.steering
	s.steering = nil
	return left
}
