package session

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	ce "github.com/aguinelo/dcode/internal/contextengine"
	"github.com/aguinelo/dcode/internal/loop"
	"github.com/aguinelo/dcode/internal/protocol"
)

// Reopening a recorded session under its own id.
//
// Continuing (CreateSessionRequest.Resume) opens a NEW session that replays an
// old one, which is right for a person typing `dcode -c` and wrong for anything
// other sessions point at by id. An orchestrator is named in its project's
// state and in every child's parent_id; a daemon restart that brought it back
// under a fresh id would orphan all of them. So a reopen keeps the id, the
// history, the sequence and the file, and is refused rather than improvised
// when any of the four cannot be had.

// Reopening is what a record says about the session it holds, read so that
// session can live again.
type Reopening struct {
	ID   string
	Path string
	// Origin is the session as it announced itself (session.created), with
	// the mode as the record last left it. It is the wire type on purpose: a
	// field the session announces — the model bundle, the loop, and the
	// parent, project, role and owns that place it under an orchestrator —
	// comes back with it, with no second list here to keep in step.
	Origin protocol.Session
	// Title is the last name the record gave the conversation; empty is none.
	Title string
	// History is the conversation the model was sent, by Rebuild.
	History []ce.Message
	// LastSeq is the highest sequence in this session's own record. The next
	// event is LastSeq+1.
	LastSeq uint64
	// Turns is the highest turn number the record used, so the engine numbers
	// the next turn after it (loop.Engine.ContinueTurns).
	Turns int
	// OpenTurn is a turn the record shows started and never completed: the
	// daemon stopped under it. Empty when every turn ended.
	OpenTurn string
	// torn is a last line cut short by a crash. It is terminated before the
	// first new event, so the next line is a line of its own.
	torn bool
}

// ReadReopening reads the record of id in dir.
//
// A record that is not there — never written, or pruned — is
// session_not_found, and a record that cannot be read as one is internal:
// both are said, and neither is answered with a fresh session under the old
// id. The one damage tolerated is the last line, which is what a crash in the
// middle of a write leaves, and which the reopen terminates rather than
// builds on.
func ReadReopening(dir, id string) (Reopening, error) {
	if err := ValidID(id); err != nil {
		return Reopening{}, err
	}
	if dir == "" {
		return Reopening{}, protocol.Errorf(protocol.CodeSessionNotFound,
			"session %s cannot be reopened: sessions are not being recorded", id)
	}
	path := filepath.Join(dir, id+".jsonl")
	raw, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return Reopening{}, protocol.Errorf(protocol.CodeSessionNotFound,
			"session %s cannot be reopened: its record is gone (%s)", id, path)
	}
	if err != nil {
		return Reopening{}, protocol.Errorf(protocol.CodeInternal,
			"session %s cannot be reopened: %v", id, err)
	}
	r, err := readRecord(id, raw)
	if err != nil {
		return Reopening{}, protocol.Errorf(protocol.CodeInternal,
			"session %s cannot be reopened: %v", id, err)
	}
	r.Path = path
	if r.History, err = Rebuild(path); err != nil {
		return Reopening{}, protocol.Errorf(protocol.CodeInternal,
			"session %s cannot be reopened: its history could not be rebuilt: %v", id, err)
	}
	return r, nil
}

// readRecord folds one record's lines into what a reopen needs.
func readRecord(id string, raw []byte) (Reopening, error) {
	r := Reopening{ID: id}
	r.torn = len(raw) > 0 && raw[len(raw)-1] != '\n'
	lines := bytes.Split(bytes.TrimSuffix(raw, []byte("\n")), []byte("\n"))
	created := false
	for i, line := range lines {
		if len(line) == 0 {
			continue
		}
		var ev protocol.Event
		if err := json.Unmarshal(line, &ev); err != nil {
			if i == len(lines)-1 && r.torn {
				continue
			}
			return Reopening{}, fmt.Errorf("line %d of its record is not an event: %v", i+1, err)
		}
		if ev.SessionID != "" && ev.SessionID != id {
			return Reopening{}, fmt.Errorf("line %d of its record belongs to session %s", i+1, ev.SessionID)
		}
		if ev.Seq > r.LastSeq {
			r.LastSeq = ev.Seq
		}
		switch ev.Type {
		case protocol.EventSessionCreated:
			if created {
				continue
			}
			if err := json.Unmarshal(ev.Payload, &r.Origin); err != nil {
				return Reopening{}, fmt.Errorf("its session.created cannot be read: %v", err)
			}
			created = true
		case protocol.EventSessionModeChanged:
			var d protocol.SessionModeChanged
			if json.Unmarshal(ev.Payload, &d) == nil && d.Mode != "" {
				r.Origin.Mode, r.Origin.SandboxMode = d.Mode, d.SandboxMode
			}
		case protocol.EventSessionRenamed:
			var d protocol.SessionRenamed
			if json.Unmarshal(ev.Payload, &d) == nil {
				r.Title = d.Name
			}
		case protocol.EventTurnStarted:
			var d protocol.TurnStarted
			if json.Unmarshal(ev.Payload, &d) == nil {
				r.OpenTurn = d.TurnID
				if n := turnNumber(d.TurnID); n > r.Turns {
					r.Turns = n
				}
			}
		case protocol.EventTurnCompleted:
			r.OpenTurn = ""
		}
	}
	if !created {
		return Reopening{}, errNotARecord
	}
	r.Origin.ID = id
	return r, nil
}

