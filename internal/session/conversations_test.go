package session

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
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
	summarizeRecord = func(path string) (row, error) {
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
	if got != want {
		t.Errorf("the live row reads\n  %+v\nwant\n  %+v", got, want)
	}

	x.StateChanged("x", protocol.SessionStateClosed)
	recorded := find(t, x, "x")
	if recorded.Live || recorded.State != protocol.ConversationRecorded {
		t.Fatalf("the ended row reads %+v, want recorded", recorded)
	}
	recorded.Live, recorded.State = live.Live, live.State
	if recorded != live {
		t.Errorf("the conversation changed when it ended:\n  live     %+v\n  recorded %+v", live, recorded)
	}
}

// A session that continues another is the same conversation, and the list
// calls it what its whole history does — the history Carry reads, and a window
// showing the conversation replays: the last name given in any of its
// sessions, else the first thing asked in the oldest. Asking something in a
// continuation does not retitle the conversation, a name given before it
// carries over, and a name cleared in it clears the one before.
func TestAContinuationIsTitledByItsWholeConversation(t *testing.T) {
	dir := t.TempDir()
	type step func(l *EventLog)
	ask := func(text string) step {
		return func(l *EventLog) {
			_, _ = l.Append(protocol.EventTurnStarted, protocol.TurnStarted{TurnID: "t", Text: text})
		}
	}
	name := func(n string) step {
		return func(l *EventLog) {
			_, _ = l.Append(protocol.EventSessionRenamed, protocol.SessionRenamed{Name: n})
		}
	}
	record := func(id, continues string, steps ...step) {
		liveRecord(t, dir, id, func(l *EventLog) {
			created(l, "/w")
			if continues != "" {
				_, _ = l.Append(protocol.EventSessionResumed, protocol.SessionResumed{SourceID: continues})
			}
			for _, s := range steps {
				s(l)
			}
		})
	}
	record("a", "", ask("onde fica a fila de webhooks?"))
	record("b", "a", ask("e no qwen, onde fica?"))
	record("c", "b")
	record("n1", "", ask("conserte o parser"), name("Parser"))
	record("n2", "n1", ask("e os testes?"))
	record("m1", "", ask("escreva o README"))
	record("m2", "m1", ask("e em inglês?"), name("README"))
	record("k1", "", ask("suba o servidor"), name("Servidor"))
	record("k2", "k1", name(""), ask("e o banco?"))
	record("p2", "p1", ask("o que sobrou?"))
	record("y1", "y2", ask("um lado"))
	record("y2", "y1", ask("o outro"))
	record("l0", "", ask("a primeira de muitas"))
	for i := 1; i <= 10; i++ {
		record(fmt.Sprintf("l%d", i), fmt.Sprintf("l%d", i-1), ask(fmt.Sprintf("a troca de modelo %d", i)))
	}

	type title struct {
		text  string
		named bool
	}
	want := map[string]title{
		"b":   {"onde fica a fila de webhooks?", false}, // asked after continuing
		"c":   {"onde fica a fila de webhooks?", false}, // continued twice, nothing asked yet
		"n2":  {"Parser", true},                         // named before it was continued
		"m1":  {"escreva o README", false},              // a continuation's name is its own
		"m2":  {"README", true},
		"k2":  {"suba o servidor", false}, // the name cleared after continuing
		"p2":  {"o que sobrou?", false},   // what it continues was pruned
		"l10": {"a primeira de muitas", false},
	}
	list, err := NewConversations(dir).List("")
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range list {
		if text, named := carriedTitle(t, dir, c.ID); c.Title != text || c.Named != named {
			t.Errorf("%s is listed as %q (named %v), and its whole history calls it %q (named %v)",
				c.ID, c.Title, c.Named, text, named)
		}
		if w, ok := want[c.ID]; ok && (c.Title != w.text || c.Named != w.named) {
			t.Errorf("%s is listed as %q (named %v), want %q (named %v)", c.ID, c.Title, c.Named, w.text, w.named)
		}
	}
}

// The list's stream says what its snapshot says, and a continuation keeps the
// conversation's title once what it continues has ended: switching model
// closes the session left right after the new one opens (D28), and nothing
// lists the record that ended in between.
func TestAContinuationKeepsItsTitleWhenWhatItContinuesEnds(t *testing.T) {
	dir := t.TempDir()
	x := NewConversations(dir)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	_, changes, err := x.Subscribe(ctx)
	if err != nil {
		t.Fatal(err)
	}
	liveRecord(t, dir, "s", func(l *EventLog) {
		l.OnRecorded(x.Observe)
		created(l, "/w")
		_, _ = l.Append(protocol.EventTurnStarted, protocol.TurnStarted{TurnID: "t1", Text: "onde fica a fila de webhooks?"})
	})
	continued := liveRecord(t, dir, "c", func(l *EventLog) {
		l.OnRecorded(x.Observe)
		created(l, "/w")
		_, _ = l.Append(protocol.EventSessionResumed, protocol.SessionResumed{SourceID: "s"})
	})
	x.StateChanged("s", protocol.SessionStateClosed)
	_, _ = continued.Append(protocol.EventTurnStarted, protocol.TurnStarted{TurnID: "t2", Text: "e no qwen, onde fica?"})

	var streamed *protocol.Conversation
	for len(changes) > 0 {
		if ch := <-changes; ch.Conversation != nil && ch.Conversation.ID == "c" {
			streamed = ch.Conversation
		}
	}
	if streamed == nil {
		t.Fatal("the stream sent nothing about the continuation")
	}
	if streamed.Title != "onde fica a fila de webhooks?" || streamed.Named {
		t.Errorf("the stream calls the continuation %q (named %v), want the conversation's first question",
			streamed.Title, streamed.Named)
	}
	if listed := find(t, x, "c"); listed.Title != streamed.Title {
		t.Errorf("the snapshot calls the continuation %q and the stream %q", listed.Title, streamed.Title)
	}
}

// carriedTitle is what a window showing a conversation calls it: the record
// and everything it continues, as Carry reads them, with the last name given,
// else the first thing asked.
func carriedTitle(t *testing.T, dir, id string) (string, bool) {
	t.Helper()
	events, _, err := Carry(filepath.Join(dir, id+".jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	var asked, name string
	for _, ev := range events {
		switch ev.Type {
		case protocol.EventTurnStarted:
			var d protocol.TurnStarted
			if json.Unmarshal(ev.Payload, &d) == nil && asked == "" {
				asked = firstLineOf(d.Text)
			}
		case protocol.EventSessionRenamed:
			var d protocol.SessionRenamed
			if json.Unmarshal(ev.Payload, &d) == nil {
				name = d.Name
			}
		}
	}
	if name != "" {
		return name, true
	}
	return asked, false
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
