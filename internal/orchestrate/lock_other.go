//go:build !unix

package orchestrate

import (
	"errors"
	"os"
)

// lockDir refuses where there is no flock: without it nothing stops a second
// daemon from conducting the same projects, and the daemon socket is unix-only
// anyway.
func lockDir(string) (*os.File, error) {
	return nil, errors.New("orchestrate: the orchestrator needs a unix host, where its directory can be locked to one daemon")
}
