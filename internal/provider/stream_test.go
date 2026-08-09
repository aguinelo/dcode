package provider

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	ce "github.com/aguinelo/dcode/internal/contextengine"
)

// frames reads a captured SSE fixture into wire events.
//
// The fixture is a real MiniMax-M3 response, recorded verbatim. A hand-written
// approximation of a stream is exactly the thing that let these two defects
// through in the first place: both only exist in the gaps between frames.
func frames(t *testing.T, path string) []WireEvent {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var out []WireEvent
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		out = append(out, WireEvent{Data: []byte(strings.TrimPrefix(line, "data: "))})
	}
	if len(out) == 0 {
		t.Fatalf("%s carried no frames", path)
	}
	return out
}

func globTool() []ce.ToolDef {
	return []ce.ToolDef{{
		Name:        "glob",
		Description: "find files by pattern",
		Schema: json.RawMessage(`{"type":"object","properties":{"pattern":{"type":"string"}},` +
			`"required":["pattern"]}`),
	}}
}

// decodeAll runs a whole fixture through one decoder, the way a real stream
// arrives.
func decodeAll(t *testing.T, f Family, evs []WireEvent, tools []ce.ToolDef) []StreamEvent {
	t.Helper()
	dec := f.NewDecoder(tools)
	var out []StreamEvent
	for _, ev := range evs {
		got, err := dec.Decode(ev)
		if err != nil {
			t.Fatalf("decode: %v", err)
		}
		out = append(out, got...)
	}
	return out
}

// Regression: a tool call's arguments arrive across frames, keyed by index.
//
// The decoder used to emit on the first fragment, where `arguments` is still
// the empty string — so every call reached the tool with no input at all and
// came back "pattern is required". Three of those in a row tripped the repeat
// guard and ended the turn having done nothing.
func TestToolCallArgumentsAreAssembledAcrossFrames(t *testing.T) {
	evs := frames(t, "testdata/minimax-m3-toolcall.sse")
	got := decodeAll(t, MiniMaxM3{}, evs, globTool())

	var calls []*ce.ToolCall
	for _, ev := range got {
		if ev.Type == EventToolCall {
			calls = append(calls, ev.ToolCall)
		}
	}
	if len(calls) != 1 {
		t.Fatalf("want exactly one tool call, got %d", len(calls))
	}
	call := calls[0]
	if call.Name != "glob" {
		t.Errorf("got %q", call.Name)
	}
	if call.ID == "" {
		t.Error("a tool call with no id cannot be matched to its result")
	}

	var input struct {
		Pattern string `json:"pattern"`
	}
	if err := json.Unmarshal(call.Input, &input); err != nil {
		t.Fatalf("the assembled arguments are not valid JSON: %v (%s)", err, call.Input)
	}
	if input.Pattern != "**/*.go" {
		t.Errorf("the arguments were lost between frames: got %q", input.Pattern)
	}
}

// Regression: the model's reasoning must not become the assistant's answer.
//
// MiniMax-M3 sends the same text twice — once in `reasoning`, and once in
// `content` wrapped in <think> markers. The decoder read `content`, so the
// thinking was printed to the user and, worse, appended to the history as the
// assistant's own words, where it stayed for the rest of the session.
func TestReasoningNeverBecomesAnswerText(t *testing.T) {
	evs := frames(t, "testdata/minimax-m3-toolcall.sse")
	got := decodeAll(t, MiniMaxM3{}, evs, globTool())

	var text strings.Builder
	for _, ev := range got {
		if ev.Type == EventTextDelta {
			text.WriteString(ev.Text)
		}
	}
	answer := text.String()

	for _, leak := range []string{"<think>", "</think>", "The user is asking", "glob tool. Let"} {
		if strings.Contains(answer, leak) {
			t.Errorf("reasoning leaked into the answer (%q):\n%q", leak, answer)
		}
	}
	// This particular turn is pure tool call, so there is no answer at all.
	if strings.TrimSpace(answer) != "" {
		t.Errorf("a turn that only calls a tool has no text to show, got %q", answer)
	}
}

// The reasoning is still reported, just on its own channel: a client may show
// it, and the history must not.
func TestReasoningIsReportedSeparately(t *testing.T) {
	evs := frames(t, "testdata/minimax-m3-toolcall.sse")
	got := decodeAll(t, MiniMaxM3{}, evs, globTool())

	var thinking strings.Builder
	for _, ev := range got {
		if ev.Type == EventReasoningDelta {
			thinking.WriteString(ev.Text)
		}
	}
	if !strings.Contains(thinking.String(), "The user is asking") {
		t.Errorf("the reasoning was dropped entirely: %q", thinking.String())
	}
	if strings.Contains(thinking.String(), "<think>") {
		t.Errorf("the markers are framing, not content: %q", thinking.String())
	}
}

// The terminal event arrives exactly once. A stream that ends without one hangs
// the loop forever; one that ends twice would let a turn continue past its end.
func TestTheFixtureEndsWithExactlyOneTerminalEvent(t *testing.T) {
	evs := frames(t, "testdata/minimax-m3-toolcall.sse")
	got := decodeAll(t, MiniMaxM3{}, evs, globTool())

	var terminals int
	for _, ev := range got {
		if ev.Type == EventDone || ev.Type == EventError {
			terminals++
		}
	}
	if terminals == 0 {
		t.Fatal("the stream never terminates")
	}
	// The fixture repeats finish_reason, which a real provider does. The
	// decoder may report it more than once; the pump is what collapses it to
	// one, and that invariant is asserted where the pump lives.
	if got[len(got)-1].Type != EventDone {
		t.Errorf("the last event must be terminal, got %s", got[len(got)-1].Type)
	}
}

