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
	"sync/atomic"
	"time"

	"github.com/aguinelo/dcode/internal/behavior"
	"github.com/aguinelo/dcode/internal/config"
	ce "github.com/aguinelo/dcode/internal/contextengine"
	"github.com/aguinelo/dcode/internal/credential"
	"github.com/aguinelo/dcode/internal/loop"
	"github.com/aguinelo/dcode/internal/memory"
	"github.com/aguinelo/dcode/internal/policy"
	"github.com/aguinelo/dcode/internal/protocol"
	"github.com/aguinelo/dcode/internal/provider"
	"github.com/aguinelo/dcode/internal/sandbox"
	"github.com/aguinelo/dcode/internal/tools"
	"github.com/aguinelo/dcode/internal/vcs"
	"github.com/aguinelo/dcode/internal/workspace"
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
	// BudgetNotice switches the occupancy warning to the model on.
	BudgetNotice bool
	// VerifyCommand is the command that COUNTS as verification. Explicit,
	// because "some bash ran" would count an `ls`.
	VerifyCommand string
	// DoneTimeout caps one criterion. A check that never finishes is not a
	// check, and hanging the turn is worse than reporting the overrun.
	DoneTimeout time.Duration
	// DoneEnabled switches re-entry on unmet criteria on. Off restores the old
	// behaviour: the turn ends when the model stops calling tools, met or not.
	DoneEnabled bool
	// Qualify makes this the session that works out what "done" means for
	// LoopSpec instead of the one that does the work. Forces plan mode and
	// offers done_propose; nothing else changes.
	Qualify bool
	// LoopSpec is a directory holding a tasks.md, read as this session's
	// definition of done instead of done.toml. Empty is the ordinary case.
	LoopSpec string
	// Protect are globs added to whatever the spec declares as protected.
	Protect []string
	// DoneFile overrides where the definition of done is declared.
	DoneFile string
	// MaxStallCycles is how many cycles without progress end a turn.
	MaxStallCycles int
	// Delegate switches the read-only delegation tool on. Off removes it from
	// the registry; nothing else changes.
	Delegate bool
	// Unreadable are paths this session may not read at all — a credential
	// store put out of reach. Resolved at the edge, from configuration.
	Unreadable []string
	// Granted are unix sockets named as reachable, and Writable are paths
	// named as writable outside the workspace. Both resolved at the edge.
	Granted  []string
	Writable []string
	// DelegateMaxIterations caps a child turn.
	DelegateMaxIterations int
	// DelegateMaxResultBytes caps the child's report.
	DelegateMaxResultBytes int
	// WorkspaceGates switches the inventory of the project's declared checks
	// on. On by default: the probe reads two files, runs nothing, and costs one
	// read at session open. A cheap probe that ships off is a probe nobody
	// turns on.
	//
	// It exists for the repository with a seventy-target Makefile, where the
	// cap still leaves a list nobody reads.
	WorkspaceGates bool
	// InstructionNotice switches the session-start warning about untranslated
	// instruction files on. It warns; it never blocks.
	InstructionNotice bool
	// InstructionForeign are the files treated as a shared format, and so as
	// candidates for translation. DCODE.md is never one of them.
	InstructionForeign string
	// Fetch switches the network tool on. Off by default: the network is the
	// one capability whose absence nobody has to work around, so it is the one
	// that earns an opt-in rather than an opt-out.
	Fetch bool
	// FetchMaxBytes caps a fetched document.
	FetchMaxBytes int
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
	// History seeds the conversation, for a session continuing a recorded one.
	//
	// It is not compacted here. The engine checks at the top of its first
	// iteration, before any request, so a seeded history that is too large is
	// handled by the same code that handles one that grew — and compacting
	// here would be a second implementation of the thing most worth having
	// exactly one of.
	History []ce.Message

	// Steer hands the running turn what the person said without ending it.
	//
	// It rides here rather than as a parameter for the same reason History
	// does: the engine is built before the session that owns the queue exists,
	// so this is a closure bound late, exactly like the emitter and the
	// approver. Nil is a session nobody can steer, which is every non-daemon
	// path and was the only behaviour until now.
	Steer func() string

	// Memory reads what earlier sessions in this workspace learned. Off is the
	// product from before this existed.
	Memory bool
	// MemoryMax is how many memories reach the prefix. See the .config spec:
	// the default is a starting value, not a defended number.
	MemoryMax int
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
			"sandbox.allow_network":   "true",
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
			"tools.fetch_enabled":      "false",
			"tools.fetch_max_bytes":    "262144",
			// The model is told how much of its budget is gone before it runs
			// out, not after. Off leaves the post-compaction notice as the only
			// signal, which is what it was.
			"budget.notice":      "true",
			"doctrine.enabled":   "true",
			"doctrine.max_bytes": "16384",
		}},
	}

	// Files come before the environment: the chain is default < user file <
	// project file < environment < flag < locked, and Resolve orders by source
	// rather than by position, so loading order here only has to be complete.
	roots, err := config.DiscoverRoots(env)
	if err != nil {
		return config.Resolved{}, err
	}
	fileLayers, err := config.FileLayers(roots, workspace, requirementsPath(env, roots))
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
		Env:                    env,
		Reminders:              r.Bool("behavior.reminders_enabled", true),
		ShowReasoning:          r.Bool("behavior.show_reasoning", true),
		Instructions:           r.Bool("behavior.instructions_enabled", true),
		Skills:                 r.Bool("behavior.skills_enabled", true),
		EditEchoDiff:           r.String("tools.edit_echo_diff", tools.EchoDiffMulti),
		SymbolMaxMatches:       r.Int("tools.symbol_max_matches", 200),
		Fetch:                  r.Bool("tools.fetch_enabled", false),
		FetchMaxBytes:          r.Int("tools.fetch_max_bytes", 262144),
		BudgetNotice:           r.Bool("budget.notice", true),
		VerifyCommand:          r.String("verify.command", ""),
		DoneTimeout:            parseDuration(r.String("done.timeout", "10m"), 10*time.Minute),
		DoneEnabled:            r.Bool("done.enabled", true),
		DoneFile:               r.String("done.file", ""),
		MaxStallCycles:         r.Int("limits.max_stall_cycles", 2),
		Delegate:               r.Bool("delegate.enabled", true),
		Granted:                sandbox.Paths(r.String("sandbox.sockets", ""), env),
		Writable:               sandbox.Paths(r.String("sandbox.writable", ""), env),
		Unreadable:             sandbox.Unreadable(r.String("sandbox.unreadable", ""), env, sandbox.Paths(r.String("sandbox.sockets", ""), env)),
		DelegateMaxIterations:  r.Int("delegate.max_iterations", 100),
		DelegateMaxResultBytes: r.Int("delegate.max_result_bytes", 32768),
		WorkspaceGates:         r.Bool("workspace.gates", true),
		InstructionNotice:      r.Bool("instruction.notice", true),
		InstructionForeign: r.String("instruction.foreign",
			strings.Join(ForeignDefault, ",")),
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
		AllowNetwork:      r.Bool("sandbox.allow_network", true),
		Parallel:          r.Int("limits.parallel", 4),
		Memory:            r.Bool("memory.enabled", true),
		MemoryMax:         r.Int("memory.max_entries", memory.DefaultMax),
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
	// State is the other half of Registry: every tool's Execute takes one, so
	// a registry handed out without it cannot be called. It is also what owns
	// the background processes, which is why "a process dies with its session"
	// is a consequence of this chain rather than a cleanup step.
	State   *tools.State
	Prompt  string
	Options Options
	// Origins is where each doctrine section came from, and Notices is what
	// the overlay loader refused to do silently. Both exist for the audit:
	// an invisible replacement would be worse than the immutability it
	// replaces (RN-12).
	// Notice is what the session has to say at the start about instruction
	// files written for another tool. Empty when there is nothing to say.
	Notice         string
	Origins        behavior.SectionOrigins
	DoctrineNotice []behavior.Notice
	// ContextWindow is what the provider reports for this model.
	ContextWindow int
	// Proposals is where a qualifying session keeps what the model proposed,
	// until the loop takes it. Nil in every other session.
	Proposals *Proposals
	// Standing is what the user has already permitted. Carried on the session
	// so the daemon attaches the same record the sandbox is asking, rather than
	// loading a second copy that could answer differently.
	Standing *StandingGrants
}

