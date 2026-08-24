package tui

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/aguinelo/dcode/internal/protocol"
)

func ev(t *testing.T, seq uint64, typ protocol.EventType, payload any) protocol.Event {
	t.Helper()
	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	return protocol.Event{Seq: seq, Type: typ, Payload: raw}
}

func apply(t *testing.T, m Model, evs ...protocol.Event) Model {
	t.Helper()
	for _, e := range evs {
		m = m.Apply(e)
	}
	return m
}

// The same sequence of events always produces the same model. That is what
// makes reattaching to a session indistinguishable from having watched it live.
func TestApplyIsAPureReducer(t *testing.T) {
	evs := []protocol.Event{
		ev(t, 1, protocol.EventSessionCreated, protocol.Session{
			ID: "s1", Workspace: "/w", Model: "m", SandboxMode: "read-only",
			State: protocol.SessionStateIdle,
		}),
		ev(t, 2, protocol.EventTurnStarted, protocol.TurnStarted{TurnID: "t1"}),
		ev(t, 3, protocol.EventMessageDelta, protocol.MessageDelta{Text: "hel"}),
		ev(t, 4, protocol.EventMessageDelta, protocol.MessageDelta{Text: "lo"}),
		ev(t, 5, protocol.EventToolRequested, protocol.ToolRequested{
			Name: "read", Input: json.RawMessage(`{"path":"main.go"}`),
		}),
		ev(t, 6, protocol.EventToolCompleted, protocol.ToolCompleted{OK: true, Output: "12 lines"}),
		ev(t, 7, protocol.EventTurnCompleted, protocol.TurnCompleted{TurnID: "t1"}),
	}

	first := apply(t, NewModel("", "", "", "", En), evs...)
	second := apply(t, NewModel("", "", "", "", En), evs...)

	if len(first.Entries) != len(second.Entries) {
		t.Fatalf("got %d and %d entries", len(first.Entries), len(second.Entries))
	}
	for i := range first.Entries {
		// reflect rather than !=: Entry carries the paths a delegated child
		// declared it owns, and a slice makes the struct incomparable.
		if !reflect.DeepEqual(first.Entries[i], second.Entries[i]) {
			t.Errorf("entry %d differs:\n%+v\n%+v", i, first.Entries[i], second.Entries[i])
		}
	}
	if first.State != protocol.SessionStateIdle || first.LastSeq != 7 {
		t.Errorf("got %v at seq %d", first.State, first.LastSeq)
	}
}

// Text arrives in fragments; appending to the open entry is what makes it feel
// alive rather than landing in one block.
func TestDeltasAppendToTheOpenAssistantEntry(t *testing.T) {
	m := apply(t, NewModel("", "", "", "", En),
		ev(t, 1, protocol.EventMessageDelta, protocol.MessageDelta{Text: "one "}),
		ev(t, 2, protocol.EventMessageDelta, protocol.MessageDelta{Text: "two"}),
	)
	if len(m.Entries) != 1 || m.Entries[0].Summary != "one two" {
		t.Fatalf("got %+v", m.Entries)
	}

	// A tool call closes the entry; the next delta starts a new one.
	m = apply(t, m,
		ev(t, 3, protocol.EventToolRequested, protocol.ToolRequested{Name: "read"}),
		ev(t, 4, protocol.EventMessageDelta, protocol.MessageDelta{Text: "after"}),
	)
	if len(m.Entries) != 3 {
		t.Fatalf("got %+v", m.Entries)
	}
}

// Failure needs attention; success needs only confirmation.
func TestErrorsOpenAndSuccessesStayCollapsed(t *testing.T) {
	m := apply(t, NewModel("", "", "", "", En),
		ev(t, 1, protocol.EventToolRequested, protocol.ToolRequested{Name: "bash"}),
		ev(t, 2, protocol.EventToolCompleted, protocol.ToolCompleted{
			OK: false, Output: "exit 1\nstack trace",
		}),
	)
	if !m.Entries[0].Expanded || !m.Entries[0].IsError {
		t.Errorf("a failure must open itself: %+v", m.Entries[0])
	}

	m2 := apply(t, NewModel("", "", "", "", En),
		ev(t, 1, protocol.EventToolRequested, protocol.ToolRequested{Name: "bash"}),
		ev(t, 2, protocol.EventToolCompleted, protocol.ToolCompleted{OK: true, Output: "12 passed"}),
	)
	if m2.Entries[0].Expanded {
		t.Errorf("a success stays collapsed: %+v", m2.Entries[0])
	}
	if m2.Entries[0].Summary != "12 passed" {
		t.Errorf("the summary is the first line, got %q", m2.Entries[0].Summary)
	}
}

