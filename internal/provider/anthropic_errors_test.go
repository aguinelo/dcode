package provider

import (
	"errors"
	"strings"
	"testing"
)

// Anthropic does not answer an overlong prompt with 413. It answers 400
// invalid_request_error and says so in the message, and a 400 aborts the turn —
// so the one failure compaction exists to solve was the one that never reached
// it.
func TestATooLongPromptIsAContextSizeAndNotABadRequest(t *testing.T) {
	for _, tc := range []struct {
		name string
		body string
		want ErrorClass
	}{
		{
			"prompt too long",
			`{"type":"error","error":{"type":"invalid_request_error","message":"prompt is too long: 210345 tokens > 200000 maximum"}}`,
			ErrClassContextSize,
		},
		{
			"input plus max_tokens over the limit",
			`{"type":"error","error":{"type":"invalid_request_error","message":"input length and ` + "`max_tokens`" + ` exceed context limit: 198000 + 32000 > 200000, decrease input length or ` + "`max_tokens`" + ` and try again"}}`,
			ErrClassContextSize,
		},
		{
			// Specific, not tolerant to the point of uselessness: every other
			// 400 is still our bug, and has to surface as one.
			"another invalid request",
			`{"type":"error","error":{"type":"invalid_request_error","message":"messages: roles must alternate"}}`,
			ErrClassBadRequest,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			pe := ClassifyStatus(400, tc.body, "")
			if pe == nil || pe.Class != tc.want {
				t.Fatalf("got %+v, want class %s", pe, tc.want)
			}
			if tc.want == ErrClassContextSize && Decide(pe) != DecisionCompact {
				t.Errorf("decision = %s, want %s", Decide(pe), DecisionCompact)
			}
		})
	}
}

// An error can arrive inside a stream that already answered 200. Ignoring the
// frame left the stream to end without a terminal event — classed transport,
// and the provider's own words about what went wrong were dropped.
func TestAnInStreamErrorKeepsTheProvidersClassAndMessage(t *testing.T) {
	for _, tc := range []struct {
		kind, message string
		want          ErrorClass
		decision      Decision
	}{
		{"overloaded_error", "Overloaded", ErrClassProvider, DecisionRetry},
		{"api_error", "Internal server error", ErrClassProvider, DecisionRetry},
		{"rate_limit_error", "Number of request tokens has exceeded your per-minute rate limit", ErrClassRateLimit, DecisionWait},
		{"invalid_request_error", "prompt is too long: 210345 tokens > 200000 maximum", ErrClassContextSize, DecisionCompact},
		{"invalid_request_error", "messages.1: unexpected role", ErrClassBadRequest, DecisionAbort},
		{"authentication_error", "invalid x-api-key", ErrClassAuth, DecisionAbort},
		{"billing_error", "credit balance is too low", ErrClassQuota, DecisionAbort},
	} {
		t.Run(tc.kind+"/"+tc.message, func(t *testing.T) {
			d := (Claude{}).NewDecoder(nil)
			decodeFrames(t, d,
				`{"type":"message_start","message":{"usage":{"input_tokens":12,"output_tokens":1}}}`,
				`{"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"hi"}}`,
			)
			frame := `{"type":"error","error":{"type":"` + tc.kind + `","message":"` + tc.message + `"}}`
			_, err := d.Decode(WireEvent{Data: []byte(frame)})
			var pe *ProviderError
			if !errors.As(err, &pe) {
				t.Fatalf("an error frame decoded to %v, want a ProviderError", err)
			}
			if pe.Class != tc.want {
				t.Errorf("class = %s, want %s", pe.Class, tc.want)
			}
			if Decide(pe) != tc.decision {
				t.Errorf("decision = %s, want %s", Decide(pe), tc.decision)
			}
			if !strings.Contains(pe.Message, tc.message) {
				t.Errorf("message %q lost the provider's %q", pe.Message, tc.message)
			}
		})
	}
}
