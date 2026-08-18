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
	"time"

	"github.com/aguinelo/dcode/internal/config"
	"github.com/aguinelo/dcode/internal/loop"
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
	// KindReasoning is the model thinking, which is not the model answering.
	KindReasoning Kind = "reasoning"
	// KindCompletion is what was and was not checked when the turn ended.
	//
	// Its own kind rather than a note, because it is the one line on screen the
	// model's prose cannot contradict: the text may claim success, the seal is
	// derived from what actually ran.
	KindCompletion Kind = "completion"
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
	// Duration is how long the tool took, as the daemon measured it.
	Duration time.Duration
	// Running marks a tool call that has not reported back yet, which is what
	// the spinner attaches to.
	Running bool
	// Diff is the unified diff of a change. When present it is what the
	// expansion shows: it is what gets reviewed, and the tool's prose summary
	// says nothing a reviewer needs.
	Diff string
	// StartedAt and Closed belong to a thought: it streams live while open and
	// collapses to one line once the turn moves on, because thinking runs
	// several times the length of the answer and would otherwise bury it.
	StartedAt time.Time
	Closed    bool
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
	// Lang is the interface language, resolved once when the client starts.
	Lang Lang
	// Copy is the selection while copy mode is open. The alternate screen costs
	// the terminal's own selection, and RN-1 says it has to be given back.
	Copy CopyState
	// Flash is a one-line notice shown until the next keystroke, for things
	// that happen and leave no other trace — a copy landing, for instance.
	Flash string
	// Verification is the seal of the last completed turn. Empty when the turn
	// had no definition of done.
	//
	// It is the guarantee that outlives a model claiming success in prose: the
	// text can lie, this is derived from what actually ran.
	Verification string
	Pending      *protocol.ApprovalRequest
	// The diff accumulated in this worktree, summed from what each tool
	// reported rather than parsed back out of its text.
	DiffAdded   int
	DiffRemoved int
	DiffFiles   int

	InputTokens  int
	OutputTokens int
	CacheTokens  int
	ContextPct   int

	LastSeq uint64

	// Window is the model's context window, so a token count can become a
	// percentage. Zero means the daemon did not report one.
	Window int

	// TurnStartedAt is when the running turn began, measured by the client.
	// Elapsed time is client-local on purpose: it must tick between events,
	// and a server-sent timestamp would only be right at the instant it
	// arrived.
	TurnStartedAt time.Time
	// Frame advances on every animation tick. Render stays pure by reading it
	// rather than a clock.
	Frame int
	// Now is the clock the view was rendered against, for elapsed time.
	Now time.Time

	// Client-local state. Nothing here is session state (RN-11).
	Cursor    int
	ScrollTop int
	// Follow keeps the newest output on screen. It turns off the moment the
	// user scrolls up — reading something while the stream pushes it away is
	// the single most irritating thing a live log can do — and back on when
	// they return to the bottom.
	Follow bool
	Queue  []string
	// Attached are pictures waiting for the next question. A picture with no
	// question is a turn the model has to guess the point of.
	Attached []protocol.TurnImage
	Input    string
	// InputCursor is the caret position within Input, in runes.
	InputCursor int
	// History is what the user has sent, newest last, with HistoryAt as the
	// position being browsed. Client-local: it is what this person typed at
	// this terminal, not session state.
	History   []string
	HistoryAt int
	// Draft holds what was typed before the user started browsing history, so
	// coming back out of the history does not lose it.
	Draft string

	// Completions is the open `/` menu, with CompletionAt as the highlighted
	// row. Empty means no menu — the menu is a consequence of what is typed,
	// never a mode the user has to leave.
	Completions  []Completion
	CompletionAt int
	// CompletionsOff suppresses the menu until the line changes, which is how
	// Esc closes it without also clearing the line. Any edit revives it —
	// dismissing the menu answered for the line as it was, not for every line
	// that follows.
	CompletionsOff bool

	// turnStarted tracks whether a turn has ever run, so the empty state can
	// disappear on the first one and never return.
	turnStarted bool
	activeTurn  string
}

