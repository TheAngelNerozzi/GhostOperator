//go:build windows
package automation

import (
	"fmt"
	"math"
	"unsafe"
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

	// 2. Check for unexpected mouse movement (only if we have a previous target)
	hasLast, lastX, lastY := e.GetLastTargetWithFlag()
	if hasLast {
		current := e.getMousePos()
		target := point{x: lastX, y: lastY}

		d := distance(current, target)
		if d > 120.0 { // 120 pixel threshold for manual takeover
			return fmt.Errorf("SAFETY KILL: User moved mouse (dist: %.1f px)", d)
		}
	}

	return nil
}

func (e *ActionExecutor) getMousePos() point {
	var p point
	ret, _, _ := procGetCursorPos.Call(uintptr(unsafe.Pointer(&p)))
	if ret == 0 {
		// Fallback: return last known target if GetCursorPos fails
		_, x, y := e.GetLastTargetWithFlag()
		return point{x: x, y: y}
	}
	return p
}

func distance(p1, p2 point) float64 {
	dx := float64(p1.x - p2.x)
	dy := float64(p1.y - p2.y)
	return math.Sqrt(dx*dx + dy*dy)
}
