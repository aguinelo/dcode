// Package tui is the terminal client.
//
// It is one client among several, not the product: no session state lives here
// beyond scroll position, panel visibility and the input queue. Closing and
// reopening it replays the session from the log and lands on the same screen.
//
// The design problem is not "how do I display a conversation". The agent
// produces ten to a hundred times more output than anyone will read, and most
// of it is not for reading — it is evidence that work is happening. Show
// everything and the user stops reading, missing the thing that mattered; show
// too little and they cannot trust what they cannot see.
//
// The view model is a pure reducer over the event log, deliberately separated
// from rendering so both are testable without a terminal.
//
// Spec: docs/specs/architecture/client-tui/202608081250-*.
package tui

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/aguinelo/dcode/internal/protocol"
)

// Kind classifies a stream entry.
type Kind string

const (
	KindUser      Kind = "user"
	KindAssistant Kind = "assistant"
	KindTool      Kind = "tool"
	KindError     Kind = "error"
	KindNote      Kind = "note"
)

// Entry is one line of the stream, with its detail available on demand.
type Entry struct {
	Kind     Kind
	Tool     string
	Target   string
	Summary  string
	Detail   string
	IsError  bool
	Expanded bool
	Seq      uint64
}

// Model is the view state, derived entirely from the event log.
type Model struct {
	SessionID string
	Workspace string
	Model     string
	Sandbox   string
	State     protocol.SessionState

	Entries []Entry
	Plan    []protocol.PlanItem
	Pending *protocol.ApprovalRequest

	InputTokens  int
	OutputTokens int
	CacheTokens  int
	ContextPct   int

	LastSeq uint64

	// Client-local state. Nothing here is session state (RN-11).
	Cursor      int
	ScrollTop   int
	PanelHidden bool
	Queue       []string
	Input       string

	// turnStarted tracks whether a turn has ever run, so the empty state can
	// disappear on the first one and never return.
	turnStarted bool
	activeTurn  string
}

// NewModel builds an empty view.
func NewModel(sessionID, workspace, model, sandbox string) Model {
	return Model{
		SessionID: sessionID, Workspace: workspace, Model: model,
		Sandbox: sandbox, State: protocol.SessionStateIdle, Cursor: -1,
	}
}

// ShowEmptyState reports whether the splash should render.
//
// It disappears on the first turn and never returns: a persistent splash steals
// height from the stream, which is the scarce resource on screen. A resumed
// session never shows it, because someone resuming wants to see where they left
// off.
func (m Model) ShowEmptyState() bool {
	return !m.turnStarted && len(m.Entries) == 0
}

// Apply folds one event into the view. Pure: the same sequence of events always
// produces the same model, which is what makes replay equal live observation.
func (m Model) Apply(ev protocol.Event) Model {
	m.LastSeq = ev.Seq

	switch ev.Type {
	case protocol.EventSessionCreated:
		var s protocol.Session
		if err := json.Unmarshal(ev.Payload, &s); err == nil {
			m.SessionID, m.Workspace = s.ID, s.Workspace
			m.Model, m.Sandbox, m.State = s.Model, s.SandboxMode, s.State
		}

	case protocol.EventTurnStarted:
		var d protocol.TurnStarted
		_ = json.Unmarshal(ev.Payload, &d)
		m.turnStarted = true
		m.activeTurn = d.TurnID
		m.State = protocol.SessionStateRunning

	case protocol.EventMessageDelta:
		var d protocol.MessageDelta
		if err := json.Unmarshal(ev.Payload, &d); err != nil {
			break
		}
		// Text streams in fragments; appending to the open assistant entry is
		// what makes it feel alive rather than arriving in one block.
		if n := len(m.Entries); n > 0 && m.Entries[n-1].Kind == KindAssistant {
			m.Entries = append([]Entry(nil), m.Entries...)
			m.Entries[n-1].Summary += d.Text
		} else {
			m.Entries = append(m.Entries, Entry{
				Kind: KindAssistant, Summary: d.Text, Seq: ev.Seq,
			})
		}

	case protocol.EventToolRequested:
		var d protocol.ToolRequested
		if err := json.Unmarshal(ev.Payload, &d); err != nil {
			break
		}
		m.Entries = append(m.Entries, Entry{
			Kind: KindTool, Tool: d.Name, Target: targetOf(d.Input),
			Summary: "…", Seq: ev.Seq,
		})

	case protocol.EventToolCompleted:
		var d protocol.ToolCompleted
		if err := json.Unmarshal(ev.Payload, &d); err != nil {
			break
		}
		m.Entries = append([]Entry(nil), m.Entries...)
		for i := len(m.Entries) - 1; i >= 0; i-- {
			if m.Entries[i].Kind != KindTool || m.Entries[i].Summary != "…" {
				continue
			}
			m.Entries[i].Detail = d.Output
			m.Entries[i].IsError = !d.OK
			m.Entries[i].Summary = summariseResult(m.Entries[i].Tool, d)
			// Errors open, successes stay collapsed: failure needs attention,
			// success needs only confirmation.
			m.Entries[i].Expanded = !d.OK
			break
		}

	case protocol.EventApprovalRequired:
		var d protocol.ApprovalRequest
		if err := json.Unmarshal(ev.Payload, &d); err == nil {
			m.Pending = &d
			m.State = protocol.SessionStateBlocked
		}

	case protocol.EventApprovalResolved:
		m.Pending = nil
		if m.State == protocol.SessionStateBlocked {
			m.State = protocol.SessionStateRunning
		}

	case protocol.EventPlanUpdated:
		var d protocol.PlanUpdated
		if err := json.Unmarshal(ev.Payload, &d); err == nil {
			m.Plan = d.Items
		}

	case protocol.EventSessionCompacted:
		m.Entries = append(m.Entries, Entry{
			Kind: KindNote, Summary: "earlier history was summarised", Seq: ev.Seq,
		})

	case protocol.EventSessionError:
		var d protocol.Error
		if err := json.Unmarshal(ev.Payload, &d); err == nil {
			m.Entries = append(m.Entries, Entry{
				Kind: KindError, Summary: d.Message, Detail: d.Code,
				IsError: true, Expanded: true, Seq: ev.Seq,
			})
		}

	case protocol.EventTurnCompleted:
		m.State = protocol.SessionStateIdle
		m.activeTurn = ""
	}
	return m
}

