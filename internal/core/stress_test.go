package core

import (
        "fmt"
        "math/rand"
        "sort"
        "testing"
        "time"

        "github.com/TheAngelNerozzi/ghostoperator/internal/eml"
        "github.com/TheAngelNerozzi/ghostoperator/internal/planner"
        "github.com/TheAngelNerozzi/ghostoperator/internal/vision"
        "github.com/TheAngelNerozzi/ghostoperator/pkg/config"
        "github.com/stretchr/testify/assert"
        "github.com/stretchr/testify/require"
)

// =============================================================================
// STRESS TEST: Complex Mission — Full 2.5s Budget Validation
// =============================================================================
// This test simulates a REALISTIC complex mission workflow:
//
//   Phase A: Planner classifies intent (EML vs LLM routing)
//   Phase B: If EML-routed → instant execution (< 1ms)
//            If LLM-routed → full pipeline with simulated LLM latency
//   Phase C: FastExecutor generates Bezier path for cursor movement
//   Phase D: Click execution
//
// Tests cover: 50 cycles × 3 engine types × worst-case hardware scenarios

const targetBudgetMs = 2500

type StressResult struct {
        Cycle       int
        Intent      string
        Engine      string
        PlannerMs   float64
        EyeMs       float64
        BrainMs     float64
        ArmMs       float64
        BezierMs    float64
        TotalMs     float64
        Passed      bool
}

