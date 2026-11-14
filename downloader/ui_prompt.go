package downloader

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"

	"zillarec/config"
)

var ErrUserExit = fmt.Errorf("user exited")

// PromptQuality asks for a quality choice. It re-prompts on invalid input
// and returns an error for an empty profile list or closed stdin.
func (r *Renderer) PromptQuality(
	in io.Reader,
	profiles []config.QualityOption,
) (string, error) {
	if len(profiles) == 0 {
		return "", fmt.Errorf("no quality profiles available")
	}

	r.write(fmt.Sprintf("\n%s=== SELECT VIDEO QUALITY ===%s\n", escBold+escCyan, escReset))
	for _, opt := range profiles {
		r.write(fmt.Sprintf(" %s[%s]%s %s\n", escGreen, opt.ID, escReset, opt.Label))
	}

	reader := bufio.NewReader(in)
	for attempt := 0; attempt < 5; attempt++ {
		r.write(fmt.Sprintf("Enter choice (default %s, 'q' to exit): ", profiles[0].ID))
		line, err := reader.ReadString('\n')
		if err != nil && line == "" {
			return "", fmt.Errorf("reading choice: %w", err)
		}
		choice := strings.TrimSpace(line)

		if strings.EqualFold(choice, "q") || strings.EqualFold(choice, "quit") || strings.EqualFold(choice, "exit") {
			return "", ErrUserExit
		}

		if choice == "" {
			return profiles[0].FormatSpec, nil
		}
		for _, opt := range profiles {
			if choice == opt.ID {
				return opt.FormatSpec, nil
			}
		}
		r.write(fmt.Sprintf("%sinvalid choice %q, try again%s\n", escYellow, choice, escReset))
	}
	return "", fmt.Errorf("too many invalid attempts")
}

// ParsePercent turns yt-dlp's percentage string into a number. Unknown
// values ("N/A", "") yield 0 rather than panicking.
func ParsePercent(s string) float64 {
	s = strings.TrimSpace(s)
	s = strings.TrimSuffix(s, "%")
	s = strings.TrimSpace(s)
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0
	}
	return v
}
