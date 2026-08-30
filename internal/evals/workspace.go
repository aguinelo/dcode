package evals

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/aguinelo/dcode/internal/policy"
	"github.com/aguinelo/dcode/internal/tools"
)

// Workspace is a scenario's files on disk, with the product's own tools
// pointed at them.
//
// It exists because the alternative was a canned answer per tool, and the
// canned answer was the string "ok". The model would glob, grep, and list,
// receive "ok" every time, and conclude — correctly, from what it was told —
// that the workspace was empty:
//
//	"The workspace is empty — no files exist, and `grep` for `Summary`
//	 returned `ok`. There's no existing `Summary` type to add a method to.
//	 I'm not going to invent a definition."
//
// That is a model behaving well being scored zero, and it was poisoning every
// multi-round scenario at once.
//
// The tools are the shipped ones rather than stand-ins. A second
// implementation inside the harness would drift, and it would drift exactly
// where it matters: RN-3 makes tool error text a behaviour surface, so a
// scenario measuring recovery from a hand-written error message measures a
// product that does not exist.
type Workspace struct {
	Dir      string
	Registry *tools.Registry
	// Delegate runs a child turn and returns what the parent would have been
	// shown, with the error flag a tool result carries.
	//
	// A field rather than a method, and set by whoever holds the provider —
	// the same shape the product uses, where the tool declares and the loop
	// decides what a child turn is allowed to be. Nil means the harness cannot
	// run one, and then it says so instead of inventing a result.
	Delegate func(ctx context.Context, task, path string, owns []string) (string, bool)
	// Qualify answers done_propose, when the scenario is a qualifying turn.
	//
	// Set from outside for the same reason as Delegate: what happens to a
	// proposal is the loop's business, not a tool's, and the harness holds the
	// half of the product that records one. Nil leaves the registry's own
	// instance to answer, which refuses — and refusing is exactly right for
	// every scenario that is not a qualifying turn.
	Qualify func(ctx context.Context, input json.RawMessage) (string, bool)
	// Mode is the boundary this scenario runs under.
	//
	// workspace-write for almost everything. A qualifying turn takes the
	// product's own read-only, because a turn that could write while working
	// out what it will be measured by can move the thing it is about to be
	// measured against — and measuring that turn under a boundary it never
	// runs under would measure a turn the product does not have.
	Mode     policy.SandboxMode
	state    *tools.State
	resolver *policy.Resolver
}

// NewWorkspace writes files under dir and builds the tools that see them.
//
// Every path is relative and confined: the policy resolver is rooted at dir,
// so a scenario cannot reach out of its own workspace even if the model asks.
func NewWorkspace(dir string, files map[string]string, offered []string) (*Workspace, error) {
	for name, body := range files {
		path := filepath.Join(dir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return nil, err
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			return nil, err
		}
	}

	resolver, err := policy.NewResolver(dir)
	if err != nil {
		return nil, err
	}
	// The fixture's tools, not the whole registry: a tool error may only
	// point at what the model was actually offered, and the scenario decides
	// that. Passing the registry here is how records-before-compaction came
	// to be told "use glob" thirty times by a session with no glob.
	state := tools.NewState(resolver, tools.DefaultLimits(), offered)

	return &Workspace{
		Dir: dir, state: state, resolver: resolver,
		Registry: ProductRegistry(), Mode: policy.ModeWorkspaceWrite,
	}, nil
}

