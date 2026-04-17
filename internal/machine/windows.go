//go:build windows
package machine

import (
	"fmt"
	"image"
	"github.com/TheAngelNerozzi/ghostoperator/internal/automation"
	"github.com/kbinani/screenshot"
)

// WindowsMachine implements the Machine interface for the Windows OS.
type WindowsMachine struct {
	executor *automation.ActionExecutor
}

func NewWindowsMachine() *WindowsMachine {
	return &WindowsMachine{
		executor: &automation.ActionExecutor{},
	}
}

// NewNativeMachine returns the Windows implementation.
func NewNativeMachine() Machine {
	return NewWindowsMachine()
}

func (w *WindowsMachine) Capture() (image.Image, error) {
	n := screenshot.NumActiveDisplays()
	if n == 0 {
		return nil, fmt.Errorf("no active displays found")
	}
	bounds := screenshot.GetDisplayBounds(0)
	return screenshot.CaptureRect(bounds)
}

func (w *WindowsMachine) Move(x, y int) error {
	w.executor.SmoothMove(int32(x), int32(y))
	return nil
}

func (w *WindowsMachine) Click(x, y int) error {
	result := w.executor.Execute(automation.Command{Type: "CLICK", Params: map[string]interface{}{"x": float64(x), "y": float64(y)}})
	if result.Status != "success" {
		return fmt.Errorf("click failed: %s", result.Message)
	}
	return nil
}

func (w *WindowsMachine) DoubleClick(x, y int) error {
	result := w.executor.Execute(automation.Command{Type: "DOUBLE_CLICK", Params: map[string]interface{}{"x": float64(x), "y": float64(y)}})
	if result.Status != "success" {
		return fmt.Errorf("double-click failed: %s", result.Message)
	}
	return nil
}

func (w *WindowsMachine) Type(text string) error {
	result := w.executor.Execute(automation.Command{Type: "TYPE", Params: map[string]interface{}{"text": text}})
	if result.Status != "success" {
		return fmt.Errorf("type failed: %s", result.Message)
	}
	return nil
}

func (w *WindowsMachine) IsInterrupted() bool {
	return w.executor.GetInterrupted()
}

func (w *WindowsMachine) ResetIntervention() {
	w.executor.SetInterrupted(false)
}
