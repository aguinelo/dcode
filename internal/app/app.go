// Package app wires the layers into a runnable agent.
//
// It is the only place that reads the environment and touches the outside
// world on behalf of the core: everything below stays pure or injectable, which
// is what keeps the rest exactly testable.
package app

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/aguinelo/dcode/internal/behavior"
	"github.com/aguinelo/dcode/internal/config"
	ce "github.com/aguinelo/dcode/internal/contextengine"
	"github.com/aguinelo/dcode/internal/credential"
	"github.com/aguinelo/dcode/internal/loop"
	"github.com/aguinelo/dcode/internal/policy"
	"github.com/aguinelo/dcode/internal/protocol"
	"github.com/aguinelo/dcode/internal/provider"
	"github.com/aguinelo/dcode/internal/sandbox"
	"github.com/aguinelo/dcode/internal/tools"
)

// Options are the resolved settings for one session.
type Options struct {
	Workspace    string
	Model        string
	Transport    string
	Family       string
	APIKey       string
	BaseURL      string
	SandboxMode  policy.SandboxMode
	Policy       policy.ApprovalPolicy
	Backend      string
	AllowNetwork bool
	Parallel     int
	Limits       loop.Limits
	DumpPrompt   bool
	// Reminders switches the appended-notice channel on.
	Reminders bool
	// ShowReasoning forwards the model's thinking to clients.
	ShowReasoning bool
	// Instructions switches the reading of AGENTS.md and DCODE.md on. Off runs
	// on the shipped doctrine alone, which is how one tells whether a behaviour
	// comes from the user's instructions or from the product.
	Instructions bool
	// Skills switches progressive disclosure on. Off removes the index from the
	// prefix, and no body is ever loaded.
	Skills bool
	// SymbolMaxMatches caps what symbol returns. Same ceiling as grep, and for
	// the same reason: a symbol matching thousands of times is a badly chosen
	// symbol, and returning all of it spends context without informing.
	SymbolMaxMatches int
	// EditEchoDiff decides when the diff of an edit goes back to the model.
	// It always reaches the client, in every mode.
	EditEchoDiff string
	// DoctrineOverlay switches the reading of the user's doctrine/ directory
	// on. Off runs on the shipped doctrine alone, which is how one tells
	// whether a behaviour comes from the user's overlay or from the product.
	DoctrineOverlay bool
	// DoctrineDir overrides where the overlay is read from. Empty means
	// doctrine/ under the user's config root — never the workspace (RN-11).
	DoctrineDir string
	// DoctrineMaxBytes caps each overlay file. Smaller than the instruction
	// cap because this is the base layer, paid on every turn of every session.
	DoctrineMaxBytes int
	// Env is how the session reaches the environment. Carried on Options rather
	// than read from the process, so a daemon serving several workspaces is not
	// forced to share one view of it.
	Env func(string) string `json:"-"`
	// CredentialFrom records where the key came from, so `dcode config` can
	// answer "which one is this" without ever printing it.
	CredentialFrom string
	// Rules ask a question the sandbox cannot, for paths and commands that are
	// different in kind from ordinary work.
	Rules policy.Rules
	// CredentialBackend selects the store. Empty chooses.
	//
	// Configuration rather than a per-command flag: a flag on the command that
	// writes, and nothing on the commands that read, stores the secret
	// somewhere nothing looks for it.
	CredentialBackend string
}

// FromEnv resolves options from the environment, applying the precedence chain.
func FromEnv(env func(string) string, workspace string) (Options, config.Resolved, error) {
	r, err := Resolve(env, workspace)
	if err != nil {
		return Options{}, config.Resolved{}, err
	}
	return fromResolved(r, env, workspace)
}

