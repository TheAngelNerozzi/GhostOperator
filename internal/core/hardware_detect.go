package core

import (
	"fmt"
	"runtime"
)

// HardwareProfile contains the detected hardware capabilities.
type HardwareProfile struct {
	TotalRAMBytes uint64 `json:"total_ram_bytes"`
	FreeRAMBytes  uint64 `json:"free_ram_bytes"`
	NumCPU        int    `json:"num_cpu"`
	IsWeak        bool   `json:"is_weak"`
	Reason        string `json:"reason"`
}

// Thresholds for weak hardware detection
const (
	// Minimum 4GB total RAM for normal mode
	minRAMNormalBytes = 4 * 1024 * 1024 * 1024 // 4 GB
	// Minimum 1GB free RAM for normal mode
	minFreeRAMNormalBytes = 1 * 1024 * 1024 * 1024 // 1 GB
	// Minimum 4 logical CPUs for normal mode
	minCPUNormal = 4

	// Budget constants (milliseconds)
	BudgetNormalMs   = 2500  // 2.5s — normal hardware
	BudgetFallbackMs = 15000 // 15.0s — weak hardware fallback
)

// DetectHardwareProfile probes the system and returns a profile indicating
// whether the machine qualifies as "weak" hardware that needs the fallback budget.
func DetectHardwareProfile() HardwareProfile {
	health := CheckHealth()
	numCPU := runtime.NumCPU()

	profile := HardwareProfile{
		TotalRAMBytes: health.TotalRAM,
		FreeRAMBytes:  health.FreeRAM,
		NumCPU:        numCPU,
		IsWeak:        false,
	}

	// Decision tree for weak hardware
	reasons := []string{}

	if health.TotalRAM > 0 && health.TotalRAM < minRAMNormalBytes {
		reasons = append(reasons, fmt.Sprintf("RAM total %.1fGB < 4GB", float64(health.TotalRAM)/(1024*1024*1024)))
	}

	if health.FreeRAM > 0 && health.FreeRAM < minFreeRAMNormalBytes {
		reasons = append(reasons, fmt.Sprintf("RAM libre %.1fGB < 1GB", float64(health.FreeRAM)/(1024*1024*1024)))
	}

	if numCPU < minCPUNormal {
		reasons = append(reasons, fmt.Sprintf("CPU cores %d < %d", numCPU, minCPUNormal))
	}

	if len(reasons) > 0 {
		profile.IsWeak = true
		profile.Reason = reasons[0]
		for i := 1; i < len(reasons); i++ {
			profile.Reason += "; " + reasons[i]
		}
	}

	return profile
}

// EffectiveBudgetMs returns the operation budget in milliseconds based on the
// hardware profile. If fallback is forced via config, it always returns the
// fallback budget regardless of hardware. When configBudgetMs > 0, it uses
// that value instead of the hardcoded BudgetFallbackMs constant.
func EffectiveBudgetMs(configFallback bool, profile HardwareProfile, configBudgetMs int) int {
	if configFallback || profile.IsWeak {
		if configBudgetMs > 0 {
			return configBudgetMs
		}
		return BudgetFallbackMs
	}
	return BudgetNormalMs
}
