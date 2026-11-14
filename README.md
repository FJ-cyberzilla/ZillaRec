<div align="center">

# ZILLA-REC®
### VCR MASTER CONTROLLER // High-Performance Video Downloader

[![Go Version](https://img.shields.io/badge/Go-1.22+-00ADD8?style=for-the-badge&logo=go)](https://go.dev/)
[![Platform](https://img.shields.io/badge/Platform-Termux%20%7C%20Android-black?style=for-the-badge&logo=android)](https://termux.dev/)
[![UI](https://img.shields.io/badge/UI-Vintage%20VHS%20VCR-85925c?style=for-the-badge)]()

**A sleek, robust, and secure local CLI & Vintage VHS Web interface for downloading YouTube videos, engineered specifically for Termux on Android.**

</div>

---

## ⚡ Features

- **Dual Interface:** Run it directly in the terminal (CLI) or start the vintage VCR web server (`yt-termux web` / `make server`) to use via your mobile browser.
- **Real-Time Streaming Progress (SSE):** Web UI streams live progress updates via Server-Sent Events, powering a retro progress bar with real-time percentage, speed, ETA, and VCR deck logs.
- **Direct Browser Cookie Extraction:** Choose between Auto-Detect, Chrome Browser (`--cookies-from-browser chrome`), Firefox Browser, or No Cookies directly from the VCR Web UI dropdown.
- **Interactive Quality Selection:** Choose video quality profiles interactively (`720p HD`, `1080p Full HD`, `4K Ultra HD`) before downloading in CLI mode.
- **Vintage VHS Web UI:** A carefully crafted VCR player aesthetic featuring a magnetic tape header, spinning tape reels, LED tape counter, CRT scanlines, olive green & red branding, and server power shutdown control (`/api/exit`).
- **Comprehensive Error Handler:** Advanced error analyzer detecting and alerting on:
  - No internet connection / network unreachable
  - DNS resolution failures
  - VPN connection & proxy issues
  - YouTube IP bans / rate limits (HTTP 429 / bot detection) & geo-restrictions
- **Cosmetic Makefile Command Center:** Rich-colored Makefile targets for building, testing, diagnostics, updating, and running.
- **Secure by Design:** The web server binds *only* to `127.0.0.1`. It is completely invisible to your local network and the internet.
- **OmniCharge Master Controller:** Full supervision featuring state machines (`IDLE`, `RUNNING`, `PAUSED`, `ERROR`, `TERMINATING`), strict path sandboxing, health watchdogs with backoff/retry, and Termux memory governance.
- **Smart Output:** Automatically registers downloaded files with Android's MediaStore so they appear instantly in your Gallery.
- **Zero External Dependencies:** The web UI is compiled *into* the Go binary using `go:embed`.

---

## 🚀 Installation & Setup

### Prerequisites
You must have Termux, `yt-dlp`, and `ffmpeg` installed.
```bash
pkg update && pkg upgrade
pkg install golang yt-dlp ffmpeg make
```

### Build and Run with Makefile
We provide a rich cosmetic `Makefile` for all tasks:
```bash
# Display help and available targets
make help

# Run system diagnostics (Go version, vet, tests, yt-dlp check)
make diag

# Build the binary
make build

# Run unit tests
make test

# Update yt-dlp and Go dependencies
make update

# Run CLI downloader
make run

# Start local VCR web server mode
make server
```

---

🕹️ Usage

### 1. Terminal CLI Mode
```bash
yt-termux
```
Paste your URL when prompted, select your desired video quality profile from the interactive menu, and watch the live animated progress bar.

### 2. Browser Web UI Mode (`Zilla-Rec`)
```bash
yt-termux web
```
Open `http://127.0.0.1:8080` in Chrome or Firefox on your Android device. You can select your cookie source (Auto-Detect, Chrome Browser cookies, Firefox, or None) and click **POWER ⏻** to shutdown the server directly from the VCR UI.

⚠️ **Security Note:** The web server binds to loopback only (`127.0.0.1`). It cannot be accessed by other devices on your Wi-Fi.

---

🛠️ Architecture & Layout

| File / Directory | Purpose |
| :--- | :--- |
| `omnicharge/` | OmniCharge Master Controller supervising app lifecycle, health watchdog, path security sandboxing, and Termux memory governance. |
| `main.go` | CLI entrypoint, quality profile prompt, and subcommand routing. |
| `web/server.go` | Web server entrypoint embedding `web/index.html` and serving SSE API. |
| `web/index.html` | The vintage VHS VCR web interface with magnetic tape header, spinning reels, and SSE progress bar. |
| `downloader/engine.go` | Core `yt-dlp` wrapper supporting `--cookies-from-browser` and progress parsing. |
| `downloader/errors.go` | Comprehensive network, DNS, VPN/proxy, and YouTube IP ban error analyzer. |
| `downloader/ui.go` | Terminal rendering, ANSI color management, and TTY detection. |
| `downloader/ui_progress.go` | In-place animated status line and progress bar renderer. |
| `downloader/ui_prompt.go` | Interactive quality profile selection prompt. |
| `downloader/media.go` | Integration with Android's MediaStore. |
| `config/` & `storage/` | Configuration settings, quality profiles, and path resolution. |

---

<div align="center">
  <i>"Bridging the gap between vintage analog recording and modern digital streams."</i><br>
  © ZILLA-REC® SYSTEMS
</div>
