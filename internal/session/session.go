package session

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	ce "github.com/aguinelo/dcode/internal/contextengine"
	"github.com/aguinelo/dcode/internal/loop"
	"github.com/aguinelo/dcode/internal/policy"
	"github.com/aguinelo/dcode/internal/protocol"
)

// Session is the server-owned unit of work. The client holds none of this:
// killing, restarting or swapping a client does not touch a session, and a turn
// in flight continues with zero clients attached.
type Session struct {
	ID            string
	Workspace     string
	Model         string
	Mode          string
	behaviourMode string
	CreatedAt     time.Time
	// ContextWindow is the model's window, so a client can turn a token count
	// into the percentage a person can act on.
	ContextWindow int

	Log    *EventLog
	engine *loop.Engine
	// steering is what the person said while a turn was running, waiting for
	// the next round to deliver it.
	steering []string

	// Standing is the record of decisions that outlive the session. Optional:
	// without one the session asks every time, which is what it did before.
	//
	// A port rather than the record itself, because this package must not learn
	// what the boundaries mean. It asks "was this already answered" and says
	// "remember this"; which crossings are worth remembering, and where the
	// answer is kept, belong to the layer that knows both.
	Standing Standing

	// Carried is the conversation this session continues, waiting to be put in
	// the log. It is held rather than emitted at construction because the log
	// has to open with this session's own creation: a record whose first line
	// is somebody else's turn is not a record anything can describe.
	Carried []protocol.Event
	// CarriedFrom and CarriedTurns describe where it came from, for the marker
	// that opens the replay.
	CarriedFrom  string
	CarriedTurns int

	mu      sync.Mutex
	state   protocol.SessionState
	cancel  context.CancelFunc
	pending map[string]*approval
	// lapsed remembers approvals the deadline denied, so a client that comes
	// back to answer one is told that rather than that somebody else did.
	lapsed      map[string]struct{}
	lapsedOrder []string
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
		behaviourMode: bornIn(engine),
		CreatedAt:     clock(), Log: log, engine: engine,
		state: protocol.SessionStateIdle, pending: map[string]*approval{},
		allowAll: map[string]bool{},
	}
}

// EmitCarried puts the continued conversation in the log, behind a marker.
//
// Called once, right after the session announces itself. The client draws the
// screen from events, so the events go in the log — but only the MARKER goes in
// the record, and the next session to continue this one follows the marker back
// rather than reading a copy.
//
// The copy was the original answer to all three, and it was the right shape
// with the wrong cost: each continuation copied every copy before it, so the
// record grew quadratically in the number of times somebody typed `-c`.
func (s *Session) EmitCarried() {
	if len(s.Carried) == 0 && s.CarriedFrom == "" {
		return
	}
	s.Emit(protocol.EventSessionResumed, protocol.SessionResumed{
		SourceID: s.CarriedFrom,
		Turns:    s.CarriedTurns,
	})
	for _, ev := range s.Carried {
		// Re-appended rather than copied: the sequence and the session id must
		// be this session's, or a client cannot tell a replayed event from one
		// it already has.
		//
		// UNRECORDED. The screen is served from the log in memory, and the
		// record does not need these: it holds the marker above, and Rebuild
		// and Carry follow it. Writing them was what made a session that
		// continued a session that continued a session hold three copies of the
		// first — 3.6 MB on this machine, growing with every `-c`.
		_, _ = s.Log.AppendUnrecorded(ev.Type, json.RawMessage(ev.Payload))
	}
	s.Carried = nil
}

// Describe returns the wire view.
func (s *Session) Describe() protocol.Session {
	s.mu.Lock()
	state := s.state
	sandbox, behaviour := s.Mode, s.behaviourMode
	s.mu.Unlock()
	return protocol.Session{
		ID: s.ID, State: state, Workspace: s.Workspace, Model: s.Model,
		SandboxMode: sandbox, Mode: behaviour,
		CreatedAt: s.CreatedAt, LastSeq: s.Log.LastSeq(),
		FirstSeq:      s.Log.FirstSeq(),
		ContextWindow: s.ContextWindow,
	}
}

