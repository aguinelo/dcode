package provider

import (
	"encoding/json"
	"fmt"
	"strings"

	ce "github.com/aguinelo/dcode/internal/contextengine"
)

// Wire names.
const (
	TransportOpenAI    = "openai"
	TransportAnthropic = "anthropic"
)

// MiniMaxM3 is the project's primary family.
//
// It speaks both dialects, which is the concrete reason transport and family
// are separate axes: with one axis, supporting both would mean two families
// with identical adaptation and identical thresholds, and the copies would
// diverge at the first maintenance.
type MiniMaxM3 struct{}

func (MiniMaxM3) Name() string         { return "minimax-m3" }
func (MiniMaxM3) Transports() []string { return []string{TransportOpenAI, TransportAnthropic} }
func (MiniMaxM3) Models() []string     { return []string{"MiniMax-M3", "minimax-m3"} }

func (MiniMaxM3) Window(string) (int, error) { return 1_000_000, nil }

// DefaultLimits allows far more iterations than a short-task model would.
// M3 is trained for long-horizon agent loops — MiniMax demonstrated a run with
// 1,959 tool calls — and a cap sized for a ten-file refactor would truncate
// legitimate work. The repeat detector remains the real defence; this is the
// backstop, and a backstop tracks the model's horizon.
func (MiniMaxM3) DefaultLimits() Limits {
	return Limits{MaxIterations: 200}
}

func (f MiniMaxM3) Encode(req Request, transport string) (WireRequest, error) {
	switch transport {
	case TransportOpenAI:
		return encodeOpenAI(req)
	case TransportAnthropic:
		return encodeAnthropic(req)
	}
	return WireRequest{}, fmt.Errorf("family %s: unsupported transport %q", f.Name(), transport)
}

func (f MiniMaxM3) Decode(ev WireEvent, tools []ce.ToolDef) (StreamEvent, error) {
	return decodeOpenAI(ev, tools)
}

// Claude is the second family. It exists to prove the axes are orthogonal: one
// implementation never validates an abstraction.
type Claude struct{}

func (Claude) Name() string         { return "claude" }
func (Claude) Transports() []string { return []string{TransportAnthropic} }
func (Claude) Models() []string     { return []string{"claude-"} }

func (Claude) Window(string) (int, error) { return 200_000, nil }

// DefaultLimits is sized from the case that shaped it: a refactor across ten
// files runs thirty to fifty tool calls.
func (Claude) DefaultLimits() Limits { return Limits{MaxIterations: 50} }

func (f Claude) Encode(req Request, transport string) (WireRequest, error) {
	if transport != TransportAnthropic {
		return WireRequest{}, fmt.Errorf("family %s: unsupported transport %q", f.Name(), transport)
	}
	return encodeAnthropic(req)
}

func (Claude) Decode(ev WireEvent, tools []ce.ToolDef) (StreamEvent, error) {
	return decodeAnthropic(ev, tools)
}

// ---------- OpenAI dialect ----------

type oaMessage struct {
	Role       string       `json:"role"`
	Content    string       `json:"content,omitempty"`
	ToolCalls  []oaToolCall `json:"tool_calls,omitempty"`
	ToolCallID string       `json:"tool_call_id,omitempty"`
}

type oaToolCall struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}

type oaTool struct {
	Type     string `json:"type"`
	Function struct {
		Name        string          `json:"name"`
		Description string          `json:"description"`
		Parameters  json.RawMessage `json:"parameters"`
	} `json:"function"`
}

type oaRequest struct {
	Model     string      `json:"model"`
	Messages  []oaMessage `json:"messages"`
	Tools     []oaTool    `json:"tools,omitempty"`
	Stream    bool        `json:"stream"`
	MaxTokens int         `json:"max_tokens,omitempty"`
}

func encodeOpenAI(req Request) (WireRequest, error) {
	body := oaRequest{Model: req.Model, Stream: true, MaxTokens: req.MaxTokens}

	for _, m := range req.Messages {
		om := oaMessage{Role: string(m.Role), Content: m.Text}
		if m.Role == ce.RoleTool && m.ToolResult != nil {
			om.ToolCallID = m.ToolResult.ToolCallID
			om.Content = m.ToolResult.Output
		}
		for _, c := range m.ToolCalls {
			var tc oaToolCall
			tc.ID = c.ID
			tc.Type = "function"
			tc.Function.Name = c.Name
			tc.Function.Arguments = string(c.Input)
			om.ToolCalls = append(om.ToolCalls, tc)
		}
		body.Messages = append(body.Messages, om)
	}

	for _, t := range req.Tools {
		var ot oaTool
		ot.Type = "function"
		ot.Function.Name = t.Name
		ot.Function.Description = t.Description
		ot.Function.Parameters = t.Schema
		body.Tools = append(body.Tools, ot)
	}

	raw, err := json.Marshal(body)
	if err != nil {
		return WireRequest{}, err
	}
	return WireRequest{Model: req.Model, Body: raw, Stream: true}, nil
}

