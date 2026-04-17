//go:build linux
package input

import (
        "context"
        "fmt"
)

// ListenForHotkey starts a listener for Alt+G on Linux.
func ListenForHotkey(ctx context.Context, callback func(), onError func(err error)) {
        fmt.Println("⚠️ [Linux] Hotkey listener (Alt+G) is not yet implemented for this platform.")
}

// WindowInfo contains information about an active window.
type WindowInfo struct {
        Title string
        PID   uint32
}

// GetActiveWindowInfo returns window info for Linux.
func GetActiveWindowInfo() (WindowInfo, error) {
        return WindowInfo{Title: "Linux Desktop", PID: 0}, nil
}

// SetDPIAware is a no-op on Linux.
func SetDPIAware() {}