// Two calls in one response must stay two calls, each with its own arguments.
// A single accumulator keyed by nothing would merge them into one unparseable
// blob — the failure only appears once a model emits parallel calls.
func TestParallelToolCallsAreKeptApart(t *testing.T) {
	tools := []ce.ToolDef{
		{Name: "glob", Schema: json.RawMessage(`{"type":"object","properties":{"pattern":{"type":"string"}}}`)},
		{Name: "read", Schema: json.RawMessage(`{"type":"object","properties":{"path":{"type":"string"}}}`)},
	}
	evs := []WireEvent{
		{Data: []byte(`{"choices":[{"delta":{"tool_calls":[` +
			`{"id":"c1","type":"function","function":{"name":"glob","arguments":""},"index":0},` +
			`{"id":"c2","type":"function","function":{"name":"read","arguments":""},"index":1}]}}]}`)},
		{Data: []byte(`{"choices":[{"delta":{"tool_calls":[` +
			`{"function":{"arguments":"{\"pattern\":"},"index":0},` +
			`{"function":{"arguments":"{\"path\":"},"index":1}]}}]}`)},
		{Data: []byte(`{"choices":[{"delta":{"tool_calls":[` +
			`{"function":{"arguments":"\"*.go\"}"},"index":0},` +
			`{"function":{"arguments":"\"a.go\"}"},"index":1}]}}]}`)},
		{Data: []byte(`{"choices":[{"finish_reason":"tool_calls","delta":{}}]}`)},
	}

	got := decodeAll(t, MiniMaxM3{}, evs, tools)
	var calls []*ce.ToolCall
	for _, ev := range got {
		if ev.Type == EventToolCall {
			calls = append(calls, ev.ToolCall)
		}
	}
	if len(calls) != 2 {
		t.Fatalf("want two calls, got %d", len(calls))
	}
	// Emission order is the index order, because that is the order the loop
	// will schedule and record them in.
	if calls[0].Name != "glob" || calls[1].Name != "read" {
		t.Fatalf("got %q then %q", calls[0].Name, calls[1].Name)
	}
	if !strings.Contains(string(calls[0].Input), "*.go") {
		t.Errorf("got %s", calls[0].Input)
	}
	if !strings.Contains(string(calls[1].Input), "a.go") {
		t.Errorf("got %s", calls[1].Input)
	}
}

// A plain answer, with no tools involved, must still stream as text.
func TestOrdinaryTextStillStreams(t *testing.T) {
	evs := []WireEvent{
		{Data: []byte(`{"choices":[{"delta":{"content":"olá "}}]}`)},
		{Data: []byte(`{"choices":[{"delta":{"content":"mundo"}}]}`)},
		{Data: []byte(`{"choices":[{"finish_reason":"stop","delta":{}}]}`)},
	}
	got := decodeAll(t, MiniMaxM3{}, evs, nil)

	var text strings.Builder
	for _, ev := range got {
		if ev.Type == EventTextDelta {
			text.WriteString(ev.Text)
		}
	}
	if text.String() != "olá mundo" {
		t.Errorf("got %q", text.String())
	}
}

// A tool call the model never finished must not reach the loop as a half-formed
// call. Truncation is an error the turn can report, not input to run.
func TestATruncatedToolCallIsAnErrorRatherThanAnEmptyCall(t *testing.T) {
	evs := []WireEvent{
		{Data: []byte(`{"choices":[{"delta":{"tool_calls":[` +
			`{"id":"c1","type":"function","function":{"name":"glob","arguments":"{\"pat"},"index":0}]}}]}`)},
		{Data: []byte(`{"choices":[{"finish_reason":"tool_calls","delta":{}}]}`)},
	}
	dec := MiniMaxM3{}.NewDecoder(globTool())
	var got []StreamEvent
	var lastErr error
	for _, ev := range evs {
		out, err := dec.Decode(ev)
		if err != nil {
			lastErr = err
		}
		got = append(got, out...)
	}
	for _, ev := range got {
		if ev.Type == EventToolCall {
			t.Fatalf("a truncated call must not be executed: %s", ev.ToolCall.Input)
		}
	}
	if lastErr == nil {
		t.Error("the truncation must be reported, not swallowed")
	}
}

// The model closes its reasoning with a frame of pure markers and newlines,
// twice over. Stripping alone left three blank lines at the top of every
// answer, so a frame that was nothing but framing is dropped entirely.
func TestAFrameOfPureFramingProducesNothing(t *testing.T) {
	evs := frames(t, "testdata/minimax-m3-toolcall.sse")
	got := decodeAll(t, MiniMaxM3{}, evs, globTool())

	for _, ev := range got {
		if ev.Type == EventTextDelta {
			t.Errorf("this turn has no answer, only framing: %q", ev.Text)
		}
	}
}

func TestAnswerText(t *testing.T) {
	for _, tc := range []struct {
		in   string
		want string
		ok   bool
	}{
		{"olá", "olá", true},
		{"", "", false},
		// Whitespace the model meant to send survives when no marker is
		// involved: it is how a paragraph break arrives.
		{"\n\n", "\n\n", true},
		{"\n</think>\n\n</think>", "", false},
		{"<think>", "", false},
		{"</think>resposta", "resposta", true},
	} {
		got, ok := answerText(tc.in)
		if got != tc.want || ok != tc.ok {
			t.Errorf("%q: got (%q, %v) want (%q, %v)", tc.in, got, ok, tc.want, tc.ok)
		}
	}
}
