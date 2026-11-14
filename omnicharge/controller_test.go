package omnicharge

import (
	"errors"
	"path/filepath"
	"testing"
)

func TestSecurityManager(t *testing.T) {
	tmpDir := t.TempDir()
	sec, err := NewSecurityManager(tmpDir)
	if err != nil {
		t.Fatalf("failed to create security manager: %v", err)
	}

	// Valid subpath
	validPath := filepath.Join(tmpDir, "downloads", "video.mp4")
	resolved, err := sec.ValidatePath(validPath)
	if err != nil {
		t.Errorf("expected valid path, got error: %v", err)
	}
	if resolved == "" {
		t.Errorf("expected non-empty resolved path")
	}

	// Traversal attack path
	badPath := filepath.Join(tmpDir, "..", "etc", "passwd")
	_, err = sec.ValidatePath(badPath)
	if err == nil {
		t.Errorf("expected security violation error for path traversal, got nil")
	}
}

func TestMemoryGovernor(t *testing.T) {
	gov := NewMemoryGovernor()
	stats := gov.Stats()
	if stats.SysMB == 0 && stats.AllocMB == 0 {
		// In some minimal test environments stats might be zero, but usually non-zero.
		t.Logf("Memory stats: %+v", stats)
	}
	gov.FreeMemory()
}

func TestHealthWatchdog(t *testing.T) {
	wd := NewHealthWatchdog(3)

	// Test nil error (healthy)
	status := wd.EvaluateError(nil)
	if !status.IsHealthy {
		t.Errorf("expected healthy status for nil error")
	}

	// Test transient network error
	netErr := errors.New("dial tcp: lookup youtube.com: no such host")
	status = wd.EvaluateError(netErr)
	if status.IsHealthy || status.ShouldAbort {
		t.Errorf("expected retryable network error, got status: %+v", status)
	}

	// Test backoff duration
	backoff := wd.BackoffDuration()
	if backoff <= 0 {
		t.Errorf("expected positive backoff duration, got %v", backoff)
	}

	// Test reset
	wd.Reset()
}

func TestMasterControllerLifecycle(t *testing.T) {
	tmpDir := t.TempDir()
	mc, err := NewMasterController(tmpDir, 3)
	if err != nil {
		t.Fatalf("failed to init master controller: %v", err)
	}

	if mc.State() != StateIdle {
		t.Errorf("expected StateIdle, got %v", mc.State())
	}

	mc.SetState(StateRunning)
	if mc.State() != StateRunning {
		t.Errorf("expected StateRunning, got %v", mc.State())
	}

	mc.Pause()
	if mc.State() != StatePaused {
		t.Errorf("expected StatePaused, got %v", mc.State())
	}

	mc.Resume()
	if mc.State() != StateRunning {
		t.Errorf("expected StateRunning after resume, got %v", mc.State())
	}

	snapshot := mc.Snapshot()
	if snapshot.State != StateRunning {
		t.Errorf("expected snapshot state StateRunning, got %v", snapshot.State)
	}
	if snapshot.BaseDir == "" {
		t.Errorf("expected non-empty base dir in snapshot")
	}

	mc.Stop()
	if mc.State() != StateIdle {
		t.Errorf("expected StateIdle after stop, got %v", mc.State())
	}
}
