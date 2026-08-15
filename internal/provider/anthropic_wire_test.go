package provider

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	ce "github.com/aguinelo/dcode/internal/contextengine"
)

// The window is what the budget bands are computed from. A family that reports
// the wrong one does not fail — it silently compacts too early or too late.
func TestClaudeReportsItsWindow(t *testing.T) {
	got, err := Claude{}.Window("claude-opus-4")
	if err != nil {
		t.Fatal(err)
	}
	if got != 200_000 {
		t.Errorf("window = %d, want 200000", got)
	}
}

// Every kind of message has to survive the crossing: a system prompt becomes a
// top-level field, a tool result becomes a user turn, and a tool call becomes a
// content block. Getting any of them wrong is a request the provider rejects
// with an error the person cannot connect to what they did.
func TestTheAnthropicEncoderCarriesEveryKindOfMessage(t *testing.T) {
	wire, err := Claude{}.Encode(Request{
		Model: "claude-x",
		Messages: []ce.Message{
			{Role: ce.RoleSystem, Text: "be brief"},
			{Role: ce.RoleUser, Text: "read a.txt"},
			{Role: ce.RoleAssistant, ToolCalls: []ce.ToolCall{
				{ID: "c1", Name: "read", Input: json.RawMessage(`{"path":"a.txt"}`)},
			}},
			{Role: ce.RoleTool, ToolResult: &ce.ToolResult{ToolCallID: "c1", Output: "hello"}},
			// A tool message with no result carries nothing, and an empty
			// content block is what the provider rejects.
			{Role: ce.RoleTool},
			// An assistant turn with neither text nor calls is dropped for the
			// same reason.
			{Role: ce.RoleAssistant},
		},
		Tools: []ce.ToolDef{{Name: "read", Description: "read a file", Schema: json.RawMessage(`{"type":"object"}`)}},
	}, TransportAnthropic)
	if err != nil {
		t.Fatal(err)
	}

	var body struct {
		System   string `json:"system"`
		Messages []struct {
			Role    string `json:"role"`
			Content []struct {
				Type      string `json:"type"`
				Text      string `json:"text"`
				ID        string `json:"id"`
				Name      string `json:"name"`
				ToolUseID string `json:"tool_use_id"`
			} `json:"content"`
		} `json:"messages"`
		Tools []struct {
			Name string `json:"name"`
		} `json:"tools"`
	}
	if err := json.Unmarshal(wire.Body, &body); err != nil {
		t.Fatal(err)
	}

	if body.System != "be brief" {
		t.Errorf("system = %q, want the system message hoisted out of the list", body.System)
	}
	if len(body.Messages) != 3 {
		t.Fatalf("got %d messages, want the three that carry something", len(body.Messages))
	}
	if body.Messages[1].Content[0].Type != "tool_use" || body.Messages[1].Content[0].ID != "c1" {
		t.Errorf("the tool call did not survive: %+v", body.Messages[1].Content[0])
	}
	if body.Messages[2].Role != "user" || body.Messages[2].Content[0].ToolUseID != "c1" {
		t.Errorf("the tool result did not become a user turn naming its call: %+v", body.Messages[2])
	}
	if len(body.Tools) != 1 || body.Tools[0].Name != "read" {
		t.Errorf("the tool declaration did not survive: %+v", body.Tools)
	}
}

// decodeAll feeds frames through a decoder the way the transport does.
func decodeFrames(t *testing.T, d Decoder, frames ...string) []StreamEvent {
	t.Helper()
	var out []StreamEvent
	for _, f := range frames {
		got, err := d.Decode(WireEvent{Data: []byte(f)})
		if err != nil {
			t.Fatalf("decoding %s: %v", f, err)
		}
		out = append(out, got...)
	}
	return out
}

