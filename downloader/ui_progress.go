package downloader

import (
	"strings"
	"time"

	"golang.org/x/term"
)

// Progress is already-parsed progress state. Callers own parsing yt-dlp
// output; the renderer stays dumb and testable.
type Progress struct {
	Percent float64 // clamped to 0..100 for display
	Speed   string
	ETA     string
}

// ProgressBar redraws a single line in place and throttles updates.
type ProgressBar struct {
	r        *Renderer
	lastDraw time.Time
	active   bool
}

func NewProgressBar(r *Renderer) *ProgressBar { return &ProgressBar{r: r} }

func (b *ProgressBar) Update(p Progress) {
	if p.Percent < 0 {
		p.Percent = 0
	}
	if p.Percent > 100 {
		p.Percent = 100
	}

	// Piped / NO_COLOR: no animation, just periodic plain lines.
	if !b.r.color.enabled {
		if time.Since(b.lastDraw) < time.Second {
			return
		}
		b.lastDraw = time.Now()
		b.r.writeRaw("progress: %.1f%% %s %s\n", p.Percent, p.Speed, p.ETA)
		return
	}

	if time.Since(b.lastDraw) < 100*time.Millisecond {
		return
	}
	b.lastDraw = time.Now()
	b.active = true

	width := b.barWidth()
	filled := int(p.Percent/100*float64(width) + 0.5)
	if filled < 0 {
		filled = 0
	}
	if filled > width {
		filled = width
	}
	bar := strings.Repeat("█", filled) + strings.Repeat("░", width-filled)

	b.r.writeRaw("\r\033[K%s[%s]%s %s%6.1f%%%s  %s%s%s  %s%s%s",
		escGreen, bar, escReset,
		escBold, p.Percent, escReset,
		escYellow, p.Speed, escReset,
		escCyan, p.ETA, escReset,
	)
}

// EndProgress terminates the in-place line. Call with defer.
func (b *ProgressBar) EndProgress() {
	if b.active {
		b.r.writeRaw("\n")
		b.active = false
	}
}

func (b *ProgressBar) barWidth() int {
	const (
		minW     = 10
		maxW     = 30
		reserved = 40
	)
	f, ok := b.r.out.(interface{ Fd() uintptr })
	if !ok {
		return 18
	}
	w, _, err := term.GetSize(int(f.Fd()))
	if err != nil || w <= 0 {
		return 18
	}
	bw := w - reserved
	if bw < minW {
		return minW
	}
	if bw > maxW {
		return maxW
	}
	return bw
}
