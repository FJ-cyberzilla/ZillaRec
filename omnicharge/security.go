package omnicharge

import (
	"fmt"
	"path/filepath"
	"strings"
)

// SecurityManager handles path sandboxing and validation to prevent directory traversal.
type SecurityManager struct {
	baseDir string
}

// NewSecurityManager creates a security manager anchored to a base directory.
func NewSecurityManager(baseDir string) (*SecurityManager, error) {
	absBase, err := filepath.Abs(baseDir)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve base directory %s: %w", baseDir, err)
	}
	return &SecurityManager{baseDir: absBase}, nil
}

// ValidatePath checks if the given target path resides within the base directory.
// It prevents directory traversal attacks (e.g. "../../../etc/passwd").
func (sm *SecurityManager) ValidatePath(targetPath string) (string, error) {
	absTarget, err := filepath.Abs(targetPath)
	if err != nil {
		return "", fmt.Errorf("failed to resolve target path %s: %w", targetPath, err)
	}

	// Ensure the target is within baseDir
	rel, err := filepath.Rel(sm.baseDir, absTarget)
	if err != nil {
		return "", fmt.Errorf("path relation check failed: %w", err)
	}

	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("security violation: path %s escapes base directory %s", targetPath, sm.baseDir)
	}

	return absTarget, nil
}

// BaseDir returns the configured base directory.
func (sm *SecurityManager) BaseDir() string {
	return sm.baseDir
}
