package vision

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"os"

	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"
)

// GridConfig represents the dimensions of the visual grid.
type GridConfig struct {
	Rows int
	Cols int
}

// DrawGrid overlays a grid on the image and returns a JPEG buffer.
func DrawGrid(img image.Image, config GridConfig) ([]byte, error) {
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	// 1. Create a canvas
	canvas := image.NewRGBA(bounds)
	draw.Draw(canvas, bounds, img, bounds.Min, draw.Src)

	cellWidth := float64(width) / float64(config.Cols)
	cellHeight := float64(height) / float64(config.Rows)

	gridColor := color.RGBA{255, 0, 0, 128} // Semi-transparent red
	labelColor := color.RGBA{255, 255, 255, 255}

	// 2. Draw Lines and Labels
	for i := 0; i <= config.Cols; i++ {
		x := int(float64(i) * cellWidth)
		drawLineV(canvas, x, 0, height, gridColor)
	}

	for j := 0; j <= config.Rows; j++ {
		y := int(float64(j) * cellHeight)
		drawLineH(canvas, 0, width, y, gridColor)
	}

	// 3. Add Alphanumeric Labels (A1, B2...)
	for r := 0; r < config.Rows; r++ {
		for c := 0; c < config.Cols; c++ {
			label := fmt.Sprintf("%c%d", 'A'+c, r+1)
			// Move to Top-Left corner of cell for maximum visibility of content
			x := int(float64(c)*cellWidth + 2)
			y := int(float64(r)*cellHeight + 12)
			addLabel(canvas, x, y, label, labelColor)
		}
	}

	// 4. Encode as compressed JPEG for radically faster LLM speed
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, canvas, &jpeg.Options{Quality: 60}); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// SaveDebugFrame saves the provided JPG buffer to the project root for UAT.
func SaveDebugFrame(data []byte) error {
	return os.WriteFile("debug_vision.jpg", data, 0644)
}

// MapLabelToPixel converts "B5" to center (X, Y)
func MapLabelToPixel(label string, bounds image.Rectangle, config GridConfig) (int, int, error) {
	if len(label) < 2 {
		return 0, 0, fmt.Errorf("invalid label format")
	}

	colChar := label[0]
	var row int
	fmt.Sscanf(label[1:], "%d", &row)

	col := int(colChar - 'A')
	rowIdx := row - 1

	// Validate bounds
	if col < 0 || col >= config.Cols || rowIdx < 0 || rowIdx >= config.Rows {
		return 0, 0, fmt.Errorf("label %s out of grid bounds (%dx%d)", label, config.Cols, config.Rows)
	}

	cellWidth := float64(bounds.Dx()) / float64(config.Cols)
	cellHeight := float64(bounds.Dy()) / float64(config.Rows)

	x := int(float64(col)*cellWidth + cellWidth/2)
	y := int(float64(rowIdx)*cellHeight + cellHeight/2)

	return x, y, nil
}

func drawLineV(img *image.RGBA, x, y1, y2 int, c color.Color) {
	for y := y1; y < y2; y++ {
		img.Set(x, y, c)
	}
}

func drawLineH(img *image.RGBA, x1, x2, y int, c color.Color) {
	for x := x1; x < x2; x++ {
		img.Set(x, y, c)
	}
}

func addLabel(img *image.RGBA, x, y int, label string, color color.Color) {
	point := fixed.Point26_6{
		X: fixed.Int26_6(x << 6),
		Y: fixed.Int26_6(y << 6),
	}
	d := &font.Drawer{
		Dst:  img,
		Src:  image.NewUniform(color),
		Face: basicfont.Face7x13,
		Dot:  point,
	}
	d.DrawString(label)
}
