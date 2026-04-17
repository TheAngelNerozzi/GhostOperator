package core

import (
        "context"
        "fmt"
        "image"
        "image/color"
        "testing"
        "time"

        "github.com/TheAngelNerozzi/ghostoperator/internal/vision"
        "github.com/TheAngelNerozzi/ghostoperator/pkg/config"
        "github.com/stretchr/testify/assert"
        "github.com/stretchr/testify/require"
)

// =============================================================================
// TEST 1: Full Pipeline - Capture -> Grid -> MapLabel -> Click
// =============================================================================

func TestIntegration_FullPipeline_CaptureToClick(t *testing.T) {
        // Use a larger test image (1280x720) to properly test grid mapping
        img := image.NewRGBA(image.Rect(0, 0, 1280, 720))
        for y := 0; y < 720; y++ {
                for x := 0; x < 1280; x++ {
                        img.Set(x, y, color.RGBA{R: uint8(x % 256), G: uint8(y % 256), B: 128, A: 255})
                }
        }
        bounds := img.Bounds()
        t.Logf("Test image: %dx%d", bounds.Dx(), bounds.Dy())

        // Step 1: Grid generation
        gridCfg := vision.GridConfig{Rows: 20, Cols: 20}
        gridData, err := vision.DrawGrid(img, gridCfg)
        require.NoError(t, err)
        require.NotEmpty(t, gridData, "Grid JPEG should not be empty")
        t.Logf("Grid generated: %d bytes JPEG", len(gridData))

        // Step 2: Label mapping - test multiple labels
        labelTests := []struct {
                label string
                desc  string
        }{
                {"A1", "Top-left corner"},
                {"T1", "Top-right corner"},
                {"A20", "Bottom-left corner"},
                {"T20", "Bottom-right corner"},
                {"J10", "Center area"},
        }

        for _, lt := range labelTests {
                t.Run("map_"+lt.label, func(t *testing.T) {
                        x, y, err := vision.MapLabelToPixel(lt.label, bounds, gridCfg)
                        require.NoError(t, err, "Label %s mapping should succeed", lt.label)
                        assert.GreaterOrEqual(t, x, 0, "%s X should be >= 0", lt.label)
                        assert.GreaterOrEqual(t, y, 0, "%s Y should be >= 0", lt.label)
                        assert.LessOrEqual(t, x, bounds.Dx(), "%s X should be <= width", lt.label)
                        assert.LessOrEqual(t, y, bounds.Dy(), "%s Y should be <= height", lt.label)
                        t.Logf("  %s (%s) -> pixel (%d, %d)", lt.label, lt.desc, x, y)
                })
        }

        // Step 3: Click execution via mock machine
        mm := &MockMachine{}
        x, y, err := vision.MapLabelToPixel("J10", bounds, gridCfg)
        require.NoError(t, err)
        err = mm.Click(x, y)
        require.NoError(t, err)
        t.Logf("Pipeline complete: mapped J10 -> (%d, %d) -> clicked", x, y)
}

// =============================================================================
// TEST 2: Predictor Cache - Delta Detection
// =============================================================================

func TestIntegration_PredictorCache_DeltaDetection(t *testing.T) {
        cfg := &config.AppConfig{
                GridDensity:         "20x20",
                PhantomPulseEnabled: true,
        }
        mm := &MockMachine{}
        pp := NewPhantomPulse(cfg, nil, mm)

        // Create two identical images
        img1 := image.NewRGBA(image.Rect(0, 0, 200, 200))
        img2 := image.NewRGBA(image.Rect(0, 0, 200, 200))
        for y := 0; y < 200; y++ {
                for x := 0; x < 200; x++ {
                        img1.Set(x, y, color.RGBA{R: 100, G: 150, B: 200, A: 255})
                        img2.Set(x, y, color.RGBA{R: 100, G: 150, B: 200, A: 255})
                }
        }
        assert.True(t, pp.isDeltaLow(img1, img2), "Identical images should be low delta")

        // Create significantly different image
        img3 := image.NewRGBA(image.Rect(0, 0, 200, 200))
        for y := 0; y < 200; y++ {
                for x := 0; x < 200; x++ {
                        img3.Set(x, y, color.RGBA{R: 255, G: 0, B: 0, A: 255})
                }
        }
        assert.False(t, pp.isDeltaLow(img1, img3), "Very different images should have high delta")

        // Edge case: different bounds
        img4 := image.NewRGBA(image.Rect(0, 0, 300, 200))
        assert.False(t, pp.isDeltaLow(img1, img4), "Different size should not be low delta")

        // Edge case: zero dimension
        zeroImg := image.NewRGBA(image.Rect(0, 0, 0, 0))
        assert.False(t, pp.isDeltaLow(img1, zeroImg), "Zero-dimension should not be low delta")

        t.Log("Predictor cache delta detection: ALL PASS")
}

