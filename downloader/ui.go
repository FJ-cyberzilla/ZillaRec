package downloader

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"yt-termux/config"
)

// ANSI Terminal Color Codes & Styles
const (
	ColorReset  = "\033[0m"
	ColorCyan   = "\033[36m"
	ColorGreen  = "\033[32m"
	ColorYellow = "\033[33m"
	ColorBold   = "\033[1m"
	ColorDim    = "\033[2m"
)

// PromptQualityWithConfig renders a colorful interactive quality menu
func PromptQualityWithConfig(profiles []config.QualityOption) string {
	fmt.Printf("\n%s=== SELECT VIDEO QUALITY ===%s\n", ColorBold+ColorCyan, ColorReset)
	for _, opt := range profiles {
		fmt.Printf(" %s[%s]%s %s\n", ColorGreen, opt.ID, ColorReset, opt.Label)
	}
	fmt.Printf("Enter choice (default 1): ")

	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)

	for _, opt := range profiles {
		if input == opt.ID {
			return opt.FormatSpec
		}
	}
	
	return profiles[0].FormatSpec
}

// ExecuteDownload provides the seamless entry point called by main, wrapping the engine with UI feedback
func ExecuteDownload(videoURL, formatSpec, cookiePath string) error {
	if cookiePath != "" {
		fmt.Printf("%s[Info] Using secure local cookies profile.%s\n", ColorCyan, ColorReset)
	}

	fmt.Printf("\n%s[Engine] Secure handshake established over Quad9 DNS.%s\n", ColorCyan+ColorBold, ColorReset)
	fmt.Printf("%s[Engine] Extracting media streams...%s\n\n", ColorDim, ColorReset)

	// Wire up UI callbacks into the pure engine
	callbacks := DownloadCallbacks{
		OnProgress: func(line string) {
			renderProgressBar(line)
		},
		OnDestination: func(dest string) {
			fmt.Printf("\n%s📥 Downloading segment:%s %s\n", ColorCyan, ColorReset, dest)
		},
		OnMerging: func() {
			fmt.Printf("\n%s⚡ Multiplexing audio and video into single file...%s\n", ColorBold+ColorGreen, ColorReset)
		},
	}

	return RunEngine(videoURL, formatSpec, cookiePath, callbacks)
}

// renderProgressBar parses yt-dlp metrics and draws a sleek block progress bar with colors
func renderProgressBar(line string) {
	fields := strings.Fields(line)
	if len(fields) < 2 {
		return
	}

	var percentStr, speedStr, etaStr string

	for i, f := range fields {
		if strings.HasSuffix(f, "%") {
			percentStr = f
		}
		if f == "at" && i+1 < len(fields) {
			speedStr = fields[i+1]
		}
		if f == "ETA" && i+1 < len(fields) {
			etaStr = fields[i+1]
		}
	}

	pVal := 0.0
	if percentStr != "" {
		cleaned := strings.TrimSuffix(percentStr, "%")
		if v, err := strconv.ParseFloat(cleaned, 64); err == nil {
			pVal = v
		}
	}

	barWidth := 18
	completed := int((pVal / 100.0) * float64(barWidth))
	if completed > barWidth {
		completed = barWidth
	}
	if completed < 0 {
		completed = 0
	}

	bar := strings.Repeat("█", completed) + strings.Repeat("░", barWidth-completed)

	fmt.Printf("\r\033[K%s[%s]%s %s%6s%s | Speed: %s%s%s | ETA: %s%s%s",
		ColorGreen, bar, ColorReset,
		ColorBold, percentStr, ColorReset,
		ColorYellow, speedStr, ColorReset,
		ColorCyan, etaStr, ColorReset,
	)
}
