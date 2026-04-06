//go:build linux

package core

import "runtime"

// CheckHealth verifies system resources on Linux.
func CheckHealth() HealthStatus {
	return HealthStatus{
		GPUAvailable: false,
		GPUType:      "CPU Optimization Mode (" + runtime.GOARCH + ")",
	}
}
