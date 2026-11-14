package downloader

import (
	"fmt"
	"io"
	"os"
	"sync"

	"golang.org/x/term"
)

// ANSI SGR codes. Unexported so callers use the Renderer methods.
const (
	escReset  = "\033[0m"
	escBold   = "\033[1m"
	escRed    = "\033[31m"
	escGreen  = "\033[32m"
	escYellow = "\033[33m"
	escCyan   = "\033[36m"
)

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
	mu    sync.Mutex
	out   io.Writer
	color colorizer
}

func NewRenderer(out io.Writer) *Renderer {
	return &Renderer{out: out, color: newColorizer(out)}
}

// Default returns a Renderer bound to the process stdout.
func DefaultRenderer() *Renderer { return NewRenderer(os.Stdout) }

func (r *Renderer) write(format string, a ...any) {
	r.mu.Lock()
	defer r.mu.Unlock()
	_, _ = fmt.Fprintf(r.out, format, a...)
}

// writeRaw writes without color wrapping (used by ProgressBar, which
// already embeds its own escapes).
func (r *Renderer) writeRaw(format string, a ...any) { r.write(format, a...) }

func (r *Renderer) Info(format string, a ...any) {
	r.write(r.color.wrap(escCyan, fmt.Sprintf(format, a...)) + "\n")
}
func (r *Renderer) Warn(format string, a ...any) {
	r.write(r.color.wrap(escYellow, fmt.Sprintf(format, a...)) + "\n")
}
func (r *Renderer) Error(format string, a ...any) {
	r.write(r.color.wrap(escRed, fmt.Sprintf(format, a...)) + "\n")
}
func (r *Renderer) Success(format string, a ...any) {
	r.write(r.color.wrap(escGreen, fmt.Sprintf(format, a...)) + "\n")
}
