package omnicharge

import (
	"os"
	"runtime"
	"runtime/debug"
	"strings"
	"sync"
	"time"
)

// Platform represents the detected runtime environment.
type Platform string

const (
	PlatformAndroid Platform = "android"
	PlatformWSL2    Platform = "wsl2"
	PlatformLinux   Platform = "linux"
	PlatformWindows Platform = "windows"
	PlatformMacOS   Platform = "darwin"
	PlatformUnknown Platform = "unknown"
)

// MemoryStats holds snapshot metrics of heap and system memory usage.
type MemoryStats struct {
	Platform     Platform
	SoftLimitMB  int64
	AllocMB      float64
	TotalAllocMB float64
	SysMB        float64
	HeapIdleMB   float64
	NumGC        uint32
}

// GovernorConfig defines optional tuning parameters for MemoryGovernor.
type GovernorConfig struct {
	SoftLimitMB  int64
	FreeCooldown time.Duration
}

// MemoryGovernor manages cross-platform memory allocation, limits, and GC hints.
type MemoryGovernor struct {
	mu           sync.Mutex
	platform     Platform
	lastFreed    time.Time
	freeCooldown time.Duration
	softLimitMB  int64
}

// DetectPlatform identifies the host OS, distinguishing WSL2 from native Linux.
func DetectPlatform() Platform {
	switch runtime.GOOS {
	case "android":
		return PlatformAndroid
	case "windows":
		return PlatformWindows
	case "darwin":
		return PlatformMacOS
	case "linux":
		if isWSL() {
			return PlatformWSL2
		}
		return PlatformLinux
	default:
		return PlatformUnknown
	}
}

func isWSL() bool {
	if os.Getenv("WSL_DISTRO_NAME") != "" || os.Getenv("WSL_INTEROP") != "" {
		return true
	}
	data, err := os.ReadFile("/proc/version")
	if err == nil {
		version := strings.ToLower(string(data))
		if strings.Contains(version, "microsoft") || strings.Contains(version, "wsl") {
			return true
		}
	}
	return false
}

func defaultSoftLimitForPlatform(p Platform) int64 {
	switch p {
	case PlatformAndroid:
		return 256
	case PlatformWSL2:
		return 1024
	case PlatformLinux, PlatformWindows, PlatformMacOS:
		return 2048
	default:
		return 512
	}
}

// NewMemoryGovernor initializes a MemoryGovernor with automatic platform tuning.
func NewMemoryGovernor(cfg ...GovernorConfig) *MemoryGovernor {
	var config GovernorConfig
	if len(cfg) > 0 {
		config = cfg[0]
	}

	platform := DetectPlatform()

	if config.FreeCooldown <= 0 {
		if platform == PlatformAndroid || platform == PlatformWSL2 {
			config.FreeCooldown = 5 * time.Second
		} else {
			config.FreeCooldown = 15 * time.Second
		}
	}

	limitMB := config.SoftLimitMB
	if limitMB == 0 {
		limitMB = defaultSoftLimitForPlatform(platform)
	} else if limitMB < 0 {
		limitMB = 0
	}

	mg := &MemoryGovernor{
		platform:     platform,
		freeCooldown: config.FreeCooldown,
		softLimitMB:  limitMB,
	}

	if limitMB > 0 {
		const mb = 1024 * 1024
		debug.SetMemoryLimit(limitMB * mb)
	}

	return mg
}

// Platform returns the detected runtime environment.
func (mg *MemoryGovernor) Platform() Platform {
	return mg.platform
}

// Stats returns current memory statistics in Megabytes with decimal precision.
func (mg *MemoryGovernor) Stats() MemoryStats {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	const mb = 1024.0 * 1024.0

	return MemoryStats{
		Platform:     mg.platform,
		SoftLimitMB:  mg.softLimitMB,
		AllocMB:      float64(m.Alloc) / mb,
		TotalAllocMB: float64(m.TotalAlloc) / mb,
		SysMB:        float64(m.Sys) / mb,
		HeapIdleMB:   float64(m.HeapIdle) / mb,
		NumGC:        m.NumGC,
	}
}

// FreeMemory triggers a garbage collection pass and returns unused memory to the OS,
// respecting the cooldown timer.
func (mg *MemoryGovernor) FreeMemory() bool {
	return mg.freeMemory(false)
}

// ForceFreeMemory bypasses the cooldown timer and immediately reclaims memory.
func (mg *MemoryGovernor) ForceFreeMemory() bool {
	return mg.freeMemory(true)
}

func (mg *MemoryGovernor) freeMemory(force bool) bool {
	mg.mu.Lock()
	if !force && !mg.lastFreed.IsZero() && time.Since(mg.lastFreed) < mg.freeCooldown {
		mg.mu.Unlock()
		return false
	}
	mg.lastFreed = time.Now()
	mg.mu.Unlock()

	debug.FreeOSMemory()
	return true
}
