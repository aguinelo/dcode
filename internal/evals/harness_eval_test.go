//go:build eval

package evals

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/aguinelo/dcode/internal/app"
	ce "github.com/aguinelo/dcode/internal/contextengine"
	"github.com/aguinelo/dcode/internal/provider"
)

// runTimeout bounds a single measured exchange. A run that hangs is a run that
// measured nothing, and twenty of them hanging is an afternoon.
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
func exchange(ctx context.Context, p provider.Provider, model string, msgs []ce.Message, tools []ce.ToolDef) (calls []ce.ToolCall, text string, err error) {
	events, err := p.Stream(ctx, provider.Request{Model: model, Messages: msgs, Tools: tools})
	if err != nil {
		return nil, "", err
	}
	for ev := range events {
		switch ev.Type {
		case provider.EventToolCall:
			if ev.ToolCall != nil {
				calls = append(calls, *ev.ToolCall)
			}
		case provider.EventTextDelta:
			text += ev.Text
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
	return calls, text, nil
}

func report(t *testing.T, r Result) {
	t.Helper()
	t.Log(r.String())
	if !r.Met() {
		t.Errorf("contract not met: %s", r)
	}
}
