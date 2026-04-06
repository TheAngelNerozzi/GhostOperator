package core

import (
	"context"
	"fmt"
	"image"
	"time"

	"github.com/TheAngelNerozzi/ghostoperator/internal/llm"
	"github.com/TheAngelNerozzi/ghostoperator/internal/machine"
	"github.com/TheAngelNerozzi/ghostoperator/internal/vision"
	"github.com/TheAngelNerozzi/ghostoperator/pkg/config"
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
	Vision         *llm.VisionClient
	Machine        machine.Machine
	Profile        HardwareProfile
	Config         *config.AppConfig
	ActiveBudgetMs int

	// Predictor Cache (Phase B: Fluidez Extrema)
	lastX      int
	lastY      int
	lastInput  image.Image
	lastIntent string
}

// NewPhantomPulse creates a new PhantomPulse engine with hardware-aware budget.
func NewPhantomPulse(cfg *config.AppConfig, visionClient *llm.VisionClient, m machine.Machine) *PhantomPulse {
	profile := DetectHardwareProfile()

	// Auto-detect: if config says auto-detect, activate fallback when hardware is weak
	if cfg.FallbackAutoDetect && profile.IsWeak && !cfg.HardwareFallback {
		cfg.HardwareFallback = true
	}

	budget := 2500 // Normal budget
	if cfg.HardwareFallback {
		budget = cfg.FallbackBudgetMs
	}

	return &PhantomPulse{
		Vision:         visionClient,
		Machine:        m,
		Profile:        profile,
		Config:         cfg,
		ActiveBudgetMs: budget,
	}
}

// FallbackActive returns true if the engine is operating in fallback mode.
func (pp *PhantomPulse) FallbackActive() bool {
	return pp.Config.HardwareFallback
}

// isDeltaLow calculates frame difference (Predictor Cache core)
func (pp *PhantomPulse) isDeltaLow(current, prev image.Image) bool {
	cb := current.Bounds()
	pb := prev.Bounds()
	if cb != pb { return false }
	// Quick sample-based diff 
	diff := 0
	samples := 100
	for i := 0; i < samples; i++ {
		x := (i * 17) % cb.Dx()
		y := (i * 31) % cb.Dy()
		r1, g1, b1, _ := current.At(x, y).RGBA()
		r2, g2, b2, _ := prev.At(x, y).RGBA()
		if r1 != r2 || g1 != g2 || b1 != b2 { diff++ }
	}
	return (float64(diff)/float64(samples)) < 0.05
}

// Execute runs the full PhantomPulse pipeline: EYE -> BRAIN -> ARM
func (pp *PhantomPulse) Execute(ctx context.Context, intent string, logger func(string)) (PulseMetrics, error) {
	metrics := PulseMetrics{}
	start := time.Now()

	// Phase 1: EYE (Capture & Compress)
	captureStart := time.Now()
	rawImg, err := pp.Machine.Capture()
	if err != nil {
		return metrics, fmt.Errorf("capture failed: %w", err)
	}
	metrics.CaptureTime = time.Since(captureStart)

	// --- PHASE 1.5: GHOST PREDICTOR (Phase B Core) ---
	if intent == pp.lastIntent && pp.lastInput != nil {
		if pp.isDeltaLow(rawImg, pp.lastInput) {
			logger("🚀 [PREDICTOR] UI estable detectada. Accionando sin inferencia (+2s ahorro)")
			return pp.executeAction(pp.lastX, pp.lastY, metrics, start)
		}
	}

	// Phase 2: BRAIN (VLM Reasoning)
	visionStart := time.Now()

	// ARS: Adaptive Resolution Scaling
	resizedImg := pp.adaptiveResize(rawImg)

	var gridCfg vision.GridConfig
	fmt.Sscanf(pp.Config.GridDensity, "%dx%d", &gridCfg.Rows, &gridCfg.Cols)

	// Fallback: reduce grid density to lower LLM token count on weak hardware
	if pp.FallbackActive() && gridCfg.Rows > 12 {
		gridCfg.Rows = 12
		gridCfg.Cols = 12
	}

	gridData, err := vision.DrawGrid(resizedImg, gridCfg)
	if err != nil {
		return metrics, fmt.Errorf("grid generation failed: %w", err)
	}

	// Save debug frame for UAT (User Acceptance Testing)
	_ = vision.SaveDebugFrame(gridData)

	label, err := pp.Vision.ReasonFast(ctx, gridData, intent)
	if err != nil {
		return metrics, fmt.Errorf("vision reasoning failed: %w", err)
	}

	bounds := rawImg.Bounds()
	pixelX, pixelY, err := vision.MapLabelToPixel(label, bounds, gridCfg)
	if err != nil {
		return metrics, fmt.Errorf("mapping label to pixel failed: %w", err)
	}

	metrics.VisionTime = time.Since(visionStart)

	// Update Predictor Cache for next cycle
	pp.lastInput = rawImg
	pp.lastIntent = intent
	pp.lastX = pixelX
	pp.lastY = pixelY

	// Phase 3: ARM (Execution)
	return pp.executeAction(pixelX, pixelY, metrics, start)
}

func (pp *PhantomPulse) executeAction(x, y int, metrics PulseMetrics, start time.Time) (PulseMetrics, error) {
	actionStart := time.Now()

	// Interruption Check BEFORE moving
	if pp.Machine.IsInterrupted() {
		return metrics, fmt.Errorf("USER_INTERRUPTED: Has movido el ratón. Deteniendo misión.")
	}

	// Execution via Universal Machine Interface
	err := pp.Machine.Click(x, y)
	if err != nil {
		return metrics, err
	}

	// Post-Action Interruption Check
	if pp.Machine.IsInterrupted() {
		return metrics, fmt.Errorf("USER_INTERRUPTED: Has movido el ratón durante la ejecución.")
	}

	metrics.ActionTime = time.Since(actionStart)
	metrics.TotalTime = time.Since(start)

	return metrics, nil
}

// adaptiveResize (ARS Algorithm) scales image to minimal effective resolution.
func (pp *PhantomPulse) adaptiveResize(src image.Image) image.Image {
	if !pp.Config.PhantomPulseEnabled {
		return src
	}

	bounds := src.Bounds()
	targetW := 1280
	targetH := 720
	if pp.FallbackActive() {
		targetW = 960
		targetH = 540
	}

	if bounds.Dx() <= targetW {
		return src
	}

	dst := image.NewRGBA(image.Rect(0, 0, targetW, targetH))
	draw.ApproxBiLinear.Scale(dst, dst.Bounds(), src, src.Bounds(), draw.Over, nil)
	return dst
}
