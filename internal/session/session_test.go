package session

import (
	"context"
	"fmt"
	ce "github.com/aguinelo/dcode/internal/contextengine"
	"github.com/aguinelo/dcode/internal/loop"
	"github.com/aguinelo/dcode/internal/policy"
	"github.com/aguinelo/dcode/internal/provider"
	"github.com/aguinelo/dcode/internal/tools"
	"sync"
	"testing"
	"time"

	"github.com/aguinelo/dcode/internal/protocol"
)

func fixedClock() Clock {
	t := time.Date(2026, 8, 8, 12, 0, 0, 0, time.UTC)
	return func() time.Time { return t }
}

func newLog(t *testing.T, retention int) *EventLog {
	t.Helper()
	return NewEventLog("s1", retention, fixedClock())
}

// The invariant the whole log rests on: strictly increasing, no gaps, no reuse,
// under concurrent writers. Assigning the sequence outside the lock is the
// classic way to break this, and it only shows up under load.
func TestSequenceIsGaplessUnderConcurrency(t *testing.T) {
	l := newLog(t, 0)
	const writers, each = 20, 50

	var wg sync.WaitGroup
	for w := 0; w < writers; w++ {
		wg.Add(1)
		go func(w int) {
			defer wg.Done()
			for i := 0; i < each; i++ {
				if _, err := l.Append(protocol.EventMessageDelta,
					protocol.MessageDelta{TurnID: fmt.Sprintf("t%d", w), Text: "x"}); err != nil {
					t.Error(err)
				}
			}
		}(w)
	}
	wg.Wait()

	events, err := l.Replay(1)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != writers*each {
		t.Fatalf("want %d events, got %d", writers*each, len(events))
	}
	for i, ev := range events {
		if ev.Seq != uint64(i+1) {
			t.Fatalf("gap or reuse at index %d: seq %d", i, ev.Seq)
		}
	}
}

func TestSequenceStartsAtOne(t *testing.T) {
	l := newLog(t, 0)
	ev, err := l.Append(protocol.EventTurnStarted, protocol.TurnStarted{TurnID: "t1"})
	if err != nil {
		t.Fatal(err)
	}
	if ev.Seq != 1 {
		t.Errorf("got %d want 1", ev.Seq)
	}
	if ev.SessionID != "s1" {
		t.Errorf("got %q", ev.SessionID)
	}
}

// Replay and live observation must produce the same sequence. If they diverge,
// resuming a session would show a different history than watching it live.
func TestReplayMatchesLiveObservation(t *testing.T) {
	l := newLog(t, 0)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	live, stop, err := l.Subscribe(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}
	defer stop()

	const n = 30
	go func() {
		for i := 0; i < n; i++ {
			_, _ = l.Append(protocol.EventMessageDelta,
				protocol.MessageDelta{TurnID: "t1", Text: fmt.Sprintf("%d", i)})
		}
	}()

	var observed []uint64
	timeout := time.After(3 * time.Second)
	for len(observed) < n {
		select {
		case ev := <-live:
			observed = append(observed, ev.Seq)
		case <-timeout:
			t.Fatalf("only saw %d of %d events", len(observed), n)
		}
	}

	replayed, err := l.Replay(1)
	if err != nil {
		t.Fatal(err)
	}
	if len(replayed) != len(observed) {
		t.Fatalf("replay has %d, live had %d", len(replayed), len(observed))
	}
	for i := range observed {
		if replayed[i].Seq != observed[i] {
			t.Fatalf("divergence at %d: replay %d, live %d",
				i, replayed[i].Seq, observed[i])
		}
	}
}

