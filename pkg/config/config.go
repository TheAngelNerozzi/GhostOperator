package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
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

// getConfigPath returns the full path to the config file, using the
// OS-specific user config directory (e.g. ~/.config/ghostoperator/config.json
// on Linux, %AppData%/ghostoperator/config.json on Windows).
func getConfigPath() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		// Fallback to CWD if UserConfigDir fails
		return "config.json"
	}
	return filepath.Join(dir, "ghostoperator", "config.json")
}

// ensureConfigDir creates the parent directory for the config file if it doesn't exist.
func ensureConfigDir(path string) error {
	dir := filepath.Dir(path)
	return os.MkdirAll(dir, 0755)
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

	configPath := getConfigPath()
	data, err := os.ReadFile(configPath)
	if err != nil {
		return defaultCfg
	}

	var cfg AppConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		fmt.Printf("⚠️ Error parsing config file %s: %v. Using defaults.\n", configPath, err)
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
	configPath := getConfigPath()
	if err := ensureConfigDir(configPath); err != nil {
		return fmt.Errorf("cannot create config directory: %w", err)
	}
	return os.WriteFile(configPath, data, 0644)
}
