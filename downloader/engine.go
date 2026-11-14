package downloader

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"syscall"
)

type Callbacks struct {
	// OnProgress receives already-parsed progress. Nil-safe.
	OnProgress func(p Progress)
	// OnDestination fires with the final file path yt-dlp reports.
	OnDestination func(dest string)
	// OnMerging fires once when yt-dlp starts merging.
	OnMerging func()
	// OnLog forwards non-progress output (info/warnings). Optional.
	OnLog func(line string)
}

type Result struct {
	OutputDir string
	Files     []string
}

// progressTemplate makes yt-dlp emit machine-parseable progress lines.
// Fields are separated by tabs to be unambiguous with printf's own spaces.
const progressTemplate = "download:%(progress._percent_str)s\t%(progress._speed_str)s\t%(progress._eta_str)s"

// destRe matches:  [download] Destination: foo.mp4
//                  [Merger]   Merging formats into "foo.mp4"
var destRe = regexp.MustCompile(`(?:Destination:\s*|Merging formats into\s*")(.+?)"?\s*$`)

func Run(ctx context.Context, videoURL, formatSpec, cookiePath, outDir string, cb Callbacks) (*Result, error) {
	if _, err := exec.LookPath("yt-dlp"); err != nil {
		return nil, fmt.Errorf("yt-dlp not found in PATH: %w", err)
	}
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return nil, fmt.Errorf("create output dir: %w", err)
	}

	// Snapshot existing mp4s so collectMP4s only returns new ones.
	before := collectMP4s(outDir)

	args := []string{
		videoURL,
		"-f", formatSpec,
		"--newline",
		"--no-warnings",
		"--progress-template", progressTemplate,
		"--socket-timeout", "15",
		"--retries", "3",
		"--fragment-retries", "3",
		"-P", outDir,
		"-o", "%(id)s-%(resolution)s.%(ext)s",
		"--restrict-filenames",
		"--merge-output-format", "mp4",
	}
	if cookiePath != "" {
		if _, err := os.Stat(cookiePath); err == nil {
			args = append(args, "--cookies", cookiePath)
		}
	}

	cmd := exec.CommandContext(ctx, "yt-dlp", args...)

	// Put yt-dlp in its own process group so cancel kills ffmpeg too.
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	// Belt-and-braces: on context cancel, kill the whole group.
	cmd.Cancel = func() error {
		if cmd.Process == nil {
			return nil
		}
		return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("stdout pipe: %w", err)
	}

	// Capture stderr instead of leaking to the terminal: it would corrupt
	// the progress bar, and we only care about it on failure.
	var stderr strings.Builder
	cmd.Stderr = &stderr

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start yt-dlp: %w", err)
	}

	destinations := make([]string, 0, 4)

	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 0, 64*1024), 1<<20)
	for scanner.Scan() {
		line := scanner.Text()
		handleLine(line, cb, &destinations)
	}
	scanErr := scanner.Err()

	waitErr := cmd.Wait()

	if scanErr != nil {
		return nil, AnalyzeError(fmt.Errorf("read yt-dlp output: %w", scanErr), stderr.String())
	}
	if waitErr != nil {
		// Surface stderr, it has the real reason.
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			return nil, AnalyzeError(fmt.Errorf("yt-dlp: %w", waitErr), "")
		}
		return nil, AnalyzeError(fmt.Errorf("yt-dlp: %w", waitErr), msg)
	}

	files := newMP4s(outDir, before)
	// Prefer files yt-dlp actually reported over directory diffing.
	if len(destinations) > 0 {
		files = dedupeAbs(destinations)
	}

	return &Result{OutputDir: outDir, Files: files}, nil
}

func handleLine(line string, cb Callbacks, destinations *[]string) {
	// Progress line from our --progress-template.
	if strings.HasPrefix(line, "download:") {
		if cb.OnProgress != nil {
			p := parseProgressTemplate(strings.TrimPrefix(line, "download:"))
			cb.OnProgress(p)
		}
		return
	}

	if m := destRe.FindStringSubmatch(line); m != nil {
		dest := strings.TrimSpace(m[1])
		*destinations = append(*destinations, dest)
		if cb.OnDestination != nil {
			cb.OnDestination(dest)
		}
		if strings.Contains(line, "Merging") && cb.OnMerging != nil {
			cb.OnMerging()
		}
		return
	}

	if cb.OnLog != nil && strings.TrimSpace(line) != "" {
		cb.OnLog(line)
	}
}

// parseProgressTemplate splits the tab-separated template output.
// yt-dlp leaves literal "N/A" when a value is unknown.
func parseProgressTemplate(s string) Progress {
	parts := strings.SplitN(s, "\t", 3)
	for len(parts) < 3 {
		parts = append(parts, "")
	}
	return Progress{
		Percent: ParsePercent(parts[0]),
		Speed:   clean(parts[1]),
		ETA:     clean(parts[2]),
	}
}

func clean(s string) string {
	s = strings.TrimSpace(s)
	if s == "N/A" || s == "NA" || s == "Unknown" {
		return ""
	}
	return s
}

func collectMP4s(dir string) map[string]struct{} {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	set := make(map[string]struct{}, len(entries))
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(strings.ToLower(e.Name()), ".mp4") {
			continue
		}
		if abs, err := filepath.Abs(filepath.Join(dir, e.Name())); err == nil {
			set[abs] = struct{}{}
		}
	}
	return set
}

func newMP4s(dir string, before map[string]struct{}) []string {
	after := collectMP4s(dir)
	var out []string
	for f := range after {
		if _, existed := before[f]; !existed {
			out = append(out, f)
		}
	}
	sort.Strings(out)
	return out
}

func dedupeAbs(paths []string) []string {
	seen := make(map[string]struct{}, len(paths))
	var out []string
	for _, p := range paths {
		abs, err := filepath.Abs(p)
		if err != nil {
			continue
		}
		if _, ok := seen[abs]; ok {
			continue
		}
		seen[abs] = struct{}{}
		out = append(out, abs)
	}
	sort.Strings(out)
	return out
}

// keep errors.Is reachable for callers that want to detect cancellation.
var _ = errors.Is
