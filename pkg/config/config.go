package config

import (
	"encoding/json"
	"os"
)

// AppConfig represents the persistent settings of GhostOperator.
type AppConfig struct {
	OllamaEndpoint      string `json:"ollama_endpoint"`
	OllamaModel         string `json:"ollama_model"`
	GridDensity         string `json:"grid_density"`
	Hotkey              string `json:"hotkey"` // e.g., "Alt+G"
	PhantomPulseEnabled bool   `json:"phantom_pulse_enabled"`
	MaxOperationTimeMs  int    `json:"max_operation_time_ms"`
	HardwareFallback    bool   `json:"hardware_fallback"`
	FallbackAutoDetect  bool   `json:"fallback_auto_detect"`
	FallbackBudgetMs    int    `json:"fallback_budget_ms"`
	Theme               string `json:"theme"`
}

// Load reads the config file or returns defaults.
func Load() *AppConfig {
	defaultCfg := &AppConfig{
		OllamaEndpoint:      "http://127.0.0.1:11434",
		OllamaModel:         "moondream",
		GridDensity:         "20x20",
		Hotkey:              "Alt+G",
		PhantomPulseEnabled: true,
		MaxOperationTimeMs:  2500,
		HardwareFallback:    false,
		FallbackAutoDetect:  true,
		FallbackBudgetMs:    15000,
		Theme:               "dark",
	}

	data, err := os.ReadFile("config.json")
	if err != nil {
		return defaultCfg
	}

	var cfg AppConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return defaultCfg
	}
	return &cfg
}

// Save writes the config to disk.
func (c *AppConfig) Save() error {
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile("config.json", data, 0644)
}
