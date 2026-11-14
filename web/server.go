package web

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"zillarec/downloader"
	"zillarec/omnicharge"
	"zillarec/storage"
)

//go:embed index.html
var webFiles embed.FS

var masterCtrl *omnicharge.MasterController

type DownloadRequest struct {
	URL          string `json:"url"`
	CookieSource string `json:"cookieSource"`
	Quality      string `json:"quality"` // <-- ADDED
}

type SSEEvent struct {
	Type     string   `json:"type"`
	Message  string   `json:"message,omitempty"`
	Percent  float64  `json:"percent,omitempty"`
	Speed    string   `json:"speed,omitempty"`
	ETA      string   `json:"eta,omitempty"`
	Title    string   `json:"title,omitempty"`
	Duration string   `json:"duration,omitempty"`
	Dir      string   `json:"dir,omitempty"`     // <-- ADDED
	Files    []string `json:"files,omitempty"`   // <-- ADDED
}

func RunServer(addr string) error {
	outDir, err := storage.OutputDir()
	if err != nil {
		outDir = "."
	}
	masterCtrl, err = omnicharge.NewMasterController(outDir, 3)
	if err != nil {
		return fmt.Errorf("failed to init omnicharge master controller: %w", err)
	}

	subFS, err := fs.Sub(webFiles, ".")
	if err != nil {
		return fmt.Errorf("sub fs: %w", err)
	}
	mux := http.NewServeMux()
	mux.Handle("/", http.FileServer(http.FS(subFS)))
	mux.HandleFunc("/api/download", handleDownload)
	mux.HandleFunc("/api/status", handleStatus)
	mux.HandleFunc("/api/exit", handleExit)

	const (
		cyan    = "\033[1;36m"
		magenta = "\033[1;35m"
		green   = "\033[1;32m"
		yellow  = "\033[1;33m"
		reset   = "\033[0m"
	)

	fmt.Printf("%s==========================================%s\n", cyan, reset)
	fmt.Printf("%s OMNICHARGE MASTER CONTROLLER - WEB UI%s\n", magenta, reset)
	fmt.Printf("%s==========================================%s\n", cyan, reset)
	fmt.Printf("%sServer running at:%s %shttp://%s%s\n", yellow, reset, green, addr, reset)
	fmt.Printf("%sPath Sandbox:%s %s\n", cyan, reset, masterCtrl.Security().BaseDir())
	fmt.Printf("%sPress Ctrl+C or click Shutdown to stop.%s\n", cyan, reset)

	return http.ListenAndServe(addr, mux)
}

func handleStatus(w http.ResponseWriter, r *http.Request) {
	if masterCtrl == nil {
		http.Error(w, `{"error":"controller not initialized"}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(masterCtrl.Snapshot())
}

func handleExit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if masterCtrl != nil {
		masterCtrl.Stop()
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"message":"Server shutting down..."}`))
	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}
	go func() {
		time.Sleep(100 * time.Millisecond)
		os.Exit(0)
	}()
}

