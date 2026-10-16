package provider

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	ce "github.com/aguinelo/dcode/internal/contextengine"
)

// cacheRequest is a session one tool round in: the shape every request of an
// agent loop has, and the one the cache has to pay off on.
func cacheRequest(model string) Request {
	return Request{
		Model: model,
		Messages: []ce.Message{
			{Role: ce.RoleSystem, Text: "be brief"},
			{Role: ce.RoleUser, Text: "read a.txt"},
			{Role: ce.RoleAssistant, ToolCalls: []ce.ToolCall{
				{ID: "c1", Name: "read", Input: json.RawMessage(`{"path":"a.txt"}`)},
			}},
			{Role: ce.RoleTool, ToolResult: &ce.ToolResult{ToolCallID: "c1", Output: "hello"}},
		},
		Tools: []ce.ToolDef{
			{Name: "read", Description: "read a file", Schema: json.RawMessage(`{"type":"object"}`)},
			{Name: "write", Description: "write a file", Schema: json.RawMessage(`{"type":"object"}`)},
		},
	}
}

// A Claude request marks three cache breakpoints: the end of the tools, the
// system prompt, and the last block of the conversation. Without them every
// round paid full price for a prefix the provider had just seen, and the
// append-only context engine was buying nothing on this family.
//
// Golden, because what is being promised is the exact bytes: a marker one
// block off caches a prefix nobody re-sends.
func TestTheClaudeRequestMarksTheCacheBreakpoints(t *testing.T) {
	wire, err := Claude{}.Encode(cacheRequest("claude-sonnet-4-5"), TransportAnthropic)
	if err != nil {
		t.Fatal(err)
	}
	var got bytes.Buffer
	if err := json.Indent(&got, wire.Body, "", "  "); err != nil {
		t.Fatal(err)
	}
	want, err := os.ReadFile(filepath.Join("testdata", "anthropic-cache-request.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(bytes.TrimSpace(got.Bytes()), bytes.TrimSpace(want)) {
		t.Errorf("encoded body differs from the golden file.\ngot:\n%s\nwant:\n%s", got.String(), want)
	}
}

// The API refuses a request with more than four breakpoints, whatever the
// conversation looks like.
func TestTheClaudeRequestNeverMarksMoreThanFourBreakpoints(t *testing.T) {
	req := cacheRequest("claude-opus-4-6")
	for i := 0; i < 30; i++ {
		req.Messages = append(req.Messages,
			ce.Message{Role: ce.RoleUser, Text: "more"},
			ce.Message{Role: ce.RoleAssistant, Text: "ok"})
	}
	wire, err := Claude{}.Encode(req, TransportAnthropic)
	if err != nil {
		t.Fatal(err)
	}
	n := strings.Count(string(wire.Body), `"cache_control"`)
	if n == 0 || n > 4 {
		t.Errorf("%d breakpoints, want between 1 and 4", n)
	}
}

// MiniMax M3 speaks the same dialect through someone else's endpoint, and
// nothing says that endpoint reads the marker. The family that sends it is the
// one that measured it.
func TestMiniMaxOnTheAnthropicDialectSendsNoCacheMarkers(t *testing.T) {
	wire, err := MiniMaxM3{}.Encode(cacheRequest("MiniMax-M3"), TransportAnthropic)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(wire.Body), "cache_control") {
		t.Errorf("M3 request carries a cache marker: %s", wire.Body)
	}
}

// The prompt side of the usage arrives on message_start — the input, and what
// the cache read and wrote — and only the output on message_delta. Reading the
// delta alone left every cache number at zero.
func TestCacheUsageFromMessageStartReachesDone(t *testing.T) {
	d := (Claude{}).NewDecoder(nil)
	got := decodeFrames(t, d,
		`{"type":"message_start","message":{"usage":{"input_tokens":12,"cache_creation_input_tokens":3000,"cache_read_input_tokens":40000,"output_tokens":1}}}`,
		`{"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"hi"}}`,
		`{"type":"message_delta","delta":{"stop_reason":"end_turn"},"usage":{"output_tokens":9}}`,
	)
	var u *Usage
	for _, ev := range got {
		if ev.Type == EventDone {
			u = ev.Usage
		}
	}
	if u == nil {
		t.Fatal("no usage on the terminal event")
	}
	want := Usage{InputTokens: 12, OutputTokens: 9, CacheReadTokens: 40000, CacheWriteTokens: 3000}
	if *u != want {
		t.Errorf("usage = %+v, want %+v", *u, want)
	}
}