// Resolve builds the configuration chain and nothing else.
//
// Split out from FromEnv because commands that are not a session — `update` is
// the one today — still have to read configuration through the same chain. A
// command reading os.Getenv directly is how `update.channel` came to be a
// declared key that a config file could never reach.
func Resolve(env func(string) string, workspace string) (config.Resolved, error) {
	defaults := policy.DefaultRules()
	layers := []config.Layer{
		{Source: config.SourceDefault, Origin: "built-in", Values: map[string]string{
			"model.name":              "MiniMax-M3",
			"sandbox.mode":            string(policy.ModeWorkspaceWrite),
			"sandbox.approval_policy": string(policy.PolicyOnRequest),
			"sandbox.backend":         sandbox.BackendAuto,
			"sandbox.allow_network":   "false",
			"limits.parallel":         "4",
			// The rules live here rather than only in code, so `--config` can
			// show them with an origin. A rule that governs behaviour and
			// cannot be inspected is the gap the audit pair exists to close.
			// Every default that governs behaviour lives here rather than only
			// in the code that reads it, so `--config` answers with a value and
			// an origin. A setting that cannot be inspected is one nobody can
			// reason about when it surprises them.
			"behavior.instructions_enabled": "true",
			"behavior.skills_enabled":       "true",
			"behavior.reminders_enabled":    "true",
			"behavior.show_reasoning":       "true",
			"rules.confirm_write":           policy.JoinList(defaults.ConfirmWrite),
			"rules.confirm_read":            policy.JoinList(defaults.ConfirmRead),
			"rules.confirm_command":         policy.JoinList(defaults.ConfirmCommand),
			// Eval is off by default because it runs a real model and costs
			// money. The default is stated here rather than only in the
			// harness so `--config eval.enabled` answers with an origin.
			"eval.enabled": "false",
			"eval.runs":    "20",
			// The overlay is read by default; without files it changes nothing.
			// The cap is smaller than the instruction cap because this is the
			// base layer, paid on every turn of every session.
			// The only case where the model cannot derive what the file now
			// says is a replace_all that hit more than one occurrence.
			"tools.edit_echo_diff":     tools.EchoDiffMulti,
			"tools.symbol_max_matches": "200",
			"doctrine.enabled":         "true",
			"doctrine.max_bytes":       "16384",
		}},
	}

	// Files come before the environment: the chain is default < user file <
	// project file < environment < flag < locked, and Resolve orders by source
	// rather than by position, so loading order here only has to be complete.
	roots, err := config.DiscoverRoots(env)
	if err != nil {
		return config.Resolved{}, err
	}
	fileLayers, err := config.FileLayers(roots, workspace)
	if err != nil {
		return config.Resolved{}, err
	}
	layers = append(layers, fileLayers...)

	// The key-to-variable mapping lives in one place, so a key that exists in
	// the file and not in the environment (or the reverse) is impossible.
	envValues := map[string]string{}
	for key, name := range config.KnownKeys {
		if v := env(name); v != "" {
			envValues[key] = v
		}
	}
	if len(envValues) > 0 {
		layers = append(layers, config.Layer{
			Source: config.SourceEnv, Origin: "environment", Values: envValues,
		})
	}

	return config.Resolve(layers), nil
}

// fromResolved turns a resolved chain into Options.
func fromResolved(r config.Resolved, env func(string) string, workspace string) (Options, config.Resolved, error) {
	mode, err := policy.ParseMode(r.String("sandbox.mode", string(policy.ModeWorkspaceWrite)))
	if err != nil {
		return Options{}, r, err
	}
	pol, err := policy.ParsePolicy(r.String("sandbox.approval_policy", string(policy.PolicyOnRequest)))
	if err != nil {
		return Options{}, r, err
	}

	ws, err := filepath.Abs(workspace)
	if err != nil {
		return Options{}, r, err
	}

	opts := Options{
		Env:               env,
		Reminders:         r.Bool("behavior.reminders_enabled", true),
		ShowReasoning:     r.Bool("behavior.show_reasoning", true),
		Instructions:      r.Bool("behavior.instructions_enabled", true),
		Skills:            r.Bool("behavior.skills_enabled", true),
		EditEchoDiff:      r.String("tools.edit_echo_diff", tools.EchoDiffMulti),
		SymbolMaxMatches:  r.Int("tools.symbol_max_matches", 200),
		DoctrineOverlay:   r.Bool("doctrine.enabled", true),
		DoctrineDir:       r.String("doctrine.dir", ""),
		DoctrineMaxBytes:  r.Int("doctrine.max_bytes", 16<<10),
		Workspace:         ws,
		Model:             r.String("model.name", "MiniMax-M3"),
		Transport:         r.String("model.transport", ""),
		Family:            r.String("model.family", ""),
		APIKey:            env("DCODE_API_KEY"),
		BaseURL:           r.String("model.base_url", ""),
		SandboxMode:       mode,
		Policy:            pol,
		Backend:           r.String("sandbox.backend", sandbox.BackendAuto),
		AllowNetwork:      r.Bool("sandbox.allow_network", false),
		Parallel:          r.Int("limits.parallel", 4),
		DumpPrompt:        r.Bool("doctrine.dump", false),
		CredentialBackend: r.String("credential.backend", ""),
		Rules:             resolveRules(r),
		Limits: loop.Limits{
			MaxIterations:     r.Int("limits.max_iterations", 0),
			MaxIdenticalCalls: r.Int("limits.identical", 3),
			MaxTurnTokens:     r.Int("limits.max_turn_tokens", 0),
		},
	}
	if opts.APIKey != "" {
		opts.CredentialFrom = "DCODE_API_KEY"
	} else {
		// The environment wins because it is explicit and scoped to one
		// invocation; the store is what makes the ordinary case not require it.
		roots, err := config.DiscoverRoots(env)
		if err != nil {
			return Options{}, r, err
		}
		secret, from := LookupCredential(roots, opts)
		opts.APIKey, opts.CredentialFrom = secret, from
	}
	return opts, r, nil
}

