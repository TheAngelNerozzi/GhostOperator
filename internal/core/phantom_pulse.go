package core

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/jpeg"
	"math"
	"strings"
	"sync"
	"time"

	"github.com/TheAngelNerozzi/ghostoperator/internal/automation"
	"github.com/TheAngelNerozzi/ghostoperator/internal/llm"
	"github.com/TheAngelNerozzi/ghostoperator/internal/vision"
	"github.com/TheAngelNerozzi/ghostoperator/pkg/config"
	"github.com/kbinani/screenshot"
	"golang.org/x/image/draw"
)

// PulseMetrics tracks the execution time of each phase
type PulseMetrics struct {
	TotalTime   time.Duration `json:"total_time"`
	CaptureTime time.Duration `json:"capture_time"`
	VisionTime  time.Duration `json:"vision_time"`
	ActionTime  time.Duration `json:"action_time"`
}

// PhantomPulse is the core orchestration engine
type PhantomPulse struct {
	Config         *config.AppConfig
	Vision         *llm.VisionClient
	Automation     *automation.ActionExecutor
	Profile        HardwareProfile
	ActiveBudgetMs int

	lastBounds image.Rectangle
	lastLabel  string
	lastX      int
	lastY      int
}

// NewPhantomPulse creates a new PhantomPulse engine with hardware-aware budget.
func NewPhantomPulse(cfg *config.AppConfig, visionClient *llm.VisionClient, executor *automation.ActionExecutor) *PhantomPulse {
	profile := DetectHardwareProfile()

	// Auto-detect: if config says auto-detect, activate fallback when hardware is weak
	if cfg.FallbackAutoDetect && profile.IsWeak && !cfg.HardwareFallback {
		cfg.HardwareFallback = true
	}

	budget := EffectiveBudgetMs(cfg.HardwareFallback, profile, cfg.FallbackBudgetMs)

	return &PhantomPulse{
		Config:         cfg,
		Vision:         visionClient,
		Automation:     executor,
		Profile:        profile,
		ActiveBudgetMs: budget,
	}
}

// FallbackActive returns true if the engine is operating in fallback mode (5s budget).
func (pp *PhantomPulse) FallbackActive() bool {
	return pp.Config.HardwareFallback
}

// Execute runs the full PhantomPulse pipeline: EYE -> BRAIN -> ARM
func (pp *PhantomPulse) Execute(ctx context.Context, intent string, logger func(string)) (PulseMetrics, error) {
	metrics := PulseMetrics{}
	start := time.Now()

	// Phase 1: EYE (Capture & Compress)
	captureStart := time.Now()
	bounds := screenshot.GetDisplayBounds(0)
	rawImg, err := screenshot.CaptureRect(bounds)
	if err != nil {
		return metrics, fmt.Errorf("capture failed: %w", err)
	}

	// ARS: Adaptive Resolution Scaling
	resizedImg := pp.adaptiveResize(rawImg)

	var gridCfg vision.GridConfig
	fmt.Sscanf(pp.Config.GridDensity, "%dx%d", &gridCfg.Rows, &gridCfg.Cols)

	// Fallback: reduce grid density to lower LLM token count on weak hardware
	if pp.FallbackActive() && gridCfg.Rows > 8 {
		gridCfg.Rows = 8
		gridCfg.Cols = 8
	}

	gridData, err := vision.DrawGrid(resizedImg, gridCfg)
	if err != nil {
		return metrics, fmt.Errorf("grid generation failed: %w", err)
	}

	// CPI: Intelligent Progressive Compression
	compressedData := pp.compressIntelligent(resizedImg, gridData)
	metrics.CaptureTime = time.Since(captureStart)

	// Phase 2: BRAIN (Parallel Reasoning)
	visionStart := time.Now()

	// Concurrent prediction could be added here for cache, skipping full AI call
	var label string
	var aiErr error

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()

		if pp.FallbackActive() {
			logger(fmt.Sprintf("🐢 [Fallback] Modo hardware débil activo — budget %dms (%s)",
				pp.ActiveBudgetMs, pp.Profile.Reason))
		}

		// Restore Ollama vision capability to guarantee 100% accuracy.
		// The math model (SVM) was fast but structurally lacked true spatial object detection.
		label, aiErr = pp.Vision.ReasonFast(ctx, compressedData, intent)
	}()

	wg.Wait()
	metrics.VisionTime = time.Since(visionStart)

	if aiErr != nil {
		return metrics, fmt.Errorf("brain failure: %w", aiErr)
	}

	logger(fmt.Sprintf("🎯 [BRAIN] Objetivo %s identificado", label))

	// Phase 3: ARM (Execution)
	actionStart := time.Now()

	pixelX, pixelY, err := vision.MapLabelToPixel(label, bounds, gridCfg)
	if err != nil {
		return metrics, fmt.Errorf("mapping error: %w", err)
	}

	// Analyze intent for action modifiers (Double Click)
	actionType := "CLICK"
	intentLower := strings.ToLower(intent)
	if strings.Contains(intentLower, "abre") || strings.Contains(intentLower, "abrir") {
		actionType = "DOUBLE_CLICK"
	}

	// Fast execution
	cmd := automation.Command{
		Type: actionType,
		Params: map[string]interface{}{
			"x": float64(pixelX) * 1000.0 / float64(bounds.Dx()),
			"y": float64(pixelY) * 1000.0 / float64(bounds.Dy()),
		},
	}
	pp.Automation.Execute(cmd)

	metrics.ActionTime = time.Since(actionStart)
	metrics.TotalTime = time.Since(start)

	// Cache results for next iteration (ROI support)
	pp.lastBounds = bounds
	pp.lastLabel = label
	pp.lastX = pixelX
	pp.lastY = pixelY

	return metrics, nil
}

