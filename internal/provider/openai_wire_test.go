package provider

import (
	"encoding/json"
	"strings"
	"testing"

	ce "github.com/aguinelo/dcode/internal/contextengine"
)

// The OpenAI dialect carries a tool result as its own role with the call it
// answers, and a tool call as a function entry. A result that loses its call id
// is a result the model cannot attach to anything it asked for.
func TestTheOpenAIEncoderCarriesToolCallsAndTheirResults(t *testing.T) {
	wire, err := MiniMaxM3{}.Encode(Request{
		Model: "MiniMax-M3",
		Messages: []ce.Message{
			{Role: ce.RoleUser, Text: "read a.txt"},
			{Role: ce.RoleAssistant, ToolCalls: []ce.ToolCall{
				{ID: "c1", Name: "read", Input: json.RawMessage(`{"path":"a.txt"}`)},
			}},
			{Role: ce.RoleTool, ToolResult: &ce.ToolResult{ToolCallID: "c1", Output: "hello"}},
		},
		Tools: []ce.ToolDef{{Name: "read", Description: "read a file", Schema: json.RawMessage(`{"type":"object"}`)}},
	}, TransportOpenAI)
	if err != nil {
		t.Fatal(err)
	}

	var body struct {
		Messages []struct {
			Role       string          `json:"role"`
			Content    json.RawMessage `json:"content"`
			ToolCallID string          `json:"tool_call_id"`
			ToolCalls  []struct {
				ID       string `json:"id"`
				Type     string `json:"type"`
				Function struct {
					Name      string `json:"name"`
					Arguments string `json:"arguments"`
				} `json:"function"`
			} `json:"tool_calls"`
		} `json:"messages"`
		Tools []struct {
			Type     string `json:"type"`
			Function struct {
				Name string `json:"name"`
			} `json:"function"`
		} `json:"tools"`
	}
	if err := json.Unmarshal(wire.Body, &body); err != nil {
		t.Fatal(err)
	}
	if len(body.Messages) != 3 {
		t.Fatalf("got %d messages, want three", len(body.Messages))
	}

	call := body.Messages[1].ToolCalls
	if len(call) != 1 || call[0].ID != "c1" || call[0].Type != "function" {
		t.Fatalf("the tool call did not survive: %+v", call)
	}
	if call[0].Function.Name != "read" || call[0].Function.Arguments != `{"path":"a.txt"}` {
		t.Errorf("the call lost its name or arguments: %+v", call[0].Function)
	}

	result := body.Messages[2]
	if result.ToolCallID != "c1" {
		t.Errorf("the result lost the call it answers: %+v", result)
	}
	if string(result.Content) != `"hello"` {
		t.Errorf("content = %s, want the output", result.Content)
	}

	if len(body.Tools) != 1 || body.Tools[0].Function.Name != "read" {
		t.Errorf("the tool declaration did not survive: %+v", body.Tools)
	}
}

// An image crosses as a data URI part alongside the text, in that order,
// because the text is what the picture is being sent about. A message with an
// image but no text carries only the image rather than an empty text part the
// provider then rejects.
func TestTheOpenAIEncoderCarriesImagesAsPartsBesideTheText(t *testing.T) {
	png := []byte{0x89, 'P', 'N', 'G'}

	for _, tc := range []struct {
		name      string
		text      string
		wantParts int
	}{
		{"with text", "what is wrong here?", 2},
		{"without text", "", 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			wire, err := MiniMaxM3{}.Encode(Request{
				Model: "MiniMax-M3",
				Messages: []ce.Message{{
					Role:   ce.RoleUser,
					Text:   tc.text,
					Images: []ce.Image{{MediaType: "image/png", Data: png}},
				}},
			}, TransportOpenAI)
			if err != nil {
				t.Fatal(err)
			}

			var body struct {
				Messages []struct {
					Content []struct {
						Type     string `json:"type"`
						Text     string `json:"text"`
						ImageURL *struct {
							URL string `json:"url"`
						} `json:"image_url"`
					} `json:"content"`
				} `json:"messages"`
			}
			if err := json.Unmarshal(wire.Body, &body); err != nil {
				t.Fatalf("the message did not become a list of parts: %v", err)
			}

			parts := body.Messages[0].Content
			if len(parts) != tc.wantParts {
				t.Fatalf("got %d parts, want %d: %+v", len(parts), tc.wantParts, parts)
			}
			last := parts[len(parts)-1]
			if last.Type != "image_url" || last.ImageURL == nil {
				t.Fatalf("the image is not the last part: %+v", parts)
			}
			// Addressed by content, never by a path or a handle: a request has
			// to encode the same way on any machine, at any time.
			if !strings.HasPrefix(last.ImageURL.URL, "data:image/png;base64,") {
				t.Errorf("url = %q, want a data URI carrying the bytes", last.ImageURL.URL)
			}
			if tc.wantParts == 2 && (parts[0].Type != "text" || parts[0].Text != tc.text) {
				t.Errorf("the text is not the first part: %+v", parts[0])
			}
		})
	}
}

