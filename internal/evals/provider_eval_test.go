//go:build eval

// Behavioural contracts of the provider adapter, section 6 of
// 202608072334-provider-adapter.p.spec.md.
//
// Behind the `eval` tag because every run here reaches a real model and costs
// money. Nothing in `make check` builds this file; `make eval` does.
package evals

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/aguinelo/dcode/internal/app"
	ce "github.com/aguinelo/dcode/internal/contextengine"
	"github.com/aguinelo/dcode/internal/provider"
)

// runTimeout bounds a single measured exchange. A run that hangs is a run that
// did not measure anything, and twenty of them hanging is an afternoon.
const runTimeout = 90 * time.Second

// setup resolves configuration and builds a provider, or skips with the reason.
//
// It skips rather than fails when eval is off, because "off" is the default and
// a suite that fails by default is a suite that gets disabled.
func setup(t *testing.T) (provider.Provider, Config) {
	t.Helper()

	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	resolved, err := app.Resolve(os.Getenv, wd)
	if err != nil {
		t.Fatalf("resolving configuration: %v", err)
	}

	cfg := FromReader(resolved)
	if reason := cfg.Unavailable(); reason != "" {
		t.Skip(reason)
	}

	key := os.Getenv("DCODE_API_KEY")
	if key == "" {
		t.Skip("no DCODE_API_KEY: a measurement against a real model needs a credential")
	}

	reg := provider.NewRegistry()
	base := resolved.String("model.base_url", "")
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
// The error return is reserved for a failure to measure. A model that answers
// badly is a verdict and comes back through calls/text, not through err.
func exchange(ctx context.Context, p provider.Provider, model string, msgs []ce.Message, tools []ce.ToolDef) (calls []ce.ToolCall, text string, err error) {
	events, err := p.Stream(ctx, provider.Request{
		Model:    model,
		Messages: msgs,
		Tools:    tools,
	})
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
			// verdict about the model and not a failure to measure. Everything
			// else is transport, and must not read as a behavioural result.
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

// ---------- toolcall-schema-valid, ≥ 97% ----------

// release is the shape tools.json declares. The judge decodes into it rather
// than trusting the adapter, which checks that arguments are valid JSON and
// that the tool was declared — not that the schema was honoured. See
// testdata/evals/toolcall-schema-valid/scenario.md.
type release struct {
	Release struct {
		Version   string `json:"version"`
		Artifacts []struct {
			OS     string `json:"os"`
			SHA256 string `json:"sha256"`
		} `json:"artifacts"`
	} `json:"release"`
}

func TestEvalToolcallSchemaValid(t *testing.T) {
	p, cfg := setup(t)
	f, err := LoadFixture(FixtureRoot, "toolcall-schema-valid")
	if err != nil {
		t.Fatal(err)
	}

	attempt := func(ctx context.Context) (bool, error) {
		calls, _, err := exchange(ctx, p, cfg.Model,
			[]ce.Message{{Role: ce.RoleUser, Text: f.Task}}, f.Tools)
		if err != nil {
			return false, err
		}
		if len(calls) != 1 || calls[0].Name != "record_release" {
			return false, nil
		}
		var got release
		if err := json.Unmarshal(calls[0].Input, &got); err != nil {
			return false, nil // malformed against the schema is a verdict, not a measurement failure
		}
		if got.Release.Version == "" || len(got.Release.Artifacts) < 2 {
			return false, nil
		}
		for _, a := range got.Release.Artifacts {
			if a.OS == "" || len(a.SHA256) != 64 {
				return false, nil
			}
		}
		return true, nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), runTimeout*time.Duration(cfg.Runs))
	defer cancel()
	report(t, Measure(ctx, cfg, "toolcall-schema-valid", 0.97, attempt))
}

// ---------- toolcall-recover, ≥ 90% ----------

func TestEvalToolcallRecover(t *testing.T) {
	p, cfg := setup(t)
	f, err := LoadFixture(FixtureRoot, "toolcall-recover")
	if err != nil {
		t.Fatal(err)
	}

	// The wording matches what RN-8 emits in production. Measuring recovery
	// against a different error text measures a different product, because
	// RN-3 of behavior-definition makes tool error text behaviour surface.
	const rejection = `arguments for "read" are not valid: "limit" is required when "offset" is set`

	attempt := func(ctx context.Context) (bool, error) {
		msgs := []ce.Message{{Role: ce.RoleUser, Text: f.Task}}

		first, _, err := exchange(ctx, p, cfg.Model, msgs, f.Tools)
		if err != nil {
			return false, err
		}
		if len(first) == 0 {
			return false, nil // nothing to recover from means nothing was measured
		}

		msgs = append(msgs,
			ce.Message{Role: ce.RoleAssistant, ToolCalls: first},
			ce.Message{Role: ce.RoleTool, ToolResult: &ce.ToolResult{
				ToolCallID: first[0].ID, Output: rejection, IsError: true,
			}})

		second, _, err := exchange(ctx, p, cfg.Model, msgs, f.Tools)
		if err != nil {
			return false, err
		}
		if len(second) == 0 || second[0].Name != "read" {
			return false, nil
		}
		var args struct {
			Path   string `json:"path"`
			Offset int    `json:"offset"`
			Limit  int    `json:"limit"`
		}
		if err := json.Unmarshal(second[0].Input, &args); err != nil {
			return false, nil
		}
		// Recovered means the correction the error named is present. Repeating
		// the same call, or answering in prose, is not recovery.
		return args.Path != "" && args.Limit > 0, nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), runTimeout*time.Duration(cfg.Runs)*2)
	defer cancel()
	report(t, Measure(ctx, cfg, "toolcall-recover", 0.90, attempt))
}

// ---------- no-phantom-tool, 100% ----------

func TestEvalNoPhantomTool(t *testing.T) {
	p, cfg := setup(t)
	f, err := LoadFixture(FixtureRoot, "no-phantom-tool")
	if err != nil {
		t.Fatal(err)
	}

	attempted := 0
	attempt := func(ctx context.Context) (bool, error) {
		calls, _, err := exchange(ctx, p, cfg.Model,
			[]ce.Message{{Role: ce.RoleUser, Text: f.Task}}, f.Tools)
		if err != nil {
			return false, err
		}
		for _, c := range calls {
			if !f.Declares(c.Name) {
				attempted++
				return false, nil
			}
		}
		return true, nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), runTimeout*time.Duration(cfg.Runs))
	defer cancel()
	r := Measure(ctx, cfg, "no-phantom-tool", 1.0, attempt)
	report(t, r)

	// Counted, never judged. What the contract forbids is an undeclared call
	// reaching the loop, not the model wanting to make one — and the threshold
	// is 100% precisely because the adapter, not the model, is what guarantees
	// it. See the scenario note.
	t.Logf("undeclared calls that reached the judge: %d of %d runs", attempted, r.Runs)
}