type oaChunk struct {
	Choices []struct {
		Delta struct {
			Content   string       `json:"content"`
			ToolCalls []oaToolCall `json:"tool_calls"`
		} `json:"delta"`
		FinishReason *string `json:"finish_reason"`
	} `json:"choices"`
	Usage *struct {
		PromptTokens        int `json:"prompt_tokens"`
		CompletionTokens    int `json:"completion_tokens"`
		PromptTokensDetails struct {
			CachedTokens int `json:"cached_tokens"`
		} `json:"prompt_tokens_details"`
	} `json:"usage"`
}

func decodeOpenAI(ev WireEvent, tools []ce.ToolDef) (StreamEvent, error) {
	data := strings.TrimSpace(string(ev.Data))
	if data == "" {
		return StreamEvent{}, nil
	}
	if data == "[DONE]" {
		return StreamEvent{Type: EventDone}, nil
	}

	var c oaChunk
	if err := json.Unmarshal([]byte(data), &c); err != nil {
		return StreamEvent{}, &ProviderError{
			Class:   ErrClassProvider,
			Message: "malformed stream frame: " + sanitize(err.Error()),
		}
	}

	if c.Usage != nil {
		return StreamEvent{Type: EventDone, Usage: &Usage{
			InputTokens:     c.Usage.PromptTokens,
			OutputTokens:    c.Usage.CompletionTokens,
			CacheReadTokens: c.Usage.PromptTokensDetails.CachedTokens,
		}}, nil
	}
	if len(c.Choices) == 0 {
		return StreamEvent{}, nil
	}

	ch := c.Choices[0]
	for _, tc := range ch.Delta.ToolCalls {
		call, err := validateToolCall(tc.ID, tc.Function.Name, tc.Function.Arguments, tools)
		if err != nil {
			return StreamEvent{}, err
		}
		return StreamEvent{Type: EventToolCall, ToolCall: call}, nil
	}
	if ch.Delta.Content != "" {
		return StreamEvent{Type: EventTextDelta, Text: ch.Delta.Content}, nil
	}
	if ch.FinishReason != nil {
		return StreamEvent{Type: EventDone}, nil
	}
	return StreamEvent{}, nil
}

// ---------- Anthropic dialect ----------

type anRequest struct {
	Model     string      `json:"model"`
	System    string      `json:"system,omitempty"`
	Messages  []anMessage `json:"messages"`
	Tools     []anTool    `json:"tools,omitempty"`
	Stream    bool        `json:"stream"`
	MaxTokens int         `json:"max_tokens"`
}

type anMessage struct {
	Role    string      `json:"role"`
	Content []anContent `json:"content"`
}

type anContent struct {
	Type      string          `json:"type"`
	Text      string          `json:"text,omitempty"`
	ID        string          `json:"id,omitempty"`
	Name      string          `json:"name,omitempty"`
	Input     json.RawMessage `json:"input,omitempty"`
	ToolUseID string          `json:"tool_use_id,omitempty"`
	Content   string          `json:"content,omitempty"`
	IsError   bool            `json:"is_error,omitempty"`
}

type anTool struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	InputSchema json.RawMessage `json:"input_schema"`
}

// encodeAnthropic hoists the system message out of the message list, which is
// where this dialect expects it. Same rules, different placement — exactly the
// difference the two-axis design exists to absorb.
func encodeAnthropic(req Request) (WireRequest, error) {
	body := anRequest{Model: req.Model, Stream: true, MaxTokens: req.MaxTokens}
	if body.MaxTokens == 0 {
		body.MaxTokens = 8192 // this dialect requires the field
	}

	for _, m := range req.Messages {
		switch m.Role {
		case ce.RoleSystem:
			body.System = m.Text
			continue
		case ce.RoleTool:
			if m.ToolResult == nil {
				continue
			}
			body.Messages = append(body.Messages, anMessage{Role: "user", Content: []anContent{{
				Type: "tool_result", ToolUseID: m.ToolResult.ToolCallID,
				Content: m.ToolResult.Output, IsError: m.ToolResult.IsError,
			}}})
			continue
		}

		am := anMessage{Role: string(m.Role)}
		if m.Text != "" {
			am.Content = append(am.Content, anContent{Type: "text", Text: m.Text})
		}
		for _, c := range m.ToolCalls {
			am.Content = append(am.Content, anContent{
				Type: "tool_use", ID: c.ID, Name: c.Name, Input: c.Input,
			})
		}
		if len(am.Content) == 0 {
			continue
		}
		body.Messages = append(body.Messages, am)
	}

	for _, t := range req.Tools {
		body.Tools = append(body.Tools, anTool{
			Name: t.Name, Description: t.Description, InputSchema: t.Schema,
		})
	}

	raw, err := json.Marshal(body)
	if err != nil {
		return WireRequest{}, err
	}
	return WireRequest{Model: req.Model, Body: raw, Stream: true}, nil
}