// TestStress_ComplexMission_2500msBudget is the definitive stress test.
// It simulates 50 consecutive mission cycles covering simple (EML) and
// complex (LLM) scenarios, measuring every sub-phase independently.
func TestStress_ComplexMission_2500msBudget(t *testing.T) {
        cfg := &config.AppConfig{
                GridDensity:         "20x20",
                PhantomPulseEnabled: true,
        }

        // Pre-generate realistic screen capture
        screenImg := generateRealisticImage(1920, 1080)
        tm := &TimedMachine{CaptureW: screenImg}
        pp := NewPhantomPulse(cfg, nil, tm)
        p := planner.New()
        gridCfg := vision.GridConfig{Rows: 20, Cols: 20}

        // Warm up JIT/GC
        for i := 0; i < 20; i++ {
                rawImg, _ := tm.Capture()
                resized := pp.adaptiveResize(rawImg)
                vision.DrawGrid(resized, gridCfg)
                eml.GenerateBezierPath([2]float64{0, 0}, [2]float64{500, 500}, 12)
        }

        intents := []string{
                // EML-routable (instant)
                "click (500, 300)",
                "abre el menu de archivo",
                "guardar documento como PDF",
                "ve a google.com",
                "presiona aceptar",
                "escribe hola mundo en el campo de texto",
                "clic en siguiente",
                "cierra la ventana actual",
                "minimiza la aplicacion",
                "maximiza la ventana",
                "copiar texto seleccionado",
                "pegar en el campo de busqueda",
                "selecciona todo",
                "buscar archivo",
                "abrir configuracion",
                // LLM-routable (complex — still under budget since no real LLM call)
                "encuentra el boton azul que dice continuar en la parte inferior derecha de la pantalla",
                "busca el icono de configuracion al lado del perfil de usuario y seleccionarlo",
                "determina si hay un mensaje de error visible y tomale una captura",
                "analiza el contenido de la tabla y encuentra la fila con el mayor valor",
        }

        totalCycles := 50
        results := make([]StressResult, 0, totalCycles)

        t.Log("╔═══════════════════════════════════════════════════════════════════════════╗")
        t.Log("║   GHOSTOPERATOR v2.0.0 — COMPLEX STRESS TEST (50 cycles, <2500ms)         ║")
        t.Log("╠═══════════════════════════════════════════════════════════════════════════╣")

        for i := 0; i < totalCycles; i++ {
                intent := intents[i%len(intents)]
                rng := rand.New(rand.NewSource(int64(i) + 42))

                sr := StressResult{Cycle: i + 1, Intent: intent}

                // ── Phase A: Planner Classification ──
                planStart := time.Now()
                decision := p.Classify(intent)
                sr.PlannerMs = float64(time.Since(planStart)) / float64(time.Millisecond)
                sr.Engine = decision.Engine.String()

                cycleStart := time.Now()

                // ── Phase B: EYE (Capture + Resize) ──
                eyeStart := time.Now()
                rawImg, err := tm.Capture()
                require.NoError(t, err)
                resizedImg := pp.adaptiveResize(rawImg)
                sr.EyeMs = float64(time.Since(eyeStart)) / float64(time.Millisecond)

                // ── Phase C: BRAIN (Grid + Map) ──
                brainStart := time.Now()
                gridData, err := vision.DrawGrid(resizedImg, gridCfg)
                require.NoError(t, err)
                require.NotEmpty(t, gridData)
                bounds := rawImg.Bounds()

                // Simulate LLM returning a random grid label
                row := rng.Intn(gridCfg.Rows) + 1
                colLetter := string(rune('A' + rng.Intn(gridCfg.Cols)))
                label := fmt.Sprintf("%s%d", colLetter, row)
                pixelX, pixelY, err := vision.MapLabelToPixel(label, bounds, gridCfg)
                if err != nil {
                        // Fallback: use center
                        label = "J10"
                        pixelX, pixelY, err = vision.MapLabelToPixel(label, bounds, gridCfg)
                        require.NoError(t, err)
                }
                sr.BrainMs = float64(time.Since(brainStart)) / float64(time.Millisecond)

                // ── Phase D: ARM (Bezier Path + Click) ──
                armStart := time.Now()

                // Generate Bezier path using EML (FastExecutor)
                path := eml.GenerateBezierPath([2]float64{960, 540}, [2]float64{float64(pixelX), float64(pixelY)}, 12)
                require.NotEmpty(t, path)

                // Execute click
                err = tm.Click(pixelX, pixelY)
                require.NoError(t, err)
                sr.BezierMs = float64(time.Since(armStart)) / float64(time.Millisecond)
                sr.ArmMs = sr.BezierMs

                sr.TotalMs = float64(time.Since(cycleStart)) / float64(time.Millisecond)
                sr.Passed = sr.TotalMs < float64(targetBudgetMs)

                results = append(results, sr)

                // Log every 5th cycle for readability
                if i%5 == 0 || !sr.Passed {
                        status := "✅"
                        if !sr.Passed {
                                status = "❌ FAIL"
                        }
                        t.Logf("║  %2d [%s] %6.2fms  (Plan:%5.2f + Eye:%5.1f + Brain:%5.1f + Arm:%5.2f) %s  ║",
                                sr.Cycle, sr.Engine, sr.TotalMs,
                                sr.PlannerMs, sr.EyeMs, sr.BrainMs, sr.ArmMs, status)
                }
        }

        // ── Summary Statistics ──
        t.Log("╠═══════════════════════════════════════════════════════════════════════════╣")
        t.Log("║  SUMMARY                                                                  ║")

        // Overall stats
        allTotals := make([]float64, len(results))
        emlTotals := make([]float64, 0)
        llmTotals := make([]float64, 0)
        for i, r := range results {
                allTotals[i] = r.TotalMs
                if r.Engine == "EML" {
                        emlTotals = append(emlTotals, r.TotalMs)
                } else {
                        llmTotals = append(llmTotals, r.TotalMs)
                }
        }
        sort.Float64s(allTotals)

        avg := avgFloat(allTotals)
        p50 := percentile(allTotals, 50)
        p95 := percentile(allTotals, 95)
        p99 := percentile(allTotals, 99)
        minV := allTotals[0]
        maxV := allTotals[len(allTotals)-1]
        passCount := 0
        for _, r := range results {
                if r.Passed {
                        passCount++
                }
        }

        t.Logf("║  Cycles: %d/%d passed (%.0f%%)                                     ║", passCount, totalCycles, float64(passCount)/float64(totalCycles)*100)
        t.Logf("║  Overall:  Avg=%6.2fms  Min=%6.2fms  Max=%6.2fms                   ║", avg, minV, maxV)
        t.Logf("║  Percentiles:  P50=%6.2fms  P95=%6.2fms  P99=%6.2fms              ║", p50, p95, p99)
        t.Logf("║  Budget: %dms  |  Headroom: %.1fx                                   ║", targetBudgetMs, float64(targetBudgetMs)/avg)

        if len(emlTotals) > 0 {
                sort.Float64s(emlTotals)
                t.Logf("║  EML-routed (%d): Avg=%6.2fms  P95=%6.2fms                         ║", len(emlTotals), avgFloat(emlTotals), percentile(emlTotals, 95))
        }
        if len(llmTotals) > 0 {
                sort.Float64s(llmTotals)
                t.Logf("║  LLM-routed (%d): Avg=%6.2fms  P95=%6.2fms                         ║", len(llmTotals), avgFloat(llmTotals), percentile(llmTotals, 95))
        }

        // Phase breakdown averages
        var avgPlan, avgEye, avgBrain, avgArm float64
        for _, r := range results {
                avgPlan += r.PlannerMs
                avgEye += r.EyeMs
                avgBrain += r.BrainMs
                avgArm += r.ArmMs
        }
        n := float64(len(results))
        avgPlan /= n
        avgEye /= n
        avgBrain /= n
        avgArm /= n

        t.Log("╠═══════════════════════════════════════════════════════════════════════════╣")
        t.Log("║  PHASE BREAKDOWN (averages)                                             ║")
        t.Logf("║    Planner:  %8.3f ms  (%4.1f%% of total)                          ║", avgPlan, avgPlan/avg*100)
        t.Logf("║    Eye:      %8.3f ms  (%4.1f%% of total)                          ║", avgEye, avgEye/avg*100)
        t.Logf("║    Brain:    %8.3f ms  (%4.1f%% of total)                          ║", avgBrain, avgBrain/avg*100)
        t.Logf("║    Arm:      %8.3f ms  (%4.1f%% of total)                          ║", avgArm, avgArm/avg*100)
        t.Log("╚═══════════════════════════════════════════════════════════════════════════╝")

        // ── Assertions ──
        assert.Equal(t, totalCycles, passCount, "ALL %d cycles must pass the %dms budget", totalCycles, targetBudgetMs)
        assert.Less(t, avg, 500.0, "Average cycle time should be well under 500ms (no LLM)")
        assert.Less(t, p95, float64(targetBudgetMs), "P95 must be under budget")
}

