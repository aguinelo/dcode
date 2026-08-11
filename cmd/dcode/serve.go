package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
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
	base, _, err := loadOptions(ws)
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
