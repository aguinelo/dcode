package session

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/aguinelo/dcode/internal/protocol"
)

// Naming a conversation, stored in the conversation's own record.
//
// The alternatives were a sidecar file per session and one index per workspace,
// and the record wins on the thing that matters: pruning deletes transcripts,
// and a name for a conversation nobody can open any more is worse than no name.
// A sidecar would be orphaned; an index would keep titles for sessions that no
// longer exist. Here the name dies with what it named.
//
// It also keeps the count at one. A second store beside the log is a second
// thing that can disagree with it, and Browse already reads every line of every
// record — so this costs nothing to read back.

// NameLimit is how long a name may be.
//
// Long enough for a sentence somebody would recognise, short enough that a
// sidebar row is still a row. What does not fit is refused rather than trimmed:
// silently keeping half of what was typed is how somebody ends up with a name
// they did not choose.
const NameLimit = 120

// ErrNoSuchSession is returned when there is no record to name.
var ErrNoSuchSession = errors.New("no such session")

// renameMu serialises appends to a record from outside the session that owns
// it. Records are append-only and the daemon is one process, so a mutex is the
// whole of the concurrency story — two renames of one conversation land in the
// order they arrived, and the later one wins by being later.
var renameMu sync.Mutex

// CleanName trims a name and reports whether it is usable.
//
// Control characters go, because a record is read back line by line and a
// newline inside a name would make one line look like two.
func CleanName(name string) (string, error) {
	clean := strings.Map(func(r rune) rune {
		if r < 0x20 || r == 0x7f {
			return -1
		}
		return r
	}, name)
	clean = strings.TrimSpace(clean)
	if len([]rune(clean)) > NameLimit {
		return "", fmt.Errorf("a name is at most %d characters", NameLimit)
	}
	return clean, nil
}

// Rename appends the name to a session's record.
//
// An empty name is not an error: it restores the title derived from the first
// question, which is the way back rather than a second command for undoing.
func Rename(dir, id, name string, now func() time.Time) error {
	if dir == "" || id == "" {
		return ErrNoSuchSession
	}
	clean, err := CleanName(name)
	if err != nil {
		return err
	}
	if now == nil {
		now = time.Now
	}

	renameMu.Lock()
	defer renameMu.Unlock()

	path := filepath.Join(dir, id+".jsonl")
	last, err := lastSeq(path)
	if err != nil {
		return err
	}

	payload, err := json.Marshal(protocol.SessionRenamed{Name: clean})
	if err != nil {
		return err
	}
	// The real envelope, not a local copy of it. A second definition of the
	// event shape is a second thing to keep in step with the first.
	ev := protocol.Event{
		Seq: last + 1, SessionID: id,
		Type: protocol.EventSessionRenamed, At: now(), Payload: payload,
	}
	line, err := json.Marshal(ev)
	if err != nil {
		return err
	}

	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err := f.Write(append(line, '\n')); err != nil {
		return err
	}
	return f.Sync()
}

// lastSeq reads the highest sequence the record carries.
//
// Read rather than assumed, because the sequence is the property the record is
// built on: appending a number that is already there would put a duplicate in a
// log whose whole contract is that there are none.
func lastSeq(path string) (uint64, error) {
	f, err := os.Open(path)
	if os.IsNotExist(err) {
		return 0, ErrNoSuchSession
	}
	if err != nil {
		return 0, err
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 8*1024*1024)
	var last uint64
	for sc.Scan() {
		var ev struct {
			Seq uint64 `json:"seq"`
		}
		if json.Unmarshal(sc.Bytes(), &ev) != nil {
			continue
		}
		if ev.Seq > last {
			last = ev.Seq
		}
	}
	if err := sc.Err(); err != nil {
		return 0, err
	}
	if last == 0 {
		return 0, ErrNoSuchSession
	}
	return last, nil
}
