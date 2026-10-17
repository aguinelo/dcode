package loop

import (
	"context"
	"errors"
	"strings"
	"testing"

	ce "github.com/aguinelo/dcode/internal/contextengine"
	"github.com/aguinelo/dcode/internal/policy"
	"github.com/aguinelo/dcode/internal/protocol"
	"github.com/aguinelo/dcode/internal/provider"
	"github.com/aguinelo/dcode/internal/tools"
)

// longSession is a history big enough that every compaction path has a span
// to cut.
func longSession() ce.Session {
	long := ce.Session{Instructions: "You are dcode."}
	for i := 0; i < 40; i++ {
		long.History = append(long.History,
			ce.Message{Role: ce.RoleUser, Text: strings.Repeat("q ", 40)},
			ce.Message{Role: ce.RoleAssistant, Text: strings.Repeat("a ", 40)},
		)
	}
	return long
}

func compactingEngine(t *testing.T, p provider.Provider, cfg ce.Config, s ce.Session,
	summarise func(context.Context, []ce.Message) (string, error)) (*Engine, *recorder) {
	t.Helper()
	res, err := policy.NewResolver(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	rec := newRecorder()
	return New(Config{
		Provider: p, Tools: tools.NewRegistry(),
		State:   tools.NewState(res, tools.DefaultLimits(), allToolNames),
		Emitter: rec, Limits: DefaultLimits(), Mode: policy.ModeWorkspaceWrite,
		Policy: policy.PolicyOnRequest, Model: "m",
		CtxConfig: cfg, Summarise: summarise,
	}, s), rec
}

// compactionFailure is the warning the loop emitted about a summary it could
// not get, or false when none was emitted.
func compactionFailure(rec *recorder) (protocol.Error, bool) {
	rec.mu.Lock()
	defer rec.mu.Unlock()
	for _, p := range rec.all {
		if e, ok := p.(protocol.Error); ok && e.Code == "compaction_failed" {
			return e, true
		}
	}
	return protocol.Error{}, false
}

func sameHistoryPrefix(t *testing.T, got, want []ce.Message) {
	t.Helper()
	if len(got) < len(want) {
		t.Fatalf("the history shrank from %d to %d messages: part of it was dropped", len(want), len(got))
	}
	for i := range want {
		if got[i].Text != want[i].Text || got[i].Role != want[i].Role {
			t.Fatalf("message %d changed: %q became %q", i, want[i].Text, got[i].Text)
		}
	}
}

var rateLimited = &provider.ProviderError{Class: provider.ErrClassRateLimit, Message: "slow down", Retryable: true}

// A summary call that failed used to be replaced by a fixed sentence, and
// the span it should have summarised was dropped with session.compacted
// announced as if it had worked. A 429 at the wrong moment erased part of
// the session and told the person it had been summarised.
//
// The automatic trigger keeps the history whole instead, says why, and lets
// the turn go on: the trigger fires at a fraction of the budget, so there is
// still room, and the provider refusing the context is what forces the issue.
func TestAFailedSummaryKeepsTheHistoryAndSaysSo_CompactionFailure(t *testing.T) {
	cases := []struct {
		name  string
		text  string
		err   error
		cause string
	}{
		{"the call failed", "", rateLimited, "rate_limit"},
		{"the summary came back empty", "  \n", nil, "empty"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := &scriptedProvider{turns: [][]provider.StreamEvent{{text("ok"), done()}}}
			long := longSession()
			e, rec := compactingEngine(t, p, ce.Config{CompactAt: 0.5, KeepTurns: 2, Window: 500}, long,
				func(context.Context, []ce.Message) (string, error) { return tc.text, tc.err })

			out, err := e.Run(context.Background(), "continue")
			if err != nil {
				t.Fatalf("a failed summary ended the turn: %v", err)
			}
			if out.Reason != protocol.StopDone {
				t.Errorf("reason = %q, want done", out.Reason)
			}
			if e.Session().Summary != nil {
				t.Errorf("a summary was applied that nobody wrote: %q", e.Session().Summary.Text)
			}
			sameHistoryPrefix(t, e.Session().History, long.History)
			if n := rec.count(protocol.EventSessionCompacted); n != 0 {
				t.Errorf("session.compacted announced %d time(s) for a compaction that did not happen", n)
			}
			w, ok := compactionFailure(rec)
			if !ok {
				t.Fatal("nothing said the summary failed")
			}
			if !strings.Contains(w.Message, tc.cause) {
				t.Errorf("the warning does not name the cause %q: %q", tc.cause, w.Message)
			}
		})
	}
}

// /compact is the person asking directly, and they are told — as an error,
// with the cause — rather than shown a compaction that threw work away.
func TestAFailedSummaryFailsTheForcedCompaction_CompactionFailure(t *testing.T) {
	long := longSession()
	e, rec := compactingEngine(t, nil, ce.Config{Window: 10_000_000}, long,
		func(context.Context, []ce.Message) (string, error) { return "", rateLimited })

	compacted, err := e.Compact(context.Background())
	if err == nil {
		t.Fatal("a compaction whose summary failed reported no error")
	}
	var perr *provider.ProviderError
	if !errors.As(err, &perr) || perr.Class != provider.ErrClassRateLimit {
		t.Errorf("the error lost its classification: %v", err)
	}
	if compacted {
		t.Error("reported compacted with no summary")
	}
	if e.Session().Summary != nil {
		t.Error("a summary was applied that nobody wrote")
	}
	sameHistoryPrefix(t, e.Session().History, long.History)
	if n := rec.count(protocol.EventSessionCompacted); n != 0 {
		t.Errorf("session.compacted announced %d time(s)", n)
	}
}

// The provider refusing the context forces a compaction. If its summary fails
// too there is nothing left to try: the turn ends on the classified error, the
// history stays whole, and both causes are said.
func TestAContextSizeRefusalWithAFailedSummaryEndsTheTurn_CompactionFailure(t *testing.T) {
	tooBig := provider.StreamEvent{Type: provider.EventError, Err: &provider.ProviderError{
		Class: provider.ErrClassContextSize, Message: "too many tokens", Retryable: true,
	}}
	p := &scriptedProvider{turns: [][]provider.StreamEvent{{tooBig}, {text("that fits now"), done()}}}
	long := longSession()
	e, rec := compactingEngine(t, p, ce.Config{Window: 10_000_000}, long,
		func(context.Context, []ce.Message) (string, error) { return "", rateLimited })

	out, err := e.Run(context.Background(), "continue")
	var perr *provider.ProviderError
	if !errors.As(err, &perr) || perr.Class != provider.ErrClassContextSize {
		t.Fatalf("the turn did not end on the context_size error: %v", err)
	}
	if out.Reason != protocol.StopError {
		t.Errorf("reason = %q, want error", out.Reason)
	}
	if e.Session().Summary != nil {
		t.Error("a summary was applied that nobody wrote")
	}
	sameHistoryPrefix(t, e.Session().History, long.History)
	if n := rec.count(protocol.EventSessionCompacted); n != 0 {
		t.Errorf("session.compacted announced %d time(s)", n)
	}
	if _, ok := compactionFailure(rec); !ok {
		t.Error("nothing said the summary failed")
	}
	var saidContextSize bool
	rec.mu.Lock()
	for _, p := range rec.all {
		if e, ok := p.(protocol.Error); ok && e.Code == string(provider.ErrClassContextSize) {
			saidContextSize = true
		}
	}
	rec.mu.Unlock()
	if !saidContextSize {
		t.Error("the turn ended on context_size without saying so")
	}
}
