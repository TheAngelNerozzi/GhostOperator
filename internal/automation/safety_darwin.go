//go:build darwin
package automation

// CheckSafety monitors for user intervention (mouse move or Esc key) on macOS.
func (e *ActionExecutor) CheckSafety() error {
	// macOS-specific safety checks (e.g., using CGEvent)
	// For now, we return nil as a placeholder.
	return nil
}
