package machine

import "image"

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
