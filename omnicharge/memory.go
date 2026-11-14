package omnicharge

import (
	"runtime"
	"runtime/debug"
)

// MemoryStats holds snapshot metrics of heap and system memory usage.
type MemoryStats struct {
	AllocMB      uint64
	TotalAllocMB uint64
	SysMB        uint64
	NumGC        uint32
}

// MemoryGovernor manages system memory allocation and GC hints for Termux.
type MemoryGovernor struct{}

// NewMemoryGovernor creates a new memory governor.
func NewMemoryGovernor() *MemoryGovernor {
	return &MemoryGovernor{}
}

// Stats returns current memory statistics converted to Megabytes.
func (mg *MemoryGovernor) Stats() MemoryStats {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	const mb = 1024 * 1024
	return MemoryStats{
		AllocMB:      m.Alloc / mb,
		TotalAllocMB: m.TotalAlloc / mb,
		SysMB:        m.Sys / mb,
		NumGC:        m.NumGC,
	}
}

// FreeMemory requests a garbage collection pass and returns memory to the OS.
// Highly recommended on Termux after large batch downloads or high-throughput streams.
func (mg *MemoryGovernor) FreeMemory() {
	runtime.GC()
	debug.FreeOSMemory()
}
