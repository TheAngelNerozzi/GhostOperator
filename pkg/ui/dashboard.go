package ui

import (
	"fmt"
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

// AppState represents the current state of the GUI.
type AppState struct {
	IsActive    bool
	GridDensity string
}

// ShowDashboard launches the Fyne window.
func ShowDashboard(version string, onStart func(density string)) {
	myApp := app.New()
	myWindow := myApp.NewWindow("GhostOperator GO - " + version)
	myWindow.Resize(fyne.NewSize(400, 300))

	state := AppState{IsActive: false, GridDensity: "10x10"}

	// 1. Status Indicator
	statusCircle := canvas.NewCircle(color.RGBA{255, 0, 0, 255})
	statusCircle.Resize(fyne.NewSize(20, 20))
	statusLabel := widget.NewLabel("🔴 Offline")

	statusBox := container.NewHBox(statusCircle, statusLabel)

	// 2. Grid Selector
	gridSelect := widget.NewSelect([]string{"10x10", "20x20", "50x50"}, func(value string) {
		fmt.Println("Grid density set to:", value)
		state.GridDensity = value
	})
	gridSelect.SetSelected("10x10")

	// 3. Action Button
	btn := widget.NewButton("Activar Agente", func() {
		state.IsActive = !state.IsActive
		if state.IsActive {
			statusCircle.FillColor = color.RGBA{0, 255, 0, 255}
			statusLabel.SetText("🟢 Online")
			statusCircle.Refresh()
			fmt.Println("Agente Activado con rejilla:", state.GridDensity)
			onStart(state.GridDensity, func(err error) {
				statusCircle.FillColor = color.RGBA{255, 165, 0, 255} // Orange for warning
				statusLabel.SetText("⚠️ Hotkey Busy")
				statusCircle.Refresh()
				dialog.ShowError(err, myWindow)
			})
		} else {
			statusCircle.FillColor = color.RGBA{255, 0, 0, 255}
			statusLabel.SetText("🔴 Offline")
			statusCircle.Refresh()
			fmt.Println("Agente Desactivado")
		}
	})
	btn.Importance = widget.HighImportance

	// 4. Layout
	content := container.NewVBox(
		widget.NewLabelWithStyle("GhostOperator Dashboard", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		widget.NewSeparator(),
		statusBox,
		widget.NewLabel("Select Grid Density:"),
		gridSelect,
		container.NewPadded(btn),
		widget.NewLabelWithStyle("Powered by Grid Vision System™", fyne.TextAlignCenter, fyne.TextStyle{Italic: true}),
	)

	myWindow.SetContent(content)
	myWindow.ShowAndRun()
}
