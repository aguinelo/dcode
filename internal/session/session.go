package session

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/aguinelo/dcode/internal/loop"
	"github.com/aguinelo/dcode/internal/protocol"
)

// Session is the server-owned unit of work. The client holds none of this:
// killing, restarting or swapping a client does not touch a session, and a turn
// in flight continues with zero clients attached.
type Session struct {
	ID        string
	Workspace string
	Model     string
	Mode      string
	CreatedAt time.Time

	Log    *EventLog
	engine *loop.Engine

	mu      sync.Mutex
	state   protocol.SessionState
	cancel  context.CancelFunc
	pending map[string]*approval
	// allowAll records an "allow for the session" decision, which is the one
	// place a user answer outlives a single crossing.
	allowAll map[string]bool
}

type approval struct {
	req      protocol.ApprovalRequest
	answer   chan protocol.ApprovalDecision
	once     sync.Once
	resolved bool
}

// New builds a session around an engine.
func New(id, workspace, model, mode string, engine *loop.Engine, log *EventLog, clock Clock) *Session {
	if clock == nil {
		clock = time.Now
	}
	return &Session{
		ID: id, Workspace: workspace, Model: model, Mode: mode,
		CreatedAt: clock(), Log: log, engine: engine,
		state: protocol.SessionStateIdle, pending: map[string]*approval{},
		allowAll: map[string]bool{},
	}
}

// Describe returns the wire view.
func (s *Session) Describe() protocol.Session {
	s.mu.Lock()
	state := s.state
	s.mu.Unlock()
	return protocol.Session{
		ID: s.ID, State: state, Workspace: s.Workspace, Model: s.Model,
		SandboxMode: s.Mode, CreatedAt: s.CreatedAt, LastSeq: s.Log.LastSeq(),
	}
}

// State returns the current state.
func (s *Session) State() protocol.SessionState {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.state
}

// Submit starts a turn.
//
// One turn per session: concurrent input is refused rather than queued here.
// The queue belongs to the client, which is what keeps the event log linearly
// ordered and the append-only context coherent.
func (s *Session) Submit(text string) error {
	s.mu.Lock()
	if s.state == protocol.SessionStateClosed {
		s.mu.Unlock()
		return protocol.Errorf(protocol.CodeSessionNotFound, "session %s is closed", s.ID)
	}
	if s.state != protocol.SessionStateIdle {
		s.mu.Unlock()
		return protocol.Errorf(protocol.CodeTurnAlreadyActive,
			"a turn is already running; wait for it to finish or interrupt it")
	}

	// The turn's context is detached from any request: a client disconnecting
	// must not cancel work the user asked for.
	ctx, cancel := context.WithCancel(context.Background())
	s.state = protocol.SessionStateRunning
	s.cancel = cancel
	s.mu.Unlock()

	go func() {
		defer func() {
			s.mu.Lock()
			if s.state != protocol.SessionStateClosed {
				s.state = protocol.SessionStateIdle
			}
			s.cancel = nil
			s.failPending()
			s.mu.Unlock()
			cancel()
		}()
		_, _ = s.engine.Run(ctx, text)
	}()
	return nil
}

// Interrupt cancels the running turn. Idempotent: interrupting an idle session
// is not an error, because the user cannot know the turn just finished.
func (s *Session) Interrupt() {
	s.mu.Lock()
	cancel := s.cancel
	s.failPending()
	s.mu.Unlock()
	if cancel != nil {
		cancel()
	}
}

// Close ends the session.
func (s *Session) Close() {
	s.mu.Lock()
	s.state = protocol.SessionStateClosed
	cancel := s.cancel
	s.failPending()
	s.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	s.Log.Close()
}

// failPending denies every waiting approval. Caller holds the lock.
//
// Failing closed: a turn that ends while someone is deciding must not leave the
// decision open, and the safe reading of an abandoned question is no.
func (s *Session) failPending() {
	for id, a := range s.pending {
		a.resolve(protocol.ApprovalDeny)
		delete(s.pending, id)
	}
}

func (a *approval) resolve(d protocol.ApprovalDecision) {
	a.once.Do(func() {
		a.resolved = true
		a.answer <- d
		close(a.answer)
	})
}

