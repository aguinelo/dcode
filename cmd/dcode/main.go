// Command dcode is the agentic coding harness.
//
// Three shapes, one binary. `dcode serve` is the daemon that owns sessions;
// `dcode tui` is the terminal client; `dcode "<task>"` is the one-shot run for
// scripts and CI. The one-shot embeds the engine directly rather than speaking
// the protocol to itself, because a pipeline does not need a session that
// outlives the command.
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/aguinelo/dcode/internal/app"
	"github.com/aguinelo/dcode/internal/config"
	"github.com/aguinelo/dcode/internal/tui"
	"github.com/aguinelo/dcode/internal/version"
)

func main() {
	if err := dispatch(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "dcode: %v\n", err)
		os.Exit(1)
	}
}

// dispatch routes to a subcommand.
//
// The subcommand must be the first token. Accepting it after flags would make
// `dcode --workspace x serve` and `dcode serve --workspace x` two different
// parses of the same intent, and the flag package cannot tell them apart.
func dispatch(args []string) error {
	if len(args) > 0 {
		switch args[0] {
		case "serve":
			return runServe(args[1:])
		case "tui":
			return runTUI(args[1:])
		case "update":
			return runUpdate(args[1:])
		case "login":
			return runLogin(args[1:])
		case "config":
			return runConfig(args[1:])
		case "help", "--help", "-h":
			usage()
			return nil
		case "version", "--version":
			fmt.Println(version.String())
			return nil
		}
	}

	// No arguments and a terminal means a person, and a person wants the TUI.
	// No arguments and a pipe means a script that forgot its task, which is an
	// error rather than an interface.
	if len(args) == 0 && isTerminal(os.Stdout) {
		return runTUI(nil)
	}
	return runOnce(args)
}

// resolveWorkspace turns a possibly empty flag into an absolute path.
func resolveWorkspace(flagValue string) (string, error) {
	ws := flagValue
	if ws == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return "", err
		}
		ws = cwd
	}
	return filepath.Abs(ws)
}

// loadOptions resolves the whole configuration chain for a workspace and
// reports anything an administrator locked away from the user.
func loadOptions(ws string) (app.Options, config.Resolved, error) {
	opts, resolved, err := app.FromEnv(os.Getenv, ws)
	if err != nil {
		return opts, resolved, err
	}
	// A locked value that was overridden must be visible. Ignoring it silently
	// leaves the user believing their change took effect.
	for _, w := range resolved.Warnings {
		fmt.Fprintf(os.Stderr, "warning: %s\n", w)
	}
	return opts, resolved, nil
}

func printConfig(r config.Resolved, key string) error {
	v, ok := r.Get(key)
	if !ok {
		// The whole surface, not only what happens to be resolved: a key with
		// no value set is exactly the one someone is looking for here.
		fmt.Printf("%s is not set\n\nKnown keys:\n", key)
		names := make([]string, 0, len(config.KnownKeys))
		for k := range config.KnownKeys {
			names = append(names, k)
		}
		sort.Strings(names)
		for _, k := range names {
			fmt.Printf("  %s\n", k)
		}
		return nil
	}
	fmt.Printf("%s = %s\n  from: %s (%s)\n", v.Key, v.Value, v.Source, v.Origin)
	if v.Locked {
		fmt.Println("  locked by the administrator")
	}
	return nil
}

// usage prints the CLI help in the interface language.
//
// Resolved here rather than passed in, because usage runs before anything else
// exists — including the configuration chain, when the argument is bad enough.
// It is the one place in the command layer that reads the environment for this,
// and it is the earliest thing that has to work.
func usage() {
	t := tui.Text(tui.Resolve(os.Getenv))
	fmt.Fprintf(os.Stderr, t.Usage, version.Short())
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