// resolveRules reads the rule lists from the resolved configuration.
//
// The defaults are a layer like any other, so a configured list *replaces*
// them: someone who writes a list has said what they want asked about, and
// quietly keeping ours underneath would make their configuration a lie. The
// empty list is how you say "nothing", and it has to be expressible.
func resolveRules(r config.Resolved) policy.Rules {
	return policy.Rules{
		ConfirmWrite:   policy.SplitList(r.String("rules.confirm_write", "")),
		ConfirmRead:    policy.SplitList(r.String("rules.confirm_read", "")),
		ConfirmCommand: policy.SplitList(r.String("rules.confirm_command", "")),
	}
}

// LookupCredential reads the stored key for these options.
//
// A failure is silent on purpose: the store not being reachable is not a reason
// to refuse to start, and the turn that needs the key reports a clear auth error
// with a message that says where to set it.
func LookupCredential(roots config.Roots, opts Options) (secret, from string) {
	name := CredentialName(opts)
	if name == "" {
		return "", ""
	}
	store, err := credential.Open(credential.Options{
		StateDir: roots.State, Backend: opts.CredentialBackend,
	})
	if err != nil {
		return "", ""
	}
	v, err := store.Get(name)
	if err != nil || v == "" {
		return "", ""
	}
	return v, store.Where()
}

// Session is a wired agent ready to take turns.
type Session struct {
	Engine   *loop.Engine
	Registry *tools.Registry
	Prompt   string
	Options  Options
	// Origins is where each doctrine section came from, and Notices is what
	// the overlay loader refused to do silently. Both exist for the audit:
	// an invisible replacement would be worse than the immutability it
	// replaces (RN-12).
	Origins        behavior.SectionOrigins
	DoctrineNotice []behavior.Notice
	// ContextWindow is what the provider reports for this model.
	ContextWindow int
}

