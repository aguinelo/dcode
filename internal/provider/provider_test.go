package provider

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	ce "github.com/aguinelo/dcode/internal/contextengine"
)

func tools() []ce.ToolDef {
	return []ce.ToolDef{
		{Name: "read", Description: "Read a file.", Schema: json.RawMessage(`{"type":"object"}`)},
		{Name: "bash", Description: "Run a command.", Schema: json.RawMessage(`{"type":"object"}`)},
	}
}

func request() Request {
	return Request{
		Model: "MiniMax-M3",
		Messages: []ce.Message{
			{Role: ce.RoleSystem, Text: "You are dcode."},
			{Role: ce.RoleUser, Text: "list the files"},
		},
		Tools: tools(),
	}
}

func registry(t *testing.T, frames ...string) (*Registry, *ReplayTransport) {
	t.Helper()
	rt := NewReplayTransport(TransportOpenAI, Transcript{Name: "t", Frames: frames})
	r := NewRegistry()
	r.RegisterTransport(rt)
	if err := r.RegisterFamily(MiniMaxM3{}); err != nil {
		t.Fatal(err)
	}
	return r, rt
}

func drain(t *testing.T, ch <-chan StreamEvent) []StreamEvent {
	t.Helper()
	var out []StreamEvent
	for ev := range ch {
		out = append(out, ev)
	}
	return out
}

// Exactly one terminal event, never both, never neither. A stream that ends
// without one hangs the loop forever, which is the worst failure this layer can
// produce because nothing times out.
func TestStreamAlwaysEndsWithExactlyOneTerminal(t *testing.T) {
	for _, tc := range []struct {
		name   string
		frames []string
	}{
		{"clean finish", []string{`{"choices":[{"delta":{"content":"hi"}}]}`, "[DONE]"}},
		{"finish reason only", []string{`{"choices":[{"delta":{"content":"hi"},"finish_reason":"stop"}]}`}},
		{"usage frame", []string{`{"usage":{"prompt_tokens":10,"completion_tokens":2}}`}},
		{"truncated mid-stream", []string{`{"choices":[{"delta":{"content":"hi"}}]}`}},
		{"empty stream", nil},
		{"trailing frames after done", []string{"[DONE]", `{"choices":[{"delta":{"content":"late"}}]}`}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r, _ := registry(t, tc.frames...)
			p, err := r.Resolve("MiniMax-M3", "")
			if err != nil {
				t.Fatal(err)
			}
			ch, err := p.Stream(context.Background(), request())
			if err != nil {
				t.Fatal(err)
			}
			evs := drain(t, ch)

			terminals := 0
			for i, ev := range evs {
				if ev.Type == EventDone || ev.Type == EventError {
					terminals++
					if i != len(evs)-1 {
						t.Errorf("terminal event at %d is not last of %d", i, len(evs))
					}
				}
			}
			if terminals != 1 {
				t.Errorf("want exactly 1 terminal event, got %d in %v", terminals, kinds(evs))
			}
		})
	}
}

func TestTruncatedStreamIsRetryableTransportError(t *testing.T) {
	r, _ := registry(t, `{"choices":[{"delta":{"content":"partial"}}]}`)
	p, _ := r.Resolve("MiniMax-M3", "")
	ch, _ := p.Stream(context.Background(), request())
	evs := drain(t, ch)

	last := evs[len(evs)-1]
	if last.Type != EventError {
		t.Fatalf("a truncated stream must end in an error, got %s", last.Type)
	}
	// Reporting success here would hand the loop a half-formed turn and the
	// model would carry on as if the answer were complete.
	if last.Err.Class != ErrClassTransport || !last.Err.Retryable {
		t.Errorf("want a retryable transport error, got %+v", last.Err)
	}
}

func TestCancelClosesChannelWithCanceled(t *testing.T) {
	r, _ := registry(t, `{"choices":[{"delta":{"content":"a"}}]}`,
		`{"choices":[{"delta":{"content":"b"}}]}`, "[DONE]")
	p, _ := r.Resolve("MiniMax-M3", "")

	ctx, cancel := context.WithCancel(context.Background())
	ch, err := p.Stream(ctx, request())
	if err != nil {
		t.Fatal(err)
	}
	cancel()

	// Whatever was already buffered may still arrive; what matters is that the
	// channel closes rather than leaking the goroutine.
	done := make(chan struct{})
	go func() { drain(t, ch); close(done) }()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("channel did not close after cancellation")
	}
}

