package provider

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	ce "github.com/aguinelo/dcode/internal/contextengine"
)

func orchestratorHistory() []ce.Message {
	return []ce.Message{
		{Role: ce.RoleSystem, Text: "be brief"},
		{Role: ce.RoleUser, Text: "split the parser work"},
		{Role: ce.RoleAssistant, Text: "Delegating.", ToolCalls: []ce.ToolCall{
			{ID: "call_d1", Name: "delegate", Input: json.RawMessage(`{"task":"parser"}`)},
			{ID: "call_c1", Name: "children", Input: json.RawMessage(`{}`)},
		}},
		{Role: ce.RoleTool, ToolResult: &ce.ToolResult{ToolCallID: "call_d1", Output: "child c1 started"}},
		{Role: ce.RoleTool, ToolResult: &ce.ToolResult{ToolCallID: "call_c1", Output: "c1 working"}},
		{Role: ce.RoleUser, Text: "a note the harness appended", Reminder: true},
		{Role: ce.RoleAssistant, Text: "c1 is on it."},
	}
}

// Calls to tools this request does not declare are history, and history is
// sent as it was: the provider reads past calls without the tool declared.
func TestAReplayedConversationWithToolsNoLongerDeclaredIsSent(t *testing.T) {
	msgs := append(orchestratorHistory(), ce.Message{Role: ce.RoleUser, Text: "keep going"})
	if err := CheckToolPairs(msgs); err != nil {
		t.Fatalf("a paired history was refused: %v", err)
	}
	req := Request{Model: "m", Messages: msgs, Tools: []ce.ToolDef{{Name: "read", Schema: json.RawMessage(`{}`)}}}
	if _, err := encodeOpenAI(req); err != nil {
		t.Errorf("openai dialect: %v", err)
	}
	if _, err := encodeAnthropic(req, 1024, false); err != nil {
		t.Errorf("anthropic dialect: %v", err)
	}
}

// Every shape a provider rejects for its pairing is refused before it is sent,
// in both dialects, naming the message — the provider's own answer named
// neither (`tool result's tool id() not found`).
func TestAnUnpairedToolMessageIsRefusedBeforeItIsSent(t *testing.T) {
	cases := map[string]struct {
		msgs []ce.Message
		want string
	}{
		"a result with no call id, the live MiniMax 400": {
			msgs: append(orchestratorHistory(),
				ce.Message{Role: ce.RoleTool, ToolResult: &ce.ToolResult{Output: `tool "delegate" was not declared`, IsError: true}},
				ce.Message{Role: ce.RoleUser, Text: "and now?"}),
			want: "message 7 is a tool result with no call id",
		},
		"a result for a call no message made": {
			msgs: []ce.Message{
				{Role: ce.RoleUser, Text: "go"},
				{Role: ce.RoleTool, ToolResult: &ce.ToolResult{ToolCallID: "c9", Output: "x"}},
			},
			want: `message 1 answers call "c9"`,
		},
		"a result for the call of an earlier message": {
			msgs: []ce.Message{
				{Role: ce.RoleAssistant, ToolCalls: []ce.ToolCall{{ID: "c1", Name: "read"}}},
				{Role: ce.RoleTool, ToolResult: &ce.ToolResult{ToolCallID: "c1"}},
				{Role: ce.RoleAssistant, Text: "ok"},
				{Role: ce.RoleTool, ToolResult: &ce.ToolResult{ToolCallID: "c1"}},
			},
			want: `message 3 answers call "c1"`,
		},
		"a call left unanswered before the next message": {
			msgs: []ce.Message{
				{Role: ce.RoleAssistant, ToolCalls: []ce.ToolCall{{ID: "c1", Name: "read"}, {ID: "c2", Name: "read"}}},
				{Role: ce.RoleTool, ToolResult: &ce.ToolResult{ToolCallID: "c1"}},
				{Role: ce.RoleUser, Text: "next"},
			},
			want: "message 2 comes before 1 call(s)",
		},
		"a call left unanswered at the end": {
			msgs: []ce.Message{
				{Role: ce.RoleAssistant, ToolCalls: []ce.ToolCall{{ID: "c1", Name: "read"}}},
			},
			want: "nobody answered",
		},
		"a call with no id": {
			msgs: []ce.Message{
				{Role: ce.RoleAssistant, ToolCalls: []ce.ToolCall{{Name: "read"}}},
				{Role: ce.RoleTool, ToolResult: &ce.ToolResult{}},
			},
			want: `message 0 makes a call to "read" with no id`,
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			req := Request{Model: "m", Messages: tc.msgs}
			for dialect, encode := range map[string]func() error{
				"openai":    func() error { _, err := encodeOpenAI(req); return err },
				"anthropic": func() error { _, err := encodeAnthropic(req, 1024, false); return err },
			} {
				err := encode()
				if err == nil {
					t.Fatalf("%s: sent a conversation the provider rejects", dialect)
				}
				if !strings.Contains(err.Error(), tc.want) {
					t.Errorf("%s: %v, want it to say %q", dialect, err, tc.want)
				}
				var pe *ProviderError
				if !errors.As(err, &pe) || Decide(pe) != DecisionAbort {
					t.Errorf("%s: %v must abort, not retry: the same history is refused again", dialect, err)
				}
			}
		})
	}
}