// New wires a session.
func New(opts Options, emitter loop.Emitter, approver loop.Approver) (*Session, error) {
	resolver, err := policy.NewResolver(opts.Workspace)
	if err != nil {
		return nil, err
	}

	// The sandbox is established before anything can run. Failing here is
	// deliberate: a session that cannot confine its own commands should not
	// start at all.
	sb, err := sandbox.New(sandbox.Config{
		Backend:      opts.Backend,
		AllowNetwork: opts.AllowNetwork,
	}, opts.SandboxMode)
	if err != nil {
		return nil, err
	}

	toolLimits := tools.DefaultLimits()
	toolLimits.EditEchoDiff = opts.EditEchoDiff
	toolLimits.SymbolMaxMatches = opts.SymbolMaxMatches
	state := tools.NewState(resolver, toolLimits)
	registry := tools.NewRegistry(
		tools.Read{}, tools.Write{}, tools.Edit{},
		tools.Glob{}, tools.Grep{}, tools.Symbol{},
		tools.Bash{
			Runner:  sandbox.Runner{Sandbox: sb, Mode: opts.SandboxMode},
			Workdir: opts.Workspace,
			Timeout: 120 * time.Second,
			// The same value the sandbox was built with, so what the tool
			// declares and what the mechanism enforces cannot disagree.
			AllowNetwork: opts.AllowNetwork,
		},
		tools.Plan{},
	)

	if opts.APIKey != "" {
		provider.RegisterSecret(opts.APIKey)
	}
	p, err := buildProvider(opts)
	if err != nil {
		return nil, err
	}

	env := opts.Env
	if env == nil {
		env = os.Getenv
	}
	roots, err := config.DiscoverRoots(env)
	if err != nil {
		return nil, err
	}
	// Off leaves the chain empty rather than skipping the call, so everything
	// downstream sees the same shape either way.
	var instructions []behavior.Instruction
	var chain []string
	if opts.Instructions {
		instructions, chain, err = loadInstructions(roots, opts.Workspace)
		if err != nil {
			return nil, err
		}
	}

	// Skills are indexed in the prefix and loaded on trigger. Discovery happens
	// once, here: a skill file written mid-session must not change the prefix,
	// because the prefix is what the cache is keyed on.
	var skills []behavior.Skill
	if opts.Skills {
		skills, err = behavior.LoadSkills([]string{
			filepath.Join(roots.Config, behavior.SkillsDirName),
			filepath.Join(opts.Workspace, ".dcode", behavior.SkillsDirName),
		}, 256<<10)
		if err != nil {
			return nil, err
		}
	}

	// The doctrine overlay is resolved once, here, like the instruction chain
	// and for the same reason (RN-5): a doctrine file written mid-session must
	// not change the prefix, because the prefix is what the cache is keyed on.
	//
	// The user's config root is the only argument. The workspace root never
	// becomes one — a cloned repository must not be able to redefine who the
	// agent thinks it is (RN-11).
	var overlay behavior.DoctrineOverlay
	var overlayNotices []behavior.Notice
	if opts.DoctrineOverlay {
		overlay, overlayNotices, err = behavior.LoadDoctrineOverlay(
			doctrineDir(opts.DoctrineDir, roots), opts.DoctrineMaxBytes)
		if err != nil {
			return nil, err
		}
	}

	prompt := behaviorBuild(registry.Names(), instructions, behavior.Index(skills), overlay)

	window, _ := p.Window(opts.Model)
	ctxCfg := ce.DefaultConfig()
	ctxCfg.Window = window

	engine := loop.New(loop.Config{
		Provider: p, Tools: registry, State: state,
		Emitter: emitter, Approver: approver,
		Limits: opts.Limits, Mode: opts.SandboxMode, Policy: opts.Policy,
		Model: opts.Model, Parallel: opts.Parallel, CtxConfig: ctxCfg,
		Rules:            opts.Rules,
		Summarise:        summariser(p, opts.Model),
		Skills:           skills,
		InstructionChain: chain,
		ReadFile:         readFileText,
		Reminders:        opts.Reminders,
		ShowReasoning:    opts.ShowReasoning,
	}, ce.Session{Instructions: prompt})

	return &Session{
		Engine: engine, Registry: registry, Prompt: prompt, Options: opts,
		ContextWindow:  window,
		Origins:        overlay.Origins(),
		DoctrineNotice: overlayNotices,
	}, nil
}

// Families are the adaptations this build supports, in registration order.
func Families() []provider.Family {
	return []provider.Family{provider.MiniMaxM3{}, provider.Claude{}}
}

// CredentialName is the name a model's credential is stored under.
//
// The family rather than the model: `MiniMax-M3` and a later `MiniMax-M4` reach
// the same account at the same provider, and asking someone to store the same
// key twice is asking them to forget one of them.
func CredentialName(opts Options) string {
	if opts.Family != "" {
		return opts.Family
	}
	if f, ok := provider.FamilyFor(opts.Model, Families()); ok {
		return f.Name()
	}
	return ""
}

func buildProvider(opts Options) (provider.Provider, error) {
	reg := provider.NewRegistry()
	reg.RegisterTransport(NewHTTPTransport(provider.TransportOpenAI, opts.BaseURL, opts.APIKey))
	reg.RegisterTransport(NewHTTPTransport(provider.TransportAnthropic, opts.BaseURL, opts.APIKey))
	for _, f := range Families() {
		if err := reg.RegisterFamily(f); err != nil {
			return nil, err
		}
	}
	if opts.Family != "" {
		return reg.ResolveFamily(opts.Family, opts.Transport)
	}
	return reg.Resolve(opts.Model, opts.Transport)
}