func handleDownload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req DownloadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || strings.TrimSpace(req.URL) == "" {
		http.Error(w, `{"type":"error","message":"Invalid request payload"}`, http.StatusBadRequest)
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

	sendEvent := func(evt SSEEvent) {
		bs, _ := json.Marshal(evt)
		_, _ = fmt.Fprintf(w, "data: %s\n\n", string(bs))
		flusher.Flush()
	}

	if masterCtrl != nil {
		masterCtrl.SetState(omnicharge.StateRunning)
		defer masterCtrl.SetState(omnicharge.StateIdle)
	}

	outDir, err := storage.OutputDir()
	if err != nil {
		sendEvent(SSEEvent{Type: "error", Message: fmt.Sprintf("Output dir error: %v", err)})
		return
	}

	if masterCtrl != nil {
		if _, err := masterCtrl.Security().ValidatePath(outDir); err != nil {
			sendEvent(SSEEvent{Type: "error", Message: fmt.Sprintf("Security violation: %v", err)})
			return
		}
	}

	// --- QUALITY SELECTION LOGIC ---
	var formatSpec string
	switch req.Quality {
	case "1080p":
		formatSpec = "bestvideo[height<=1080]+bestaudio/bestvideo[width<=1080]+bestaudio/best[height<=1080]/best"
	case "720p":
		formatSpec = "bestvideo[height<=720]+bestaudio/best[height<=720]/best"
	case "480p":
		formatSpec = "bestvideo[height<=480]+bestaudio/best[height<=480]/best"
	default:
		// "best" or fallback
		formatSpec = "bestvideo+bestaudio/best"
	}
	sendEvent(SSEEvent{Type: "log", Message: fmt.Sprintf("[FORMAT] Requested quality: %s", req.Quality)})

	// --- COOKIE LOGIC ---
	cookiePath := req.CookieSource
	if cookiePath == "" || cookiePath == "auto" {
		cookiePath = "cookies.txt"
		if _, err := os.Stat(cookiePath); os.IsNotExist(err) {
			if home, err := os.UserHomeDir(); err == nil {
				altCookie := filepath.Join(home, ".termux", "cookies.txt")
				if _, err := os.Stat(altCookie); err == nil {
					cookiePath = altCookie
				} else {
					cookiePath = ""
				}
			} else {
				cookiePath = ""
			}
		}
	} else if cookiePath == "none" {
		cookiePath = ""
	}

	if cookiePath == "chrome" || cookiePath == "firefox" {
		sendEvent(SSEEvent{Type: "log", Message: fmt.Sprintf("[COOKIES] Extracting cookies directly from browser: %s", cookiePath)})
	} else if cookiePath != "" {
		sendEvent(SSEEvent{Type: "log", Message: fmt.Sprintf("[COOKIES] Using cookie file: %s", cookiePath)})
	} else {
		sendEvent(SSEEvent{Type: "log", Message: "[COOKIES] No cookies selected. Proceeding without cookies."})
	}

	cb := downloader.Callbacks{
		OnProgress: func(p downloader.Progress) {
			sendEvent(SSEEvent{
				Type:     "progress",
				Percent:  p.Percent * 100, // Convert 0.0-1.0 to 0-100
				Speed:    p.Speed,
				ETA:      p.ETA,
				Title:    p.Title,
				Duration: p.Duration,
			})
		},
		OnDestination: func(dest string) {
			sendEvent(SSEEvent{Type: "destination", Message: dest})
		},
		OnMerging: func() {
			sendEvent(SSEEvent{Type: "merging"}) // <-- Explicit Type for Frontend
		},
		OnLog: func(line string) {
			sendEvent(SSEEvent{Type: "log", Message: line})
		},
	}

	res, err := downloader.Run(r.Context(), req.URL, formatSpec, cookiePath, outDir, cb)

	if err != nil {
		if masterCtrl != nil {
			healthStatus := masterCtrl.Watchdog().EvaluateError(err)
			sendEvent(SSEEvent{
				Type:    "error",
				Message: fmt.Sprintf("[OMNICHARGE WATCHDOG] %s (Retry %d/%d)", healthStatus.LastMessage, healthStatus.RetryCount, healthStatus.MaxRetries),
			})
		} else {
			sendEvent(SSEEvent{Type: "error", Message: err.Error()})
		}
		return
	}

	downloader.RegisterWithMediaStore(res.Files)
	if masterCtrl != nil {
		masterCtrl.Memory().FreeMemory()
		for _, file := range res.Files {
			if _, _, err := masterCtrl.Checksum().GenerateSidecar(file); err == nil {
				sendEvent(SSEEvent{Type: "log", Message: fmt.Sprintf("[OMNICHARGE CHECKSUM] Generated SHA-256 sidecar for %s", filepath.Base(file))})
			}
		}
	}

	// Send final success event with files and dir
	sendEvent(SSEEvent{
		Type:  "success",
		Dir:   res.OutputDir,
		Files: res.Files,
		Message: fmt.Sprintf("✓ Download completed! Saved %d file(s)", len(res.Files)),
	})
}