// ProductRegistry is the product's tool suite, definitions and all.
//
// One source for both halves of a scenario: what the model is offered, and what
// runs when it calls. They were separate, and they drifted — a fixture declared
// `plan` with a free-string `state` field while the product declares `status`
// with an enum, so the model dutifully sent `{"state":"in_progress"}`, got an
// error, and spent every remaining round guessing at a shape the fixture had
// described wrongly.
//
// bash carries a zero Runner because only its definition is wanted here.
// Execute intercepts the name before dispatch, so the Runner is never reached.
func ProductRegistry() *tools.Registry {
	return tools.NewRegistry(
		tools.Read{}, tools.Write{}, tools.Edit{},
		tools.Glob{}, tools.Grep{}, tools.Symbol{},
		tools.Plan{}, tools.Bash{}, &tools.Explore{}, tools.Fetch{},
		// Fixed provenance so a scenario's result does not change with the
		// clock: a memory stamped with today's date would make two identical
		// runs produce different files, and the contracts read those files.
		tools.Remember{Commit: "eva1c0m", Today: "2026-01-01"},
		// Definition only. A DonePropose with no Submit is what the product
		// offers a turn that is not qualifying, and it answers by refusing —
		// so a scenario that declares the tool without being a qualifying turn
		// gets the product's own refusal rather than a harness invention.
		tools.DonePropose{},
	)
}

// shellRefusal is what a scenario answers when the model reaches for the shell.
//
// The harness will not execute model-written commands, and saying so is the
// only honest answer available. Pretending the command ran and returned
// nothing is what produced "shell responses look empty. Let me try again with
// different approaches" — a model burning its remaining rounds on a lie.
//
// This does not soften any contract about shell use. Every one of those is
// judged on the *reach*, which has already happened by the time this is read.
const shellRefusal = "the eval harness does not execute shell commands. " +
	"Use the dedicated tools for anything you can do with them, and state what you could not check."

// fetchRefusal is what a scenario answers when the model reaches the network.
const fetchRefusal = "the eval harness does not reach the network. " +
	"Answer from what is in the workspace, and state what you could not look up."

// delegationRefusal is what a scenario answers when nothing can run a child.
//
// Kept, because a harness without a delegator has to say so rather than invent
// a result — the shellRefusal lesson, and it has not stopped being true. What
// changed is that it is now the exception rather than the rule.
const delegationRefusal = "the eval harness does not run delegated turns. " +
	"Do the reading yourself with the tools you have, and say what you could not cover."

// BeginCycle and UndoCycle expose the product's own rollback to a scenario
// that runs a verification cycle. Through the same State the tools write
// through, so what is put back is what they wrote.
func (w *Workspace) BeginCycle() { w.state.BeginCycle() }

// BeginTurn starts the set of changes a rollback can reach.
func (w *Workspace) BeginTurn() { w.state.BeginTurn() }

// UndoCycle puts back what this cycle wrote.
func (w *Workspace) UndoCycle() (restored, refused []string, err error) {
	return w.state.UndoCycle()
}

// Execute runs one call and returns what the model would have been shown.
//
// A tool that fails returns its message with isErr set, exactly as the loop
// would forward it: the message is the product's own, which is the point.
func (w *Workspace) Execute(ctx context.Context, name string, input json.RawMessage) (output string, isErr bool) {
	switch name {
	case "bash":
		return shellRefusal, true
	case "fetch":
		// Same reasoning as the shell. A contract that reached the network
		// would depend on whatever happened to be online that afternoon, and a
		// measurement nobody can reproduce is not a measurement.
		return fetchRefusal, true
	case "done_propose":
		// A qualifying turn's proposal is recorded by the product's own
		// recorder, which is also what writes the answer the model reads back.
		// Left to the registry it would be refused, which is right everywhere
		// else and would make the qualifier contracts unmeasurable.
		if w.Qualify != nil {
			return w.Qualify(ctx, input)
		}
	case "explore":
		// A delegated call is answered by running a child turn, when the
		// harness has been given something that can run one.
		//
		// It used to be refused outright, and the refusal was honest about not
		// running a sub-agent. What it was not honest about was its effect:
		// the message told the model to "do the reading yourself", the model
		// believed it, and every later delegation in that run disappeared. A
		// contract about reaching for delegation was measuring a harness that
		// had just talked the model out of it — visible in three measurements
		// that spread twenty-five points and in the digest of every failing
		// run, each of which said in as many words that the harness does not
		// run delegated turns.
		if w.Delegate == nil {
			return delegationRefusal, true
		}
		var in struct {
			Task string   `json:"task"`
			Path string   `json:"path"`
			Owns []string `json:"owns"`
		}
		if err := json.Unmarshal(input, &in); err != nil {
			return fmt.Sprintf("explore: %v", err), true
		}
		return w.Delegate(ctx, in.Task, in.Path, in.Owns)
	}
	tool, ok := w.Registry.Get(name)
	if !ok {
		// Not a tool the product has. Nothing invented reaches the model, so
		// this is a scenario offering something the registry does not carry.
		return fmt.Sprintf("unknown tool %q", name), true
	}
	// The gate the loop applies, applied here too. Calling Execute directly
	// skipped it, and a tool resolves a path without asking whether it may:
	// `read` on /etc/passwd came back with the file. The harness had handed a
	// real model unrestricted read access to the machine it was running on.
	//
	// Declare-then-evaluate rather than a path check, because that is the
	// structure the product relies on — the tool says what it would touch, and
	// something else decides.
	req, err := tool.Declare(input)
	if err != nil {
		return err.Error(), true
	}
	if v := w.evaluate(req); v.Decision == policy.DecisionDeny {
		return v.Reason, true
	}

	res, err := tool.Execute(ctx, input, w.state)
	if err != nil {
		return err.Error(), true
	}
	return res.Output, res.IsError
}

