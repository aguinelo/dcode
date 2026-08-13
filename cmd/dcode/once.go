package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/aguinelo/dcode/internal/app"
	"github.com/aguinelo/dcode/internal/loop"
	"github.com/aguinelo/dcode/internal/version"
)

// runOnce is the non-interactive form: one task, one exit code, no session.
func runOnce(args []string) error {
	fs := flag.NewFlagSet("dcode", flag.ContinueOnError)
	fs.Usage = usage
	var (
		showVersion = fs.Bool("version", false, "print the version and exit")
		workspace   = fs.String("workspace", "", "workspace root (default: current directory)")
		verbose     = fs.Bool("verbose", false, "show successful tool output as well as failures")
		dumpPrompt  = fs.Bool("dump-prompt", false, "print the assembled system prompt and exit")
		showConfig  = fs.String("config", "", "print the effective value of a key and where it came from")
		yes         = fs.Bool("yes", false, "answer every approval automatically (denies, since nobody is asked)")
	)
	if err := fs.Parse(args); err != nil {
		return err
	}

	if *showVersion {
		fmt.Println(version.String())
		return nil
	}

	ws, err := resolveWorkspace(*workspace)
	if err != nil {
		return err
	}

	opts, resolved, err := loadOptions(ws)
	if err != nil {
		return err
	}
	opts.DumpPrompt = opts.DumpPrompt || *dumpPrompt

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
	// A one-shot run is still a session, and a background command started in it
	// must not outlive the binary. Nothing on unix kills a child when its
	// parent exits — it is reparented and keeps running — so leaving this out
	// is exactly how `dcode "start the server"` would strand a server with
	// nobody left who knows its name.
	defer session.Engine.Close()

	// The audit answer to "what exactly goes to the model". A harness that
	// cannot show its own prompt asks for blind trust in a program with shell
	// access.
	if opts.DumpPrompt {
		fmt.Println(session.Prompt)
		fmt.Print(app.DoctrineAudit(session))
		return nil
	}

	task := strings.TrimSpace(strings.Join(fs.Args(), " "))
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
