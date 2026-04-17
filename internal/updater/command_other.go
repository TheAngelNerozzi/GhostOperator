//go:build !windows

package updater

import "os/exec"

// createWindowsCommand is a no-op on non-Windows platforms.
// It should never be called on these platforms.
func createWindowsCommand(_ string) *exec.Cmd {
	return exec.Command("true")
}
