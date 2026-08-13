package app

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/aguinelo/dcode/internal/config"
	ce "github.com/aguinelo/dcode/internal/contextengine"
	"github.com/aguinelo/dcode/internal/policy"
	"github.com/aguinelo/dcode/internal/provider"
	"github.com/aguinelo/dcode/internal/sandbox"
)

func baseOpts(t *testing.T) Options {
	t.Helper()
	return Options{
		Workspace:   t.TempDir(),
		Model:       "MiniMax-M3",
		APIKey:      "sk-test-key-abcdefghijklmnop",
		SandboxMode: policy.ModeWorkspaceWrite,
		Policy:      policy.PolicyOnRequest,
		Backend:     "auto",
		Parallel:    4,
		// Options carries no defaults of its own — the configuration chain
		// supplies them — so a feature switch has to be set explicitly here.
		Delegate: true,
	}
}

func TestNewWiresAWorkingSession(t *testing.T) {
	t.Cleanup(provider.ClearSecrets)
	opts := baseOpts(t)

	s, err := New(opts, &ConsoleEmitter{W: &bytes.Buffer{}}, DenyAll{})
	if err != nil {
		if strings.Contains(err.Error(), "sandbox") {
			t.Skipf("no sandbox available: %v", err)
		}
		t.Fatal(err)
	}
	if s.Engine == nil || s.Registry == nil {
		t.Fatal("the session is not fully wired")
	}
	// Every tool, or the agent is missing a capability the doctrine already
	// promises the model it has. The names are asserted rather than the count:
	// a count tells you a tool went missing, not which one.
	want := []string{"bash", "edit", "explore", "glob", "grep", "plan", "process", "read", "symbol", "write"}
	got := s.Registry.Names()
	if !slices.Equal(got, want) {
		t.Errorf("tools = %v, want %v", got, want)
	}
	if s.Prompt == "" {
		t.Error("the prompt should be assembled at wiring time")
	}
}

// The credential is registered for redaction before anything can log it.
func TestNewRegistersTheKeyForRedaction(t *testing.T) {
	t.Cleanup(provider.ClearSecrets)
	const key = "sk-SENTINEL-abcdefghijklmnop"
	opts := baseOpts(t)
	opts.APIKey = key

	if _, err := New(opts, &ConsoleEmitter{W: &bytes.Buffer{}}, DenyAll{}); err != nil {
		if strings.Contains(err.Error(), "sandbox") {
			t.Skipf("no sandbox: %v", err)
		}
		t.Fatal(err)
	}
	if got := provider.Sanitize("failure with " + key); strings.Contains(got, key) {
		t.Errorf("the key must be redacted after wiring: %q", got)
	}
}

// A session that cannot confine its own commands must not start at all.
func TestNewFailsWhenTheSandboxCannotBeEstablished(t *testing.T) {
	opts := baseOpts(t)
	opts.Backend = "none" // legal only with full-access

	if _, err := New(opts, nil, nil); err == nil {
		t.Fatal("a mode claiming a boundary with no mechanism must be refused")
	}
}

func TestNewRejectsARelativeWorkspace(t *testing.T) {
	opts := baseOpts(t)
	opts.Workspace = "relative/path"
	if _, err := New(opts, nil, nil); err == nil {
		t.Error("a relative workspace must be rejected")
	}
}

func TestBuildProviderResolvesBothFamilies(t *testing.T) {
	for _, tc := range []struct {
		model     string
		wantFam   string
		transport string
	}{
		{"MiniMax-M3", "minimax-m3", provider.TransportOpenAI},
		{"claude-sonnet-4", "claude", provider.TransportAnthropic},
	} {
		opts := baseOpts(t)
		opts.Model = tc.model
		p, err := buildProvider(opts)
		if err != nil {
			t.Fatalf("%s: %v", tc.model, err)
		}
		if p.Family().Name() != tc.wantFam {
			t.Errorf("%s: got family %q want %q", tc.model, p.Family().Name(), tc.wantFam)
		}
		if p.Transport().Name() != tc.transport {
			t.Errorf("%s: got transport %q want %q", tc.model, p.Transport().Name(), tc.transport)
		}
	}
}

