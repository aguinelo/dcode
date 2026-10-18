package server

import (
	"context"
	"testing"
	"time"

	"github.com/aguinelo/dcode/internal/protocol"
	"github.com/aguinelo/dcode/internal/session"
)

// Over the wire, what the desktop does after a restart: follow from 1 a
// session whose memory starts well after 1. With a record behind it, the
// stream is the whole session in order — never 410 events_expired, which the
// desktop can only say as "the events stopped arriving".
func TestARecordedSessionIsStreamedFromOneOverTheWire(t *testing.T) {
	c, mgr := newDaemon(t)
	rec, err := session.NewRecord(t.TempDir(), "s-rec")
	if err != nil {
		t.Fatal(err)
	}
	log := session.NewEventLog("s-rec", 3, fixedClock())
	log.SetRecord(rec)
	sess := session.New("s-rec", "/w", "m", "workspace-write", nil, log, fixedClock())
	if err := mgr.Add(sess); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 20; i++ {
		sess.Emit(protocol.EventMessageDelta, protocol.MessageDelta{Text: "x"})
	}
	if log.FirstSeq() == 1 {
		t.Fatal("retention did not trim: the test proves nothing")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	events, errs := c.Subscribe(ctx, "s-rec", 1)
	go func() {
		time.Sleep(100 * time.Millisecond)
		sess.Emit(protocol.EventMessageDelta, protocol.MessageDelta{Text: "live"})
	}()
	for want := uint64(1); want <= 21; want++ {
		select {
		case ev, open := <-events:
			if !open {
				t.Fatalf("stream closed before seq %d", want)
			}
			if ev.Seq != want {
				t.Fatalf("got seq %d, want %d: a gap or a duplicate", ev.Seq, want)
			}
		case err := <-errs:
			t.Fatalf("stream refused at seq %d: %v", want, err)
		case <-ctx.Done():
			t.Fatalf("stalled before seq %d", want)
		}
	}
}
