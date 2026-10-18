package session

import (
	"context"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/aguinelo/dcode/internal/protocol"
)

// liveRecord writes a session's record the way a live session writes it.
func liveRecord(t *testing.T, dir, id string, events func(l *EventLog)) *EventLog {
	t.Helper()
	at := time.Date(2026, 9, 30, 12, 0, 0, 0, time.UTC)
	log := NewEventLog(id, 1000, func() time.Time { at = at.Add(time.Second); return at })
	rec, err := NewRecord(dir, id)
	if err != nil {
		t.Fatal(err)
	}
	log.SetRecord(rec)
	if events != nil {
		events(log)
	}
	return log
}

func created(l *EventLog, ws string) {
	_, _ = l.Append(protocol.EventSessionCreated, protocol.Session{
		ID: "x", State: protocol.SessionStateIdle, Workspace: ws, Model: "MiniMax-M3", Branch: "main",
	})
}

// A list is asked for often and a record can be megabytes: a record is read
// again only when the file changed, and only that one.
func TestARecordIsReadAgainOnlyWhenItChanged(t *testing.T) {
	dir := t.TempDir()
	a := liveRecord(t, dir, "a", func(l *EventLog) { created(l, "/w") })
	liveRecord(t, dir, "b", func(l *EventLog) { created(l, "/w") })

	reads := map[string]int{}
	restore := summarizeRecord
	summarizeRecord = func(path string) (protocol.Conversation, error) {
		reads[filepath.Base(path)]++
		return restore(path)
	}
	defer func() { summarizeRecord = restore }()

	x := NewConversations(dir)
	for i := 0; i < 2; i++ {
		if _, err := x.List(""); err != nil {
			t.Fatal(err)
		}
	}
	if reads["a.jsonl"] != 1 || reads["b.jsonl"] != 1 {
		t.Fatalf("two listings read the records %v times, want once each", reads)
	}

	if _, err := a.Append(protocol.EventTurnStarted, protocol.TurnStarted{TurnID: "t1", Text: "mais um"}); err != nil {
		t.Fatal(err)
	}
	list, err := x.List("")
	if err != nil {
		t.Fatal(err)
	}
	if reads["a.jsonl"] != 2 || reads["b.jsonl"] != 1 {
		t.Errorf("after a changed, the records were read %v times, want a twice and b once", reads)
	}
	for _, c := range list {
		if c.ID == "a" && c.Title != "mais um" {
			t.Errorf("the changed record still reads %+v", c)
		}
	}
}

// The live row and the recorded row of one conversation are the same row: they
// are folded from the same events by the same code, so a conversation that
// ends does not change what the list says about it beyond that it ended.
func TestALiveSummaryMatchesItsRecord(t *testing.T) {
	dir := t.TempDir()
	x := NewConversations(dir)
	liveRecord(t, dir, "x", func(l *EventLog) {
		l.OnRecorded(x.Observe)
		created(l, "/w")
		_, _ = l.Append(protocol.EventTurnStarted, protocol.TurnStarted{TurnID: "t1", Text: "conserte o parser\nsegunda linha"})
		_, _ = l.Append(protocol.EventMessageDelta, protocol.MessageDelta{Text: "feito"})
		_, _ = l.Append(protocol.EventToolCompleted, protocol.ToolCompleted{ToolCallID: "c1", OK: true, Added: 3, Removed: 1})
		_, _ = l.Append(protocol.EventToolCompleted, protocol.ToolCompleted{ToolCallID: "c2", OK: true})
		_, _ = l.Append(protocol.EventTurnCompleted, protocol.TurnCompleted{TurnID: "t1", Reason: "done",
			Completion: &protocol.Completion{Verification: "passed"}})
		_, _ = l.Append(protocol.EventSessionRenamed, protocol.SessionRenamed{Name: "parser"})
	})

	live := find(t, x, "x")
	if !live.Live || live.State != protocol.SessionStateIdle {
		t.Fatalf("the live row reads %+v, want live and idle", live)
	}
	want := protocol.Conversation{
		ID: "x", Title: "parser", Named: true, Workspace: "/w", Branch: "main", Model: "MiniMax-M3",
		Turns: 1, Verification: "passed", Added: 3, Removed: 1, Files: 1,
		LastEvent: protocol.EventSessionRenamed,
	}
	got := live
	got.Live, got.State, got.Started, got.LastActivity = false, "", time.Time{}, time.Time{}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("the live row reads\n  %+v\nwant\n  %+v", got, want)
	}

	x.StateChanged("x", protocol.SessionStateClosed)
	recorded := find(t, x, "x")
	if recorded.Live || recorded.State != protocol.ConversationRecorded {
		t.Fatalf("the ended row reads %+v, want recorded", recorded)
	}
	recorded.Live, recorded.State = live.Live, live.State
	if !reflect.DeepEqual(recorded, live) {
		t.Errorf("the conversation changed when it ended:\n  live     %+v\n  recorded %+v", live, recorded)
	}
}