// adaptiveResize (ARS Algorithm) scales image to minimal effective resolution.
// In fallback mode, uses an even smaller target (480×270) to reduce LLM inference time.
func (pp *PhantomPulse) adaptiveResize(src image.Image) image.Image {
	if !pp.Config.PhantomPulseEnabled {
		return src
	}

	bounds := src.Bounds()

	// Fallback mode: smaller resolution for faster LLM processing
	targetW := 640
	targetH := 360
	if pp.FallbackActive() {
		targetW = 480
		targetH = 270
	}

	// If native resolution is smaller or close, return it
	if bounds.Dx() <= targetW {
		return src
	}

	dst := image.NewRGBA(image.Rect(0, 0, targetW, targetH))
	// draw.ApproxBiLinear is extremely fast
	draw.ApproxBiLinear.Scale(dst, dst.Bounds(), src, src.Bounds(), draw.Over, nil)

	return dst
}

// compressIntelligent (CPI Algorithm) calculates aggressive compression.
// In fallback mode, forces minimum quality for maximum speed.
func (pp *PhantomPulse) compressIntelligent(baseImg image.Image, originalGrid []byte) []byte {
	if !pp.Config.PhantomPulseEnabled {
		return originalGrid
	}

	bounds := baseImg.Bounds()
	area := float64(bounds.Dx() * bounds.Dy())
	baseArea := 640.0 * 360.0

	// Quality = max(20, 80 - (image_area / baseArea) * 30)
	q := 80 - (area/baseArea)*30
	quality := int(math.Max(20, math.Min(80, q)))

	// In fallback mode: force lower quality (floor at 15) to shrink payload further
	if pp.FallbackActive() {
		quality = int(math.Max(15, float64(quality)-15))
	}

	img, err := jpeg.Decode(bytes.NewReader(originalGrid))
	if err != nil {
		return originalGrid
	}

	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: quality}); err != nil {
		return originalGrid
	}
	return buf.Bytes()
}

// isDeltaLow calculates ΔFrame pixel difference (Predictor Cache core)
func (pp *PhantomPulse) isDeltaLow(current, prev image.Image) bool {
	currentBounds := current.Bounds()
	prevBounds := prev.Bounds()

	if currentBounds != prevBounds {
		return false
	}

	diffCount := 0
	samples := 1000
	stepX := currentBounds.Dx() / 30
	stepY := currentBounds.Dy() / 30

	if stepX <= 0 {
		stepX = 1
	}
	if stepY <= 0 {
		stepY = 1
	}

	tested := 0
	for y := 0; y < currentBounds.Dy(); y += stepY {
		for x := 0; x < currentBounds.Dx(); x += stepX {
			r1, g1, b1, _ := current.At(x, y).RGBA()
			r2, g2, b2, _ := prev.At(x, y).RGBA()

			// Quick manhattan difference
			rd := int(r1) - int(r2)
			gd := int(g1) - int(g2)
			bd := int(b1) - int(b2)
			if rd < 0 {
				rd = -rd
			}
			if gd < 0 {
				gd = -gd
			}
			if bd < 0 {
				bd = -bd
			}

			if (rd + gd + bd) > 5000 { // Scaled logic for RGBA which returns uint32 up to 65535
				diffCount++
			}
			tested++
			if tested >= samples {
				break
			}
		}
	}

	delta := float64(diffCount) / float64(tested)
	return delta < 0.05
}