// summariser produces compaction text. It lives here rather than in the loop
// because it needs a model call, which is what would drag I/O into the pure
// layers below.
func summariser(p provider.Provider, model string) func(context.Context, []ce.Message) (string, error) {
	return func(ctx context.Context, span []ce.Message) (string, error) {
		var b strings.Builder
		for _, m := range span {
			if m.Text != "" {
				fmt.Fprintf(&b, "%s: %s\n", m.Role, m.Text)
			}
		}
		req := provider.Request{
			Model: model,
			Messages: []ce.Message{
				{Role: ce.RoleSystem, Text: "Summarise the exchange below. " +
					"Keep the current task, decisions already made and file paths touched. " +
					"Drop exploration that led nowhere. Be brief."},
				{Role: ce.RoleUser, Text: b.String()},
			},
		}
		ch, err := p.Stream(ctx, req)
		if err != nil {
			return "", err
		}
		var out strings.Builder
		for ev := range ch {
			if ev.Type == provider.EventTextDelta {
				out.WriteString(ev.Text)
			}
		}
		return out.String(), nil
	}
}

// behaviorBuild renders a prompt from a tool set and instructions. Small
// indirection so tests can assemble one without wiring a whole session.
func behaviorBuild(toolNames []string, instructions []behavior.Instruction, index []behavior.SkillIndexEntry, overlay behavior.DoctrineOverlay) string {
	return behavior.Build(behavior.Prompt{
		Doctrine:     behavior.DefaultDoctrine(toolNames).Apply(overlay),
		Tools:        toolNames,
		Instructions: instructions,
		SkillIndex:   index,
	})
}

// loadInstructions builds the frozen instruction chain.
//
// It returns the paths as well as the instructions: the chain is what later
// tells an instruction the session never loaded from one it deliberately
// ranked, and only the paths can answer that.
func loadInstructions(roots config.Roots, workspace string) ([]behavior.Instruction, []string, error) {
	var (
		out   []behavior.Instruction
		chain []string
	)

	// The user's own root first, and lowest. It is the only place an
	// instruction from outside the workspace may enter, and it enters by an
	// explicit path rather than by discovery.
	for _, name := range config.InstructionNames {
		path := filepath.Join(roots.Config, name)
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		chain = append(chain, path)
		out = append(out, behavior.Instruction{
			Source: behavior.SourceUser, Text: string(data),
		})
	}

	files, err := config.DiscoverInstructions(workspace, workspace, nil, 65536, 8)
	if err != nil {
		return nil, nil, err
	}
	for _, f := range files {
		src := behavior.SourceProject
		if f.Source == "directory" {
			src = behavior.SourceDirectory
		}
		chain = append(chain, f.Path)
		out = append(out, behavior.Instruction{
			Source: src, Scope: filepath.Base(filepath.Dir(f.Path)), Text: f.Text,
		})
	}
	return out, chain, nil
}

