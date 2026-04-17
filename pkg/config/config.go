package config

import (
        "encoding/json"
        "fmt"
        "os"
        "path/filepath"
)

// maxGridDimension is the maximum allowed rows/columns for grid density
// to prevent resource exhaustion from extremely large grids.
const maxGridDimension = 50

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
// OS-specific user config directory.
func getConfigPath() string {
        dir, err := os.UserConfigDir()
        if err != nil {
                fmt.Fprintf(os.Stderr, "Warning: cannot determine config directory: %v\n", err)
                return filepath.Join(os.TempDir(), "ghostoperator", "config.json")
        }
        return filepath.Join(dir, "ghostoperator", "config.json")
}

// ensureConfigDir creates the parent directory for the config file if it doesn't exist.
func ensureConfigDir(path string) error {
        dir := filepath.Dir(path)
        return os.MkdirAll(dir, 0700)
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
                if !os.IsNotExist(err) {
                        fmt.Fprintf(os.Stderr, "Warning: could not read config file %s: %v. Using defaults.\n", configPath, err)
                }
                return defaultCfg
        }

        var cfg AppConfig
        if err := json.Unmarshal(data, &cfg); err != nil {
                fmt.Printf("Warning: Error parsing config file %s: %v. Using defaults.\n", configPath, err)
                return defaultCfg
        }

        cfg.validate()
        return &cfg
}

// validate ensures config values are within safe ranges.
func (c *AppConfig) validate() {
        if c.MaxOperationTimeMs <= 0 {
                c.MaxOperationTimeMs = 2500
        }
        if c.FallbackBudgetMs <= 0 {
                c.FallbackBudgetMs = 15000
        }
        if c.OllamaEndpoint == "" {
                c.OllamaEndpoint = "http://127.0.0.1:11434"
        }
        if c.OllamaModel == "" {
                c.OllamaModel = "moondream"
        }
        if c.GridDensity == "" {
                c.GridDensity = "20x20"
        } else {
                // Validate format and enforce upper bounds to prevent resource exhaustion
                var rows, cols int
                if n, _ := fmt.Sscanf(c.GridDensity, "%dx%d", &rows, &cols); n != 2 || rows < 1 || cols < 1 || rows > maxGridDimension || cols > maxGridDimension {
                        c.GridDensity = "20x20"
                }
        }
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
        return os.WriteFile(configPath, data, 0600)
}
