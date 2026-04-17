//go:build windows
package automation

import (
        "time"
        "unsafe"

        "golang.org/x/sys/windows"
)

var (
        modUser32        = windows.NewLazySystemDLL("user32.dll")
        procSendInput    = modUser32.NewProc("SendInput")
        procGetCursorPos = modUser32.NewProc("GetCursorPos")
        procSetCursorPos = modUser32.NewProc("SetCursorPos")
)

const (
        inputMouse    = 0
        inputKeyboard = 1
)

const (
        mouseEventLeftDown = 0x0002
        mouseEventLeftUp   = 0x0004

        keyEventKeyDown = 0x0000
        keyEventKeyUp   = 0x0002

        vkBack    = 0x08
        vkReturn  = 0x0D
        vkSpace   = 0x20
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

// keyboardInput matches the Windows KEYBDINPUT structure
type keyboardInput struct {
        wVk         uint16
        wScan       uint16
        dwFlags     uint32
        time        uint32
        dwExtraInfo uintptr
}

// inputType matches the Windows INPUT structure for 64-bit
type inputType struct {
        typ uint32
        _   uint32 // Padding for 64-bit alignment of the union (4 bytes)
        mi  mouseInput
        ki  keyboardInput
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

        // x,y are absolute pixel coordinates (not normalized)
        pixelX := int32(x)
        pixelY := int32(y)

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

        // x,y are absolute pixel coordinates (not normalized)
        pixelX := int32(x)
        pixelY := int32(y)

        // Smooth glide from current position
        e.SmoothMove(pixelX, pixelY)

        // First click
        e.sendMouseClick(mouseEventLeftDown)
        time.Sleep(20 * time.Millisecond)
        e.sendMouseClick(mouseEventLeftUp)

        // Standard double-click interval
        time.Sleep(50 * time.Millisecond)

        // Second click
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

                // Verification: Did the user move the mouse?
                var checkPt point
                procGetCursorPos.Call(uintptr(unsafe.Pointer(&checkPt)))
                if i > 1 {
                        lastX, lastY := e.GetLastTarget()
                        dist := (checkPt.x-lastX)*(checkPt.x-lastX) +
                                   (checkPt.y-lastY)*(checkPt.y-lastY)
                        if dist > 100 { // Approx 10 pixels movement
                                e.SetInterrupted(true)
                                if e.OnInterruption != nil {
                                        e.OnInterruption()
                                }
                                return
                        }

                        // Check safety during movement
                        if err := e.CheckSafety(); err != nil {
                                e.SetInterrupted(true)
                                if e.OnInterruption != nil {
                                        e.OnInterruption()
                                }
                                return
                        }
                }

                procSetCursorPos.Call(uintptr(stepX), uintptr(stepY))
                e.SetLastTarget(stepX, stepY)
                time.Sleep(12 * time.Millisecond)
        }
}