// New wires a session.
func New(opts Options, emitter loop.Emitter, approver loop.Approver) (*Session, error) {
	resolver, err := policy.NewResolver(opts.Workspace)
	if err != nil {
		return nil, err
	}

	// What the user has already permitted. Loaded before the sandbox, because
	// the sandbox asks it per command: a grant given at the first crossing has
	// to take effect for that crossing, not after a restart.
	grantEnv := opts.Env
	if grantEnv == nil {
		grantEnv = os.Getenv
	}
	grantRoots, err := config.DiscoverRoots(grantEnv)
	if err != nil {
		return nil, err
	}
	standing, err := NewStandingGrants(grantRoots.Config, opts.Workspace)
	if err != nil {
		return nil, err
	}

	// The sandbox is established before anything can run. Failing here is
	// deliberate: a session that cannot confine its own commands should not
	// start at all.
	sb, err := sandbox.New(sandbox.Config{
		Backend: opts.Backend,
		// Configuration says yes, or the user did — and the second can happen
		// while the session is running.
		AllowNetwork: func() bool { return opts.AllowNetwork || standing.NetworkNow() },
		// Without these a compiled language cannot build inside the sandbox,
		// so the agent can change files and never check them.
		Scratch:    sandbox.Scratch(opts.Env),
		Sockets:    sandbox.LocalSockets(opts.Env),
		Unreadable: opts.Unreadable,
		Granted:    opts.Granted,
		Writable:   opts.Writable,
	}, opts.SandboxMode)
	if err != nil {
		return nil, err
	}

	toolLimits := tools.DefaultLimits()
	toolLimits.EditEchoDiff = opts.EditEchoDiff
	toolLimits.SymbolMaxMatches = opts.SymbolMaxMatches
	// Frozen here, with the instruction chain and for the same reason: a fact
	// that changed halfway through a session is worse than one the model knows
	// is a snapshot. The read is bounded and a directory that is not a
	// repository produces nothing at all.
	//
	// Read before the tools rather than beside the prompt, because `remember`
	// stamps a memory with the same commit the prefix describes. Two readings of
	// git in one session can disagree, and disagreeing about where the session
	// is is worse than not knowing.
	repo := vcs.Read(context.Background(), opts.Workspace)

	// explore is registered as a pointer because its delegator is the engine,
	// which does not exist yet: the registry is what the engine is built with.
	// The knot is tied after.
	explore := &tools.Explore{}
	// live is the second knot, and it exists for the same reason: the shell
	// runs under the boundary the session is in RIGHT NOW, and the session's
	// mode lives on the engine that does not exist yet either.
	live := &liveMode{fallback: opts.SandboxMode}
	toolset := []tools.Tool{
		tools.Read{}, tools.Write{}, tools.Edit{},
		tools.Glob{}, tools.Grep{}, tools.Symbol{},
		tools.Bash{
			Runner:     sandbox.Runner{Sandbox: sb, Mode: live.get},
			Background: background{sandbox.Runner{Sandbox: sb, Mode: live.get}},
			Workdir:    opts.Workspace,
			Timeout:    120 * time.Second,
		},
		tools.Plan{}, tools.Process{},
	}
	if opts.Fetch {
		// Off unless asked for. Reaching the network is the one capability
		// whose absence nobody has to work around — every other tool here is
		// needed to do the job at all — so it is the one that earns an opt-in.
		toolset = append(toolset, tools.Fetch{Limit: opts.FetchMaxBytes})
	}
	if opts.Delegate {
		toolset = append(toolset, explore)
	}
	if opts.Memory {
		toolset = append(toolset, tools.Remember{
			Commit: headOf(repo),
			Today:  time.Now().Format(time.DateOnly),
		})
	}
	var proposals *Proposals
	if opts.Qualify && opts.LoopSpec != "" {
		// Only here. A tool that can redefine done, within reach of a working
		// turn, is the shortest way out of a loop — the agent rewrites the
		// ruler instead of meeting it.
		proposals = &Proposals{}
		toolset = append(toolset, QualifyingTool(opts.LoopSpec, proposals))
	}
	registry := tools.NewRegistry(toolset...)
	// After the registry, because the session's tool names are what an error
	// message may point at. A tool error naming a capability this build does
	// not carry sends the model somewhere that does not exist.
	state := tools.NewState(resolver, toolLimits, registry.Names())

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
		// An instruction from the administrator's locked file is marked as
		// such. Without the mark, the precedence table had a top row nothing
		// could ever occupy.
		for i := range instructions {
			if instructions[i].Source == behavior.SourceLocked {
				instructions[i].Locked = true
			}
		}
	}

	// What earlier sessions in this workspace learned, read back as the weakest
	// instruction there is. Frozen here with the chain and for the same reason:
	// a memory written during this session must not change this session's
	// prefix, which is what the cache is keyed on.
	if opts.Memory {
		learned, merr := memory.Read(opts.Workspace)
		if merr != nil {
			// A memory nobody can read is worth saying and not worth stopping
			// for: refusing to open a session over it would be the memory
			// holding the product hostage, which is what the record already
			// refuses to do.
			emitter.Emit(protocol.EventSessionError, protocol.Error{
				Code: "memory_unreadable", Message: merr.Error(),
			})
		} else if block := memory.Render(learned, opts.MemoryMax, knownCommits(opts.Workspace, learned)); block != "" {
			instructions = append(instructions, behavior.Instruction{
				Source: behavior.SourceLearned,
				Scope:  memory.FileName,
				Text:   block,
			})
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

	// The definition of done is read once, at session creation, like the
	// instruction chain and for the same reason. A criterion written mid-session
	// must not change what the turn is measured against.
	doneSet, err := sessionDoneSet(opts)
	if err != nil {
		return nil, err
	}

	// RN-10's second half. The instruction is not modified and not dropped —
	// the rest of it is legitimate, and discarding a whole file over one
	// sentence is the silent-filter failure refused everywhere else. What
	// changes is that the attempt is now visible.
	safetyNotices := behavior.SafetyClaims(instructions)

	prompt, err := behaviorBuild(registry.Names(), instructions, behavior.Index(skills), overlay, CredentialName(opts), repo,
		declaredGates(opts.Workspace, opts.WorkspaceGates))
	if err != nil {
		return nil, err
	}

	window, _ := p.Window(opts.Model)
	ctxCfg := ce.DefaultConfig()
	ctxCfg.Window = window

	engine := loop.New(loop.Config{
		Provider: p, Tools: registry, State: state,
		Emitter: emitter, Approver: approver,
		Limits: opts.Limits, Mode: opts.SandboxMode, Policy: opts.Policy,
		Model: opts.Model, Parallel: opts.Parallel, CtxConfig: ctxCfg,
		Steer:                  opts.Steer,
		BudgetNotice:           opts.BudgetNotice,
		DelegateMaxIterations:  opts.DelegateMaxIterations,
		DelegateMaxResultBytes: opts.DelegateMaxResultBytes,
		Done:                   doneSet,
		DoneEnabled:            opts.DoneEnabled,
		MaxStallCycles:         opts.MaxStallCycles,
		DoneTimeout:            opts.DoneTimeout,
		RunCriterion:           criterionRunner(sb, opts),
		WrittenPaths:           state.Written,
		WriteSeq:               state.WriteSeq,
		// Stamped by the product after the turn, never asked of the model: a
		// prompt requesting the marker would be a prompt hoping for it, and the
		// digests have to be of the bytes actually read.
		AfterTurn: func(written []string) {
			StampGenerated(opts.Workspace, foreignFiles(opts.InstructionForeign), written)
		},
		Rules: opts.Rules,
		// The same answer the sandbox is given, so the question and the
		// boundary cannot disagree: asking about a network the sandbox already
		// opened is a question whose answer changes nothing.
		NetworkGrant: policy.NetworkGrantFunc(
			func() bool { return opts.AllowNetwork || standing.NetworkNow() }),
		Summarise:        summariser(p, opts.Model),
		Skills:           skills,
		InstructionChain: chain,
		ReadFile:         readFileText,
		Reminders:        opts.Reminders,
		ShowReasoning:    opts.ShowReasoning,
	}, ce.Session{Instructions: prompt, History: opts.History})

	// The knot: the tool needed the engine, and the engine needed the registry
	// the tool is in. Tied here, once, where both exist.
	explore.Delegator = engine
	live.follow(engine)

	notice := ""
	// A model nobody measured is usable and must say so. Reading a difference
	// in behaviour as a defect in dcode is the cost of not saying it.
	if resolvedFamily(opts) == provider.GenericName {
		notice = provider.GenericWarning
	}
	if opts.InstructionNotice {
		notice = InstructionNotice(opts.Workspace,
			foreignFiles(opts.InstructionForeign), registry.Names())
	}

	return &Session{
		Engine: engine, Registry: registry, State: state, Prompt: prompt, Options: opts,
		Proposals:      proposals,
		Standing:       standing,
		Notice:         notice,
		ContextWindow:  window,
		Origins:        overlay.Origins(),
		DoctrineNotice: append(overlayNotices, safetyNotices...),
	}, nil
}

// Families are the adaptations this build supports, in registration order.
func Families() []provider.Family {
	return []provider.Family{provider.MiniMaxM3{}, provider.Claude{}, provider.Generic{}}
}

// CredentialName is the family this model belongs to.
//
// It also names the credential store, which is why it is called that: the
// family rather than the model, because `MiniMax-M3` and a later `MiniMax-M4`
// reach the same account at the same provider.
//
// It answers the prompt's question too — RN-8 puts the FORMULATION with the
// family — and the same resolution serves both because it is the same fact.
// resolvedFamily is the family this session will actually use.
func resolvedFamily(opts Options) string { return CredentialName(opts) }

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
func behaviorBuild(toolNames []string, instructions []behavior.Instruction, index []behavior.SkillIndexEntry, overlay behavior.DoctrineOverlay, family string, repo *behavior.Repo, ws *behavior.Workspace) (string, error) {
	return behavior.Build(behavior.Prompt{
		Doctrine:     behavior.DefaultDoctrine(toolNames).Apply(overlay),
		Tools:        toolNames,
		Instructions: instructions,
		SkillIndex:   index,
		Repo:         repo,
		Workspace:    ws,
	}, behavior.FormulationFor(family))
}

// declaredGates probes what the project says it checks itself with, capped and
// saying so when it cuts.
//
// Returns nil when the key is off or nothing was declared. Nil renders nothing,
// and nothing is right: a project with no declared gate is ordinary, and the
// prefix must not claim it declares none — it must simply not mention them.
func declaredGates(dir string, enabled bool) *behavior.Workspace {
	if !enabled {
		return nil
	}
	found := workspace.Probe(context.Background(), dir)
	if len(found) == 0 {
		return nil
	}
	ws := &behavior.Workspace{}
	if len(found) > workspace.MaxGates {
		found = found[:workspace.MaxGates]
		ws.Truncated = true
	}
	for _, g := range found {
		ws.Gates = append(ws.Gates, behavior.Gate{Name: g.Name, Command: g.Command, Source: g.Source})
	}
	return ws
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
		// The Retry-After header travels with the status. Read here because this
		// is the only place holding the response; classification stays pure.
		if pe := provider.ClassifyStatus(resp.StatusCode, string(body), resp.Header.Get("Retry-After")); pe != nil {
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
		{"Practices", s.Origins.Practices},
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

// requirementsPath is where the administrator's locked configuration lives.
//
// It is read from the environment rather than from the configuration chain, and
// it has to be: a locked file whose location the chain decides is a locked file
// the user can move. The environment variable is set by whatever deployed the
// machine — an MDM profile, a container image, a shell profile the user does
// not own — and pointing it somewhere else is a decision at that level, not at
// this one.
//
// Absent, there is no locked layer at all, which is the normal case for a
// single user on their own laptop.
func requirementsPath(env func(string) string, roots config.Roots) string {
	if p := strings.TrimSpace(env("DCODE_REQUIREMENTS_FILE")); p != "" {
		return p
	}
	// The default sits beside the user's configuration but is not the same
	// file: an organisation drops it in, and nothing dcode does writes there.
	p := filepath.Join(roots.Config, config.RequirementsFileName)
	if _, err := os.Stat(p); err != nil {
		return ""
	}
	return p
}

// background ties a concrete *sandbox.Proc to the Handle the tools package
// asks for.
//
// Go returns are not covariant, so sandbox.Runner cannot satisfy
// tools.BackgroundRunner directly without importing tools — and that import
// would point the wrong way: the boundary must not be described in terms of
// the thing it confines. One adapter here costs less than that inversion, and
// this file already ties the one other knot of its kind.
type background struct{ sandbox.Runner }

func (b background) Start(ctx context.Context, workdir, command string) (tools.Handle, error) {
	p, err := b.Runner.Start(ctx, workdir, command)
	if err != nil {
		return nil, err
	}
	return p, nil
}

// knownCommits asks the repository which of the memories' commits it still has.
//
// Nil whenever nothing could be asked, and the renderer reads that as "we did
// not look" rather than "they are gone". A memory marked stale because git was
// missing would be the heuristic deciding on no evidence at all.
func knownCommits(workspace string, f memory.File) map[string]bool {
	var shas []string
	seen := map[string]bool{}
	for _, e := range f.Entries {
		if e.Commit == "" || seen[e.Commit] {
			continue
		}
		seen[e.Commit] = true
		shas = append(shas, e.Commit)
	}
	return vcs.Known(context.Background(), workspace, shas)
}

// headOf is the commit a memory written now was true at.
//
// Taken from the snapshot the prefix already describes rather than asked of git
// again: the two have to agree about where the session is. The log's first line
// is "<short sha> <subject>", and the short sha is what `git cat-file` accepts
// when the memory is checked for staleness later.
func headOf(r *behavior.Repo) string {
	if r == nil || len(r.Commits) == 0 {
		return ""
	}
	sha, _, _ := strings.Cut(r.Commits[0], " ")
	return sha
}

// liveMode answers "which boundary is the shell under right now".
//
// The sandbox used to be handed the mode as a value, captured when the session
// was built. `/mode auto` then changed the policy's answer and nothing else:
// the verdict said allow, the badge said auto, and the write still came back
// EPERM because the OS was still enforcing what the session started with. A
// mode whose whole promise is "no boundary" left one standing.
//
// Held through an atomic rather than a plain field. The knot is tied on the
// construction goroutine and read on every turn's, which is exactly the shape
// of the SetMode race this repository fixed the same day — a plain field would
// work in practice and be wrong under -race, and "works in practice" is what
// that race said too.
type liveMode struct {
	engine   atomic.Pointer[loop.Engine]
	fallback policy.SandboxMode
}

// follow ties this to the engine that owns the session's mode.
func (l *liveMode) follow(e *loop.Engine) { l.engine.Store(e) }

// get is the source the sandbox runner asks once per command.
func (l *liveMode) get() policy.SandboxMode {
	if e := l.engine.Load(); e != nil {
		m, _ := e.Mode()
		return m
	}
	// Before the knot: the mode the session was asked for. Not read-only —
	// this is the mode that was chosen, and refusing it here would break every
	// command that ran before the engine existed rather than fail safe.
	return l.fallback
}
