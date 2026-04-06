//go:build darwin
package input

import "fmt"

// ListenForHotkey starts a listener for Alt+G on macOS.
func ListenForHotkey(callback func(), onError func(err error)) {
	fmt.Println("⚠️ [macOS] Hotkey listener (Alt+G) is not yet implemented for this platform.")
}

// WindowInfo contains information about an active window.
type WindowInfo struct {
	Title string
	PID   uint32
}

// GetActiveWindowInfo returns window info for macOS.
func GetActiveWindowInfo() (WindowInfo, error) {
	return WindowInfo{Title: "macOS Desktop", PID: 0}, nil
}

// SetDPIAware is a no-op on macOS.
func SetDPIAware() {}
