package app

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/aguinelo/dcode/internal/behavior"
	"github.com/aguinelo/dcode/internal/config"
	ce "github.com/aguinelo/dcode/internal/contextengine"
	"github.com/aguinelo/dcode/internal/loop"
	"github.com/aguinelo/dcode/internal/policy"
	"github.com/aguinelo/dcode/internal/protocol"
	"github.com/aguinelo/dcode/internal/provider"
	"github.com/aguinelo/dcode/internal/sandbox"
	"github.com/aguinelo/dcode/internal/tools"
)

// envFrom builds a synthetic environment.
//
// DCODE_HOME is filled in when the case does not set it, pointing at a
// directory that does not exist: option resolution now reads config.toml from
// the user and project roots, and a test that inherited the developer's real
// home would pass or fail depending on whose machine it ran on.
func envFrom(m map[string]string) func(string) string {
	if m["DCODE_HOME"] == "" && m["HOME"] == "" {
		m["DCODE_HOME"] = filepath.Join(os.TempDir(), "dcode-no-such-home")
	}
	return func(k string) string { return m[k] }
}

func TestFromEnvAppliesDefaults(t *testing.T) {
	opts, _, err := FromEnv(envFrom(map[string]string{}), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if opts.Model != "MiniMax-M3" {
		t.Errorf("M3 is the project's primary model, got %q", opts.Model)
	}
	if opts.SandboxMode != policy.ModeWorkspaceWrite {
		t.Errorf("got %q", opts.SandboxMode)
	}
	if opts.Policy != policy.PolicyOnRequest {
		t.Errorf("got %q", opts.Policy)
	}
	// Unset means "ask the family", which is how a long-horizon model gets a
	// higher ceiling than a short-task one.
	if opts.Limits.MaxIterations != 0 {
		t.Errorf("the iteration cap should default to the family, got %d", opts.Limits.MaxIterations)
	}
	if opts.Limits.MaxIdenticalCalls != 3 {
		t.Errorf("got %d", opts.Limits.MaxIdenticalCalls)
	}
}

func TestFromEnvHonoursTheEnvironment(t *testing.T) {
	opts, _, err := FromEnv(envFrom(map[string]string{
		"DCODE_MODEL":               "claude-sonnet",
		"DCODE_SANDBOX_MODE":        "read-only",
		"DCODE_APPROVAL_POLICY":     "never",
		"DCODE_MAX_ITERATIONS":      "7",
		"DCODE_MAX_IDENTICAL_CALLS": "2",
		"DCODE_ALLOW_NETWORK":       "true",
	}), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if opts.Model != "claude-sonnet" || opts.SandboxMode != policy.ModeReadOnly ||
		opts.Policy != policy.PolicyNever || opts.Limits.MaxIterations != 7 ||
		opts.Limits.MaxIdenticalCalls != 2 || !opts.AllowNetwork {
		t.Errorf("got %+v", opts)
	}
}

// A misspelled mode must stop the session rather than fall back. Falling back
// would either surprise the user with less access than asked for, or — far
// worse — with more.
func TestFromEnvRejectsAnInvalidMode(t *testing.T) {
	if _, _, err := FromEnv(envFrom(map[string]string{
		"DCODE_SANDBOX_MODE": "yolo",
	}), t.TempDir()); err == nil {
		t.Error("an unknown sandbox mode must be rejected")
	}
	if _, _, err := FromEnv(envFrom(map[string]string{
		"DCODE_APPROVAL_POLICY": "sometimes",
	}), t.TempDir()); err == nil {
		t.Error("an unknown approval policy must be rejected")
	}
}

// ---------- end to end ----------

// The test that proves the whole thing works: a scripted model drives the real
// tools against a real workspace, and the file on disk changes.
func TestAgentEditsARealFileEndToEnd(t *testing.T) {
	ws := t.TempDir()
	target := filepath.Join(ws, "main.go")
	original := "package main\n\nfunc main() {\n\tprintln(\"old\")\n}\n"
	if err := os.WriteFile(target, []byte(original), 0o644); err != nil {
		t.Fatal(err)
	}

	// The model plans, reads, edits, then reports — the ordinary shape of a
	// small task.
	frames := []string{
		frameToolCall("c1", "plan",
			`{"items":[{"id":1,"text":"read main.go","status":"active"},{"id":2,"text":"change the string","status":"pending"}]}`),
		"[DONE]",
	}
	second := []string{frameToolCall("c2", "read", `{"path":"main.go"}`), "[DONE]"}
	third := []string{
		frameToolCall("c3", "edit",
			`{"path":"main.go","old_string":"println(\"old\")","new_string":"println(\"new\")"}`),
		"[DONE]",
	}
	fourth := []string{frameText("Changed the string in main.go."), "[DONE]"}

	sess := wireSession(t, ws, [][]string{frames, second, third, fourth})

	out, err := sess.Engine.Run(context.Background(), "change old to new in main.go")
	if err != nil {
		t.Fatalf("the turn failed: %v", err)
	}
	if out.Reason != protocol.StopDone {
		t.Fatalf("got %q want done", out.Reason)
	}

	got, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(got), `println("new")`) {
		t.Errorf("the file was not edited:\n%s", got)
	}
	if strings.Contains(string(got), `println("old")`) {
		t.Errorf("the old text survived:\n%s", got)
	}
}

// The invariant has to hold through the whole stack, not just in the tool: an
// edit without a prior read must fail even when everything else is wired up.
func TestEditWithoutReadIsRefusedThroughTheWholeStack(t *testing.T) {
	ws := t.TempDir()
	target := filepath.Join(ws, "a.go")
	if err := os.WriteFile(target, []byte("package a\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	sess := wireSession(t, ws, [][]string{
		{frameToolCall("c1", "edit",
			`{"path":"a.go","old_string":"package a","new_string":"package b"}`), "[DONE]"},
		{frameText("Right, I need to read it first."), "[DONE]"},
	})

	if _, err := sess.Engine.Run(context.Background(), "rename the package"); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(target)
	if string(got) != "package a\n" {
		t.Errorf("the file changed without being read first:\n%s", got)
	}
}

// Nothing reaches the network without consent, and with nobody to ask the
// answer is no.
// Nothing reaches the network without consent — but consent is only asked for
// when there is a crossing to consent to.
//
// With the network open at the sandbox, a shell command really can reach out,
// so the approval is the only thing standing in the way and a refusal must stop
// it before it runs.
func TestNetworkAccessIsDeniedWithoutApprovalWhenTheNetworkIsOpen(t *testing.T) {
	ws := t.TempDir()
	sess := wireSessionNet(t, ws, [][]string{
		{frameToolCall("c1", "bash", `{"command":"curl https://example.com"}`), "[DONE]"},
		{frameText("Understood, no network."), "[DONE]"},
	}, DenyAll{}, true)

	if _, err := sess.Engine.Run(context.Background(), "fetch a page"); err != nil {
		t.Fatal(err)
	}
	var refused bool
	for _, m := range sess.Engine.Session().History {
		if m.ToolResult != nil && m.ToolResult.IsError &&
			strings.Contains(strings.ToLower(m.ToolResult.Output), "denied") {
			refused = true
		}
	}
	if !refused {
		t.Error("a network crossing must be refused when nobody approves it")
	}
}

// Regression: with the network shut at the sandbox there is no crossing, so the
// command is not gated on one — and it still cannot reach the network, because
// the OS is what stops it rather than the prompt.
//
// The old behaviour asked on every single command and the answer changed
// nothing: approving still could not resolve a host, and denying stopped the
// whole command rather than its network access.
func TestAShellCommandIsNotGatedOnACrossingTheSandboxPrevents(t *testing.T) {
	ws := t.TempDir()
	sess := wireSessionWith(t, ws, [][]string{
		{frameToolCall("c1", "bash", `{"command":"curl -sS -m 5 https://example.com"}`), "[DONE]"},
		{frameText("The network is unavailable here."), "[DONE]"},
	}, DenyAll{})

	if _, err := sess.Engine.Run(context.Background(), "fetch a page"); err != nil {
		t.Fatal(err)
	}

	var result string
	for _, m := range sess.Engine.Session().History {
		if m.ToolResult != nil {
			result = m.ToolResult.Output
		}
	}
	if strings.Contains(strings.ToLower(result), "denied") {
		t.Errorf("nothing was crossed, so nothing should have been refused: %q", result)
	}
	// And the network is still unreachable — which is the invariant that
	// actually matters, and it is the OS that holds it.
	if strings.Contains(result, "<!doctype") || strings.Contains(result, "<html") {
		t.Errorf("the network must remain unreachable: %q", result)
	}
}

// The prompt has to be inspectable: a harness that cannot show what it sends
// asks for blind trust in a program with shell access.
func TestPromptIsAssembledAndInspectable(t *testing.T) {
	ws := t.TempDir()
	if err := os.WriteFile(filepath.Join(ws, "AGENTS.md"),
		[]byte("Always run gofmt before finishing."), 0o644); err != nil {
		t.Fatal(err)
	}
	sess := wireSession(t, ws, [][]string{{frameText("ok"), "[DONE]"}})

	if !strings.Contains(sess.Prompt, "## Safety") {
		t.Error("the safety section must be present")
	}
	if !strings.Contains(sess.Prompt, "gofmt") {
		t.Error("project instructions must be picked up from AGENTS.md")
	}
	for _, tool := range []string{"read", "edit", "bash", "plan"} {
		if !strings.Contains(sess.Prompt, tool) {
			t.Errorf("the prompt should name the %s tool", tool)
		}
	}
}

// ---------- wiring helpers ----------

func wireSession(t *testing.T, ws string, turns [][]string) *Session {
	return wireSessionWith(t, ws, turns, DenyAll{})
}

func wireSessionWith(t *testing.T, ws string, turns [][]string, approver loop.Approver) *Session {
	return wireSessionNet(t, ws, turns, approver, false)
}

// wireSessionNet wires a session with the network open or shut, which is the
// axis that decides whether there is a crossing to consent to at all.
func wireSessionNet(t *testing.T, ws string, turns [][]string, approver loop.Approver, net bool) *Session {
	t.Helper()

	resolver, err := policy.NewResolver(ws)
	if err != nil {
		t.Fatal(err)
	}
	sb, err := sandbox.New(sandbox.Config{AllowNetwork: func() bool { return net }}, policy.ModeWorkspaceWrite)
	if err != nil {
		// Without a real boundary the end-to-end assertion would be testing
		// nothing, so skipping is the honest outcome.
		t.Skipf("no sandbox available: %v", err)
	}

	state := tools.NewState(resolver, tools.DefaultLimits())
	registry := tools.NewRegistry(
		tools.Read{}, tools.Write{}, tools.Edit{}, tools.Glob{}, tools.Grep{},
		tools.Bash{
			Runner:  sandbox.Runner{Sandbox: sb, Mode: policy.ModeWorkspaceWrite},
			Workdir: ws, AllowNetwork: net,
		},
		tools.Plan{},
	)

	p := &sequenced{turns: turns}
	prompt := "You are dcode."
	engine := loop.New(loop.Config{
		Provider: p, Tools: registry, State: state,
		Emitter: &ConsoleEmitter{W: &bytes.Buffer{}}, Approver: approver,
		Limits: loop.Limits{MaxIterations: 20, MaxIdenticalCalls: 3},
		Mode:   policy.ModeWorkspaceWrite, Policy: policy.PolicyOnRequest,
		Model: "MiniMax-M3", Parallel: 4,
	}, ce.Session{Instructions: prompt})

	return &Session{Engine: engine, Registry: registry, Prompt: buildTestPrompt(t, ws, registry)}
}

func buildTestPrompt(t *testing.T, ws string, reg *tools.Registry) string {
	t.Helper()
	instructions, _, err := loadInstructions(config.Roots{Config: filepath.Join(ws, ".absent")}, ws)
	if err != nil {
		t.Fatal(err)
	}
	prompt, err := behaviorBuild(reg.Names(), instructions, nil, behavior.DoctrineOverlay{}, "minimax-m3")
	if err != nil {
		t.Fatal(err)
	}
	return prompt
}

// sequenced replays one recorded turn per call through the real family decoder,
// so the end-to-end test exercises actual wire parsing rather than a shortcut.
type sequenced struct {
	turns [][]string
	call  int
}

func (s *sequenced) Family() provider.Family       { return provider.MiniMaxM3{} }
func (s *sequenced) Transport() provider.Transport { return nil }
func (s *sequenced) Window(string) (int, error)    { return 1_000_000, nil }
func (s *sequenced) Limits() provider.Limits       { return provider.Limits{MaxIterations: 200} }

func (s *sequenced) Stream(ctx context.Context, req provider.Request) (<-chan provider.StreamEvent, error) {
	idx := s.call
	s.call++

	var frames []string
	if idx < len(s.turns) {
		frames = s.turns[idx]
	} else {
		frames = []string{"[DONE]"}
	}

	rt := provider.NewReplayTransport(provider.TransportOpenAI, provider.Transcript{Frames: frames})
	reg := provider.NewRegistry()
	reg.RegisterTransport(rt)
	if err := reg.RegisterFamily(provider.MiniMaxM3{}); err != nil {
		return nil, err
	}
	p, err := reg.Resolve("MiniMax-M3", provider.TransportOpenAI)
	if err != nil {
		return nil, err
	}
	return p.Stream(ctx, req)
}

func frameText(s string) string {
	b, _ := json.Marshal(map[string]any{
		"choices": []any{map[string]any{"delta": map[string]any{"content": s}}},
	})
	return string(b)
}

func frameToolCall(id, name, args string) string {
	b, _ := json.Marshal(map[string]any{
		"choices": []any{map[string]any{"delta": map[string]any{
			"tool_calls": []any{map[string]any{
				"id": id, "function": map[string]any{"name": name, "arguments": args},
			}},
		}}},
	})
	return string(b)
}

// ---------- console rendering ----------

func TestConsoleCollapsesSuccessAndExpandsFailure(t *testing.T) {
	var buf bytes.Buffer
	c := &ConsoleEmitter{W: &buf}

	c.Emit(protocol.EventToolCompleted, protocol.ToolCompleted{
		ToolCallID: "c1", OK: true, Output: "SHOULD-NOT-APPEAR",
	})
	if strings.Contains(buf.String(), "SHOULD-NOT-APPEAR") {
		t.Error("successful output should stay collapsed; success needs only confirmation")
	}

	c.Emit(protocol.EventToolCompleted, protocol.ToolCompleted{
		ToolCallID: "c2", OK: false, Output: "MUST-APPEAR: file not found",
	})
	if !strings.Contains(buf.String(), "MUST-APPEAR") {
		t.Error("a failure must be shown; failure needs attention")
	}
}

func TestVerboseShowsSuccessToo(t *testing.T) {
	var buf bytes.Buffer
	c := &ConsoleEmitter{W: &buf, Verbose: true}
	c.Emit(protocol.EventToolCompleted, protocol.ToolCompleted{OK: true, Output: "VISIBLE"})
	if !strings.Contains(buf.String(), "VISIBLE") {
		t.Error("verbose should show successful output")
	}
}

// The command must be rendered, not described: asking for consent to "access
// the network" without showing what will run is asking for it blind.
func TestApprovalPromptShowsTheCommand(t *testing.T) {
	var buf bytes.Buffer
	c := &ConsoleEmitter{W: &buf}
	c.Emit(protocol.EventApprovalRequired, protocol.ApprovalRequest{
		Tool: "bash", Command: "curl -X POST https://example.com", BoundaryCrossed: "network",
	})
	out := buf.String()
	if !strings.Contains(out, "curl -X POST") {
		t.Errorf("the command must be visible: %q", out)
	}
	if !strings.Contains(out, "network") {
		t.Errorf("the boundary crossed must be named: %q", out)
	}
}

func TestPlanRendersBlockedReason(t *testing.T) {
	var buf bytes.Buffer
	c := &ConsoleEmitter{W: &buf}
	c.Emit(protocol.EventPlanUpdated, protocol.PlanUpdated{Items: []protocol.PlanItem{
		{ID: 2, Text: "run tests", Status: protocol.PlanBlocked, Blocked: "missing dependency"},
		{ID: 1, Text: "read code", Status: protocol.PlanDone},
	}})
	out := buf.String()
	// A block with no visible cause is worse than no block at all.
	if !strings.Contains(out, "missing dependency") {
		t.Errorf("the blocking reason must be shown: %q", out)
	}
	// Sorted by id, so the list does not jump around between updates.
	if strings.Index(out, "read code") > strings.Index(out, "run tests") {
		t.Errorf("items should render in id order: %q", out)
	}
}

// Anything other than an explicit yes is a refusal, and the default costs the
// least effort.
func TestConsoleApproverDefaultsToDeny(t *testing.T) {
	for _, tc := range []struct {
		input string
		want  protocol.ApprovalDecision
	}{
		{"\n", protocol.ApprovalDeny},
		{"d\n", protocol.ApprovalDeny},
		{"yes\n", protocol.ApprovalDeny},
		{"", protocol.ApprovalDeny},
		{"a\n", protocol.ApprovalAllow},
		{"A\n", protocol.ApprovalAllowSession},
	} {
		var out bytes.Buffer
		a := &ConsoleApprover{In: strings.NewReader(tc.input), Out: &out}
		got, err := a.Approve(context.Background(), protocol.ApprovalRequest{})
		if err != nil {
			t.Fatal(err)
		}
		if got != tc.want {
			t.Errorf("input %q: got %s want %s", tc.input, got, tc.want)
		}
	}
}

func TestDenyAllAlwaysDenies(t *testing.T) {
	got, err := DenyAll{}.Approve(context.Background(), protocol.ApprovalRequest{})
	if err != nil || got != protocol.ApprovalDeny {
		t.Errorf("got %s %v", got, err)
	}
}

func TestSummariseInput(t *testing.T) {
	for _, tc := range []struct{ in, want string }{
		{`{"path":"main.go"}`, "main.go"},
		{`{"pattern":"*.go"}`, "*.go"},
		{`{"command":"go test"}`, "go test"},
		{`{"other":1}`, ""},
		{`not json`, ""},
	} {
		if got := summariseInput(json.RawMessage(tc.in)); got != tc.want {
			t.Errorf("%s: got %q want %q", tc.in, got, tc.want)
		}
	}
	long := `{"command":"` + strings.Repeat("x", 200) + `"}`
	if got := summariseInput(json.RawMessage(long)); len(got) > 70 {
		t.Errorf("a long command should be clipped, got %d chars", len(got))
	}
}

func TestHTTPTransportRefusesWithoutAKey(t *testing.T) {
	tr := NewHTTPTransport(provider.TransportOpenAI, "", "")
	_, err := tr.Do(context.Background(), provider.WireRequest{})
	if err == nil {
		t.Fatal("no key must fail before any request is made")
	}
	var pe *provider.ProviderError
	if !asProviderErr(err, &pe) || pe.Class != provider.ErrClassAuth {
		t.Errorf("got %v", err)
	}
	if !strings.Contains(pe.Message, "DCODE_API_KEY") {
		t.Errorf("the error should say how to fix it: %q", pe.Message)
	}
}

func TestHTTPTransportDefaultsPerDialect(t *testing.T) {
	if got := NewHTTPTransport(provider.TransportAnthropic, "", "k").baseURL; !strings.Contains(got, "anthropic") {
		t.Errorf("got %q", got)
	}
	if got := NewHTTPTransport(provider.TransportOpenAI, "", "k").baseURL; !strings.Contains(got, "minimax") {
		t.Errorf("M3 is the primary model, got %q", got)
	}
	if got := NewHTTPTransport(provider.TransportOpenAI, "https://proxy/v1/", "k").baseURL; got != "https://proxy/v1" {
		t.Errorf("an explicit base URL should win and lose its trailing slash, got %q", got)
	}
}

func asProviderErr(err error, target **provider.ProviderError) bool {
	if pe, ok := err.(*provider.ProviderError); ok {
		*target = pe
		return true
	}
	return false
}
