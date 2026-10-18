//go:build unix

package orchestrate

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"
)

const lockFile = "daemon.lock"

// lockDir takes orchestrator.dir for this daemon alone. Two daemons conducting
// the same projects would reopen one orchestrator twice and write one decision
// log with two sequences; the second is refused, saying why. The lock goes with
// the process, so a daemon that crashed holds nothing.
func lockDir(dir string) (*os.File, error) {
	path := filepath.Join(dir, lockFile)
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0o600)
	if err != nil {
		return nil, fmt.Errorf("orchestrate: open %s: %w", path, err)
	}
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		_ = f.Close()
		return nil, fmt.Errorf("orchestrate: %s is held by another daemon, which conducts these projects: %w", dir, err)
	}
	return f, nil
}
