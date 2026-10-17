package loop

import (
	"context"
	"errors"
	"testing"

	"github.com/aguinelo/dcode/internal/protocol"
	"github.com/aguinelo/dcode/internal/provider"
	"github.com/aguinelo/dcode/internal/tools"
)

// A model that refuses ends the turn refused: never done, never retried, and
// said. Before, the refusal arrived as a clean done and the turn read it — or
// the half an answer before it — as the answer.
func TestARefusalEndsTheTurnRefusedAndNotDone(t *testing.T) {
	refused := &provider.ProviderError{
		Class: provider.ErrClassRefused, Message: "the model refused (category cyber)",
	}
	p := &scriptedProvider{turns: [][]provider.StreamEvent{
		{text("half an"), {
			Type: provider.EventError, Err: refused,
			Usage: &provider.Usage{InputTokens: 12, OutputTokens: 7},
		}},
		{text("a second try nobody asked for"), done()},
	}}
	e, rec := newEngine(t, p, tools.NewRegistry(tools.Read{}))

	out, err := e.Run(context.Background(), "go")
	var pe *provider.ProviderError
	if !errors.As(err, &pe) || pe.Class != provider.ErrClassRefused {
		t.Errorf("err = %v, want the refusal", err)
	}
	if out.Reason != protocol.StopRefused {
		t.Errorf("reason = %s, want %s", out.Reason, protocol.StopRefused)
	}
	if len(p.seen) != 1 {
		t.Errorf("requests = %d, want 1: a refusal is not retried", len(p.seen))
	}
	if got, ok := rec.last[protocol.EventSessionError].(protocol.Error); !ok || got.Code != string(provider.ErrClassRefused) || got.Message != refused.Message {
		t.Errorf("session.error = %+v, want code %q with the provider's message", rec.last[protocol.EventSessionError], provider.ErrClassRefused)
	}
	for _, m := range e.Session().History {
		if m.Text == "half an" {
			t.Error("the refused half answer entered the history as if it were one")
		}
	}
	if out.Usage.OutputTokens != 7 {
		t.Errorf("output tokens = %d, want the 7 the refused answer was billed", out.Usage.OutputTokens)
	}
}
