package loop

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	ce "github.com/aguinelo/dcode/internal/contextengine"
	"github.com/aguinelo/dcode/internal/protocol"
	"github.com/aguinelo/dcode/internal/provider"
	"github.com/aguinelo/dcode/internal/tools"
)

// unpaired reports the first message of msgs a provider rejects for its tool
// pairing, read the way the OpenAI dialect reads it: a tool message answers a
// call of the assistant message the run of tool messages it is in follows, by
// a non-empty id, and every call of that message is answered before anything
// else is said. Empty means the history is one a provider accepts.
//
// Written here rather than borrowed from the encoder, because the encoder's own
// guard is one of the things this test is about.
func unpaired(msgs []ce.Message) string {
	open := map[string]bool{} // the calls the last assistant message made, still unanswered
	inRun := false            // whether the tool messages so far follow that message
	for i, m := range msgs {
		if m.Role == ce.RoleTool {
			if m.ToolResult == nil || m.ToolResult.ToolCallID == "" {
				return fmt.Sprintf("message %d is a tool result with no call id", i)
			}
			if !inRun || !open[m.ToolResult.ToolCallID] {
				return fmt.Sprintf("message %d answers call %q, which no preceding assistant message made",
					i, m.ToolResult.ToolCallID)
			}
			delete(open, m.ToolResult.ToolCallID)
			continue
		}
		if len(open) > 0 {
			return fmt.Sprintf("message %d comes before every call of the assistant message above it was answered", i)
		}
		inRun = false
		if m.Role == ce.RoleAssistant && len(m.ToolCalls) > 0 {
			for _, c := range m.ToolCalls {
				if c.ID == "" {
					return fmt.Sprintf("message %d makes a call with no id", i)
				}
				open[c.ID] = true
			}
			inRun = true
		}
	}
	if len(open) > 0 {
		return "the history ends on calls nobody answered"
	}
	return ""
}

// An ordinary session continuing an orchestrator's conversation reads calls to
// tools it does not have, and a model that reads them calls one. The provider
// refuses that call before it runs (tool_schema), and the refusal has to reach
// the model as something a provider accepts on the next request.
//
// It reached it as a tool message with no call id and no call above it — which
// MiniMax-M3 answered, on the next turn, with `tool result's tool id() not
// found (2013)`, and the continued session could not take another turn.
func TestARefusedCallLeavesAHistoryTheProviderAccepts(t *testing.T) {
	// What Rebuild hands a session that continues an orchestrator's record.
	seeded := []ce.Message{
		{Role: ce.RoleUser, Text: "split the parser work"},
		{Role: ce.RoleAssistant, Text: "Delegating.", ToolCalls: []ce.ToolCall{
			{ID: "call_d1", Name: "delegate", Input: json.RawMessage(`{"task":"parser"}`)},
			{ID: "call_c1", Name: "children", Input: json.RawMessage(`{}`)},
		}},
		{Role: ce.RoleTool, ToolResult: &ce.ToolResult{ToolCallID: "call_d1", Output: "child c1 started"}},
		{Role: ce.RoleTool, ToolResult: &ce.ToolResult{ToolCallID: "call_c1", Output: "c1 working"}},
		{Role: ce.RoleAssistant, Text: "c1 is on it."},
	}
	p := &scriptedProvider{turns: [][]provider.StreamEvent{
		{text("Delegating again."), {Type: provider.EventError, Err: &provider.ProviderError{
			Class:   provider.ErrClassToolSchema,
			Message: `tool "delegate" was not declared; available: read`,
		}}},
		{text("I cannot delegate here, so I read it myself."), done()},
		{text("Done."), done()},
	}}
	e, rec := newEngine(t, p, tools.NewRegistry(tools.Read{}))
	e.session.History = append(e.session.History, seeded...)

	if _, err := e.Run(context.Background(), "keep going"); err != nil {
		t.Fatal(err)
	}
	if len(p.seen) != 2 {
		t.Errorf("the turn made %d requests, want a second one for the model to correct the refused call", len(p.seen))
	}
	if rec.count(protocol.EventSessionError) != 1 {
		t.Error("a call that did not run has to be said on the stream")
	}
	if _, err := e.Run(context.Background(), "and now?"); err != nil {
		t.Fatal(err)
	}

	for n, req := range p.seen {
		if why := unpaired(req.Messages); why != "" {
			t.Errorf("request %d would be refused by the provider: %s", n+1, why)
		}
	}
	told := false
	for _, m := range e.Session().History {
		if strings.Contains(m.Text, `tool "delegate" was not declared`) {
			told = true
		}
	}
	if !told {
		t.Error("the refusal has to reach the model, or it calls the same tool again")
	}
}
