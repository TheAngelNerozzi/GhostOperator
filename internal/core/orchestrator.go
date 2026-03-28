package core

import (
	"context"
	"fmt"
	"time"

	"github.com/TheAngelNerozzi/ghostoperator/internal/automation"
	"github.com/TheAngelNerozzi/ghostoperator/internal/llm"
	"github.com/TheAngelNerozzi/ghostoperator/pkg/config"
)

// Orchestrator manages complex automation missions.
type Orchestrator struct {
	Vision     *llm.VisionClient
	Automation *automation.ActionExecutor
	Config     *config.AppConfig
}

// ProcessMission starts an autonomous loop until the user's intent is fulfilled.
func (o *Orchestrator) ProcessMission(ctx context.Context, intent string, log func(string)) error {
	log("🛸 Iniciando misión: " + intent)

	// Create PhantomPulse engine for optimized execution
	pp := NewPhantomPulse(o.Config, o.Vision, o.Automation)

	// Report active mode to the user
	if pp.FallbackActive() {
		log(fmt.Sprintf("⚠️ Modo Fallback activado: budget %dms (hardware débil detectado: %s)",
			pp.ActiveBudgetMs, pp.Profile.Reason))
	} else {
		log(fmt.Sprintf("🚀 PhantomPulse™ normal: budget %dms", pp.ActiveBudgetMs))
	}

	// Execute single-step mission via PhantomPulse pipeline.
	// TODO(v2): implement multi-step evaluator loop for complex, sequential missions.
	log("⚡️ Paso 1/1: PhantomPulse pipeline activo...")

	metrics, err := pp.Execute(ctx, intent, log)
	if err != nil {
		return fmt.Errorf("fallo de ciclo PhantomPulse: %w", err)
	}

	totalMs := metrics.TotalTime.Milliseconds()
	log(fmt.Sprintf("⏱️ Ciclo completado en %dms (Ojo: %dms, Cerebro: %dms, Brazo: %dms)",
		totalMs,
		metrics.CaptureTime.Milliseconds(),
		metrics.VisionTime.Milliseconds(),
		metrics.ActionTime.Milliseconds()))

	log("✅ Acción ejecutada. Esperando reacción del sistema...")

	// Adaptive post-execution delay: give weak hardware breathing room
	if pp.FallbackActive() {
		time.Sleep(800 * time.Millisecond)
	} else {
		time.Sleep(200 * time.Millisecond)
	}

	return nil
}