// NewModel builds an empty view.
// NewModel builds the client state.
//
// The language is a parameter rather than resolved here, for the same reason
// the palette is not built here: this package renders, and the environment is
// read once, at the edge, by whoever starts the client. A zero Lang lands on the
// fallback, which is the documented behaviour and not an accident.
func NewModel(sessionID, workspace, model, sandbox string, lang Lang) Model {
	return Model{
		SessionID: sessionID, Workspace: workspace, Model: model,
		Sandbox: sandbox, Lang: lang, State: protocol.SessionStateIdle, Cursor: -1,
		Follow: true, HistoryAt: -1,
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
			m.Window = s.ContextWindow
		}

	case protocol.EventSessionResumed:
		// The events that follow happened in another session. Saying so is what
		// stops a continued conversation from reading as work this one did —
		// and, for somebody who ran `-r` and expected to find their afternoon,
		// it is the line that says it is there.
		var d protocol.SessionResumed
		if err := json.Unmarshal(ev.Payload, &d); err == nil {
			m.Entries = append(m.Entries, Entry{
				Kind: KindNote, Seq: ev.Seq,
				Summary: fmt.Sprintf(Text(m.Lang).Resumed, d.SourceID, d.Turns),
			})
		}

	case protocol.EventTurnSteered:
		// The person's own words, in the stream as the person. From the event
		// rather than echoed locally, for the reason turn.started already
		// records: the only copy would live in the client that did the typing,
		// and a second client would see the turn change direction for no
		// visible reason.
		var d protocol.TurnSteered
		_ = json.Unmarshal(ev.Payload, &d)
		if d.Text != "" {
			m.Entries = append(m.Entries, Entry{Kind: KindUser, Summary: d.Text})
		}

	case protocol.EventTurnStarted:
		var d protocol.TurnStarted
		_ = json.Unmarshal(ev.Payload, &d)
		// The question, from the event rather than echoed locally when it was
		// typed. Echoing locally meant a second client — or this one after a
		// replay — saw answers to questions it never saw, because the only
		// copy lived in the client that did the typing.
		if d.Text != "" {
			m.Entries = append(m.Entries, Entry{Kind: KindUser, Summary: d.Text})
		}
		m.turnStarted = true
		m.activeTurn = d.TurnID
		m.State = protocol.SessionStateRunning
		m.TurnStartedAt = m.Now

	case protocol.EventMessageReasoning:
		var d protocol.MessageReasoning
		if err := json.Unmarshal(ev.Payload, &d); err != nil {
			break
		}
		// Appended to the open thought, the same way text is: thinking arrives
		// in fragments and reads as thinking only when it flows.
		if n := len(m.Entries); n > 0 && m.Entries[n-1].Kind == KindReasoning &&
			!m.Entries[n-1].Closed {
			m.Entries = append([]Entry(nil), m.Entries...)
			m.Entries[n-1].Summary += d.Text
		} else {
			m.Entries = append(m.Entries, Entry{
				Kind: KindReasoning, Summary: d.Text, Seq: ev.Seq, StartedAt: m.Now,
			})
		}

	case protocol.EventMessageDelta:
		var d protocol.MessageDelta
		if err := json.Unmarshal(ev.Payload, &d); err != nil {
			break
		}
		// Text streams in fragments; appending to the open assistant entry is
		// what makes it feel alive rather than arriving in one block.
		m = m.closeThought()
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
		m = m.closeThought()
		m.Entries = append(m.Entries, Entry{
			Kind: KindTool, Tool: d.Name, Target: targetOf(d.Input),
			Summary: "", Running: true, Seq: ev.Seq,
		})

	case protocol.EventToolCompleted:
		var d protocol.ToolCompleted
		if err := json.Unmarshal(ev.Payload, &d); err != nil {
			break
		}
		// What changed here, summed as the tools report it. A call that changed
		// nothing is not a file touched: counting a read would make the number
		// mean "calls" while reading as "files".
		if d.Added > 0 || d.Removed > 0 {
			m.DiffAdded += d.Added
			m.DiffRemoved += d.Removed
			m.DiffFiles++
		}
		m.Entries = append([]Entry(nil), m.Entries...)
		for i := len(m.Entries) - 1; i >= 0; i-- {
			if m.Entries[i].Kind != KindTool || !m.Entries[i].Running {
				continue
			}
			m.Entries[i].Detail = d.Output
			m.Entries[i].Diff = d.Diff
			m.Entries[i].IsError = !d.OK
			m.Entries[i].Summary = summariseResult(m.Entries[i].Tool, d)
			m.Entries[i].Duration = time.Duration(d.DurationMS) * time.Millisecond
			m.Entries[i].Running = false
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
			// A newly blocked item is news, and the panel states it in three
			// words. The stream is where it can be said in full, in the place
			// the reader already is.
			for _, note := range newlyBlocked(m.Plan, d.Items) {
				m.Entries = append(m.Entries, Entry{
					Kind: KindNote, Summary: note, Seq: ev.Seq, Expanded: true,
				})
			}
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
		m = m.closeThought()
		var d protocol.TurnCompleted
		_ = json.Unmarshal(ev.Payload, &d)
		if d.Usage != nil {
			m.InputTokens = d.Usage.InputTokens
			m.OutputTokens = d.Usage.OutputTokens
			m.CacheTokens = d.Usage.CacheReadTokens
			// The input of the last turn is what the context currently costs,
			// which is the number a person can act on.
			if m.Window > 0 {
				m.ContextPct = 100 * d.Usage.InputTokens / m.Window
			}
		}
		if e, ok := completionEntry(d.Completion, m.Lang); ok {
			m.Entries = append(m.Entries, e)
		}
		m.Verification = ""
		if d.Completion != nil {
			m.Verification = d.Completion.Verification
		}
		m.State = protocol.SessionStateIdle
		m.activeTurn = ""
		m.TurnStartedAt = time.Time{}
	}
	return m
}

