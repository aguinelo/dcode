package session

import (
	"bufio"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/aguinelo/dcode/internal/protocol"
)

// Conversations is the list a client shows: every conversation live in this
// daemon, folded from its events as they are recorded, and every one that
// ended and stayed on disk, read from its record. One list and one stream, so
// a sidebar watching twenty conversations holds one connection, not twenty.
//
// Two locks, and the order matters. mu guards the live rows and the
// subscribers, and is taken from inside a session — its log's lock, its own —
// so nothing under it may block or call back. cacheMu guards the rows read
// from disk, and is held while files are read; mu is never taken under it.
type Conversations struct {
	dir string

	mu      sync.Mutex
	live    map[string]*row
	subs    map[int]chan protocol.ConversationChange
	nextSub int

	cacheMu sync.Mutex
	cache   map[string]cachedRecord
}

// row is a conversation as it is folded. The first question is kept apart
// from the name, because clearing a name gives the derived title back.
type row struct {
	c     protocol.Conversation
	asked string
	name  string
}

type cachedRecord struct {
	size int64
	mod  time.Time
	conv protocol.Conversation
}

// subscriberBuffer is how far a list stream may fall behind before it is
// dropped. Dropping is safe: the client reconnects and gets a fresh snapshot.
const subscriberBuffer = 64

// NewConversations lists the conversations recorded in dir, and the live ones
// it is told about. An empty dir records nothing, and then a conversation that
// ends leaves the list.
func NewConversations(dir string) *Conversations {
	return &Conversations{
		dir:   dir,
		live:  map[string]*row{},
		subs:  map[int]chan protocol.ConversationChange{},
		cache: map[string]cachedRecord{},
	}
}

// Observe folds one recorded event of a live session into its row, and sends
// the row when something a list shows changed. Called under the session log's
// lock (EventLog.OnRecorded).
func (x *Conversations) Observe(ev protocol.Event) {
	x.mu.Lock()
	defer x.mu.Unlock()
	r, ok := x.live[ev.SessionID]
	if !ok {
		// A row starts where the session does. Anything before it belongs to
		// a session this list was not told about.
		if ev.Type != protocol.EventSessionCreated {
			return
		}
		r = &row{c: protocol.Conversation{ID: ev.SessionID, Live: true}}
		x.live[ev.SessionID] = r
	}
	before := x.view(r)
	r.fold(ev)
	if after := x.view(r); !sameRow(before, after) {
		x.send(protocol.ConversationChange{Kind: protocol.ConversationChanged, Conversation: &after})
	}
}

// StateChanged moves a live row to the session's state. Called under the
// session's lock (Session.OnState). A session that closes leaves the live set:
// its record is what remains, and the stream says so.
func (x *Conversations) StateChanged(id string, st protocol.SessionState) {
	x.mu.Lock()
	defer x.mu.Unlock()
	r, ok := x.live[id]
	if !ok || r.c.State == st {
		return
	}
	if st != protocol.SessionStateClosed {
		r.c.State = st
		v := x.view(r)
		x.send(protocol.ConversationChange{Kind: protocol.ConversationChanged, Conversation: &v})
		return
	}
	delete(x.live, id)
	if !x.recorded(id) {
		x.send(protocol.ConversationChange{Kind: protocol.ConversationRemoved, ID: id})
		return
	}
	ended := x.view(r)
	ended.Live, ended.State = false, protocol.ConversationRecorded
	x.send(protocol.ConversationChange{Kind: protocol.ConversationChanged, Conversation: &ended})
}

// recorded reports whether a conversation has a record to stay in the list as.
func (x *Conversations) recorded(id string) bool {
	if x.dir == "" {
		return false
	}
	_, err := os.Stat(filepath.Join(x.dir, id+".jsonl"))
	return err == nil
}

