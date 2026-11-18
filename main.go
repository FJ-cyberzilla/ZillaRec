package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"golang.org/x/term"

	"zillarec/config"
	"zillarec/downloader"
	"zillarec/omnicharge"
	"zillarec/storage"
	"zillarec/web"
)

// Compact mobile-friendly ASCII art banner
var bannerLines = []string{
	"███████╗██╗██╗     ██╗      █████╗         ██████╗ ███████╗ ██████╗ ",
	"╚══███╔╝██║██║     ██║     ██╔══██╗        ██╔══██╗██╔════╝██╔════╝ ",
	"  ███╔╝ ██║██║     ██║     ███████║        ██████╔╝█████╗  ██║      ",
	" ███╔╝  ██║██║     ██║     ██╔══██║        ██╔══██╗██╔══╝  ██║      ",
	"███████╗██║███████╗███████╗██║  ██║        ██║  ██║███████╗╚██████╗ ",
}

func renderBanner() {
	termWidth := 80
	if w, _, err := term.GetSize(int(os.Stdout.Fd())); err == nil && w > 0 {
		termWidth = w
	}

	// ANSI color codes: Purple/Magenta, Hot Pink, Ocean Blue
	colors := []string{
		"\033[35m", // Purple
		"\033[31m", // Pink / Bright Red
		"\033[36m", // Ocean Blue / Cyan
		"\033[34m", // Deep Blue
	}
	reset := "\033[0m"

	for idx, line := range bannerLines {
		if len(line) > termWidth && termWidth > 10 {
			line = line[:termWidth]
		}
		color := colors[idx%len(colors)]
		fmt.Println(color + line + reset)
	}

	// Sub-deck subtitle badge (fixed redundant newline for go vet)
	fmt.Println(" \033[36m:: UNIVERSAL VCR CAPTURE DECK v1.0.0 ::\033[0m")
	fmt.Println()
}

func isTTY(f *os.File) bool {
	fi, err := f.Stat()
	if err != nil {
		return false
	}
	return (fi.Mode() & os.ModeCharDevice) != 0
}

const defaultFormat = "bestvideo[height<=1080]+bestaudio/best[height<=1080]/best"

func main() {
	r := downloader.DefaultRenderer()

	// Check for web mode subcommand
	if len(os.Args) >= 2 && os.Args[1] == "web" {
		addr := "127.0.0.1:8080"
		if len(os.Args) >= 3 {
			addr = os.Args[2]
		}
		if err := web.RunServer(addr); err != nil {
			r.Error("web server error: %v", err)
			os.Exit(1)
		}
		return
	}

	renderBanner()

	// Read URL: from argv, or prompt if none given.
	var url string
	if len(os.Args) >= 2 {
		url = strings.TrimSpace(os.Args[1])
		if strings.EqualFold(url, "q") || strings.EqualFold(url, "quit") || strings.EqualFold(url, "exit") {
			r.Info("Exited.")
			os.Exit(0)
		}
	} else {
		r.Info("Insert tape / Paste video URL (or 'q' to exit):")
		fmt.Print("  › ")
		reader := bufio.NewReader(os.Stdin)
		line, _ := reader.ReadString('\n')
		url = strings.TrimSpace(line)
		if strings.EqualFold(url, "q") || strings.EqualFold(url, "quit") || strings.EqualFold(url, "exit") {
			r.Info("Exited.")
			os.Exit(0)
		}
	}
	if url == "" {
		r.Error("no URL provided")
		os.Exit(2)
	}

	// Prompt for quality selection menu
	profiles := config.QualityProfiles()
	formatSpec, err := r.PromptQuality(os.Stdin, profiles)
	if err != nil {
		if errors.Is(err, downloader.ErrUserExit) || strings.Contains(err.Error(), "user exited") {
			r.Info("Exited.")
			os.Exit(0)
		}
		r.Warn("using default quality: %v", err)
		formatSpec = defaultFormat
	}

	outDir, err := storage.OutputDir()
	if err != nil {
		r.Error("output dir: %v", err)
		os.Exit(1)
	}

	cfg := config.LoadDefaultConfig()

	bar := downloader.NewProgressBar(r)
	defer bar.EndProgress()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cb := downloader.Callbacks{
		OnProgress: bar.Update,
		OnDestination: func(dest string) {
			bar.SetStatus("Saved: " + filepath.Base(dest))
		},
		OnMerging: func() {
			bar.SetStatus("Merging formats...")
		},
		OnLog: func(line string) {
			short := line
			if idx := strings.Index(line, "]: "); idx != -1 {
				short = line[idx+3:]
			} else if idx := strings.Index(line, "]:"); idx != -1 {
				short = line[idx+2:]
			}
			bar.SetStatus(strings.TrimSpace(short))
		},
	}

	res, err := downloader.Run(ctx, url, formatSpec, cfg.CookieFile, outDir, cb)
	bar.EndProgress()
	if err != nil {
		r.Error("%v", err)
		os.Exit(1)
	}

	downloader.RegisterWithMediaStore(res.Files)

	if mc, err := omnicharge.NewMasterController(outDir, 3); err == nil && mc != nil {
		for _, file := range res.Files {
			if _, _, err := mc.Checksum().GenerateSidecar(file); err == nil {
				r.Info("OmniCharge: Generated SHA-256 checksum sidecar for %s", filepath.Base(file))
			}
		}
	}

	r.Success("✓ Download completed! Saved %d file(s) to %s", len(res.Files), res.OutputDir)
}
