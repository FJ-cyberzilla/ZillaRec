package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"yt-termux/config"
	"yt-termux/downloader"
	"yt-termux/storage"
)

// bannerLines: YT DOWNLOADER in a compact mini-font that fits phone screens.
var bannerLines = []string{
	` _   _ _____   ____   _____        _______ ____  `,
	`| | | |_   _| |  _ \ / _ \ \      / / ____|  _ \ `,
	`| |_| | | |   | | | | | | \ \ /\ / /|  _| | |_) |`,
	` \__, | | |   | |_| | |_| |\ V  V / | |___|  _ < `,
	` |___/  |_|   |____/ \___/  \_/\_/  |_____|_| \_\`,
}

func renderBanner() {
	// Gradient stops: Gold (220) -> Red (196) -> Deep Blue (21)
	left, right := 220, 21

	if !isTTY(os.Stdout) {
		for _, line := range bannerLines {
			fmt.Println(line)
		}
		fmt.Println()
		return
	}

	for _, line := range bannerLines {
		runes := []rune(line)
		if len(runes) == 0 {
			fmt.Println()
			continue
		}
		var b strings.Builder
		for i, ch := range runes {
			t := float64(i) / float64(len(runes)-1)
			code := interpColor(left, right, t)
			b.WriteString(fmt.Sprintf("\033[38;5;%dm%c", code, ch))
		}
		b.WriteString("\033[0m")
		fmt.Println(b.String())
	}
	fmt.Println()
}

func interpColor(a, b int, t float64) int {
	if t <= 0 {
		return a
	}
	if t >= 1 {
		return b
	}
	ar, ag, ab := cube(a)
	br, bg, bb := cube(b)
	r := int(float64(ar)*(1-t) + float64(br)*t)
	g := int(float64(ag)*(1-t) + float64(bg)*t)
	bl := int(float64(ab)*(1-t) + float64(bb)*t)
	return to256(r, g, bl)
}

func cube(c int) (r, g, b int) {
	switch {
	case c >= 232:
		v := 8 + (c-232)*10
		return v, v, v
	case c >= 16:
		c -= 16
		r = c / 36
		g = (c % 36) / 6
		b = c % 6
		return r * 51, g * 51, b * 51
	default:
		return 255, 0, 0
	}
}

func to256(r, g, b int) int {
	toCube := func(v int) int {
		if v < 48 {
			return 0
		}
		if v < 114 {
			return 1
		}
		return (v - 35) / 40
	}
	return 16 + 36*toCube(r) + 6*toCube(g) + toCube(b)
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
	renderBanner()

	// Read URL: from argv, or prompt if none given.
	var url string
	if len(os.Args) >= 2 {
		url = strings.TrimSpace(os.Args[1])
	} else {
		r.Info("Paste video URL:")
		fmt.Print("  › ")
		var line string
		fmt.Fscanln(os.Stdin, &line)
		url = strings.TrimSpace(line)
	}
	if url == "" {
		r.Error("no URL provided")
		os.Exit(2)
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

	res, err := downloader.Run(ctx, url, defaultFormat, cfg.CookieFile, outDir, cb)
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
