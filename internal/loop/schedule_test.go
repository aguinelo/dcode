package loop

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	ce "github.com/aguinelo/dcode/internal/contextengine"
	"github.com/aguinelo/dcode/internal/policy"
	"github.com/aguinelo/dcode/internal/protocol"
	"github.com/aguinelo/dcode/internal/provider"
	"github.com/aguinelo/dcode/internal/tools"
)

func exec(idx int, tool string, paths ...policy.Access) ToolExecution {
	return ToolExecution{
		Index:   idx,
		Call:    ce.ToolCall{ID: "c", Name: tool},
		Declare: policy.Request{Tool: tool, Paths: paths},
	}
}

func rd(p string) policy.Access { return policy.Access{Path: p} }
func wr(p string) policy.Access { return policy.Access{Path: p, Write: true} }

// Reads of different paths are exactly where parallelism is both safe and worth
// having, so they must end up together.
func TestIndependentReadsShareAGroup(t *testing.T) {
	groups := Schedule([]ToolExecution{
		exec(0, "read", rd("a.go")),
		exec(1, "read", rd("b.go")),
		exec(2, "read", rd("c.go")),
	}, 4, nil)
	if len(groups) != 1 || len(groups[0]) != 3 {
		t.Fatalf("want one group of three, got %v", shape(groups))
	}
}

func TestConflictingPathsAreSeparated(t *testing.T) {
	for _, tc := range []struct {
		name  string
		execs []ToolExecution
	}{
		{"two writes to the same path", []ToolExecution{exec(0, "edit", wr("a.go")), exec(1, "edit", wr("a.go"))}},
		{"read and write of the same path", []ToolExecution{exec(0, "read", rd("a.go")), exec(1, "edit", wr("a.go"))}},
		{"write then read of the same path", []ToolExecution{exec(0, "edit", wr("a.go")), exec(1, "read", rd("a.go"))}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			groups := Schedule(tc.execs, 4, nil)
			if len(groups) != 2 {
				t.Errorf("racing calls must not share a group, got %v", shape(groups))
			}
		})
	}
}

func TestWritesToDifferentPathsCanShareAGroup(t *testing.T) {
	groups := Schedule([]ToolExecution{
		exec(0, "edit", wr("a.go")),
		exec(1, "edit", wr("b.go")),
	}, 4, nil)
	if len(groups) != 1 {
		t.Errorf("independent writes are safe together, got %v", shape(groups))
	}
}

// A shell command is opaque: it could touch anything, so nothing can be proven
// safe alongside it.
func TestSystemCommandsAlwaysRunAlone(t *testing.T) {
	sys := ToolExecution{
		Index:   1,
		Call:    ce.ToolCall{Name: "bash"},
		Declare: policy.Request{Tool: "bash", Command: "rm -rf build", Network: true},
	}
	groups := Schedule([]ToolExecution{
		exec(0, "read", rd("a.go")),
		sys,
		exec(2, "read", rd("b.go")),
	}, 4, nil)

	for _, g := range groups {
		if len(g) > 1 {
			for _, e := range g {
				if e.Declare.Command != "" {
					t.Fatalf("a system command shared a group: %v", shape(groups))
				}
			}
		}
	}
	if len(groups) != 3 {
		t.Errorf("want the command isolated between the reads, got %v", shape(groups))
	}
}

func TestTwoSystemCommandsNeverOverlap(t *testing.T) {
	mk := func(i int) ToolExecution {
		return ToolExecution{Index: i, Call: ce.ToolCall{Name: "bash"},
			Declare: policy.Request{Tool: "bash", Command: "echo"}}
	}
	groups := Schedule([]ToolExecution{mk(0), mk(1)}, 4, nil)
	if len(groups) != 2 {
		t.Errorf("got %v", shape(groups))
	}
}

// A declaration failure says nothing about what the call would touch, so
// nothing can be scheduled alongside it.
func TestADeclarationFailureRunsAlone(t *testing.T) {
	bad := ToolExecution{Index: 1, Call: ce.ToolCall{Name: "x"}, Err: context.Canceled}
	groups := Schedule([]ToolExecution{exec(0, "read", rd("a")), bad, exec(2, "read", rd("b"))}, 4, nil)
	if len(groups) != 3 {
		t.Errorf("got %v", shape(groups))
	}
}

