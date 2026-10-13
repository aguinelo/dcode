package session

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/aguinelo/dcode/internal/protocol"
)

// idLimit is the longest id a record may be named by. NewID makes nineteen
// characters; the room above that is for ids from tests and older builds, not
// for paths.
const idLimit = 128

// ValidID refuses an id that could name anything but a record inside the
// record directory.
//
// An id becomes a file name — `<dir>/<id>.jsonl` — so one carrying a
// separator, or a `..`, is a path: read as given, `../x` answers any `.jsonl`
// the daemon's user can open. The rule is a closed alphabet rather than a list
// of what is dangerous, because a list is a list of what someone thought of.
func ValidID(id string) error {
	if id == "" || len(id) > idLimit {
		return protocol.Errorf(protocol.CodeInvalidInput,
			"a conversation id is 1 to %d characters, got %d", idLimit, len(id))
	}
	for _, r := range id {
		ok := r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-' || r == '_'
		if !ok {
			return protocol.Errorf(protocol.CodeInvalidInput,
				"a conversation id is letters, digits, '-' and '_', got %q", id)
		}
	}
	return nil
}

// History reads a recorded conversation as a client shows it, without loading
// it.
//
// It is Carry's reading — the chain the record continues, oldest first, no
// creation and no question already answered — renumbered as one sequence from
// 1 under the conversation's own id. Renumbered because the records of a chain
// each count from 1: read as they are, two events would share a number, and a
// client that keeps the one it already has would drop the other.
//
// A record that is not there is ErrNoSuchSession, never an empty conversation:
// the window that showed an empty thread for a conversation that had ended
// made it look lost.
func History(dir, id string) ([]protocol.Event, error) {
	if err := ValidID(id); err != nil {
		return nil, err
	}
	if dir == "" {
		return nil, ErrNoSuchSession
	}
	evs, _, err := Carry(filepath.Join(dir, id+".jsonl"))
	if errors.Is(err, os.ErrNotExist) {
		return nil, ErrNoSuchSession
	}
	if err != nil {
		return nil, err
	}
	for i := range evs {
		evs[i].Seq = uint64(i + 1)
		evs[i].SessionID = id
	}
	return evs, nil
}
