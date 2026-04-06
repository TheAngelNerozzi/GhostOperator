//go:build darwin
package machine

import (
	"fmt"
	"image"
	"github.com/kbinani/screenshot"
)

type DarwinMachine struct {
	// macOS specific fields (e.g. CGO handle)
}

func NewDarwinMachine() *DarwinMachine {
	return &DarwinMachine{}
}

// NewNativeMachine returns the macOS implementation.
func NewNativeMachine() Machine {
	return NewDarwinMachine()
}

func (d *DarwinMachine) Capture() (image.Image, error) {
	bounds := screenshot.GetDisplayBounds(0)
	return screenshot.CaptureRect(bounds)
}

func (d *DarwinMachine) Move(x, y int) error {
	fmt.Printf("[macOS] Moving mouse to %d, %d\n", x, y)
	return nil
}

func (d *DarwinMachine) Click(x, y int) error {
	fmt.Printf("[macOS] Clicking at %d, %d\n", x, y)
	return nil
}

func (d *DarwinMachine) DoubleClick(x, y int) error {
	fmt.Printf("[macOS] Double-clicking at %d, %d\n", x, y)
	return nil
}

func (d *DarwinMachine) Type(text string) error {
	fmt.Printf("[macOS] Typing: %s\n", text)
	return nil
}

func (d *DarwinMachine) IsInterrupted() bool {
	return false
}

func (d *DarwinMachine) ResetIntervention() {}
