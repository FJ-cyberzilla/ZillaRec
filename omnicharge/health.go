package omnicharge

import (
	"strings"
	"time"
	"zillarec/downloader"
)

// ErrorCategory classifies download and network errors.
type ErrorCategory string

const (
	CategoryNone         ErrorCategory = "NONE"
	CategoryNetwork      ErrorCategory = "NETWORK"
	CategoryDNS          ErrorCategory = "DNS"
	CategoryProxyVPN     ErrorCategory = "PROXY_VPN"
	CategoryIPBan        ErrorCategory = "IP_BAN"
	CategoryGeneric      ErrorCategory = "GENERIC"
)

// HealthStatus represents the current health state of the application.
type HealthStatus struct {
	IsHealthy     bool
	LastCategory  ErrorCategory
	LastMessage   string
	RetryCount    int
	MaxRetries    int
	ShouldAbort   bool
}

// HealthWatchdog monitors download and network health using downloader error analysis.
type HealthWatchdog struct {
	maxRetries int
	retryCount int
}

// NewHealthWatchdog creates a new health watchdog with max retry limits.
func NewHealthWatchdog(maxRetries int) *HealthWatchdog {
	return &HealthWatchdog{
		maxRetries: maxRetries,
		retryCount: 0,
	}
}

// EvaluateError processes an error using downloader.AnalyzeError and determines recovery action.
func (hw *HealthWatchdog) EvaluateError(err error) HealthStatus {
	if err == nil {
		hw.retryCount = 0
		return HealthStatus{
			IsHealthy:   true,
			LastCategory: CategoryNone,
			ShouldAbort: false,
		}
	}

	analyzed := downloader.AnalyzeError(err, "")
	msg := analyzed.Error()
	lower := strings.ToLower(msg)

	cat := CategoryGeneric
	shouldRetry := true

	if strings.Contains(lower, "youtube restriction") || strings.Contains(lower, "ip ban") {
		cat = CategoryIPBan
		shouldRetry = false // IP bans require abort / manual intervention
	} else if strings.Contains(lower, "dns error") {
		cat = CategoryDNS
	} else if strings.Contains(lower, "network error") {
		cat = CategoryNetwork
	} else if strings.Contains(lower, "vpn / proxy") {
		cat = CategoryProxyVPN
	}

	if shouldRetry {
		if hw.retryCount < hw.maxRetries {
			hw.retryCount++
		} else {
			shouldRetry = false
		}
	}

	return HealthStatus{
		IsHealthy:    false,
		LastCategory: cat,
		LastMessage:  msg,
		RetryCount:   hw.retryCount,
		MaxRetries:   hw.maxRetries,
		ShouldAbort:  !shouldRetry,
	}
}

// Reset clears retry counters on successful completion.
func (hw *HealthWatchdog) Reset() {
	hw.retryCount = 0
}

// BackoffDuration calculates exponential backoff delay based on current retry count.
func (hw *HealthWatchdog) BackoffDuration() time.Duration {
	if hw.retryCount <= 0 {
		return time.Second
	}
	// 2^retryCount seconds, capped at 30 seconds
	sec := 1 << uint(hw.retryCount)
	if sec > 30 {
		sec = 30
	}
	return time.Duration(sec) * time.Second
}
