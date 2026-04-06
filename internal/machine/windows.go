//go:build windows
package machine

import (
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
	bounds := screenshot.GetDisplayBounds(0)
	return screenshot.CaptureRect(bounds)
}

func (w *WindowsMachine) Move(x, y int) error {
	w.executor.SmoothMove(int32(x), int32(y))
	return nil
}

func (w *WindowsMachine) Click(x, y int) error {
	w.Move(x, y)
	w.executor.Execute(automation.Command{Type: "CLICK", Params: map[string]interface{}{"x": float64(x), "y": float64(y)}})
	return nil
}

func (w *WindowsMachine) DoubleClick(x, y int) error {
	w.Move(x, y)
	w.executor.Execute(automation.Command{Type: "DOUBLE_CLICK", Params: map[string]interface{}{"x": float64(x), "y": float64(y)}})
	return nil
}

func (w *WindowsMachine) Type(text string) error {
	w.executor.Execute(automation.Command{Type: "TYPE", Params: map[string]interface{}{"text": text}})
	return nil
}

func (w *WindowsMachine) IsInterrupted() bool {
	return w.executor.Interrupted
}

func (w *WindowsMachine) ResetIntervention() {
	w.executor.Interrupted = false
}