// evaluate resolves the declared paths and asks the policy, exactly as the
// loop does.
//
// workspace-write with approvals off: anything inside the workspace runs,
// anything outside is denied rather than queued for a person who is not there.
// A measurement that paused for approval would measure nothing at all.
func (w *Workspace) evaluate(req policy.Request) policy.Verdict {
	resolved := policy.Request{Tool: req.Tool, Network: req.Network, Command: req.Command}
	for _, a := range req.Paths {
		acc, err := w.resolver.Resolve(a.Path, a.Write)
		if err != nil {
			// Cannot tell is never allow — the same reading the loop takes.
			resolved.Paths = append(resolved.Paths, policy.Access{Path: a.Path, Write: a.Write})
			continue
		}
		resolved.Paths = append(resolved.Paths, acc)
	}
	// The fixture grants the network and carries no rules: a scenario measures
	// behaviour, and a question nobody is there to answer would measure the
	// harness instead.
	return policy.Evaluate(resolved, w.Mode, policy.PolicyNever,
		policy.Rules{}, policy.GrantedNetwork{}, w.resolver.InWorkspace)
}

// WorkspaceRoot is the miniature repository every scenario explores.
//
// Shared rather than per-fixture because the scenarios were written against
// one codebase and say so: they name `stats.go`, `internal/config/toml.go`,
// `internal/version/version.go`. Thirty copies of the same file would drift,
// and a scenario asking about a type that three fixtures define differently
// is a scenario measuring which fixture it landed in.
//
// A fixture overlays its own `files/` on top when its question needs something
// the shared workspace should not carry.
const WorkspaceRoot = "testdata/workspace"

// loadFiles reads a tree into slash-separated relative paths and content.
//
// Slash-separated so a fixture reads the same on any platform and a scenario
// note can name a path the way the task does.
func loadFiles(root string) (map[string]string, error) {
	if _, err := os.Stat(root); os.IsNotExist(err) {
		return nil, nil
	}
	out := map[string]string{}
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		body, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		out[filepath.ToSlash(rel)] = string(body)
		return nil
	})
	if err != nil {
		return nil, err
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("%s exists and is empty, which is a workspace the model will report as missing", root)
	}
	return out, nil
}

// overlay lays a fixture's own files over the shared workspace.
//
// The fixture wins. A scenario that needs a file to be broken in a particular
// way says so in its own directory, and the shared copy stays the one every
// other scenario reads.
func overlay(base, own map[string]string) map[string]string {
	out := make(map[string]string, len(base)+len(own))
	for k, v := range base {
		out[k] = v
	}
	for k, v := range own {
		out[k] = v
	}
	return out
}
