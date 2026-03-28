package core

import (
	"golang.org/x/sys/windows"
	"runtime"
	"unsafe"
)

type memoryStatusEx struct {
	dwLength                uint32
	dwMemoryLoad            uint32
	ullTotalPhys            uint64
	ullAvailPhys            uint64
	ullTotalPageFile        uint64
	ullAvailPageFile        uint64
	ullTotalVirtual         uint64
	ullAvailVirtual         uint64
	ullAvailExtendedVirtual uint64
}

var (
	kernel32                 = windows.NewLazySystemDLL("kernel32.dll")
	procGlobalMemoryStatusEx = kernel32.NewProc("GlobalMemoryStatusEx")
)

// HealthStatus represents the system's readiness for local models.
type HealthStatus struct {
	GPUAvailable bool   `json:"gpu_available"`
	GPUType      string `json:"gpu_type"`
	TotalRAM     uint64 `json:"total_ram"`
	FreeRAM      uint64 `json:"free_ram"`
}

// CheckHealth verifies system resources.
func CheckHealth() HealthStatus {
	status := HealthStatus{
		GPUAvailable: false,
		GPUType:      "CPU Optimization Mode (" + runtime.GOARCH + ")",
	}

	var memInfo memoryStatusEx
	memInfo.dwLength = uint32(unsafe.Sizeof(memInfo))
	ret, _, _ := procGlobalMemoryStatusEx.Call(uintptr(unsafe.Pointer(&memInfo)))

	if ret != 0 {
		status.TotalRAM = memInfo.ullTotalPhys
		status.FreeRAM = memInfo.ullAvailPhys
	}

	return status
}