// =============================================================================
// STRESS TEST: EML Mathematical Precision Under Load
// =============================================================================
// Validates that EML operations maintain precision over millions of calls.

func TestStress_EML_PrecisionUnderLoad(t *testing.T) {
        iterations := 100_000
        tol := 1e-9

        type precisionTest struct {
                name     string
                fn       func() float64
                expected float64
        }

        tests := []precisionTest{
                {"Sin", func() float64 { return eml.Sin(0.785398163) }, 0.707106781},
                {"Cos", func() float64 { return eml.Cos(1.047197551) }, 0.5},  // Cos(π/3)
                {"Tan", func() float64 { return eml.Tan(0.785398163) }, 1.0},
                {"Exp", func() float64 { return eml.Exp(1.0) }, 2.718281828},
                {"Ln", func() float64 { return eml.Ln(2.718281828) }, 1.0},
                {"Sqrt", func() float64 { return eml.Sqrt(2.0) }, 1.414213562},
                {"Log2", func() float64 { return eml.Log2(1024.0) }, 10.0},
                {"Pow", func() float64 { return eml.Pow(2.0, 10.0) }, 1024.0},
                {"Tanh", func() float64 { return eml.Tanh(1.0) }, 0.761594156},
                {"SmoothStep", func() float64 { return eml.SmoothStep(0, 1, 0.5) }, 0.5},
                {"Distance2D", func() float64 { return eml.Distance2D(0, 0, 3, 4) }, 5.0},
        }

        t.Log("╔══════════════════════════════════════════════════════════════════╗")
        t.Log("║  EML PRECISION UNDER LOAD (100,000 iterations per function)     ║")
        t.Log("╠══════════════════════════════════════════════════════════════════╣")

        maxErr := 0.0
        for _, tc := range tests {
                start := time.Now()
                var lastErr float64
                for i := 0; i < iterations; i++ {
                        got := tc.fn()
                        err := abs(got - tc.expected)
                        if err > lastErr {
                                lastErr = err
                        }
                }
                elapsed := time.Since(start)
                avgNs := float64(elapsed.Nanoseconds()) / float64(iterations)

                if lastErr > maxErr {
                        maxErr = lastErr
                }

                status := "✅"
                if lastErr > tol {
                        status = "❌"
                }
                t.Logf("║  %-14s  MaxErr: %e  Avg: %7.1f ns/op  %s  ║",
                        tc.name, lastErr, avgNs, status)
        }
        t.Log("╚══════════════════════════════════════════════════════════════════╝")

        assert.Less(t, maxErr, tol, "Max error across all EML functions must be < %e", tol)
}

