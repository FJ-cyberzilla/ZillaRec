<div align="center">

# ZILLA-REC®
### VCR MASTER CONTROLLER // Universal Media Capture Deck

[![Go Version](https://img.shields.io/badge/Go-1.22+-00ADD8?style=for-the-badge&logo=go)](https://go.dev/)
[![Platform](https://img.shields.io/badge/Platform-Termux%20%7C%20Android-black?style=for-the-badge&logo=android)](https://termux.dev/)
[![UI](https://img.shields.io/badge/UI-Vintage%20VHS%20VCR-39ff14?style=for-the-badge)]()
[![License](https://img.shields.io/badge/License-MIT-d97736?style=for-the-badge)]()

**A sleek, robust, and secure local CLI & Vintage VHS Web interface. Back in the day, a VCR captured anything you plugged into it. Today, Zilla-Rec captures any stream you paste into it. Powered by `yt-dlp`, engineered specifically for Termux on Android.**

</div>

---

## ⚡ Features

- **Universal Capture Deck:** Just like a VCR's `LINE IN` ports, paste a URL from YouTube, Twitter/X, TikTok, Instagram, Twitch, Vimeo, Facebook, or 1000+ other supported platforms, and the deck records it.
- **Dual Interface:** Run it directly in the terminal (CLI) or start the vintage VCR web server (`make server`) to use via your mobile browser.
- **Vintage VHS Web UI:** A meticulously crafted VCR aesthetic featuring:
  - **Transparent VHS Watermark Background** for a physical tape feel.
  - **Zilla Green & Rec Red Branding** with a live, blinking recording indicator dot.
  - **Dynamic Spinning Reels** that only activate when a download begins.
  - **CRT Scanlines & LED Displays** (amber timer, green status lights).
  - **Vintage Orange Shutdown** styling for the `POWER ⏻` button and footer.
- **Real-Time Streaming Progress (SSE):** Web UI streams live progress updates via Server-Sent Events, powering a retro progress bar with real-time percentage, speed, ETA, and VCR deck logs.
- **Direct Browser Cookie Extraction:** Choose between Auto-Detect, Chrome Browser (`--cookies-from-browser chrome`), Firefox Browser, or No Cookies directly from the VCR Web UI dropdown (essential for locked/private streams).
- **Interactive Quality Selection:** Choose video quality profiles interactively (`720p HD`, `1080p Full HD`, `4K Ultra HD`) before downloading in CLI mode.
- **OmniCharge Master Controller:** Full supervision featuring:
  - **State Machine:** (`IDLE`, `RUNNING`, `PAUSED`, `ERROR`, `TERMINATING`).
  - **Smart Pause and Resume:** Track downloaded byte offsets and operational state so users can pause a file, switch networks, or recover from dropped connections without restarting from zero.
  - **Dynamic Link Refreshing:** Allow updating broken or expired source URLs for active/paused downloads while keeping partially downloaded data intact.
  - **Adaptive Error Handling:** Automatically retry failed chunks with exponential backoff (`HealthCatwatch`) when a server drops a connection or rate-limits requests.
  - **SHA-256 Checksum & Sidecars:** Automatic SHA-256 computation and `.sha256` sidecar file generation upon download completion.
  - **Health Watchdog:** Monitors system health and network recovery.
  - **Memory Governor:** Termux-specific memory usage tracking (`/api/status`).
  - **Strict Path Sandboxing:** Secures the base directory against traversal.
- **Comprehensive Error Handler:** Advanced error analyzer detecting and alerting on:
  - No internet connection / network unreachable
  - DNS resolution failures
  - VPN connection & proxy issues
  - YouTube IP bans / rate limits (HTTP 429 / bot detection) & geo-restrictions
- **Cosmetic Makefile Command Center:** Rich-colored Makefile targets for building, testing, diagnostics, updating, and running.
- **Secure by Design:** The web server binds *only* to `127.0.0.1`. It is completely invisible to your local network and the internet.
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

Build and Run with the Makefile

We provide a rich cosmetic Makefile acting as your command center.

```bash
# Display the command center help menu
make help

# Run system diagnostics (Go version, vet, tests, yt-dlp check)
make diag

# Build the binary to ./zillarec
make build

# Run all unit tests with verbose output
make test

# Update yt-dlp and Go dependencies
make update

# Run CLI downloader
make run

# Start local VCR web server mode
make server

# Clean build artifacts and temporary files
make clean
```

Output of make help:

```text
======================================================================
  ZILLA-REC(R) VCR MASTER CONTROLLER & COMMAND CENTER
======================================================================
  Available Makefile Targets:
    make build   - Build the zillarec binary
    make test    - Run all unit tests with verbose output
    make diag    - Run diagnostics (go vet, tests, yt-dlp check)
    make update  - Update yt-dlp and Go dependencies
    make run     - Run the CLI downloader application
    make server  - Start the Zilla-Rec web server interface
    make clean   - Clean build artifacts and temporary files
======================================================================
```

---

🕹️ Usage

1. Terminal CLI Mode (Local Capture)

```bash
make run
# OR
./zillarec
```

Paste your URL when prompted, select your desired video quality profile from the interactive menu, and watch the live animated progress bar.

2. Browser Web UI Mode (Zilla-Rec)

```bash
make server
```

Open http://127.0.0.1:8080 in Chrome or Firefox on your Android device. Paste any stream URL, select your cookie source (Auto-Detect, Chrome Browser cookies, Firefox, or None), and click REC ●. Press POWER ⏻ to gracefully shut down the OmniCharge server directly from the VCR UI.

⚠️ Security Note: The web server binds to loopback only (127.0.0.1). It cannot be accessed by other devices on your Wi-Fi.

---

🛠️ Architecture & Layout

```text
yt-termux/
├── Makefile                # Cosmetic Command Center
├── README.md
├── bin/                    # Build output (yt-downloader)
├── zillarec                # Compiled binary
├── config/                 # Settings, quality profiles
├── downloader/             # Core yt-dlp engine, error analyzer, CLI UI
│   ├── engine.go           # yt-dlp wrapper (cookies, progress parsing)
│   ├── errors.go           # Network, DNS, VPN, IP ban analyzer
│   ├── media.go            # Android MediaStore integration
│   ├── ui.go               # ANSI color & TTY management
│   ├── ui_progress.go      # Animated CLI progress bar
│   └── ui_prompt.go        # Interactive quality profile prompt
├── downloads/              # Default output directory
├── omnicharge/             # Master Controller (State, Health, Memory, Security)
│   ├── controller.go       # State machine & lifecycle
│   ├── health.go           # Watchdog & retries
│   ├── memory.go           # Termux memory governor
│   └── security.go         # Path sandboxing
├── storage/                # Path resolution
├── web/                    # Web Server & Vintage VCR UI
│   ├── index.html          # VCR interface (embedded via go:embed)
│   └── server.go           # SSE API, /api/download, /api/status, /api/exit
└── main.go                 # CLI entrypoint & routing
```

🔌 API Endpoints

Endpoint Method Description
/ GET Serves the embedded index.html VCR UI.
/api/download POST Accepts {url, cookieSource} and streams SSE progress.
/api/status GET Returns the OmniCharge ControllerSnapshot (State, Memory, Health, Uptime).
/api/exit POST Triggers graceful shutdown of the MasterController.

---

<div align="center">
  <i>"Back in the day, a VCR captured anything you plugged into it. Today, we capture any stream you paste into it."</i><br>
  <b>© ZILLA-REC® SYSTEMS</b>
</div>
```
