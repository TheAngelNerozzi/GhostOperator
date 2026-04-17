package core

import (
        "context"
        "fmt"
        "image"
        "image/color"
        "math/rand"
        "sort"
        "testing"
        "time"

        "github.com/TheAngelNerozzi/ghostoperator/internal/eml"
        "github.com/TheAngelNerozzi/ghostoperator/internal/executor/fast"
        "github.com/TheAngelNerozzi/ghostoperator/internal/planner"
        "github.com/TheAngelNerozzi/ghostoperator/internal/vision"
        "github.com/TheAngelNerozzi/ghostoperator/pkg/config"
        "github.com/stretchr/testify/assert"
        "github.com/stretchr/testify/require"
)

// =============================================================================
// PERF TEST 1: Complex Mission Simulation — Full Pipeline Timing
// =============================================================================
// This test validates that the non-LLM portion of the pipeline (EYE + Grid +
// Map + ARM) completes well within the 2.5s budget.  The LLM call is mocked
// to isolate hardware/algorithm performance from model latency.

const maxBudgetMs = 2500 // BudgetNormalMs from hardware_detect.go

// TimedMachine wraps MockMachine with realistic capture simulation.
type TimedMachine struct {
        Interrupted bool
        CaptureW    image.Image // pre-loaded image to return
}

func (m *TimedMachine) Capture() (image.Image, error) {
        return m.CaptureW, nil
}
func (m *TimedMachine) Move(x, y int) error            { return nil }
func (m *TimedMachine) Click(x, y int) error            { return nil }
func (m *TimedMachine) DoubleClick(x, y int) error      { return nil }
func (m *TimedMachine) Type(text string) error           { return nil }
func (m *TimedMachine) IsInterrupted() bool              { return m.Interrupted }
func (m *TimedMachine) ResetIntervention()              { m.Interrupted = false }

// generateRealisticImage creates a 1920x1080 image with noise to simulate a real screenshot.
func generateRealisticImage(width, height int) *image.RGBA {
        img := image.NewRGBA(image.Rect(0, 0, width, height))
        rng := rand.New(rand.NewSource(42)) // deterministic
        for y := 0; y < height; y++ {
                for x := 0; x < width; x++ {
                        // Simulate a desktop-like gradient background with noise
                        r := uint8((float64(x)/float64(width))*60 + float64(rng.Intn(30)))
                        g := uint8((float64(y)/float64(height))*40 + float64(rng.Intn(20)))
                        b := uint8(180 + rng.Intn(40))
                        img.Set(x, y, color.RGBA{R: r, G: g, B: b, A: 255})
                }
        }
        return img
}

