package provider

import (
	"strings"
	"testing"

	ce "github.com/aguinelo/dcode/internal/contextengine"
)

// A Claude answer that ends with stop_reason "refusal" is the model declining,
// on a 200. Read as any other stop it ended as a clean done, and the done check
// took the refusal — or the half an answer before it — for the answer.
//
// It ends as a classified failure instead: not done, not retried, with what the
// provider said about why and with the usage the refused answer was billed.
func TestARefusalEndsAsAClassifiedFailureAndNotAsDone(t *testing.T) {
	for _, tc := range []struct {
		name    string
		details string
		want    []string
	}{
		{"with a category", `,"stop_details":{"type":"refusal","category":"cyber","explanation":"declined by policy"}`,
			[]string{"refused", "cyber", "declined by policy"}},
		{"with no details", ``, []string{"refused"}},
		{"with a null category", `,"stop_details":{"type":"refusal","category":null}`, []string{"refused"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			d := (Claude{}).NewDecoder([]ce.ToolDef{{Name: "read"}})
			got := decodeFrames(t, d,
				`{"type":"message_start","message":{"usage":{"input_tokens":12,"output_tokens":1}}}`,
				`{"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"half an"}}`,
				`{"type":"content_block_start","index":1,"content_block":{"type":"tool_use","id":"c1","name":"read"}}`,
				`{"type":"content_block_delta","index":1,"delta":{"type":"input_json_delta","partial_json":"{\"path\":\"a.go\"}"}}`,
				`{"type":"message_delta","delta":{"stop_reason":"refusal"`+tc.details+`},"usage":{"output_tokens":7}}`,
				`{"type":"message_stop"}`,
			)

			var terminal []StreamEvent
			for _, ev := range got {
				switch ev.Type {
				case EventDone, EventError:
					terminal = append(terminal, ev)
				case EventToolCall:
					t.Errorf("a refused answer released a tool call: %+v", ev.ToolCall)
				}
			}
			if len(terminal) != 1 {
				t.Fatalf("terminal events = %+v, want exactly one", terminal)
			}
			ev := terminal[0]
			if ev.Type != EventError || ev.Err == nil {
				t.Fatalf("a refusal ended as %s, want an error", ev.Type)
			}
			if ev.Err.Class != ErrClassRefused {
				t.Errorf("class = %s, want %s", ev.Err.Class, ErrClassRefused)
			}
			if ev.Err.Retryable || Decide(ev.Err) != DecisionAbort {
				t.Errorf("a refusal must never be retried: retryable=%v decision=%s", ev.Err.Retryable, Decide(ev.Err))
			}
			for _, w := range tc.want {
				if !strings.Contains(ev.Err.Message, w) {
					t.Errorf("message %q does not say %q", ev.Err.Message, w)
				}
			}
			if ev.Usage == nil || ev.Usage.OutputTokens != 7 || ev.Usage.InputTokens != 12 {
				t.Errorf("usage = %+v, want the billed 12 in and 7 out", ev.Usage)
			}
		})
	}
}