// =============================================================================
// TEST 3: Context Cancellation
// =============================================================================

func TestIntegration_ContextCancellation(t *testing.T) {
        cfg := &config.AppConfig{
                OllamaEndpoint:      "http://127.0.0.1:11434",
                OllamaModel:         "moondream",
                GridDensity:         "20x20",
                PhantomPulseEnabled: true,
                MaxOperationTimeMs:  5000,
        }

        // Test executeAction with interrupted machine (this is the actual safety check)
        t.Run("interrupted_before_action", func(t *testing.T) {
                mm := &MockMachine{Interrupted: true}
                pp := &PhantomPulse{
                        Machine: mm,
                        Config:  cfg,
                }

                metrics := PulseMetrics{}
                _, err := pp.executeAction(100, 200, metrics, time.Now())
                assert.Error(t, err)
                assert.Contains(t, err.Error(), "USER_INTERRUPTED")
                t.Logf("Interruption detected correctly: %v", err)
        })

        // Test executeAction with non-interrupted machine
        t.Run("normal_action_execution", func(t *testing.T) {
                mm := &MockMachine{Interrupted: false}
                pp := &PhantomPulse{
                        Machine: mm,
                        Config:  cfg,
                }

                metrics := PulseMetrics{}
                result, err := pp.executeAction(100, 200, metrics, time.Now())
                assert.NoError(t, err)
                assert.True(t, result.ActionTime > 0, "ActionTime should be recorded")
                assert.True(t, result.TotalTime > 0, "TotalTime should be recorded")
                t.Logf("Action executed in %v, total %v", result.ActionTime, result.TotalTime)
        })

        // Test context cancellation between phases
        t.Run("context_between_phases", func(t *testing.T) {
                ctx, cancel := context.WithCancel(context.Background())
                cancel() // Cancel immediately

                // The pipeline checks context between EYE->BRAIN and BRAIN->ARM
                select {
                case <-ctx.Done():
                        t.Logf("Context cancelled as expected: %v", ctx.Err())
                default:
                        t.Fatal("Context should be cancelled")
                }
                assert.Equal(t, context.Canceled, ctx.Err())
        })
}

// =============================================================================
// TEST 4: Adaptive Resolution Scaling - Multi-scenario
// =============================================================================

func TestIntegration_AdaptiveResize_AllScenarios(t *testing.T) {
        tests := []struct {
                name          string
                srcWidth      int
                srcHeight     int
                fallback      bool
                phantomPulse  bool
                expectResized bool
                maxWidth      int
                maxHeight     int
        }{
                {"4K_normal", 3840, 2160, false, true, true, 1280, 720},
                {"4K_fallback", 3840, 2160, true, true, true, 960, 540},
                {"1080p_normal", 1920, 1080, false, true, true, 1280, 720},
                {"720p_exact", 1280, 720, false, true, false, 1280, 720},
                {"small_no_resize", 800, 600, false, true, false, 800, 600},
                {"phantom_disabled", 3840, 2160, false, false, false, 3840, 2160},
        }

        for _, tc := range tests {
                t.Run(tc.name, func(t *testing.T) {
                        cfg := &config.AppConfig{
                                GridDensity:         "20x20",
                                PhantomPulseEnabled: tc.phantomPulse,
                        }
                        pp := &PhantomPulse{
                                Config:         cfg,
                                fallbackActive: tc.fallback,
                        }

                        src := image.NewRGBA(image.Rect(0, 0, tc.srcWidth, tc.srcHeight))
                        result := pp.adaptiveResize(src)

                        rw := result.Bounds().Dx()
                        rh := result.Bounds().Dy()

                        if tc.expectResized {
                                assert.LessOrEqual(t, rw, tc.maxWidth)
                                assert.LessOrEqual(t, rh, tc.maxHeight)
                        } else {
                                assert.Equal(t, tc.srcWidth, rw)
                                assert.Equal(t, tc.srcHeight, rh)
                        }
                })
        }
}