func TestToolTargetComesFromTheInput(t *testing.T) {
	for input, want := range map[string]string{
		`{"path":"main.go"}`:    "main.go",
		`{"pattern":"*.go"}`:    "*.go",
		`{"command":"go test"}`: "go test",
		`{"items":[]}`:          "",
	} {
		m := apply(t, NewModel("", "", "", "", En),
			ev(t, 1, protocol.EventToolRequested, protocol.ToolRequested{
				Name: "x", Input: json.RawMessage(input),
			}))
		if got := m.Entries[0].Target; got != want {
			t.Errorf("%s: got %q want %q", input, got, want)
		}
	}

	// A tool whose input is not an object still gets an entry: dropping the
	// call would leave the user watching nothing happen.
	m := NewModel("", "", "", "", En).Apply(protocol.Event{
		Seq: 1, Type: protocol.EventToolRequested,
		Payload: json.RawMessage(`{"name":"x","input":"not an object"}`),
	})
	if len(m.Entries) != 1 || m.Entries[0].Target != "" {
		t.Errorf("got %+v", m.Entries)
	}
}

func TestApprovalMovesTheSessionToBlockedAndBack(t *testing.T) {
	m := apply(t, NewModel("", "", "", "", En),
		ev(t, 1, protocol.EventApprovalRequired, protocol.ApprovalRequest{
			ApprovalID: "a1", Tool: "bash", BoundaryCrossed: "network",
		}))
	if m.Pending == nil || m.State != protocol.SessionStateBlocked {
		t.Fatalf("got %+v %v", m.Pending, m.State)
	}
	m = apply(t, m, ev(t, 2, protocol.EventApprovalResolved, protocol.ApprovalResolved{ApprovalID: "a1"}))
	if m.Pending != nil || m.State != protocol.SessionStateRunning {
		t.Errorf("got %+v %v", m.Pending, m.State)
	}
}

func TestCompactionAndErrorsBecomeEntries(t *testing.T) {
	m := apply(t, NewModel("", "", "", "", En),
		ev(t, 1, protocol.EventSessionCompacted, protocol.SessionCompacted{}),
		ev(t, 2, protocol.EventSessionError, protocol.Error{Code: "boom", Message: "it broke"}),
	)
	if len(m.Entries) != 2 {
		t.Fatalf("got %+v", m.Entries)
	}
	if m.Entries[0].Kind != KindNote {
		t.Errorf("compaction is a note, not an error: %+v", m.Entries[0])
	}
	if m.Entries[1].Kind != KindError || !m.Entries[1].Expanded {
		t.Errorf("a session error must be visible: %+v", m.Entries[1])
	}
}

// A malformed payload must not take the client down: the daemon and the client
// version independently.
func TestApplyIgnoresUnreadablePayloads(t *testing.T) {
	m := NewModel("", "", "", "", En)
	for _, typ := range []protocol.EventType{
		protocol.EventSessionCreated, protocol.EventMessageDelta,
		protocol.EventToolRequested, protocol.EventToolCompleted,
		protocol.EventApprovalRequired, protocol.EventPlanUpdated,
		protocol.EventSessionError,
	} {
		m = m.Apply(protocol.Event{Seq: 1, Type: typ, Payload: json.RawMessage(`"nope"`)})
	}
	if len(m.Entries) != 0 {
		t.Errorf("garbage must be dropped, not rendered: %+v", m.Entries)
	}
}

// ---------- empty state ----------

// A persistent splash steals height from the stream, which is the scarce
// resource on screen.
func TestEmptyStateDisappearsOnTheFirstTurnAndNeverReturns(t *testing.T) {
	m := NewModel("s", "/w", "m", "read-only", En)
	if !m.ShowEmptyState() {
		t.Fatal("a fresh session shows the splash")
	}
	m = apply(t, m, ev(t, 1, protocol.EventTurnStarted, protocol.TurnStarted{TurnID: "t1"}))
	if m.ShowEmptyState() {
		t.Error("the splash goes on the first turn")
	}
	m = apply(t, m, ev(t, 2, protocol.EventTurnCompleted, protocol.TurnCompleted{TurnID: "t1"}))
	if m.ShowEmptyState() {
		t.Error("and it does not come back")
	}
}

// ---------- queue ----------

