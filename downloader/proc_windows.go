//go:build windows

package downloader

import (
	"os/exec"
	"strconv"
)

func configureProcessGroup(cmd *exec.Cmd) {
	cmd.Cancel = func() error {
		if cmd.Process == nil {
			return nil
		}
		// /T terminates the child process tree (python + yt-dlp + ffmpeg)
		return exec.Command("taskkill", "/F", "/T", "/PID", strconv.Itoa(cmd.Process.Pid)).Run()
	}
}
