// Command dcode is the agentic coding harness.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/aguinelo/dcode/internal/app"
	"github.com/aguinelo/dcode/internal/config"
	"github.com/aguinelo/dcode/internal/loop"
	"github.com/aguinelo/dcode/internal/version"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "dcode: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	var (
		showVersion = flag.Bool("version", false, "print the version and exit")
		workspace   = flag.String("workspace", "", "workspace root (default: current directory)")
		verbose     = flag.Bool("verbose", false, "show successful tool output as well as failures")
		dumpPrompt  = flag.Bool("dump-prompt", false, "print the assembled system prompt and exit")
		showConfig  = flag.String("config", "", "print the effective value of a key and where it came from")
		yes         = flag.Bool("yes", false, "answer every approval automatically (denies, since nobody is asked)")
	)
	flag.Usage = usage
	flag.Parse()

	if *showVersion {
		fmt.Println(version.String())
		return nil
	}

	ws := *workspace
	if ws == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return err
		}
		ws = cwd
	}
	ws, err := filepath.Abs(ws)
	if err != nil {
		return err
	}

	opts, resolved, err := app.FromEnv(os.Getenv, ws)
	if err != nil {
		return err
	}
	opts.DumpPrompt = opts.DumpPrompt || *dumpPrompt

	// A locked value that was overridden must be visible. Ignoring it silently
	// leaves the user believing their change took effect.
	for _, w := range resolved.Warnings {
		fmt.Fprintf(os.Stderr, "warning: %s\n", w)
	}

	if *showConfig != "" {
		return printConfig(resolved, *showConfig)
	}

	// Non-interactive runs deny every crossing rather than granting in
	// silence: with nobody to ask, denial is the only safe reading.
	var approver loop.Approver = &app.ConsoleApprover{In: os.Stdin, Out: os.Stdout}
	if *yes {
		approver = app.DenyAll{}
	}
	emitter := &app.ConsoleEmitter{W: os.Stdout, Verbose: *verbose}

	session, err := app.New(opts, emitter, approver)
	if err != nil {
		return err
	}

	// The audit answer to "what exactly goes to the model". A harness that
	// cannot show its own prompt asks for blind trust in a program with shell
	// access.
	if opts.DumpPrompt {
		fmt.Println(session.Prompt)
		return nil
	}

	task := strings.TrimSpace(strings.Join(flag.Args(), " "))
	if task == "" {
		return fmt.Errorf("no task given. Try: dcode \"list the Go files\"")
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	printHeader(session, ws)

	out, err := session.Engine.Run(ctx, task)
	if err != nil {
		return err
	}

	fmt.Printf("\n%s · %d iteration(s)", out.Reason, out.Iterations)
	if out.Usage.InputTokens > 0 {
		fmt.Printf(" · %d in / %d out", out.Usage.InputTokens, out.Usage.OutputTokens)
		if out.Usage.CacheReadTokens > 0 {
			// Surfaced because it is the only direct evidence that append-only
			// context is paying for itself.
			pct := 100 * out.Usage.CacheReadTokens / max(1, out.Usage.InputTokens)
			fmt.Printf(" · %d%% from cache", pct)
		}
	}
	fmt.Println()
	return nil
}

func printHeader(s *app.Session, ws string) {
	mode := string(s.Options.SandboxMode)
	if s.Options.SandboxMode == "full-access" {
		// The one piece of state where being wrong is dangerous, so it is never
		// quiet.
		mode = "⚠ FULL-ACCESS"
	}
	fmt.Printf("dcode %s · %s · %s · %s\n\n",
		version.Short(), s.Options.Model, mode, ws)
}

func printConfig(r config.Resolved, key string) error {
	v, ok := r.Get(key)
	if !ok {
		fmt.Printf("%s is not set\n\nKnown keys:\n", key)
		for _, k := range r.Keys() {
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

func usage() {
	fmt.Fprintf(os.Stderr, `dcode %s — an agentic coding harness

Usage:
  dcode [flags] <task>

Examples:
  dcode "add a test for the parser"
  dcode --dump-prompt
  dcode --config model.name

Flags:
`, version.Short())
	flag.PrintDefaults()
	fmt.Fprintf(os.Stderr, `
Environment:
  DCODE_API_KEY            model credential (required to run a task)
  DCODE_MODEL              model name (default MiniMax-M3)
  DCODE_TRANSPORT          wire format: openai or anthropic
  DCODE_SANDBOX_MODE       read-only, workspace-write or full-access
  DCODE_APPROVAL_POLICY    untrusted, on-request or never
  DCODE_ALLOW_NETWORK      grant network without asking
`)
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