// A tool the model invented must never reach the loop.
func TestUndeclaredToolNeverReachesTheLoop(t *testing.T) {
	r, _ := registry(t,
		`{"choices":[{"delta":{"tool_calls":[{"id":"c1","function":{"name":"rm_rf","arguments":"{}"}}]}}]}`)
	p, _ := r.Resolve("MiniMax-M3", "")
	ch, _ := p.Stream(context.Background(), request())
	evs := drain(t, ch)

	for _, ev := range evs {
		if ev.Type == EventToolCall {
			t.Fatalf("undeclared tool %q reached the loop", ev.ToolCall.Name)
		}
	}
	last := evs[len(evs)-1]
	if last.Type != EventError || last.Err.Class != ErrClassToolSchema {
		t.Fatalf("want a tool_schema error, got %+v", last)
	}
	if !strings.Contains(last.Err.Message, "read") {
		t.Errorf("the error should list what is available so the model can correct: %q",
			last.Err.Message)
	}
}

func TestMalformedToolArgumentsNeverReachTheLoop(t *testing.T) {
	r, _ := registry(t,
		`{"choices":[{"delta":{"tool_calls":[{"id":"c1","function":{"name":"read","arguments":"{not json"}}]}}]}`)
	p, _ := r.Resolve("MiniMax-M3", "")
	evs := drain(t, mustStream(t, p))

	for _, ev := range evs {
		if ev.Type == EventToolCall {
			t.Fatal("a call with unparseable arguments reached the loop")
		}
	}
	if last := evs[len(evs)-1]; last.Err == nil || last.Err.Class != ErrClassToolSchema {
		t.Fatalf("want tool_schema, got %+v", last)
	}
}

func TestEmptyToolArgumentsBecomeAnEmptyObject(t *testing.T) {
	// A model that omits arguments for a no-argument tool is behaving
	// correctly; rejecting that would be our bug, not its.
	r, _ := registry(t,
		`{"choices":[{"delta":{"tool_calls":[{"id":"c1","function":{"name":"read","arguments":""}}]}}]}`,
		"[DONE]")
	p, _ := r.Resolve("MiniMax-M3", "")
	evs := drain(t, mustStream(t, p))

	var call *ce.ToolCall
	for _, ev := range evs {
		if ev.Type == EventToolCall {
			call = ev.ToolCall
		}
	}
	if call == nil {
		t.Fatal("the call should have been delivered")
	}
	if string(call.Input) != "{}" {
		t.Errorf("empty arguments should become {}, got %q", call.Input)
	}
}

// CacheReadTokens is the only direct measure that append-only context works.
func TestUsageCarriesCacheReadTokens(t *testing.T) {
	r, _ := registry(t,
		`{"usage":{"prompt_tokens":1000,"completion_tokens":50,"prompt_tokens_details":{"cached_tokens":900}}}`)
	p, _ := r.Resolve("MiniMax-M3", "")
	evs := drain(t, mustStream(t, p))

	last := evs[len(evs)-1]
	if last.Usage == nil {
		t.Fatal("the terminal event should carry usage")
	}
	if last.Usage.CacheReadTokens != 900 {
		t.Errorf("cache read tokens: got %d want 900", last.Usage.CacheReadTokens)
	}
}

// The test that justifies the whole two-axis design: one family, two dialects,
// two different valid bodies, no duplicated adaptation.
func TestOneFamilyEncodesForBothTransports(t *testing.T) {
	fam := MiniMaxM3{}
	req := request()

	oa, err := fam.Encode(req, TransportOpenAI)
	if err != nil {
		t.Fatal(err)
	}
	an, err := fam.Encode(req, TransportAnthropic)
	if err != nil {
		t.Fatal(err)
	}

	if string(oa.Body) == string(an.Body) {
		t.Fatal("the two dialects must serialize differently")
	}
	if !json.Valid(oa.Body) || !json.Valid(an.Body) {
		t.Fatal("both bodies must be valid JSON")
	}

	// The dialects differ in exactly the way that matters: one keeps the
	// system prompt in the message list, the other hoists it to its own field.
	var anBody map[string]any
	if err := json.Unmarshal(an.Body, &anBody); err != nil {
		t.Fatal(err)
	}
	if anBody["system"] != "You are dcode." {
		t.Errorf("anthropic must hoist the system prompt, got %v", anBody["system"])
	}

	var oaBody struct {
		Messages []struct {
			Role string `json:"role"`
		} `json:"messages"`
	}
	if err := json.Unmarshal(oa.Body, &oaBody); err != nil {
		t.Fatal(err)
	}
	if len(oaBody.Messages) == 0 || oaBody.Messages[0].Role != "system" {
		t.Error("openai must keep the system prompt as the first message")
	}
}

