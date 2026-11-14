package downloader

import (
	"bufio"
	"os"
	"os/exec"
	"strings"
)

// DownloadCallbacks defines clean hooks for UI updates during download execution
type DownloadCallbacks struct {
	OnProgress    func(line string)
	OnDestination func(dest string)
	OnMerging     func()
}

// RunEngine executes yt-dlp safely and streams line events back through callbacks
func RunEngine(videoURL, formatSpec, cookiePath string, callbacks DownloadCallbacks) error {
	args := []string{
		videoURL,
		"-f", formatSpec,
		"--progress",
		"--newline",
		"--no-warnings",
	}

	if cookiePath != "" {
		if _, err := os.Stat(cookiePath); err == nil {
			args = append(args, "--cookies", cookiePath)
		}
	}

	cmd := exec.Command("yt-dlp", args...)
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return err
	}

	scanner := bufio.NewScanner(stdoutPipe)
	for scanner.Scan() {
		line := scanner.Text()

		if callbacks.OnProgress != nil && strings.Contains(line, "[download]") && strings.Contains(line, "%") {
			callbacks.OnProgress(line)
		} else if callbacks.OnDestination != nil && strings.Contains(line, "Destination:") {
			parts := strings.SplitN(line, "Destination:", 2)
			if len(parts) == 2 {
				callbacks.OnDestination(strings.TrimSpace(parts[1]))
			}
		} else if callbacks.OnMerging != nil && strings.Contains(line, "Merging") {
			callbacks.OnMerging()
		}
	}

	return cmd.Wait()
}
