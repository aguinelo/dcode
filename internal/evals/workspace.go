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
	state    *tools.State
}

// NewWorkspace writes files under dir and builds the tools that see them.
//
// Every path is relative and confined: the policy resolver is rooted at dir,
// so a scenario cannot reach out of its own workspace even if the model asks.
func NewWorkspace(dir string, files map[string]string) (*Workspace, error) {
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
	state := tools.NewState(resolver, tools.DefaultLimits())

	return &Workspace{
		Dir:   dir,
		state: state,
		Registry: tools.NewRegistry(
			tools.Read{}, tools.Write{}, tools.Edit{},
			tools.Glob{}, tools.Grep{}, tools.Symbol{},
			tools.Plan{},
			// bash is deliberately absent, and shellRefusal below is why.
		),
	}, nil
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

// Execute runs one call and returns what the model would have been shown.
//
// A tool that fails returns its message with isErr set, exactly as the loop
// would forward it: the message is the product's own, which is the point.
func (w *Workspace) Execute(ctx context.Context, name string, input json.RawMessage) (output string, isErr bool) {
	if name == "bash" {
		return shellRefusal, true
	}
	tool, ok := w.Registry.Get(name)
	if !ok {
		// Not a tool the product has. Nothing invented reaches the model, so
		// this is a scenario offering something the registry does not carry.
		return fmt.Sprintf("unknown tool %q", name), true
	}
	res, err := tool.Execute(ctx, input, w.state)
	if err != nil {
		return err.Error(), true
	}
	return res.Output, res.IsError
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
