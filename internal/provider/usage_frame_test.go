package provider

import (
	"testing"

	ce "github.com/aguinelo/dcode/internal/contextengine"
)

// A frame carrying the call AND the usage yields the call.
//
// This is the shape Gemini sends, captured from a real response: one frame with
// the tool call, the finish reason and the usage together, then `[DONE]`. It
// went in and nothing came out — the usage branch ran first, flushed a decoder
// that had absorbed nothing, emitted the terminal event and returned. The call
// was dropped with the stream reporting a clean end.
//
// The failure had no symptom that pointed at it. From the outside a model that
// answers nothing and a decoder that discards the answer look identical, and the
// usage numbers came back correct the whole time, which reads as a healthy call.
// It took sending the client's own request body to the provider by hand, and
// getting a tool call back that the client had not seen, to tell the two apart.
//
// Note the tool_calls entry has no `index` field. That is also Gemini, and it is
// harmless — absent means zero, and zero is the first call — but it is kept here
// verbatim rather than tidied, because a fixture edited into what the wire
// "should" say is a fixture that stops testing the wire.
func TestAFrameCarryingUsageStillYieldsItsToolCall(t *testing.T) {
	d := &openAIDecoder{tools: []ce.ToolDef{{
		Name:   "glob",
		Schema: []byte(`{"type":"object","properties":{"pattern":{"type":"string"}},"required":["pattern"]}`),
	}}}

	frame := `{"choices":[{"delta":{"role":"assistant","tool_calls":[{"function":{"arguments":"{\"pattern\":\"*\"}","name":"glob"},"id":"function-call-18439433602034082669","type":"function"}]},"finish_reason":"tool_calls","index":0}],"created":1788465147,"model":"gemini-2.5-flash","object":"chat.completion.chunk","usage":{"completion_tokens":12,"prompt_tokens":3379,"total_tokens":3459}}`

	events, err := d.Decode(WireEvent{Data: []byte(frame)})
	if err != nil {
		t.Fatalf("decoding: %v", err)
	}

	var call *ce.ToolCall
	var usage *Usage
	for _, ev := range events {
		switch ev.Type {
		case EventToolCall:
			call = ev.ToolCall
		case EventDone:
			usage = ev.Usage
		}
	}

	if call == nil {
		t.Fatal("the frame carried a tool call and none came out; the whole family cannot call a tool")
	}
	if call.Name != "glob" {
		t.Errorf("call name is %q, want glob", call.Name)
	}
	if got := string(call.Input); got != `{"pattern":"*"}` {
		t.Errorf("call arguments are %s", got)
	}

	// And the usage still arrives. The fix must not buy the call by losing the
	// accounting, which is the trade the original ordering was making in the
	// other direction.
	if usage == nil {
		t.Fatal("no usage reached the terminal event")
	}
	if usage.InputTokens != 3379 || usage.OutputTokens != 12 {
		t.Errorf("usage is in %d out %d, want in 3379 out 12", usage.InputTokens, usage.OutputTokens)
	}
}

// Usage on a frame of its own still terminates, and does not re-emit.
//
// This is the OpenAI and MiniMax shape, and it is what held while the Gemini one
// was broken. Kept beside it so the fix is pinned from both sides: a decoder
// that emitted the call twice would run every tool twice, which is worse than
// the defect being fixed.
func TestUsageOnItsOwnFrameTerminatesWithoutRepeatingTheCall(t *testing.T) {
	d := &openAIDecoder{tools: []ce.ToolDef{{
		Name:   "glob",
		Schema: []byte(`{"type":"object","properties":{"pattern":{"type":"string"}},"required":["pattern"]}`),
	}}}

	finish := `{"choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"id":"c1","type":"function","function":{"name":"glob","arguments":"{\"pattern\":\"*\"}"}}]},"finish_reason":"tool_calls"}]}`
	later := `{"choices":[],"usage":{"prompt_tokens":100,"completion_tokens":7}}`

	first, err := d.Decode(WireEvent{Data: []byte(finish)})
	if err != nil {
		t.Fatalf("decoding the finish frame: %v", err)
	}
	calls := 0
	for _, ev := range first {
		if ev.Type == EventToolCall {
			calls++
		}
	}
	if calls != 1 {
		t.Fatalf("the finish frame produced %d calls, want 1", calls)
	}

	second, err := d.Decode(WireEvent{Data: []byte(later)})
	if err != nil {
		t.Fatalf("decoding the usage frame: %v", err)
	}
	var usage *Usage
	for _, ev := range second {
		if ev.Type == EventToolCall {
			t.Error("the usage frame emitted the call a second time; every tool would run twice")
		}
		if ev.Type == EventDone {
			usage = ev.Usage
		}
	}
	if usage == nil || usage.InputTokens != 100 || usage.OutputTokens != 7 {
		t.Errorf("usage did not survive the separate frame: %+v", usage)
	}
}
