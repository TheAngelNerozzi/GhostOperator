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

var executor = &action.ActionExecutor{}

var rootCmd = &cobra.Command{
	Use:     "ghost",
	Version: "0.1.0",
	Short:   "GhostOperator (GO) - Open Source Visual Automation Agent",
	Long: `
  ________  ___  ___  ________  ________  _________   
 |\   ____\|\  \|\  \|\   __  \|\   ____\|\___  ___\ 
 \ \  \___|\ \  \\\  \ \  \|\  \ \  \___|\|___ \  \_| 
  \ \  \  __\ \   __  \ \  \\\  \ \_____  \   \ \  \  
   \ \  \|\  \ \  \ \  \ \  \\\  \|____|\  \   \ \  \ 
    \ \_______\ \__\ \__\ \_______\____\_\  \   \ \__\
     \|_______|\|__|\|__|\|_______|\|_________|   \|__|
                                   \|_________|        
                                                       
GhostOperator is a high-performance, local-first visual automation agent.
Powered by Grid Vision System™ for sub-pixel AI precision.`,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Activate the GhostOperator agent",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("👻 GhostOperator activado. Presiona Alt + Espacio para capturar...")

		status := brain.CheckHealth()
		fmt.Printf("Health Check: %s | GPU: %v\n", status.GPUType, status.GPUAvailable)

		input.ListenForHotkey(func() {
			fmt.Println("🚀 Hotkey Triggered! Capturing Screen with Grid...")
			bounds := screenshot.GetDisplayBounds(0)
			rawImg, err := screenshot.CaptureRect(bounds)
			if err != nil {
				fmt.Printf("Capture Error: %v\n", err)
				return
			}

			config := screen.GridConfig{Rows: 10, Cols: 10}
			gridJPG, err := screen.DrawGrid(rawImg, config)
			if err != nil {
				fmt.Printf("Grid Error: %v\n", err)
				return
			}

			fmt.Printf("Grid Overlay Generated (%d bytes). Ready for LMM.\n", len(gridJPG))

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

		fmt.Println("Press Ctrl+C to stop.")
		sigs := make(chan os.Signal, 1)
		signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
		<-sigs
	},
}

func init() {
	rootCmd.AddCommand(startCmd)
	cobra.AddTemplateFunc("styleHeading", func(s string) string {
		return fmt.Sprintf("\033[1;36m%s\033[0m", s)
	})

	rootCmd.SetHelpTemplate(`{{with (or .Long .Short)}}{{. | trimTrailingWhitespaces}}

{{end}}{{styleHeading "USAGE:"}}
  {{.UseLine}}

{{styleHeading "COMMANDS:"}}{{range .Commands}}{{if (or .IsAvailableCommand (eq .Name "help"))}}
  {{rpad .Name .NamePadding}} {{.Short}}{{end}}{{end}}

{{styleHeading "FLAGS:"}}
{{.LocalFlags.FlagUsages | trimTrailingWhitespaces}}

{{styleHeading "LEARN MORE:"}}
  Visit https://github.com/TheAngelNerozzi/GhostOperator
`)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