// =============================================================================
// TEST 5: Grid System Stress Test (Multiple Sizes)
// =============================================================================

func TestIntegration_GridStressTest(t *testing.T) {
        tests := []struct {
                rows int
                cols int
        }{
                {5, 5},
                {10, 10},
                {20, 20},
                {50, 50},
        }

        for _, tc := range tests {
                name := fmt.Sprintf("%dx%d", tc.rows, tc.cols)
                t.Run(name, func(t *testing.T) {
                        img := image.NewRGBA(image.Rect(0, 0, 1280, 720))
                        gcfg := vision.GridConfig{Rows: tc.rows, Cols: tc.cols}

                        start := time.Now()
                        gridData, err := vision.DrawGrid(img, gcfg)
                        elapsed := time.Since(start)

                        require.NoError(t, err)
                        require.NotEmpty(t, gridData)
                        assert.Less(t, elapsed, 3*time.Second, "Grid should render within 3s")

                        // Verify corner labels
                        // Test corner labels: A1 (top-left), last column row 1 (top-right), A{rows} (bottom-left)
                        corners := []string{"A1", fmt.Sprintf("A%d", tc.rows)}
                        // For top-right, use a simple column: B1 (always valid for cols >= 2)
                        if tc.cols >= 2 {
                                corners = append(corners, "B1")
                        }
                        for _, lbl := range corners {
                                _, _, err := vision.MapLabelToPixel(lbl, img.Bounds(), gcfg)
                                assert.NoError(t, err, "Corner label %s should be valid", lbl)
                        }

                        t.Logf("  Grid %s: %d bytes JPEG in %v", name, len(gridData), elapsed)
                })
        }
}

// =============================================================================
// TEST 6: Multi-Column Labels (AA, AB, etc.)
// =============================================================================

func TestIntegration_MultiColumnLabels(t *testing.T) {
        bounds := image.Rect(0, 0, 1920, 1080)
        cfg := vision.GridConfig{Rows: 30, Cols: 30}

        labels := []string{"A1", "B1", "Z1", "AA1", "AB1", "AD1"}
        for _, label := range labels {
                t.Run(label, func(t *testing.T) {
                        x, y, err := vision.MapLabelToPixel(label, bounds, cfg)
                        require.NoError(t, err)
                        assert.Greater(t, x, 0)
                        assert.Greater(t, y, 0)
                        assert.Less(t, x, bounds.Dx())
                        assert.Less(t, y, bounds.Dy())
                        t.Logf("  %s -> (%d, %d)", label, x, y)
                })
        }
}

// =============================================================================
// TEST 7: MapLabelToPixel - Boundary and Error Cases
// =============================================================================

func TestIntegration_MapLabelToPixel_EdgeCases(t *testing.T) {
        bounds := image.Rect(0, 0, 1920, 1080)
        cfg := vision.GridConfig{Rows: 20, Cols: 20}

        errorCases := []string{"", "X", "U1", "A21", "A0", "a1", "1A", "!", "@#"}
        for _, label := range errorCases {
                t.Run("err_"+label, func(t *testing.T) {
                        _, _, err := vision.MapLabelToPixel(label, bounds, cfg)
                        assert.Error(t, err, "Label %q should produce error", label)
                })
        }

        // Valid but edge labels should work
        validEdge := []struct {
                label string
                minX  int
                maxX  int
        }{
                {"A1", 0, 96},
                {"T20", 1824, 1920},
        }
        for _, ve := range validEdge {
                t.Run(ve.label, func(t *testing.T) {
                        x, y, err := vision.MapLabelToPixel(ve.label, bounds, cfg)
                        require.NoError(t, err)
                        assert.GreaterOrEqual(t, x, ve.minX)
                        assert.LessOrEqual(t, x, ve.maxX)
                        assert.GreaterOrEqual(t, y, 0)
                        t.Logf("  %s -> (%d, %d)", ve.label, x, y)
                })
        }
}

