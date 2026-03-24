package screen

import (
	"bytes"
	"fmt"
	"image/png"

	"github.com/kbinani/screenshot"
)

// CaptureFullScreenshot captures all monitors and returns a byte buffer of the image in PNG format.
func CaptureFullScreenshot() ([]byte, error) {
	n := screenshot.NumActiveDisplays()
	if n <= 0 {
		return nil, fmt.Errorf("no active displays found")
	}

	// Capture the first monitor (primary) for simplicity, or we could merge them.
	// For "GhostOperator", usually we want the primary screen or active screen.
	bounds := screenshot.GetDisplayBounds(0)
	img, err := screenshot.CaptureRect(bounds)
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// GetDisplayCount returns the number of active displays.
func GetDisplayCount() int {
	return screenshot.NumActiveDisplays()
}
