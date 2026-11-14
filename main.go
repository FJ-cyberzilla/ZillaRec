package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"yt-termux/config"
	"yt-termux/downloader"
	"yt-termux/storage"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: yt-downloader <video-url>")
		os.Exit(2)
	}
	url := os.Args[1]

	outDir, err := storage.OutputDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "output dir: %v\n", err)
		os.Exit(1)
	}

	cfg := config.LoadDefaultConfig()
	profiles := config.QualityProfiles()

	r := downloader.DefaultRenderer()
	bar := downloader.NewProgressBar(r)
	defer bar.EndProgress()

	spec, err := r.PromptQuality(os.Stdin, profiles)
	if err != nil {
		r.Error("%v", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cb := downloader.Callbacks{
		OnProgress: bar.Update,
		OnDestination: func(dest string) {
			bar.EndProgress()
			r.Info("→ %s", filepath.Base(dest))
		},
		OnMerging: func() {
			bar.EndProgress()
			r.Info("merging…")
		},
		OnLog: func(line string) {
			bar.EndProgress()
			r.Info("%s", line)
		},
	}

	res, err := downloader.Run(ctx, url, spec, cfg.CookieFile, outDir, cb)
	bar.EndProgress()
	if err != nil {
		r.Error("%v", err)
		os.Exit(1)
	}

	downloader.RegisterWithMediaStore(res.Files)

	r.Success("saved %d file(s) to %s", len(res.Files), res.OutputDir)
	for _, f := range res.Files {
		r.Info("  %s", f)
	}
}
