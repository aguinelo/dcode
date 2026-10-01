package app

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/aguinelo/dcode/internal/protocol"
	"github.com/aguinelo/dcode/pkg/client"
)

// conversationDaemon is a daemon that records, with a scripted model behind it:
// enough for conversations to be live, to end, and to stay on disk.
type conversationDaemon struct {
	d     *Daemon
	c     *client.Client
	ws    string
	model *scriptedModel
}

func newConversationDaemon(t *testing.T) conversationDaemon {
	t.Helper()
	model := &scriptedModel{}
	srv := httptest.NewServer(http.HandlerFunc(model.serve))
	t.Cleanup(srv.Close)

	ws := t.TempDir()
	env := envFrom(map[string]string{
		"DCODE_HOME":     t.TempDir(),
		"DCODE_BASE_URL": srv.URL,
		"DCODE_API_KEY":  "sk-test-key-abcdefghijklmnop",
	})
	base, _, err := FromEnv(env, ws)
	if err != nil {
		t.Fatal(err)
	}
	requireSandbox(t, base)
	dir, err := os.MkdirTemp("", "dc")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })
	d := NewDaemon(DaemonOptions{
		SocketPath: filepath.Join(dir, "d.sock"),
		RecordDir:  t.TempDir(),
		Base:       base,
	})
	if err := d.Listen(); err != nil {
		t.Skipf("cannot bind a unix socket here: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	go d.Serve(ctx)
	c := client.New(d.Addr())
	for deadline := time.Now().Add(3 * time.Second); c.Health(ctx) != nil; {
		if time.Now().After(deadline) {
			t.Fatal("the daemon never became healthy")
		}
		time.Sleep(5 * time.Millisecond)
	}
	return conversationDaemon{d: d, c: c, ws: ws, model: model}
}

// One list for the sidebar: the conversations live in this daemon and the ones
// that ended and stayed on disk, each once — a live one with the state it has
// now, an ended one as recorded.
func TestTheListJoinsLiveAndRecordedConversationsOnce(t *testing.T) {
	cd := newConversationDaemon(t)
	ctx := context.Background()

	gone, err := cd.c.CreateSession(ctx, protocol.CreateSessionRequest{Workspace: cd.ws})
	if err != nil {
		t.Fatal(err)
	}
	if err := cd.c.DeleteSession(ctx, gone.ID); err != nil {
		t.Fatal(err)
	}
	live, err := cd.c.CreateSession(ctx, protocol.CreateSessionRequest{Workspace: cd.ws})
	if err != nil {
		t.Fatal(err)
	}

	list, err := cd.c.ListConversations(ctx, "")
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]protocol.Conversation{}
	for _, c := range list {
		if _, twice := seen[c.ID]; twice {
			t.Errorf("%s is listed twice", c.ID)
		}
		seen[c.ID] = c
	}
	if c, ok := seen[live.ID]; !ok || !c.Live || c.State != protocol.SessionStateIdle {
		t.Errorf("the live conversation reads %+v (present %v), want live and idle", c, ok)
	}
	if c, ok := seen[gone.ID]; !ok || c.Live || c.State != protocol.ConversationRecorded {
		t.Errorf("the ended conversation reads %+v (present %v), want recorded", c, ok)
	}
	if c := seen[live.ID]; c.Workspace != cd.ws || c.Started.IsZero() {
		t.Errorf("the live conversation does not say where and when it started: %+v", c)
	}

	only, err := cd.c.ListConversations(ctx, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if len(only) != 0 {
		t.Errorf("a filter on another workspace listed %d conversations", len(only))
	}
}

// One stream for every conversation, so a sidebar watching twenty does not hold
// twenty connections. It opens with the whole list, then sends what changed —
// and a turn's text arriving in fragments is not a change of the list.
func TestTheListStreamOpensWithASnapshotThenSendsChanges(t *testing.T) {
	cd := newConversationDaemon(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	changes, _ := cd.c.WatchConversations(ctx)
	first := next(t, changes)
	if first.Kind != protocol.ConversationSnapshot {
		t.Fatalf("the stream opened with %q, want the snapshot", first.Kind)
	}

	cd.model.turns = [][]string{{frameText("Oi,"), frameText(" tudo"), frameText(" bem"), frameText(" por"), frameText(" aqui.")}}
	sess, err := cd.c.CreateSession(ctx, protocol.CreateSessionRequest{Workspace: cd.ws})
	if err != nil {
		t.Fatal(err)
	}
	if err := cd.c.Submit(ctx, sess.ID, "diga oi"); err != nil {
		t.Fatal(err)
	}

	var frames []protocol.Conversation
	sawRunning := false
	for {
		ch := next(t, changes)
		if ch.Kind != protocol.ConversationChanged || ch.Conversation == nil || ch.Conversation.ID != sess.ID {
			continue
		}
		c := *ch.Conversation
		if len(frames) > 0 && sameConversation(frames[len(frames)-1], c) {
			t.Errorf("a frame repeated the conversation it had just sent: %+v", c)
		}
		frames = append(frames, c)
		if c.State == protocol.SessionStateRunning && c.Title == "diga oi" {
			sawRunning = true
		}
		if c.State == protocol.SessionStateIdle && c.Turns == 1 {
			break
		}
	}
	if !sawRunning {
		t.Errorf("the turn never showed as running with its question as the title: %+v", frames)
	}
	if len(frames) >= 8 {
		t.Errorf("%d frames for one short turn: the fragments of its text reached the list", len(frames))
	}
}

// A conversation that ends stays in the list, recorded, and the stream says so:
// a sidebar must not lose the row it is pointing at.
func TestAConversationThatEndsStaysAsRecorded(t *testing.T) {
	cd := newConversationDaemon(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sess, err := cd.c.CreateSession(ctx, protocol.CreateSessionRequest{Workspace: cd.ws})
	if err != nil {
		t.Fatal(err)
	}
	changes, _ := cd.c.WatchConversations(ctx)
	next(t, changes)
	if err := cd.c.DeleteSession(ctx, sess.ID); err != nil {
		t.Fatal(err)
	}
	for {
		ch := next(t, changes)
		if ch.Kind == protocol.ConversationChanged && ch.Conversation != nil && ch.Conversation.ID == sess.ID {
			if ch.Conversation.Live || ch.Conversation.State != protocol.ConversationRecorded {
				t.Fatalf("the ended conversation was sent as %+v, want recorded", *ch.Conversation)
			}
			return
		}
	}
}

func next(t *testing.T, changes <-chan protocol.ConversationChange) protocol.ConversationChange {
	t.Helper()
	select {
	case ch, ok := <-changes:
		if !ok {
			t.Fatal("the list stream closed")
		}
		return ch
	case <-time.After(10 * time.Second):
		t.Fatal("the list stream sent nothing within 10 s")
	}
	return protocol.ConversationChange{}
}

// sameConversation compares what a list row shows, leaving out when it last
// moved: a frame that changes only that is a frame that changes nothing a
// person reads.
func sameConversation(a, b protocol.Conversation) bool {
	a.LastActivity, b.LastActivity = time.Time{}, time.Time{}
	return a == b
}
