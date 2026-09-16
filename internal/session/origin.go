package session

import (
	"bufio"
	"encoding/json"
	"os"

	"github.com/aguinelo/dcode/internal/protocol"
)

// Origin reads the model bundle a record was created with: the first
// session.created event, undecoded past the fields Describe() already puts
// there.
//
// Used by a resume, not a listing — Browse already reads this same event for
// Summary.Model, but a Summary is a person's answer ("which model") and this
// is the daemon's ("which endpoint"), so it goes back to the wire type rather
// than a second copy of the same fields with different names.
func Origin(path string) (protocol.Session, error) {
	f, err := os.Open(path)
	if err != nil {
		return protocol.Session{}, err
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64<<10), 8<<20)
	for sc.Scan() {
		var ev protocol.Event
		if err := json.Unmarshal(sc.Bytes(), &ev); err != nil {
			continue
		}
		if ev.Type != protocol.EventSessionCreated {
			continue
		}
		var d protocol.Session
		if err := json.Unmarshal(ev.Payload, &d); err != nil {
			return protocol.Session{}, err
		}
		return d, nil
	}
	return protocol.Session{}, errNotARecord
}
