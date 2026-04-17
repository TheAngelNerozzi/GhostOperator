//go:build !linux && !darwin && !windows

package core

import "runtime"

// CheckHealth returns a basic health status for unsupported platforms.
// RAM fields will be zero since platform-specific detection is unavailable.
func CheckHealth() HealthStatus {
	return HealthStatus{
		TotalRAM:     0,
		FreeRAM:      0,
		GPUAvailable: false,
		GPUType:      "CPU Optimization Mode (" + runtime.GOARCH + ")",
	}
}
