package loop

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	ce "github.com/aguinelo/dcode/internal/contextengine"
	"github.com/aguinelo/dcode/internal/policy"
	"github.com/aguinelo/dcode/internal/protocol"
	"github.com/aguinelo/dcode/internal/provider"
	"github.com/aguinelo/dcode/internal/tools"
)

// failingProvider rejects the call outright rather than through the stream, so
// the error travels a different path than a mid-stream failure.
type failingProvider struct {
	err   error
	calls int
}

func (f *failingProvider) Family() provider.Family       { return nil }
func (f *failingProvider) Transport() provider.Transport { return nil }
func (f *failingProvider) Window(string) (int, error)    { return 1000, nil }
func (f *failingProvider) Limits() provider.Limits       { return provider.Limits{MaxIterations: 5} }

func (f *failingProvider) Stream(context.Context, provider.Request) (<-chan provider.StreamEvent, error) {
	f.calls++
	return nil, f.err
}

// An error returned before the stream opens must still be classified, or the
// loop cannot decide between retrying, compacting and aborting.
func TestStreamSetupFailureIsClassified(t *testing.T) {
	for _, tc := range []struct {
		name string
		err  error
		want string
	}{
		{
			"already classified",
			&provider.ProviderError{Class: provider.ErrClassAuth, Message: "bad key"},
			protocol.StopError,
		},
		{
			"wrapped classified",
			fmt.Errorf("dialing: %w", &provider.ProviderError{Class: provider.ErrClassQuota}),
			protocol.StopError,
		},
		{
			"plain error becomes a transport failure",
			errors.New("connection refused"),
			protocol.StopError,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ws := t.TempDir()
			res, err := policy.NewResolver(ws)
			if err != nil {
				t.Fatal(err)
			}
			e := New(Config{
				Provider: &failingProvider{err: tc.err},
				Tools:    tools.NewRegistry(),
				State:    tools.NewState(res, tools.DefaultLimits(), allToolNames),
				Emitter:  newRecorder(),
				Limits:   DefaultLimits(),
				Mode:     policy.ModeWorkspaceWrite,
				Policy:   policy.PolicyOnRequest,
				Model:    "m",
				// Instant, so a test of classification does not spend the real
				// backoff proving it.
				Sleep: func(context.Context, time.Duration) bool { return true },
			}, ce.Session{Instructions: "You are dcode."})

			out, runErr := e.Run(context.Background(), "go")
			if runErr == nil {
				t.Fatal("the failure must surface")
			}
			if out.Reason != tc.want {
				t.Errorf("got %q want %q", out.Reason, tc.want)
			}
		})
	}
}

// The provider saying the context is too large is the one case where the local
// estimate was wrong. Compacting and retrying beats failing a turn the user
// could still have completed.
func TestContextSizeErrorTriggersCompactionAndRetries(t *testing.T) {
	tooBig := provider.StreamEvent{Type: provider.EventError, Err: &provider.ProviderError{
		Class: provider.ErrClassContextSize, Message: "too many tokens", Retryable: true,
	}}
	p := &scriptedProvider{turns: [][]provider.StreamEvent{
		{tooBig},
		{text("that fits now"), done()},
	}}

	long := ce.Session{Instructions: "You are dcode."}
	for i := 0; i < 30; i++ {
		long.History = append(long.History,
			ce.Message{Role: ce.RoleUser, Text: strings.Repeat("q ", 30)},
			ce.Message{Role: ce.RoleAssistant, Text: strings.Repeat("a ", 30)},
		)
	}

	ws := t.TempDir()
	res, _ := policy.NewResolver(ws)
	rec := newRecorder()
	e := New(Config{
		Provider: p, Tools: tools.NewRegistry(),
		State: tools.NewState(res, tools.DefaultLimits(), allToolNames), Emitter: rec,
		Limits: DefaultLimits(), Mode: policy.ModeWorkspaceWrite,
		Policy: policy.PolicyOnRequest, Model: "m",
		// A window large enough that the local estimate would not have
		// compacted on its own — the provider is the only signal here.
		CtxConfig: ce.Config{Window: 10_000_000},
	}, long)

	out, err := e.Run(context.Background(), "continue")
	if err != nil {
		t.Fatalf("the loop should have recovered: %v", err)
	}
	if out.Reason != protocol.StopDone {
		t.Errorf("got %q want done", out.Reason)
	}
	if rec.count(protocol.EventSessionCompacted) == 0 {
		t.Error("the recovery path should have compacted")
	}
	if p.call < 2 {
		t.Error("the call should have been retried after compacting")
	}
}

