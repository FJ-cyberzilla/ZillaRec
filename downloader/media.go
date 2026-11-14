package downloader

import (
	"os/exec"
	"runtime"
)

func RegisterWithMediaStore(paths []string) {
	if runtime.GOOS != "android" {
		if _, err := exec.LookPath("am"); err != nil {
			return
		}
	}
	for _, p := range paths {
		_ = exec.Command("am", "broadcast",
			"-a", "android.intent.action.MEDIA_SCANNER_SCAN_FILE",
			"-d", "file://"+p,
		).Run()
	}
}