// The protocol refuses a concurrent turn, so the queue is what turns a refusal
// into a usable experience.
func TestQueueJoinsIntoOneTurnInTypingOrder(t *testing.T) {
	m := NewModel("", "", "", "", En)
	for _, s := range []string{"first", "second"} {
		var ok bool
		m, ok = m.Enqueue(s, 10)
		if !ok {
			t.Fatalf("%q was refused", s)
		}
	}
	m, text := m.DrainQueue()
	if !strings.HasPrefix(text, "first") || !strings.Contains(text, "second") {
		t.Errorf("got %q", text)
	}
	if strings.Index(text, "first") > strings.Index(text, "second") {
		t.Errorf("typing order must survive: %q", text)
	}
	if len(m.Queue) != 0 {
		t.Errorf("the queue must drain, got %v", m.Queue)
	}
	if _, empty := m.DrainQueue(); empty != "" {
		t.Errorf("draining an empty queue yields nothing, got %q", empty)
	}
}

// Refusing and saying so beats dropping silently: the user would otherwise
// believe the message was sent.
func TestQueueRefusesWhenFullAndRefusesBlankInput(t *testing.T) {
	m := NewModel("", "", "", "", En)
	m, _ = m.Enqueue("one", 1)
	if _, ok := m.Enqueue("two", 1); ok {
		t.Error("a full queue must refuse")
	}
	if _, ok := m.Enqueue("   ", 10); ok {
		t.Error("blank input is not a message")
	}
}

func TestRemoveFromQueue(t *testing.T) {
	m := NewModel("", "", "", "", En)
	m, _ = m.Enqueue("a", 10)
	m, _ = m.Enqueue("b", 10)
	m = m.RemoveFromQueue(0)
	if len(m.Queue) != 1 || m.Queue[0] != "b" {
		t.Errorf("got %v", m.Queue)
	}
	// Out of range is a no-op rather than a panic: the index comes from a
	// keystroke, and the queue can drain between the keypress and the handler.
	if got := m.RemoveFromQueue(9); len(got.Queue) != 1 {
		t.Errorf("got %v", got.Queue)
	}
	if got := m.RemoveFromQueue(-1); len(got.Queue) != 1 {
		t.Errorf("got %v", got.Queue)
	}
}

// ---------- plan ----------

func TestPlanCountsAndSummary(t *testing.T) {
	m := apply(t, NewModel("", "", "", "", En),
		ev(t, 1, protocol.EventPlanUpdated, protocol.PlanUpdated{Items: []protocol.PlanItem{
			{ID: 1, Text: "a", Status: protocol.PlanDone},
			{ID: 2, Text: "b", Status: protocol.PlanActive},
			{ID: 3, Text: "c", Status: protocol.PlanBlocked, Blocked: "no network"},
		}}))
	done, total, blocked := m.PlanCounts()
	if done != 1 || total != 3 || blocked != 1 {
		t.Fatalf("got %d/%d, %d blocked", done, total, blocked)
	}
	got := m.PlanSummary()
	if !strings.Contains(got, "1 of 3") || !strings.Contains(got, "1 blocked") {
		t.Errorf("got %q", got)
	}
	if empty := NewModel("", "", "", "", En).PlanSummary(); empty != "" {
		t.Errorf("no plan, nothing to say, got %q", empty)
	}
}

func TestToggleAt(t *testing.T) {
	m := apply(t, NewModel("", "", "", "", En),
		ev(t, 1, protocol.EventToolRequested, protocol.ToolRequested{Name: "read"}),
		ev(t, 2, protocol.EventToolCompleted, protocol.ToolCompleted{OK: true, Output: "x"}),
	)
	if m.ToggleAt(0).Entries[0].Expanded == m.Entries[0].Expanded {
		t.Error("toggling must flip the entry")
	}
	// Out of range is a no-op: the cursor starts at -1.
	if got := m.ToggleAt(-1); len(got.Entries) != len(m.Entries) {
		t.Error("an out-of-range toggle changes nothing")
	}
	if got := m.ToggleAt(99); got.Entries[0].Expanded != m.Entries[0].Expanded {
		t.Error("an out-of-range toggle changes nothing")
	}
}