func TestParallelismLimitIsRespected(t *testing.T) {
	var execs []ToolExecution
	for i := 0; i < 10; i++ {
		execs = append(execs, exec(i, "read", rd(string(rune('a'+i))+".go")))
	}
	groups := Schedule(execs, 3, nil)
	for _, g := range groups {
		if len(g) > 3 {
			t.Errorf("group of %d exceeds the limit of 3", len(g))
		}
	}
	if len(groups) != 4 {
		t.Errorf("want 4 groups for 10 calls at 3 wide, got %d", len(groups))
	}
}

func TestZeroParallelismSerialises(t *testing.T) {
	groups := Schedule([]ToolExecution{
		exec(0, "read", rd("a")), exec(1, "read", rd("b")),
	}, 0, nil)
	if len(groups) != 2 {
		t.Errorf("a limit of zero must mean one at a time, got %v", shape(groups))
	}
}

func TestScheduleIsDeterministic(t *testing.T) {
	execs := []ToolExecution{
		exec(0, "read", rd("a")), exec(1, "edit", wr("a")),
		exec(2, "read", rd("b")), exec(3, "read", rd("c")),
	}
	first := shape(Schedule(execs, 4, nil))
	for i := 0; i < 50; i++ {
		if got := shape(Schedule(execs, 4, nil)); !equal(got, first) {
			t.Fatalf("run %d differs: %v vs %v", i, got, first)
		}
	}
}

func TestScheduleKeepsEmissionOrder(t *testing.T) {
	execs := []ToolExecution{
		exec(0, "read", rd("a")), exec(1, "edit", wr("a")), exec(2, "read", rd("b")),
	}
	var seen []int
	for _, g := range Schedule(execs, 4, nil) {
		for _, e := range g {
			seen = append(seen, e.Index)
		}
	}
	for i, idx := range seen {
		if idx != i {
			t.Fatalf("scheduling reordered calls: %v", seen)
		}
	}
}

func TestEmptyScheduleIsEmpty(t *testing.T) {
	if got := Schedule(nil, 4, nil); len(got) != 0 {
		t.Errorf("got %v", shape(got))
	}
}

func shape(groups []Group) []int {
	out := make([]int, len(groups))
	for i, g := range groups {
		out[i] = len(g)
	}
	return out
}