// readFileText backs the changed-on-disk check and out-of-chain discovery. It
// is the one filesystem hook the loop keeps, injected rather than called
// directly so the loop stays testable without a disk.
func readFileText(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// ---------- HTTP transport ----------

// HTTPTransport speaks a wire format over HTTP with SSE. It knows nothing about
// families: a `if family == X` in here would collapse the two axes back into
// one, and the symptom only shows up at the third family.
type HTTPTransport struct {
	name    string
	baseURL string
	apiKey  string
	client  *http.Client
}

// NewHTTPTransport builds a transport for a wire format.
func NewHTTPTransport(name, baseURL, apiKey string) *HTTPTransport {
	if baseURL == "" {
		baseURL = defaultBaseURL(name)
	}
	return &HTTPTransport{
		name: name, baseURL: strings.TrimRight(baseURL, "/"), apiKey: apiKey,
		client: &http.Client{Timeout: 10 * time.Minute},
	}
}

func defaultBaseURL(name string) string {
	switch name {
	case provider.TransportAnthropic:
		return "https://api.anthropic.com/v1"
	default:
		return "https://api.minimax.io/v1"
	}
}

func (t *HTTPTransport) Name() string { return t.name }

func (t *HTTPTransport) Do(ctx context.Context, wire provider.WireRequest) (<-chan provider.WireEvent, error) {
	if t.apiKey == "" {
		return nil, &provider.ProviderError{
			Class:   provider.ErrClassAuth,
			Message: "no API key. Set DCODE_API_KEY.",
		}
	}

	path := "/chat/completions"
	if t.name == provider.TransportAnthropic {
		path = "/messages"
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, t.baseURL+path,
		bytes.NewReader(wire.Body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")
	if t.name == provider.TransportAnthropic {
		req.Header.Set("x-api-key", t.apiKey)
		req.Header.Set("anthropic-version", "2023-06-01")
	} else {
		req.Header.Set("Authorization", "Bearer "+t.apiKey)
	}

	resp, err := t.client.Do(req)
	if err != nil {
		// Sanitised because a transport error can echo the URL, and a URL can
		// carry a key.
		return nil, &provider.ProviderError{
			Class:     provider.ErrClassTransport,
			Message:   provider.Sanitize(err.Error()),
			Retryable: true,
		}
	}
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 8192))
		resp.Body.Close()
		if pe := provider.ClassifyStatus(resp.StatusCode, string(body)); pe != nil {
			return nil, pe
		}
	}

	out := make(chan provider.WireEvent)
	go func() {
		defer close(out)
		defer resp.Body.Close()

		sc := bufio.NewScanner(resp.Body)
		sc.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
		for sc.Scan() {
			line := sc.Text()
			if !strings.HasPrefix(line, "data:") {
				continue
			}
			data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
			select {
			case out <- provider.WireEvent{Data: []byte(data)}:
			case <-ctx.Done():
				return
			}
		}
		if err := sc.Err(); err != nil {
			select {
			case out <- provider.WireEvent{Err: err}:
			case <-ctx.Done():
			}
		}
	}()
	return out, nil
}

// ---------- console adapters ----------

// ConsoleEmitter renders events for the development entry point.
//
// Deliberately minimal: it exists so the core can be exercised before the TUI,
// and it holds no session state of its own, exactly like any other client.
type ConsoleEmitter struct {
	W       io.Writer
	Verbose bool
	inText  bool
}

// Emit writes one event.
func (c *ConsoleEmitter) Emit(t protocol.EventType, payload any) {
	switch t {
	case protocol.EventMessageDelta:
		if d, ok := payload.(protocol.MessageDelta); ok {
			c.inText = true
			fmt.Fprint(c.W, d.Text)
		}
	case protocol.EventToolRequested:
		if d, ok := payload.(protocol.ToolRequested); ok {
			c.newline()
			fmt.Fprintf(c.W, "⏺ %s %s\n", d.Name, summariseInput(d.Input))
		}
	case protocol.EventToolCompleted:
		if d, ok := payload.(protocol.ToolCompleted); ok {
			// Errors expand, successes collapse: failure needs attention,
			// success needs only confirmation.
			if d.OK && !c.Verbose {
				return
			}
			c.newline()
			fmt.Fprintf(c.W, "  %s\n", indent(strings.TrimSpace(d.Output)))
		}
	case protocol.EventApprovalRequired:
		if d, ok := payload.(protocol.ApprovalRequest); ok {
			c.newline()
			fmt.Fprintf(c.W, "\n⚠ approval needed — %s crosses %s\n", d.Tool, d.BoundaryCrossed)
			if d.Command != "" {
				fmt.Fprintf(c.W, "  %s\n", d.Command)
			}
		}
	case protocol.EventPlanUpdated:
		if d, ok := payload.(protocol.PlanUpdated); ok {
			c.newline()
			fmt.Fprintln(c.W, "\nPLAN")
			for _, it := range loop.SortedPlan(d.Items) {
				fmt.Fprintf(c.W, "  %s %d %s\n", planMark(it.Status), it.ID, it.Text)
				if it.Status == protocol.PlanBlocked && it.Blocked != "" {
					fmt.Fprintf(c.W, "      └ %s\n", it.Blocked)
				}
			}
			fmt.Fprintln(c.W)
		}
	case protocol.EventSessionError:
		if d, ok := payload.(protocol.Error); ok {
			c.newline()
			fmt.Fprintf(c.W, "\nerror [%s]: %s\n", d.Code, d.Message)
		}
	case protocol.EventTurnCompleted:
		c.newline()
	}
}

