//go:build darwin
package automation

import "fmt"

// CheckSafety monitors for user intervention (mouse move or Esc key) on macOS.
func (e *ActionExecutor) CheckSafety() error {
        // Check if the executor has been interrupted
        if e.GetInterrupted() {
                return fmt.Errorf("SAFETY KILL: execution interrupted by user")
        }
        // macOS-specific safety checks (e.g., using CGEvent) can be added here
        return nil
}