func TestSummariseResultFallsBackToOK(t *testing.T) {
	for _, tool := range []string{"read", "bash", "unknown"} {
		m := apply(t, NewModel("", "", "", "", En),
			ev(t, 1, protocol.EventToolRequested, protocol.ToolRequested{Name: tool}),
			ev(t, 2, protocol.EventToolCompleted, protocol.ToolCompleted{OK: true, Output: ""}),
		)
		if got := m.Entries[0].Summary; got == "…" {
			t.Errorf("%s: the placeholder must be replaced, got %q", tool, got)
		}
	}
	m := apply(t, NewModel("", "", "", "", En),
		ev(t, 1, protocol.EventToolRequested, protocol.ToolRequested{Name: "bash"}),
		ev(t, 2, protocol.EventToolCompleted, protocol.ToolCompleted{OK: false, Output: ""}),
	)
	if m.Entries[0].Summary != "failed" {
		t.Errorf("got %q", m.Entries[0].Summary)
	}
}

// The one-line summary comes from what the tool reported, not from parsing its
// prose. Rebuilding these numbers by matching text breaks the day the wording
// changes, and every client would have to reimplement the same parsing.
func TestToolSummariesComeFromTheMetadata(t *testing.T) {
	for name, tc := range map[string]struct {
		tool string
		done protocol.ToolCompleted
		want string
	}{
		"read":           {"read", protocol.ToolCompleted{OK: true, Lines: 240}, "240 lines"},
		"read truncated": {"read", protocol.ToolCompleted{OK: true, Lines: 2000, Truncated: true}, "2000 lines (truncated)"},
		"edit":           {"edit", protocol.ToolCompleted{OK: true, Added: 24, Removed: 2}, "+24 -2"},
		"write new":      {"write", protocol.ToolCompleted{OK: true, Added: 120}, "created, 120 lines"},
		"write replace":  {"write", protocol.ToolCompleted{OK: true, Added: 120, Removed: 118}, "+120 -118"},
		"glob":           {"glob", protocol.ToolCompleted{OK: true, Files: 18}, "18 files"},
		"glob one":       {"glob", protocol.ToolCompleted{OK: true, Files: 1}, "1 file"},
		"grep":           {"grep", protocol.ToolCompleted{OK: true, Lines: 18, Files: 4}, "18 matches in 4 files"},
		"grep none":      {"grep", protocol.ToolCompleted{OK: true}, "no matches"},
		"bash ok":        {"bash", protocol.ToolCompleted{OK: true, HasExit: true}, "exit 0"},
		"bash failed":    {"bash", protocol.ToolCompleted{OK: true, HasExit: true, ExitCode: 1}, "exit 1"},
		// A failure says what went wrong, whatever the tool.
		"failure":                     {"read", protocol.ToolCompleted{Output: "no such file\nstack"}, "no such file"},
		"failure with nothing to say": {"read", protocol.ToolCompleted{}, "failed"},
		// A tool with no metadata still gets a usable line rather than a blank.
		"unknown tool":  {"custom", protocol.ToolCompleted{OK: true, Output: "pronto\nmais"}, "pronto"},
		"unknown empty": {"custom", protocol.ToolCompleted{OK: true}, "ok"},
	} {
		if got := summariseResult(tc.tool, tc.done); got != tc.want {
			t.Errorf("%s: got %q want %q", name, got, tc.want)
		}
	}
}

// A call that is still running shows that it is, and one that took long enough
// to notice says how long. Every call carrying a duration turns the column into
// noise nobody reads.
func TestTheStreamShowsRunningStateAndSlowCalls(t *testing.T) {
	m := NewModel("s", "/w", "m", "read-only", En)
	m.Entries = []Entry{
		{Kind: KindTool, Tool: "bash", Target: "go test", Running: true},
		{Kind: KindTool, Tool: "read", Target: "a.go", Summary: "12 lines", Duration: 30 * time.Millisecond},
		{Kind: KindTool, Tool: "bash", Target: "go build", Summary: "exit 0", Duration: 4 * time.Second},
	}
	got := Render(m, DefaultGeometry(100, 12))

	if !strings.Contains(got, "4.0s") {
		t.Errorf("a slow call must say so:\n%s", got)
	}
	if strings.Contains(got, "30ms") {
		t.Errorf("a fast call must not clutter the column:\n%s", got)
	}
}

// The duration the daemon measured is what the client shows: the client cannot
// see when the tool actually started.
func TestDurationComesFromTheEvent(t *testing.T) {
	m := apply(t, NewModel("", "", "", "", En),
		ev(t, 1, protocol.EventToolRequested, protocol.ToolRequested{Name: "bash"}),
		ev(t, 2, protocol.EventToolCompleted, protocol.ToolCompleted{
			OK: true, HasExit: true, DurationMS: 1500,
		}),
	)
	if m.Entries[0].Duration != 1500*time.Millisecond {
		t.Errorf("got %v", m.Entries[0].Duration)
	}
	if m.Entries[0].Running {
		t.Error("a completed call is not running")
	}
}

