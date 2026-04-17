package core

// HealthStatus represents the system's readiness for local models.
type HealthStatus struct {
	GPUAvailable bool   `json:"gpu_available"`
	GPUType      string `json:"gpu_type"`
	TotalRAM     uint64 `json:"total_ram"`
	FreeRAM      uint64 `json:"free_ram"`
}
