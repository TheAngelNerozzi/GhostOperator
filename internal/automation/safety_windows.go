package automation

import (
	"fmt"
)

var (
	procGetAsyncKeyState = modUser32.NewProc("GetAsyncKeyState")
)

const (
	vkEscape = 0x1B
)

// CheckSafety monitors for user intervention (mouse move or Esc key).
func (e *ActionExecutor) CheckSafety() error {
	// 1. Check for ESC key
	ret, _, _ := procGetAsyncKeyState.Call(uintptr(vkEscape))
	if ret&0x8000 != 0 {
		return fmt.Errorf("SAFETY KILL: ESC key pressed")
	}

	// 2. We could store the "intended" mouse position and compare with actual.
	// For now, this is a placeholder for a more complex movement-based kill-switch.
	return nil
}
