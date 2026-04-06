package core

import (
	"context"
	"fmt"

	"github.com/TheAngelNerozzi/ghostoperator/internal/llm"
	"github.com/TheAngelNerozzi/ghostoperator/internal/machine"
	"github.com/TheAngelNerozzi/ghostoperator/pkg/config"
)

// Orchestrator manages complex automation missions.
type Orchestrator struct {
	Vision     *llm.VisionClient
	Machine    machine.Machine
	Config     *config.AppConfig
}

// ProcessMission starts a simple mission execution.
func (o *Orchestrator) ProcessMission(ctx context.Context, intent string, log func(string)) error {
	log("🛸 Iniciando misión: " + intent)

	pp := NewPhantomPulse(o.Config, o.Vision, o.Machine)

	// Attempt mission execution
	metrics, err := pp.Execute(ctx, intent, log)
	if err != nil {
		return fmt.Errorf("MISSION_FAILED: %w", err)
	}

	o.logMetrics(metrics, log)
	log("✅ Misión completada con éxito.")
	return nil
}

func (o *Orchestrator) logMetrics(m PulseMetrics, log func(string)) {
	log(fmt.Sprintf("⏱️ Ciclo completado en %dms (Ojo: %dms, Cerebro: %dms, Brazo: %dms)",
		m.TotalTime.Milliseconds(),
		m.CaptureTime.Milliseconds(),
		m.VisionTime.Milliseconds(),
		m.ActionTime.Milliseconds()))
}
