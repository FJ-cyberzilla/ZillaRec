package omnicharge

import (
	"os"
	"path/filepath"
	"testing"
)

func TestChecksumManager(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "sample.mp4")
	content := []byte("fake video content for checksum verification")
	if err := os.WriteFile(testFile, content, 0o644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	cm := NewChecksumManager()

	// Test ComputeSHA256
	hash, err := cm.ComputeSHA256(testFile)
	if err != nil {
		t.Fatalf("ComputeSHA256 failed: %v", err)
	}
	if hash == "" {
		t.Errorf("expected non-empty hash")
	}

	// Test GenerateSidecar
	computedHash, sidecarPath, err := cm.GenerateSidecar(testFile)
	if err != nil {
		t.Fatalf("GenerateSidecar failed: %v", err)
	}
	if computedHash != hash {
		t.Errorf("expected hash %s, got %s", hash, computedHash)
	}
	if _, err := os.Stat(sidecarPath); err != nil {
		t.Errorf("expected sidecar file to exist at %s: %v", sidecarPath, err)
	}

	// Test VerifyFile with sidecar
	valid, err := cm.VerifyFile(testFile, "")
	if err != nil || !valid {
		t.Errorf("expected file verification against sidecar to succeed, got valid=%v, err=%v", valid, err)
	}

	// Test VerifyFile with explicit expected hash
	valid, err = cm.VerifyFile(testFile, hash)
	if err != nil || !valid {
		t.Errorf("expected verification against explicit hash to succeed, got valid=%v, err=%v", valid, err)
	}

	// Test VerifyFile with incorrect hash
	valid, err = cm.VerifyFile(testFile, "invalidhash123")
	if err != nil || valid {
		t.Errorf("expected verification with incorrect hash to fail, got valid=%v, err=%v", valid, err)
	}
}