// List is every conversation, newest activity first: the live ones as they are
// now, and the recorded ones that are not live. workspace filters, and empty
// lists them all.
func (x *Conversations) List(workspace string) ([]protocol.Conversation, error) {
	recorded, err := x.readRecords()
	if err != nil {
		return nil, err
	}
	x.mu.Lock()
	out := make([]protocol.Conversation, 0, len(x.live)+len(recorded))
	for _, r := range x.live {
		out = append(out, x.view(r))
	}
	for _, c := range recorded {
		if _, live := x.live[c.ID]; !live {
			out = append(out, c)
		}
	}
	x.mu.Unlock()

	inheritTitles(out)
	if workspace != "" {
		kept := out[:0]
		for _, c := range out {
			if c.Workspace == workspace {
				kept = append(kept, c)
			}
		}
		out = kept
	}
	sort.SliceStable(out, func(i, j int) bool {
		if !out[i].LastActivity.Equal(out[j].LastActivity) {
			return out[i].LastActivity.After(out[j].LastActivity)
		}
		return out[i].ID > out[j].ID
	})
	return out, nil
}

// Subscribe opens the list's stream: the whole list now, then each change. The
// subscription is registered before the snapshot is taken, so a change in
// between is sent rather than lost — at worst twice, which a client applying
// changes cannot tell from once. Cancelling ctx closes the channel, and so does
// falling too far behind.
func (x *Conversations) Subscribe(ctx context.Context) ([]protocol.Conversation, <-chan protocol.ConversationChange, error) {
	ch := make(chan protocol.ConversationChange, subscriberBuffer)
	x.mu.Lock()
	id := x.nextSub
	x.nextSub++
	x.subs[id] = ch
	x.mu.Unlock()

	unsubscribe := func() {
		x.mu.Lock()
		defer x.mu.Unlock()
		if c, ok := x.subs[id]; ok {
			delete(x.subs, id)
			close(c)
		}
	}
	snapshot, err := x.List("")
	if err != nil {
		unsubscribe()
		return nil, nil, err
	}
	go func() {
		<-ctx.Done()
		unsubscribe()
	}()
	return snapshot, ch, nil
}

// send fans a change out. Caller holds mu. A subscriber that stopped reading
// is dropped rather than waited for: a session must never hold up on a list.
func (x *Conversations) send(ch protocol.ConversationChange) {
	for id, sub := range x.subs {
		select {
		case sub <- ch:
		default:
			delete(x.subs, id)
			close(sub)
		}
	}
}

// view is the row as a list shows it, with the title a continued conversation
// takes from the one it continues while it has none of its own. Caller holds
// mu; the source is looked up among the live rows and the records already
// read, never on disk.
func (x *Conversations) view(r *row) protocol.Conversation {
	c := r.c
	if c.Title == "" && c.ContinuedFrom != "" {
		if src, ok := x.live[c.ContinuedFrom]; ok {
			c.Title = src.c.Title
		} else {
			x.cacheMu.Lock()
			if cached, ok := x.cache[c.ContinuedFrom+".jsonl"]; ok {
				c.Title = cached.conv.Title
			}
			x.cacheMu.Unlock()
		}
	}
	return c
}

// readRecords summarises every record in dir, reading again only the files
// whose size or time changed since the last read.
func (x *Conversations) readRecords() ([]protocol.Conversation, error) {
	if x.dir == "" {
		return nil, nil
	}
	entries, err := os.ReadDir(x.dir)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	x.cacheMu.Lock()
	defer x.cacheMu.Unlock()
	seen := map[string]bool{}
	var out []protocol.Conversation
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".jsonl") {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		name := e.Name()
		seen[name] = true
		if c, ok := x.cache[name]; ok && c.size == info.Size() && c.mod.Equal(info.ModTime()) {
			out = append(out, c.conv)
			continue
		}
		conv, err := summarizeRecord(filepath.Join(x.dir, name))
		if err != nil {
			// Not a record, or not one yet. One unreadable file must not make
			// the other forty unlistable — the same rule Browse keeps.
			continue
		}
		x.cache[name] = cachedRecord{size: info.Size(), mod: info.ModTime(), conv: conv}
		out = append(out, conv)
	}
	for name := range x.cache {
		if !seen[name] {
			delete(x.cache, name)
		}
	}
	return out, nil
}

