// Package session owns the server-side session: its append-only event log and
// its state machine.
//
// The log is the primitive everything else falls out of. Resume, a second
// client attaching mid-turn, and session density are all the same mechanism:
// a monotonic sequence a reader can rejoin at any point. It is the same
// principle as the model context, one layer up.
//
// Spec: docs/specs/architecture/client-server-protocol/202608072240-*.
package session

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/aguinelo/dcode/internal/protocol"
)

// Clock is injected so tests can produce reproducible timestamps. Events carry
// a time for humans reading a log; nothing in the system depends on it, and
// golden comparisons zero it.
type Clock func() time.Time

// EventLog is an append-only sequence per session.
type EventLog struct {
	mu        sync.RWMutex
	sessionID string
	clock     Clock
	retention int
	// spill keeps trimmed events readable. Nil means retention is a hard
	// horizon and a client away too long gets events_expired.
	spill *Spill

	seq    uint64
	events []protocol.Event
	// firstSeq is the lowest sequence still held. Replay below it is refused
	// rather than answered with a gap the client cannot detect.
	firstSeq uint64

	subs map[int]*subscriber
	next int
}

type subscriber struct {
	ch   chan protocol.Event
	from uint64
}

// NewEventLog builds a log. A retention of zero keeps everything.
func NewEventLog(sessionID string, retention int, clock Clock) *EventLog {
	if clock == nil {
		clock = time.Now
	}
	return &EventLog{
		sessionID: sessionID,
		clock:     clock,
		retention: retention,
		firstSeq:  1,
		subs:      map[int]*subscriber{},
	}
}

// Append records a fact and returns the stored event.
//
// Sequence assignment and storage happen under one lock. Splitting them is the
// classic source of gaps under concurrency: two appends would both read the
// counter before either wrote.
func (l *EventLog) Append(t protocol.EventType, payload any) (protocol.Event, error) {
	raw, err := json.Marshal(payload)
	if err != nil {
		return protocol.Event{}, err
	}

	l.mu.Lock()
	l.seq++
	ev := protocol.Event{
		Seq:       l.seq,
		SessionID: l.sessionID,
		Type:      t,
		At:        l.clock(),
		Payload:   raw,
	}
	l.events = append(l.events, ev)
	l.trim()

	// Fan out under the same lock so a subscriber cannot observe events out of
	// order relative to the log itself.
	for id, s := range l.subs {
		if ev.Seq < s.from {
			continue
		}
		select {
		case s.ch <- ev:
		default:
			// A subscriber that stopped reading must never hold up the agent.
			// Dropping it is safe because the client controls its own read
			// position and rejoins with `from`.
			close(s.ch)
			delete(l.subs, id)
		}
	}
	l.mu.Unlock()
	return ev, nil
}

// trim drops events beyond retention. Caller holds the lock.
func (l *EventLog) trim() {
	if l.retention <= 0 || len(l.events) <= l.retention {
		return
	}
	drop := len(l.events) - l.retention
	// To disk before out of memory. Dropping first and writing after would
	// lose them on any failure in between, and a gap a client cannot detect is
	// worse than a refused replay.
	if err := l.spill.Append(l.events[:drop]); err != nil {
		// Nothing is dropped if it could not be kept. Memory grows, which is
		// visible and recoverable; a silent hole in the session is neither.
		return
	}
	l.firstSeq = l.events[drop].Seq
	l.events = append([]protocol.Event(nil), l.events[drop:]...)
}

// SetSpill attaches a spill file. Called at session creation, before anything
// is appended.
func (l *EventLog) SetSpill(s *Spill) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.spill = s
}

// LastSeq returns the highest sequence assigned.
func (l *EventLog) LastSeq() uint64 {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.seq
}

// Replay returns events from `from` onward.
//
// A `from` below what is still held is an error rather than a truncated
// answer: silently starting later would leave the client believing it had the
// whole history.
func (l *EventLog) Replay(from uint64) ([]protocol.Event, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	var out []protocol.Event
	if from < l.firstSeq {
		// Trimmed out of memory is not gone if there is a spill file. The
		// event log is the session, and a client that took too long to come
		// back should get the session, not a refusal about bookkeeping.
		spilled, err := l.spill.Replay(from)
		if err != nil {
			return nil, protocol.Errorf(protocol.CodeInternal,
				"the spilled events could not be read: %v", err)
		}
		if len(spilled) == 0 && l.spill == nil {
			return nil, protocol.Errorf(protocol.CodeEventsExpired,
				"events before %d are no longer held; the earliest available is %d",
				from, l.firstSeq)
		}
		out = append(out, spilled...)
	}
	for _, ev := range l.events {
		if ev.Seq >= from {
			out = append(out, ev)
		}
	}
	return out, nil
}

// Subscribe replays from `from` and then streams live events.
//
// The join is the delicate part: opening the live feed before replay finishes
// duplicates events, opening it after leaves a gap. Registering the subscriber
// and snapshotting the backlog under one lock makes both impossible, and the
// buffered prefix is deduplicated by sequence.
func (l *EventLog) Subscribe(ctx context.Context, from uint64) (<-chan protocol.Event, func(), error) {
	l.mu.Lock()
	if from < l.firstSeq {
		l.mu.Unlock()
		return nil, nil, protocol.Errorf(protocol.CodeEventsExpired,
			"events before %d are no longer held; the earliest available is %d",
			from, l.firstSeq)
	}

	backlog := make([]protocol.Event, 0, len(l.events))
	for _, ev := range l.events {
		if ev.Seq >= from {
			backlog = append(backlog, ev)
		}
	}

	live := make(chan protocol.Event, 256)
	id := l.next
	l.next++
	l.subs[id] = &subscriber{ch: live, from: from}
	l.mu.Unlock()

	cancel := func() {
		l.mu.Lock()
		if s, ok := l.subs[id]; ok {
			delete(l.subs, id)
			close(s.ch)
		}
		l.mu.Unlock()
	}

	out := make(chan protocol.Event, 256)
	go func() {
		defer close(out)
		var last uint64
		for _, ev := range backlog {
			select {
			case out <- ev:
				last = ev.Seq
			case <-ctx.Done():
				cancel()
				return
			}
		}
		for {
			select {
			case ev, open := <-live:
				if !open {
					return
				}
				if ev.Seq <= last {
					continue // already delivered from the backlog
				}
				select {
				case out <- ev:
					last = ev.Seq
				case <-ctx.Done():
					cancel()
					return
				}
			case <-ctx.Done():
				cancel()
				return
			}
		}
	}()
	return out, cancel, nil
}

// Subscribers reports how many readers are attached. Used by tests and by the
// server's shutdown path.
func (l *EventLog) Subscribers() int {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return len(l.subs)
}

// Close drops every subscriber.
func (l *EventLog) Close() {
	l.mu.Lock()
	defer l.mu.Unlock()
	for id, s := range l.subs {
		close(s.ch)
		delete(l.subs, id)
	}
	// The spill exists so a client that was away can still read what it missed.
	// Once the session is gone nobody can ask — the id resolves to nothing and
	// there is no route to it — so the file is garbage with a session id in its
	// name, and one per session adds up for as long as the daemon is useful.
	//
	// A failure here is not worth failing a close over: the session is already
	// finished, and refusing to shut down because a file could not be removed
	// trades a leaked file for a stuck daemon.
	if l.spill != nil {
		_ = l.spill.Remove()
		l.spill = nil
	}
}