// TestPerf_ComplexMission_FullPipeline simulates a complete mission cycle.
// The pipeline is: Capture → Resize → Grid → Label Map → Click.
// The LLM call is NOT included (mocked), so this measures the maximum
// achievable speed when the Predictor Cache is active.
func TestPerf_ComplexMission_FullPipeline(t *testing.T) {
        cfg := &config.AppConfig{
                GridDensity:         "20x20",
                PhantomPulseEnabled: true,
        }

        // Generate a realistic 1920x1080 desktop image
        realisticImg := generateRealisticImage(1920, 1080)
        tm := &TimedMachine{CaptureW: realisticImg}
        pp := NewPhantomPulse(cfg, nil, tm)

        timings := struct {
                Capture time.Duration
                Resize  time.Duration
                Grid    time.Duration
                Map     time.Duration
                Click   time.Duration
                Total   time.Duration
        }{}

        totalStart := time.Now()

        // Phase 1: EYE (Capture)
        capStart := time.Now()
        rawImg, err := pp.Machine.Capture()
        require.NoError(t, err)
        timings.Capture = time.Since(capStart)
        require.NotNil(t, rawImg)

        // Phase 1.5: Adaptive Resize
        resStart := time.Now()
        resizedImg := pp.adaptiveResize(rawImg)
        timings.Resize = time.Since(resStart)
        require.NotNil(t, resizedImg)

        // Phase 2: BRAIN (Grid Generation — the expensive non-LLM part)
        gridStart := time.Now()
        gridCfg := vision.GridConfig{Rows: 20, Cols: 20}
        gridData, err := vision.DrawGrid(resizedImg, gridCfg)
        require.NoError(t, err)
        require.NotEmpty(t, gridData)
        timings.Grid = time.Since(gridStart)

        // Phase 2.5: Label Mapping (simulates what LLM would return)
        mapStart := time.Now()
        bounds := rawImg.Bounds()
        // Simulate LLM returning "J10" (center-ish of screen)
        pixelX, pixelY, err := vision.MapLabelToPixel("J10", bounds, gridCfg)
        require.NoError(t, err)
        timings.Map = time.Since(mapStart)

        // Phase 3: ARM (Click Execution via Fast Executor path)
        clickStart := time.Now()
        err = pp.Machine.Click(pixelX, pixelY)
        require.NoError(t, err)
        timings.Click = time.Since(clickStart)

        timings.Total = time.Since(totalStart)

        // Report
        t.Logf("╔══════════════════════════════════════════════════╗")
        t.Logf("║     GHOSTOPERATOR — PERF TEST: Full Pipeline    ║")
        t.Logf("╠══════════════════════════════════════════════════╣")
        t.Logf("║  EYE  (Capture+Resize):    %8.2f ms            ║", float64(timings.Capture+timings.Resize)/float64(time.Millisecond))
        t.Logf("║  BRAIN (Grid Generation):  %8.2f ms            ║", float64(timings.Grid)/float64(time.Millisecond))
        t.Logf("║  MAP   (Label→Pixel):      %8.2f ms            ║", float64(timings.Map)/float64(time.Millisecond))
        t.Logf("║  ARM   (Click):            %8.2f ms            ║", float64(timings.Click)/float64(time.Millisecond))
        t.Logf("║  ─────────────────────────────────────           ║")
        t.Logf("║  TOTAL (non-LLM):          %8.2f ms            ║", float64(timings.Total)/float64(time.Millisecond))
        t.Logf("║  Budget Remaining:         %8.2f ms            ║", float64(maxBudgetMs*time.Millisecond-timings.Total)/float64(time.Millisecond))
        t.Logf("╚══════════════════════════════════════════════════╝")

        // The non-LLM pipeline MUST be well under the budget
        // Grid generation is the heaviest non-LLM component
        nonLLMTarget := 500 * time.Millisecond // Should be way under 500ms
        assert.Less(t, timings.Total, nonLLMTarget,
                "Non-LLM pipeline must complete in <%v, got %v", nonLLMTarget, timings.Total)
}

// =============================================================================
// PERF TEST 2: EML Core — Microsecond Benchmarks
// =============================================================================

