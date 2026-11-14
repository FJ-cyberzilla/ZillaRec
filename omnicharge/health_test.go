package omnicharge

import (
	"errors"
	"testing"
)

func TestHealthWatchdogEvaluation(t *testing.T) {
	wd := NewHealthCatwatch(2)

	// Test nil error (healthy)
	status := wd.EvaluateError(nil)
	if !status.IsHealthy {
		t.Errorf("expected healthy status for nil error")
	}

	// Test network error (should retry)
	netErr := errors.New("network error: connection refused")
	status = wd.EvaluateError(netErr)
	if status.IsHealthy || status.LastCategory != CategoryNetwork {
		t.Errorf("expected network category error, got %v", status.LastCategory)
	}

	// Test IP ban error (should abort immediately)
	banErr := errors.New("youtube restriction: IP ban detected")
	status = wd.EvaluateError(banErr)
	if !status.ShouldAbort || status.LastCategory != CategoryIPBan {
		t.Errorf("expected IP ban error to trigger immediate abort, got ShouldAbort=%v, Cat=%v", status.ShouldAbort, status.LastCategory)
	}

	// Test reset
	wd.Reset()
	if wd.BackoffDuration() <= 0 {
		t.Errorf("expected valid backoff duration")
	}
}
