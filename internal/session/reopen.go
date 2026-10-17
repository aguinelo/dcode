package session

import (
	ce "github.com/aguinelo/dcode/internal/contextengine"
	"github.com/aguinelo/dcode/internal/loop"
	"github.com/aguinelo/dcode/internal/protocol"
)

// Red: the shape of reopening, with today's answer — a recorded session
// cannot come back under its own id.

type Reopening struct {
	ID       string
	Path     string
	Origin   protocol.Session
	Title    string
	History  []ce.Message
	LastSeq  uint64
	Turns    int
	OpenTurn string
}

func NewReopened(r Reopening, engine *loop.Engine, log *EventLog, clock Clock) *Session {
	return New(r.ID, r.Origin.Workspace, r.Origin.Model, r.Origin.SandboxMode, engine, log, clock)
}

type ReopenBuilder func(r Reopening, log *EventLog) (*Session, error)

func (m *Manager) Reopen(dir, id string, retention int, clock Clock, build ReopenBuilder) (*Session, error) {
	return nil, protocol.Errorf(protocol.CodeInternal, "a recorded session cannot be reopened under its own id")
}

func (s *Session) SubmitOrigin(text, origin string) error {
	return s.SubmitAttached(text, loop.Attachments{})
}