// turnNumber reads the n of an engine's turn id, "t<n>" or "x<n>".
func turnNumber(id string) int {
	if len(id) < 2 {
		return 0
	}
	n, err := strconv.Atoi(strings.TrimLeft(id[1:], "0"))
	if err != nil {
		return 0
	}
	return n
}

// Log opens the event log the reopened session writes to: its own record,
// opened for appending, with the sequence continuing from the last event in
// it. What came before is read from the record (EventLog.Replay below the
// horizon), so a client attaching from 1 gets the whole session.
func (r Reopening) Log(dir string, retention int, clock Clock) (*EventLog, error) {
	if r.torn {
		// The crash's half line ends here, so the first new event is a line
		// of its own rather than glued to garbage nobody can read.
		f, err := os.OpenFile(r.Path, os.O_APPEND|os.O_WRONLY, 0o600)
		if err != nil {
			return nil, err
		}
		_, err = io.WriteString(f, "\n")
		if cerr := f.Close(); err == nil {
			err = cerr
		}
		if err != nil {
			return nil, err
		}
	}
	rec, err := NewRecord(dir, r.ID)
	if err != nil {
		return nil, err
	}
	l := NewEventLog(r.ID, retention, clock)
	l.seq, l.firstSeq = r.LastSeq, r.LastSeq+1
	l.SetRecord(rec)
	return l, nil
}

// NewReopened builds the session a record held, under its own id, around an
// engine built from r — History seeded, the model bundle and the mode of
// r.Origin. What the record says the session was is what it is again: the
// workspace, the bundle, the branch, the loop and when it was created.
func NewReopened(r Reopening, engine *loop.Engine, log *EventLog, clock Clock) *Session {
	o := r.Origin
	s := New(r.ID, o.Workspace, o.Model, o.SandboxMode, engine, log, clock)
	s.Family, s.Transport, s.BaseURL, s.Branch = o.Family, o.Transport, o.BaseURL, o.Branch
	s.LoopID, s.ContextWindow, s.CreatedAt = o.LoopID, o.ContextWindow, o.CreatedAt
	s.ParentID, s.ProjectID, s.Role, s.Owns = o.ParentID, o.ProjectID, o.Role, o.Owns
	if engine != nil {
		engine.ContinueTurns(r.Turns)
	} else if o.Mode != "" {
		s.behaviourMode = o.Mode
	}
	return s
}

// ReopenBuilder builds the session for r on log. The daemon's: it alone knows
// how an engine is made. It must use NewReopened, or give the session r.ID and
// log by other means.
type ReopenBuilder func(r Reopening, log *EventLog) (*Session, error)

// Reopen brings the recorded session id back to life under the same id.
//
// Refused while id is live or already being reopened: two sessions writing one
// record is a record with two sequences in it. A turn the record shows open is
// closed with turn.completed{reason: daemon_restart} before the session is
// held, so it is the first new event and nobody can start a turn ahead of it.
func (m *Manager) Reopen(dir, id string, retention int, clock Clock, build ReopenBuilder) (*Session, error) {
	if err := ValidID(id); err != nil {
		return nil, err
	}
	if err := m.claim(id); err != nil {
		return nil, err
	}
	defer m.release(id)

	r, err := ReadReopening(dir, id)
	if err != nil {
		return nil, err
	}
	log, err := r.Log(dir, retention, clock)
	if err != nil {
		return nil, protocol.Errorf(protocol.CodeInternal,
			"session %s cannot be reopened: its record cannot be opened: %v", id, err)
	}
	s, err := build(r, log)
	if err != nil {
		log.Close()
		return nil, err
	}
	if s == nil || s.ID != id || s.Log != log {
		log.Close()
		return nil, protocol.Errorf(protocol.CodeInternal,
			"session %s cannot be reopened: the builder did not build it on its own record", id)
	}
	if r.OpenTurn != "" {
		if _, err := log.Append(protocol.EventTurnCompleted, protocol.TurnCompleted{
			TurnID: r.OpenTurn, Reason: protocol.StopDaemonRestart,
		}); err != nil {
			s.Close()
			return nil, protocol.Errorf(protocol.CodeInternal,
				"session %s cannot be reopened: its open turn could not be closed: %v", id, err)
		}
	}
	if err := m.Add(s); err != nil {
		s.Close()
		return nil, err
	}
	return s, nil
}

// claim marks id as being reopened, refusing one that is live or already
// being reopened.
func (m *Manager) claim(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, live := m.sessions[id]; live || m.reopening[id] {
		return protocol.Errorf(protocol.CodeInvalidInput,
			"session %s is already open; a live session is not reopened", id)
	}
	if m.reopening == nil {
		m.reopening = map[string]bool{}
	}
	m.reopening[id] = true
	return nil
}

func (m *Manager) release(id string) {
	m.mu.Lock()
	delete(m.reopening, id)
	m.mu.Unlock()
}

// SubmitOrigin starts a turn that says who started it: empty is the person,
// protocol.OriginChildren is the harness bringing children's reports.
func (s *Session) SubmitOrigin(text, origin string) error {
	if origin != "" && origin != protocol.OriginChildren {
		return protocol.Errorf(protocol.CodeInvalidInput,
			"unknown turn origin %q (want empty or %q)", origin, protocol.OriginChildren)
	}
	return s.SubmitAttached(text, loop.Attachments{Origin: origin})
}
