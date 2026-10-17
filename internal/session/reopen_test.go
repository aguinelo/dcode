package session

import (
	"bufio"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	ce "github.com/aguinelo/dcode/internal/contextengine"
	"github.com/aguinelo/dcode/internal/loop"
	"github.com/aguinelo/dcode/internal/policy"
	"github.com/aguinelo/dcode/internal/protocol"
	"github.com/aguinelo/dcode/internal/tools"
)

const reopenID = "0199aabbccd00000001"

var reopenCreated = time.Date(2026, 10, 17, 9, 0, 0, 0, time.UTC)

// recordSession writes a session's record the way a live one does: through its
// log, into its own file. open leaves the last turn started and not completed,
// which is what a daemon that stopped under it leaves.
func recordSession(t *testing.T, dir string, open bool) {
	t.Helper()
	rec, err := NewRecord(dir, reopenID)
	if err != nil {
		t.Fatal(err)
	}
	l := NewEventLog(reopenID, 0, fixedClock())
	l.SetRecord(rec)
	must := func(_ protocol.Event, err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	must(l.Append(protocol.EventSessionCreated, protocol.Session{
		ID: reopenID, State: protocol.SessionStateIdle, Workspace: "/w",
		Model: "m-1", Family: "minimax", Transport: "anthropic", BaseURL: "http://x",
		Branch: "main", SandboxMode: "read-only", Mode: "plan", CreatedAt: reopenCreated,
		ContextWindow: 200000, LoopID: "loop-1",
		ParentID: "0199aabbccd00000000", ProjectID: "a1b2c3d4e5f6a1b2c3d4e5f6",
		Role: protocol.RoleChild, Owns: []string{"internal/session/", "changelog.d/x.md"},
	}))
	must(l.Append(protocol.EventTurnStarted, protocol.TurnStarted{TurnID: "t1", Text: "hello"}))
	must(l.Append(protocol.EventMessageDelta, protocol.MessageDelta{TurnID: "t1", Text: "hi there"}))
	must(l.Append(protocol.EventTurnCompleted, protocol.TurnCompleted{TurnID: "t1", Reason: protocol.StopDone}))
	must(l.Append(protocol.EventSessionModeChanged, protocol.SessionModeChanged{
		Previous: "plan", Mode: "assist", SandboxMode: "workspace-write",
	}))
	must(l.Append(protocol.EventSessionRenamed, protocol.SessionRenamed{Name: "the orchestrator"}))
	if open {
		must(l.Append(protocol.EventTurnStarted, protocol.TurnStarted{TurnID: "t2", Text: "go on"}))
		must(l.Append(protocol.EventMessageDelta, protocol.MessageDelta{TurnID: "t2", Text: "working"}))
	}
	l.Close()
}

// reopenBare reopens with no engine, which is everything this package decides.
func reopenBare(m *Manager, dir string) (*Session, Reopening, error) {
	var got Reopening
	s, err := m.Reopen(dir, reopenID, 0, fixedClock(), func(r Reopening, log *EventLog) (*Session, error) {
		got = r
		return NewReopened(r, nil, log, fixedClock()), nil
	})
	return s, got, err
}

// recordedSeqs reads the sequence numbers the file holds, in file order.
func recordedSeqs(t *testing.T, dir string) []protocol.Event {
	t.Helper()
	f, err := os.Open(filepath.Join(dir, reopenID+".jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	var out []protocol.Event
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		var ev protocol.Event
		if json.Unmarshal(sc.Bytes(), &ev) != nil {
			continue
		}
		out = append(out, ev)
	}
	return out
}

// The .i's step 3 contract: reopening a recorded session gives back the same
// id, the same history and the next Seq with no gap — in memory and in the
// file, which is the one record of it.
func TestAReopenedSessionKeepsItsIdHistoryAndSequence(t *testing.T) {
	dir := t.TempDir()
	recordSession(t, dir, false)
	m := NewManager(4)

	s, r, err := reopenBare(m, dir)
	if err != nil {
		t.Fatal(err)
	}
	if s.ID != reopenID {
		t.Fatalf("reopened as %q, want %q", s.ID, reopenID)
	}
	if got, err := m.Get(reopenID); err != nil || got != s {
		t.Fatalf("the manager does not hold the reopened session under its id: %v", err)
	}
	want := []ce.Message{
		{Role: ce.RoleUser, Text: "hello"},
		{Role: ce.RoleAssistant, Text: "hi there"},
	}
	if len(r.History) != len(want) {
		t.Fatalf("history has %d messages, want %d: %+v", len(r.History), len(want), r.History)
	}
	for i := range want {
		if r.History[i].Role != want[i].Role || r.History[i].Text != want[i].Text {
			t.Errorf("history[%d] = %+v, want %+v", i, r.History[i], want[i])
		}
	}
	if r.Title != "the orchestrator" {
		t.Errorf("title %q, want the name the record gave it", r.Title)
	}

	ev, err := s.Log.Append(protocol.EventTurnStarted, protocol.TurnStarted{TurnID: "t2", Text: "again"})
	if err != nil {
		t.Fatal(err)
	}
	if ev.Seq != 7 || ev.SessionID != reopenID {
		t.Fatalf("next event is seq %d of %q, want 7 of %q", ev.Seq, ev.SessionID, reopenID)
	}
	s.Close()
	for i, ev := range recordedSeqs(t, dir) {
		if ev.Seq != uint64(i+1) {
			t.Fatalf("the record has seq %d at line %d: a gap or a repeat", ev.Seq, i+1)
		}
	}

	d := s.Describe()
	if d.Mode != "assist" || d.SandboxMode != "workspace-write" {
		t.Errorf("mode %s/%s, want the one the record last changed to", d.Mode, d.SandboxMode)
	}
	if d.Model != "m-1" || d.Family != "minimax" || d.BaseURL != "http://x" || d.ContextWindow != 200000 {
		t.Errorf("model bundle lost: %+v", d)
	}
	if d.LoopID != "loop-1" || d.Workspace != "/w" || !d.CreatedAt.Equal(reopenCreated) || d.Branch != "main" {
		t.Errorf("identity lost: %+v", d)
	}
}

// The orchestrator and its children are named by these: a reopen that lost
// them would bring back a session nobody's tree points at.
func TestAReopenedSessionKeepsItsPlaceUnderTheOrchestrator(t *testing.T) {
	dir := t.TempDir()
	recordSession(t, dir, false)
	s, _, err := reopenBare(NewManager(4), dir)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	d := s.Describe()
	if d.ParentID != "0199aabbccd00000000" || d.ProjectID != "a1b2c3d4e5f6a1b2c3d4e5f6" || d.Role != protocol.RoleChild {
		t.Errorf("placement lost: parent %q project %q role %q", d.ParentID, d.ProjectID, d.Role)
	}
	if len(d.Owns) != 2 || d.Owns[0] != "internal/session/" || d.Owns[1] != "changelog.d/x.md" {
		t.Errorf("owns %v, want the two the record holds", d.Owns)
	}
}

// A client attaching from 1 gets the whole session: what came before the
// reopen comes from the record.
func TestAReopenedSessionReplaysFromTheStart(t *testing.T) {
	dir := t.TempDir()
	recordSession(t, dir, false)
	s, _, err := reopenBare(NewManager(4), dir)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	evs, err := s.Log.Replay(1)
	if err != nil {
		t.Fatal(err)
	}
	if len(evs) != 6 || evs[0].Type != protocol.EventSessionCreated {
		t.Fatalf("replay from 1 gave %d events, want the 6 recorded", len(evs))
	}
	if d := s.Describe(); d.LastSeq != 6 || d.FirstSeq != 7 {
		t.Errorf("last %d first %d, want 6 and 7", d.LastSeq, d.FirstSeq)
	}
}

// The .i's step 3 contract: a session recorded in the middle of a turn
// reopens with daemon_restart as the last end of turn, before any new event.
func TestASessionRecordedMidTurnReopensWithDaemonRestart(t *testing.T) {
	dir := t.TempDir()
	recordSession(t, dir, true)
	s, r, err := reopenBare(NewManager(4), dir)
	if err != nil {
		t.Fatal(err)
	}
	if r.OpenTurn != "t2" || r.Turns != 2 {
		t.Errorf("open turn %q of %d, want t2 of 2", r.OpenTurn, r.Turns)
	}
	s.Emit(protocol.EventSessionError, protocol.Error{Code: "x", Message: "after"})
	s.Close()

	evs := recordedSeqs(t, dir)
	var last *protocol.TurnCompleted
	lastAt := 0
	for i, ev := range evs {
		if ev.Type == protocol.EventTurnCompleted {
			var d protocol.TurnCompleted
			if err := json.Unmarshal(ev.Payload, &d); err != nil {
				t.Fatal(err)
			}
			last, lastAt = &d, i
		}
	}
	if last == nil || last.Reason != protocol.StopDaemonRestart || last.TurnID != "t2" {
		t.Fatalf("last end of turn %+v, want t2 ended by daemon_restart", last)
	}
	if evs[lastAt].Seq != 9 || lastAt != len(evs)-2 {
		t.Errorf("daemon_restart is seq %d at %d of %d, want 9, the first new event",
			evs[lastAt].Seq, lastAt, len(evs))
	}
}

// Two sessions writing one record is a record with two sequences in it.
func TestALiveSessionIsNotReopened(t *testing.T) {
	dir := t.TempDir()
	recordSession(t, dir, false)
	m := NewManager(4)
	s, _, err := reopenBare(m, dir)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if _, _, err := reopenBare(m, dir); codeOf(err) != protocol.CodeInvalidInput {
		t.Fatalf("second reopen gave %v, want invalid_input", err)
	}
	if m.Count() != 1 {
		t.Errorf("%d sessions, want the one", m.Count())
	}
}

// A record that is gone — pruned, or never written — is said, and nothing
// opens: a fresh session under the old id would be the orchestrator with its
// memory wiped and nobody told.
func TestAMissingRecordIsNotReopenedAsAFreshSession(t *testing.T) {
	for name, dir := range map[string]string{"pruned": t.TempDir(), "recording off": ""} {
		t.Run(name, func(t *testing.T) {
			m := NewManager(4)
			built := false
			_, err := m.Reopen(dir, reopenID, 0, fixedClock(), func(r Reopening, log *EventLog) (*Session, error) {
				built = true
				return NewReopened(r, nil, log, fixedClock()), nil
			})
			if codeOf(err) != protocol.CodeSessionNotFound {
				t.Fatalf("got %v, want session_not_found", err)
			}
			if built || m.Count() != 0 {
				t.Errorf("built=%v count=%d: something opened", built, m.Count())
			}
		})
	}
}

func TestABadIdIsRefusedBeforeItBecomesAPath(t *testing.T) {
	_, err := NewManager(4).Reopen(t.TempDir(), "../x", 0, nil, nil)
	if codeOf(err) != protocol.CodeInvalidInput {
		t.Fatalf("got %v, want invalid_input", err)
	}
}

// A line in the middle that is not an event is a damaged record, and a reopen
// on top of it would build on a history nobody can vouch for.
func TestACorruptRecordIsNotReopened(t *testing.T) {
	cases := map[string]func(string) string{
		"line in the middle": func(s string) string {
			i := strings.Index(s, "\n")
			return s[:i+1] + "not json\n" + s[i+1:]
		},
		"no session.created": func(s string) string { return s[strings.Index(s, "\n")+1:] },
		"another session's line": func(s string) string {
			return s + `{"seq":9,"session_id":"other","type":"progress","at":"2026-10-17T09:00:00Z","payload":{}}` + "\n"
		},
	}
	for name, damage := range cases {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			recordSession(t, dir, false)
			path := filepath.Join(dir, reopenID+".jsonl")
			raw, _ := os.ReadFile(path)
			if err := os.WriteFile(path, []byte(damage(string(raw))), 0o600); err != nil {
				t.Fatal(err)
			}
			m := NewManager(4)
			if _, _, err := reopenBare(m, dir); codeOf(err) != protocol.CodeInternal {
				t.Fatalf("got %v, want internal", err)
			}
			if m.Count() != 0 {
				t.Error("a corrupt record opened")
			}
		})
	}
}

