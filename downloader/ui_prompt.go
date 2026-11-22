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

// PromptQuality asks for a quality choice supporting option numbers (1, 2...),
// shorthand (best, 1080p, 1080, 720, 480, audio), and exact ID matches.
func (r *Renderer) PromptQuality(
	in io.Reader,
	profiles []config.QualityOption,
) (string, error) {
	if len(profiles) == 0 {
		return "", fmt.Errorf("no quality profiles available")
	}

	r.write(fmt.Sprintf("\n%s=== SELECT VIDEO QUALITY ===%s\n", escBold+escCyan, escReset))
	for idx, opt := range profiles {
		r.write(fmt.Sprintf(" %s[%d] [%s]%s %s\n", escGreen, idx+1, opt.ID, escReset, opt.Label))
	}

	reader := bufio.NewReader(in)
	for attempt := 0; attempt < 5; attempt++ {
		r.write(fmt.Sprintf("Enter choice (default 1 / %s, 'q' to exit): ", profiles[0].ID))
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

		// Check if choice is a numeric index (1, 2, 3...)
		if n, err := strconv.Atoi(choice); err == nil {
			if n >= 1 && n <= len(profiles) {
				return profiles[n-1].FormatSpec, nil
			}
		}

		// Check exact or normalized ID / shorthand match
		lowerChoice := strings.ToLower(strings.TrimSuffix(choice, "p"))
		for _, opt := range profiles {
			lowerID := strings.ToLower(strings.TrimSuffix(opt.ID, "p"))
			if strings.EqualFold(choice, opt.ID) || lowerChoice == lowerID {
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
