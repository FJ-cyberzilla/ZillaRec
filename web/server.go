package web

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"strings"

	"yt-termux/config"
	"yt-termux/downloader"
	"yt-termux/storage"
)

//go:embed index.html
var webFiles embed.FS

type DownloadRequest struct {
	URL string `json:"url"`
}

type DownloadResponse struct {
	Message string `json:"message"`
}

func RunServer(addr string) error {
	subFS, err := fs.Sub(webFiles, ".")
	if err != nil {
		return fmt.Errorf("sub fs: %w", err)
	}
	mux := http.NewServeMux()
	mux.Handle("/", http.FileServer(http.FS(subFS)))
	mux.HandleFunc("/api/download", handleDownload)

	fmt.Println("==========================================")
	fmt.Println(" FJ™ CYBERTRONIC SYSTEMS - WEB INTERFACE")
	fmt.Println("==========================================")
	fmt.Printf("Server running at: http://%s\n", addr)
	fmt.Println("Press Ctrl+C to stop.")

	return http.ListenAndServe(addr, mux)
}

func handleDownload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req DownloadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || strings.TrimSpace(req.URL) == "" {
		http.Error(w, `{"message":"Invalid request payload"}`, http.StatusBadRequest)
		return
	}

	outDir, err := storage.OutputDir()
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"message":"Output dir error: %v"}`, err), http.StatusInternalServerError)
		return
	}

	cfg := config.LoadDefaultConfig()
	
	var logBuilder strings.Builder
	
	cb := downloader.Callbacks{
		OnDestination: func(dest string) {
			logBuilder.WriteString(fmt.Sprintf("> DESTINATION: %s\n", dest))
		},
		OnMerging: func() {
			logBuilder.WriteString("> MERGING STREAMS...\n")
		},
		OnLog: func(line string) {
			logBuilder.WriteString(fmt.Sprintf("> %s\n", line))
		},
	}

	res, err := downloader.Run(context.Background(), req.URL,
		"bestvideo[height<=1080]+bestaudio/best[height<=1080]/best",
		cfg.CookieFile, outDir, cb)

	w.Header().Set("Content-Type", "application/json")
	
	if err != nil {
		logBuilder.WriteString(fmt.Sprintf("> ERROR: %v\n", err))
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(DownloadResponse{Message: logBuilder.String()})
		return
	}

	logBuilder.WriteString(fmt.Sprintf("> SAVED %d FILE(S) TO %s\n", len(res.Files), res.OutputDir))
	for _, f := range res.Files {
		logBuilder.WriteString(fmt.Sprintf("> FILE: %s\n", f))
	}

	_ = json.NewEncoder(w).Encode(DownloadResponse{Message: logBuilder.String()})
}
