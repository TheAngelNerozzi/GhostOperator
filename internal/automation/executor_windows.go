package automation

import (
	"fmt"
	"time"
	"unsafe"

	"github.com/kbinani/screenshot"
	"golang.org/x/sys/windows"
)

var (
	modUser32        = windows.NewLazySystemDLL("user32.dll")
	procSendInput    = modUser32.NewProc("SendInput")
	procGetCursorPos = modUser32.NewProc("GetCursorPos")
	procSetCursorPos = modUser32.NewProc("SetCursorPos")
)

const (
	inputMouse = 0
)

const (
	mouseEventLeftDown = 0x0002
	mouseEventLeftUp   = 0x0004
)

// mouseInput matches the Windows MOUSEINPUT structure
type mouseInput struct {
	dx          int32
	dy          int32
	mouseData   uint32
	dwFlags     uint32
	time        uint32
	dwExtraInfo uintptr
}

// inputType matches the Windows INPUT structure for 64-bit
type inputType struct {
	typ uint32
	_   uint32     // Padding for 64-bit alignment of the union (4 bytes)
	mi  mouseInput // MOUSEINPUT structure (32 bytes)
}

type point struct {
	x, y int32
}

func (e *ActionExecutor) handleClick(params map[string]interface{}) ActionResult {
	x, xOk := params["x"].(float64)
	y, yOk := params["y"].(float64)
	if !xOk || !yOk {
		return ActionResult{Status: "error", Message: "Missing x or y", Action: "CLICK"}
	}

	bounds := screenshot.GetDisplayBounds(0)
	pixelX := int32((x * float64(bounds.Dx())) / 1000.0)
	pixelY := int32((y * float64(bounds.Dy())) / 1000.0)

	// Smooth glide from current position
	e.SmoothMove(pixelX, pixelY)

	// Perform click with slight delay to ensure OS registers it
	e.sendMouseClick(mouseEventLeftDown)
	time.Sleep(20 * time.Millisecond)
	e.sendMouseClick(mouseEventLeftUp)

	return ActionResult{Status: "success", Action: "CLICK", Metadata: map[string]int32{"x": pixelX, "y": pixelY}}
}

func (e *ActionExecutor) handleDoubleClick(params map[string]interface{}) ActionResult {
	x, xOk := params["x"].(float64)
	y, yOk := params["y"].(float64)
	if !xOk || !yOk {
		return ActionResult{Status: "error", Message: "Missing x or y", Action: "DOUBLE_CLICK"}
	}

	bounds := screenshot.GetDisplayBounds(0)
	pixelX := int32((x * float64(bounds.Dx())) / 1000.0)
	pixelY := int32((y * float64(bounds.Dy())) / 1000.0)

	// Smooth glide from current position
	e.SmoothMove(pixelX, pixelY)

	// Primer clic
	e.sendMouseClick(mouseEventLeftDown)
	time.Sleep(20 * time.Millisecond)
	e.sendMouseClick(mouseEventLeftUp)

	// Intervalo estándar de doble clic para OS
	time.Sleep(50 * time.Millisecond)

	// Segundo clic
	e.sendMouseClick(mouseEventLeftDown)
	time.Sleep(20 * time.Millisecond)
	e.sendMouseClick(mouseEventLeftUp)

	return ActionResult{Status: "success", Action: "DOUBLE_CLICK", Metadata: map[string]int32{"x": pixelX, "y": pixelY}}
}

// SmoothMove glides the mouse from current position to target.
func (e *ActionExecutor) SmoothMove(targetX, targetY int32) {
	var pt point
	procGetCursorPos.Call(uintptr(unsafe.Pointer(&pt)))

	// Make the movement visibly human-like (approx 600ms)
	steps := 50
	for i := 1; i <= steps; i++ {
		t := float64(i) / float64(steps)
		// Ease out cubic for a very natural, decelerating glide
		t = 1 - (1-t)*(1-t)*(1-t)

		stepX := pt.x + int32(float64(targetX-pt.x)*t)
		stepY := pt.y + int32(float64(targetY-pt.y)*t)

		procSetCursorPos.Call(uintptr(stepX), uintptr(stepY))
		time.Sleep(12 * time.Millisecond)
	}

	// Record intended position for safety checks
	e.LastTargetX = targetX
	e.LastTargetY = targetY
}

func (e *ActionExecutor) handleType(params map[string]interface{}) ActionResult {
	text, _ := params["text"].(string)
	fmt.Printf("Simulating Keyboard: %s\n", text)
	return ActionResult{Status: "success", Action: "TYPE"}
}

func (e *ActionExecutor) sendMouseClick(flags uint32) {
	var i inputType
	i.typ = inputMouse // 0
	i.mi.dx = 0
	i.mi.dy = 0
	i.mi.dwFlags = flags

	// Call SendInput with 1 input and the correct size (40 bytes on 64-bit)
	procSendInput.Call(1, uintptr(unsafe.Pointer(&i)), uintptr(unsafe.Sizeof(i)))
}