// ToggleAt expands or collapses one entry.
func (m Model) ToggleAt(i int) Model {
	if i < 0 || i >= len(m.Entries) {
		return m
	}
	entries := append([]Entry(nil), m.Entries...)
	entries[i].Expanded = !entries[i].Expanded
	m.Entries = entries
	return m
}

// Enqueue accepts input while a turn is running.
//
// The queue is client-local: the protocol refuses a concurrent turn, so waiting
// here is what turns a refusal into a usable experience instead of an error the
// user has to work around.
func (m Model) Enqueue(text string, max int) (Model, bool) {
	if strings.TrimSpace(text) == "" {
		return m, false
	}
	if max > 0 && len(m.Queue) >= max {
		// Refusing and saying so beats dropping silently: the user would
		// otherwise believe the message was sent.
		return m, false
	}
	m.Queue = append(append([]string(nil), m.Queue...), text)
	return m, true
}

// DrainQueue returns the queued messages joined into one turn.
//
// One turn, not several: multiple turns would violate the one-turn-per-session
// rule and reorder the event log.
func (m Model) DrainQueue() (Model, string) {
	if len(m.Queue) == 0 {
		return m, ""
	}
	joined := strings.Join(m.Queue, "\n\n")
	m.Queue = nil
	return m, joined
}

// RemoveFromQueue drops a queued message before it is sent.
func (m Model) RemoveFromQueue(i int) Model {
	if i < 0 || i >= len(m.Queue) {
		return m
	}
	q := append([]string(nil), m.Queue...)
	m.Queue = append(q[:i], q[i+1:]...)
	return m
}

// PlanCounts returns done, total and blocked.
func (m Model) PlanCounts() (done, total, blocked int) {
	for _, it := range m.Plan {
		total++
		switch it.Status {
		case protocol.PlanDone:
			done++
		case protocol.PlanBlocked:
			blocked++
		}
	}
	return
}

// PlanSummary is the footer line.
//
// The same string is used in the panel and in the status bar when the panel
// collapses. One formulation, or the two drift apart at the first change.
func (m Model) PlanSummary() string {
	done, total, blocked := m.PlanCounts()
	if total == 0 {
		return ""
	}
	s := fmt.Sprintf("%d of %d", done, total)
	if blocked > 0 {
		s += fmt.Sprintf(" · %d blocked", blocked)
	}
	return s
}

func targetOf(raw json.RawMessage) string {
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		return ""
	}
	for _, k := range []string{"path", "pattern", "command"} {
		if v, ok := m[k].(string); ok {
			return v
		}
	}
	return ""
}

// summariseResult renders the one-line form of a completed tool call. `go test`
// produces two hundred lines; what the user needs to see is "12 passed".
func summariseResult(tool string, d protocol.ToolCompleted) string {
	first := strings.TrimSpace(d.Output)
	if i := strings.IndexByte(first, '\n'); i >= 0 {
		first = first[:i]
	}
	if !d.OK {
		if first == "" {
			first = "failed"
		}
		return first
	}
	switch tool {
	case "bash":
		return first
	case "read", "edit", "write", "glob", "grep", "plan":
		if first == "" {
			return "ok"
		}
		return first
	}
	if first == "" {
		return "ok"
	}
	return first
}
