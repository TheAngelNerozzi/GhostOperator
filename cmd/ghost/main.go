package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/TheAngelNerozzi/ghostoperator/internal/core"
	"github.com/TheAngelNerozzi/ghostoperator/internal/input"
	"github.com/TheAngelNerozzi/ghostoperator/internal/llm"
	"github.com/TheAngelNerozzi/ghostoperator/internal/machine"
	"github.com/TheAngelNerozzi/ghostoperator/pkg/config"
	"github.com/TheAngelNerozzi/ghostoperator/pkg/ui"
	"github.com/spf13/cobra"
)

const version = "1.1.0"

var m machine.Machine = machine.NewNativeMachine()

var rootCmd = &cobra.Command{
	Use:     "ghost",
	Version: version,
	Short:   "GhostOperator (GO) - Visual Automation Agent",
	Long: `GhostOperator is a high-performance, local-first visual automation agent.
Powered by Grid Vision System™ and Ollama for sub-pixel AI precision.`,
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
	Short: "Correr GhostOperator en modo terminal",
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
			fmt.Printf("\033[1;33m[FALLBACK]\033[0m Hardware débil detectado. Budget aumentado a %dms.\n", cfg.FallbackBudgetMs)
		}
	}

	client, _ := llm.NewVisionClient(cfg.OllamaEndpoint, cfg.OllamaModel)

	orch := &core.Orchestrator{
		Vision:  client,
		Machine: m,
		Config:  cfg,
	}

	err := orch.ProcessMission(context.Background(), mission, func(s string) {
		fmt.Printf("\033[1;32m[MISSION]\033[0m %s\n", s)
		if uiLog != nil {
			uiLog(s)
		}
	})
	if err != nil {
		fmt.Printf("❌ Error en misión: %v\n", err)
	}
}

func launchGUI() {
	cfg := config.Load()
	ui.ShowDashboard(version, cfg, m, startAgent)
}

func init() {
	rootCmd.AddCommand(startCmd)
}

func main() {
	// Set DPI Awareness for Windows
	input.SetDPIAware()

	// 1. Check for Ollama + Moondream model
	if core.IsFirstRun() {
		fmt.Println("\n\033[1;37m  👻 Bienvenido a GhostOperator v" + version + "\033[0m")
		fmt.Println("  Configuración inicial de IA Local...")

		if core.EnsureOllamaRunning() {
			cfg := config.Load()
			core.EnsureModel(cfg.OllamaModel)
			core.MarkSetupDone()
		}
		fmt.Println()
	}

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
