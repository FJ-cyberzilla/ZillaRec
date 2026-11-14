package storage

import (
	"os"
	"path/filepath"
	"testing"
)

func TestOutputDir(t *testing.T) {
	// Test that OutputDir returns a valid path and successfully ensures directory creation
	dir, err := OutputDir()
	if err != nil {
		t.Fatalf("OutputDir failed: %v", err)
	}
	if dir == "" {
		t.Errorf("OutputDir returned empty string")
	}

	// Verify the returned directory exists on disk and is a directory
	info, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("failed to stat output dir %s: %v", dir, err)
	}
	if !info.IsDir() {
		t.Errorf("output path %s is not a directory", dir)
	}
}

func TestDirExists(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "test_dir_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	if !dirExists(tmpDir) {
		t.Errorf("dirExists returned false for existing directory %s", tmpDir)
	}

	nonExistent := filepath.Join(tmpDir, "does_not_exist")
	if dirExists(nonExistent) {
		t.Errorf("dirExists returned true for non-existent path %s", nonExistent)
	}
}
