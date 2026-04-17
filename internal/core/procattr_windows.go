//go:build windows

package core

import "os/exec"

// setProcessGroup is a no-op on Windows; process detachment is handled
// by using "cmd /C start" in the caller.
func setProcessGroup(cmd *exec.Cmd) {
	// No-op on Windows
}
