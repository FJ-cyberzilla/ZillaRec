package omnicharge

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// ChecksumManager handles automatic computation, sidecar generation, and verification of file checksums (SHA-256)
// as part of the OmniCharge Master Controller duty.
type ChecksumManager struct{}

// NewChecksumManager initializes a new ChecksumManager.
func NewChecksumManager() *ChecksumManager {
	return &ChecksumManager{}
}

// ComputeSHA256 calculates the SHA-256 hex digest of a file at the given path.
func (cm *ChecksumManager) ComputeSHA256(filePath string) (string, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to open file for checksum: %w", err)
	}
	defer f.Close()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, f); err != nil {
		return "", fmt.Errorf("failed to compute hash: %w", err)
	}

	return hex.EncodeToString(hasher.Sum(nil)), nil
}

// GenerateSidecar computes the SHA-256 checksum of a downloaded file and writes a .sha256 sidecar file automatically.
func (cm *ChecksumManager) GenerateSidecar(filePath string) (string, string, error) {
	hash, err := cm.ComputeSHA256(filePath)
	if err != nil {
		return "", "", err
	}

	sidecarPath := filePath + ".sha256"
	content := fmt.Sprintf("%s  %s\n", hash, filepath.Base(filePath))
	if err := os.WriteFile(sidecarPath, []byte(content), 0o644); err != nil {
		return hash, "", fmt.Errorf("failed to write checksum sidecar: %w", err)
	}

	return hash, sidecarPath, nil
}

// VerifyFile verifies a file against its .sha256 sidecar file or an expected hash string.
func (cm *ChecksumManager) VerifyFile(filePath string, expectedHash string) (bool, error) {
	actualHash, err := cm.ComputeSHA256(filePath)
	if err != nil {
		return false, err
	}

	if expectedHash != "" {
		return strings.EqualFold(actualHash, strings.TrimSpace(expectedHash)), nil
	}

	// Read sidecar file
	sidecarPath := filePath + ".sha256"
	sidecarBytes, err := os.ReadFile(sidecarPath)
	if err != nil {
		return false, fmt.Errorf("failed to read checksum sidecar: %w", err)
	}

	parts := strings.Fields(string(sidecarBytes))
	if len(parts) == 0 {
		return false, fmt.Errorf("invalid checksum sidecar content")
	}

	expected := parts[0]
	return strings.EqualFold(actualHash, strings.TrimSpace(expected)), nil
}