// When there is nothing left to compact, the failure has to surface rather than
// loop forever asking the model to try again.
func TestContextSizeErrorWithNothingToCompactFails(t *testing.T) {
	tooBig := provider.StreamEvent{Type: provider.EventError, Err: &provider.ProviderError{
		Class: provider.ErrClassContextSize, Message: "too many tokens",
	}}
	p := &scriptedProvider{turns: [][]provider.StreamEvent{{tooBig}}}

	ws := t.TempDir()
	res, _ := policy.NewResolver(ws)
	e := New(Config{
		Provider: p, Tools: tools.NewRegistry(),
		State: tools.NewState(res, tools.DefaultLimits(), allToolNames), Emitter: newRecorder(),
		Limits: DefaultLimits(), Mode: policy.ModeWorkspaceWrite,
		Policy: policy.PolicyOnRequest, Model: "m",
	}, ce.Session{Instructions: "You are dcode."}) // empty history

	out, err := e.Run(context.Background(), "go")
	if err == nil {
		t.Fatal("with nothing to compact the error must surface")
	}
	if out.Reason != protocol.StopError {
		t.Errorf("got %q", out.Reason)
	}
}

func TestFamilyIterationsWithNoProvider(t *testing.T) {
	if got := familyIterations(nil); got != 0 {
		t.Errorf("got %d", got)
	}
	// And the fallback keeps the loop bounded even then.
	l := Limits{}.withFamily(0)
	if l.MaxIterations <= 0 {
		t.Errorf("a turn must always be bounded, got %d", l.MaxIterations)
	}
}

func TestNegativeIdenticalCallsClampsToZero(t *testing.T) {
	if got := (Limits{MaxIdenticalCalls: -5}).withFamily(10); got.MaxIdenticalCalls != 0 {
		t.Errorf("got %d", got.MaxIdenticalCalls)
	}
}

// A path the resolver cannot make sense of must be treated as outside the
// workspace. The safe reading of "cannot tell" is never "allow".
func TestUnresolvablePathIsTreatedAsOutside(t *testing.T) {
	reg := tools.NewRegistry(slowTool{name: "weird", path: "", write: true})
	p := &scriptedProvider{turns: [][]provider.StreamEvent{
		{call("c1", "weird", `{}`), done()},
		{text("ok"), done()},
	}}
	e, rec := newEngine(t, p, reg) // no approver, so a crossing is denied

	if _, err := e.Run(context.Background(), "go"); err != nil {
		t.Fatal(err)
	}
	var refused bool
	for _, m := range e.Session().History {
		if m.ToolResult != nil && m.ToolResult.IsError {
			refused = true
		}
	}
	if !refused && rec.count(protocol.EventApprovalRequired) == 0 {
		t.Error("an unresolvable path must not be quietly allowed")
	}
}

func TestMaxTurnTokensStopsTheTurn(t *testing.T) {
	reg := tools.NewRegistry(tools.Read{})
	usage := provider.StreamEvent{Type: provider.EventDone, Usage: &provider.Usage{OutputTokens: 500}}
	p := &scriptedProvider{turns: [][]provider.StreamEvent{
		{call("c1", "read", `{"path":"a.go"}`), usage},
		{call("c2", "read", `{"path":"b.go"}`), usage},
		{text("still going"), done()},
	}}
	e, _ := newEngine(t, p, reg, func(c *Config) {
		c.Limits = Limits{MaxIterations: 100, MaxIdenticalCalls: 0, MaxTurnTokens: 400}
	})

	out, err := e.Run(context.Background(), "go")
	if err != nil {
		t.Fatal(err)
	}
	if out.Reason != protocol.StopMaxTokens {
		t.Errorf("got %q want max_tokens", out.Reason)
	}
}

