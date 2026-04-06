//go:build linux
package automation

// CheckSafety monitors for user intervention (mouse move or Esc key) on Linux.
func (e *ActionExecutor) CheckSafety() error {
	// Linux-specific safety checks (e.g., using XQueryPointer)
	// For now, we return nil as a placeholder.
	return nil
}
