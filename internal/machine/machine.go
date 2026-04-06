package machine

import (
	"image"
	"runtime"
)

// Machine is the universal interface for interacting with the OS.
// This allows GhostOperator to be 100% Cross-Platform.
type Machine interface {
	Capture() (image.Image, error)
	Move(x, y int) error
	Click(x, y int) error
	DoubleClick(x, y int) error
	Type(text string) error
	
	// User Intervention Check
	IsInterrupted() bool
	ResetIntervention()
}

// NewNativeMachine returns the appropriate implementation for the current OS.
func NewNativeMachine() Machine {
	switch runtime.GOOS {
	case "windows":
		return NewWindowsMachine()
	case "darwin":
		return NewDarwinMachine()
	case "linux":
		return NewLinuxMachine()
	default:
		// Fallback or panic if OS is unsupported
		return NewWindowsMachine() 
	}
}
