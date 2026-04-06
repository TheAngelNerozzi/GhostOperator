//go:build linux
package machine

import (
	"fmt"
	"image"
	"github.com/kbinani/screenshot"
)

type LinuxMachine struct {
	// Linux specific fields
}

func NewLinuxMachine() *LinuxMachine {
	return &LinuxMachine{}
}

// NewNativeMachine returns the Linux implementation.
func NewNativeMachine() Machine {
	return NewLinuxMachine()
}

func (l *LinuxMachine) Capture() (image.Image, error) {
	bounds := screenshot.GetDisplayBounds(0)
	return screenshot.CaptureRect(bounds)
}

func (l *LinuxMachine) Move(x, y int) error {
	fmt.Printf("[Linux] Moving mouse to %d, %d\n", x, y)
	return nil
}

func (l *LinuxMachine) Click(x, y int) error {
	fmt.Printf("[Linux] Clicking at %d, %d\n", x, y)
	return nil
}

func (l *LinuxMachine) DoubleClick(x, y int) error {
	fmt.Printf("[Linux] Double-clicking at %d, %d\n", x, y)
	return nil
}

func (l *LinuxMachine) Type(text string) error {
	fmt.Printf("[Linux] Typing: %s\n", text)
	return nil
}

func (l *LinuxMachine) IsInterrupted() bool {
	return false
}

func (l *LinuxMachine) ResetIntervention() {}
