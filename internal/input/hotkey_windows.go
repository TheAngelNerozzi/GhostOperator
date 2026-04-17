//go:build windows
package input

import (
	"context"
	"fmt"
	"runtime"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	modUser32          = windows.NewLazySystemDLL("user32.dll")
	procRegisterHotKey = modUser32.NewProc("RegisterHotKey")
	procUnregisterHotKey = modUser32.NewProc("UnregisterHotKey")
	procGetMessage     = modUser32.NewProc("GetMessageW")
	procPostThreadMessage = modUser32.NewProc("PostThreadMessageW")
)

const (
	modAlt = 0x0001
	wmQuit = 0x0012
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

// ListenForHotkey starts a listener for Alt+G (Windows Syscall version).
// It accepts a context for cancellation, a success callback, and an error callback.
func ListenForHotkey(ctx context.Context, callback func(), onError func(err error)) {
	go func() {
		// CRITICAL: Windows HotKeys are bound to the thread that registers them.
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()

		threadID := windows.GetCurrentThreadId()

		// Register Alt+G (ID 1)
		ret, _, _ := procRegisterHotKey.Call(0, 1, modAlt, 0x47)
		if ret == 0 {
			err := fmt.Errorf("failed to register hotkey Alt+G (Error: %v)", windows.GetLastError())
			fmt.Printf("❌ %v\n", err)
			if onError != nil {
				onError(err)
			}
			return
		}
		defer func() {
			procUnregisterHotKey.Call(0, 1)
		}()

		fmt.Println("✅ [Windows] Hotkey registered successfully: Alt+G")

		var m msg
		for {
			select {
			case <-ctx.Done():
				// Post WM_QUIT to unblock GetMessage
				procPostThreadMessage.Call(uintptr(threadID), wmQuit, 0, 0)
				return
			default:
			}

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