func (e *ActionExecutor) handleType(params map[string]interface{}) ActionResult {
        text, ok := params["text"].(string)
        if !ok || text == "" {
                return ActionResult{Status: "error", Message: "Missing or empty text parameter", Action: "TYPE"}
        }

        for _, ch := range text {
                var vk uint16
                var shift bool

                switch {
                case ch == ' ':
                        vk = vkSpace
                case ch == '\n' || ch == '\r':
                        vk = vkReturn
                case ch == '\b':
                        vk = vkBack
                case ch >= 'A' && ch <= 'Z':
                        vk = uint16(ch)
                case ch >= 'a' && ch <= 'z':
                        vk = uint16(ch - 32) // Convert to uppercase VK code
                case ch >= '0' && ch <= '9':
                        vk = uint16(ch)
                default:
                        if ch >= 0x20 && ch <= 0x7E {
                                switch ch {
                                case '!':
                                        vk = 0x31; shift = true
                                case '@':
                                        vk = 0x32; shift = true
                                case '#':
                                        vk = 0x33; shift = true
                                case '$':
                                        vk = 0x34; shift = true
                                case '%':
                                        vk = 0x35; shift = true
                                case '^':
                                        vk = 0x36; shift = true
                                case '&':
                                        vk = 0x37; shift = true
                                case '*':
                                        vk = 0x38; shift = true
                                case '(':
                                        vk = 0x39; shift = true
                                case ')':
                                        vk = 0x30; shift = true
                                case '-':
                                        vk = 0xBD
                                case '=':
                                        vk = 0xBB
                                case '[':
                                        vk = 0xDB
                                case ']':
                                        vk = 0xDD
                                case ';':
                                        vk = 0xBA
                                case '\'':
                                        vk = 0xDE
                                case ',':
                                        vk = 0xBC
                                case '.':
                                        vk = 0xBE
                                case '/':
                                        vk = 0xBF
                                case '\\':
                                        vk = 0xDC
                                case '`':
                                        vk = 0xC0
                                case '_':
                                        vk = 0xBD; shift = true
                                case '+':
                                        vk = 0xBB; shift = true
                                case '{':
                                        vk = 0xDB; shift = true
                                case '}':
                                        vk = 0xDD; shift = true
                                case ':':
                                        vk = 0xBA; shift = true
                                case '"':
                                        vk = 0xDE; shift = true
                                case '<':
                                        vk = 0xBC; shift = true
                                case '>':
                                        vk = 0xBE; shift = true
                                case '?':
                                        vk = 0xBF; shift = true
                                case '|':
                                        vk = 0xDC; shift = true
                                case '~':
                                        vk = 0xC0; shift = true
                                default:
                                        vk = uint16(ch)
                                }
                        } else {
                                // Non-ASCII: use Unicode scan code with KEYEVENTF_UNICODE
                                e.sendKeyUnicode(uint16(ch))
                                time.Sleep(10 * time.Millisecond)
                                continue
                        }
                }

                if shift {
                        e.sendKeyPress(0x10, true) // VK_SHIFT down
                }
                e.sendKeyPress(vk, true)  // Key down
                e.sendKeyPress(vk, false) // Key up
                if shift {
                        e.sendKeyPress(0x10, false) // VK_SHIFT up
                }
                time.Sleep(10 * time.Millisecond)
        }

        return ActionResult{Status: "success", Action: "TYPE"}
}

func (e *ActionExecutor) sendMouseClick(flags uint32) {
        var i inputType
        i.typ = inputMouse
        i.mi.dx = 0
        i.mi.dy = 0
        i.mi.dwFlags = flags

        procSendInput.Call(1, uintptr(unsafe.Pointer(&i)), uintptr(unsafe.Sizeof(i)))
}

// sendKeyPress sends a key down or key up event using SendInput.
func (e *ActionExecutor) sendKeyPress(vk uint16, keyDown bool) {
        var i inputType
        i.typ = inputKeyboard
        i.ki.wVk = vk
        if keyDown {
                i.ki.dwFlags = keyEventKeyDown
        } else {
                i.ki.dwFlags = keyEventKeyUp
        }
        procSendInput.Call(1, uintptr(unsafe.Pointer(&i)), uintptr(unsafe.Sizeof(i)))
}

// sendKeyUnicode sends a Unicode character using KEYEVENTF_UNICODE.
func (e *ActionExecutor) sendKeyUnicode(ch uint16) {
        // Key down
        var down inputType
        down.typ = inputKeyboard
        down.ki.wScan = ch
        down.ki.dwFlags = 0x0004 // KEYEVENTF_UNICODE
        procSendInput.Call(1, uintptr(unsafe.Pointer(&down)), uintptr(unsafe.Sizeof(down)))

        time.Sleep(5 * time.Millisecond)

        // Key up
        var up inputType
        up.typ = inputKeyboard
        up.ki.wScan = ch
        up.ki.dwFlags = 0x0004 | keyEventKeyUp // KEYEVENTF_UNICODE | KEYEVENTF_KEYUP
        procSendInput.Call(1, uintptr(unsafe.Pointer(&up)), uintptr(unsafe.Sizeof(up)))
}