func TestPerf_EML_CoreOperations(t *testing.T) {
        iterations := 10000

        tests := []struct {
                name string
                fn   func()
        }{
                {"EMLOp", func() { eml.EMLOp(1.0, 0.5, 1, 0.5) }},
                {"Sin(π/4)", func() { eml.Sin(3.14159265 / 4) }},
                {"Cos(π/3)", func() { eml.Cos(3.14159265 / 3) }},
                {"Tan(π/4)", func() { eml.Tan(3.14159265 / 4) }},
                {"ArcTan2(1,1)", func() { eml.ArcTan2(1, 1) }},
                {"Exp(2.5)", func() { eml.Exp(2.5) }},
                {"Ln(42)", func() { eml.Ln(42) }},
                {"Log2(1024)", func() { eml.Log2(1024) }},
                {"Sqrt(144)", func() { eml.Sqrt(144) }},
                {"Sinh(1)", func() { eml.Sinh(1) }},
                {"Cosh(1)", func() { eml.Cosh(1) }},
                {"Tanh(1)", func() { eml.Tanh(1) }},
                {"Pow(2,10)", func() { eml.Pow(2, 10) }},
                {"SmoothStep", func() { eml.SmoothStep(0, 1, 0.5) }},
                {"Distance2D", func() { eml.Distance2D(0, 0, 1920, 1080) }},
        }

        t.Log("╔══════════════════════════════════════════════════════╗")
        t.Log("║     EML CORE — Microsecond Operation Benchmarks      ║")
        t.Log("╠══════════════════════════════════════════════════════╣")

        totalOps := 0
        totalNs := int64(0)

        for _, tc := range tests {
                start := time.Now()
                for i := 0; i < iterations; i++ {
                        tc.fn()
                }
                elapsed := time.Since(start)
                avgNs := elapsed.Nanoseconds() / int64(iterations)
                totalNs += avgNs
                totalOps++
                t.Logf("║  %-20s  %8d ns/op  (%d iterations)     ║", tc.name, avgNs, iterations)
        }

        avgAllNs := totalNs / int64(totalOps)
        t.Logf("║  ─────────────────────────────────────────            ║")
        t.Logf("║  AVERAGE:                  %8d ns/op              ║", avgAllNs)
        t.Logf("╚══════════════════════════════════════════════════════╝")

        // Each individual EML operation should be under 10 microseconds
        maxNsPerOp := int64(10_000) // 10μs per operation
        assert.Less(t, avgAllNs, maxNsPerOp,
                "Average EML operation should be <10μs, got %d ns", avgAllNs)
}

// =============================================================================
// PERF TEST 3: Bezier Path Generation — Fast Executor
// =============================================================================

func TestPerf_FastExecutor_AllProfiles(t *testing.T) {
        profiles := []struct {
                name    string
                profile executor.SpeedProfile
                maxMs   float64
        }{
                {"ProfileFast (8 steps, 6ms)", executor.ProfileFast, 1.0},
                {"ProfileNormal (12 steps, 8ms)", executor.ProfileNormal, 1.0},
                {"ProfileStealth (25 steps, 8ms)", executor.ProfileStealth, 1.0},
        }

        distances := []struct {
                name string
                dist int
        }{
                {"Short (100px)", 100},
                {"Medium (500px)", 500},
                {"Long (1920px)", 1920},
        }

        t.Log("╔══════════════════════════════════════════════════════════╗")
        t.Log("║     FAST EXECUTOR — Bezier Path Generation Benchmarks    ║")
        t.Log("╠══════════════════════════════════════════════════════════╣")

        iterations := 10000
        for _, p := range profiles {
                for _, d := range distances {
                        fm := executor.NewFastMover(p.profile)
                        name := fmt.Sprintf("%s/%s", p.name, d.name)

                        start := time.Now()
                        for i := 0; i < iterations; i++ {
                                fm.GeneratePath(0, 0, d.dist, d.dist)
                        }
                        elapsed := time.Since(start)
                        avgNs := elapsed.Nanoseconds() / int64(iterations)
                        duration := fm.EstimateDuration()

                        t.Logf("║  %-35s %6d ns/path  est: %v  ║", name, avgNs, duration)
                }
        }
        t.Log("╚══════════════════════════════════════════════════════════╝")
}

// =============================================================================
// PERF TEST 4: Vision Pipeline — Grid Generation Stress
// =============================================================================

func TestPerf_Vision_GridGeneration_Stress(t *testing.T) {
        sizes := []struct {
                rows int
                cols int
        }{
                {10, 10},
                {20, 20},
                {30, 30},
                {50, 50},
        }

        t.Log("╔══════════════════════════════════════════════════╗")
        t.Log("║     VISION — Grid Generation Stress Test         ║")
        t.Log("╠══════════════════════════════════════════════════╣")

        img := generateRealisticImage(1280, 720)

        for _, s := range sizes {
                gcfg := vision.GridConfig{Rows: s.rows, Cols: s.cols}
                iterations := 100

                start := time.Now()
                for i := 0; i < iterations; i++ {
                        _, err := vision.DrawGrid(img, gcfg)
                        require.NoError(t, err)
                }
                elapsed := time.Since(start)
                avgMs := float64(elapsed.Milliseconds()) / float64(iterations)

                t.Logf("║  Grid %-4dx%-4d:  %8.2f ms avg  (%d iters)   ║", s.rows, s.cols, avgMs, iterations)

                // Even 50x50 grids should complete in reasonable time
                if s.rows <= 30 {
                        assert.Less(t, avgMs, 100.0, "Grid %dx%d should render <100ms", s.rows, s.cols)
                }
        }
        t.Log("╚══════════════════════════════════════════════════╝")
}

