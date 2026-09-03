//go:build eval

package evals

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/aguinelo/dcode/internal/app"
	ce "github.com/aguinelo/dcode/internal/contextengine"
	"github.com/aguinelo/dcode/internal/provider"
)

// runTimeout bounds a single measured exchange. A run that hangs is a run that
// measured nothing, and twenty of them hanging is an afternoon.
//
// One run is allowed this per round, and no more: a scenario with a ceiling of
// twelve rounds gets twelve times this and then it is cut. The budget is per
// RUN and never per contract — a contract-wide deadline meant the first hang
// consumed what the other nineteen runs needed, and they then failed instantly
// against a deadline that had nothing to do with them.
const runTimeout = 90 * time.Second

// setup resolves configuration and builds a provider, or skips with the reason.
//
// It skips rather than fails when eval is off, because off is the default and a
// suite that fails by default is a suite that gets disabled.
func setup(t *testing.T) (provider.Provider, Config) {
	t.Helper()

	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	// FromEnv rather than Resolve, so the credential is found the same way the
	// product finds it: the environment first, then the store `dcode login`
	// wrote to. Reading only DCODE_API_KEY meant the suite skipped on the
	// machine of anyone who had logged in properly, which is everyone.
	opts, resolved, err := app.FromEnv(os.Getenv, wd)
	if err != nil {
		t.Fatalf("resolving configuration: %v", err)
	}

	cfg := FromReader(resolved)
	if reason := cfg.Unavailable(); reason != "" {
		t.Skip(reason)
	}

	key := opts.APIKey
	if key == "" {
		t.Skip("no credential: set DCODE_API_KEY or run `dcode login`. A measurement against a real model needs one.")
	}

	reg := provider.NewRegistry()
	base := opts.BaseURL
	reg.RegisterTransport(app.NewHTTPTransport(provider.TransportOpenAI, base, key))
	reg.RegisterTransport(app.NewHTTPTransport(provider.TransportAnthropic, base, key))
	for _, f := range app.Families() {
		if err := reg.RegisterFamily(f); err != nil {
			t.Fatalf("registering family: %v", err)
		}
	}
	p, err := reg.Resolve(cfg.Model, resolved.String("model.transport", ""))
	if err != nil {
		t.Fatalf("resolving provider for %q: %v", cfg.Model, err)
	}
	return p, cfg
}

// exchange sends one request and drains the stream into its parts.
//
// The error return is reserved for a failure to MEASURE. A model that answers
// badly is a verdict and comes back through calls and text, not through err.
// cost, when non-nil, accumulates what this exchange consumed.
//
// A parameter rather than a return value: every caller wants it folded into a
// running total and none wants it on its own, and threading a fourth return
// through two call sites to add it up in both is the shape that lets the two
// disagree.
func exchange(ctx context.Context, p provider.Provider, model string, msgs []ce.Message, tools []ce.ToolDef, cost *Cost) (calls []ce.ToolCall, text string, err error) {
	events, err := p.Stream(ctx, provider.Request{Model: model, Messages: msgs, Tools: tools})
	if err != nil {
		return nil, "", err
	}
	// Counted here, before the stream is drained, so a call that starts and
	// fails still counts as a call. It was paid for.
	var usage *provider.Usage
	if cost != nil {
		defer func() { cost.Add(usage) }()
	}
	for ev := range events {
		switch ev.Type {
		case provider.EventToolCall:
			if ev.ToolCall != nil {
				calls = append(calls, *ev.ToolCall)
			}
		case provider.EventTextDelta:
			text += ev.Text
		case provider.EventDone:
			usage = ev.Usage
		case provider.EventError:
			// A tool_schema error is the adapter refusing a call, which is a
			// verdict about the model rather than a failure to measure — and
			// it is exactly what no-phantom-tool is looking for. Everything
			// else is transport.
			if ev.Err != nil && ev.Err.Class == provider.ErrClassToolSchema {
				return calls, text, nil
			}
			if ev.Err != nil {
				return nil, "", ev.Err
			}
		}
	}
	// An empty completion is a failure to MEASURE, not a verdict.
	//
	// Nothing came back: no call, no text, and the provider's own accounting
	// says zero output tokens. That is not a model declining to act — declining
	// costs tokens and leaves a sentence. It is a call that produced nothing,
	// and counting it as behaviour is the same error as reading a lost packet
	// as a regression, which this suite already refuses to do.
	//
	// Measured, not assumed: sending one scenario's exact request body to the
	// provider by hand returned a tool call twice and nothing three times out of
	// five. Left as a verdict, that provider alone would have printed 0% for a
	// contract whose behaviour was never exercised — and 0%% is the reading that
	// looks most like a finding.
	//
	// Returned as an error so it goes through the retry every transport blip
	// goes through, and if it persists the measurement is reported unsound
	// rather than green or red. The one outcome that must not happen is a
	// confident number with no exchange behind it.
	//
	// The line is "nothing a judge can read", not "zero output tokens". The
	// token count was the first cut and it was too narrow: the provider also
	// returns frames that spend tokens and carry no content, and those reached
	// the judge as "did not call the tool" just the same.
	//
	// It is NOT "no tool call". A model that answers in prose without calling
	// is a real verdict, and several contracts exist to catch exactly that —
	// swallowing it here would hide the failure they are for.
	if unreadable(len(calls), text) {
		spent := 0
		if usage != nil {
			spent = usage.OutputTokens
		}
		return nil, "", &provider.ProviderError{
			Class: provider.ErrClassProvider,
			Message: fmt.Sprintf(
				"empty completion: no tool call and no text (%d output tokens spent)", spent),
		}
	}
	return calls, text, nil
}

func report(t *testing.T, r Result) {
	t.Helper()
	t.Log(r.String())
	if !r.Met() {
		t.Errorf("contract not met: %s", r)
	}
}
