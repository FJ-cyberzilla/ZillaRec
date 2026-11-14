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

	"yt-termux/downloader"
	"yt-termux/storage"
)

//go:embed index.html
var webFiles embed.FS

type DownloadRequest struct {
	URL          string `json:"url"`
	CookieSource string `json:"cookieSource"`
}

func RunServer(addr string) error {
	subFS, err := fs.Sub(webFiles, ".")
	if err != nil {
		return fmt.Errorf("sub fs: %w", err)
	}
	mux := http.NewServeMux()
	mux.Handle("/", http.FileServer(http.FS(subFS)))
	mux.HandleFunc("/api/download", handleDownload)
	mux.HandleFunc("/api/exit", handleExit)

	const (
		cyan    = "\033[1;36m"
		magenta = "\033[1;35m"
		green   = "\033[1;32m"
		yellow  = "\033[1;33m"
		reset   = "\033[0m"
	)

	fmt.Printf("%s==========================================%s\n", cyan, reset)
	fmt.Printf("%s FJ™ CYBERTRONIC SYSTEMS - WEB INTERFACE%s\n", magenta, reset)
	fmt.Printf("%s==========================================%s\n", cyan, reset)
	fmt.Printf("%sServer running at:%s %shttp://%s%s\n", yellow, reset, green, addr, reset)
	fmt.Printf("%sPress Ctrl+C or click Shutdown to stop.%s\n", cyan, reset)

	return http.ListenAndServe(addr, mux)
}

func handleExit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(`{"message":"Server shutting down..."}`))
	go func() {
		time.Sleep(200 * time.Millisecond)
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

	sendEvent := func(evt map[string]interface{}) {
		bs, _ := json.Marshal(evt)
		_, _ = fmt.Fprintf(w, "data: %s\n\n", string(bs))
		flusher.Flush()
	}

	outDir, err := storage.OutputDir()
	if err != nil {
		sendEvent(map[string]interface{}{"type": "error", "message": fmt.Sprintf("Output dir error: %v", err)})
		return
	}

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
		sendEvent(map[string]interface{}{"type": "log", "message": fmt.Sprintf("[COOKIES] Extracting cookies directly from browser: %s", cookiePath)})
	} else if cookiePath != "" {
		sendEvent(map[string]interface{}{"type": "log", "message": fmt.Sprintf("[COOKIES] Using cookie file: %s", cookiePath)})
	} else {
		sendEvent(map[string]interface{}{"type": "log", "message": "[COOKIES] No cookies selected. Proceeding without cookies."})
	}

	cb := downloader.Callbacks{
		OnProgress: func(p downloader.Progress) {
			sendEvent(map[string]interface{}{
				"type":    "progress",
				"percent": p.Percent,
				"speed":   p.Speed,
				"eta":     p.ETA,
			})
		},
		OnDestination: func(dest string) {
			sendEvent(map[string]interface{}{"type": "log", "message": fmt.Sprintf("[DEST] %s", dest)})
		},
		OnMerging: func() {
			sendEvent(map[string]interface{}{"type": "log", "message": "[MERGE] Merging video and audio streams..."})
		},
		OnLog: func(line string) {
			sendEvent(map[string]interface{}{"type": "log", "message": line})
		},
	}

	res, err := downloader.Run(r.Context(), req.URL,
		"bestvideo[height<=1080]+bestaudio/best[height<=1080]/best",
		cookiePath, outDir, cb)

	if err != nil {
		sendEvent(map[string]interface{}{"type": "error", "message": err.Error()})
		return
	}

	var files []string
	if res != nil {
		files = res.Files
	}
	sendEvent(map[string]interface{}{
		"type":  "success",
		"files": files,
		"dir":   outDir,
	})
}