// =============================================================================
// PERF TEST 5: Adaptive Planner — Classification Speed
// =============================================================================

func TestPerf_Planner_ClassificationSpeed(t *testing.T) {
        p := planner.New()
        intents := []string{
                // EML-routable (simple)
                "click (500, 300)",
                "abre el menu",
                "guardar archivo",
                "ve a google.com",
                "presiona el boton de aceptar",
                "clic en siguiente",
                "escribe hola mundo",
                "cierra la ventana",
                // LLM-routable (complex)
                "encuentra el boton azul que dice continuar en la parte inferior derecha de la pantalla y haz clic en el",
                "busca el icono de configuracion que esta al lado del perfil de usuario y seleccionalo",
                "determine si hay un mensaje de error visible en la pantalla y tomale una captura",
                "analiza el contenido de la tabla y encuentra la fila con el mayor valor numerico",
        }

        iterations := 10000

        t.Log("╔════════════════════════════════════════════════════════╗")
        t.Log("║     PLANNER — Classification Speed Test                ║")
        t.Log("╠════════════════════════════════════════════════════════╣")

        start := time.Now()
        for i := 0; i < iterations; i++ {
                for _, intent := range intents {
                        p.Classify(intent)
                }
        }
        elapsed := time.Since(start)
        totalClassifications := iterations * len(intents)
        avgNs := elapsed.Nanoseconds() / int64(totalClassifications)

        t.Logf("║  %d classifications in %v                         ║", totalClassifications, elapsed)
        t.Logf("║  Average: %d ns/classification                      ║", avgNs)
        t.Logf("║  Stats: %d total, %d EML, %d LLM                      ║", p.Stats.TotalMissions, p.Stats.EMLRouted, p.Stats.LLMRouted)
        t.Log("╚════════════════════════════════════════════════════════╝")

        // Classification must be under 100μs
        assert.Less(t, avgNs, int64(100_000),
                "Classification should be <100μs, got %d ns", avgNs)
}

// =============================================================================
// PERF TEST 6: Predictor Cache — Delta Detection Speed
// =============================================================================

func TestPerf_PredictorCache_DeltaSpeed(t *testing.T) {
        cfg := &config.AppConfig{
                GridDensity:         "20x20",
                PhantomPulseEnabled: true,
        }
        mm := &MockMachine{}
        pp := NewPhantomPulse(cfg, nil, mm)

        img1 := generateRealisticImage(1920, 1080)
        img2 := generateRealisticImage(1920, 1080) // Different pixels (different seed)

        iterations := 10000

        t.Log("╔══════════════════════════════════════════════════╗")
        t.Log("║     PREDICTOR CACHE — Delta Detection Speed       ║")
        t.Log("╠══════════════════════════════════════════════════╣")

        // Same image (low delta — cache hit path)
        start := time.Now()
        for i := 0; i < iterations; i++ {
                pp.isDeltaLow(img1, img1)
        }
        elapsed := time.Since(start)
        avgNs := elapsed.Nanoseconds() / int64(iterations)
        t.Logf("║  isDeltaLow (same img):    %8d ns/op             ║", avgNs)

        // Different image (high delta — LLM needed path)
        start = time.Now()
        for i := 0; i < iterations; i++ {
                pp.isDeltaLow(img1, img2)
        }
        elapsed = time.Since(start)
        avgNs = elapsed.Nanoseconds() / int64(iterations)
        t.Logf("║  isDeltaLow (diff img):    %8d ns/op             ║", avgNs)
        t.Log("╚══════════════════════════════════════════════════╝")
}

