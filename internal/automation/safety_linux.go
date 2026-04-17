//go:build linux
package automation

import "fmt"

// CheckSafety monitors for user intervention (mouse move or Esc key) on Linux.
func (e *ActionExecutor) CheckSafety() error {
        // Check if the executor has been interrupted
        if e.GetInterrupted() {
                return fmt.Errorf("SAFETY KILL: execution interrupted by user")
        }
        // Linux-specific safety checks (e.g., using XQueryPointer) can be added here
        return nil
}
