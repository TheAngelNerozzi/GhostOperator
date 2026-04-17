//go:build windows
package input

import (
	"fmt"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	user32                       = windows.NewLazySystemDLL("user32.dll")
	procGetForegroundWindow      = user32.NewProc("GetForegroundWindow")
	procGetWindowTextW           = user32.NewProc("GetWindowTextW")
	procGetWindowThreadProcessId = user32.NewProc("GetWindowThreadProcessId")
	procSetProcessDPIAware       = user32.NewProc("SetProcessDPIAware")
)

// SetDPIAware enables DPI awareness for the current process.
func SetDPIAware() {
	ret, _, err := procSetProcessDPIAware.Call()
	if ret == 0 {
		fmt.Printf("Warning: SetProcessDPIAware failed: %v\n", err)
	}
}

// WindowInfo contains information about an active window.
type WindowInfo struct {
	Title string
	PID   uint32
}

// GetActiveWindowInfo returns the title and PID of the currently active window on Windows.
func GetActiveWindowInfo() (WindowInfo, error) {
	hwnd, _, _ := procGetForegroundWindow.Call()
	if hwnd == 0 {
		return WindowInfo{}, fmt.Errorf("no foreground window found")
	}

	// Get Title (buffer increased to 512 for long titles)
	b := make([]uint16, 512)
	ret, _, err := procGetWindowTextW.Call(hwnd, uintptr(unsafe.Pointer(&b[0])), uintptr(len(b)))
	if ret == 0 {
		return WindowInfo{}, fmt.Errorf("GetWindowTextW failed: %v", err)
	}
	title := syscall.UTF16ToString(b)

	// Get PID
	var pid uint32
	_, _, _ = procGetWindowThreadProcessId.Call(hwnd, uintptr(unsafe.Pointer(&pid)))

	return WindowInfo{
		Title: title,
		PID:   pid,
	}, nil
}
