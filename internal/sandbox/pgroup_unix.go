//go:build unix

package sandbox

import (
	"os/exec"
	"syscall"
)

// setGroup puts the command in a process group of its own, so everything it
// spawns can be ended together. Without it the only reachable process is the
// wrapper, and the wrapper is never the one holding the port.
func setGroup(cmd *exec.Cmd) {
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.SysProcAttr.Setpgid = true
}

// killGroup signals the whole group. The negative pid is what makes it the
// group rather than the leader.
//
// Falling back to the single process matters: if Setpgid did not take, killing
// nothing at all would be worse than killing the part that can be reached.
func killGroup(cmd *exec.Cmd) {
	pid := cmd.Process.Pid
	if err := syscall.Kill(-pid, syscall.SIGKILL); err != nil {
		_ = cmd.Process.Kill()
	}
}
