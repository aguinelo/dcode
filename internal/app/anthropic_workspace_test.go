package app

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/aguinelo/dcode/internal/config"
	"github.com/aguinelo/dcode/internal/provider"
)

// An Anthropic key that is not scoped to a workspace has to name one on every
// request, and the API refuses the request otherwise. The header belongs to
// that dialect alone: sent to an OpenAI-compatible endpoint it is a stranger's
// header on someone else's wire.
func TestTheWorkspaceHeaderGoesOnlyOnTheAnthropicTransport(t *testing.T) {
	for _, tc := range []struct {
		name, transport, workspace, want string
	}{
		{"anthropic with a workspace names it", provider.TransportAnthropic, "wrkspc_01abc", "wrkspc_01abc"},
		{"anthropic without one sends nothing", provider.TransportAnthropic, "", ""},
		{"openai never sends it", provider.TransportOpenAI, "wrkspc_01abc", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := make(chan string, 1)
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				got <- r.Header.Get("anthropic-workspace-id")
				w.Header().Set("Content-Type", "text/event-stream")
			}))
			defer srv.Close()

			tr := NewHTTPTransport(tc.transport, srv.URL, "k")
			tr.workspace = tc.workspace
			events, err := tr.Do(context.Background(), provider.WireRequest{Body: []byte(`{}`)})
			if err != nil {
				t.Fatal(err)
			}
			for range events {
			}
			if h := <-got; h != tc.want {
				t.Errorf("anthropic-workspace-id = %q, want %q", h, tc.want)
			}
		})
	}
}

// The configured workspace reaches the transport that speaks the dialect.
func TestBuildProviderHandsTheWorkspaceToTheAnthropicTransport(t *testing.T) {
	p, err := buildProvider(Options{Model: "claude-sonnet-4-5", APIKey: "k", AnthropicWorkspace: "wrkspc_01abc"})
	if err != nil {
		t.Fatal(err)
	}
	tr, ok := p.Transport().(*HTTPTransport)
	if !ok {
		t.Fatalf("transport is %T", p.Transport())
	}
	if tr.workspace != "wrkspc_01abc" {
		t.Errorf("workspace = %q, want the configured one", tr.workspace)
	}
}

// The refusal the API gives names a header, and a person cannot connect a
// header to anything they can set. The error names the key that fixes it.
func TestAMissingWorkspaceRefusalNamesTheKeyThatFixesIt(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"type":"error","error":{"type":"invalid_request_error",` +
			`"message":"This API key must include the anthropic-workspace-id header."}}`))
	}))
	defer srv.Close()

	_, err := NewHTTPTransport(provider.TransportAnthropic, srv.URL, "k").
		Do(context.Background(), provider.WireRequest{Body: []byte(`{}`)})
	var pe *provider.ProviderError
	if !asProviderErr(err, &pe) {
		t.Fatalf("got %v, want a provider error", err)
	}
	if pe.Class != provider.ErrClassBadRequest {
		t.Errorf("class = %s, want bad_request", pe.Class)
	}
	for _, want := range []string{"model.anthropic_workspace", "DCODE_ANTHROPIC_WORKSPACE"} {
		if !strings.Contains(pe.Message, want) {
			t.Errorf("message %q does not name %s", pe.Message, want)
		}
	}
}

// Any other refusal is left as the provider said it.
func TestAnUnrelatedRefusalIsNotExplainedAsAWorkspace(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"type":"error","error":{"type":"invalid_request_error","message":"max_tokens: too large"}}`))
	}))
	defer srv.Close()

	_, err := NewHTTPTransport(provider.TransportAnthropic, srv.URL, "k").
		Do(context.Background(), provider.WireRequest{Body: []byte(`{}`)})
	var pe *provider.ProviderError
	if !asProviderErr(err, &pe) {
		t.Fatalf("got %v", err)
	}
	if strings.Contains(pe.Message, "anthropic_workspace") || !strings.Contains(pe.Message, "max_tokens") {
		t.Errorf("message = %q, want the provider's own words", pe.Message)
	}
}

// A profile names one account as much as one endpoint, so it can carry the
// workspace too; one that names none leaves the configured one standing.
func TestAProfileCarriesItsWorkspace(t *testing.T) {
	opts := Options{AnthropicWorkspace: "wrkspc_base", Profiles: map[string]config.Profile{
		"work":  {Name: "work", Model: "claude-opus-4-1", AnthropicWorkspace: "wrkspc_work"},
		"plain": {Name: "plain", Model: "claude-opus-4-1"},
	}}
	if got := applyModelRequest(opts, "work").AnthropicWorkspace; got != "wrkspc_work" {
		t.Errorf("work: got %q, want the profile's workspace", got)
	}
	if got := applyModelRequest(opts, "plain").AnthropicWorkspace; got != "wrkspc_base" {
		t.Errorf("plain: got %q, want the configured workspace left standing", got)
	}
}

// The value becomes an HTTP header, so it is checked where it enters.
func TestAWorkspaceThatIsNotAnIDIsRefusedAtStart(t *testing.T) {
	ws := t.TempDir()
	home := t.TempDir()
	env := func(k string) string {
		switch k {
		case "DCODE_HOME":
			return home
		case "HOME":
			return home
		case "DCODE_ANTHROPIC_WORKSPACE":
			return "wrkspc 01; x"
		}
		return ""
	}
	if _, _, err := FromEnv(env, ws); err == nil || !strings.Contains(err.Error(), "model.anthropic_workspace") {
		t.Errorf("got %v, want a refusal naming the key", err)
	}
}
