package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"github.com/aguinelo/dcode/internal/config"
	"github.com/aguinelo/dcode/internal/session"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/aguinelo/dcode/internal/app"
	"github.com/aguinelo/dcode/internal/version"
)

// runServe starts the daemon in the foreground.
//
// Foreground on purpose: a process supervisor, a shell job or a container all
// know how to background something, and none of them knows how to adopt a
// process that daemonised itself.
func runServe(args []string) error {
	fs := flag.NewFlagSet("dcode serve", flag.ContinueOnError)
	var (
		socket      = fs.String("socket", "", "socket path (default: $DCODE_SOCKET, else a per-user path)")
		workspace   = fs.String("workspace", "", "default workspace for sessions that do not name one")
		maxSessions = fs.Int("max-sessions", 64, "refuse to create more than this many sessions")
		retention   = fs.Int("event-retention", 10000, "events kept per session for replay")
		timeout     = fs.Duration("approval-timeout", 2*time.Minute, "deny an approval nobody answered within this")
	)
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "dcode serve — run the daemon\n\nFlags:\n")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		return err
	}

	ws, err := resolveWorkspace(*workspace)
	if err != nil {
		return err
	}
	base, resolved, err := loadOptions(ws)
	if err != nil {
		return err
	}

	path := *socket
	if path == "" {
		path = app.DefaultSocketPath(os.Getenv)
	}

	d := app.NewDaemon(app.DaemonOptions{
		Log:             func(msg string) { fmt.Fprintln(os.Stderr, msg) },
		SocketPath:      path,
		MaxSessions:     *maxSessions,
		EventRetention:  *retention,
		RecordDir:       recordDir(resolved),
		RecordBudget:    recordBudget(resolved),
		ApprovalTimeout: *timeout,
		Base:            base,
	})

	// Bind before announcing. Printing an address the daemon then failed to
	// take would send every client to a socket that never existed.
	if err := d.Listen(); err != nil {
		return err
	}
	fmt.Printf("dcode %s · listening on %s\n", version.Short(), d.Addr())

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := d.Serve(ctx); err != nil && !errors.Is(err, context.Canceled) {
		return err
	}
	return nil
}

// spillDir is where trimmed events are kept.
//
// Defaults to the state root rather than to nothing. Retention without a spill
// is a hard horizon, and the failure it produces — a client reattaching to a
// long session and being told the events expired — is one nobody would think to
// switch a setting on to avoid, because it looks like the session broke.
func recordDir(r config.Resolved) string {
	if !r.Bool("record.enabled", true) {
		return ""
	}
	if v := r.String("record.dir", ""); v != "" {
		return v
	}
	roots, err := config.DiscoverRoots(os.Getenv)
	if err != nil {
		return ""
	}
	return filepath.Join(roots.State, "sessions")
}

// recordBudget is how much session history survives.
//
// Thirty days and half a gigabyte, and both have to hold: age alone lets a busy
// fortnight fill a disk inside the window, and size alone deletes this
// morning's work on a busy day.
//
// The floor is not configurable, and that is the point. Someone who used dcode
// twice last year should still find those two sessions, and a policy that can
// be set to zero is a policy that empties the directory on a typo.
func recordBudget(r config.Resolved) session.PruneBudget {
	return session.PruneBudget{
		MaxAge:      time.Duration(r.Int("record.keep_days", 30)) * 24 * time.Hour,
		MaxBytes:    int64(r.Int("record.max_bytes", 512<<20)),
		KeepAtLeast: 10,
	}
}