// completionEntry renders what the turn ended knowing.
//
// Nothing is shown when there was no definition of done: a line saying
// "nothing to check" on every turn is a line that stops being read, and it
// would drown the turns where there IS something to say.
func completionEntry(c *protocol.Completion, lang Lang) (Entry, bool) {
	t := Text(lang)
	if c == nil {
		return Entry{}, false
	}
	// A clean turn changed nothing, so there was nothing to verify and nothing
	// worth a line.
	if c.Verification == string(loop.VerificationClean) {
		return Entry{}, false
	}

	summary := completionSummary(c, lang)
	var detail strings.Builder
	for _, group := range []struct {
		label string
		names []string
	}{
		{t.CompletionMet, c.Met},
		{t.CompletionUnmet, c.Unmet},
		{t.CompletionUnchecked, c.Unavailable},
	} {
		if len(group.names) > 0 {
			fmt.Fprintf(&detail, "%-22s %s\n", group.label, strings.Join(group.names, ", "))
		}
	}
	if len(c.TouchedProtected) > 0 {
		// Never folded into the detail silently. Changing what measures the
		// work is sometimes right and always worth seeing.
		fmt.Fprintf(&detail, "%-22s %s\n", t.CompletionMeasure, strings.Join(c.TouchedProtected, ", "))
	}
	return Entry{Kind: KindCompletion, Summary: summary, Detail: strings.TrimRight(detail.String(), "\n")}, true
}

func completionSummary(c *protocol.Completion, lang Lang) string {
	t := Text(lang)
	switch c.Verification {
	case string(loop.VerificationPassed):
		return fmt.Sprintf(t.VerifiedSummary, len(c.Met), plural(len(c.Met), "check", "checks"))
	case string(loop.VerificationFailed):
		return fmt.Sprintf(t.NotVerifiedSummary, strings.Join(c.Unmet, ", "))
	case string(loop.VerificationUnavailable):
		return t.NothingCouldCheck
	case string(loop.VerificationStale):
		return t.ChangedAfterCheck
	default:
		return c.Verification
	}
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

// summariseResult renders the one-line form of a completed tool call.
//
// It reads the metadata the tool reported rather than parsing the output. `go
// test` prints two hundred lines; what the user needs on screen is "12 passed",
// and rebuilding that by matching prose breaks the day the wording changes.
func summariseResult(tool string, d protocol.ToolCompleted) string {
	if !d.OK {
		if s := firstLine(d.Output); s != "" {
			return s
		}
		return "failed"
	}

	switch tool {
	case "read":
		if d.Lines > 0 {
			s := fmt.Sprintf("%d lines", d.Lines)
			if d.Truncated {
				s += " (truncated)"
			}
			return s
		}
	case "edit":
		return fmt.Sprintf("+%d −%d", d.Added, d.Removed)
	case "write":
		if d.Removed == 0 && d.Added > 0 {
			return fmt.Sprintf("created, %d lines", d.Added)
		}
		return fmt.Sprintf("+%d −%d", d.Added, d.Removed)
	case "glob":
		return plural(d.Files, "file", "files")
	case "grep":
		if d.Lines == 0 {
			return "no matches"
		}
		return fmt.Sprintf("%s in %s",
			plural(d.Lines, "match", "matches"), plural(d.Files, "file", "files"))
	case "bash":
		if d.HasExit {
			if d.ExitCode == 0 {
				return "exit 0"
			}
			return fmt.Sprintf("exit %d", d.ExitCode)
		}
	}
	if s := firstLine(d.Output); s != "" {
		return s
	}
	return "ok"
}

func firstLine(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		s = s[:i]
	}
	return s
}

