package input

import (
	"fmt"
	"syscall"
	"unsafe"
)

var (
	modUser32        = syscall.NewLazyDLL("user32.dll")
	procRegisterHotKey = modUser32.NewProc("RegisterHotKey")
	procGetMessage     = modUser32.NewProc("GetMessageW")
)

const (
	modAlt = 0x0001
	// modControl = 0x0002
	// modShift = 0x0004
	// modWin = 0x0008
)

type msg struct {
	HWND    uintptr
	Message uint32
	WParam  uintptr
	LParam  uintptr
	Time    uint32
	Pt      struct {
		X, Y int32
	}
}

// ListenForHotkey starts a listener for Alt+Space (Windows Syscall version).
func ListenForHotkey(callback func()) {
	go func() {
		// Register Alt+Space (ID 1)
		// Virtual Key for Space is 0x20
		ret, _, _ := procRegisterHotKey.Call(0, 1, modAlt, 0x20)
		if ret == 0 {
			fmt.Println("Failed to register hotkey Alt+Space")
			return
		}

		fmt.Println("Hotkey registered: Alt+Space (via Syscall)")

		var m msg
		for {
			ret, _, _ := procGetMessage.Call(uintptr(unsafe.Pointer(&m)), 0, 0, 0)
			if ret <= 0 {
				break
			}
			if m.Message == 0x0312 { // WM_HOTKEY
				callback()
			}
		}
	}()
}
