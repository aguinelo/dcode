//go:build unix

package orchestrate

import (
	"os/exec"
	"syscall"
)

// setGateGroup puts the gate in a process group of its own: `make check`
// starts compilers and test binaries, and a kill that reaches only the shell
// leaves them running past the timeout.
func setGateGroup(cmd *exec.Cmd) {
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.SysProcAttr.Setpgid = true
}

// killGateGroup signals the whole group; the negative pid makes it the group.
// When that fails, the process that can be reached is killed rather than none.
func killGateGroup(cmd *exec.Cmd) error {
	if err := syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL); err != nil {
		return cmd.Process.Kill()
	}
	return nil
}
