package vision

import (
	"image"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMapLabelToPixel(t *testing.T) {
	bounds := image.Rect(0, 0, 1000, 1000)
	config := GridConfig{Rows: 10, Cols: 10}

	tests := []struct {
		label   string
		wantX   int
		wantY   int
		wantErr bool
	}{
		{"A1", 50, 50, false},
		{"B2", 150, 150, false},
		{"J10", 950, 950, false},
		{"K11", 0, 0, true},
		{"Z99", 0, 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.label, func(t *testing.T) {
			x, y, err := MapLabelToPixel(tt.label, bounds, config)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantX, x)
				assert.Equal(t, tt.wantY, y)
			}
		})
	}
}

func TestGridConfig_Validation(t *testing.T) {
	config := GridConfig{Rows: 10, Cols: 10}
	assert.Equal(t, 10, config.Rows)
	assert.Equal(t, 10, config.Cols)
}
