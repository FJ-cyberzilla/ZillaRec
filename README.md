<div align="center">

# ZILLA-REC®
## Universal Media Capture Deck

[![Go Version](https://img.shields.io/badge/Go-1.22+-00ADD8?style=for-the-badge&logo=go)](https://go.dev/)
[![Platform](https://img.shields.io/badge/Platform-Termux%20%7C%20Android%20%7C%20Linux%20%7C%20WSL2%20%7C%20macOS%20%7C%20Windows-black?style=for-the-badge&logo=android)](https://termux.dev/)
[![UI](https://img.shields.io/badge/UI-Vintage%20VHS%20VCR-39ff14?style=for-the-badge)]()
[![License](https://img.shields.io/badge/License-MIT-d97736?style=for-the-badge)]()

**A sleek, robust, and secure local CLI & Vintage VHS Web interface. Back in the day, a VCR captured anything you plugged into it. Today, Zilla-Rec captures any stream you paste into it. Powered by FJ-cyberzilla engineered specifically for Termux on Android with advanced cross-platform memory governance and health watchcats.**

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
- **Direct Browser Cookie Extraction:** Choose between Auto-Detect, Chrome Browser (`--cookies-from-browser chrome`), Firefox Browser, or No Cookies directly from the VCR Web UI dropdown (essential for locked/private streams) with rigorous path sandboxing.
- **Interactive Quality Selection:** Choose video quality profiles interactively (`720p HD`, `1080p Full HD`, `4K Ultra HD`) before downloading in CLI mode.
- **OmniCharge Master Controller:** Full supervision featuring:
  - **State Machine:** (`IDLE`, `RUNNING`, `PAUSED`, `ERROR`, `TERMINATING`).
  - **Smart Pause and Resume:** Track downloaded byte offsets and operational state so users can pause a file, switch networks, or recover from dropped connections without restarting from zero.
  - **Cross-Platform Memory Governor:** Automatic platform detection (`DetectPlatform` for Android/Termux, WSL2, Linux, Windows, macOS) with configurable soft memory limits (`debug.SetMemoryLimit`), cooldown timers, and `FreeMemory`/`ForceFreeMemory` APIs.
  - **Feline-Vigilant Health Catwatch & Background Watcher:** `StartWatcher` background goroutine monitoring heap usage against soft limits with automatic RAM reclamation, exponential backoff with jitter, and bit-shift overflow protection (`math/rand/v2`).
  - **SHA-256 Checksum & Sidecars:** Automatic SHA-256 computation and `.sha256` sidecar file generation upon download completion.
  - **Strict Path Sandboxing:** Secures the base directory against traversal.
- **Comprehensive Error Handler:** Advanced error analyzer detecting and alerting on:
  - No internet connection / network unreachable
  - DNS resolution failures
  - VPN connection & proxy issues
  - YouTube IP bans / rate limits (HTTP 429 / bot detection) & geo-restrictions
- **Cross-Platform Terminal Compatibility:** Build-tagged non-Windows terminal stubs (`downloader/ui_nonwindows.go`) ensuring seamless compilation and operation across Linux, Android/Termux, WSL2, and macOS.
- **Graceful Server Shutdown:** Robust HTTP server lifecycle management with `srv.Shutdown`, OS signal trapping (`SIGINT`, `SIGTERM`), and active download tracking.
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

Build from source

```bash
git clone https://github.com/FJ-cyberzilla/ZillaRec.git
cd ZillaRec

# Display the command center help menu
make help

# Run system diagnostics (Go version, vet, tests, yt-dlp check)
make diag

# Build the native binary → bin/zillarec
make build

# Run all unit tests
make test

# Update yt-dlp and Go dependencies
make update

# Run CLI downloader
make run

# Start local VCR web server mode
make server

# Clean build artifacts
make clean
```

Cross-platform builds

```bash
make build-linux      # → bin/zillarec-linux-amd64  (static, x86-64)
make build-android    # → bin/zillarec-android-arm64
make build-darwin     # → bin/zillarec-darwin-arm64
make build-windows    # → bin/zillarec-windows-amd64.exe
make build-all        # → all platforms
```

Release

```bash
# Generate checksums for all platform binaries
make checksums

# Upload binaries + checksums to a GitHub Release
make release
```

make help output:

```text
======================================================================
  ZILLA-REC(R) VCR MASTER CONTROLLER & COMMAND CENTER
======================================================================
  Available Makefile Targets:
    make build          - Build native zillarec (this machine)
    make build-linux    - Static Linux x86-64 binary
    make build-android  - Android/Termux arm64 binary
    make build-darwin   - macOS arm64 binary
    make build-windows  - Windows x86-64 .exe
    make build-all      - All platform binaries
    make install-wsl    - Run PowerShell suite to deploy into WSL2
    make test           - Run all unit tests (verbose)
    make diag           - Diagnostics (go vet, tests, yt-dlp check)
    make update         - Update yt-dlp and Go dependencies
    make run            - Run the CLI downloader
    make server         - Start the Zilla-Rec web server
    make checksums      - SHA-256 files for all binaries in bin/
    make release        - Upload linux+android binaries to GitHub Release
    make clean          - Remove build artifacts
======================================================================
```

---

🕹️ Usage

1. Terminal CLI Mode (Local Capture)

```bash
make run
# OR
./bin/zillarec
```

Paste your URL when prompted, select your desired video quality profile from the interactive menu, and watch the live animated progress bar.

2. Browser Web UI Mode (Zilla-Rec)

```bash
make server
```

Open http://127.0.0.1:8080 in Chrome or Firefox on your Android device. Paste any stream URL, select your cookie source (Auto-Detect, Chrome Browser cookies, Firefox, or None), and click REC ●. Press POWER ⏻ to gracefully shut down the OmniCharge server directly from the VCR UI.

🦺 Safty Note: The web server binds to loopback ONLY (127.0.0.1). It CANNOT be accessed by other devices on your Wi-Fi.
---

🛠️ Architecture & Layout

```text
ZillaRec/
├── .github/workflows/
│   └── makefile.yml         # CI: build+test on push, publish release on tag
├── .gitignore
├── LICENSE
├── Makefile                 # Cosmetic Command Center + release automation
├── README.md
├── go.mod
├── go.sum
├── main.go                  # CLI entrypoint & routing
├── install.ps1              # Optional WSL2 deployment helper
├── config/                  # Settings, quality profiles
├── downloader/              # Core yt-dlp engine, error analyzer, CLI UI
│   ├── engine.go            # yt-dlp wrapper (cookies, progress parsing)
│   ├── errors.go            # Network, DNS, VPN, IP ban analyzer
│   ├── media.go             # Android MediaStore integration
│   ├── ui.go                # ANSI color & TTY management
│   ├── ui_nonwindows.go     # Non-Windows terminal stubs (build-tagged)
│   ├── ui_progress.go       # Animated CLI progress bar
│   └── ui_prompt.go         # Interactive quality profile prompt
├── downloads/               # Default output directory (git-ignored)
├── omnicharge/              # Master Controller (State, Health, Memory, Security)
│   ├── controller.go        # State machine & lifecycle
│   ├── health.go            # Watchdog, background memory watcher & retries
│   ├── memory.go            # Cross-platform memory governor & limits
│   ├── security.go          # Path sandboxing
│   └── checksum.go          # SHA-256 sidecar generation
├── storage/                 # Path resolution
├── web/                     # Web Server & Vintage VCR UI
│   ├── index.html           # VCR interface (embedded via go:embed)
│   └── server.go            # SSE API, graceful shutdown, download handlers
└── bin/                     # Build output (git-ignored)
```

🔌 API Endpoints

| Endpoint | Method | Description |
|---|---|---|
| `/` | GET | Serves the embedded index.html VCR UI. |
| `/api/download` | POST | Accepts `{url, cookieSource, quality}` and streams SSE progress. |
| `/api/status` | GET | Returns the OmniCharge ControllerSnapshot (State, Memory, Health, Uptime). |
| `/api/exit` | POST | Triggers graceful shutdown of the MasterController and HTTP server. |


🧪 Testing

```bash
make test          # verbose unit tests
make diag          # go vet + tests + yt-dlp presence check
```

CI runs both on every push and pull request via .github/workflows/makefile.yml.
---

<div align="center">
  <i>"Back in the day, a VCR captured anything you plugged into it. Today, we capture any stream you paste into it."</i><br>
  
  <b>© ZILLA-REC® --- FJ™ cybertronic systems --- MMXXVI --- Autumn </b>
</div>
