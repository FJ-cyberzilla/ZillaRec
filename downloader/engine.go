package downloader

import (
        "bufio"
        "context"
        "encoding/json"
        "fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
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

type Progress struct {
	Percent  float64
	Speed    string
	ETA      string
	Title    string
	Duration string
}

// ServerState tracks runtime metrics for the Zilla-Rec® VR-8600 deck.
type ServerState struct {
	startTime time.Time
	activeJob atomic.Bool
}

func NewServerState() *ServerState {
	return &ServerState{startTime: time.Now()}
}

// --- Orchestrated Job & Worker Pool Support ---

type Job struct {
	URL        string
	Format     string
	CookiePath string
	OutDir     string
	Callbacks  Callbacks
	ResultChan chan<- JobResult
}

type JobResult struct {
	Result *Result
	Error  error
}

type WorkerPool struct {
	maxWorkers int
	jobChan    chan Job
	wg         sync.WaitGroup
	ctx        context.Context
	cancel     context.CancelFunc
}

func NewWorkerPool(ctx context.Context, maxWorkers int, queueSize int) *WorkerPool {
	c, cancel := context.WithCancel(ctx)
	wp := &WorkerPool{
		maxWorkers: maxWorkers,
		jobChan:    make(chan Job, queueSize),
		ctx:        c,
		cancel:     cancel,
	}
	wp.start()
	return wp
}

func (wp *WorkerPool) start() {
	for i := 0; i < wp.maxWorkers; i++ {
		wp.wg.Add(1)
		go func() {
			defer wp.wg.Done()
			for {
				select {
				case <-wp.ctx.Done():
					return
				case job, ok := <-wp.jobChan:
					if !ok {
						return
					}
					res, err := RunWithRetry(wp.ctx, job.URL, job.Format, job.CookiePath, job.OutDir, job.Callbacks, 3)
					job.ResultChan <- JobResult{Result: res, Error: err}
				}
			}
		}()
	}
}

func (wp *WorkerPool) Submit(job Job) {
	select {
	case <-wp.ctx.Done():
		job.ResultChan <- JobResult{Result: nil, Error: wp.ctx.Err()}
	case wp.jobChan <- job:
	}
}

func (wp *WorkerPool) Close() {
	wp.cancel()
	close(wp.jobChan)
	wp.wg.Wait()
}

// --- Python Worker IPC & Retry Orchestrator ---

func RunWithRetry(ctx context.Context, videoURL, formatSpec, cookiePath, outDir string, cb Callbacks, maxRetries int) (*Result, error) {
	var lastErr error
	backoff := 1 * time.Second

	for attempt := 1; attempt <= maxRetries; attempt++ {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}

		res, err := Run(ctx, videoURL, formatSpec, cookiePath, outDir, cb)
		if err == nil {
			return res, nil
		}

		lastErr = err
		if cb.OnLog != nil {
			cb.OnLog(fmt.Sprintf("[RETRY] Attempt %d/%d failed: %v. Retrying in %v...", attempt, maxRetries, err, backoff))
		}

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(backoff):
			backoff *= 2
			if backoff > 10*time.Second {
				backoff = 10 * time.Second
			}
		}
	}
	return nil, fmt.Errorf("failed after %d retries: %w", maxRetries, lastErr)
}

func Run(ctx context.Context, videoURL, formatSpec, cookiePath, outDir string, cb Callbacks) (*Result, error) {
	if _, err := exec.LookPath("python3"); err != nil {
		return nil, fmt.Errorf("python3 not found in PATH: %w", err)
	}

	workerScript := "worker.py"
	if _, err := os.Stat(workerScript); os.IsNotExist(err) {
		return nil, fmt.Errorf("worker.py not found: %w", err)
	}

	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return nil, fmt.Errorf("create output dir: %w", err)
	}

	before := collectMP4s(outDir)

	args := []string{workerScript, videoURL, "--format", formatSpec, "--output-dir", outDir}
	if cookiePath != "" {
		args = append(args, "--cookies", cookiePath)
	}

	cmd := exec.CommandContext(ctx, "python3", args...)
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
		return nil, fmt.Errorf("start worker.py: %w", err)
	}

	var destinations []string

	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 0, 64*1024), 1<<20)

	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.SplitN(line, "\t", 2)
		prefix := parts[0]
		payload := ""
		if len(parts) > 1 {
			payload = parts[1]
		}

		switch prefix {
		case "download":
			if cb.OnProgress != nil {
				p := parseProgressTemplate(payload)
				cb.OnProgress(p)
			}
		case "DEST":
			destinations = append(destinations, payload)
			if cb.OnDestination != nil {
				cb.OnDestination(payload)
			}
			if strings.Contains(payload, "Merging") && cb.OnMerging != nil {
				cb.OnMerging()
			}
		case "LOG":
			if cb.OnLog != nil && payload != "" {
				cb.OnLog(payload)
			}
		case "SUCCESS":
			if cb.OnLog != nil {
				cb.OnLog(payload)
			}
		case "ERROR":
			if cb.OnLog != nil {
				cb.OnLog(payload)
			}
		default:
			if cb.OnLog != nil && line != "" {
				cb.OnLog(line)
			}
		}
	}

        scanErr := scanner.Err()
	waitErr := cmd.Wait()

	if scanErr != nil {
		return nil, fmt.Errorf("read worker output: %w", scanErr)
	}
	if waitErr != nil {
		return nil, fmt.Errorf("worker execution failed: %w (stderr: %s)", waitErr, strings.TrimSpace(stderr.String()))
	}

	files := newMP4s(outDir, before)
	if len(destinations) > 0 {
		files = dedupeAbs(destinations)
	}

	return &Result{OutputDir: outDir, Files: files}, nil
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

	flusher, ok := w.(http.Flusher)
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
		flusher.Flush()
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
	formatSpec := qualityToFormatSpec(req.Quality)

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

func qualityToFormatSpec(quality string) string {
	switch quality {
	case "1080p":
		return "bestvideo[height<=1080]+bestaudio/best[height<=1080]"
	case "720p":
		return "bestvideo[height<=720]+bestaudio/best[height<=720]"
	case "480p":
		return "bestvideo[height<=480]+bestaudio/best[height<=480]"
	case "audio":
		return "bestaudio/best"
	default:
		return "bestvideo+bestaudio/best"
	}
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