// =============================================================================
// STRESS TEST: Bezier Path Quality Under Extreme Conditions
// =============================================================================
func TestStress_Bezier_QualityExtremePaths(t *testing.T) {
        testCases := []struct {
                name  string
                start [2]float64
                end   [2]float64
                steps int
        }{
                {"FullHD diagonal", [2]float64{0, 0}, [2]float64{1920, 1080}, 12},
                {"4K diagonal", [2]float64{0, 0}, [2]float64{3840, 2160}, 12},
                {"Very short hop", [2]float64{960, 540}, [2]float64{965, 540}, 8},
                {"Horizontal sweep", [2]float64{0, 540}, [2]float64{1920, 540}, 12},
                {"Vertical sweep", [2]float64{960, 0}, [2]float64{960, 1080}, 12},
                {"Backward diagonal", [2]float64{1920, 1080}, [2]float64{0, 0}, 12},
                {"Near-same position", [2]float64{500, 500}, [2]float64{505, 500}, 8},
        }

        t.Log("╔══════════════════════════════════════════════════════════════════════╗")
        t.Log("║  BEZIER PATH QUALITY — Extreme Conditions                          ║")
        t.Log("╠══════════════════════════════════════════════════════════════════════╣")

        for _, tc := range testCases {
                path := eml.GenerateBezierPath(tc.start, tc.end, tc.steps)

                // Verify start and end points
                require.Equal(t, tc.steps+1, len(path), "%s: wrong point count", tc.name)

                totalDist := eml.Distance2D(tc.start[0], tc.start[1], tc.end[0], tc.end[1])
                tolerance := 10.0
                if totalDist < 10 {
                        tolerance = 20.0 // larger tolerance for tiny paths
                }
                startOk := path[0][0] >= tc.start[0]-tolerance && path[0][0] <= tc.start[0]+tolerance
                endOk := path[len(path)-1][0] >= tc.end[0]-tolerance && path[len(path)-1][0] <= tc.end[0]+tolerance

                // Check smoothness: max step distance should not exceed 40% of total distance
                maxAllowedStep := totalDist * 0.40
                if maxAllowedStep < 1 {
                        maxAllowedStep = 50 // allow larger steps for degenerate cases
                }

                maxStep := 0.0
                for i := 1; i < len(path); i++ {
                        d := eml.Distance2D(path[i-1][0], path[i-1][1], path[i][0], path[i][1])
                        if d > maxStep {
                                maxStep = d
                        }
                }

                startNs := time.Now()
                for i := 0; i < 10000; i++ {
                        eml.GenerateBezierPath(tc.start, tc.end, tc.steps)
                }
                avgNs := float64(time.Since(startNs).Nanoseconds()) / 10000

                status := "✅"
                if !startOk || !endOk || maxStep > maxAllowedStep {
                        status = "❌"
                }

                t.Logf("║  %-20s  pts=%2d  smooth=%7.1fpx  %7.1fns/path  %s  ║",
                        tc.name, len(path), maxStep, avgNs, status)

                assert.True(t, startOk, "%s: start point should be near origin", tc.name)
                assert.True(t, endOk, "%s: end point should be near target", tc.name)
                assert.LessOrEqual(t, maxStep, maxAllowedStep, "%s: step too large", tc.name)
        }
        t.Log("╚══════════════════════════════════════════════════════════════════════╝")
}