// Approve is the loop's side of a boundary crossing: it registers the question
// and blocks until a client answers or the deadline passes.
func (s *Session) Approve(ctx context.Context, req protocol.ApprovalRequest, timeout time.Duration) (
	protocol.ApprovalDecision, error,
) {
	key := req.Tool + "\x00" + req.Command
	s.mu.Lock()
	if s.allowAll[key] {
		s.mu.Unlock()
		return protocol.ApprovalAllowSession, nil
	}
	if timeout > 0 {
		req.ExpiresAt = time.Now().Add(timeout)
	}
	a := &approval{req: req, answer: make(chan protocol.ApprovalDecision, 1)}
	s.pending[req.ApprovalID] = a
	s.state = protocol.SessionStateBlocked
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		delete(s.pending, req.ApprovalID)
		if s.state == protocol.SessionStateBlocked {
			s.state = protocol.SessionStateRunning
		}
		s.mu.Unlock()
	}()

	var timer <-chan time.Time
	if timeout > 0 {
		t := time.NewTimer(timeout)
		defer t.Stop()
		timer = t.C
	}

	select {
	case d := <-a.answer:
		if d == protocol.ApprovalAllowSession {
			s.mu.Lock()
			s.allowAll[key] = true
			s.mu.Unlock()
		}
		return d, nil
	case <-timer:
		// Nobody answered in time. Denying is the only safe reading: the
		// alternative would be granting because everyone looked away.
		return protocol.ApprovalDeny, nil
	case <-ctx.Done():
		return protocol.ApprovalDeny, nil
	}
}

// Resolve answers a pending approval. First writer wins; the rest get a
// conflict, so two clients cannot both believe they decided.
func (s *Session) Resolve(approvalID string, d protocol.ApprovalDecision) error {
	if !d.Valid() {
		return protocol.Errorf(protocol.CodeInternal, "unknown decision %q", d)
	}
	s.mu.Lock()
	a, ok := s.pending[approvalID]
	if !ok {
		s.mu.Unlock()
		return protocol.Errorf(protocol.CodeApprovalResolved,
			"approval %s is not pending; it was already answered or it expired", approvalID)
	}
	if a.resolved {
		s.mu.Unlock()
		return protocol.Errorf(protocol.CodeApprovalResolved,
			"approval %s was already answered", approvalID)
	}
	delete(s.pending, approvalID)
	s.mu.Unlock()

	a.resolve(d)
	return nil
}

// Pending lists unanswered approvals, so a client attaching mid-turn can render
// the question it missed.
func (s *Session) Pending() []protocol.ApprovalRequest {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]protocol.ApprovalRequest, 0, len(s.pending))
	for _, a := range s.pending {
		out = append(out, a.req)
	}
	return out
}

// Emit records an event. Implements loop.Emitter.
func (s *Session) Emit(t protocol.EventType, payload any) {
	_, _ = s.Log.Append(t, payload)
}

// Manager owns the live sessions.
type Manager struct {
	mu       sync.RWMutex
	sessions map[string]*Session
	max      int
	order    []string
}

// NewManager builds a manager with a ceiling on live sessions.
func NewManager(max int) *Manager {
	if max <= 0 {
		max = 64
	}
	return &Manager{sessions: map[string]*Session{}, max: max}
}

// Add registers a session, refusing past the ceiling.
func (m *Manager) Add(s *Session) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.sessions) >= m.max {
		return protocol.Errorf(protocol.CodeMaxSessionsReached,
			"%d sessions are already open, which is the configured maximum", m.max)
	}
	m.sessions[s.ID] = s
	m.order = append(m.order, s.ID)
	return nil
}

// Get returns a session.
func (m *Manager) Get(id string) (*Session, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	s, ok := m.sessions[id]
	if !ok {
		return nil, protocol.Errorf(protocol.CodeSessionNotFound, "no session %s", id)
	}
	return s, nil
}

// List returns live sessions in creation order, so a client's list does not
// reshuffle between calls.
func (m *Manager) List() []protocol.Session {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]protocol.Session, 0, len(m.sessions))
	for _, id := range m.order {
		if s, ok := m.sessions[id]; ok {
			out = append(out, s.Describe())
		}
	}
	return out
}

// Remove closes and forgets a session.
func (m *Manager) Remove(id string) error {
	m.mu.Lock()
	s, ok := m.sessions[id]
	if ok {
		delete(m.sessions, id)
		for i, o := range m.order {
			if o == id {
				m.order = append(m.order[:i], m.order[i+1:]...)
				break
			}
		}
	}
	m.mu.Unlock()

	if !ok {
		return protocol.Errorf(protocol.CodeSessionNotFound, "no session %s", id)
	}
	s.Close()
	return nil
}

// CloseAll shuts every session down.
func (m *Manager) CloseAll() {
	m.mu.Lock()
	all := make([]*Session, 0, len(m.sessions))
	for _, s := range m.sessions {
		all = append(all, s)
	}
	m.sessions = map[string]*Session{}
	m.order = nil
	m.mu.Unlock()

	for _, s := range all {
		s.Close()
	}
}

// Count returns the number of live sessions.
func (m *Manager) Count() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.sessions)
}

// NewID returns a time-ordered, filesystem-safe identifier.
//
// Not a ULID library: the requirement is "sortable by time and safe in a
// filename", which twenty lines cover without another dependency in a binary
// that promises to stay small.
func NewID(clock Clock, entropy func() uint32) string {
	if clock == nil {
		clock = time.Now
	}
	ms := clock().UTC().UnixMilli()
	var suffix uint32
	if entropy != nil {
		suffix = entropy()
	}
	return fmt.Sprintf("%011x%08x", ms, suffix)
}
