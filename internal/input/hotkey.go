package input

import (
	"fmt"
	"runtime"
	"syscall"
	"unsafe"
)

var (
	modUser32          = syscall.NewLazyDLL("user32.dll")
	procRegisterHotKey = modUser32.NewProc("RegisterHotKey")
	procGetMessage     = modUser32.NewProc("GetMessageW")
)

const (
	modAlt = 0x0001
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
// It accepts a success callback and an error callback.
func ListenForHotkey(callback func(), onError func(err error)) {
	go func() {
		// CRITICAL: Windows HotKeys are bound to the thread that registers them.
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()

		// Register Alt+G (ID 1)
		// Virtual Key for 'G' is 0x47
		ret, _, _ := procRegisterHotKey.Call(0, 1, modAlt, 0x47)
		if ret == 0 {
			err := fmt.Errorf("failed to register hotkey Alt+G (Error: %v)", syscall.GetLastError())
			fmt.Printf("❌ %v\n", err)
			if onError != nil {
				onError(err)
			}
			return
		}

		fmt.Println("✅ [Windows] Hotkey registered successfully: Alt+G")

		var m msg
		for {
			ret, _, _ := procGetMessage.Call(uintptr(unsafe.Pointer(&m)), 0, 0, 0)
			if ret <= 0 {
				break
			}
			if m.Message == 0x0312 { // WM_HOTKEY
				fmt.Println("🎯 Hotkey message received (WM_HOTKEY)")
				callback()
			}
		}
	}()
}
