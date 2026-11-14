package web

import (
	"context"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"zillarec/downloader"
	"zillarec/omnicharge"
	"zillarec/storage"
)

//go:embed index.html
var webFiles embed.FS

var (
	masterCtrl      *omnicharge.MasterController
	ui              = downloader.DefaultRenderer()
	activeDownloads atomic.Int32
	shutdownOnce    sync.Once
	triggerShutdown func()
)

type DownloadRequest struct {
	URL          string `json:"url"`
	CookieSource string `json:"cookieSource"`
	Quality      string `json:"quality"`
}

type SSEEvent struct {
	Type     string   `json:"type"`
	Message  string   `json:"message,omitempty"`
	Percent  float64  `json:"percent,omitempty"`
	Speed    string   `json:"speed,omitempty"`
	ETA      string   `json:"eta,omitempty"`
	Title    string   `json:"title,omitempty"`
	Duration string   `json:"duration,omitempty"`
	Dir      string   `json:"dir,omitempty"`
	Files    []string `json:"files,omitempty"`
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

	serverCtx, stopRequests := context.WithCancel(context.Background())
	defer stopRequests()

	srv := &http.Server{
		Addr:    addr,
		Handler: mux,
		BaseContext: func(_ net.Listener) context.Context {
			return serverCtx
		},
	}

	shutdownDone := make(chan error, 1)
	triggerShutdown = func() {
		shutdownOnce.Do(func() {
			go func() {
				ui.Warn("Shutting down server gracefully...")
				stopRequests()

				if masterCtrl != nil {
					masterCtrl.Stop()
				}

				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer cancel()

				shutdownDone <- srv.Shutdown(ctx)
			}()
		})
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(sigCh)

	go func() {
		if _, ok := <-sigCh; ok {
			triggerShutdown()
		}
	}()

	ui.Info("==========================================")
	ui.Info(" OMNICHARGE MASTER CONTROLLER - WEB UI")
	ui.Info("==========================================")
	ui.Success("Server running at: http://%s", addr)
	ui.Info("Path Sandbox: %s", masterCtrl.Security().BaseDir())
	ui.Warn("Press Ctrl+C or click Shutdown to stop.")

	err = srv.ListenAndServe()
	if errors.Is(err, http.ErrServerClosed) {
		if shutdownErr := <-shutdownDone; shutdownErr != nil {
			ui.Error("Forced shutdown after timeout: %v", shutdownErr)
			return shutdownErr
		}
		ui.Success("Server stopped cleanly.")
		return nil
	}
	return err
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

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"message":"Server shutting down gracefully..."}`))
	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}

	if triggerShutdown != nil {
		triggerShutdown()
	}
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

	var sseMu sync.Mutex
	sendEvent := func(evt SSEEvent) {
		if r.Context().Err() != nil {
			return
		}
		bs, err := json.Marshal(evt)
		if err != nil {
			return
		}
		sseMu.Lock()
		defer sseMu.Unlock()
		_, _ = fmt.Fprintf(w, "data: %s\n\n", string(bs))
		flusher.Flush()
	}

	if masterCtrl != nil {
		if activeDownloads.Add(1) == 1 {
			masterCtrl.SetState(omnicharge.StateRunning)
		}
		defer func() {
			if activeDownloads.Add(-1) == 0 {
				masterCtrl.SetState(omnicharge.StateIdle)
			}
		}()
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

	formatSpec := resolveFormatSpec(req.Quality)
	sendEvent(SSEEvent{Type: "log", Message: fmt.Sprintf("[FORMAT] Requested quality: %s", req.Quality)})

	cookiePath := resolveCookieSource(req.CookieSource, sendEvent)

	cb := downloader.Callbacks{
		OnProgress: func(p downloader.Progress) {
			sendEvent(SSEEvent{
				Type:     "progress",
				Percent:  p.Percent * 100,
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
			sendEvent(SSEEvent{Type: "merging"})
		},
		OnLog: func(line string) {
			sendEvent(SSEEvent{Type: "log", Message: line})
		},
	}

	res, err := downloader.Run(r.Context(), req.URL, formatSpec, cookiePath, outDir, cb)
	if err != nil {
		if r.Context().Err() != nil {
			return
		}
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
				sendEvent(SSEEvent{
					Type:    "log",
					Message: fmt.Sprintf("[OMNICHARGE CHECKSUM] Generated SHA-256 sidecar for %s", filepath.Base(file)),
				})
			}
		}
	}

	sendEvent(SSEEvent{
		Type:    "success",
		Dir:     res.OutputDir,
		Files:   res.Files,
		Message: fmt.Sprintf("✓ Download completed! Saved %d file(s)", len(res.Files)),
	})
}

func resolveFormatSpec(quality string) string {
	switch quality {
	case "1080p":
		return "bestvideo[height<=1080]+bestaudio/bestvideo[width<=1080]+bestaudio/best[height<=1080]/best"
	case "720p":
		return "bestvideo[height<=720]+bestaudio/best[height<=720]/best"
	case "480p":
		return "bestvideo[height<=480]+bestaudio/best[height<=480]/best"
	default:
		return "bestvideo+bestaudio/best"
	}
}

func resolveCookieSource(source string, sendEvent func(SSEEvent)) string {
	switch source {
	case "none":
		sendEvent(SSEEvent{Type: "log", Message: "[COOKIES] No cookies selected. Proceeding without cookies."})
		return ""
	case "chrome", "firefox":
		sendEvent(SSEEvent{Type: "log", Message: fmt.Sprintf("[COOKIES] Extracting cookies directly from browser: %s", source)})
		return source
	case "", "auto":
		if _, err := os.Stat("cookies.txt"); err == nil {
			sendEvent(SSEEvent{Type: "log", Message: "[COOKIES] Using cookie file: cookies.txt"})
			return "cookies.txt"
		}
		if home, err := os.UserHomeDir(); err == nil {
			altCookie := filepath.Join(home, ".termux", "cookies.txt")
			if _, err := os.Stat(altCookie); err == nil {
				sendEvent(SSEEvent{Type: "log", Message: fmt.Sprintf("[COOKIES] Using cookie file: %s", altCookie)})
				return altCookie
			}
		}
		sendEvent(SSEEvent{Type: "log", Message: "[COOKIES] No cookies found. Proceeding without cookies."})
		return ""
	default:
		if masterCtrl != nil {
			safePath, err := masterCtrl.Security().ValidatePath(source)
			if err != nil {
				sendEvent(SSEEvent{Type: "log", Message: fmt.Sprintf("[COOKIES] Rejected unsafe cookie path: %v", err)})
				return ""
			}
			sendEvent(SSEEvent{Type: "log", Message: fmt.Sprintf("[COOKIES] Using cookie file: %s", safePath)})
			return safePath
		}
		sendEvent(SSEEvent{Type: "log", Message: fmt.Sprintf("[COOKIES] Using cookie file: %s", source)})
		return source
	}
}
