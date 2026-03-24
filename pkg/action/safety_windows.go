package action

import (
	"fmt"
	"math"
	"unsafe"
)

var (
	procGetCursorPos = modUser32.NewProc("GetCursorPos")
	procGetAsyncKeyState = modUser32.NewProc("GetAsyncKeyState")
)

type point struct {
	x, y int32
}

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

func (e *ActionExecutor) getMousePos() point {
	var p point
	procGetCursorPos.Call(uintptr(unsafe.Pointer(&p)))
	return p
}

// Distance calculates pixels between two points
func distance(p1, p2 point) float64 {
	return math.Sqrt(math.Pow(float64(p1.x-p2.x), 2) + math.Pow(float64(p1.y-p2.y), 2))
}
