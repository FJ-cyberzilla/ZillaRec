package omnicharge

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSecurityManagerValidation(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "sec_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	sm, err := NewSecurityManager(tmpDir)
	if err != nil {
		t.Fatalf("failed to create SecurityManager: %v", err)
	}

	if sm.BaseDir() == "" {
		t.Errorf("expected BaseDir to be non-empty")
	}

	// Valid path inside base dir
	validTarget := filepath.Join(tmpDir, "subdir", "file.mp4")
	if _, err := sm.ValidatePath(validTarget); err != nil {
		t.Errorf("expected valid path to be accepted, got: %v", err)
	}

	// Traversal attempt
	traversal := filepath.Join(tmpDir, "..", "secret.txt")
	if _, err := sm.ValidatePath(traversal); err == nil {
		t.Errorf("expected directory traversal attempt to be rejected")
	}
}