// A tool call arrives split across frames, and reasoning and text arrive
// interleaved with it. Reassembling that wrong is a call executed with half its
// arguments.
func TestTheAnthropicDecoderReassemblesACallSplitAcrossFrames(t *testing.T) {
	d := (Claude{}).NewDecoder([]ce.ToolDef{{Name: "read"}})

	got := decodeFrames(t, d,
		`{"type":"content_block_start","index":0,"content_block":{"type":"tool_use","id":"c1","name":"read"}}`,
		// The same index opening twice must not start a second call.
		`{"type":"content_block_start","index":0,"content_block":{"type":"tool_use","id":"c1","name":"read"}}`,
		`{"type":"content_block_delta","index":0,"delta":{"partial_json":"{\"path\":"}}`,
		`{"type":"content_block_delta","index":0,"delta":{"partial_json":"\"a.txt\"}"}}`,
		`{"type":"content_block_delta","index":1,"delta":{"thinking":"weighing it"}}`,
		`{"type":"content_block_delta","index":1,"delta":{"text":"reading"}}`,
		// A delta with nothing in it is not an event.
		`{"type":"content_block_delta","index":1}`,
		`{"type":"message_delta","usage":{"input_tokens":11,"output_tokens":7}}`,
		// Stopping after the stream already ended must not end it twice.
		`{"type":"message_stop"}`,
	)

	var calls, reasoning, text, done int
	for _, e := range got {
		switch e.Type {
		case EventToolCall:
			calls++
			if string(e.ToolCall.Input) != `{"path":"a.txt"}` {
				t.Errorf("arguments = %s, want the two fragments joined", e.ToolCall.Input)
			}
		case EventReasoningDelta:
			reasoning++
		case EventTextDelta:
			text++
		case EventDone:
			done++
			if e.Usage == nil || e.Usage.InputTokens != 11 {
				t.Errorf("usage did not survive: %+v", e.Usage)
			}
		}
	}
	if calls != 1 {
		t.Errorf("got %d tool calls, want one — the second open started another", calls)
	}
	if reasoning != 1 || text != 1 {
		t.Errorf("got %d reasoning and %d text deltas, want one each", reasoning, text)
	}
	if done != 1 {
		t.Errorf("the stream ended %d times", done)
	}
}

// A call with no arguments at all is a call with empty arguments, not a
// malformed one. Refusing it would fail every zero-argument tool.
func TestACallThatSentNoArgumentsBecomesAnEmptyObject(t *testing.T) {
	d := (Claude{}).NewDecoder([]ce.ToolDef{{Name: "status"}})
	got := decodeFrames(t, d,
		`{"type":"content_block_start","index":0,"content_block":{"type":"tool_use","id":"c1","name":"status"}}`,
		`{"type":"message_stop"}`,
	)
	for _, e := range got {
		if e.Type == EventToolCall && string(e.ToolCall.Input) != "{}" {
			t.Errorf("arguments = %s, want an empty object", e.ToolCall.Input)
		}
	}
}

// Two rules stand between the stream and the loop, and executing a guessed call
// is how an agent corrupts a file. Both failures have to reach the model as
// something it can correct rather than as a crash.
func TestACallTheStreamInventedIsRefusedBeforeItRuns(t *testing.T) {
	for _, tc := range []struct {
		name  string
		frame string
		want  string
	}{
		{"undeclared", `{"type":"content_block_start","index":0,"content_block":{"type":"tool_use","id":"c1","name":"rm"}}`, "was not declared"},
		{"unnamed", `{"type":"content_block_start","index":0,"content_block":{"type":"tool_use","id":"c1"}}`, "no name"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			d := (Claude{}).NewDecoder([]ce.ToolDef{{Name: "read"}})
			if _, err := d.Decode(WireEvent{Data: []byte(tc.frame)}); err != nil {
				t.Fatalf("opening the block already failed: %v", err)
			}
			_, err := d.Decode(WireEvent{Data: []byte(`{"type":"message_stop"}`)})
			if err == nil {
				t.Fatal("the call was accepted")
			}
			var pe *ProviderError
			if !errors.As(err, &pe) || pe.Class != ErrClassToolSchema {
				t.Fatalf("err = %v, want a tool-schema error the model can correct", err)
			}
			if !strings.Contains(pe.Message, tc.want) {
				t.Errorf("message = %q, want it to say %q", pe.Message, tc.want)
			}
		})
	}
}

// Arguments that are not JSON are refused for the same reason, and the message
// names the tool so the model knows which call to send again.
func TestArgumentsThatAreNotJSONAreRefused(t *testing.T) {
	d := (Claude{}).NewDecoder([]ce.ToolDef{{Name: "read"}})
	if _, err := d.Decode(WireEvent{Data: []byte(
		`{"type":"content_block_start","index":0,"content_block":{"type":"tool_use","id":"c1","name":"read"}}`)}); err != nil {
		t.Fatal(err)
	}
	if _, err := d.Decode(WireEvent{Data: []byte(
		`{"type":"content_block_delta","index":0,"delta":{"partial_json":"{not json"}}`)}); err != nil {
		t.Fatal(err)
	}
	_, err := d.Decode(WireEvent{Data: []byte(`{"type":"message_stop"}`)})
	if err == nil || !strings.Contains(err.Error(), `"read"`) {
		t.Fatalf("err = %v, want a refusal naming the tool", err)
	}
}
