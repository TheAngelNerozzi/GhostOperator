//go:build darwin

package core

import (
        "encoding/binary"
        "fmt"
        "os"
        "os/exec"
        "runtime"
        "strconv"
        "strings"
        "syscall"
)

// CheckHealth verifies system resources on macOS.
func CheckHealth() HealthStatus {
        status := HealthStatus{
                GPUAvailable: false,
                GPUType:      "CPU Optimization Mode (" + runtime.GOARCH + ")",
        }

        // Total RAM via syscall
        totalRAM, err := getTotalRAM()
        if err != nil {
                fmt.Fprintf(os.Stderr, "health_darwin: failed to read total RAM: %v\n", err)
        } else {
                status.TotalRAM = totalRAM
        }

        // Free RAM via vm_stat command
        freeRAM, err := getFreeRAM()
        if err != nil {
                fmt.Fprintf(os.Stderr, "health_darwin: failed to read free RAM: %v\n", err)
        } else {
                status.FreeRAM = freeRAM
        }

        return status
}

// getTotalRAM reads hw.memsize via syscall (returns bytes).
func getTotalRAM() (uint64, error) {
        val, err := syscall.Sysctl("hw.memsize")
        if err != nil {
                return 0, err
        }
        // hw.memsize is returned as a uint64 in a byte slice
        if len(val) < 8 {
                return 0, fmt.Errorf("hw.memsize returned insufficient data")
        }
        // The Sysctl call for hw.memsize returns the value as a little-endian uint64.
        // In Go 1.26+, syscall.Sysctl returns a string, so we must cast to []byte.
        result := binary.LittleEndian.Uint64([]byte(val))
        return result, nil
}

// getFreeRAM runs vm_stat to compute free memory (returns bytes).
func getFreeRAM() (uint64, error) {
        out, err := exec.Command("/usr/bin/vm_stat").Output()
        if err != nil {
                return 0, err
        }

        var freePages, inactivePages uint64
        pageSize := uint64(syscall.Getpagesize())

        for _, line := range strings.Split(string(out), "\n") {
                parts := strings.SplitN(line, ":", 2)
                if len(parts) != 2 {
                        continue
                }
                key := strings.TrimSpace(parts[0])
                valStr := strings.TrimSpace(strings.TrimRight(parts[1], "."))

                val, err := strconv.ParseUint(valStr, 10, 64)
                if err != nil {
                        continue
                }

                switch key {
                case "Pages free":
                        freePages = val
                case "Pages inactive":
                        inactivePages = val
                }
        }

        // Available RAM = free pages + inactive pages (macOS can reclaim inactive pages)
        return (freePages + inactivePages) * pageSize, nil
}