func plural(n int, one, many string) string {
	if n == 1 {
		return "1 " + one
	}
	return fmt.Sprintf("%d %s", n, many)
}

// FormatDuration renders a tool's elapsed time the way a reader scans it:
// milliseconds while it is fast enough not to notice, seconds once it is not.
func FormatDuration(d time.Duration) string {
	switch {
	case d <= 0:
		return ""
	case d < time.Second:
		return fmt.Sprintf("%dms", d.Milliseconds())
	case d < time.Minute:
		return fmt.Sprintf("%.1fs", d.Seconds())
	default:
		return fmt.Sprintf("%dm%02ds", int(d.Minutes()), int(d.Seconds())%60)
	}
}

// ---------- the input line ----------
//
// Editing lives on the model rather than in the key handler so it is testable
// without a terminal, and so the caret and the text can never disagree about
// where they are.

// Insert types text at the caret.
func (m Model) Insert(s string) Model {
	runes := []rune(m.Input)
	at := clampInt(m.InputCursor, 0, len(runes))
	m.Input = string(runes[:at]) + s + string(runes[at:])
	m.InputCursor = at + len([]rune(s))
	// Typing leaves the history: what is on the line is now the user's, not a
	// recalled command they might still want back.
	m.HistoryAt = -1
	// And it reopens the menu: Esc dismissed the menu for the line as it was,
	// not for every line that follows.
	m.CompletionsOff = false
	return m
}

// Backspace deletes before the caret.
func (m Model) Backspace() Model {
	runes := []rune(m.Input)
	at := clampInt(m.InputCursor, 0, len(runes))
	if at == 0 {
		return m
	}
	m.Input = string(runes[:at-1]) + string(runes[at:])
	m.InputCursor = at - 1
	m.CompletionsOff = false
	return m
}

// DeleteForward deletes under the caret.
func (m Model) DeleteForward() Model {
	runes := []rune(m.Input)
	at := clampInt(m.InputCursor, 0, len(runes))
	if at >= len(runes) {
		return m
	}
	m.Input = string(runes[:at]) + string(runes[at+1:])
	m.CompletionsOff = false
	return m
}

// DeleteWord removes the word before the caret, trailing spaces included.
func (m Model) DeleteWord() Model {
	runes := []rune(m.Input)
	at := clampInt(m.InputCursor, 0, len(runes))
	i := at
	for i > 0 && runes[i-1] == ' ' {
		i--
	}
	for i > 0 && runes[i-1] != ' ' {
		i--
	}
	m.Input = string(runes[:i]) + string(runes[at:])
	m.InputCursor = i
	m.CompletionsOff = false
	return m
}

// SetInput replaces the line and puts the caret at its end.
func (m Model) SetInput(s string) Model {
	m.Input = s
	m.InputCursor = len([]rune(s))
	return m
}

// ---------- input history ----------

// Remember records a sent line. Consecutive duplicates are collapsed: pressing
// up twice should reach two different commands, not the same one again.
func (m Model) Remember(text string) Model {
	text = strings.TrimSpace(text)
	if text == "" {
		return m
	}
	if n := len(m.History); n > 0 && m.History[n-1] == text {
		m.HistoryAt = -1
		return m
	}
	m.History = append(append([]string(nil), m.History...), text)
	m.HistoryAt = -1
	return m
}

