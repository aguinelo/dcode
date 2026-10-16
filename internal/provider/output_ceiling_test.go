package provider

import (
	"encoding/json"
	"testing"

	ce "github.com/aguinelo/dcode/internal/contextengine"
)

// The ceiling of one answer is a property of the model, and the Anthropic
// dialect requires it on every request. A fixed 8192 cut long answers on every
// model that allows far more, and said nothing.
func TestTheClaudeFamilyDeclaresAnOutputCeilingPerModel(t *testing.T) {
	for _, tc := range []struct {
		model string
		want  int
	}{
		{"claude-opus-5-5", 128_000},
		{"claude-fable-5-1", 128_000},
		{"claude-sonnet-4-6", 128_000},
		{"claude-opus-4-6", 128_000},
		{"claude-haiku-4-5", 64_000},
		{"claude-haiku-4-5-20251001", 64_000},
		{"claude-opus-4-5-20251101", 64_000},
		{"claude-sonnet-4-5", 64_000},
		{"claude-sonnet-4-20250514", 64_000},
		{"claude-opus-4-1", 32_000},
		{"claude-opus-4-20250514", 32_000},
		{"claude-3-haiku-20240307", 4_096},
		// A model this table has not heard of gets the conservative answer,
		// not the largest one: too large is a request the API refuses.
		{"claude-next", 32_000},
	} {
		if got := (Claude{}).OutputCeiling(tc.model); got != tc.want {
			t.Errorf("%s: ceiling = %d, want %d", tc.model, got, tc.want)
		}
	}
}

func maxTokensOf(t *testing.T, wire WireRequest) int {
	t.Helper()
	var body struct {
		MaxTokens int `json:"max_tokens"`
	}
	if err := json.Unmarshal(wire.Body, &body); err != nil {
		t.Fatal(err)
	}
	return body.MaxTokens
}

// An unset ceiling is the family's answer for the model; a configured one wins.
func TestTheAnthropicRequestCarriesTheModelsCeiling(t *testing.T) {
	msgs := []ce.Message{{Role: ce.RoleUser, Text: "write it all"}}

	wire, err := Claude{}.Encode(Request{Model: "claude-opus-4-6", Messages: msgs}, TransportAnthropic)
	if err != nil {
		t.Fatal(err)
	}
	if got := maxTokensOf(t, wire); got != 128_000 {
		t.Errorf("unset: max_tokens = %d, want the model's 128000", got)
	}

	wire, err = Claude{}.Encode(Request{Model: "claude-opus-4-6", Messages: msgs, MaxTokens: 1000}, TransportAnthropic)
	if err != nil {
		t.Fatal(err)
	}
	if got := maxTokensOf(t, wire); got != 1000 {
		t.Errorf("configured: max_tokens = %d, want 1000", got)
	}
}

// An answer that stopped at the ceiling says so, in both dialects. Ending it as
// an ordinary finish is how a cut answer looked like a complete one.
func TestAnAnswerCutAtTheCeilingSaysSo(t *testing.T) {
	for _, tc := range []struct {
		name   string
		dec    Decoder
		frames []string
		want   bool
	}{
		{"anthropic max_tokens", (Claude{}).NewDecoder(nil), []string{
			`{"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"half an"}}`,
			`{"type":"message_delta","delta":{"stop_reason":"max_tokens"},"usage":{"output_tokens":8192}}`,
		}, true},
		{"anthropic end_turn", (Claude{}).NewDecoder(nil), []string{
			`{"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"done"}}`,
			`{"type":"message_delta","delta":{"stop_reason":"end_turn"},"usage":{"output_tokens":3}}`,
		}, false},
		{"openai length", (MiniMaxM3{}).NewDecoder(nil), []string{
			`{"choices":[{"delta":{"content":"half an"},"finish_reason":"length"}]}`,
			`[DONE]`,
		}, true},
		{"openai stop", (MiniMaxM3{}).NewDecoder(nil), []string{
			`{"choices":[{"delta":{"content":"done"},"finish_reason":"stop"}]}`,
			`[DONE]`,
		}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var done *StreamEvent
			for _, ev := range decodeFrames(t, tc.dec, tc.frames...) {
				if ev.Type == EventDone {
					ev := ev
					done = &ev
				}
			}
			if done == nil {
				t.Fatal("no terminal event")
			}
			if done.Truncated != tc.want {
				t.Errorf("Truncated = %v, want %v", done.Truncated, tc.want)
			}
		})
	}
}
