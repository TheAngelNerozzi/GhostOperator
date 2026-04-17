//go:build linux

package core

import (
	"bufio"
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"
)

// CheckHealth verifies system resources on Linux.
func CheckHealth() HealthStatus {
	status := HealthStatus{
		GPUAvailable: false,
		GPUType:      "CPU Optimization Mode (" + runtime.GOARCH + ")",
	}

	totalRAM, freeRAM, err := readProcMeminfo()
	if err != nil {
		fmt.Fprintf(os.Stderr, "health_linux: failed to read /proc/meminfo: %v\n", err)
	} else {
		status.TotalRAM = totalRAM
		status.FreeRAM = freeRAM
	}

	return status
}

// readProcMeminfo reads TotalRAM and FreeRAM from /proc/meminfo.
// Values in /proc/meminfo are in kB; we convert to bytes.
func readProcMeminfo() (totalRAM, freeRAM uint64, err error) {
	f, err := os.Open("/proc/meminfo")
	if err != nil {
		return 0, 0, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}

		val, parseErr := strconv.ParseUint(parts[1], 10, 64)
		if parseErr != nil {
			continue
		}

		switch parts[0] {
		case "MemTotal:":
			totalRAM = val * 1024 // kB to bytes
		case "MemAvailable:":
			freeRAM = val * 1024 // kB to bytes
		}
	}

	if err := scanner.Err(); err != nil {
		return 0, 0, err
	}

	return totalRAM, freeRAM, nil
}