// HistoryPrev walks back through what was sent.
func (m Model) HistoryPrev() Model {
	if len(m.History) == 0 {
		return m
	}
	if m.HistoryAt < 0 {
		// Entering the history keeps whatever was being typed, so leaving it
		// again does not silently discard a half-written message.
		m.Draft = m.Input
		m.HistoryAt = len(m.History)
	}
	if m.HistoryAt == 0 {
		return m
	}
	m.HistoryAt--
	return m.SetInput(m.History[m.HistoryAt])
}

// HistoryNext walks forward, and past the newest entry returns the draft.
func (m Model) HistoryNext() Model {
	if m.HistoryAt < 0 {
		return m
	}
	m.HistoryAt++
	if m.HistoryAt >= len(m.History) {
		m.HistoryAt = -1
		return m.SetInput(m.Draft)
	}
	return m.SetInput(m.History[m.HistoryAt])
}

func clampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// newlyBlocked reports items that became blocked in this update.
//
// Only the transition, never the state: an item that stays blocked would
// otherwise announce itself on every plan update, and a message repeated
// without new information is one the reader learns to skip.
func newlyBlocked(before, after []protocol.PlanItem) []string {
	was := make(map[int]string, len(before))
	for _, it := range before {
		was[it.ID] = it.Status
	}
	var out []string
	for _, it := range after {
		if it.Status != protocol.PlanBlocked || was[it.ID] == protocol.PlanBlocked {
			continue
		}
		note := fmt.Sprintf("task %d blocked: %s", it.ID, it.Text)
		if it.Blocked != "" {
			note += " — " + it.Blocked
		}
		out = append(out, note)
	}
	return out
}

// ---------- the completion menu ----------

// Refresh recomputes the menu from the current line.
//
// Derived rather than toggled: a menu with its own open/closed state drifts out
// of step with the text the moment anything else edits the line.
func (m Model) Refresh(user config.CommandSet) Model {
	if m.CompletionsOff {
		m.Completions = nil
		return m
	}
	got := Complete(m.Input, user, m.Lang)
	// The highlight resets whenever the candidate set changes, or it would
	// point at a different command than the one it pointed at a keystroke ago.
	if len(got) != len(m.Completions) {
		m.CompletionAt = 0
	}
	m.Completions = got
	if m.CompletionAt >= len(got) {
		m.CompletionAt = 0
	}
	return m
}

// MoveCompletion walks the menu, wrapping at both ends.
func (m Model) MoveCompletion(delta int) Model {
	if n := len(m.Completions); n > 0 {
		m.CompletionAt = ((m.CompletionAt+delta)%n + n) % n
	}
	return m
}

// AcceptCompletion puts the highlighted command on the line.
//
// A trailing space when the command takes arguments: the user's next keystroke
// is the argument, and making them type the separator is a small tax on every
// single use.
func (m Model) AcceptCompletion() Model {
	if len(m.Completions) == 0 {
		return m
	}
	c := m.Completions[m.CompletionAt]
	text := "/" + c.Name
	if c.Args != "" {
		text += " "
	}
	m = m.SetInput(text)
	m.Completions = nil
	return m
}

// CloseCompletions hides the menu until the line changes.
func (m Model) CloseCompletions() Model {
	m.Completions = nil
	m.CompletionsOff = true
	return m
}

// closeThought collapses the thought in progress, if there is one.
//
// A thought closes when the turn moves on — the first answer fragment, the
// first tool call, the end of the turn. Live it is the most informative thing
// on screen; once the model has acted, it is scratch work sitting in front of
// the result.
func (m Model) closeThought() Model {
	n := len(m.Entries)
	if n == 0 || m.Entries[n-1].Kind != KindReasoning || m.Entries[n-1].Closed {
		return m
	}
	m.Entries = append([]Entry(nil), m.Entries...)
	m.Entries[n-1].Closed = true
	if !m.Entries[n-1].StartedAt.IsZero() && !m.Now.IsZero() {
		m.Entries[n-1].Duration = m.Now.Sub(m.Entries[n-1].StartedAt)
	}
	return m
}
