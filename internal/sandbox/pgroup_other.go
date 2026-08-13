//go:build !unix

package sandbox

import "os/exec"

// No sandbox backend exists off unix, so nothing here can start a confined
// command in the first place. These keep the package building for a
// cross-compilation check rather than promising a lifetime they cannot hold.
func setGroup(*exec.Cmd) {}

func killGroup(cmd *exec.Cmd) { _ = cmd.Process.Kill() }