// The join between backlog and live feed is the delicate part: too early
// duplicates, too late leaves a gap.
func TestSubscribeJoinsBacklogAndLiveWithoutGapOrDuplicate(t *testing.T) {
	l := newLog(t, 0)
	for i := 0; i < 10; i++ {
		_, _ = l.Append(protocol.EventMessageDelta, protocol.MessageDelta{Text: "old"})
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ch, stop, err := l.Subscribe(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}
	defer stop()

	for i := 0; i < 10; i++ {
		_, _ = l.Append(protocol.EventMessageDelta, protocol.MessageDelta{Text: "new"})
	}

	seen := map[uint64]int{}
	var order []uint64
	timeout := time.After(3 * time.Second)
	for len(order) < 20 {
		select {
		case ev := <-ch:
			seen[ev.Seq]++
			order = append(order, ev.Seq)
		case <-timeout:
			t.Fatalf("only saw %d of 20: %v", len(order), order)
		}
	}
	for seq, count := range seen {
		if count != 1 {
			t.Errorf("seq %d delivered %d times", seq, count)
		}
	}
	for i, seq := range order {
		if seq != uint64(i+1) {
			t.Fatalf("out of order at %d: %v", i, order)
		}
	}
}

func TestSubscribeFromLaterSequenceSkipsEarlier(t *testing.T) {
	l := newLog(t, 0)
	for i := 0; i < 5; i++ {
		_, _ = l.Append(protocol.EventMessageDelta, protocol.MessageDelta{Text: "x"})
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ch, stop, err := l.Subscribe(ctx, 4)
	if err != nil {
		t.Fatal(err)
	}
	defer stop()

	select {
	case ev := <-ch:
		if ev.Seq != 4 {
			t.Errorf("got %d want 4", ev.Seq)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("nothing delivered")
	}
}

// Asking for history that was dropped must be an error, not a quiet start
// later: the client would otherwise believe it had everything.
func TestExpiredReplayIsRefused(t *testing.T) {
	l := newLog(t, 5)
	for i := 0; i < 20; i++ {
		_, _ = l.Append(protocol.EventMessageDelta, protocol.MessageDelta{Text: "x"})
	}
	_, err := l.Replay(1)
	if err == nil {
		t.Fatal("replaying dropped history must fail")
	}
	pe, ok := protocol.AsError(err)
	if !ok || pe.Code != protocol.CodeEventsExpired {
		t.Errorf("got %v", err)
	}
	if _, _, err := l.Subscribe(context.Background(), 1); err == nil {
		t.Error("subscribing below retention must fail the same way")
	}
	// What is still held must still be replayable.
	if _, err := l.Replay(16); err != nil {
		t.Errorf("recent history should still replay: %v", err)
	}
}

// A client that stops reading must never hold up the agent. It is dropped and
// rejoins with `from`, which is why the client owns its read position.
func TestSlowSubscriberIsDroppedAndDoesNotBlockAppends(t *testing.T) {
	l := newLog(t, 0)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if _, _, err := l.Subscribe(ctx, 1); err != nil {
		t.Fatal(err)
	}

	done := make(chan struct{})
	go func() {
		defer close(done)
		for i := 0; i < 2000; i++ {
			if _, err := l.Append(protocol.EventMessageDelta,
				protocol.MessageDelta{Text: "flood"}); err != nil {
				t.Error(err)
				return
			}
		}
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("a subscriber that stopped reading blocked the log")
	}
}

func TestUnsubscribeStopsDelivery(t *testing.T) {
	l := newLog(t, 0)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	_, stop, err := l.Subscribe(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}
	if l.Subscribers() != 1 {
		t.Fatalf("got %d subscribers", l.Subscribers())
	}
	stop()
	if l.Subscribers() != 0 {
		t.Errorf("got %d subscribers after stopping", l.Subscribers())
	}
	stop() // must be idempotent; a client closing twice is routine
}

func TestCloseDropsSubscribers(t *testing.T) {
	l := newLog(t, 0)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if _, _, err := l.Subscribe(ctx, 1); err != nil {
		t.Fatal(err)
	}
	l.Close()
	if l.Subscribers() != 0 {
		t.Errorf("got %d", l.Subscribers())
	}
}

func TestAppendRejectsUnserialisablePayload(t *testing.T) {
	l := newLog(t, 0)
	if _, err := l.Append(protocol.EventSessionError, make(chan int)); err == nil {
		t.Error("a payload that cannot be encoded must fail rather than store garbage")
	}
	// And the sequence must not have advanced past a rejected append.
	if l.LastSeq() != 0 {
		t.Errorf("a failed append consumed a sequence number: %d", l.LastSeq())
	}
}

// ---------- session ----------

func newSession(t *testing.T) *Session {
	t.Helper()
	log := newLog(t, 0)
	return New("s1", "/w", "MiniMax-M3", "workspace-write", nil, log, fixedClock())
}

// Two clients answering the same approval: exactly one wins, the other is told
// so. Both believing they decided is the failure to avoid.
func TestFirstApprovalAnswerWinsAndTheOtherConflicts(t *testing.T) {
	s := newSession(t)
	req := protocol.ApprovalRequest{ApprovalID: "a1", Tool: "bash", Command: "curl x"}

	answered := make(chan protocol.ApprovalDecision, 1)
	go func() {
		d, _ := s.Approve(context.Background(), req, 2*time.Second)
		answered <- d
	}()

	// Wait for the question to register.
	deadline := time.After(2 * time.Second)
	for len(s.Pending()) == 0 {
		select {
		case <-deadline:
			t.Fatal("the approval never registered")
		default:
			time.Sleep(time.Millisecond)
		}
	}

	var okCount, conflictCount int
	var mu sync.Mutex
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := s.Resolve("a1", protocol.ApprovalAllow)
			mu.Lock()
			defer mu.Unlock()
			if err == nil {
				okCount++
				return
			}
			if pe, ok := protocol.AsError(err); ok && pe.Code == protocol.CodeApprovalResolved {
				conflictCount++
			}
		}()
	}
	wg.Wait()

	if okCount != 1 {
		t.Errorf("exactly one answer should be accepted, got %d", okCount)
	}
	if conflictCount != 7 {
		t.Errorf("the rest should conflict, got %d", conflictCount)
	}
	select {
	case d := <-answered:
		if d != protocol.ApprovalAllow {
			t.Errorf("got %s", d)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("the loop was never unblocked")
	}
}

// Nobody answered. Denying is the only safe reading: the alternative is
// granting because everyone looked away.
func TestUnansweredApprovalExpiresAsDenied(t *testing.T) {
	s := newSession(t)
	start := time.Now()
	d, err := s.Approve(context.Background(),
		protocol.ApprovalRequest{ApprovalID: "a1"}, 50*time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	if d != protocol.ApprovalDeny {
		t.Errorf("got %s want deny", d)
	}
	if time.Since(start) > 2*time.Second {
		t.Error("the deadline did not fire")
	}
	if len(s.Pending()) != 0 {
		t.Error("an expired approval must not stay pending")
	}
}

// allow_session is the one answer that outlives a single crossing, and it must
// only apply to the same tool and command.
func TestAllowForTheSessionAppliesToTheSameCommandOnly(t *testing.T) {
	s := newSession(t)
	req := protocol.ApprovalRequest{ApprovalID: "a1", Tool: "bash", Command: "go test ./..."}

	go func() {
		for len(s.Pending()) == 0 {
			time.Sleep(time.Millisecond)
		}
		_ = s.Resolve("a1", protocol.ApprovalAllowSession)
	}()
	if d, _ := s.Approve(context.Background(), req, 2*time.Second); d != protocol.ApprovalAllowSession {
		t.Fatalf("got %s", d)
	}

	// The same command now passes without asking.
	req2 := protocol.ApprovalRequest{ApprovalID: "a2", Tool: "bash", Command: "go test ./..."}
	done := make(chan protocol.ApprovalDecision, 1)
	go func() { d, _ := s.Approve(context.Background(), req2, time.Second); done <- d }()
	select {
	case d := <-done:
		if d != protocol.ApprovalAllowSession {
			t.Errorf("got %s", d)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("the remembered decision was not applied")
	}

	// A different command must still ask.
	req3 := protocol.ApprovalRequest{ApprovalID: "a3", Tool: "bash", Command: "rm -rf /"}
	start := time.Now()
	if d, _ := s.Approve(context.Background(), req3, 60*time.Millisecond); d != protocol.ApprovalDeny {
		t.Errorf("a different command must not inherit the decision, got %s", d)
	}
	if time.Since(start) < 40*time.Millisecond {
		t.Error("it should have waited for an answer rather than reusing one")
	}
}

func TestResolvingAnUnknownApprovalConflicts(t *testing.T) {
	s := newSession(t)
	err := s.Resolve("nope", protocol.ApprovalAllow)
	if pe, ok := protocol.AsError(err); !ok || pe.Code != protocol.CodeApprovalResolved {
		t.Errorf("got %v", err)
	}
}

func TestResolveRejectsAnUnknownDecision(t *testing.T) {
	s := newSession(t)
	if err := s.Resolve("a1", protocol.ApprovalDecision("maybe")); err == nil {
		t.Error("an unknown decision must be rejected")
	}
}

// A turn ending while someone is deciding must not leave the question open.
func TestClosingDeniesPendingApprovals(t *testing.T) {
	s := newSession(t)
	done := make(chan protocol.ApprovalDecision, 1)
	go func() {
		d, _ := s.Approve(context.Background(),
			protocol.ApprovalRequest{ApprovalID: "a1"}, 10*time.Second)
		done <- d
	}()
	for len(s.Pending()) == 0 {
		time.Sleep(time.Millisecond)
	}
	s.Close()

	select {
	case d := <-done:
		if d != protocol.ApprovalDeny {
			t.Errorf("got %s want deny", d)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("closing left the approval hanging")
	}
}

func TestDescribeReportsState(t *testing.T) {
	s := newSession(t)
	d := s.Describe()
	if d.ID != "s1" || d.Workspace != "/w" || d.State != protocol.SessionStateIdle {
		t.Errorf("got %+v", d)
	}
	if !d.State.Valid() {
		t.Error("the reported state must be a valid one")
	}
}

func TestSubmitOnAClosedSessionFails(t *testing.T) {
	s := newSession(t)
	s.Close()
	err := s.Submit("go")
	if pe, ok := protocol.AsError(err); !ok || pe.Code != protocol.CodeSessionNotFound {
		t.Errorf("got %v", err)
	}
}

func TestInterruptIsIdempotent(t *testing.T) {
	s := newSession(t)
	// Interrupting an idle session is not an error: the user cannot know the
	// turn just finished.
	s.Interrupt()
	s.Interrupt()
}

func TestEmitAppendsToTheLog(t *testing.T) {
	s := newSession(t)
	s.Emit(protocol.EventTurnStarted, protocol.TurnStarted{TurnID: "t1"})
	if s.Log.LastSeq() != 1 {
		t.Errorf("got %d", s.Log.LastSeq())
	}
}

// ---------- manager ----------

func TestManagerEnforcesTheCeiling(t *testing.T) {
	m := NewManager(2)
	for i := 0; i < 2; i++ {
		s := New(fmt.Sprintf("s%d", i), "/w", "m", "workspace-write", nil, newLog(t, 0), fixedClock())
		if err := m.Add(s); err != nil {
			t.Fatal(err)
		}
	}
	err := m.Add(New("s3", "/w", "m", "workspace-write", nil, newLog(t, 0), fixedClock()))
	if pe, ok := protocol.AsError(err); !ok || pe.Code != protocol.CodeMaxSessionsReached {
		t.Errorf("got %v", err)
	}
}

func TestManagerListIsInCreationOrder(t *testing.T) {
	m := NewManager(10)
	for _, id := range []string{"c", "a", "b"} {
		if err := m.Add(New(id, "/w", "m", "workspace-write", nil, newLog(t, 0), fixedClock())); err != nil {
			t.Fatal(err)
		}
	}
	got := m.List()
	// Creation order, not map order: a list that reshuffles between calls is
	// unusable in a client.
	for i, want := range []string{"c", "a", "b"} {
		if got[i].ID != want {
			t.Errorf("position %d: got %s want %s", i, got[i].ID, want)
		}
	}
}

func TestManagerRemove(t *testing.T) {
	m := NewManager(10)
	s := New("s1", "/w", "m", "workspace-write", nil, newLog(t, 0), fixedClock())
	if err := m.Add(s); err != nil {
		t.Fatal(err)
	}
	if err := m.Remove("s1"); err != nil {
		t.Fatal(err)
	}
	if m.Count() != 0 {
		t.Errorf("got %d", m.Count())
	}
	if err := m.Remove("s1"); err == nil {
		t.Error("removing twice must fail")
	}
	if _, err := m.Get("s1"); err == nil {
		t.Error("a removed session must not resolve")
	}
}

func TestManagerCloseAll(t *testing.T) {
	m := NewManager(10)
	for i := 0; i < 3; i++ {
		if err := m.Add(New(fmt.Sprintf("s%d", i), "/w", "m", "workspace-write",
			nil, newLog(t, 0), fixedClock())); err != nil {
			t.Fatal(err)
		}
	}
	m.CloseAll()
	if m.Count() != 0 {
		t.Errorf("got %d", m.Count())
	}
}

func TestManagerDefaultsToAUsableCeiling(t *testing.T) {
	if NewManager(0).max <= 0 {
		t.Error("a non-positive ceiling must fall back to a usable default")
	}
}

// IDs must sort by time so a session list and a directory listing agree.
func TestNewIDIsTimeOrderedAndFilenameSafe(t *testing.T) {
	base := time.Date(2026, 8, 8, 12, 0, 0, 0, time.UTC)
	var counter uint32
	entropy := func() uint32 { counter++; return counter }

	var ids []string
	for i := 0; i < 5; i++ {
		at := base.Add(time.Duration(i) * time.Second)
		ids = append(ids, NewID(func() time.Time { return at }, entropy))
	}
	for i := 1; i < len(ids); i++ {
		if ids[i] <= ids[i-1] {
			t.Errorf("ids must sort by time: %q then %q", ids[i-1], ids[i])
		}
	}
	for _, id := range ids {
		for _, c := range id {
			if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
				t.Errorf("id %q contains %q, which is not filename-safe", id, c)
			}
		}
	}
	if NewID(nil, nil) == "" {
		t.Error("NewID must work with defaults")
	}
}

// ---------- turns ----------

// idleProvider answers one turn with a single line of text and no tool calls,
// which is the shortest possible complete turn.
type idleProvider struct {
	started chan struct{}
	release chan struct{}
}

func (p *idleProvider) Family() provider.Family       { return provider.MiniMaxM3{} }
func (p *idleProvider) Transport() provider.Transport { return nil }
func (p *idleProvider) Window(string) (int, error)    { return 100000, nil }
func (p *idleProvider) Limits() provider.Limits       { return provider.Limits{MaxIterations: 5} }

func (p *idleProvider) Stream(ctx context.Context, _ provider.Request) (<-chan provider.StreamEvent, error) {
	ch := make(chan provider.StreamEvent, 4)
	go func() {
		defer close(ch)
		if p.started != nil {
			select {
			case p.started <- struct{}{}:
			default:
			}
		}
		if p.release != nil {
			select {
			case <-p.release:
			case <-ctx.Done():
				ch <- provider.StreamEvent{Type: provider.EventError, Err: &provider.ProviderError{
					Class: provider.ErrClassCanceled, Message: "cancelled",
				}}
				return
			}
		}
		ch <- provider.StreamEvent{Type: provider.EventTextDelta, Text: "done"}
		ch <- provider.StreamEvent{Type: provider.EventDone, Usage: &provider.Usage{}}
	}()
	return ch, nil
}

func sessionWithEngine(t *testing.T, p provider.Provider) *Session {
	t.Helper()
	res, err := policy.NewResolver(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	log := newLog(t, 0)
	s := New("s1", "/w", "m", "workspace-write", nil, log, fixedClock())
	s.engine = loop.New(loop.Config{
		Provider: p,
		Tools:    tools.NewRegistry(),
		State:    tools.NewState(res, tools.DefaultLimits()),
		Emitter:  s,
		Limits:   loop.DefaultLimits(),
		Mode:     policy.ModeWorkspaceWrite,
		Policy:   policy.PolicyOnRequest,
		Model:    "m",
	}, ce.Session{Instructions: "x"})
	return s
}

func waitState(t *testing.T, s *Session, want protocol.SessionState) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if s.State() == want {
			return
		}
		time.Sleep(2 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %s, session is %s", want, s.State())
}

func TestSubmitRunsATurnAndReturnsToIdle(t *testing.T) {
	s := sessionWithEngine(t, &idleProvider{})
	if err := s.Submit("hello"); err != nil {
		t.Fatal(err)
	}
	waitState(t, s, protocol.SessionStateIdle)

	// The turn is observable through the log, which is the only thing a client
	// ever sees.
	evs, err := s.Log.Replay(1)
	if err != nil {
		t.Fatal(err)
	}
	var started, completed bool
	for _, e := range evs {
		switch e.Type {
		case protocol.EventTurnStarted:
			started = true
		case protocol.EventTurnCompleted:
			completed = true
		}
	}
	if !started || !completed {
		t.Errorf("got %d events, started=%v completed=%v", len(evs), started, completed)
	}
}

// One turn per session: the queue belongs to the client, which is what keeps
// the event log linearly ordered.
func TestSubmitRefusesAConcurrentTurn(t *testing.T) {
	p := &idleProvider{started: make(chan struct{}, 1), release: make(chan struct{})}
	s := sessionWithEngine(t, p)

	if err := s.Submit("first"); err != nil {
		t.Fatal(err)
	}
	<-p.started

	err := s.Submit("second")
	pe, ok := protocol.AsError(err)
	if !ok || pe.Code != protocol.CodeTurnAlreadyActive {
		t.Fatalf("got %v", err)
	}

	close(p.release)
	waitState(t, s, protocol.SessionStateIdle)
}

// A client disconnecting must not cancel work the user asked for, so the turn's
// context is detached; only an explicit interrupt stops it.
func TestInterruptStopsARunningTurn(t *testing.T) {
	p := &idleProvider{started: make(chan struct{}, 1), release: make(chan struct{})}
	s := sessionWithEngine(t, p)

	if err := s.Submit("go"); err != nil {
		t.Fatal(err)
	}
	<-p.started
	s.Interrupt()
	waitState(t, s, protocol.SessionStateIdle)
}

func TestStateReportsClosed(t *testing.T) {
	s := sessionWithEngine(t, &idleProvider{})
	if s.State() != protocol.SessionStateIdle {
		t.Fatalf("got %s", s.State())
	}
	s.Close()
	if s.State() != protocol.SessionStateClosed {
		t.Errorf("got %s", s.State())
	}
	// A closed session stays closed even if a turn was in flight.
	if err := s.Submit("x"); err == nil {
		t.Error("a closed session accepts nothing")
	}
}

func TestNewFillsInAClock(t *testing.T) {
	s := New("s1", "/w", "m", "read-only", nil, newLog(t, 0), nil)
	if s.CreatedAt.IsZero() {
		t.Error("a session without a clock still has a creation time")
	}
}

func TestNewEventLogFillsInDefaults(t *testing.T) {
	log := NewEventLog("s1", 0, nil)
	if _, err := log.Append(protocol.EventTurnStarted, protocol.TurnStarted{TurnID: "t1"}); err != nil {
		t.Fatal(err)
	}
	evs, err := log.Replay(1)
	if err != nil {
		t.Fatal(err)
	}
	if len(evs) != 1 || evs[0].At.IsZero() {
		t.Errorf("got %+v", evs)
	}
}

// The other half of the grant key, and the half that decides how often a person
// is asked: when a RULE produced the question, the grant is keyed by the rule.
//
// Approving `.git/config` therefore also covers `.git/hooks/pre-commit`. That
// is the point — the user said yes to "writing inside .git", which is what they
// were actually shown, and asking again for the next path under the same rule
// is how people learn to approve without reading.
func TestAllowForTheSessionIsKeyedByTheRuleThatAsked(t *testing.T) {
	s := newSession(t)
	ctx := context.Background()

	first := protocol.ApprovalRequest{
		ApprovalID: "a1", Tool: "write", Command: "",
		Rule: ".git/**", Reason: "writing inside .git",
	}
	go func() {
		for len(s.Pending()) == 0 {
			time.Sleep(time.Millisecond)
		}
		_ = s.Resolve("a1", protocol.ApprovalAllowSession)
	}()
	if d, err := s.Approve(ctx, first, 2*time.Second); err != nil || d != protocol.ApprovalAllowSession {
		t.Fatalf("first approval = %v, %v", d, err)
	}

	// A different path, same rule: already answered.
	same := protocol.ApprovalRequest{
		ApprovalID: "a2", Tool: "write", Command: "",
		Rule: ".git/**", Reason: "writing inside .git",
	}
	d, err := s.Approve(ctx, same, 100*time.Millisecond)
	if err != nil || d != protocol.ApprovalAllowSession {
		t.Errorf("the same rule asked again: %v, %v — the user answered this question", d, err)
	}

	// A different rule is a different question, and must still be asked. Without
	// this half, one grant would quietly become a grant for everything.
	other := protocol.ApprovalRequest{
		ApprovalID: "a3", Tool: "write", Command: "",
		Rule: "vendor/**", Reason: "writing inside vendor",
	}
	go func() {
		for len(s.Pending()) == 0 {
			time.Sleep(time.Millisecond)
		}
		_ = s.Resolve("a3", protocol.ApprovalDeny)
	}()
	if d, err := s.Approve(ctx, other, 2*time.Second); err != nil || d != protocol.ApprovalDeny {
		t.Errorf("a different rule was answered by the earlier grant: %v, %v", d, err)
	}
}

// An approval that nobody answered in time and one that another client answered
// are opposite facts, and the client was told the same thing for both. The
// message even named both possibilities — "it was already answered or it
// expired" — because the code could not tell which.
//
// The difference is what the user does next. "Already answered" says a decision
// was made and the work went ahead on it. "Expired" says nobody decided, so it
// was denied and the work did not happen. A client showing the first when the
// second is true tells them the opposite of what occurred.
func TestAnApprovalThatRanOutOfTimeSaysSoRatherThanClaimingSomeoneAnsweredIt(t *testing.T) {
	s := newSession(t)
	req := protocol.ApprovalRequest{ApprovalID: "a1", Tool: "bash", Command: "curl x"}

	// Nobody answers; the deadline denies it.
	if d, err := s.Approve(context.Background(), req, 10*time.Millisecond); err != nil || d != protocol.ApprovalDeny {
		t.Fatalf("a lapsed approval resolved to %v, %v", d, err)
	}

	// A client that was away comes back and tries to answer.
	err := s.Resolve("a1", protocol.ApprovalAllow)
	if err == nil {
		t.Fatal("a lapsed approval accepted an answer")
	}
	if got := codeOf(err); got != protocol.CodeApprovalExpired {
		t.Errorf("code = %q, want %q — the client is told a decision was made when "+
			"the deadline denied it", got, protocol.CodeApprovalExpired)
	}
}

// The other half, and the reason the two need separate codes at all: a genuine
// second answer still reports a conflict.
func TestAnApprovalSomebodyAnsweredStillReportsAConflict(t *testing.T) {
	s := newSession(t)
	req := protocol.ApprovalRequest{ApprovalID: "a1", Tool: "bash", Command: "curl x"}

	go func() {
		for len(s.Pending()) == 0 {
			time.Sleep(time.Millisecond)
		}
		_ = s.Resolve("a1", protocol.ApprovalAllow)
	}()
	if _, err := s.Approve(context.Background(), req, 2*time.Second); err != nil {
		t.Fatal(err)
	}

	err := s.Resolve("a1", protocol.ApprovalDeny)
	if got := codeOf(err); got != protocol.CodeApprovalResolved {
		t.Errorf("code = %q, want %q", got, protocol.CodeApprovalResolved)
	}
}

// An id nobody ever asked about is neither expired nor answered. Reporting it as
// expired would invent a question the session never posed.
func TestAnApprovalNobodyEverAskedAboutIsNotReportedAsExpired(t *testing.T) {
	s := newSession(t)
	err := s.Resolve("never-existed", protocol.ApprovalAllow)
	if got := codeOf(err); got == protocol.CodeApprovalExpired {
		t.Error("an id the session never posed was reported as having expired")
	}
	if err == nil {
		t.Error("resolving an unknown approval succeeded")
	}
}

// The record of what lapsed is bounded. A daemon runs for weeks, and a set that
// grows once per approval is a leak with a long fuse.
func TestTheRecordOfLapsedApprovalsIsBounded(t *testing.T) {
	s := newSession(t)
	for i := 0; i < 200; i++ {
		req := protocol.ApprovalRequest{ApprovalID: fmt.Sprintf("a%d", i), Tool: "bash"}
		if _, err := s.Approve(context.Background(), req, time.Millisecond); err != nil {
			t.Fatal(err)
		}
	}
	if n := s.lapsedCount(); n > maxLapsed {
		t.Errorf("%d lapsed approvals remembered, cap is %d", n, maxLapsed)
	}
	// The most recent are the ones a returning client could still ask about.
	if got := codeOf(s.Resolve("a199", protocol.ApprovalAllow)); got != protocol.CodeApprovalExpired {
		t.Errorf("the most recent lapse was forgotten: %q", got)
	}
}

// codeOf reads the protocol code from an error, or "" when there is none.
func codeOf(err error) string {
	if pe, ok := protocol.AsError(err); ok {
		return pe.Code
	}
	return ""
}
