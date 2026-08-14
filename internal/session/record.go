package session

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"

	"github.com/aguinelo/dcode/internal/protocol"
)

// Record is everything a session did, on disk.
//
// It was a spill: only what retention pushed out of memory, deleted when the
// session closed. Both halves made sense for the one job it had — letting a
// client that was away rejoin — and both were wrong for the job it did not
// have. A session that fits in memory, which is nearly all of them, left
// nothing at all, and what little was written went away exactly when someone
// might want to read it.
//
// So there was no way to audit what dcode actually did, no way to reconstruct
// a session afterwards, and no evidence to reason from when its behaviour
// needed improving. The mechanism was there, with a config key and a state
// directory; nothing in the default path turned it on and closing threw it
// away.
//
// Append-only, one JSON object per line. No index and no rewriting: reading is
// sequential from a sequence number, which a scan answers directly, and jq
// answers everything else.
type Record struct {
	path string

	mu sync.Mutex
	f  *os.File
	w  *bufio.Writer
}

// NewRecord opens the file for a session. An empty directory disables
// recording, which is what turns it off for someone who does not want it.
//
// 0700 on the directory and 0600 on the file: a transcript holds whatever the
// person typed and whatever the agent read, which is the most private thing
// this program touches.
func NewRecord(dir, sessionID string) (*Record, error) {
	if dir == "" {
		return nil, nil
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}
	path := filepath.Join(dir, sessionID+".jsonl")
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return nil, err
	}
	return &Record{path: path, f: f, w: bufio.NewWriter(f)}, nil
}

// Append writes events to the record.
//
// The handle stays open. It used to open and close per call, which was fine
// when the only caller was retention overflow and is not when the caller is
// every event: a streamed answer is hundreds of deltas, and a syscall each
// would be paid on every token.
//
// Anything that is not a text delta is flushed as it happens. A crash mid-turn
// must not cost the tool calls and approvals that led up to it, and those are
// the low-frequency events — flushing them costs almost nothing, while
// flushing deltas would give back the syscall the buffer just saved.
func (r *Record) Append(events []protocol.Event) error {
	if r == nil || len(events) == 0 {
		return nil
	}
	r.mu.Lock()
	defer r.mu.Unlock()

	enc := json.NewEncoder(r.w)
	durable := false
	for _, ev := range events {
		if err := enc.Encode(ev); err != nil {
			return err
		}
		if ev.Type != protocol.EventMessageDelta {
			durable = true
		}
	}
	if durable {
		return r.w.Flush()
	}
	return nil
}

// Close flushes what is buffered and releases the file. The file stays.
func (r *Record) Close() error {
	if r == nil {
		return nil
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.f == nil {
		return nil
	}
	err := r.w.Flush()
	if cerr := r.f.Close(); err == nil {
		err = cerr
	}
	r.f = nil
	return err
}

// Replay returns the spilled events at or after from, in order.
//
// A missing file is not an error: nothing has been trimmed yet, which is the
// normal case for most of a session's life.
func (r *Record) Replay(from uint64) ([]protocol.Event, error) {
	if r == nil {
		return nil, nil
	}
	// Flush first: what is still buffered is part of the answer.
	r.mu.Lock()
	if r.w != nil {
		_ = r.w.Flush()
	}
	r.mu.Unlock()

	f, err := os.Open(r.path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var out []protocol.Event
	sc := bufio.NewScanner(f)
	// An event carries a payload, and a large one — a diff — can exceed the
	// default line limit. Hitting it would silently truncate the replay.
	sc.Buffer(make([]byte, 0, 64*1024), 8*1024*1024)
	for sc.Scan() {
		var ev protocol.Event
		if err := json.Unmarshal(sc.Bytes(), &ev); err != nil {
			return nil, err
		}
		if ev.Seq >= from {
			out = append(out, ev)
		}
	}
	return out, sc.Err()
}