func find(t *testing.T, x *Conversations, id string) protocol.Conversation {
	t.Helper()
	list, err := x.List("")
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range list {
		if c.ID == id {
			return c
		}
	}
	t.Fatalf("%s is not in the list: %+v", id, list)
	return protocol.Conversation{}
}

// A child's place under its orchestrator is part of its record: the list reads
// it from session.created, so it is the same after the daemon restarts as it
// was live.
func TestARecordedConversationKeepsItsParent(t *testing.T) {
	dir := t.TempDir()
	liveRecord(t, dir, "c", func(l *EventLog) {
		_, _ = l.Append(protocol.EventSessionCreated, protocol.Session{
			ID: "c", State: protocol.SessionStateIdle, Workspace: "/p/children/c", Model: "MiniMax-M3",
			ParentID: "o", ProjectID: "a1b2c3d4e5f6a1b2c3d4e5f6", Role: protocol.RoleChild,
			Owns: []string{"internal/index/"},
		})
	})
	got := find(t, NewConversations(dir), "c")
	if got.ParentID != "o" || got.ProjectID != "a1b2c3d4e5f6a1b2c3d4e5f6" || got.Role != protocol.RoleChild ||
		len(got.Owns) != 1 || got.Owns[0] != "internal/index/" {
		t.Errorf("the recorded child reads %+v, want its parent, project, role and owns", got)
	}
}

// A session reopened under its own id writes no second session.created — its
// record already has one — so the list is told, and the row it had on disk is
// live again: the orchestrator after a daemon restart is the live session of
// its project, never a conversation that ended.
func TestAReopenedSessionIsLiveInTheList(t *testing.T) {
	dir := t.TempDir()
	log := liveRecord(t, dir, "o", func(l *EventLog) {
		_, _ = l.Append(protocol.EventSessionCreated, protocol.Session{
			ID: "o", State: protocol.SessionStateIdle, Workspace: "/p", Model: "MiniMax-M3",
			ProjectID: "a1b2c3d4e5f6a1b2c3d4e5f6", Role: protocol.RoleOrchestrator,
		})
		_, _ = l.Append(protocol.EventTurnStarted, protocol.TurnStarted{TurnID: "t1", Text: "plan the index"})
	})
	x := NewConversations(dir)
	if before := find(t, x, "o"); before.Live {
		t.Fatalf("before the reopen the row reads %+v, want recorded", before)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	_, changes, err := x.Subscribe(ctx)
	if err != nil {
		t.Fatal(err)
	}

	if err := x.Reopened("o"); err != nil {
		t.Fatal(err)
	}
	log.OnRecorded(x.Observe)
	_, _ = log.Append(protocol.EventTurnCompleted, protocol.TurnCompleted{TurnID: "t1", Reason: protocol.StopDaemonRestart})

	got := find(t, x, "o")
	if !got.Live || got.State != protocol.SessionStateIdle || got.Role != protocol.RoleOrchestrator ||
		got.Title != "plan the index" || got.Turns != 1 {
		t.Errorf("the reopened row reads %+v, want live, idle, the orchestrator, its title and the turn that ended", got)
	}
	select {
	case ch := <-changes:
		if ch.Conversation == nil || ch.Conversation.ID != "o" || !ch.Conversation.Live {
			t.Errorf("the stream said %+v, want the row live again", ch)
		}
	default:
		t.Error("the stream said nothing when the session came back")
	}
	if err := x.Reopened("nothing"); err == nil {
		t.Error("reopening a record that is not there was not refused")
	}
}