// summarizeRecord reads one record into the row a list shows, folding its
// events with the same code that folds a live session's. A variable so a test
// can count the reads.
var summarizeRecord = func(path string) (protocol.Conversation, error) {
	f, err := os.Open(path)
	if err != nil {
		return protocol.Conversation{}, err
	}
	defer f.Close()

	r := &row{c: protocol.Conversation{ID: strings.TrimSuffix(filepath.Base(path), ".jsonl")}}
	opened := false
	sc := bufio.NewScanner(f)
	// A payload can be a diff, larger than the scanner's default line.
	sc.Buffer(make([]byte, 0, 64*1024), 8*1024*1024)
	for sc.Scan() {
		var ev protocol.Event
		if err := json.Unmarshal(sc.Bytes(), &ev); err != nil {
			continue
		}
		if ev.Type == protocol.EventSessionCreated {
			opened = true
		}
		r.fold(ev)
	}
	if err := sc.Err(); err != nil {
		return protocol.Conversation{}, err
	}
	if !opened {
		return protocol.Conversation{}, errNotARecord
	}
	r.c.Live, r.c.State = false, protocol.ConversationRecorded
	return r.c, nil
}

// fold applies one event to a row: the one place a conversation's summary is
// computed, live or from a record.
func (r *row) fold(ev protocol.Event) {
	c := &r.c
	c.LastActivity = ev.At
	switch ev.Type {
	case protocol.EventSessionCreated:
		var s protocol.Session
		if json.Unmarshal(ev.Payload, &s) == nil {
			c.Workspace, c.Branch, c.Model = s.Workspace, s.Branch, s.Model
			if s.State != "" && c.Live {
				c.State = s.State
			}
		}
		c.Started = ev.At
	case protocol.EventSessionResumed:
		var d protocol.SessionResumed
		if json.Unmarshal(ev.Payload, &d) == nil {
			c.ContinuedFrom = d.SourceID
		}
	case protocol.EventTurnStarted:
		var d protocol.TurnStarted
		if json.Unmarshal(ev.Payload, &d) == nil && r.asked == "" {
			r.asked = firstLineOf(d.Text)
		}
	case protocol.EventTurnCompleted:
		c.Turns++
		var d protocol.TurnCompleted
		c.Verification = ""
		if json.Unmarshal(ev.Payload, &d) == nil && d.Completion != nil {
			c.Verification = d.Completion.Verification
		}
	case protocol.EventToolCompleted:
		var d protocol.ToolCompleted
		if json.Unmarshal(ev.Payload, &d) == nil && (d.Added > 0 || d.Removed > 0) {
			c.Added += d.Added
			c.Removed += d.Removed
			c.Files++
		}
	case protocol.EventSessionRenamed:
		var d protocol.SessionRenamed
		if json.Unmarshal(ev.Payload, &d) == nil {
			// The last name wins, and an empty one gives the derived title back.
			r.name = d.Name
		}
	case protocol.EventApprovalRequired, protocol.EventApprovalResolved, protocol.EventSessionError:
	default:
		// Text arriving, reasoning, progress and the rest move the clock and
		// nothing a list shows.
		return
	}
	c.LastEvent = ev.Type
	c.Title, c.Named = r.asked, false
	if r.name != "" {
		c.Title, c.Named = r.name, true
	}
}

// inheritTitles gives a continued conversation with no title of its own the
// title of the one it continues, following the chain a few steps.
func inheritTitles(list []protocol.Conversation) {
	byID := make(map[string]int, len(list))
	for i, c := range list {
		byID[c.ID] = i
	}
	for i := range list {
		c := &list[i]
		for hops, from := 0, c.ContinuedFrom; c.Title == "" && from != "" && hops < 8; hops++ {
			j, ok := byID[from]
			if !ok {
				break
			}
			c.Title, from = list[j].Title, list[j].ContinuedFrom
		}
	}
}

// sameRow compares what a list shows, leaving out when the row last moved: a
// frame that changes only that changes nothing a person reads.
func sameRow(a, b protocol.Conversation) bool {
	a.LastActivity, b.LastActivity = time.Time{}, time.Time{}
	return a == b
}
