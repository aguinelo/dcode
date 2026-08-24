package session

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/aguinelo/dcode/internal/protocol"
)

// Carry reads a record as the events a continuing session replays, and how
// many turns they represent.
//
// Rebuild answers the neighbouring question — what the model is sent — and the
// two are deliberately different readings of the same file. The model is sent
// messages, and a message is not watchable: it has no tool crossing, no
// approval, no reasoning, none of what a person needs to see to know the work
// survived. Carrying only what the model needs is what left the screen blank.
func Carry(path string) ([]protocol.Event, int, error) {
	return carry(path, map[string]bool{})
}

// carry reads one record and everything it continues, oldest first.
//
// The chain is followed rather than copied. A record holds the marker naming
// the conversation it continues and NOT that conversation's events, so reading
// one means walking back — which is what keeps a record linear in its own
// session instead of quadratic in the number of times somebody typed `-c`.
//
// `seen` refuses a cycle. A record naming itself, or two naming each other, is
// a corrupt pair rather than an impossible one — the id is a timestamp and a
// random suffix, and nothing enforces the arrow points backwards.
func carry(path string, seen map[string]bool) ([]protocol.Event, int, error) {
	id := strings.TrimSuffix(filepath.Base(path), ".jsonl")
	if seen[id] {
		return nil, 0, nil
	}
	seen[id] = true

	f, err := os.Open(path)
	if err != nil {
		return nil, 0, err
	}
	defer f.Close()

	var out []protocol.Event
	turns := 0

	sc := bufio.NewScanner(f)
	// A payload can be a diff, which exceeds the default line limit. Hitting it
	// would cut the conversation mid-event and replay the tail as its own.
	sc.Buffer(make([]byte, 0, 64<<10), 8<<20)

	for sc.Scan() {
		var ev protocol.Event
		if err := json.Unmarshal(sc.Bytes(), &ev); err != nil {
			// A line that is not an event is skipped rather than fatal, for the
			// same reason a crooked record does not stop a listing: one bad
			// line must not cost the whole conversation.
			continue
		}
		switch ev.Type {
		case protocol.EventSessionCreated:
			// The new session announced itself already. A second creation in
			// the stream would describe a workspace, a model and a sandbox that
			// are not the ones now in force.
			continue
		case protocol.EventApprovalRequired:
			// A crossing already decided. Replaying it would put a modal in
			// front of somebody for a question answered yesterday.
			continue
		case protocol.EventSessionResumed:
			// The marker names the conversation this one continues. Read it
			// first, and drop the marker itself: the session being built will
			// emit its own, naming the record it was actually asked to
			// continue rather than one further up the chain.
			var d protocol.SessionResumed
			if json.Unmarshal(ev.Payload, &d) != nil || d.SourceID == "" {
				continue
			}
			older, n, oerr := carry(filepath.Join(filepath.Dir(path), d.SourceID+".jsonl"), seen)
			if oerr != nil {
				// A missing ancestor is a shorter conversation, not a failed
				// one. Refusing here would make one pruned record unreadable
				// for every session that ever continued it.
				continue
			}
			out = append(out, older...)
			turns += n
			continue
		case protocol.EventTurnCompleted:
			turns++
		}
		out = append(out, ev)
	}
	if err := sc.Err(); err != nil {
		return nil, 0, err
	}
	return out, turns, nil
}