func equal(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// ---------- repeat detection ----------

func TestIsRepeatIgnoresKeyOrder(t *testing.T) {
	a := ce.ToolCall{Name: "read", Input: json.RawMessage(`{"path":"a.go","limit":10}`)}
	b := ce.ToolCall{Name: "read", Input: json.RawMessage(`{"limit":10,"path":"a.go"}`)}
	c := ce.ToolCall{Name: "read", Input: json.RawMessage(`{"path":"a.go", "limit": 10}`)}

	// Without canonicalisation a model that reorders keys would loop forever
	// while the detector saw three different calls.
	if !IsRepeat([]ce.ToolCall{a, b, c}, 3) {
		t.Error("reordered keys and whitespace are the same call")
	}
}

func TestIsRepeatDistinguishesDifferentInputs(t *testing.T) {
	a := ce.ToolCall{Name: "read", Input: json.RawMessage(`{"path":"a.go"}`)}
	b := ce.ToolCall{Name: "read", Input: json.RawMessage(`{"path":"b.go"}`)}
	if IsRepeat([]ce.ToolCall{a, a, b}, 3) {
		t.Error("a different path is a different call")
	}
}

func TestIsRepeatDistinguishesDifferentTools(t *testing.T) {
	a := ce.ToolCall{Name: "read", Input: json.RawMessage(`{}`)}
	b := ce.ToolCall{Name: "glob", Input: json.RawMessage(`{}`)}
	if IsRepeat([]ce.ToolCall{a, a, b}, 3) {
		t.Error("a different tool is a different call")
	}
}

func TestIsRepeatHandlesEdgeCases(t *testing.T) {
	a := ce.ToolCall{Name: "read", Input: json.RawMessage(`{}`)}
	if IsRepeat([]ce.ToolCall{a, a}, 3) {
		t.Error("fewer calls than the threshold is not a repeat")
	}
	if IsRepeat([]ce.ToolCall{a, a, a}, 0) {
		t.Error("a threshold of zero disables the detector")
	}
	if IsRepeat(nil, 3) {
		t.Error("no calls is not a repeat")
	}
}

func TestIsRepeatWithUnparseableInput(t *testing.T) {
	// Refusing to fingerprint invalid JSON would disable the detector exactly
	// when the model is producing garbage, which is when it is needed most.
	a := ce.ToolCall{Name: "read", Input: json.RawMessage(`{not json`)}
	if !IsRepeat([]ce.ToolCall{a, a, a}, 3) {
		t.Error("identical unparseable inputs are still identical")
	}
}

func TestIsRepeatWithNestedStructures(t *testing.T) {
	a := ce.ToolCall{Name: "plan", Input: json.RawMessage(
		`{"items":[{"id":1,"status":"active"},{"id":2,"status":"pending"}]}`)}
	b := ce.ToolCall{Name: "plan", Input: json.RawMessage(
		`{"items":[{"status":"active","id":1},{"status":"pending","id":2}]}`)}
	if !IsRepeat([]ce.ToolCall{a, b, a}, 3) {
		t.Error("nested objects must canonicalise too")
	}
}

func TestIsRepeatWithEmptyInput(t *testing.T) {
	a := ce.ToolCall{Name: "plan"}
	b := ce.ToolCall{Name: "plan", Input: json.RawMessage(`{}`)}
	if !IsRepeat([]ce.ToolCall{a, b, a}, 3) {
		t.Error("absent and empty arguments are the same call")
	}
}

// ---------- compaction ----------

func TestCompactionRunsOnceAndIsAnnounced(t *testing.T) {
	reg := tools.NewRegistry(tools.Read{})
	p := &scriptedProvider{turns: [][]provider.StreamEvent{
		{call("c1", "read", `{"path":"a.go"}`), done()},
		{text("ok"), done()},
	}}

	long := ce.Session{Instructions: "You are dcode."}
	for i := 0; i < 40; i++ {
		long.History = append(long.History,
			ce.Message{Role: ce.RoleUser, Text: strings.Repeat("q ", 40)},
			ce.Message{Role: ce.RoleAssistant, Text: strings.Repeat("a ", 40)},
		)
	}

	ws := t.TempDir()
	res, err := policy.NewResolver(ws)
	if err != nil {
		t.Fatal(err)
	}
	rec := newRecorder()
	summarised := 0
	e := New(Config{
		Provider: p, Tools: reg, State: tools.NewState(res, tools.DefaultLimits()),
		Emitter: rec, Limits: DefaultLimits(), Mode: policy.ModeWorkspaceWrite,
		Policy: policy.PolicyOnRequest, Model: "m",
		CtxConfig: ce.Config{CompactAt: 0.5, KeepTurns: 2, Window: 500},
		Summarise: func(context.Context, []ce.Message) (string, error) {
			summarised++
			return "earlier: we looked at some files", nil
		},
	}, long)

	if _, err := e.Run(context.Background(), "continue"); err != nil {
		t.Fatal(err)
	}
	if rec.count(protocol.EventSessionCompacted) == 0 {
		t.Fatal("compaction should have been announced")
	}
	if summarised == 0 {
		t.Error("the caller's summariser should have been used")
	}
	if e.Session().Summary == nil {
		t.Error("the session should carry a summary afterwards")
	}
	// Once per iteration, never more: a second check point is how compaction
	// turns incremental and starts costing the cache every turn.
	if got := rec.count(protocol.EventSessionCompacted); got > 3 {
		t.Errorf("compaction ran %d times in a short turn", got)
	}
}

func TestCompactionFallsBackWhenSummarisingFails(t *testing.T) {
	reg := tools.NewRegistry()
	p := &scriptedProvider{turns: [][]provider.StreamEvent{{text("ok"), done()}}}

	long := ce.Session{Instructions: "You are dcode."}
	for i := 0; i < 40; i++ {
		long.History = append(long.History,
			ce.Message{Role: ce.RoleUser, Text: strings.Repeat("q ", 40)},
			ce.Message{Role: ce.RoleAssistant, Text: strings.Repeat("a ", 40)},
		)
	}

	ws := t.TempDir()
	res, _ := policy.NewResolver(ws)
	e := New(Config{
		Provider: p, Tools: reg, State: tools.NewState(res, tools.DefaultLimits()),
		Emitter: newRecorder(), Limits: DefaultLimits(), Mode: policy.ModeWorkspaceWrite,
		Policy: policy.PolicyOnRequest, Model: "m",
		CtxConfig: ce.Config{CompactAt: 0.5, KeepTurns: 2, Window: 500},
		Summarise: func(context.Context, []ce.Message) (string, error) {
			return "", context.Canceled
		},
	}, long)

	if _, err := e.Run(context.Background(), "continue"); err != nil {
		t.Fatal(err)
	}
	// A failed summary must not abort the turn: losing detail beats losing the
	// session.
	if e.Session().Summary == nil {
		t.Error("compaction should still have happened with placeholder text")
	}
}

func TestSortedPlanIsStable(t *testing.T) {
	in := []protocol.PlanItem{{ID: 3}, {ID: 1}, {ID: 2}}
	got := SortedPlan(in)
	for i, it := range got {
		if it.ID != i+1 {
			t.Fatalf("got %+v", got)
		}
	}
	// The input must not be reordered under the caller.
	if in[0].ID != 3 {
		t.Error("SortedPlan mutated its argument")
	}
}

// The fourth row of table 4.2, and the only one that was never implemented: a
// call needing approval runs alone, because a user's decision is sequential by
// nature.
//
// Two escalating calls in one group means two questions asked at once about
// work already in flight. The client shows one; the other waits behind it,
// invisible, having already been started. Whichever the user answers, they
// answered it without being shown the other — and "allow" on a batch nobody
// described is not consent.
func TestACallNeedingApprovalRunsAlone(t *testing.T) {
	escalating := map[string]bool{"reach": true}
	needsApproval := func(e ToolExecution) bool { return escalating[e.Declare.Tool] }

	groups := Schedule([]ToolExecution{
		exec(0, "read", rd("a")),
		exec(1, "read", rd("b")),
		exec(2, "reach", rd("/etc/passwd")),
		exec(3, "reach", rd("/etc/hosts")),
		exec(4, "read", rd("c")),
	}, 4, needsApproval)

	for _, g := range groups {
		asking := 0
		for _, e := range g {
			if needsApproval(e) {
				asking++
			}
		}
		if asking > 0 && len(g) > 1 {
			t.Errorf("a call needing approval shares a group with %d others; the user is "+
				"asked about work already running beside it", len(g)-1)
		}
		if asking > 1 {
			t.Errorf("%d approvals in one group; the questions would be asked at once", asking)
		}
	}

	// And the reads either side are still allowed to be parallel: isolating the
	// question must not serialise the whole batch, or approval would quietly
	// become a performance setting.
	var widest int
	for _, g := range groups {
		if len(g) > widest {
			widest = len(g)
		}
	}
	if widest < 2 {
		t.Error("nothing ran in parallel; isolating the approval serialised the batch")
	}
}

// Without a predicate the scheduler behaves exactly as before. The engine is
// the only thing that can evaluate a verdict, and a Schedule that demanded one
// would put policy inside a function whose whole job is ordering.
func TestSchedulingWithoutAPredicateIsUnchanged(t *testing.T) {
	execs := []ToolExecution{exec(0, "read", rd("a")), exec(1, "read", rd("b"))}
	if got := Schedule(execs, 4, nil); len(got) != 1 || len(got[0]) != 2 {
		t.Fatalf("nil predicate changed the grouping: %v", shape(got))
	}
}
