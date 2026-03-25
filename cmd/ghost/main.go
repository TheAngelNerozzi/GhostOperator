package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/TheAngelNerozzi/ghostoperator/pkg/action"
	"github.com/TheAngelNerozzi/ghostoperator/pkg/brain"
	"github.com/TheAngelNerozzi/ghostoperator/pkg/input"
	"github.com/TheAngelNerozzi/ghostoperator/pkg/screen"
	"github.com/spf13/cobra"
	"github.com/kbinani/screenshot"
)

// version is now a constant for v0.1.1 release integrity.
const version = "0.1.1"
var executor = &action.ActionExecutor{}

var rootCmd = &cobra.Command{
	Use:     "ghost",
	Version: version,
	Short:   "GhostOperator (GO) - Open Source Visual Automation Agent",
	Long: `GhostOperator is a high-performance, local-first visual automation agent.
Powered by Grid Vision System™ for sub-pixel AI precision.`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			fmt.Println("\033[1;33m[INFO]\033[0m GhostOperator GUI richiede CGO (GCC).")
			fmt.Println("       Questa versione è compilata in modalita 'CGO-FREE' (Solo CLI).")
			fmt.Println("       Per attivare l'agente usa: \033[1;32m./ghost_new.exe start\033[0m")
			return
		}
		cmd.Help()
	},
}

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Correr GhostOperator en modo terminal (segundo plano)",
	Run: func(cmd *cobra.Command, args []string) {
		startAgent("10x10", nil)

		fmt.Println("👻 GhostOperator activo. Presiona Ctrl+C para salir.")
		sigs := make(chan os.Signal, 1)
		signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
		<-sigs
	},
}

func startAgent(density string, uiErrorHandler func(error)) {
	// Robust Hotkey Registration
	input.ListenForHotkey(func() {
		RunAutomation(density)
	}, func(err error) {
		errorMessage := fmt.Sprintf("\n⚠️ Error: Alt+Space ya está en uso por otra aplicación.\n")
		fmt.Printf("\033[1;31m%s\033[0m", errorMessage)
		if uiErrorHandler != nil {
			uiErrorHandler(err)
		}
	})

	fmt.Printf("\033[1;36m[GHOST]\033[0m Motor v%s iniciado. Densidad: %s\n", version, density)
	
	status := brain.CheckHealth()
	fmt.Printf("\033[1;34m[BRAIN]\033[0m Health Check: %s | GPU: %v\n", status.GPUType, status.GPUAvailable)

	fmt.Println("👻 Esperando Alt + Espacio para la captura...")
}

// RunAutomation handles the Master Capture Flow
func RunAutomation(density string) {
	// 1. DPI Awareness (Force hardware resolution)
	input.SetDPIAware()

	// 2. [EYE] Capture Screen
	fmt.Printf("\033[1;36m[EYE]\033[0m Capturando pantalla...\n")
	bounds := screenshot.GetDisplayBounds(0)
	rawImg, err := screenshot.CaptureRect(bounds)
	if err != nil {
		fmt.Printf("❌ Error de captura: %v\n", err)
		return
	}

	// 3. [GRID] Overlay & Save
	var rows, cols int
	fmt.Sscanf(density, "%dx%d", &rows, &cols)
	config := screen.GridConfig{Rows: rows, Cols: cols}

	gridData, err := screen.DrawGrid(rawImg, config)
	if err != nil {
		fmt.Printf("❌ Error de rejilla: %v\n", err)
		return
	}

	// Forzado de debug_view.png
	if err := os.WriteFile("debug_view.png", gridData, 0644); err != nil {
		fmt.Printf("❌ Error al guardar debug_view.png: %v\n", err)
	} else {
		fmt.Printf("\033[1;32m[GRID]\033[0m Rejilla aplicada. Archivo: debug_view.png\n")
	}

	// 4. [BRAIN] Decision (Simulated)
	targetLabel := "B2"
	pixelX, pixelY, _ := screen.MapLabelToPixel(targetLabel, bounds, config)
	fmt.Printf("\033[1;33m[BRAIN]\033[0m Objetivo detectado en %s -> (%d, %d)\n", targetLabel, pixelX, pixelY)

	// 5. [ARM] Action execution
	fmt.Printf("\033[1;35m[ARM]\033[0m Ejecutando Click en visual...\n")
	mockCmd := action.Command{
		Type: "CLICK",
		Params: map[string]interface{}{
			"x": float64(pixelX * 1000 / bounds.Dx()),
			"y": float64(pixelY * 1000 / bounds.Dy()),
		},
	}
	executor.Execute(mockCmd)
}

func init() {
	rootCmd.AddCommand(startCmd)
}

func main() {
	// 1. Set DPI Awareness for Windows
	input.SetDPIAware()

	// 2. Inject Version into Cobra
	rootCmd.Version = version

	// 3. UAT Readiness Check
	PrintReady()

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// PrintReady prints the final engine status for verification.
func PrintReady() {
	fmt.Printf("\033[1;32m[READY]\033[0m GhostOperator v%s Engine Loaded.\n", version)
}
