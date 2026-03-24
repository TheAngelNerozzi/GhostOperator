package action

import (
	"fmt"
	"unsafe"

	"github.com/kbinani/screenshot"
	"golang.org/x/sys/windows"
)

var (
	modUser32       = windows.NewLazySystemDLL("user32.dll")
	procSendInput   = modUser32.NewProc("SendInput")
)

const (
	inputMouse    = 0
	inputKeyboard = 1
)

const (
	mouseEventMove      = 0x0001
	mouseEventLeftDown  = 0x0002
	mouseEventLeftUp    = 0x0004
	mouseEventRightDown = 0x0008
	mouseEventRightUp   = 0x0010
	mouseEventAbsolute  = 0x8000
)

const (
	keyEventKeyDown = 0x0000
	keyEventKeyUp   = 0x0002
)

type mouseInput struct {
	dx          int32
	dy          int32
	mouseData   uint32
	dwFlags     uint32
	time        uint32
	dwExtraInfo uintptr
}

type keybdInput struct {
	wVk         uint16
	wScan       uint16
	dwFlags     uint32
	time        uint32
	dwExtraInfo uintptr
}

type hardwareInput struct {
	uMsg    uint32
	wParamL uint16
	wParamH uint16
}

type inputType struct {
	typ uint32
	// Union for Mouse, Keyboard, Hardware
	data [24]byte
}

func (e *ActionExecutor) handleClick(params map[string]interface{}) ActionResult {
	x, xOk := params["x"].(float64)
	y, yOk := params["y"].(float64)
	if !xOk || !yOk {
		return ActionResult{Status: "error", Message: "Missing x or y", Action: "CLICK"}
	}

	// 1. Normalize Coordinates (0-1000) to Pixels
	bounds := screenshot.GetDisplayBounds(0)
	pixelX := int32((x * float64(bounds.Dx())) / 1000.0)
	pixelY := int32((y * float64(bounds.Dy())) / 1000.0)

	// 2. Prepare Windows SendInput (Absolute coordinates require 0-65535 range for mouse_event)
	// Actually, easier to move mouse normally then click.
	e.sendMouseInput(pixelX, pixelY, mouseEventMove|mouseEventAbsolute)
	e.sendMouseInput(pixelX, pixelY, mouseEventLeftDown)
	e.sendMouseInput(pixelX, pixelY, mouseEventLeftUp)

	return ActionResult{Status: "success", Action: "CLICK", Metadata: map[string]int32{"x": pixelX, "y": pixelY}}
}

func (e *ActionExecutor) handleType(params map[string]interface{}) ActionResult {
	text, _ := params["text"].(string)
	// Implementation for typing strings would involve mapping chars to virtual keys
	// For now, let's log the action.
	fmt.Printf("Simulating Keyboard: %s\n", text)
	return ActionResult{Status: "success", Action: "TYPE"}
}

func (e *ActionExecutor) sendMouseInput(x, y int32, flags uint32) {
	bounds := screenshot.GetDisplayBounds(0)
	
	// Windows absolute coordinates for mouse_event are 0 to 65535
	ax := (x * 65535) / int32(bounds.Dx())
	ay := (y * 65535) / int32(bounds.Dy())

	var i inputType
	i.typ = inputMouse
	mi := (*mouseInput)(unsafe.Pointer(&i.data[0]))
	mi.dx = ax
	mi.dy = ay
	mi.dwFlags = flags

	procSendInput.Call(1, uintptr(unsafe.Pointer(&i)), uintptr(unsafe.Sizeof(i)))
}