// SetMode switches the session behavioural mode (plan, assist, auto).
//
// The mapping is: plan -> read-only + never, assist -> workspace-write +
// on-request, auto -> full-access + on-request. The engine takes the new
// SandboxMode and ApprovalPolicy atomically (loop.Engine.SetMode), and the
// session announces the change over the event log so a client attaching after
// reads the current mode.
//
// No-op (no event, no error) when name is already the current mode. Live turns
// are NOT interrupted.
func (s *Session) SetMode(name string) error {
	if !protocol.ValidMode(name) {
		return protocol.Errorf(protocol.CodeInvalidInput,
			"unknown mode %q (want plan, assist or auto)", name)
	}
	if s.engine == nil {
		return protocol.Errorf(protocol.CodeInternal, "session has no engine")
	}
	sandbox, pol := modeToSandbox(name)

	// Check and set under one hold. Read-then-write with the lock dropped in
	// between lets two concurrent switches both pass the no-op test and land in
	// either order, leaving the announced mode and the engine disagreeing.
	s.mu.Lock()
	previous := s.behaviourMode
	if previous == name {
		s.mu.Unlock()
		return nil
	}
	s.behaviourMode = name
	// Mode is the sandbox this session advertises in Describe. Leaving it
	// behind would answer the old boundary to every client that asks.
	s.Mode = string(sandbox)
	s.mu.Unlock()

	s.engine.SetMode(sandbox, pol)
	s.Emit(protocol.EventSessionModeChanged, protocol.SessionModeChanged{
		Previous: previous, Mode: name, SandboxMode: string(sandbox),
	})
	return nil
}

// bornIn names the mode the engine is already running, so a session opens
// under the badge that matches its boundary.
//
// It used to answer assist unconditionally, which made a full-access session
// show the bounded badge — and made the switch BACK to assist a silent no-op,
// because the session believed it was already there. An engine-less session
// (tests, and only tests) has no boundary to name.
func bornIn(engine *loop.Engine) string {
	if engine == nil {
		return ""
	}
	return modeFrom(engine.Mode())
}

// modeFrom names the pair, and is the exact inverse of modeToSandbox.
//
// A pair that is none of the three gets no name rather than the nearest one:
// read-only that still asks is a legitimate configuration, and calling it plan
// would put a word on the bar the engine does not answer to.
func modeFrom(sandbox policy.SandboxMode, pol policy.ApprovalPolicy) string {
	for _, name := range []string{protocol.ModePlan, protocol.ModeAssist, protocol.ModeAuto} {
		if m, p := modeToSandbox(name); m == sandbox && p == pol {
			return name
		}
	}
	return ""
}

