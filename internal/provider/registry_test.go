package provider

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	ce "github.com/aguinelo/dcode/internal/contextengine"
)

func fullRegistry(t *testing.T, frames ...string) *Registry {
	t.Helper()
	r := NewRegistry()
	r.RegisterTransport(NewReplayTransport(TransportOpenAI, Transcript{Frames: frames}))
	r.RegisterTransport(NewReplayTransport(TransportAnthropic, Transcript{Frames: frames}))
	for _, f := range []Family{MiniMaxM3{}, Claude{}} {
		if err := r.RegisterFamily(f); err != nil {
			t.Fatal(err)
		}
	}
	return r
}

// An unknown model must fail with the available families named. Falling back to
// a generic family would ship unmeasured tool-calling with no signal at all.
func TestUnknownModelFailsAndListsFamilies(t *testing.T) {
	r := fullRegistry(t)
	_, err := r.Resolve("gpt-9-turbo", "")
	if err == nil {
		t.Fatal("an unknown model must not resolve")
	}
	for _, want := range []string{"minimax-m3", "claude"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error should name family %q: %v", want, err)
		}
	}
}

func TestResolvePicksTheFamilyPreferredTransport(t *testing.T) {
	r := fullRegistry(t)
	p, err := r.Resolve("MiniMax-M3", "")
	if err != nil {
		t.Fatal(err)
	}
	if got := p.Transport().Name(); got != TransportOpenAI {
		t.Errorf("m3 prefers the openai dialect, got %q", got)
	}
	if got := p.Family().Name(); got != "minimax-m3" {
		t.Errorf("family: got %q", got)
	}
}

func TestTransportOverrideIsHonouredAndValidated(t *testing.T) {
	r := fullRegistry(t)

	p, err := r.Resolve("MiniMax-M3", TransportAnthropic)
	if err != nil {
		t.Fatal(err)
	}
	if got := p.Transport().Name(); got != TransportAnthropic {
		t.Errorf("override ignored, got %q", got)
	}

	// A dialect the family does not speak must fail naming what it does speak,
	// so the user can fix it without reading the source.
	_, err = r.Resolve("claude-sonnet", TransportOpenAI)
	if err == nil {
		t.Fatal("claude does not speak openai; this must fail")
	}
	if !strings.Contains(err.Error(), TransportAnthropic) {
		t.Errorf("error should name the compatible dialects: %v", err)
	}
}

func TestUnregisteredTransportFails(t *testing.T) {
	r := NewRegistry()
	if err := r.RegisterFamily(MiniMaxM3{}); err != nil {
		t.Fatal(err)
	}
	if _, err := r.Resolve("MiniMax-M3", ""); err == nil {
		t.Error("resolving with no transport registered must fail")
	}
}

// Overlapping prefixes would silently resolve to whichever family was added
// first, which is exactly the kind of ambiguity that produces a model running
// under another model's measured thresholds.
func TestOverlappingModelPrefixesAreRejected(t *testing.T) {
	r := NewRegistry()
	if err := r.RegisterFamily(MiniMaxM3{}); err != nil {
		t.Fatal(err)
	}
	err := r.RegisterFamily(fakeFamily{name: "impostor", models: []string{"MiniMax"}})
	if err == nil {
		t.Fatal("an overlapping prefix must be rejected at registration")
	}
	if !strings.Contains(err.Error(), "overlap") {
		t.Errorf("the error should say what is wrong: %v", err)
	}
}

// A family claiming a prefix already taken cannot be registered at all, so
// resolution never has to disambiguate. Rejecting at registration beats
// resolving cleverly at runtime: the ambiguity is a configuration bug, and the
// place to report a configuration bug is at startup.
func TestNestedPrefixIsAlsoRejected(t *testing.T) {
	r := NewRegistry()
	if err := r.RegisterFamily(fakeFamily{name: "broad", models: []string{"acme-"}}); err != nil {
		t.Fatal(err)
	}
	if err := r.RegisterFamily(fakeFamily{name: "specific", models: []string{"acme-pro-"}}); err == nil {
		t.Fatal("a prefix nested inside an existing one is still ambiguous and must be rejected")
	}
}

