package tools

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/aguinelo/dcode/internal/protocol"
)

func planItems(items ...protocol.PlanItem) PlanInput { return PlanInput{Items: items} }

func item(id int, status string, blocked ...string) protocol.PlanItem {
	it := protocol.PlanItem{ID: id, Text: "step", Status: status}
	if len(blocked) > 0 {
		it.Blocked = blocked[0]
	}
	return it
}

func TestPlanAcceptsAValidPlan(t *testing.T) {
	s, _ := setup(t)
	res := run(t, Plan{}, s, planItems(
		item(1, protocol.PlanDone),
		item(2, protocol.PlanActive),
		item(3, protocol.PlanPending),
	))
	if res.IsError {
		t.Fatalf("%+v", res)
	}
	if got := s.Plan(); len(got) != 3 {
		t.Fatalf("want 3 items, got %d", len(got))
	}
	if !strings.Contains(res.Output, "1 done") {
		t.Errorf("the summary should report progress: %q", res.Output)
	}
}

// A rejected plan must leave the previous one exactly as it was, or a malformed
// call would wipe the panel the user is reading.
func TestRejectedPlanLeavesThePreviousOneIntact(t *testing.T) {
	s, _ := setup(t)
	run(t, Plan{}, s, planItems(item(1, protocol.PlanActive)))
	before := s.Plan()

	for _, tc := range []struct {
		name string
		in   PlanInput
		code string
	}{
		{"two active", planItems(item(1, protocol.PlanActive), item(2, protocol.PlanActive)), CodePlanMultiActive},
		{"blocked with no reason", planItems(item(1, protocol.PlanBlocked)), CodePlanNoReason},
		{"unknown status", planItems(item(1, "maybe")), CodeBadInput},
	} {
		t.Run(tc.name, func(t *testing.T) {
			res := run(t, Plan{}, s, tc.in)
			if !res.IsError || res.Code != tc.code {
				t.Fatalf("want %s, got %+v", tc.code, res)
			}
			got := s.Plan()
			if len(got) != len(before) || got[0].Status != before[0].Status {
				t.Errorf("the plan changed despite the refusal: %+v", got)
			}
		})
	}
}

// A blocked item with no reason shows the user a stop with nothing to act on,
// which is worse than no block at all.
func TestPlanAcceptsBlockedWithAReason(t *testing.T) {
	s, _ := setup(t)
	res := run(t, Plan{}, s, planItems(item(1, protocol.PlanBlocked, "missing dependency")))
	if res.IsError {
		t.Fatalf("%+v", res)
	}
	if !strings.Contains(res.Output, "1 blocked") {
		t.Errorf("blocked count should be visible: %q", res.Output)
	}
}

func TestPlanReplacesWholesale(t *testing.T) {
	s, _ := setup(t)
	run(t, Plan{}, s, planItems(item(1, protocol.PlanDone), item(2, protocol.PlanPending)))
	run(t, Plan{}, s, planItems(item(1, protocol.PlanActive)))
	if got := s.Plan(); len(got) != 1 || got[0].Status != protocol.PlanActive {
		t.Errorf("a call replaces the plan entirely, got %+v", got)
	}
}

func TestPlanReturnsACopy(t *testing.T) {
	s, _ := setup(t)
	run(t, Plan{}, s, planItems(item(1, protocol.PlanPending)))
	got := s.Plan()
	got[0].Status = protocol.PlanDone
	// Handing out the live slice would let a client mutate session state.
	if s.Plan()[0].Status != protocol.PlanPending {
		t.Error("Plan() must return a copy")
	}
}

// ---------- bash ----------

type fakeRunner struct {
	out  string
	code int
	err  error
	wait time.Duration
	seen []string
}

func (f *fakeRunner) Run(ctx context.Context, workdir, command string) (string, int, error) {
	f.seen = append(f.seen, command)
	if f.wait > 0 {
		select {
		case <-time.After(f.wait):
		case <-ctx.Done():
			return f.out, -1, ctx.Err()
		}
	}
	return f.out, f.code, f.err
}