func TestUsageAccumulatesAcrossIterations(t *testing.T) {
	reg := tools.NewRegistry(tools.Read{})
	usage := provider.StreamEvent{Type: provider.EventDone, Usage: &provider.Usage{
		InputTokens: 100, OutputTokens: 10, CacheReadTokens: 90,
	}}
	p := &scriptedProvider{turns: [][]provider.StreamEvent{
		{call("c1", "read", `{"path":"a.go"}`), usage},
		{text("done"), usage},
	}}
	e, _ := newEngine(t, p, reg)

	out, err := e.Run(context.Background(), "go")
	if err != nil {
		t.Fatal(err)
	}
	if out.Usage.InputTokens != 200 || out.Usage.OutputTokens != 20 {
		t.Errorf("usage should accumulate, got %+v", out.Usage)
	}
	// Cache reads are the only direct evidence that append-only context is
	// paying off, so they must survive aggregation.
	if out.Usage.CacheReadTokens != 180 {
		t.Errorf("cache reads should accumulate, got %d", out.Usage.CacheReadTokens)
	}
}

func TestEachTurnGetsItsOwnID(t *testing.T) {
	p := &scriptedProvider{turns: [][]provider.StreamEvent{
		{text("one"), done()}, {text("two"), done()},
	}}
	e, _ := newEngine(t, p, tools.NewRegistry())

	first, _ := e.Run(context.Background(), "a")
	second, _ := e.Run(context.Background(), "b")
	if first.TurnID == second.TurnID {
		t.Errorf("turn ids must differ: %q", first.TurnID)
	}
	// And history keeps growing rather than restarting.
	if len(e.Session().History) < 4 {
		t.Errorf("history should accumulate across turns, got %d", len(e.Session().History))
	}
}

// A rate limit and a dropped connection were both falling through to abort:
// Decide classified them as wait and retry, and the loop's switch had no arm
// for either. A 429 ended the turn, with the work already done in it lost.
func TestARetryableFailureIsRetriedBeforeGivingUp(t *testing.T) {
	for _, tc := range []struct {
		name string
		err  *provider.ProviderError
	}{
		{"rate limit", &provider.ProviderError{Class: provider.ErrClassRateLimit, Retryable: true}},
		{"transport", &provider.ProviderError{Class: provider.ErrClassTransport, Retryable: true}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ws := t.TempDir()
			res, err := policy.NewResolver(ws)
			if err != nil {
				t.Fatal(err)
			}
			p := &failingProvider{err: tc.err}
			waits := 0
			e := New(Config{
				Provider: p,
				Tools:    tools.NewRegistry(),
				State:    tools.NewState(res, tools.DefaultLimits(), allToolNames),
				Emitter:  newRecorder(),
				Limits:   DefaultLimits(),
				Mode:     policy.ModeWorkspaceWrite,
				Policy:   policy.PolicyOnRequest,
				Model:    "m",
				Sleep: func(context.Context, time.Duration) bool {
					waits++
					return true
				},
			}, ce.Session{Instructions: "You are dcode."})

			out, runErr := e.Run(context.Background(), "go")
			if runErr == nil {
				t.Fatal("a failure that never clears must still surface")
			}
			if out.Reason != protocol.StopError {
				t.Errorf("reason = %q, want error once the attempts run out", out.Reason)
			}
			if waits == 0 {
				t.Fatal("nothing waited: a retryable failure went straight to abort, which is the defect")
			}
			if waits >= provider.DefaultBackoff().Tries {
				t.Errorf("waited %d times, more than the policy allows", waits)
			}
		})
	}
}

// A user who asked to stop should not sit through a backoff they cannot see.
func TestAnInterruptDuringBackoffEndsTheTurn(t *testing.T) {
	ws := t.TempDir()
	res, err := policy.NewResolver(ws)
	if err != nil {
		t.Fatal(err)
	}
	e := New(Config{
		Provider: &failingProvider{err: &provider.ProviderError{
			Class: provider.ErrClassTransport, Retryable: true,
		}},
		Tools:   tools.NewRegistry(),
		State:   tools.NewState(res, tools.DefaultLimits(), allToolNames),
		Emitter: newRecorder(),
		Limits:  DefaultLimits(),
		Mode:    policy.ModeWorkspaceWrite,
		Policy:  policy.PolicyOnRequest,
		Model:   "m",
		// The wait is where the interruption lands.
		Sleep: func(context.Context, time.Duration) bool { return false },
	}, ce.Session{Instructions: "You are dcode."})

	out, _ := e.Run(context.Background(), "go")
	if out.Reason != protocol.StopInterrupted {
		t.Fatalf("reason = %q, want interrupted", out.Reason)
	}
}