// The Anthropic stream ends explicitly with message_stop, so closing adds
// nothing. A stream that ended without one really was truncated, and inventing
// a completion here would hide that.
func TestClosingAnAnthropicStreamAddsNothing(t *testing.T) {
	d := (Claude{}).NewDecoder([]ce.ToolDef{{Name: "read"}})
	if got := d.Close(); len(got) != 0 {
		t.Errorf("closing produced %+v, want nothing", got)
	}
}

// A failed tool result says it failed on every dialect, in the same words.
//
// The Anthropic dialect has a field for it and the OpenAI one does not, so for
// as long as the mark lived only in is_error, a model behind the OpenAI dialect
// — MiniMax over it, Gemini, anything generic — read a failure exactly as it
// read a success. Whether the model was told "this call failed" depended on
// the family running it, and the harness is the part that should not.
//
// One family speaking both dialects is the cleanest witness: the same result,
// encoded twice, has to say the same thing twice.
func TestAFailedToolResultSaysSoOnEveryDialect(t *testing.T) {
	for _, tc := range []struct {
		name   string
		failed bool
	}{
		{"failed", true},
		{"succeeded", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			result := &ce.ToolResult{ToolCallID: "c1", Output: "no such file: a.txt", IsError: tc.failed}
			req := Request{
				Model: "MiniMax-M3",
				Messages: []ce.Message{
					{Role: ce.RoleUser, Text: "read a.txt"},
					{Role: ce.RoleAssistant, ToolCalls: []ce.ToolCall{
						{ID: "c1", Name: "read", Input: json.RawMessage(`{"path":"a.txt"}`)},
					}},
					{Role: ce.RoleTool, ToolResult: result},
				},
			}

			oaWire, err := MiniMaxM3{}.Encode(req, TransportOpenAI)
			if err != nil {
				t.Fatal(err)
			}
			var oa struct {
				Messages []struct {
					Content string `json:"content"`
				} `json:"messages"`
			}
			if err := json.Unmarshal(oaWire.Body, &oa); err != nil {
				t.Fatal(err)
			}

			anWire, err := MiniMaxM3{}.Encode(req, TransportAnthropic)
			if err != nil {
				t.Fatal(err)
			}
			var an struct {
				Messages []struct {
					Content []struct {
						Content string `json:"content"`
						IsError bool   `json:"is_error"`
					} `json:"content"`
				} `json:"messages"`
			}
			if err := json.Unmarshal(anWire.Body, &an); err != nil {
				t.Fatal(err)
			}

			oaSays := oa.Messages[2].Content
			anBlock := an.Messages[2].Content[0]

			if oaSays != anBlock.Content {
				t.Errorf("the dialects say different things for one result:\n openai:    %q\n anthropic: %q",
					oaSays, anBlock.Content)
			}
			if anBlock.IsError != tc.failed {
				t.Errorf("anthropic is_error = %v, want %v", anBlock.IsError, tc.failed)
			}
			if !strings.HasSuffix(oaSays, result.Output) {
				t.Errorf("content = %q, want it to end with the tool's own output", oaSays)
			}
			if tc.failed && oaSays == result.Output {
				t.Errorf("openai content = %q: a failure reads exactly like a success", oaSays)
			}
			if !tc.failed && oaSays != result.Output {
				t.Errorf("content = %q: a success is marked, want the output untouched", oaSays)
			}
		})
	}
}
