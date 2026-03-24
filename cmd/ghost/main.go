package main

import (
	"fmt"
	"image"
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

var executor = &action.ActionExecutor{}

var rootCmd = &cobra.Command{
	Use:   "ghost",
	Short: "GhostOperator (GO) - Open Source Visual Automation Agent",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("👻 GhostOperator is starting...")
		
		status := brain.CheckHealth()
		fmt.Printf("Health Check: %s | GPU: %v\n", status.GPUType, status.GPUAvailable)

		// 1. Watch for Hotkey
		input.ListenForHotkey(func() {
			fmt.Println("🚀 Hotkey Triggered! Capturing Screen with Grid...")
			
			// 2. Capture Primary Screen
			bounds := screenshot.GetDisplayBounds(0)
			rawImg, err := screenshot.CaptureRect(bounds)
			if err != nil {
				fmt.Printf("Capture Error: %v\n", err)
				return
			}

			// 3. Apply Grid Vision System (10x10)
			config := screen.GridConfig{Rows: 10, Cols: 10}
			gridJPG, err := screen.DrawGrid(rawImg, config)
			if err != nil {
				fmt.Printf("Grid Error: %v\n", err)
				return
			}

			fmt.Printf("Grid Overlay Generated (%d bytes). Ready for LMM.\n", len(gridJPG))

			// 4. Mock AI Response (User sees 'Chrome' at B2)
			targetLabel := "B2"
			pixelX, pixelY, _ := screen.MapLabelToPixel(targetLabel, bounds, config)
			fmt.Printf("LMM Selected Grid Cell: %s -> Target Pixels: (%d, %d)\n", targetLabel, pixelX, pixelY)

			// 5. Execute Action via Protocol
			mockCmd := action.Command{
				Type: "CLICK",
				Params: map[string]interface{}{"x": float64(pixelX*1000/bounds.Dx()), "y": float64(pixelY*1000/bounds.Dy())},
			}
			executor.Execute(mockCmd)
		})

		fmt.Println("Press Ctrl+C to stop.")
		sigs := make(chan os.Signal, 1)
		signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
		<-sigs
	},
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
