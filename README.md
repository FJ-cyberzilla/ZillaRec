<div align="center">

# FJ™ CYBERTRONIC SYSTEMS
### YT-TERMUX // High-Performance Video Downloader

[![Go Version](https://img.shields.io/badge/Go-1.22+-00ADD8?style=for-the-badge&logo=go)](https://go.dev/)
[![Platform](https://img.shields.io/badge/Platform-Termux%20%7C%20Android-black?style=for-the-badge&logo=android)](https://termux.dev/)
[![UI](https://img.shields.io/badge/UI-1986%20Terminal%20Retro-33ff00?style=for-the-badge)]()

**A sleek, robust, and secure local CLI & Web interface for downloading YouTube videos, engineered specifically for Termux on Android.**

</div>

---

## ⚡ Features

- **Dual Interface:** Run it directly in the terminal (CLI) or spin up a local web server to use via your mobile browser.
- **Retro Web UI:** A carefully crafted, pure-CSS "1986 Cybertron" terminal interface with CRT scanlines and glow effects.
- **Secure by Design:** The web server binds *only* to `127.0.0.1`. It is completely invisible to your local network and the internet.
- **Robust Engine:** Uses `yt-dlp` with `--progress-template` for exact, reliable progress parsing. Kills orphaned processes on `Ctrl+C` using process groups.
- **Smart Output:** Automatically registers downloaded files with Android's MediaStore so they appear instantly in your Gallery.
- **Zero External Dependencies:** The web UI is compiled *into* the Go binary using `go:embed`. No Node.js, no React, no NPM.

---

## 🚀 Installation

### Prerequisites
You must have Termux, `yt-dlp`, and `ffmpeg` installed.
```bash
pkg update && pkg upgrade
pkg install golang yt-dlp ffmpeg

Build from Source

```bash
git clone https://github.com/yourusername/yt-termux.git
cd yt-termux

# Build the CLI version
go build -o bin/yt-downloader main.go

# Build the Web Server version
go build -o bin/yt-web server.go
```

---

🕹️ Usage

1. The Terminal (CLI)

The easiest way to download. Just run the binary and paste your link.

```bash
./bin/yt-downloader
```

You will see the gradient ASCII banner. Paste your URL and hit Enter.

2. The Browser (Web UI)

Start the local web server:

```bash
./bin/yt-web
```

Open http://127.0.0.1:8080 in Chrome or Firefox on your Android device.

⚠️ Security Note: The web server is bound to loopback only (127.0.0.1). It cannot be accessed by other devices on your Wi-Fi, guaranteeing your downloads and cookies remain private.

---

🛠️ Architecture

File / Directory Purpose
main.go CLI entrypoint. Displays the gradient ASCII banner and prompts for input.
server.go Web server entrypoint. Embeds web/index.html and serves the API.
web/index.html The 1986 retro web interface (CSS/JS).
downloader/engine.go Core yt-dlp wrapper. Handles process groups, stderr capture, and output parsing.
downloader/ui.go Terminal rendering, ANSI color management, and TTY detection.
downloader/media.go Integration with Android's MediaStore.
config/ & storage/ Path resolution and default quality profiles.

---

📸 

"FJ™ Cybertronic Systems" - The Retro Web UI

---

🔒 Security

This tool is designed for personal, single-user environments.

· Command Injection Safe: URLs are passed as isolated arguments to exec.Command, preventing shell injection.
· No Cloud Processing: Everything runs 100% locally on your device.
· Keep your cookies.txt file private and never commit it to GitHub.

<div align="center">
  <i>"Bridging the gap between 1986 and 2026."</i><br>
  © FJ Cyberzilla
</div>
```
