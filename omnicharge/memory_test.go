package omnicharge

import (
	"testing"
)

func TestMemoryGovernorOperations(t *testing.T) {
	gov := NewMemoryGovernor()
	if gov == nil {
		t.Fatalf("NewMemoryGovernor returned nil")
	}

	stats := gov.Stats()
	// AllocMB or SysMB should be greater than 0 in a running Go runtime
	if stats.SysMB == 0 {
		t.Errorf("expected System memory stats to be non-zero")
	}

	// Test memory reclamation
	gov.FreeMemory()
}