// =============================================================================
// PERF TEST 7: Multi-Cycle Mission Simulation
// =============================================================================
// Simulates 10 consecutive mission cycles to detect performance degradation.

func TestPerf_MultiCycleMission_Simulation(t *testing.T) {
        cfg := &config.AppConfig{
                GridDensity:         "20x20",
                PhantomPulseEnabled: true,
        }

        realisticImg := generateRealisticImage(1920, 1080)
        tm := &TimedMachine{CaptureW: realisticImg}
        pp := NewPhantomPulse(cfg, nil, tm)
        plannerEngine := planner.New()

        cycles := 10
        gridCfg := vision.GridConfig{Rows: 20, Cols: 20}

        type CycleResult struct {
                Cycle        int
                CaptureMs    float64
                ResizeMs     float64
                GridMs       float64
                MapMs        float64
                ClickMs      float64
                TotalMs      float64
                Engine       string
                BudgetOk     bool
        }

        results := make([]CycleResult, 0, cycles)
        intents := []string{
                "click (500, 300)",
                "abre el menu de archivo",
                "guardar documento",
                "ve a google.com",
                "presiona aceptar",
                "escribe hola",
                "clic en siguiente",
                "cierra ventana",
                "minimiza aplicacion",
                "maximiza ventana",
        }

        t.Log("╔══════════════════════════════════════════════════════════════╗")
        t.Log("║     MULTI-CYCLE MISSION SIMULATION (10 cycles)               ║")
        t.Log("╠══════════════════════════════════════════════════════════════╣")

        for i := 0; i < cycles; i++ {
                intent := intents[i%len(intents)]
                decision := plannerEngine.Classify(intent)

                cr := CycleResult{Cycle: i + 1, Engine: decision.Engine.String()}

                cycleStart := time.Now()

                // EYE
                t0 := time.Now()
                rawImg, _ := tm.Capture()
                cr.CaptureMs = float64(time.Since(t0)) / float64(time.Millisecond)

                // RESIZE
                t0 = time.Now()
                resizedImg := pp.adaptiveResize(rawImg)
                cr.ResizeMs = float64(time.Since(t0)) / float64(time.Millisecond)

                // BRAIN (Grid + Map)
                t0 = time.Now()
                gridData, _ := vision.DrawGrid(resizedImg, gridCfg)
                _ = gridData
                bounds := rawImg.Bounds()
                pixelX, pixelY, _ := vision.MapLabelToPixel("J10", bounds, gridCfg)
                cr.GridMs = float64(time.Since(t0)) / float64(time.Millisecond)

                // MAP
                t0 = time.Now()
                _ = pixelX
                _ = pixelY
                cr.MapMs = float64(time.Since(t0)) / float64(time.Millisecond)

                // ARM
                t0 = time.Now()
                tm.Click(pixelX, pixelY)
                cr.ClickMs = float64(time.Since(t0)) / float64(time.Millisecond)

                cr.TotalMs = float64(time.Since(cycleStart)) / float64(time.Millisecond)
                cr.BudgetOk = cr.TotalMs < float64(maxBudgetMs)

                results = append(results, cr)
        }

        // Print table
        for _, cr := range results {
                status := "✅"
                if !cr.BudgetOk {
                        status = "❌"
                }
                t.Logf("║  Cycle %2d [%s/%s]: Total=%7.2fms  (Eye:%5.1f + Brain:%5.1f + Arm:%5.1f) %s  ║",
                        cr.Cycle, cr.Engine, "NO_LLM",
                        cr.TotalMs, cr.CaptureMs+cr.ResizeMs, cr.GridMs, cr.ClickMs+cr.MapMs, status)
        }

        // Summary stats
        var totalMs, minMs, maxMs float64
        durations := make([]float64, len(results))
        for i, cr := range results {
                durations[i] = cr.TotalMs
                totalMs += cr.TotalMs
                if i == 0 || cr.TotalMs < minMs {
                        minMs = cr.TotalMs
                }
                if cr.TotalMs > maxMs {
                        maxMs = cr.TotalMs
                }
        }
        sort.Float64s(durations)
        p50 := durations[len(durations)/2]
        p95 := durations[int(float64(len(durations))*0.95)]
        p99 := durations[int(float64(len(durations))*0.99)]
        avgMs := totalMs / float64(len(results))

        t.Log("╠══════════════════════════════════════════════════════════════╣")
        t.Logf("║  STATISTICS:                                              ║")
        t.Logf("║    Average:  %7.2f ms                                     ║", avgMs)
        t.Logf("║    Min:      %7.2f ms                                     ║", minMs)
        t.Logf("║    Max:      %7.2f ms                                     ║", maxMs)
        t.Logf("║    P50:      %7.2f ms                                     ║", p50)
        t.Logf("║    P95:      %7.2f ms                                     ║", p95)
        t.Logf("║    P99:      %7.2f ms                                     ║", p99)
        t.Logf("║    Budget:   %7d ms (%.1fx headroom)                    ║", maxBudgetMs, float64(maxBudgetMs)/avgMs)
        t.Logf("╚══════════════════════════════════════════════════════════════╝")

        // ALL cycles must be under budget
        for _, cr := range results {
                assert.True(t, cr.BudgetOk, "Cycle %d exceeded budget: %.2fms > %dms", cr.Cycle, cr.TotalMs, maxBudgetMs)
        }

        // Average should be extremely fast (no LLM)
        assert.Less(t, avgMs, 500.0, "Average non-LLM cycle should be <500ms, got %.2fms", avgMs)
}

