package downloader

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"

	"golang.org/x/term"
)

// ANSI SGR codes matching the Zilla-Rec® VR-8600 phosphor & amber VFD palette.
const (
	escReset     = "\033[0m"
	escBold      = "\033[1m"
	escDim       = "\033[2m"
	escRed       = "\033[31m"
	escGreen     = "\033[32m"
	escYellow    = "\033[33m"
	escCyan      = "\033[36m"
	escHiRed     = "\033[91m"
	escHiGreen   = "\033[92m"
	escHiYellow  = "\033[93m"
	escHiBlack   = "\033[90m"
	escClearLine = "\r\033[K"
)

// EventListener receives structured deck events so the Web UI SSE stream
// (/api/download) stays synchronized with CLI output.
type EventListener func(eventType, message string)

// colorizer wraps text in SGR codes only when the output is a real TTY.
type colorizer struct{ enabled bool }

func newColorizer(out io.Writer) colorizer {
	if os.Getenv("NO_COLOR") != "" {
		return colorizer{}
	}
	f, ok := out.(*os.File)
	if !ok {
		return colorizer{}
	}
	return colorizer{enabled: term.IsTerminal(int(f.Fd()))}
}

func (c colorizer) wrap(code, s string) string {
	if !c.enabled || code == "" {
		return s
	}
	return code + s + escReset
}

// Renderer is the single owner of stdout for the program. Every write goes
// through mu so progress redraws and log lines never interleave.
type Renderer struct {
	mu       sync.Mutex
	out      io.Writer
	color    colorizer
	onEvent  EventListener
	inProgress bool
}

func NewRenderer(out io.Writer) *Renderer {
	return &Renderer{out: out, color: newColorizer(out)}
}

// DefaultRenderer returns a Renderer bound to the process stdout.
func DefaultRenderer() *Renderer { return NewRenderer(os.Stdout) }

// SetEventListener attaches an optional hook (e.g., to forward logs to SSE).
func (r *Renderer) SetEventListener(fn EventListener) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.onEvent = fn
}

func (r *Renderer) write(s string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.inProgress && r.color.enabled {
		_, _ = fmt.Fprint(r.out, escClearLine)
		r.inProgress = false
	}
	_, _ = fmt.Fprint(r.out, s)
}

// writeRaw writes with formatting (used by ProgressBar).
func (r *Renderer) writeRaw(format string, a ...any) {
	r.mu.Lock()
	defer r.mu.Unlock()
	_, _ = fmt.Fprintf(r.out, format, a...)
}

func (r *Renderer) emit(eventType, msg string) {
	r.mu.Lock()
	fn := r.onEvent
	r.mu.Unlock()
	if fn != nil {
		fn(eventType, msg)
	}
}

// Banner renders the Zilla-Rec® // F.J VR-8600 gunmetal deck header in the terminal.
func (r *Renderer) Banner() {
	border := r.color.wrap(escHiBlack, "╔══════════════════════════════════════════════════════════════════╗")
	mid := r.color.wrap(escHiBlack, "╠══════════════════════════════════════════════════════════════════╣")
	bottom := r.color.wrap(escHiBlack, "╚══════════════════════════════════════════════════════════════════╝")

	brand := r.color.wrap(escBold+escHiGreen, "Zilla") +
		r.color.wrap(escBold+escHiRed, "Rec® ●") +
		r.color.wrap(escHiBlack, "  │  ") +
		r.color.wrap(escDim, "UNIVERSAL CAPTURE DECK // VR-8600") +
		r.color.wrap(escHiBlack, "  │  ") +
		r.color.wrap(escBold+escHiYellow, "[F.J]")

	status := r.color.wrap(escHiGreen, "> ZILLA-REC VCR MASTER CONTROLLER ONLINE.") + "\n" +
		r.color.wrap(escHiGreen, "> OMNICHARGE SYSTEM LINKED & HEALTHY.") + "\n" +
		r.color.wrap(escHiGreen, "> READY FOR TAPE RECORDING & STREAM CAPTURE")

	r.write(fmt.Sprintf("%s\n%s %s %s\n%s\n%s\n%s\n",
		border,
		r.color.wrap(escHiBlack, "║"), brand, r.color.wrap(escHiBlack, "║"),
		mid,
		status,
		bottom,
	))
}

// Mode prints a VFD channel/transport status update (e.g., "CH 03 // DOWNLOADING").
func (r *Renderer) Mode(mode string) {
	tag := r.color.wrap(escBold+escHiYellow, fmt.Sprintf("[VFD // %s]", strings.ToUpper(mode)))
	r.write(tag + "\n")
	r.emit("log", fmt.Sprintf("> VFD STATUS: %s", strings.ToUpper(mode)))
}

// Info logs standard CRT phosphor green telemetry lines matching the Web UI terminal.
func (r *Renderer) Info(format string, a ...any) {
	msg := fmt.Sprintf(format, a...)
	prefix := r.color.wrap(escHiGreen, "> ")
	r.write(prefix + r.color.wrap(escGreen, msg) + "\n")
	r.emit("log", "> "+msg)
}

// Warn logs amber VFD advisories.
func (r *Renderer) Warn(format string, a ...any) {
	msg := fmt.Sprintf(format, a...)
	prefix := r.color.wrap(escBold+escHiYellow, "> [WARN] ")
	r.write(prefix + r.color.wrap(escYellow, msg) + "\n")
	r.emit("log", "> WARN: "+msg)
}

// Error logs crimson fault alerts matching the Web UI's .crt-output.error state.
func (r *Renderer) Error(format string, a ...any) {
	msg := fmt.Sprintf(format, a...)
	prefix := r.color.wrap(escBold+escHiRed, "> VCR ERROR: ")
	r.write(prefix + r.color.wrap(escRed, msg) + "\n")
	r.emit("error", msg)
}

// Success logs completion messages matching the Web UI's "CH 03 // PLAY (DONE)" readout.
func (r *Renderer) Success(format string, a ...any) {
	msg := fmt.Sprintf(format, a...)
	prefix := r.color.wrap(escBold+escHiGreen, "> [PLAY // DONE] ")
	r.write(prefix + r.color.wrap(escHiGreen, msg) + "\n")
	r.emit("log", "> "+msg)
}

// TapeProgress renders an inline VFD amber/green tape progress bar in the terminal.
func (r *Renderer) TapeProgress(pct float64, speed, eta string) {
	if pct < 0 {
		pct = 0
	}
	if pct > 100 {
		pct = 100
	}

	const width = 24
	filled := int((pct / 100.0) * width)
	bar := strings.Repeat("█", filled) + strings.Repeat("░", width-filled)

	line := fmt.Sprintf("%s %s %s  %s  %s",
		r.color.wrap(escBold+escHiRed, "REC ●"),
		r.color.wrap(escHiYellow, "["+bar+"]"),
		r.color.wrap(escBold+escHiYellow, fmt.Sprintf("%5.1f%%", pct)),
		r.color.wrap(escHiGreen, "SPD:"+speed),
		r.color.wrap(escDim, "ETA:"+eta),
	)

	r.mu.Lock()
	defer r.mu.Unlock()
	if r.color.enabled {
		r.inProgress = true
		_, _ = fmt.Fprintf(r.out, "%s%s", escClearLine, line)
		if pct >= 100 {
			_, _ = fmt.Fprint(r.out, "\n")
			r.inProgress = false
		}
	} else if pct >= 100 {
		_, _ = fmt.Fprintln(r.out, line)
	}
}
