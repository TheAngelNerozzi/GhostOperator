package core

import (
	"image"
	"image/color"
	"testing"

	"github.com/TheAngelNerozzi/ghostoperator/pkg/config"
	"github.com/stretchr/testify/assert"
)

type MockMachine struct {
	Interrupted bool
}
func (m *MockMachine) Capture() (image.Image, error) { return image.NewRGBA(image.Rect(0,0,10,10)), nil }
func (m *MockMachine) Move(x, y int) error { return nil }
func (m *MockMachine) Click(x, y int) error { return nil }
func (m *MockMachine) DoubleClick(x, y int) error { return nil }
func (m *MockMachine) Type(text string) error { return nil }
func (m *MockMachine) IsInterrupted() bool { return m.Interrupted }
func (m *MockMachine) ResetIntervention() { m.Interrupted = false }

func TestPhantomPulse_AdaptiveResize(t *testing.T) {
	cfg := &config.AppConfig{PhantomPulseEnabled: true}
	mm := &MockMachine{}
	pp := NewPhantomPulse(cfg, nil, mm)

	// Create a large mock image
	img := image.NewRGBA(image.Rect(0, 0, 1920, 1080))

	resized := pp.adaptiveResize(img)
	bounds := resized.Bounds()

	assert.Equal(t, 640, bounds.Dx())
	assert.Equal(t, 360, bounds.Dy())
}

func TestPhantomPulse_AdaptiveResize_Fallback(t *testing.T) {
	cfg := &config.AppConfig{
		PhantomPulseEnabled: true,
		HardwareFallback:    true,
	}
	mm := &MockMachine{}
	pp := NewPhantomPulse(cfg, nil, mm)

	img := image.NewRGBA(image.Rect(0, 0, 1920, 1080))
	resized := pp.adaptiveResize(img)
	bounds := resized.Bounds()

	// In fallback mode, should resize to 480x270
	assert.Equal(t, 480, bounds.Dx())
	assert.Equal(t, 270, bounds.Dy())
}

func TestPhantomPulse_IsDeltaLow(t *testing.T) {
	cfg := &config.AppConfig{PhantomPulseEnabled: true}
	mm := &MockMachine{}
	pp := NewPhantomPulse(cfg, nil, mm)

	img1 := image.NewRGBA(image.Rect(0, 0, 100, 100))
	img2 := image.NewRGBA(image.Rect(0, 0, 100, 100))

	// Both images are empty (transparent black), delta should be low
	assert.True(t, pp.isDeltaLow(img1, img2))

	// Modify img2 significantly to trigger high delta
	for y := 0; y < 100; y++ {
		for x := 0; x < 100; x++ {
			img2.Set(x, y, color.RGBA{R: 255, G: 255, B: 255, A: 255})
		}
	}

	assert.False(t, pp.isDeltaLow(img1, img2))
}

func TestEffectiveBudgetMs_Normal(t *testing.T) {
	profile := HardwareProfile{IsWeak: false}
	budget := EffectiveBudgetMs(false, profile, 0)
	assert.Equal(t, BudgetNormalMs, budget)
}

func TestEffectiveBudgetMs_WeakHardware(t *testing.T) {
	profile := HardwareProfile{IsWeak: true, Reason: "low RAM"}
	budget := EffectiveBudgetMs(false, profile, 0)
	assert.Equal(t, BudgetFallbackMs, budget)
}

func TestEffectiveBudgetMs_ForcedFallback(t *testing.T) {
	// Even with strong hardware, forced fallback wins
	profile := HardwareProfile{IsWeak: false}
	budget := EffectiveBudgetMs(true, profile, 0)
	assert.Equal(t, BudgetFallbackMs, budget)
}

func TestEffectiveBudgetMs_CustomBudget(t *testing.T) {
	// Custom budget from config overrides the hardcoded constant
	profile := HardwareProfile{IsWeak: true, Reason: "low RAM"}
	budget := EffectiveBudgetMs(false, profile, 7000)
	assert.Equal(t, 7000, budget)

	// Custom budget also works with forced fallback
	profile2 := HardwareProfile{IsWeak: false}
	budget2 := EffectiveBudgetMs(true, profile2, 3500)
	assert.Equal(t, 3500, budget2)
}

func TestDetectHardwareProfile_Structure(t *testing.T) {
	profile := DetectHardwareProfile()
	// Regardless of actual hardware, the structure should be populated
	assert.GreaterOrEqual(t, profile.NumCPU, 1)
	// TotalRAMBytes should be non-zero on any real machine
	assert.Greater(t, profile.TotalRAMBytes, uint64(0))
}

func TestFallbackActive(t *testing.T) {
	cfg := &config.AppConfig{HardwareFallback: true}
	mm := &MockMachine{}
	pp := NewPhantomPulse(cfg, nil, mm)
	assert.True(t, pp.FallbackActive())

	cfg2 := &config.AppConfig{HardwareFallback: false}
	mm2 := &MockMachine{}
	pp2 := NewPhantomPulse(cfg2, nil, mm2)
	assert.False(t, pp2.FallbackActive())
}