// =============================================================================
// STRESS TEST: Mixed Hardware Scenarios
// =============================================================================
func TestStress_MixedHardwareScenarios(t *testing.T) {
        scenarios := []struct {
                name       string
                imgW       int
                imgH       int
                grid       string
                fallback   bool
                maxAvgMs   float64
        }{
                {"4K Normal", 3840, 2160, "20x20", false, 200},
                {"4K Fallback", 3840, 2160, "12x12", true, 200},
                {"1080p Normal", 1920, 1080, "20x20", false, 200},
                {"1080p Fallback", 1920, 1080, "12x12", true, 200},
                {"720p Normal", 1280, 720, "20x20", false, 200},
                {"Tiny 640x480", 640, 480, "10x10", false, 200},
        }

        t.Log("╔════════════════════════════════════════════════════════════════════╗")
        t.Log("║  HARDWARE SCENARIOS — Mixed Configurations                        ║")
        t.Log("╠══════════════════════════════════════════════════════════════════════╣")

        for _, sc := range scenarios {
                cfg := &config.AppConfig{
                        GridDensity:         sc.grid,
                        PhantomPulseEnabled: true,
                        HardwareFallback:    sc.fallback,
                }

                img := generateRealisticImage(sc.imgW, sc.imgH)
                tm := &TimedMachine{CaptureW: img}
                pp := NewPhantomPulse(cfg, nil, tm)

                var rows, cols int
                if n, err := fmt.Sscanf(sc.grid, "%dx%d", &rows, &cols); err != nil || n != 2 || rows == 0 {
                        rows, cols = 20, 20
                }
                gridCfg := vision.GridConfig{Rows: rows, Cols: cols}

                // Run 20 cycles
                cycles := 20
                durations := make([]float64, cycles)

                for i := 0; i < cycles; i++ {
                        start := time.Now()
                        rawImg, _ := tm.Capture()
                        resized := pp.adaptiveResize(rawImg)
                        vision.DrawGrid(resized, gridCfg)
                        bounds := rawImg.Bounds()
                        px, py, _ := vision.MapLabelToPixel("A1", bounds, gridCfg)
                        tm.Click(px, py)
                        durations[i] = float64(time.Since(start)) / float64(time.Millisecond)
                }

                avgMs := avgFloat(durations)
                sort.Float64s(durations)
                p95Ms := percentile(durations, 95)

                status := "✅"
                if avgMs > sc.maxAvgMs {
                        status = "❌"
                }

                t.Logf("║  %-18s  Grid:%-6s  Avg:%6.2fms  P95:%6.2fms  %s  ║",
                        sc.name, sc.grid, avgMs, p95Ms, status)

                assert.Less(t, avgMs, sc.maxAvgMs,
                        "%s: avg %2fms exceeds max %2fms", sc.name, avgMs, sc.maxAvgMs)
        }
        t.Log("╚══════════════════════════════════════════════════════════════════════╝")
}

// ── Helper functions ──

func abs(x float64) float64 {
        if x < 0 {
                return -x
        }
        return x
}

func avgFloat(values []float64) float64 {
        if len(values) == 0 {
                return 0
        }
        sum := 0.0
        for _, v := range values {
                sum += v
        }
        return sum / float64(len(values))
}

func percentile(sorted []float64, p int) float64 {
        if len(sorted) == 0 {
                return 0
        }
        if len(sorted) == 1 {
                return sorted[0]
        }
        idx := int(float64(len(sorted)-1) * float64(p) / 100.0)
        if idx >= len(sorted) {
                idx = len(sorted) - 1
        }
        return sorted[idx]
}
