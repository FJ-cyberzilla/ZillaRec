package omnicharge

import (
	"strings"
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
}

// HealthCatwatch monitors download and network health using downloader error analysis (with feline vigilance).
type HealthCatwatch struct {
	maxRetries int
	retryCount int
}

// NewHealthCatwatch creates a new health catwatch with max retry limits.
func NewHealthCatwatch(maxRetries int) *HealthCatwatch {
	return &HealthCatwatch{
		maxRetries: maxRetries,
		retryCount: 0,
	}
}

// EvaluateError processes an error using downloader.AnalyzeError and determines recovery action.
func (hc *HealthCatwatch) EvaluateError(err error) HealthStatus {
	if err == nil {
		hc.retryCount = 0
		return HealthStatus{
			IsHealthy:    true,
			LastCategory: CategoryNone,
			ShouldAbort:  false,
		}
	}

	analyzed := downloader.AnalyzeError(err, "")
	msg := analyzed.Error()
	lower := strings.ToLower(msg)

	cat := CategoryGeneric
	shouldRetry := true

	if strings.Contains(lower, "youtube restriction") || strings.Contains(lower, "ip ban") {
		cat = CategoryIPBan
		shouldRetry = false
	} else if strings.Contains(lower, "dns error") {
		cat = CategoryDNS
	} else if strings.Contains(lower, "network error") {
		cat = CategoryNetwork
	} else if strings.Contains(lower, "vpn / proxy") {
		cat = CategoryProxyVPN
	}

	if shouldRetry {
		if hc.retryCount < hc.maxRetries {
			hc.retryCount++
		} else {
			shouldRetry = false
		}
	}

	return HealthStatus{
		IsHealthy:    false,
		LastCategory: cat,
		LastMessage:  msg,
		RetryCount:   hc.retryCount,
		MaxRetries:   hc.maxRetries,
		ShouldAbort:  !shouldRetry,
	}
}

// Reset clears retry counters on successful completion.
func (hc *HealthCatwatch) Reset() {
	hc.retryCount = 0
}

// BackoffDuration calculates exponential backoff delay based on current retry count.
func (hc *HealthCatwatch) BackoffDuration() time.Duration {
	if hc.retryCount <= 0 {
		return time.Second
	}
	sec := 1 << uint(hc.retryCount)
	if sec > 30 {
		sec = 30
	}
	return time.Duration(sec) * time.Second
}
