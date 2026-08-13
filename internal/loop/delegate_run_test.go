package loop

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	ce "github.com/aguinelo/dcode/internal/contextengine"
	"github.com/aguinelo/dcode/internal/policy"
	"github.com/aguinelo/dcode/internal/protocol"
	"github.com/aguinelo/dcode/internal/provider"
	"github.com/aguinelo/dcode/internal/tools"
)

// delegateEngine builds a parent whose child will be driven by the script.
func delegateEngine(t *testing.T, turns [][]provider.StreamEvent) (*Engine, string) {
	t.Helper()
	ws := t.TempDir()
	if err := os.WriteFile(filepath.Join(ws, "pay.go"), []byte("package pay\n\nfunc Validate() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	res, err := policy.NewResolver(ws)
	if err != nil {
		t.Fatal(err)
	}
	e := New(Config{
		Provider: &scriptedProvider{turns: turns},
		Tools: tools.NewRegistry(
			tools.Read{}, tools.Write{}, tools.Glob{}, tools.Grep{}, tools.Symbol{},
		),
		State:                  tools.NewState(res, tools.DefaultLimits(), allToolNames),
		Emitter:                newRecorder(),
		Limits:                 DefaultLimits(),
		Mode:                   policy.ModeWorkspaceWrite,
		Policy:                 policy.PolicyOnRequest,
		Model:                  "m",
		Parallel:               4,
		DelegateMaxIterations:  5,
		DelegateMaxResultBytes: 8192,
	}, ce.Session{Instructions: "You are dcode."})
	return e, ws
}

// The whole point, end to end: the child reads in ITS window and the parent
// gets back a conclusion plus the paths it looked at.
func TestADelegatedTurnReturnsAConclusionAndWhereItLooked(t *testing.T) {
	e, _ := delegateEngine(t, [][]provider.StreamEvent{
		{call("c1", "read", `{"path":"pay.go"}`), done()},
		{text("Validation happens in pay.go:3."), done()},
	})

	res, err := e.Delegate(context.Background(), "where is payment validated", "", DelegateLimits{
		MaxIterations: 5, MaxResultBytes: 8192,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(res.Conclusion, "pay.go:3") {
		t.Fatalf("conclusion = %q, want the child's answer", res.Conclusion)
	}
	if len(res.Read) == 0 || !strings.Contains(strings.Join(res.Read, ","), "pay.go") {
		t.Fatalf("read = %v, want the file the child opened — the list is what turns \"trust me\" into something checkable", res.Read)
	}
	// The accounting is NOT asserted here. It used to be, as
	// `e.delegated.OutputTokens < 0` — a condition a token count cannot meet,
	// so the line could only ever pass. Making it real needs a fixture that
	// declares a cost, which belongs to the test about cost:
	// TestChildTokensAreDebitedFromTheParent.
}

// The child receives the TASK, never the parent's history. Copying the history
// back would return exactly the cost delegation exists to avoid.
func TestTheChildIsGivenTheTaskAndNotTheParentHistory(t *testing.T) {
	e, _ := delegateEngine(t, [][]provider.StreamEvent{{text("nothing found"), done()}})
	e.session.History = append(e.session.History, ce.Message{
		Role: ce.RoleUser, Text: "a long and expensive conversation the parent already had",
	})

	if _, err := e.Delegate(context.Background(), "find the thing", "sub/dir", DelegateLimits{MaxIterations: 3}); err != nil {
		t.Fatal(err)
	}
	seen := e.cfg.Provider.(*scriptedProvider).seen
	if len(seen) == 0 {
		t.Fatal("the child never reached the provider")
	}
	for _, m := range seen[0].Messages {
		if strings.Contains(m.Text, "expensive conversation") {
			t.Fatal("the parent's history was copied into the child")
		}
	}
	var joined string
	for _, m := range seen[0].Messages {
		joined += m.Text
	}
	if !strings.Contains(joined, "find the thing") || !strings.Contains(joined, "sub/dir") {
		t.Errorf("the child did not receive the task and its path:\n%s", joined)
	}
}

func TestALongReportIsCutAndSaysSo(t *testing.T) {
	long := strings.Repeat("detail. ", 400)
	e, _ := delegateEngine(t, [][]provider.StreamEvent{{text(long), done()}})

	res, err := e.Delegate(context.Background(), "explain", "", DelegateLimits{
		MaxIterations: 3, MaxResultBytes: 200,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Conclusion) != 200 {
		t.Errorf("conclusion is %d bytes, want it cut to 200", len(res.Conclusion))
	}
	if !res.Truncated || !strings.Contains(res.String(), "truncated") {
		t.Error("a cut report did not declare it; the cost of reading would return through the answer")
	}
}

// The adapter the tool actually calls.
func TestExploreAdaptsTheEngineToTheToolInterface(t *testing.T) {
	e, _ := delegateEngine(t, [][]provider.StreamEvent{
		{call("c1", "read", `{"path":"pay.go"}`), done()},
		{text("found it"), done()},
	})
	conclusion, read, unread, truncated, err := e.Explore(context.Background(), "find it", "")
	if err != nil {
		t.Fatal(err)
	}
	if conclusion != "found it" {
		t.Errorf("conclusion = %q", conclusion)
	}
	if len(read) == 0 {
		t.Error("no paths reported")
	}
	if len(unread) != 0 || truncated {
		t.Errorf("unexpected unread=%v truncated=%v", unread, truncated)
	}
}

func TestTheChildInstructionsNameOnlyTheToolsItHas(t *testing.T) {
	got := delegateInstructions([]string{"read", "grep"})
	if !strings.Contains(got, "read, grep") {
		t.Errorf("the child was not told what it has:\n%s", got)
	}
	for _, absent := range []string{"write", "edit", "bash", "explore"} {
		if strings.Contains(got, absent) {
			t.Errorf("the child was told about %q, which it does not have", absent)
		}
	}
}

func TestTaskWithoutAPathIsUnchanged(t *testing.T) {
	if got := delegateTask("find it", "   "); got != "find it" {
		t.Errorf("got %q, want the task alone", got)
	}
}

func TestLastTextIgnoresEverythingButTheFinalAnswer(t *testing.T) {
	s := ce.Session{History: []ce.Message{
		{Role: ce.RoleUser, Text: "the question"},
		{Role: ce.RoleAssistant, Text: "thinking out loud"},
		{Role: ce.RoleTool, Text: "tool noise"},
		{Role: ce.RoleAssistant, Text: "the answer"},
	}}
	if got := lastText(s); got != "the answer" {
		t.Fatalf("lastText = %q, want the last assistant message", got)
	}
	if got := lastText(ce.Session{}); got != "" {
		t.Errorf("an empty session produced %q", got)
	}
}

func TestUniqueSortedDropsRepeats(t *testing.T) {
	got := uniqueSortedStrings([]string{"b", "a", "b", "c", "a"})
	if strings.Join(got, ",") != "a,b,c" {
		t.Fatalf("got %v", got)
	}
}

func TestReportNamesByState(t *testing.T) {
	r := Report{States: map[string]CriterionState{
		"tests": CriterionMet, "lint": CriterionUnmet,
		"vet": CriterionUnmet, "build": CriterionUnavailable,
	}}
	if got := strings.Join(r.Names(CriterionUnmet), ","); got != "lint,vet" {
		t.Errorf("unmet = %q, want lint,vet sorted", got)
	}
	if got := strings.Join(r.Names(CriterionUnavailable), ","); got != "build" {
		t.Errorf("unavailable = %q", got)
	}
}

// The real wait, exercised once so the branch is not only reachable through the
// injected one.
func TestTheRealSleepReturnsAndIsInterruptible(t *testing.T) {
	e := New(Config{}, ce.Session{Instructions: "x"})
	if !e.sleep(context.Background(), 0) {
		t.Error("a zero wait did not complete")
	}
	if !e.sleep(context.Background(), time.Millisecond) {
		t.Error("a short wait did not complete")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if e.sleep(ctx, time.Hour) {
		t.Error("a cancelled context waited an hour")
	}
}

// The child's read-only mode is the structural half of RN-11, and the test that
// stood for it asserted a string constant's spelling. That is not the claim: a
// child built with the parent's mode would pass that too.
//
// Two locks stand between a delegated turn and a write: its registry has no
// writing tool, and its mode refuses one. Defence in depth is only real if each
// layer is verified ALONE, so this deliberately defeats the outer lock — the
// writing tool is allowed into the child's registry for the duration — and
// asserts the inner one still holds.
func TestTheChildIsReadOnlyEvenWithAWritingToolInReach(t *testing.T) {
	readOnlyTools["scribble"] = true
	t.Cleanup(func() { delete(readOnlyTools, "scribble") })

	ws := t.TempDir()
	res, err := policy.NewResolver(ws)
	if err != nil {
		t.Fatal(err)
	}
	spy := &countingApprover{}
	e := New(Config{
		Provider: &scriptedProvider{turns: [][]provider.StreamEvent{
			{call("c1", "scribble", `{}`), done()},
			{text("could not write"), done()},
		}},
		Tools:                  tools.NewRegistry(tools.Read{}, scribble{dir: ws}),
		State:                  tools.NewState(res, tools.DefaultLimits(), allToolNames),
		Emitter:                newRecorder(),
		Approver:               spy,
		Limits:                 DefaultLimits(),
		Mode:                   policy.ModeWorkspaceWrite, // the PARENT may write
		Policy:                 policy.PolicyOnRequest,
		Model:                  "m",
		Parallel:               4,
		DelegateMaxIterations:  5,
		DelegateMaxResultBytes: 8192,
	}, ce.Session{Instructions: "You are dcode."})

	if _, err := e.Delegate(context.Background(), "look around", "", DelegateLimits{MaxIterations: 3}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(ws, "scribbled.txt")); err == nil {
		t.Error("the child wrote to the workspace; its mode is read-only and no input can produce another")
	}
	// And it was refused rather than escalated. Consent given to the parent is
	// not consent for the child, and a user who asked one question must not be
	// interrupted by N they did not ask.
	if spy.asked != 0 {
		t.Errorf("the delegated turn asked for approval %d times; a read-only child has nothing to ask about", spy.asked)
	}
}

// scribble genuinely writes when it runs, which is what makes the assertion
// above about the mode rather than about a tool that never touched the disk.
type scribble struct{ dir string }

func (scribble) Name() string            { return "scribble" }
func (scribble) Description() string     { return "writes a file" }
func (scribble) Schema() json.RawMessage { return json.RawMessage(`{"type":"object"}`) }

func (s scribble) Declare(json.RawMessage) (policy.Request, error) {
	return policy.Request{Tool: "scribble", Paths: []policy.Access{{Path: "scribbled.txt", Write: true}}}, nil
}

func (s scribble) Execute(context.Context, json.RawMessage, *tools.State) (tools.Result, error) {
	if err := os.WriteFile(filepath.Join(s.dir, "scribbled.txt"), []byte("x"), 0o644); err != nil {
		return tools.Result{}, err
	}
	return tools.Result{Output: "wrote it"}, nil
}

// countingApprover fails the test by being used at all.
type countingApprover struct{ asked int }

func (a *countingApprover) Approve(context.Context, protocol.ApprovalRequest) (protocol.ApprovalDecision, error) {
	a.asked++
	return protocol.ApprovalAllow, nil
}
