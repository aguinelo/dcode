package session

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/aguinelo/dcode/internal/protocol"
)

// Spill is where trimmed events go so they are not lost.
//
// The event log IS the session: a client holds one number and everything else
// lives here, and that is what makes reattaching indistinguishable from having
// watched it live. Retention was memory-only, so a client that was away long
// enough got `events_expired` and the session it was watching became
// unreadable — for a reason that has nothing to do with the session and
// everything to do with how long it took to come back.
//
// Append-only, one JSON object per line. No index and no rewriting: replay is
// sequential from a sequence number, which a scan answers directly, and the
// only operation that ever needs to be fast is the one nobody does often.
type Spill struct {
	path string
}

// NewSpill prepares the file for a session. An empty directory disables it,
// which is the default and the right one for a short-lived process.
func NewSpill(dir, sessionID string) (*Spill, error) {
	if dir == "" {
		return nil, nil
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}
	return &Spill{path: filepath.Join(dir, sessionID+".events.jsonl")}, nil
}

// Append writes events that are leaving memory.
//
// A failure here is returned rather than swallowed. Losing them quietly would
// give a client a gap it cannot detect, which is the one outcome worse than
// refusing the replay outright.
func (s *Spill) Append(events []protocol.Event) error {
	if s == nil || len(events) == 0 {
		return nil
	}
	f, err := os.OpenFile(s.path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()

	w := bufio.NewWriter(f)
	enc := json.NewEncoder(w)
	for _, ev := range events {
		if err := enc.Encode(ev); err != nil {
			return err
		}
	}
	return w.Flush()
}

// Replay returns the spilled events at or after from, in order.
//
// A missing file is not an error: nothing has been trimmed yet, which is the
// normal case for most of a session's life.
func (s *Spill) Replay(from uint64) ([]protocol.Event, error) {
	if s == nil {
		return nil, nil
	}
	f, err := os.Open(s.path)
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

// Remove deletes the file, for when a session is done with.
func (s *Spill) Remove() error {
	if s == nil {
		return nil
	}
	err := os.Remove(s.path)
	if os.IsNotExist(err) {
		return nil
	}
	return err
}
