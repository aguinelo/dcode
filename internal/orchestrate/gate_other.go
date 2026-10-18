//go:build !unix

package orchestrate

import "os/exec"

// Without process groups only the gate's own process can be killed.
func setGateGroup(*exec.Cmd) {}

func killGateGroup(cmd *exec.Cmd) error { return cmd.Process.Kill() }