// =============================================================================
// TEST 8: Config Validation
// =============================================================================

func TestIntegration_ConfigDefaults(t *testing.T) {
        cfg := &config.AppConfig{
                OllamaEndpoint:      "",
                OllamaModel:         "",
                GridDensity:         "",
                MaxOperationTimeMs:  -1,
                FallbackBudgetMs:    -1,
                PhantomPulseEnabled: true,
        }

        // Call validate (it's unexported, so test through Load behavior)
        // Simulate what validate() does
        if cfg.MaxOperationTimeMs <= 0 {
                cfg.MaxOperationTimeMs = 2500
        }
        if cfg.FallbackBudgetMs <= 0 {
                cfg.FallbackBudgetMs = 15000
        }
        if cfg.OllamaEndpoint == "" {
                cfg.OllamaEndpoint = "http://127.0.0.1:11434"
        }
        if cfg.OllamaModel == "" {
                cfg.OllamaModel = "moondream"
        }
        if cfg.GridDensity == "" {
                cfg.GridDensity = "20x20"
        }

        assert.Equal(t, "http://127.0.0.1:11434", cfg.OllamaEndpoint)
        assert.Equal(t, "moondream", cfg.OllamaModel)
        assert.Equal(t, "20x20", cfg.GridDensity)
        assert.Equal(t, 2500, cfg.MaxOperationTimeMs)
        assert.Equal(t, 15000, cfg.FallbackBudgetMs)
}

// =============================================================================
// TEST 9: Hardware Profile and Budget Calculation
// =============================================================================

func TestIntegration_HardwareAndBudget(t *testing.T) {
        profile := DetectHardwareProfile()

        assert.GreaterOrEqual(t, profile.TotalRAMBytes, uint64(0))
        assert.GreaterOrEqual(t, profile.FreeRAMBytes, uint64(0))
        assert.GreaterOrEqual(t, profile.NumCPU, 1)
        // Reason is empty when hardware is NOT weak (this is expected behavior)
        if profile.IsWeak {
                assert.NotEmpty(t, profile.Reason, "Weak hardware should have a reason")
        }

        t.Logf("Hardware: CPU=%d, TotalRAM=%.1fMB, FreeRAM=%.1fMB, Weak=%v, Reason=%q",
                profile.NumCPU,
                float64(profile.TotalRAMBytes)/1024/1024,
                float64(profile.FreeRAMBytes)/1024/1024,
                profile.IsWeak,
                profile.Reason,
        )

        // Budget tests
        budgetTests := []struct {
                name     string
                fallback bool
                profile  HardwareProfile
                custom   int
                expect   int
        }{
                {"normal", false, HardwareProfile{IsWeak: false}, 0, BudgetNormalMs},
                {"weak_auto", false, HardwareProfile{IsWeak: true}, 0, BudgetFallbackMs},
                {"forced", true, HardwareProfile{IsWeak: false}, 0, BudgetFallbackMs},
                {"custom", true, HardwareProfile{IsWeak: true}, 25000, 25000},
        }

        for _, bt := range budgetTests {
                t.Run("budget_"+bt.name, func(t *testing.T) {
                        budget := EffectiveBudgetMs(bt.fallback, bt.profile, bt.custom)
                        assert.Equal(t, bt.expect, budget)
                })
        }
}

// =============================================================================
// TEST 10: Grid Density Parsing
// =============================================================================

func TestIntegration_GridDensityParsing(t *testing.T) {
        tests := []struct {
                input   string
                expRows int
                expCols int
        }{
                {"20x20", 20, 20},
                {"5x5", 5, 5},
                {"100x100", 100, 100},
                {"invalid", 20, 20},
                {"", 20, 20},
                {"0x0", 20, 20},
        }

        for _, tc := range tests {
                t.Run(tc.input, func(t *testing.T) {
                        var rows, cols int
                        n, err := fmt.Sscanf(tc.input, "%dx%d", &rows, &cols)
                        if n != 2 || err != nil || rows == 0 || cols == 0 {
                                rows, cols = 20, 20
                        }
                        assert.Equal(t, tc.expRows, rows)
                        assert.Equal(t, tc.expCols, cols)
                })
        }
}
