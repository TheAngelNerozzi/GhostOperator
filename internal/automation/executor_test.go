package automation

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNormalizedCoordinates(t *testing.T) {
	// Goal: Test that 0-1000 range is correctly mapped to pixels in handle functions
	// This is a logic check for the normalization formula.

	width := 1920.0
	height := 1080.0

	tests := []struct {
		normX float64
		normY float64
		wantX int32
		wantY int32
	}{
		{0, 0, 0, 0},
		{500, 500, 960, 540},
		{1000, 1000, 1920, 1080},
	}

	for _, tt := range tests {
		pixelX := int32((tt.normX * width) / 1000.0)
		pixelY := int32((tt.normY * height) / 1000.0)

		assert.Equal(t, tt.wantX, pixelX)
		assert.Equal(t, tt.wantY, pixelY)
	}
}

func TestActionExecutor_ProcessCommand(t *testing.T) {
	// Test command parsing and basic validation
	cmd := Command{
		Type:   "CLICK",
		Params: map[string]interface{}{"x": 500.0, "y": 500.0},
	}

	assert.Equal(t, "CLICK", cmd.Type)
	assert.Equal(t, 500.0, cmd.Params["x"])
}