// The last line cut short is what a crash mid-write leaves. It is tolerated,
// and the first new event goes on a line of its own.
func TestATornLastLineIsEndedAndNotBuiltOn(t *testing.T) {
	dir := t.TempDir()
	recordSession(t, dir, false)
	path := filepath.Join(dir, reopenID+".jsonl")
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = f.WriteString(`{"seq":7,"session_id":"` + reopenID + `","type":"message.del`)
	_ = f.Close()

	s, _, err := reopenBare(NewManager(4), dir)
	if err != nil {
		t.Fatal(err)
	}
	ev, err := s.Log.Append(protocol.EventTurnStarted, protocol.TurnStarted{TurnID: "t2"})
	if err != nil {
		t.Fatal(err)
	}
	s.Close()
	evs := recordedSeqs(t, dir)
	if got := evs[len(evs)-1]; got.Seq != ev.Seq || ev.Seq != 7 {
		t.Fatalf("the new event is seq %d and the file's last readable is %d, want 7 for both", ev.Seq, got.Seq)
	}
}

// A builder that builds something else is caught: the point is the same id on
// the same record.
func TestABuilderThatBuildsAnotherSessionIsRefused(t *testing.T) {
	dir := t.TempDir()
	recordSession(t, dir, false)
	m := NewManager(4)
	_, err := m.Reopen(dir, reopenID, 0, fixedClock(), func(r Reopening, _ *EventLog) (*Session, error) {
		return New("other", "/w", "m", "", nil, NewEventLog("other", 0, nil), nil), nil
	})
	if codeOf(err) != protocol.CodeInternal || m.Count() != 0 {
		t.Fatalf("got %v with %d sessions, want internal and none", err, m.Count())
	}
	_, err = m.Reopen(dir, reopenID, 0, fixedClock(), func(Reopening, *EventLog) (*Session, error) {
		return nil, protocol.Errorf(protocol.CodeWorkspaceInvalid, "gone")
	})
	if codeOf(err) != protocol.CodeWorkspaceInvalid {
		t.Fatalf("the builder's own failure was not returned: %v", err)
	}
}

