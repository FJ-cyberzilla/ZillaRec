package storage

import (
	"fmt"
	"os"
)

const (
	SharedMovies  = "/storage/emulated/0/Movies"
	AppSubfolder  = "YT-Downloader"
	LocalFallback = "downloads"
)

func OutputDir() (string, error) {
	if dirExists(SharedMovies) {
		dir := SharedMovies + "/" + AppSubfolder
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return "", fmt.Errorf("create shared output dir: %w", err)
		}
		return dir, nil
	}
	if err := os.MkdirAll(LocalFallback, 0o755); err != nil {
		return "", fmt.Errorf("create local output dir: %w", err)
	}
	return LocalFallback, nil
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}