// The escape hatch: an unsupported model can still run under a named family,
// and the caller is expected to warn that no thresholds were measured for it.
func TestResolveFamilyBypassesPrefixMatching(t *testing.T) {
	r := fullRegistry(t)
	p, err := r.ResolveFamily("minimax-m3", "")
	if err != nil {
		t.Fatal(err)
	}
	if p.Family().Name() != "minimax-m3" {
		t.Errorf("got %q", p.Family().Name())
	}
	if _, err := r.ResolveFamily("nope", ""); err == nil {
		t.Error("an unknown family name must fail")
	}
	if _, err := r.ResolveFamily("claude", TransportOpenAI); err == nil {
		t.Error("an incompatible dialect must fail even on the explicit path")
	}
}

func TestFamilyWithoutTransportFails(t *testing.T) {
	r := NewRegistry()
	if err := r.RegisterFamily(fakeFamily{name: "broken", models: []string{"broken-"}}); err != nil {
		t.Fatal(err)
	}
	if _, err := r.Resolve("broken-1", ""); err == nil {
		t.Error("a family declaring no transport must fail loudly")
	}
	if _, err := r.ResolveFamily("broken", ""); err == nil {
		t.Error("same on the explicit path")
	}
}

func TestLimitsComeFromTheFamily(t *testing.T) {
	r := fullRegistry(t)
	m3, _ := r.Resolve("MiniMax-M3", "")
	cl, _ := r.Resolve("claude-sonnet", "")

	// The whole point of per-family limits: a long-horizon model must not be
	// capped by a number sized for a short-task one.
	if m3.Limits().MaxIterations <= cl.Limits().MaxIterations {
		t.Errorf("m3 should allow more iterations than claude: %d vs %d",
			m3.Limits().MaxIterations, cl.Limits().MaxIterations)
	}
	if w, err := m3.Window("MiniMax-M3"); err != nil || w <= 0 {
		t.Errorf("window: %d %v", w, err)
	}
}

func TestRegistryNamesAreSorted(t *testing.T) {
	r := fullRegistry(t)
	if got := r.FamilyNames(); len(got) != 2 || got[0] != "claude" {
		t.Errorf("families should be sorted for a stable error message: %v", got)
	}
	if got := r.TransportNames(); len(got) != 2 || got[0] != TransportAnthropic {
		t.Errorf("transports should be sorted: %v", got)
	}
}

func TestAnthropicDecode(t *testing.T) {
	for _, tc := range []struct {
		name  string
		frame string
		want  StreamEventType
	}{
		{"text delta", `{"type":"content_block_delta","delta":{"type":"text_delta","text":"hi"}}`, EventTextDelta},
		{"thinking", `{"type":"content_block_delta","delta":{"type":"thinking_delta","thinking":"hmm"}}`, EventReasoningDelta},
		// A tool block only *opens* here; its arguments are still to come, so
		// nothing is emitted until the message ends.
		{"tool use opens", `{"type":"content_block_start","content_block":{"type":"tool_use","id":"c1","name":"read","input":{}}}`, ""},
		{"message stop", `{"type":"message_stop"}`, EventDone},
		{"ping is ignored", `{"type":"ping"}`, ""},
		{"empty frame", ``, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dec := Claude{}.NewDecoder(tools())
			got, err := dec.Decode(WireEvent{Data: []byte(tc.frame)})
			if err != nil {
				t.Fatal(err)
			}
			if tc.want == "" {
				if len(got) != 0 {
					t.Errorf("want nothing, got %+v", got)
				}
				return
			}
			if len(got) != 1 || got[0].Type != tc.want {
				t.Errorf("got %+v want %q", got, tc.want)
			}
		})
	}
}

