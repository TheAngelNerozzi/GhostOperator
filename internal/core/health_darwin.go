//go:build darwin

package core

import "runtime"

// CheckHealth verifies system resources on macOS.
func CheckHealth() HealthStatus {
	return HealthStatus{
		GPUAvailable: false,
		GPUType:      "CPU Optimization Mode (" + runtime.GOARCH + ")",
	}
}
