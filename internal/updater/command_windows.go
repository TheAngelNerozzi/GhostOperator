//go:build windows

package updater

import "os/exec"

// createWindowsCommand creates a command that runs detached on Windows.
func createWindowsCommand(batchPath string) *exec.Cmd {
	return exec.Command("cmd", "/C", batchPath)
}