// A non-zero exit is data the model needs to decide. Turning it into a tool
// error hides exactly the signal that drives recovery.
func TestBashNonZeroExitIsAResultNotAnError(t *testing.T) {
	s, _ := setup(t)
	r := &fakeRunner{out: "FAIL\tpkg 0.1s\n", code: 1}
	res := run(t, Bash{Runner: r}, s, BashInput{Command: "go test ./..."})

	if res.IsError {
		t.Fatalf("a failing command is a result, not a tool error: %+v", res)
	}
	if !strings.Contains(res.Output, "exit 1") {
		t.Errorf("the exit code must be visible: %q", res.Output)
	}
	if !strings.Contains(res.Output, "FAIL") {
		t.Errorf("the output must survive: %q", res.Output)
	}
}

func TestBashReportsEmptyOutputPlainly(t *testing.T) {
	s, _ := setup(t)
	res := run(t, Bash{Runner: &fakeRunner{out: "", code: 0}}, s, BashInput{Command: "true"})
	if !strings.Contains(res.Output, "no output") {
		t.Errorf("silence should be stated, not left blank: %q", res.Output)
	}
}

func TestBashTimeoutKeepsPartialOutput(t *testing.T) {
	s, _ := setup(t)
	r := &fakeRunner{out: "started…", wait: time.Second, err: context.DeadlineExceeded}
	res := run(t, Bash{Runner: r, Timeout: 20 * time.Millisecond}, s,
		BashInput{Command: "sleep 10"})

	if res.Code != CodeTimeout {
		t.Fatalf("want timeout, got %+v", res)
	}
	// Whatever ran before the cut still helps the model decide what to do.
	if !strings.Contains(res.Output, "started") {
		t.Errorf("partial output should survive a timeout: %q", res.Output)
	}
	if !res.Truncated {
		t.Error("a timeout is a truncation and must say so")
	}
}

func TestBashWithoutASandboxRefusesToRun(t *testing.T) {
	s, _ := setup(t)
	// Failing closed: no boundary means no execution, rather than quietly
	// running outside one.
	res := run(t, Bash{}, s, BashInput{Command: "echo hi"})
	if !res.IsError || res.Code != CodeDenied {
		t.Fatalf("want denied, got %+v", res)
	}
}

func TestBashRejectsAnEmptyCommand(t *testing.T) {
	s, _ := setup(t)
	res := run(t, Bash{Runner: &fakeRunner{}}, s, BashInput{Command: "   "})
	if !res.IsError || res.Code != CodeBadInput {
		t.Fatalf("want bad_input, got %+v", res)
	}
}

func TestBashSurfacesARunnerFailure(t *testing.T) {
	s, _ := setup(t)
	r := &fakeRunner{code: -1, err: errors.New("sandbox unavailable")}
	res := run(t, Bash{Runner: r}, s, BashInput{Command: "echo hi"})
	if !res.IsError {
		t.Fatalf("a runner failure is not a normal result: %+v", res)
	}
}

func TestBashClampsHugeOutput(t *testing.T) {
	s, _ := setup(t)
	s.Limits.MaxToolOutput = 100
	r := &fakeRunner{out: strings.Repeat("x", 10_000), code: 0}
	res := run(t, Bash{Runner: r}, s, BashInput{Command: "cat big"})
	if len(res.Output) > 400 {
		t.Errorf("untrusted output must be clamped, got %d bytes", len(res.Output))
	}
	if !strings.Contains(res.Output, "truncated") {
		t.Errorf("the clamp must be declared: %q", res.Output)
	}
}

func TestLocalRunnerRunsAndReportsExit(t *testing.T) {
	var r LocalRunner
	out, code, err := r.Run(context.Background(), t.TempDir(), "echo hello")
	if err != nil || code != 0 || !strings.Contains(out, "hello") {
		t.Fatalf("out=%q code=%d err=%v", out, code, err)
	}
	_, code, err = r.Run(context.Background(), t.TempDir(), "exit 3")
	if err != nil || code != 3 {
		t.Errorf("a non-zero exit should be reported, not raised: code=%d err=%v", code, err)
	}
}

// ---------- registry ----------

