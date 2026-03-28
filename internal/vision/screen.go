package vision

import (
	"bytes"
	"fmt"
	"image"
	"image/jpeg"

	"github.com/kbinani/screenshot"
)

// CaptureFullScreenshot captures all monitors and returns a byte buffer of the image in JPEG format.
func CaptureFullScreenshot() ([]byte, error) {
	n := screenshot.NumActiveDisplays()
	if n <= 0 {
		return nil, fmt.Errorf("no active displays found")
	}

	bounds := screenshot.GetDisplayBounds(0)
	img, err := screenshot.CaptureRect(bounds)
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 90}); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// CaptureROI captures a specific region of interest on the screen.
func CaptureROI(bounds image.Rectangle) ([]byte, image.Image, error) {
	img, err := screenshot.CaptureRect(bounds)
	if err != nil {
		return nil, nil, err
	}

	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 90}); err != nil {
		return nil, nil, err
	}
	return buf.Bytes(), img, nil
}

// GetDisplayCount returns the number of active displays.
func GetDisplayCount() int {
	return screenshot.NumActiveDisplays()
}
