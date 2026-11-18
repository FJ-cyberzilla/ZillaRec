package downloader

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync/atomic"
	"syscall"
	"time"
)

// --- Structs & Callbacks ---

type Callbacks struct {
	OnProgress    func(p Progress)
	OnDestination func(dest string)
	OnMerging     func()
	OnLog         func(line string)
}

type Result struct {
	OutputDir string
	Files     []string
}

// ServerState tracks runtime metrics for the Zilla-Rec® VR-8600 deck.
type ServerState struct {
	startTime time.Time
	activeJob atomic.Bool
}

func NewServerState() *ServerState {
	return &ServerState{startTime: time.Now()}
}

// --- yt-dlp Runner Logic ---

const progressTemplate = "download:%(progress._percent_str)s\t%(progress._speed_str)s\t%(progress._eta_str)s\t%(title)s\t%(duration_string)s"

var destRe = regexp.MustCompile(`(?:Destination:\s*|Merging formats into\s*")(.+?)"?\s*$`)

func Run(ctx context.Context, videoURL, formatSpec, cookiePath, outDir string, cb Callbacks) (*Result, error) {
	if _, err := exec.LookPath("yt-dlp"); err != nil {
		return nil, fmt.Errorf("yt-dlp not found in PATH: %w", err)
	}
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return nil, fmt.Errorf("create output dir: %w", err)
	}

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
		if cookiePath == "chrome" || cookiePath == "firefox" {
			args = append(args, "--cookies-from-browser", cookiePath)
		} else {
			if _, err := os.Stat(cookiePath); err == nil {
				args = append(args, "--cookies", cookiePath)
			}
		}
	}

	cmd := exec.CommandContext(ctx, "yt-dlp", args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
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
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			return nil, AnalyzeError(fmt.Errorf("yt-dlp: %w", waitErr), "")
		}
		return nil, AnalyzeError(fmt.Errorf("yt-dlp: %w", waitErr), msg)
	}

	files := newMP4s(outDir, before)
	if len(destinations) > 0 {
		files = dedupeAbs(destinations)
	}

	return &Result{OutputDir: outDir, Files: files}, nil
}

func handleLine(line string, cb Callbacks, destinations *[]string) {
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

func parseProgressTemplate(s string) Progress {
	parts := strings.SplitN(s, "\t", 5)
	for len(parts) < 5 {
		parts = append(parts, "")
	}
	return Progress{
		Percent:  ParsePercent(parts[0]),
		Speed:    clean(parts[1]),
		ETA:      clean(parts[2]),
		Title:    clean(parts[3]),
		Duration: clean(parts[4]),
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

var _ = errors.Is

// --- HTTP Server Handlers ---

func (s *ServerState) HandleStatus(w http.ResponseWriter, r *http.Request) {
	uptime := int(time.Since(s.startTime).Seconds())
	allocatedMB := 18.5
	if mem, err := getProcessAllocatedMB(); err == nil {
		allocatedMB = mem
	}

	stateStr := "IDLE"
	if s.activeJob.Load() {
		stateStr = "RECORDING"
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"state": stateStr,
		"memory": map[string]any{
			"allocated_mb": allocatedMB,
		},
		"health": map[string]any{
			"is_healthy": true,
		},
		"uptime_sec": uptime,
	})
}

func (s *ServerState) HandleDownload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		URL          string `json:"url"`
		Quality      string `json:"quality"`
		CookieSource string `json:"cookieSource"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	flener, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	sendEvent := func(eventType string, payload map[string]any) {
		payload["type"] = eventType
		data, _ := json.Marshal(payload)
		_, _ = w.Write([]byte("data: " + string(data) + "\n\n"))
		flener.Flush()
	}

	s.activeJob.Store(true)
	defer s.activeJob.Store(false)

	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	cb := Callbacks{
		OnProgress: func(p Progress) {
			sendEvent("progress", map[string]any{
				"percent": p.Percent,
				"speed":   p.Speed,
				"eta":     p.ETA,
			})
		},
		OnDestination: func(dest string) {
			sendEvent("destination", map[string]any{
				"message": dest,
			})
		},
		OnMerging: func() {
			sendEvent("merging", map[string]any{})
		},
		OnLog: func(line string) {
			sendEvent("log", map[string]any{
				"message": line,
			})
		},
	}

	outDir := "./downloads"
	formatSpec := "bestvideo+bestaudio/best"
	if req.Quality == "1080p" {
		formatSpec = "bestvideo[height<=1080]+bestaudio/best[height<=1080]"
	} else if req.Quality == "720p" {
		formatSpec = "bestvideo[height<=720]+bestaudio/best[height<=720]"
	} else if req.Quality == "480p" {
		formatSpec = "bestvideo[height<=480]+bestaudio/best[height<=480]"
	}

	cookiePath := ""
	if req.CookieSource != "none" {
		cookiePath = req.CookieSource
	}

	res, err := Run(ctx, req.URL, formatSpec, cookiePath, outDir, cb)
	if err != nil {
		sendEvent("error", map[string]any{
			"message": err.Error(),
		})
		return
	}

	sendEvent("success", map[string]any{
		"dir":   res.OutputDir,
		"files": res.Files,
	})
}

func HandleExit(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"shutting down"}`))
	go func() {
		time.Sleep(200 * time.Millisecond)
		os.Exit(0)
	}()
}

func getProcessAllocatedMB() (float64, error) {
	pid := os.Getpid()
	cmd := exec.Command("ps", "-o", "rss=", "-p", fmt.Sprintf("%d", pid))
	out, err := cmd.Output()
	if err != nil {
		return 15.0, err
	}
	var kb float64
	_, err = fmt.Sscan(string(out), &kb)
	if err != nil {
		return 15.0, err
	}
	return kb / 1024.0, nil
}