func (c *ConsoleEmitter) newline() {
	if c.inText {
		fmt.Fprintln(c.W)
		c.inText = false
	}
}

func planMark(status string) string {
	switch status {
	case protocol.PlanDone:
		return "✓"
	case protocol.PlanActive:
		return "▸"
	case protocol.PlanBlocked:
		return "⊘"
	}
	return " "
}

func summariseInput(raw json.RawMessage) string {
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		return ""
	}
	for _, k := range []string{"path", "pattern", "command"} {
		if v, ok := m[k].(string); ok {
			if len(v) > 60 {
				v = v[:60] + "…"
			}
			return v
		}
	}
	return ""
}

func indent(s string) string {
	lines := strings.Split(s, "\n")
	if len(lines) > 12 {
		lines = append(lines[:12], fmt.Sprintf("… %d more lines", len(lines)-12))
	}
	return strings.Join(lines, "\n  ")
}

// ConsoleApprover asks on the terminal.
type ConsoleApprover struct {
	In  io.Reader
	Out io.Writer
}

// Approve prompts and reads a decision. Anything other than an explicit yes is
// a refusal: the safe answer must be the one that costs least effort.
func (a *ConsoleApprover) Approve(_ context.Context, req protocol.ApprovalRequest) (protocol.ApprovalDecision, error) {
	fmt.Fprintf(a.Out, "  [d] deny  [a] allow  [A] allow for the session  (default: deny) > ")
	sc := bufio.NewScanner(a.In)
	if !sc.Scan() {
		return protocol.ApprovalDeny, nil
	}
	switch strings.TrimSpace(sc.Text()) {
	case "a":
		return protocol.ApprovalAllow, nil
	case "A":
		return protocol.ApprovalAllowSession, nil
	default:
		return protocol.ApprovalDeny, nil
	}
}

// DenyAll refuses every crossing. Used for non-interactive runs, where there is
// nobody to ask and granting in silence would be the only alternative.
type DenyAll struct{}

// Approve always denies.
func (DenyAll) Approve(context.Context, protocol.ApprovalRequest) (protocol.ApprovalDecision, error) {
	return protocol.ApprovalDeny, nil
}

// doctrineDir decides where the overlay is read from.
//
// It takes the config root and NOT the workspace, and that is the whole point:
// the overlay redefines who the agent thinks it is, and a repository someone
// cloned is not the user (RN-11). There is no branch here that could reach the
// workspace, because the workspace is not a parameter.
func doctrineDir(override string, roots config.Roots) string {
	if override != "" {
		return override
	}
	return filepath.Join(roots.Config, behavior.DoctrineDirName)
}

// DoctrineAudit renders where each doctrine section came from, and everything
// the overlay loader refused to do silently.
//
// It is printed under --dump-prompt because a replacement nobody can see would
// be worse than the immutability it replaces: before the overlay existed, the
// prompt itself was the whole answer to "what is in force". Now it is not, and
// this is the rest of the answer.
func DoctrineAudit(s *Session) string {
	var b strings.Builder
	b.WriteString("\n--- doctrine ---\n")
	for _, row := range []struct {
		name   string
		origin behavior.Origin
	}{
		{"Identity", s.Origins.Identity},
		{"Using tools", s.Origins.ToolPolicy},
		{"Safety", s.Origins.Safety},
		{"Style", s.Origins.Style},
	} {
		fmt.Fprintf(&b, "  %-12s %s\n", row.name, row.origin)
	}
	if s.Origins.Safety != behavior.OriginBuiltin {
		// Unreachable: the overlay type has no Safety field. Stated anyway,
		// because the day it stops being unreachable is the day someone needs
		// to be told loudly rather than to notice.
		b.WriteString("  !! Safety is not builtin. This should be impossible.\n")
	}
	if len(s.DoctrineNotice) > 0 {
		b.WriteString("\n--- doctrine notices ---\n")
		for _, n := range s.DoctrineNotice {
			fmt.Fprintf(&b, "  %s\n", n)
		}
	}
	return b.String()
}