// The Anthropic dialect splits a call the same way, in its own shape: the block
// opens empty and the arguments follow as input_json_delta.
func TestAnthropicToolCallIsAssembledFromItsFragments(t *testing.T) {
	dec := Claude{}.NewDecoder(tools())
	var out []StreamEvent
	for _, frame := range []string{
		`{"type":"content_block_start","index":0,"content_block":{"type":"tool_use","id":"c1","name":"read","input":{}}}`,
		`{"type":"content_block_delta","index":0,"delta":{"type":"input_json_delta","partial_json":"{\"path\":"}}`,
		`{"type":"content_block_delta","index":0,"delta":{"type":"input_json_delta","partial_json":"\"a.go\"}"}}`,
		`{"type":"content_block_stop","index":0}`,
		`{"type":"message_stop"}`,
	} {
		got, err := dec.Decode(WireEvent{Data: []byte(frame)})
		if err != nil {
			t.Fatal(err)
		}
		out = append(out, got...)
	}

	var calls []*ce.ToolCall
	for _, ev := range out {
		if ev.Type == EventToolCall {
			calls = append(calls, ev.ToolCall)
		}
	}
	if len(calls) != 1 {
		t.Fatalf("want one call, got %d", len(calls))
	}
	if !strings.Contains(string(calls[0].Input), "a.go") {
		t.Errorf("the arguments were lost between frames: %s", calls[0].Input)
	}
}

func TestAnthropicUsageCarriesCacheTokens(t *testing.T) {
	got, err := Claude{}.NewDecoder(tools()).Decode(WireEvent{Data: []byte(
		`{"type":"message_delta","usage":{"input_tokens":100,"output_tokens":9,"cache_read_input_tokens":80}}`)})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Usage == nil || got[0].Usage.CacheReadTokens != 80 {
		t.Errorf("cache read tokens must survive decoding: %+v", got)
	}
}

func TestMalformedFrameIsAProviderError(t *testing.T) {
	for _, fam := range []Family{MiniMaxM3{}, Claude{}} {
		_, err := fam.NewDecoder(tools()).Decode(WireEvent{Data: []byte(`{"broken`)})
		if err == nil {
			t.Fatalf("%s: a malformed frame must be an error", fam.Name())
		}
		if pe := classify(context.Background(), err); pe.Class != ErrClassProvider {
			t.Errorf("%s: got class %s", fam.Name(), pe.Class)
		}
	}
}

func TestDecideMapsEveryClass(t *testing.T) {
	for _, tc := range []struct {
		class ErrorClass
		want  Decision
	}{
		{ErrClassAuth, DecisionAbort},
		{ErrClassQuota, DecisionAbort},
		{ErrClassBadRequest, DecisionAbort},
		{ErrClassRateLimit, DecisionWait},
		{ErrClassContextSize, DecisionCompact},
		{ErrClassToolSchema, DecisionFeedToolIn},
		{ErrClassTransport, DecisionRetry},
		{ErrClassProvider, DecisionRetry},
		{ErrClassCanceled, DecisionSilent},
	} {
		if got := Decide(&ProviderError{Class: tc.class}); got != tc.want {
			t.Errorf("%s: got %s want %s", tc.class, got, tc.want)
		}
	}
	// A nil error reaching Decide is a bug upstream; abort is the safe reading.
	if got := Decide(nil); got != DecisionAbort {
		t.Errorf("nil: got %s", got)
	}
}

func TestClassifyStatus(t *testing.T) {
	for _, tc := range []struct {
		status int
		want   ErrorClass
		retry  bool
	}{
		{401, ErrClassAuth, false},
		{403, ErrClassAuth, false},
		{402, ErrClassQuota, false},
		{429, ErrClassRateLimit, true},
		{413, ErrClassContextSize, true},
		{400, ErrClassBadRequest, false},
		{500, ErrClassProvider, true},
		{503, ErrClassProvider, true},
	} {
		got := ClassifyStatus(tc.status, "body", "")
		if got == nil || got.Class != tc.want {
			t.Errorf("%d: got %+v want %s", tc.status, got, tc.want)
			continue
		}
		if got.Retryable != tc.retry {
			t.Errorf("%d: retryable got %v want %v", tc.status, got.Retryable, tc.retry)
		}
	}
	if ClassifyStatus(200, "", "") != nil {
		t.Error("a success status is not an error")
	}
}

