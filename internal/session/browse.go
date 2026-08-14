package session

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/aguinelo/dcode/internal/protocol"
)

// errNotARecord marks a file that ends in .jsonl and is not a session.
var errNotARecord = errors.New("session: not a record")

// Summary is one session, as a person picking from a list needs to see it.
//
// The title is the load-bearing field. A list of twelve-character ids is a list
// nobody chooses from, and everything else here — a time, a size, a count — is
// true of every session and distinguishes none of them.
type Summary struct {
	ID        string
	Workspace string
	Model     string
	// Title is the first thing that was asked, trimmed to one line.
	Title string
	// Turns is how many completed, which is the cheapest honest measure of
	// how much happened.
	Turns   int
	Started time.Time
	Bytes   int64
}

// titleLimit is how much of the question survives into a listing.
//
// Long enough to tell two sessions apart and short enough that a column of them
// still reads as a column.
const titleLimit = 72

// Browse reads the record directory and describes each session.
//
// workspace filters, and empty means all of them. Filtering is the default at
// the call site rather than here because "what was I doing in this project" is
// the question being asked almost every time, and the other one is rare enough
// to deserve a flag.
//
// A file that is not a record, or a line that is not an event, is skipped
// rather than failing the listing. One corrupt file must not make the other
// forty unreadable — that is the same reasoning that keeps a failed record from
// stopping a session.
func Browse(dir, workspace string) ([]Summary, error) {
	if dir == "" {
		return nil, nil
	}
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var out []Summary
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".jsonl") {
			continue
		}
		s, err := describe(filepath.Join(dir, e.Name()))
		if err != nil {
			continue
		}
		if workspace != "" && s.Workspace != workspace {
			continue
		}
		if info, err := e.Info(); err == nil {
			s.Bytes = info.Size()
		}
		out = append(out, s)
	}

	// Newest first: the one you want is almost always the last one you had.
	sort.Slice(out, func(i, j int) bool { return out[i].Started.After(out[j].Started) })
	return out, nil
}

// describe reads one record far enough to summarise it.
func describe(path string) (Summary, error) {
	f, err := os.Open(path)
	if err != nil {
		return Summary{}, err
	}
	defer f.Close()

	s := Summary{ID: strings.TrimSuffix(filepath.Base(path), ".jsonl")}
	// A record opens with session.created. Without one this is a file that
	// happens to end in .jsonl, and describing it would put something in the
	// listing that is not a session.
	opened := false
	sc := bufio.NewScanner(f)
	// A payload can be a diff, which exceeds the scanner's default line limit.
	// Hitting it silently truncates the read, and the summary would then be of
	// a session that stopped where the buffer did.
	sc.Buffer(make([]byte, 0, 64*1024), 8*1024*1024)

	for sc.Scan() {
		var ev protocol.Event
		if err := json.Unmarshal(sc.Bytes(), &ev); err != nil {
			continue
		}
		switch ev.Type {
		case protocol.EventSessionCreated:
			var d protocol.Session
			if json.Unmarshal(ev.Payload, &d) == nil {
				s.Workspace, s.Model, s.Started = d.Workspace, d.Model, ev.At
				opened = true
			}
		case protocol.EventTurnStarted:
			var d protocol.TurnStarted
			if json.Unmarshal(ev.Payload, &d) == nil && s.Title == "" {
				s.Title = firstLineOf(d.Text)
			}
		case protocol.EventTurnCompleted:
			s.Turns++
		}
	}
	if !opened {
		return Summary{}, errNotARecord
	}
	if s.Started.IsZero() {
		if info, err := f.Stat(); err == nil {
			s.Started = info.ModTime()
		}
	}
	if s.Title == "" {
		// A session that has been opened and not yet asked anything is still a
		// session, and hiding it would hide the one you are sitting in.
		s.Title = "(nothing asked yet)"
	}
	return s, nil
}

// firstLineOf trims a question to something a column can hold.
func firstLineOf(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		s = strings.TrimSpace(s[:i])
	}
	runes := []rune(s)
	if len(runes) > titleLimit {
		return strings.TrimSpace(string(runes[:titleLimit])) + "…"
	}
	return s
}

// Transcript renders a record as the conversation it was.
//
// Reading, and nothing else: no model, no reconstruction, no cost. That is why
// it comes before continuing a session — it answers "what did it do" on its
// own, and it answers it for a session that ended a month ago.
//
// Deltas are joined. They are how text arrives, not how it was said, and one
// answer printed as sixteen fragments on sixteen lines is a transcript nobody
// reads twice.
func Transcript(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	var b strings.Builder
	var answer strings.Builder
	flush := func() {
		if answer.Len() == 0 {
			return
		}
		// A blank line before the answer. Without it the model's reply runs
		// straight on from the last tool line and the two read as one thing.
		if b.Len() > 0 && !strings.HasSuffix(b.String(), "\n\n") {
			b.WriteString("\n")
		}
		b.WriteString(strings.TrimSpace(answer.String()))
		b.WriteString("\n\n")
		answer.Reset()
	}

	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 8*1024*1024)
	for sc.Scan() {
		var ev protocol.Event
		if err := json.Unmarshal(sc.Bytes(), &ev); err != nil {
			continue
		}
		switch ev.Type {
		case protocol.EventSessionCreated:
			var d protocol.Session
			if json.Unmarshal(ev.Payload, &d) == nil {
				fmt.Fprintf(&b, "session %s · %s · %s\n%s\n\n",
					d.ID, d.Model, d.Workspace, ev.At.Format(time.RFC3339))
			}
		case protocol.EventTurnStarted:
			var d protocol.TurnStarted
			if json.Unmarshal(ev.Payload, &d) == nil && d.Text != "" {
				flush()
				fmt.Fprintf(&b, "> %s\n\n", d.Text)
			}
		case protocol.EventToolRequested:
			var d protocol.ToolRequested
			if json.Unmarshal(ev.Payload, &d) == nil {
				flush()
				fmt.Fprintf(&b, "  ⏺ %s %s\n", d.Name, firstLineOf(string(d.Input)))
			}
		case protocol.EventToolCompleted:
			var d protocol.ToolCompleted
			if json.Unmarshal(ev.Payload, &d) == nil && !d.OK {
				fmt.Fprintf(&b, "    failed: %s\n", firstLineOf(d.Output))
			}
		case protocol.EventApprovalResolved:
			var d protocol.ApprovalResolved
			if json.Unmarshal(ev.Payload, &d) == nil {
				fmt.Fprintf(&b, "  approval: %s\n", d.Decision)
			}
		case protocol.EventMessageDelta:
			var d protocol.MessageDelta
			if json.Unmarshal(ev.Payload, &d) == nil {
				answer.WriteString(d.Text)
			}
		case protocol.EventSessionCompacted:
			flush()
			b.WriteString("  … earlier history was summarised here\n\n")
		}
	}
	// A record cut off mid-turn still reads. Refusing to show it would refuse
	// exactly when someone most wants to look.
	flush()
	return b.String(), sc.Err()
}
