package main

import (
	"github.com/TheAngelNerozzi/ghostoperator/pkg/config"
	"github.com/TheAngelNerozzi/ghostoperator/pkg/ui"
)

func launchGUI() {
	cfg := config.Load()
	ui.ShowDashboard(version, cfg, func(mission string, uiLog func(string)) {
		startAgent(mission, uiLog)
	})
}
