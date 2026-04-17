//go:build !windows

package core

import (
	"os/exec"
	"syscall"
)

// setProcessGroup detaches the child process into its own process group
// on Unix systems, preventing it from becoming a zombie.
func setProcessGroup(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}