// The context meter needs a denominator: "12400 tokens" answers nothing.
func TestContextPercentNeedsTheWindow(t *testing.T) {
	withWindow := apply(t, NewModel("", "", "", "", En),
		ev(t, 1, protocol.EventSessionCreated, protocol.Session{ContextWindow: 100000}),
		ev(t, 2, protocol.EventTurnCompleted, protocol.TurnCompleted{
			Usage: &protocol.Usage{InputTokens: 34000, OutputTokens: 800},
		}),
	)
	if withWindow.ContextPct != 34 {
		t.Errorf("got %d%%", withWindow.ContextPct)
	}
	if !strings.Contains(Render(withWindow, DefaultGeometry(100, 10)), "ctx 34%") {
		t.Error("the meter must be on the status bar")
	}

	// A large window is where the meter matters most early on, and it is
	// exactly where integer division erases it.
	big := apply(t, NewModel("", "", "", "", En),
		ev(t, 1, protocol.EventSessionCreated, protocol.Session{ContextWindow: 1_000_000}),
		ev(t, 2, protocol.EventTurnCompleted, protocol.TurnCompleted{
			Usage: &protocol.Usage{InputTokens: 5200},
		}),
	)
	if got := Render(big, DefaultGeometry(100, 10)); !strings.Contains(got, "ctx <1%") {
		t.Errorf("a small fraction of a big window must still show:\n%s", got)
	}

	// Without a window there is no percentage to state, and inventing one
	// would be worse than saying nothing.
	noWindow := apply(t, NewModel("", "", "", "", En),
		ev(t, 1, protocol.EventTurnCompleted, protocol.TurnCompleted{
			Usage: &protocol.Usage{InputTokens: 34000},
		}),
	)
	if noWindow.ContextPct != 0 {
		t.Errorf("got %d%%", noWindow.ContextPct)
	}
	if strings.Contains(Render(noWindow, DefaultGeometry(100, 10)), "ctx") {
		t.Error("no window, no meter")
	}
}

// The same file reached the model under two spellings and became two rows in
// the sidebar, two line counters, and a header claiming more files were touched
// than were.
//
// Asserted through Apply rather than on the helper, because the workspace this
// depends on arrives in an event: a test that passed the workspace by hand
// would pass even if nothing ever read session.created.
func TestAFileIsCountedOnceWhicheverWayTheToolSpeltIt(t *testing.T) {
	m := NewModel("", "", "", "", En)
	m = m.Apply(ev(t, 1, protocol.EventSessionCreated, protocol.Session{
		ID: "s1", Workspace: "/w/craw", State: protocol.SessionStateIdle,
	}))
	for i, target := range []string{"DCODE.md", "/w/craw/DCODE.md", "./DCODE.md"} {
		m = m.Apply(ev(t, uint64(2+i), protocol.EventToolRequested, protocol.ToolRequested{
			ToolCallID: fmt.Sprintf("c%d", i), Name: "write",
			Input: json.RawMessage(fmt.Sprintf(`{"path":%q}`, target)),
		}))
	}

	rows := FileTree(m.Entries)
	if len(rows) != 1 {
		t.Fatalf("one file drew %d rows: %+v", len(rows), rows)
	}
	if rows[0].Path != "DCODE.md" || rows[0].Folder {
		t.Errorf("row is %+v, want the workspace-relative file", rows[0])
	}
}

// A path the workspace does not contain keeps the only spelling that finds it.
// Trimming it to a ladder of "../.." would name a file nobody can open.
func TestAPathOutsideTheWorkspaceKeepsItsFullSpelling(t *testing.T) {
	for _, c := range []struct{ target, workspace, want string }{
		{"/tmp/clickbus.html", "/w/craw", "/tmp/clickbus.html"},
		{"/w/craw/src/cli.py", "/w/craw", "src/cli.py"},
		{"/w/craw/src/cli.py", "/w/craw/", "src/cli.py"},
		{"ls -la /w/craw", "/w/craw", "ls -la /w/craw"},
		{"alpha", "/w/craw", "alpha"},
		{"/w/craw", "/w/craw", "/w/craw"},
		{"", "/w/craw", ""},
	} {
		if got := relativise(c.target, c.workspace); got != c.want {
			t.Errorf("relativise(%q, %q) = %q, want %q", c.target, c.workspace, got, c.want)
		}
	}
}