func TestAnthropicEncodesToolResultsAsUserContent(t *testing.T) {
	req := request()
	req.Messages = append(req.Messages,
		ce.Message{Role: ce.RoleAssistant, ToolCalls: []ce.ToolCall{
			{ID: "c1", Name: "read", Input: json.RawMessage(`{"path":"a.go"}`)}}},
		ce.Message{Role: ce.RoleTool, ToolResult: &ce.ToolResult{
			ToolCallID: "c1", Output: "package main", IsError: false}},
	)
	wire, err := MiniMaxM3{}.Encode(req, TransportAnthropic)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(wire.Body), "tool_result") {
		t.Errorf("tool results must use this dialect's shape: %s", wire.Body)
	}
	if !strings.Contains(string(wire.Body), "tool_use") {
		t.Errorf("tool calls must use this dialect's shape: %s", wire.Body)
	}
}

func TestAnthropicSuppliesMaxTokensWhenAbsent(t *testing.T) {
	// The dialect requires the field; omitting it is a 400 the user cannot act
	// on, so the family fills it rather than passing the failure through.
	wire, err := Claude{}.Encode(Request{Model: "claude-x", Messages: []ce.Message{
		{Role: ce.RoleUser, Text: "hi"}}}, TransportAnthropic)
	if err != nil {
		t.Fatal(err)
	}
	var body struct {
		MaxTokens int `json:"max_tokens"`
	}
	if err := json.Unmarshal(wire.Body, &body); err != nil {
		t.Fatal(err)
	}
	if body.MaxTokens <= 0 {
		t.Errorf("max_tokens must be present, got %d", body.MaxTokens)
	}
}

func TestFamilyRejectsUnsupportedTransport(t *testing.T) {
	if _, err := (Claude{}).Encode(request(), TransportOpenAI); err == nil {
		t.Error("claude does not speak the openai dialect and must say so")
	}
	if _, err := (MiniMaxM3{}).Encode(request(), "grpc"); err == nil {
		t.Error("an unknown transport must be rejected")
	}
}

func mustStream(t *testing.T, p Provider) <-chan StreamEvent {
	t.Helper()
	ch, err := p.Stream(context.Background(), request())
	if err != nil {
		t.Fatal(err)
	}
	return ch
}

func kinds(evs []StreamEvent) []StreamEventType {
	out := make([]StreamEventType, len(evs))
	for i, e := range evs {
		out[i] = e.Type
	}
	return out
}

func TestProviderErrorFormatting(t *testing.T) {
	e := &ProviderError{Class: ErrClassAuth, Message: "rejected"}
	if e.Error() != "auth: rejected" {
		t.Errorf("got %q", e.Error())
	}
	bare := &ProviderError{Class: ErrClassQuota}
	if bare.Error() != "quota" {
		t.Errorf("got %q", bare.Error())
	}
}

func TestClassifyPreservesAnAlreadyClassifiedError(t *testing.T) {
	orig := &ProviderError{Class: ErrClassQuota, Message: "out of credit"}
	if got := classify(errors.Join(errors.New("ctx"), orig)); got.Class != ErrClassQuota {
		t.Errorf("wrapping must not lose the class, got %s", got.Class)
	}
	if got := classify(context.Canceled); got.Class != ErrClassCanceled {
		t.Errorf("got %s", got.Class)
	}
	if got := classify(nil); got.Class != ErrClassProvider {
		t.Errorf("got %s", got.Class)
	}
	if got := classify(errors.New("connection reset")); got.Class != ErrClassTransport || !got.Retryable {
		t.Errorf("an unknown error should be a retryable transport failure, got %+v", got)
	}
}
