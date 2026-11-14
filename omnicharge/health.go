package omnicharge

import (
	"context"
	"math/rand/v2"
	"strings"
	"sync"
	"time"
	"zillarec/downloader"
)

// ErrorCategory classifies download and network errors.
type ErrorCategory string

const (
	CategoryNone     ErrorCategory = "NONE"
	CategoryNetwork  ErrorCategory = "NETWORK"
	CategoryDNS      ErrorCategory = "DNS"
	CategoryProxyVPN ErrorCategory = "PROXY_VPN"
	CategoryIPBan    ErrorCategory = "IP_BAN"
	CategoryMemory   ErrorCategory = "MEMORY_PRESSURE"
	CategoryGeneric  ErrorCategory = "GENERIC"
)

// HealthStatus represents the current health state observed by the Catwatch.
type HealthStatus struct {
	IsHealthy    bool
	LastCategory ErrorCategory
	LastMessage  string
	RetryCount   int
	MaxRetries   int
	ShouldAbort  bool
	MemoryStats  *MemoryStats
}

// WatcherConfig configures the background memory and health watcher.
type WatcherConfig struct {
	CheckInterval        time.Duration
	HeapThresholdPercent float64
	OnMemoryFreed        func(stats MemoryStats)
}

// HealthCatwatch monitors download, network, and memory health with feline vigilance.
type HealthCatwatch struct {
	mu           sync.Mutex
	maxRetries   int
	retryCount   int
	lastCategory ErrorCategory
	lastMessage  string
	governor     *MemoryGovernor
}

// NewHealthCatwatch creates a thread-safe health monitor.
// Optionally accepts an existing *MemoryGovernor; otherwise creates one automatically.
func NewHealthCatwatch(maxRetries int, gov ...*MemoryGovernor) *HealthCatwatch {
	if maxRetries < 0 {
		maxRetries = 0
	}

	var governor *MemoryGovernor
	if len(gov) > 0 && gov[0] != nil {
		governor = gov[0]
	} else {
		governor = NewMemoryGovernor()
	}

	return &HealthCatwatch{
		maxRetries:   maxRetries,
		retryCount:   0,
		lastCategory: CategoryNone,
		governor:     governor,
	}
}

// StartWatcher runs a background goroutine that periodically checks heap usage
// against the MemoryGovernor soft limit and reclaims RAM automatically.
func (hc *HealthCatwatch) StartWatcher(ctx context.Context, cfg ...WatcherConfig) {
	var config WatcherConfig
	if len(cfg) > 0 {
		config = cfg[0]
	}
	if config.CheckInterval <= 0 {
		config.CheckInterval = 3 * time.Second
	}
	if config.HeapThresholdPercent <= 0 || config.HeapThresholdPercent > 100 {
		config.HeapThresholdPercent = 80.0
	}

	go func() {
		ticker := time.NewTicker(config.CheckInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				stats := hc.governor.Stats()
				if stats.SoftLimitMB <= 0 {
					continue
				}

				usagePercent := (stats.AllocMB / float64(stats.SoftLimitMB)) * 100.0
				if usagePercent >= config.HeapThresholdPercent {
					if hc.governor.FreeMemory() {
						hc.mu.Lock()
						hc.lastCategory = CategoryMemory
						hc.lastMessage = "auto-reclaimed heap memory above threshold"
						hc.mu.Unlock()

						if config.OnMemoryFreed != nil {
							config.OnMemoryFreed(hc.governor.Stats())
						}
					}
				}
			}
		}
	}()
}

// EvaluateError processes an error using downloader.AnalyzeError and determines recovery action.
func (hc *HealthCatwatch) EvaluateError(err error) HealthStatus {
	hc.mu.Lock()
	defer hc.mu.Unlock()

	stats := hc.governor.Stats()

	if err == nil {
		hc.retryCount = 0
		hc.lastCategory = CategoryNone
		hc.lastMessage = ""
		return HealthStatus{
			IsHealthy:    true,
			LastCategory: CategoryNone,
			MaxRetries:   hc.maxRetries,
			ShouldAbort:  false,
			MemoryStats:  &stats,
		}
	}

	analyzed := downloader.AnalyzeError(err, "")
	msg := analyzed.Error()
	lower := strings.ToLower(msg)

	cat := CategoryGeneric
	shouldRetry := true

	switch {
	case strings.Contains(lower, "youtube restriction") || strings.Contains(lower, "ip ban"):
		cat = CategoryIPBan
		shouldRetry = false
	case strings.Contains(lower, "dns error"):
		cat = CategoryDNS
	case strings.Contains(lower, "network error"):
		cat = CategoryNetwork
	case strings.Contains(lower, "vpn / proxy"):
		cat = CategoryProxyVPN
	}

	if shouldRetry {
		if hc.retryCount < hc.maxRetries {
			hc.retryCount++
		} else {
			shouldRetry = false
		}
	}

	hc.lastCategory = cat
	hc.lastMessage = msg

	return HealthStatus{
		IsHealthy:    false,
		LastCategory: cat,
		LastMessage:  msg,
		RetryCount:   hc.retryCount,
		MaxRetries:   hc.maxRetries,
		ShouldAbort:  !shouldRetry,
		MemoryStats:  &stats,
	}
}

// Reset clears retry counters and error state on successful completion.
func (hc *HealthCatwatch) Reset() {
	hc.mu.Lock()
	defer hc.mu.Unlock()
	hc.retryCount = 0
	hc.lastCategory = CategoryNone
	hc.lastMessage = ""
}

// BackoffDuration calculates exponential backoff with jitter and overflow protection.
func (hc *HealthCatwatch) BackoffDuration() time.Duration {
	hc.mu.Lock()
	retries := hc.retryCount
	hc.mu.Unlock()

	if retries <= 0 {
		return time.Second
	}
	if retries > 5 {
		retries = 5
	}

	baseSec := 1 << uint(retries)
	if baseSec > 30 {
		baseSec = 30
	}

	jitter := time.Duration(rand.IntN(500)) * time.Millisecond
	return time.Duration(baseSec)*time.Second + jitter
}
