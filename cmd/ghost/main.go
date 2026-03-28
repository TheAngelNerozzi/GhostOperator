package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/TheAngelNerozzi/ghostoperator/internal/automation"
	"github.com/TheAngelNerozzi/ghostoperator/internal/core"
	"github.com/TheAngelNerozzi/ghostoperator/internal/input"
	"github.com/TheAngelNerozzi/ghostoperator/internal/llm"
	"github.com/TheAngelNerozzi/ghostoperator/internal/vision"
	"github.com/TheAngelNerozzi/ghostoperator/pkg/config"
	"github.com/kbinani/screenshot"
	"github.com/spf13/cobra"
)

// version is now a constant for v0.1.1 release integrity.
const version = "0.1.1"

var executor = &automation.ActionExecutor{}

var rootCmd = &cobra.Command{
	Use:     "ghost",
	Version: version,
	Short:   "GhostOperator (GO) - Open Source Visual Automation Agent",
	Long: `GhostOperator is a high-performance, local-first visual automation agent.
Powered by Grid Vision System™ for sub-pixel AI precision.`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			launchGUI()
			return
		}
		cmd.Help()
	},
}

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Correr GhostOperator en modo terminal (segundo plano)",
	Run: func(cmd *cobra.Command, args []string) {
		mission := "10x10"
		if len(args) > 0 {
			mission = args[0]
		}
		startAgent(mission, nil)

		fmt.Println("👻 GhostOperator activo. Presiona Ctrl+C para salir.")
		sigs := make(chan os.Signal, 1)
		signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
		<-sigs
	},
}

func startAgent(mission string, uiLog func(string)) {
	cfg := config.Load()

	// Auto-detect weak hardware and enable fallback mode
	if cfg.FallbackAutoDetect {
		profile := core.DetectHardwareProfile()
		if profile.IsWeak && !cfg.HardwareFallback {
			cfg.HardwareFallback = true
			effBudget := core.EffectiveBudgetMs(cfg.HardwareFallback, profile, cfg.FallbackBudgetMs)
			fmt.Printf("\033[1;33m[FALLBACK]\033[0m Hardware débil detectado (%s). Budget aumentado a %dms.\n",
				profile.Reason, effBudget)
		} else if !profile.IsWeak {
			fmt.Printf("\033[1;32m[HW]\033[0m Hardware OK (RAM: %.1fGB, CPUs: %d). Budget: %dms.\n",
				float64(profile.TotalRAMBytes)/(1024*1024*1024), profile.NumCPU, core.BudgetNormalMs)
		}
	}

	client, _ := llm.NewVisionClient(cfg.OllamaEndpoint, cfg.OllamaModel)

	orch := &core.Orchestrator{
		Vision:     client,
		Automation: executor,
		Config:     cfg,
	}

	// If mission is a density string (CLI legacy), it just listens for hotkey.
	// If it's a natural language mission, it executes immediately.
	if len(mission) < 6 && (mission == "10x10" || mission == "20x20") {
		input.ListenForHotkey(func() {
			RunAutomation(mission)
		}, func(err error) {
			fmt.Printf("\033[1;31m⚠️ Error: Alt+G ya está en uso por otra aplicación.\033[0m\n")
		})
		fmt.Printf("\033[1;36m[GHOST]\033[0m Modo Escucha Activo. Alt + G para capturar.\n")
	} else {
		err := orch.ProcessMission(context.Background(), mission, func(s string) {
			fmt.Printf("\033[1;32m[MISSION]\033[0m %s\n", s)
			if uiLog != nil {
				uiLog(s)
			}
		})
		if err != nil {
			fmt.Printf("❌ Error en misión: %v\n", err)
			if uiLog != nil {
				uiLog("Error: " + err.Error())
			}
		}
	}
}

// RunAutomation handles the Master Capture Flow with AI Reasoning
func RunAutomation(density string) {
	cfg := config.Load()
	input.SetDPIAware()

	fmt.Printf("\033[1;36m[EYE]\033[0m Capturando...\n")
	bounds := screenshot.GetDisplayBounds(0)
	rawImg, err := screenshot.CaptureRect(bounds)
	if err != nil {
		return
	}

	var rows, cols int
	fmt.Sscanf(density, "%dx%d", &rows, &cols)
	gridCfg := vision.GridConfig{Rows: rows, Cols: cols}
	gridData, err := vision.DrawGrid(rawImg, gridCfg)
	if err != nil {
		return
	}

	client, _ := llm.NewVisionClient(cfg.OllamaEndpoint, cfg.OllamaModel)
	targetLabel, err := client.Reason(context.Background(), gridData, "")
	if err != nil {
		fmt.Printf("❌ Error: %v\n", err)
		return
	}

	pixelX, pixelY, _ := vision.MapLabelToPixel(targetLabel, bounds, gridCfg)
	executor.Execute(automation.Command{
		Type: "CLICK",
		Params: map[string]interface{}{
			"x": float64(pixelX * 1000 / bounds.Dx()),
			"y": float64(pixelY * 1000 / bounds.Dy()),
		},
	})
}

func init() {
	rootCmd.AddCommand(startCmd)
}

func main() {
	// 1. Set DPI Awareness for Windows
	input.SetDPIAware()

	// 2. First-run: auto-bootstrap Ollama + Moondream
	if core.IsFirstRun() {
		fmt.Println("\n\033[1;37m  👻 Bienvenido a GhostOperator v" + version + "\033[0m")
		fmt.Println("  Configuración inicial del Motor de IA Local...")

		cfg := config.Load()

		// Ensure Ollama is serve-ready and we have the vision model
		if core.EnsureOllamaRunning() {
			core.EnsureModel(cfg.OllamaModel)
			core.MarkSetupDone()
		}
		fmt.Println()
	}

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

// PrintReady prints the final engine status for verification.
func PrintReady() {
	fmt.Printf("\033[1;32m[READY]\033[0m GhostOperator v%s Engine Loaded.\n", version)
}
