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
	"github.com/TheAngelNerozzi/ghostoperator/pkg/ui"
	"github.com/spf13/cobra"
	"github.com/kbinani/screenshot"
)

// version is injected at build time using ldflags
var version = "dev"
var executor = &action.ActionExecutor{}

var rootCmd = &cobra.Command{
	Use:     "ghost",
	Version: version,
	Short:   "GhostOperator (GO) - Open Source Visual Automation Agent",
	Long: `GhostOperator is a high-performance, local-first visual automation agent.
Powered by Grid Vision System™ for sub-pixel AI precision.`,
	Run: func(cmd *cobra.Command, args []string) {
		// If no arguments, launch GUI
		if len(args) == 0 {
			fmt.Println("Launching GhostOperator GUI...")
			ui.ShowDashboard(version, func(density string) {
				startAgent(density)
			})
			return
		}
		cmd.Help()
	},
}

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Activate the GhostOperator agent in background mode",
	Run: func(cmd *cobra.Command, args []string) {
		startAgent("10x10")
		
		fmt.Println("Press Ctrl+C to stop.")
		sigs := make(chan os.Signal, 1)
		signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
		<-sigs
	},
}

func startAgent(density string) {
	fmt.Printf("👻 GhostOperator activado [%s]. Presiona Alt + Espacio para capturar...\n", density)
	
	status := brain.CheckHealth()
	fmt.Printf("Health Check: %s | GPU: %v\n", status.GPUType, status.GPUAvailable)

	input.ListenForHotkey(func() {
		fmt.Printf("🚀 Hotkey Triggered! Capturing Screen with Grid %s...\n", density)
		bounds := screenshot.GetDisplayBounds(0)
		rawImg, err := screenshot.CaptureRect(bounds)
		if err != nil {
			fmt.Printf("Capture Error: %v\n", err)
			return
		}

		// Parse density (e.g., "20x20")
		var rows, cols int
		fmt.Sscanf(density, "%dx%d", &rows, &cols)
		config := screen.GridConfig{Rows: rows, Cols: cols}
		
		gridJPG, err := screen.DrawGrid(rawImg, config)
		if err != nil {
			fmt.Printf("Grid Error: %v\n", err)
			return
		}

		fmt.Printf("Grid Overlay Generated (%d bytes). Ready for LMM.\n", len(gridJPG))

		// Mock LMM logic
		targetLabel := "B2"
		pixelX, pixelY, _ := screen.MapLabelToPixel(targetLabel, bounds, config)
		fmt.Printf("LMM Selected Grid Cell: %s -> Target Pixels: (%d, %d)\n", targetLabel, pixelX, pixelY)

		mockCmd := action.Command{
			Type: "CLICK",
			Params: map[string]interface{}{
				"x": float64(pixelX * 1000 / bounds.Dx()),
				"y": float64(pixelY * 1000 / bounds.Dy()),
			},
		}
		executor.Execute(mockCmd)
	})
}

func init() {
	rootCmd.AddCommand(startCmd)
}

func main() {
	// Sync rootCmd.Version with injected variable
	rootCmd.Version = version
	
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