// Tool definitions land in the cached prefix, so their order must be identical
// across runs; an unstable one would invalidate every session's cache on every
// restart.
func TestRegistryDefsAreStableAndSorted(t *testing.T) {
	r := NewRegistry(Bash{}, Read{}, Plan{}, Edit{}, Glob{}, Grep{}, Write{})

	first, err := json.Marshal(r.Defs())
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 20; i++ {
		again, err := json.Marshal(NewRegistry(Grep{}, Write{}, Read{}, Plan{}, Bash{}, Edit{}, Glob{}).Defs())
		if err != nil {
			t.Fatal(err)
		}
		if string(again) != string(first) {
			t.Fatalf("definitions differ by registration order on run %d", i)
		}
	}

	names := r.Names()
	if len(names) != 7 {
		t.Fatalf("want 7 tools, got %d: %v", len(names), names)
	}
	if names[0] != "bash" {
		t.Errorf("names must be sorted, got %v", names)
	}
}

func TestRegistryGet(t *testing.T) {
	r := NewRegistry(Read{}, Plan{})
	if _, ok := r.Get("read"); !ok {
		t.Error("read should be registered")
	}
	if _, ok := r.Get("nope"); ok {
		t.Error("an unregistered tool must not resolve")
	}
}

func TestEverySchemaIsValidJSON(t *testing.T) {
	for _, tool := range []Tool{Read{}, Write{}, Edit{}, Glob{}, Grep{}, Bash{}, Plan{}} {
		if !json.Valid(tool.Schema()) {
			t.Errorf("%s: schema is not valid JSON", tool.Name())
		}
		if tool.Description() == "" {
			t.Errorf("%s: a tool with no description is a tool the model will misuse", tool.Name())
		}
	}
}

func TestMalformedArgumentsAreRecoverable(t *testing.T) {
	s, _ := setup(t)
	for _, tool := range []Tool{Read{}, Write{}, Edit{}, Bash{}} {
		res, err := tool.Execute(context.Background(), json.RawMessage(`{"path":42}`), s)
		if err != nil {
			t.Errorf("%s: bad arguments must not be a hard error: %v", tool.Name(), err)
		}
		if !res.IsError {
			t.Errorf("%s: bad arguments should be reported to the model", tool.Name())
		}
	}
}

func TestToolErrorRendersReasonDetailAndHint(t *testing.T) {
	e := &ToolError{
		Tool: "edit", Code: CodeNoMatch,
		Reason: "old_string was not found", Detail: "main.go", Hint: "Read the file again.",
	}
	res := e.Result()
	for _, want := range []string{"not found", "main.go", "Read the file again"} {
		if !strings.Contains(res.Output, want) {
			t.Errorf("the model needs %q in the message: %q", want, res.Output)
		}
	}
	if e.Error() != "old_string was not found" {
		t.Errorf("got %q", e.Error())
	}
}

func TestCheckEditableCanBeDisabled(t *testing.T) {
	s, ws := setup(t)
	s.Limits.RequireReadBefore = false
	writeFileT(t, ws, "a.go", "x\n")
	// Documented as strongly discouraged, but it must actually work when set,
	// or the escape hatch is a lie.
	res := run(t, Edit{}, s, EditInput{Path: "a.go", OldString: "x", NewString: "y"})
	if res.IsError {
		t.Fatalf("%+v", res)
	}
}

func TestNonAtomicWriteStillWrites(t *testing.T) {
	s, ws := setup(t)
	s.Limits.AtomicWrite = false
	res := run(t, Write{}, s, WriteInput{Path: "a.go", Content: "hello\n"})
	if res.IsError {
		t.Fatalf("%+v", res)
	}
	if got := string(mustRead(t, ws+"/a.go")); got != "hello\n" {
		t.Errorf("got %q", got)
	}
}

func TestEmptyPathIsRejected(t *testing.T) {
	s, _ := setup(t)
	for _, tool := range []Tool{Read{}, Write{}, Edit{}} {
		res, err := tool.Execute(context.Background(), json.RawMessage(`{"path":"  "}`), s)
		if err != nil {
			t.Fatal(err)
		}
		if !res.IsError {
			t.Errorf("%s: an empty path must be rejected", tool.Name())
		}
	}
}
