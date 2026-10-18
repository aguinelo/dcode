package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/aguinelo/dcode/internal/app"
	"github.com/aguinelo/dcode/internal/config"
	"github.com/aguinelo/dcode/internal/protocol"
	"github.com/aguinelo/dcode/internal/tui"
	"github.com/aguinelo/dcode/pkg/client"
)

// runOrchestrate opens the terminal client on the orchestrator of the project
// this folder belongs to (orchestration .i step 13, decision 4). `dcode` on
// its own stays the ordinary conversation.
//
// It attaches to a running daemon and never starts one of its own: the
// orchestrator lives in the daemon and outlives this terminal, and one
// embedded here would end with it — taking the children with it.
//
// Spec: docs/specs/architecture/orchestration/202610132000-orchestration.p.spec.md §5.1, §11.
func runOrchestrate(args []string) error {
	fs := flag.NewFlagSet("dcode orchestrate", flag.ContinueOnError)
	socket := fs.String("socket", "", "the daemon to attach to (default: the usual socket)")
	workspace := fs.String("workspace", "", "any folder of the project (default: current directory)")
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "dcode orchestrate — open this project's orchestrator\n\n"+
			"Needs a running `dcode serve` with orchestrator.enabled and record.enabled.\n\nFlags:\n")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() > 0 {
		return fmt.Errorf("dcode orchestrate takes no arguments, got %q: the task is said to the orchestrator once it is open", fs.Arg(0))
	}

	ws, err := resolveWorkspace(*workspace)
	if err != nil {
		return err
	}
	opts, resolved, err := loadOptions(ws)
	if err != nil {
		return err
	}
	roots, err := config.DiscoverRoots(os.Getenv)
	if err != nil {
		return err
	}
	commands, err := config.DiscoverCommands(roots, ws, config.CommandMaxBytes)
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	path := *socket
	if path == "" {
		path = app.DefaultSocketPath(os.Getenv)
	}
	if err := app.SecureSocketDir(path); err != nil {
		return err
	}
	c := client.New(path)
	sess, err := openOrchestrator(ctx, c, path, ws)
	if err != nil {
		return err
	}

	return tui.Run(ctx, tui.Options{
		SessionID: sess.ID,
		Workspace: sess.Workspace,
		Model:     sess.Model,
		Sandbox:   sess.SandboxMode,
		Branch:    sess.Branch,
		Window:    sess.ContextWindow,
		// An orchestrator that came back from a restart has its whole history
		// in the log: it is read as history, not watched as it happens.
		Backlog:       sess.LastSeq,
		From:          sess.FirstSeq,
		Transport:     c,
		Orchestration: c,
		Geometry:      geometry(),
		Commands:      commands,
		AcceptsImages: acceptsImages(sess.Model),
		Lookup:        lookup(resolved),
		ProfileNames:  profileNames(opts.Profiles),
		Lang:          langOf(resolved),
		Notice:        versionNotice,
		Update:        updater,
	})
}

// projectDaemon is what `dcode orchestrate` asks of the daemon: that it
// answers, the project of a folder, and the session that conducts it.
type projectDaemon interface {
	Health(ctx context.Context) error
	OpenProject(ctx context.Context, workspace string) (protocol.Project, error)
	Orchestrator(ctx context.Context, projectID string) (protocol.Session, error)
}

var _ projectDaemon = (*client.Client)(nil)

// openOrchestrator resolves the project of ws through the daemon — the
// daemon's POST /v1/projects, which finds the project a worktree or a link
// already is rather than opening a second — and returns its orchestrator.
//
// Every way it cannot is said with what to do about it: no daemon names the
// command that starts one, and a daemon that conducts no projects names the
// two keys that decide it.
func openOrchestrator(ctx context.Context, d projectDaemon, socket, ws string) (protocol.Session, error) {
	probe, cancel := context.WithTimeout(ctx, 500*time.Millisecond)
	err := d.Health(probe)
	cancel()
	if err != nil {
		return protocol.Session{}, fmt.Errorf("no daemon answered on %s (%w): the orchestrator lives in the daemon "+
			"and outlives this terminal, so dcode orchestrate does not start one of its own — run `dcode serve` first", socket, err)
	}
	p, err := d.OpenProject(ctx, ws)
	if err != nil {
		if pe, ok := protocol.AsError(err); ok && pe.Code == protocol.CodeOrchestratorUnavailable {
			return protocol.Session{}, fmt.Errorf("the daemon on %s conducts no projects: %s\n"+
				"orchestrator.enabled and record.enabled must both be true where dcode serve reads its configuration; "+
				"`dcode config orchestrator.enabled` says where yours comes from, and dcode serve reads it when it starts",
				socket, pe.Message)
		}
		return protocol.Session{}, fmt.Errorf("the daemon could not open the project of %s: %w", ws, err)
	}
	sess, err := d.Orchestrator(ctx, p.ID)
	if err != nil {
		return protocol.Session{}, fmt.Errorf("the orchestrator of %s (project %s) could not be opened: %w", p.Root, p.ID, err)
	}
	if sess.Role != protocol.RoleOrchestrator {
		return protocol.Session{}, fmt.Errorf("the daemon answered session %s for the orchestrator of %s, "+
			"and it is not one (role %q): nothing was opened", sess.ID, p.Root, sess.Role)
	}
	return sess, nil
}