// A credential in an error message is the leak that only gets found after
// publication, so it gets a dedicated sweep with a sentinel value.
func TestCredentialsNeverAppearInErrorMessages(t *testing.T) {
	const sentinel = "sk-SENTINEL-DO-NOT-LEAK-0123456789"
	RegisterSecret(sentinel)
	t.Cleanup(ClearSecrets)

	for _, tc := range []struct {
		name string
		body string
	}{
		{"echoed key", "invalid api key: " + sentinel},
		{"authorization header", "request failed: Authorization: Bearer " + sentinel},
		{"json body", `{"error":{"message":"bad key ` + sentinel + `"}}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			pe := ClassifyStatus(400, tc.body, "")
			if strings.Contains(pe.Error(), sentinel) {
				t.Errorf("credential leaked: %s", pe.Error())
			}
			if !strings.Contains(pe.Error(), "redacted") {
				t.Errorf("redaction should be visible, got %q", pe.Error())
			}
		})
	}
}

// Keys the harness never registered still get caught by shape, because a
// provider that echoes one back would otherwise leak it.
func TestUnregisteredCredentialShapesAreRedacted(t *testing.T) {
	for _, in := range []string{
		"Authorization: Bearer abcdefghijklmnop123456",
		`{"api_key": "abcdefghijklmnop123456"}`,
		"failed with sk-abcdefghijklmnop123456",
	} {
		if got := Sanitize(in); !strings.Contains(got, "redacted") {
			t.Errorf("credential-shaped text not redacted: %q -> %q", in, got)
		}
	}
}

func TestShortSecretsAreNotRegistered(t *testing.T) {
	t.Cleanup(ClearSecrets)
	RegisterSecret("abc")
	// Redacting a three-character value would blank out unrelated text
	// everywhere it happened to appear.
	if got := Sanitize("abc is a common substring"); strings.Contains(got, "redacted") {
		t.Errorf("a short value must not be treated as a secret: %q", got)
	}
}

func TestRegisterSecretIsIdempotent(t *testing.T) {
	t.Cleanup(ClearSecrets)
	const s = "sk-duplicate-value-12345"
	RegisterSecret(s)
	RegisterSecret(s)
	if got := Sanitize("key " + s); strings.Count(got, "redacted") != 1 {
		t.Errorf("got %q", got)
	}
}

func TestSanitizeEmptyString(t *testing.T) {
	if Sanitize("") != "" {
		t.Error("empty in, empty out")
	}
}

// Determinism is the reason transcripts exist: the same fixture must produce
// the same events every run, or nothing downstream can be golden-tested.
func TestReplayIsDeterministic(t *testing.T) {
	frames := []string{
		`{"choices":[{"delta":{"content":"one"}}]}`,
		`{"choices":[{"delta":{"content":"two"}}]}`,
		"[DONE]",
	}
	var first []StreamEventType
	for run := 0; run < 20; run++ {
		r, _ := registry(t, frames...)
		p, _ := r.Resolve("MiniMax-M3", "")
		got := kinds(drain(t, mustStream(t, p)))
		if run == 0 {
			first = got
			continue
		}
		if len(got) != len(first) {
			t.Fatalf("run %d differs: %v vs %v", run, got, first)
		}
		for i := range got {
			if got[i] != first[i] {
				t.Fatalf("run %d differs at %d: %v vs %v", run, i, got, first)
			}
		}
	}
}

func TestReplayRecordsWhatWasSent(t *testing.T) {
	rt := NewReplayTransport(TransportOpenAI, Transcript{Frames: []string{"[DONE]"}})
	r := NewRegistry()
	r.RegisterTransport(rt)
	if err := r.RegisterFamily(MiniMaxM3{}); err != nil {
		t.Fatal(err)
	}
	p, _ := r.Resolve("MiniMax-M3", "")
	drain(t, mustStream(t, p))

	if len(rt.Sent) != 1 {
		t.Fatalf("want 1 recorded request, got %d", len(rt.Sent))
	}
	if !strings.Contains(string(rt.Sent[0].Body), "list the files") {
		t.Errorf("the encoded body should carry the user message: %s", rt.Sent[0].Body)
	}
}

func TestReplayCanReproduceAFailure(t *testing.T) {
	for _, tc := range []struct {
		name string
		tr   Transcript
		want ErrorClass
	}{
		{"by status", Transcript{FailWith: &RecordedError{Status: 429}}, ErrClassRateLimit},
		{"by class", Transcript{FailWith: &RecordedError{Class: string(ErrClassQuota), Body: "no credit"}}, ErrClassQuota},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := NewRegistry()
			r.RegisterTransport(NewReplayTransport(TransportOpenAI, tc.tr))
			if err := r.RegisterFamily(MiniMaxM3{}); err != nil {
				t.Fatal(err)
			}
			p, _ := r.Resolve("MiniMax-M3", "")
			_, err := p.Stream(context.Background(), request())
			if err == nil {
				t.Fatal("the recorded failure should surface")
			}
			if pe := classify(context.Background(), err); pe.Class != tc.want {
				t.Errorf("got %s want %s", pe.Class, tc.want)
			}
		})
	}
}

func TestLoadTranscript(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "t.json")
	want := Transcript{Name: "sample", Transport: TransportOpenAI, Frames: []string{"[DONE]"}}
	b, _ := json.Marshal(want)
	if err := os.WriteFile(path, b, 0o600); err != nil {
		t.Fatal(err)
	}

	got, err := LoadTranscript(path)
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != want.Name || len(got.Frames) != 1 {
		t.Errorf("got %+v", got)
	}
	if _, err := LoadTranscript(filepath.Join(dir, "missing.json")); err == nil {
		t.Error("a missing file must fail")
	}
	if err := os.WriteFile(path, []byte("{not json"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadTranscript(path); err == nil {
		t.Error("malformed JSON must fail")
	}
}

func TestParseSSE(t *testing.T) {
	body := "event: message\ndata: {\"a\":1}\n\ndata: [DONE]\n\n: heartbeat\n"
	got := ParseSSE(body)
	if len(got) != 2 || got[0] != `{"a":1}` || got[1] != "[DONE]" {
		t.Errorf("got %q", got)
	}
}

func TestEncodeFailureSurfacesBeforeAnyRequest(t *testing.T) {
	r := NewRegistry()
	rt := NewReplayTransport("grpc", Transcript{})
	r.RegisterTransport(rt)
	if err := r.RegisterFamily(fakeFamily{name: "f", models: []string{"f-"}, transports: []string{"grpc"}, failEncode: true}); err != nil {
		t.Fatal(err)
	}
	p, err := r.Resolve("f-1", "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := p.Stream(context.Background(), Request{Model: "f-1"}); err == nil {
		t.Fatal("an encode failure must surface")
	}
	if len(rt.Sent) != 0 {
		t.Error("nothing should reach the transport when encoding failed")
	}
}

// fakeFamily exists to exercise registry paths the real families cannot reach.
type fakeFamily struct {
	name       string
	models     []string
	transports []string
	failEncode bool
}

func (f fakeFamily) Name() string               { return f.name }
func (f fakeFamily) Transports() []string       { return f.transports }
func (f fakeFamily) Models() []string           { return f.models }
func (f fakeFamily) Window(string) (int, error) { return 1000, nil }
func (f fakeFamily) DefaultLimits() Limits      { return Limits{MaxIterations: 10} }
func (f fakeFamily) Encode(Request, string) (WireRequest, error) {
	if f.failEncode {
		return WireRequest{}, context.DeadlineExceeded
	}
	return WireRequest{Body: json.RawMessage(`{}`)}, nil
}
func (f fakeFamily) NewDecoder([]ce.ToolDef) Decoder { return fakeDecoder{} }

type fakeDecoder struct{}

func (fakeDecoder) Decode(WireEvent) ([]StreamEvent, error) {
	return []StreamEvent{{Type: EventDone}}, nil
}

func (fakeDecoder) Close() []StreamEvent { return nil }

func TestCanceledEventReportsDeadline(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Nanosecond)
	defer cancel()
	<-ctx.Done()
	ev := canceledEvent(ctx)
	if ev.Err.Class != ErrClassCanceled || !strings.Contains(ev.Err.Message, "deadline") {
		t.Errorf("got %+v", ev.Err)
	}
}