func TestBuildProviderHonoursATransportOverride(t *testing.T) {
	opts := baseOpts(t)
	opts.Transport = provider.TransportAnthropic
	p, err := buildProvider(opts)
	if err != nil {
		t.Fatal(err)
	}
	// M3 speaks both dialects, which is the reason the override exists.
	if p.Transport().Name() != provider.TransportAnthropic {
		t.Errorf("got %q", p.Transport().Name())
	}
	if p.Family().Name() != "minimax-m3" {
		t.Errorf("the family must not change with the dialect, got %q", p.Family().Name())
	}
}

func TestBuildProviderRejectsAnUnknownModel(t *testing.T) {
	opts := baseOpts(t)
	opts.Model = "gpt-9"
	if _, err := buildProvider(opts); err == nil {
		t.Error("an unknown model must not silently fall back to a family")
	}
}

func TestBuildProviderSupportsTheExplicitFamilyEscapeHatch(t *testing.T) {
	opts := baseOpts(t)
	opts.Model = "some-unknown-model"
	opts.Family = "minimax-m3"
	p, err := buildProvider(opts)
	if err != nil {
		t.Fatal(err)
	}
	if p.Family().Name() != "minimax-m3" {
		t.Errorf("got %q", p.Family().Name())
	}
}

func TestSummariserAsksTheModelAndReturnsText(t *testing.T) {
	rt := provider.NewReplayTransport(provider.TransportOpenAI, provider.Transcript{
		Frames: []string{frameText("We read main.go and changed a string."), "[DONE]"},
	})
	reg := provider.NewRegistry()
	reg.RegisterTransport(rt)
	if err := reg.RegisterFamily(provider.MiniMaxM3{}); err != nil {
		t.Fatal(err)
	}
	p, err := reg.Resolve("MiniMax-M3", "")
	if err != nil {
		t.Fatal(err)
	}

	got, err := summariser(p, "MiniMax-M3")(context.Background(), []ce.Message{
		{Role: ce.RoleUser, Text: "change the string"},
		{Role: ce.RoleAssistant, Text: "done"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "changed a string") {
		t.Errorf("got %q", got)
	}
	// The summariser asks for the task and decisions to survive, which is what
	// keeps a long session usable after compaction.
	if len(rt.Sent) != 1 {
		t.Fatalf("want one request, got %d", len(rt.Sent))
	}
	body := string(rt.Sent[0].Body)
	for _, want := range []string{"current task", "decisions"} {
		if !strings.Contains(strings.ToLower(body), want) {
			t.Errorf("the summarise instruction should mention %q: %s", want, body)
		}
	}
}

func TestSummariserSurfacesAFailure(t *testing.T) {
	rt := provider.NewReplayTransport(provider.TransportOpenAI, provider.Transcript{
		FailWith: &provider.RecordedError{Status: 500},
	})
	reg := provider.NewRegistry()
	reg.RegisterTransport(rt)
	if err := reg.RegisterFamily(provider.MiniMaxM3{}); err != nil {
		t.Fatal(err)
	}
	p, _ := reg.Resolve("MiniMax-M3", "")

	if _, err := summariser(p, "m")(context.Background(), nil); err == nil {
		t.Error("a failed summary must be reported so the caller can fall back")
	}
}

// ---------- HTTP transport ----------

func TestHTTPTransportParsesAnSSEStream(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer key-12345678" {
			t.Errorf("authorization header: %q", got)
		}
		w.Header().Set("Content-Type", "text/event-stream")
		for _, f := range []string{frameText("hello"), frameText(" world"), "[DONE]"} {
			w.Write([]byte("data: " + f + "\n\n"))
		}
	}))
	defer srv.Close()

	tr := NewHTTPTransport(provider.TransportOpenAI, srv.URL, "key-12345678")
	ch, err := tr.Do(context.Background(), provider.WireRequest{Body: []byte(`{}`)})
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for ev := range ch {
		if ev.Err != nil {
			t.Fatalf("stream error: %v", ev.Err)
		}
		got = append(got, string(ev.Data))
	}
	if len(got) != 3 || got[2] != "[DONE]" {
		t.Errorf("got %q", got)
	}
}