type anEvent struct {
	Type         string `json:"type"`
	ContentBlock *struct {
		Type  string          `json:"type"`
		ID    string          `json:"id"`
		Name  string          `json:"name"`
		Input json.RawMessage `json:"input"`
	} `json:"content_block"`
	Delta *struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"delta"`
	Usage *struct {
		InputTokens              int `json:"input_tokens"`
		OutputTokens             int `json:"output_tokens"`
		CacheReadInputTokens     int `json:"cache_read_input_tokens"`
		CacheCreationInputTokens int `json:"cache_creation_input_tokens"`
	} `json:"usage"`
}

func decodeAnthropic(ev WireEvent, tools []ce.ToolDef) (StreamEvent, error) {
	data := strings.TrimSpace(string(ev.Data))
	if data == "" {
		return StreamEvent{}, nil
	}

	var e anEvent
	if err := json.Unmarshal([]byte(data), &e); err != nil {
		return StreamEvent{}, &ProviderError{
			Class:   ErrClassProvider,
			Message: "malformed stream frame: " + sanitize(err.Error()),
		}
	}

	switch e.Type {
	case "content_block_delta":
		if e.Delta != nil && e.Delta.Text != "" {
			return StreamEvent{Type: EventTextDelta, Text: e.Delta.Text}, nil
		}
	case "content_block_start":
		if e.ContentBlock != nil && e.ContentBlock.Type == "tool_use" {
			call, err := validateToolCall(
				e.ContentBlock.ID, e.ContentBlock.Name, string(e.ContentBlock.Input), tools)
			if err != nil {
				return StreamEvent{}, err
			}
			return StreamEvent{Type: EventToolCall, ToolCall: call}, nil
		}
	case "message_delta", "message_stop":
		se := StreamEvent{Type: EventDone}
		if e.Usage != nil {
			se.Usage = &Usage{
				InputTokens:      e.Usage.InputTokens,
				OutputTokens:     e.Usage.OutputTokens,
				CacheReadTokens:  e.Usage.CacheReadInputTokens,
				CacheWriteTokens: e.Usage.CacheCreationInputTokens,
			}
		}
		return se, nil
	}
	return StreamEvent{}, nil
}

// ---------- shared validation ----------

// validateToolCall enforces two rules before a call ever reaches the loop:
// the tool must be one we declared, and the arguments must be valid JSON.
//
// Executing a guessed call is how an agent corrupts a file. There is no case
// where running it anyway is worth the risk, so a failure here becomes a
// tool_schema error the model reads and corrects.
func validateToolCall(id, name, args string, tools []ce.ToolDef) (*ce.ToolCall, error) {
	if name == "" {
		return nil, &ProviderError{Class: ErrClassToolSchema, Message: "tool call has no name"}
	}
	declared := false
	for _, t := range tools {
		if t.Name == name {
			declared = true
			break
		}
	}
	if !declared {
		return nil, &ProviderError{
			Class:   ErrClassToolSchema,
			Message: fmt.Sprintf("tool %q was not declared; available: %s", name, toolNames(tools)),
		}
	}

	if strings.TrimSpace(args) == "" {
		args = "{}"
	}
	if !json.Valid([]byte(args)) {
		return nil, &ProviderError{
			Class:   ErrClassToolSchema,
			Message: fmt.Sprintf("arguments for %q are not valid JSON", name),
		}
	}
	return &ce.ToolCall{ID: id, Name: name, Input: json.RawMessage(args)}, nil
}

func toolNames(tools []ce.ToolDef) string {
	names := make([]string, 0, len(tools))
	for _, t := range tools {
		names = append(names, t.Name)
	}
	return strings.Join(names, ", ")
}