// A reopened engine numbers its next turn after the record's, so one log never
// holds two turns named t1.
func TestAReopenedEngineContinuesTheTurnNumbers(t *testing.T) {
	dir := t.TempDir()
	recordSession(t, dir, false)
	res, err := policy.NewResolver(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	s, err := NewManager(4).Reopen(dir, reopenID, 0, fixedClock(), func(r Reopening, log *EventLog) (*Session, error) {
		var sess *Session
		eng := loop.New(loop.Config{
			Provider: &idleProvider{}, Tools: tools.NewRegistry(),
			State:   tools.NewState(res, tools.DefaultLimits(), allToolNames),
			Emitter: emitTo(func() *Session { return sess }), Limits: loop.DefaultLimits(),
			Mode: policy.ModeWorkspaceWrite, Policy: policy.PolicyOnRequest, Model: "m",
		}, ce.Session{Instructions: "x", History: r.History})
		sess = NewReopened(r, eng, log, fixedClock())
		return sess, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if err := s.Submit("next"); err != nil {
		t.Fatal(err)
	}
	waitState(t, s, protocol.SessionStateIdle)
	evs, _ := s.Log.Replay(s.Log.FirstSeq())
	for _, ev := range evs {
		if ev.Type == protocol.EventTurnStarted {
			var d protocol.TurnStarted
			_ = json.Unmarshal(ev.Payload, &d)
			if d.TurnID != "t2" {
				t.Fatalf("the next turn is %q, want t2", d.TurnID)
			}
			return
		}
	}
	t.Fatal("no turn started")
}

type emitTo func() *Session

func (f emitTo) Emit(t protocol.EventType, payload any) {
	if s := f(); s != nil {
		s.Emit(t, payload)
	}
}

// SubmitOrigin: the turn says who started it, and an origin nobody defined is
// refused before the turn starts.
func TestATurnSaysWhoStartedIt(t *testing.T) {
	s := sessionWithEngine(t, &idleProvider{})
	if err := s.SubmitOrigin("reports", "nobody"); codeOf(err) != protocol.CodeInvalidInput {
		t.Fatalf("unknown origin gave %v, want invalid_input", err)
	}
	if err := s.SubmitOrigin("reports", protocol.OriginChildren); err != nil {
		t.Fatal(err)
	}
	waitState(t, s, protocol.SessionStateIdle)
	evs, _ := s.Log.Replay(1)
	for _, ev := range evs {
		if ev.Type == protocol.EventTurnStarted {
			var d protocol.TurnStarted
			_ = json.Unmarshal(ev.Payload, &d)
			if d.Origin != protocol.OriginChildren {
				t.Fatalf("turn.started origin %q, want children", d.Origin)
			}
			return
		}
	}
	t.Fatal("no turn started")
}

// The .i's step 3 contract: an approval with no deadline is still pending
// after the time an approval with one would have lapsed.
func TestAnApprovalWithoutADeadlineOutlivesTheTimeout(t *testing.T) {
	const timeout = 20 * time.Millisecond
	s := newSession(t)
	get := func() *Session { return s }

	hurried := NewApprover(get, timeout)
	req := protocol.ApprovalRequest{ApprovalID: "a1", Tool: "bash", Command: "x", ExpiresAt: hurried.Deadline()}
	if d, _ := hurried.Approve(context.Background(), req); d != protocol.ApprovalDeny {
		t.Fatalf("a deadlined question gave %s, want it to lapse into deny", d)
	}

	patient := NewApprover(get, 0)
	if !patient.Deadline().IsZero() {
		t.Fatalf("a question with no deadline was given %v", patient.Deadline())
	}
	req = protocol.ApprovalRequest{ApprovalID: "a2", Tool: "bash", Command: "x", ExpiresAt: patient.Deadline()}
	done := make(chan protocol.ApprovalDecision, 1)
	go func() {
		d, _ := patient.Approve(context.Background(), req)
		done <- d
	}()
	time.Sleep(3 * timeout)
	select {
	case d := <-done:
		t.Fatalf("the question without a deadline was answered %s on its own", d)
	default:
	}
	if len(s.Pending()) != 1 {
		t.Fatalf("%d pending, want the one waiting for a person", len(s.Pending()))
	}
	if err := s.Resolve("a2", protocol.ApprovalAllow); err != nil {
		t.Fatal(err)
	}
	if d := <-done; d != protocol.ApprovalAllow {
		t.Errorf("got %s, want the person's answer", d)
	}
}

func TestAnApproverWithoutASessionDenies(t *testing.T) {
	for _, a := range []Approver{NewApprover(nil, 0), NewApprover(func() *Session { return nil }, time.Second)} {
		if d, _ := a.Approve(context.Background(), protocol.ApprovalRequest{}); d != protocol.ApprovalDeny {
			t.Errorf("got %s, want deny", d)
		}
	}
}