// modeToSandbox maps a behavioural mode to the (SandboxMode, ApprovalPolicy)
// pair the engine actually runs under. Kept here rather than in policy because
// it is the session that decides what each mode MEANS — the policy package
// stays technical.
func modeToSandbox(name string) (policy.SandboxMode, policy.ApprovalPolicy) {
	switch name {
	case protocol.ModePlan:
		return policy.ModeReadOnly, policy.PolicyNever
	case protocol.ModeAuto:
		return policy.ModeFullAccess, policy.PolicyOnRequest
	default:
		return policy.ModeWorkspaceWrite, policy.PolicyOnRequest
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
func (s *Session) Submit(text string, images ...ce.Image) error {
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
	// Checked after the states a caller can act on, because "closed" and "a
	// turn is running" are answers and this one is not. It cannot happen in a
	// running daemon, and that is why refusing matters: the failure would be a
	// nil dereference inside a goroutine, taking the whole process with it
	// rather than the one request that caused it.
	if s.engine == nil {
		s.mu.Unlock()
		return protocol.Errorf(protocol.CodeInternal, "session %s has no engine", s.ID)
	}

	// The turn's context is detached from any request: a client disconnecting
	// must not cancel work the user asked for.
	ctx, cancel := context.WithCancel(context.Background())
	s.state = protocol.SessionStateRunning
	s.cancel = cancel
	s.mu.Unlock()

	go func() {
		defer func() {
			// Anything still queued was meant for the turn that just ended.
			// Carrying it forward would deliver "no, do it the other way"
			// about work the model is no longer doing, so it is dropped and
			// said out loud rather than silently.
			if left := s.dropSteering(); len(left) > 0 {
				s.Emit(protocol.EventSessionError, protocol.Error{
					Code: protocol.CodeNoActiveTurn,
					Message: fmt.Sprintf("the turn ended before %d message(s) reached it; send them again",
						len(left)),
				})
			}
			s.mu.Lock()
			if s.state != protocol.SessionStateClosed {
				s.state = protocol.SessionStateIdle
			}
			s.cancel = nil
			s.failPending()
			s.mu.Unlock()
			cancel()
		}()
		_, _ = s.engine.Run(ctx, text, images...)
	}()
	return nil
}

// Exec runs a command the person typed, outside any turn.
//
// It takes the same lock Submit does and refuses for the same reasons: a
// command run beside a turn would interleave its tool events with the turn's
// and edit the history the turn is in the middle of reading.
//
// Synchronous, unlike Submit. A typed command is short and the person is
// waiting on its output; there is nothing to watch in the meantime.
func (s *Session) Exec(ctx context.Context, command string) error {
	s.mu.Lock()
	switch {
	case s.state == protocol.SessionStateClosed:
		s.mu.Unlock()
		return protocol.Errorf(protocol.CodeSessionNotFound, "session %s is closed", s.ID)
	case s.state != protocol.SessionStateIdle:
		s.mu.Unlock()
		return protocol.Errorf(protocol.CodeTurnAlreadyActive,
			"a turn is running; wait for it to finish or interrupt it")
	case s.engine == nil:
		s.mu.Unlock()
		return protocol.Errorf(protocol.CodeInternal, "session %s has no engine", s.ID)
	}
	s.state = protocol.SessionStateRunning
	engine := s.engine
	s.mu.Unlock()

	_, err := engine.Exec(ctx, command)

	s.mu.Lock()
	if s.state != protocol.SessionStateClosed {
		s.state = protocol.SessionStateIdle
	}
	s.mu.Unlock()
	return err
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

// Undo puts back what the last turn changed.
//
// Refused while a turn is running: the files are moving under it, and undoing
// half of something still being written is worse than waiting.
func (s *Session) Undo() (protocol.UndoResult, error) {
	s.mu.Lock()
	state := s.state
	s.mu.Unlock()
	if state == protocol.SessionStateRunning {
		return protocol.UndoResult{}, protocol.Errorf(protocol.CodeTurnAlreadyActive,
			"a turn is running; interrupt it before undoing")
	}
	if s.engine == nil {
		return protocol.UndoResult{}, nil
	}
	restored, refused, err := s.engine.Undo()
	if err != nil {
		return protocol.UndoResult{}, protocol.Errorf(protocol.CodeInternal, "%v", err)
	}
	return protocol.UndoResult{Restored: restored, Refused: refused}, nil
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
	// Cancelling the context ends the turn; it does not end what the turn
	// deliberately started outside itself. A background command is bound to
	// the session rather than to a turn precisely so it survives between them,
	// which means the session is what has to end it.
	if s.engine != nil {
		s.engine.Close()
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

// maxLapsed bounds what is remembered about approvals nobody answered.
//
// A daemon runs for weeks and a set that grows once per approval is a leak with
// a long fuse. The bound is generous for what it is for — a client that was away
// coming back to answer — and anything older than that is a question whose
// context is gone anyway.
const maxLapsed = 64

// noteLapsed records that an approval ran out of time, keeping the most recent.
func (s *Session) noteLapsed(id string) {
	if s.lapsed == nil {
		s.lapsed = map[string]struct{}{}
	}
	s.lapsed[id] = struct{}{}
	s.lapsedOrder = append(s.lapsedOrder, id)
	for len(s.lapsedOrder) > maxLapsed {
		delete(s.lapsed, s.lapsedOrder[0])
		s.lapsedOrder = s.lapsedOrder[1:]
	}
}

func (s *Session) lapsedCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.lapsed)
}

// grantKey is what an "allow for the session" answer is remembered against.
//
// The rule when a rule asked, so approving a write to `.git/config` covers
// `.git/**` rather than that one file. Editing three files under a directory is
// one decision, not three, and three prompts is how people learn to approve
// without reading.
//
// Without a rule it stays the exact tool and command, which is the conservative
// key: a shell command is opaque, and "the same kind of command" is not
// something this can judge.
func grantKey(req protocol.ApprovalRequest) string {
	if req.Rule != "" {
		return req.Tool + "\x00rule:" + req.Rule
	}
	return req.Tool + "\x00" + req.Command
}

// Approve is the loop's side of a boundary crossing: it registers the question
// and blocks until a client answers or the deadline passes.
func (s *Session) Approve(ctx context.Context, req protocol.ApprovalRequest, timeout time.Duration) (
	protocol.ApprovalDecision, error,
) {
	// A question the user already answered, in a previous session or a previous
	// week. Checked before anything is registered, so nothing is ever pending
	// for a crossing nobody needs to be asked about.
	if s.Standing != nil {
		// Any recorded decision answers, including a refusal. "No" is a
		// decision, and asking again until the answer changes is not a boundary.
		if d := s.Standing.Granted(req); d.Valid() {
			return d, nil
		}
	}

	key := grantKey(req)
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
		// Every granting answer is forwarded, not only the standing ones. An
		// "allow once" still has to take effect for the command being asked
		// about, and only the layer that owns the record knows which answers
		// outlive the session and which merely open it.
		//
		// A failure to write must not cancel the decision: the user answered,
		// the answer applies now, and what is lost is that they will be asked
		// again next time — the safe direction to fail in.
		if d.Grants() && s.Standing != nil {
			_ = s.Standing.Remember(req, d)
		}
		return d, nil
	case <-timer:
		// Nobody answered in time. Denying is the only safe reading: the
		// alternative would be granting because everyone looked away.
		//
		// Recorded, because a client that comes back to answer must be told
		// which of the two happened: a decision was made, or the deadline made
		// one. They are opposite facts about whether the work went ahead.
		s.mu.Lock()
		s.noteLapsed(req.ApprovalID)
		s.mu.Unlock()
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
		_, lapsed := s.lapsed[approvalID]
		s.mu.Unlock()
		if lapsed {
			return protocol.Errorf(protocol.CodeApprovalExpired,
				"approval %s ran out of time and was denied; nobody answered it", approvalID)
		}
		return protocol.Errorf(protocol.CodeApprovalResolved,
			"approval %s is not pending; it was already answered", approvalID)
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

// LiveIDs is every session the manager currently holds.
//
// It exists for pruning: a record being written is not history, and deciding
// that from the file alone would mean guessing from a timestamp that is
// changing as you read it.
func (m *Manager) LiveIDs() map[string]bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make(map[string]bool, len(m.sessions))
	for id := range m.sessions {
		out[id] = true
	}
	return out
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

// Standing is the record of decisions that outlive a session.
//
// Deliberately ignorant of what a boundary means. This package knows a crossing
// was declared and that someone must answer; which crossings are worth
// remembering, and where an answer is kept, belong to the layer that knows both
// the policy and the user's configuration.
type Standing interface {
	// Granted reports a decision already made for this crossing, or an empty
	// decision when the question still has to be asked.
	Granted(req protocol.ApprovalRequest) protocol.ApprovalDecision
	// Remember writes down an answer meant to outlive the session.
	Remember(req protocol.ApprovalRequest, d protocol.ApprovalDecision) error
}