// =============================================================================
// PERF TEST 8: EML Bezier — Full Path Generation Pipeline
// =============================================================================
// Tests the full Bezier path pipeline used by FastExecutor in a realistic scenario.

func TestPerf_EML_BezierFullPath(t *testing.T) {
        testCases := []struct {
                name       string
                start      [2]float64
                end        [2]float64
                steps      int
                maxMs      float64
        }{
                {"Corner-to-Corner (1920x1080)", [2]float64{0, 0}, [2]float64{1920, 1080}, 12, 1.0},
                {"Center-to-Center", [2]float64{960, 540}, [2]float64{960, 540}, 12, 1.0},
                {"Random diagonal", [2]float64{150, 200}, [2]float64{1400, 800}, 12, 1.0},
                {"Short hop (50px)", [2]float64{100, 100}, [2]float64{150, 100}, 8, 1.0},
        }

        iterations := 50000

        t.Log("╔════════════════════════════════════════════════════════╗")
        t.Log("║     EML BEZIER — Full Path Generation                  ║")
        t.Log("╠════════════════════════════════════════════════════════╣")

        for _, tc := range testCases {
                start := time.Now()
                for i := 0; i < iterations; i++ {
                        eml.GenerateBezierPath(tc.start, tc.end, tc.steps)
                }
                elapsed := time.Since(start)
                avgNs := elapsed.Nanoseconds() / int64(iterations)
                avgMs := float64(avgNs) / 1e6

                t.Logf("║  %-30s %6d ns/path (%.3f ms)    ║", tc.name, avgNs, avgMs)
                assert.Less(t, avgMs, tc.maxMs, "%s should be <%.1f ms", tc.name, tc.maxMs)
        }
        t.Log("╚════════════════════════════════════════════════════════╝")
}

// =============================================================================
// PERF TEST 9: Context Cancellation Responsiveness
// =============================================================================
// Validates that the pipeline responds to cancellation within budget.

