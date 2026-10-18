package main

import (
	"context"

	"github.com/aguinelo/dcode/internal/protocol"
)

// projectDaemon is what `dcode orchestrate` asks of the daemon.
type projectDaemon interface {
	Health(ctx context.Context) error
	OpenProject(ctx context.Context, workspace string) (protocol.Project, error)
	Orchestrator(ctx context.Context, projectID string) (protocol.Session, error)
}

func runOrchestrate(args []string) error { return nil }

func openOrchestrator(ctx context.Context, d projectDaemon, socket, ws string) (protocol.Session, error) {
	return protocol.Session{}, nil
}