func TestHTTPTransportSetsTheAnthropicHeaders(t *testing.T) {
	var seenKey, seenVersion string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenKey = r.Header.Get("x-api-key")
		seenVersion = r.Header.Get("anthropic-version")
		w.Write([]byte("data: {}\n\n"))
	}))
	defer srv.Close()

	tr := NewHTTPTransport(provider.TransportAnthropic, srv.URL, "key-12345678")
	ch, err := tr.Do(context.Background(), provider.WireRequest{Body: []byte(`{}`)})
	if err != nil {
		t.Fatal(err)
	}
	for range ch {
	}
	// Each dialect authenticates its own way; that is transport concern, not
	// family concern.
	if seenKey != "key-12345678" || seenVersion == "" {
		t.Errorf("key=%q version=%q", seenKey, seenVersion)
	}
}

func TestHTTPTransportClassifiesAnErrorStatus(t *testing.T) {
	for _, tc := range []struct {
		status int
		want   provider.ErrorClass
	}{
		{401, provider.ErrClassAuth},
		{429, provider.ErrClassRateLimit},
		{500, provider.ErrClassProvider},
	} {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(tc.status)
			w.Write([]byte(`{"error":"nope"}`))
		}))
		tr := NewHTTPTransport(provider.TransportOpenAI, srv.URL, "key-12345678")
		_, err := tr.Do(context.Background(), provider.WireRequest{Body: []byte(`{}`)})
		srv.Close()

		if err == nil {
			t.Errorf("%d: expected an error", tc.status)
			continue
		}
		var pe *provider.ProviderError
		if !asProviderErr(err, &pe) || pe.Class != tc.want {
			t.Errorf("%d: got %v want class %s", tc.status, err, tc.want)
		}
	}
}

// A transport error can echo the URL, and a URL can carry a key.
func TestHTTPTransportSanitisesConnectionErrors(t *testing.T) {
	const key = "sk-LEAKME-abcdefghijklmnop"
	provider.RegisterSecret(key)
	t.Cleanup(provider.ClearSecrets)

	tr := NewHTTPTransport(provider.TransportOpenAI, "http://127.0.0.1:1/"+key, key)
	_, err := tr.Do(context.Background(), provider.WireRequest{Body: []byte(`{}`)})
	if err == nil {
		t.Fatal("connecting to a closed port must fail")
	}
	if strings.Contains(err.Error(), key) {
		t.Errorf("the credential leaked into a transport error: %v", err)
	}
}

func TestHTTPTransportName(t *testing.T) {
	if got := NewHTTPTransport(provider.TransportOpenAI, "", "k").Name(); got != provider.TransportOpenAI {
		t.Errorf("got %q", got)
	}
}

func TestLoadInstructionsPicksUpBothFileNames(t *testing.T) {
	ws := t.TempDir()
	if err := os.WriteFile(filepath.Join(ws, "AGENTS.md"), []byte("SHARED-RULE"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(ws, "DCODE.md"), []byte("SPECIFIC-RULE"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, _, err := loadInstructions(config.Roots{Config: filepath.Join(ws, ".absent")}, ws)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("want both files, got %d", len(got))
	}
	// Reading AGENTS.md is what lets rules written for another agent apply here
	// without being rewritten.
	if !strings.Contains(got[0].Text, "SHARED") || !strings.Contains(got[1].Text, "SPECIFIC") {
		t.Errorf("got %+v", got)
	}
}

// The adapter exists because Go returns are not covariant, and an adapter that
// nothing exercises is an adapter that compiles and does not work. Both halves
// are asserted: the one that hands the process across, and the one that must
// not hand a typed nil across when starting failed.
func TestTheBackgroundAdapterCarriesTheProcessAndTheFailure(t *testing.T) {
	sb, err := sandbox.New(sandbox.Config{Backend: sandbox.BackendNone}, policy.ModeFullAccess)
	if err != nil {
		t.Fatal(err)
	}
	b := background{sandbox.Runner{Sandbox: sb, Mode: policy.ModeFullAccess}}

	h, err := b.Start(context.Background(), t.TempDir(), "sleep 30")
	if err != nil {
		t.Fatal(err)
	}
	if h == nil {
		t.Fatal("a started process came back as no handle at all")
	}
	h.Stop()

	// A nil *Proc wrapped in a non-nil interface is the classic way this shape
	// goes wrong: the caller checks the error, gets one, and a later nil check
	// on the handle passes anyway.
	failed := background{sandbox.Runner{}}
	h, err = failed.Start(context.Background(), t.TempDir(), "sleep 30")
	if err == nil {
		t.Fatal("starting without a sandbox reported success")
	}
	if h != nil {
		t.Error("a failed start still handed back a handle")
	}
}