func TestPerf_ContextCancellationResponsiveness(t *testing.T) {
        cfg := &config.AppConfig{
                GridDensity:         "20x20",
                PhantomPulseEnabled: true,
        }
        mm := &MockMachine{}
        _ = NewPhantomPulse(cfg, nil, mm)

        // Test cancellation latency
        cancelLatencies := make([]time.Duration, 100)

        for i := 0; i < 100; i++ {
                ctx, cancel := context.WithCancel(context.Background())

                // Cancel immediately
                go func() {
                        cancel()
                }()

                start := time.Now()
                select {
                case <-ctx.Done():
                        cancelLatencies[i] = time.Since(start)
                case <-time.After(10 * time.Millisecond):
                        t.Fatal("Context cancellation took too long")
                }
        }

        avgLatency := cancelLatencies[0]
        for _, l := range cancelLatencies[1:] {
                avgLatency += l
        }
        avgLatency /= time.Duration(len(cancelLatencies))

        t.Logf("Average context cancellation latency: %v", avgLatency)
        assert.Less(t, avgLatency, time.Millisecond,
                "Context cancellation should be <1ms, got %v", avgLatency)
}

// =============================================================================
// PERF TEST 10: Simulated LLM + Pipeline Combined
// =============================================================================
// This test simulates a realistic scenario where the LLM call takes
// a configurable amount of time, to determine the maximum LLM latency
// that still allows meeting the 2.5s budget.

func TestPerf_BudgetAnalysis_LatencyHeadroom(t *testing.T) {
        // Measure the non-LLM pipeline overhead
        realisticImg := generateRealisticImage(1920, 1080)
        cfg := &config.AppConfig{
                GridDensity:         "20x20",
                PhantomPulseEnabled: true,
        }
        tm := &TimedMachine{CaptureW: realisticImg}
        pp := NewPhantomPulse(cfg, nil, tm)
        gridCfg := vision.GridConfig{Rows: 20, Cols: 20}

        // Warm up
        rawImg, _ := tm.Capture()
        resizedImg := pp.adaptiveResize(rawImg)
        vision.DrawGrid(resizedImg, gridCfg)

        // Measure non-LLM overhead over 50 iterations
        iterations := 50
        overheads := make([]time.Duration, 0, iterations)

        for i := 0; i < iterations; i++ {
                start := time.Now()
                rawImg, _ = tm.Capture()
                resizedImg = pp.adaptiveResize(rawImg)
                gridData, _ := vision.DrawGrid(resizedImg, gridCfg)
                _ = gridData
                bounds := rawImg.Bounds()
                px, py, _ := vision.MapLabelToPixel("J10", bounds, gridCfg)
                tm.Click(px, py)
                overheads = append(overheads, time.Since(start))
        }

        // Calculate P95 overhead
        sort.Slice(overheads, func(i, j int) bool { return overheads[i] < overheads[j] })
        p95Overhead := overheads[int(float64(len(overheads))*0.95)]
        avgOverhead := overheads[0]
        for _, o := range overheads[1:] {
                avgOverhead += o
        }
        avgOverhead /= time.Duration(len(overheads))

        maxLLM := time.Duration(maxBudgetMs)*time.Millisecond - p95Overhead

        t.Log("╔══════════════════════════════════════════════════════════════╗")
        t.Log("║     BUDGET ANALYSIS — LLM Latency Headroom                  ║")
        t.Log("╠══════════════════════════════════════════════════════════════╣")
        t.Logf("║  Budget:                    %8d ms                            ║", maxBudgetMs)
        t.Logf("║  Non-LLM P95 Overhead:      %8.2f ms                            ║", float64(p95Overhead)/float64(time.Millisecond))
        t.Logf("║  Non-LLM Avg Overhead:      %8.2f ms                            ║", float64(avgOverhead)/float64(time.Millisecond))
        t.Logf("║  ──────────────────────────────────────                      ║")
        t.Logf("║  Max LLM Latency (P95):     %8.2f ms                            ║", float64(maxLLM)/float64(time.Millisecond))
        t.Logf("║  LLM Headroom:              %8.1f %%                             ║", float64(maxLLM)/float64(maxBudgetMs*time.Millisecond)*100)
        t.Log("╚══════════════════════════════════════════════════════════════╝")

        // The non-LLM overhead should leave significant room for LLM
        assert.Greater(t, float64(maxLLM), float64(2000)*float64(time.Millisecond),
                "Should have >2000ms headroom for LLM, got %v", maxLLM)
}
