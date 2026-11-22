#!/usr/bin/env python3
import sys
import os
import subprocess
import argparse

def main():
    parser = argparse.ArgumentParser(description="Zilla-Rec Python yt-dlp Worker")
    parser.add_argument("url", help="Target media URL")
    parser.add_argument("--format", default="bestvideo+bestaudio/best", help="Format specification")
    parser.add_argument("--output-dir", default=".", help="Output directory")
    parser.add_argument("--cookies", default="", help="Path to cookies file or browser name")
    args = parser.parse_args()

    if not args.url.strip():
        print("ERROR\tNo URL provided", flush=True)
        sys.exit(1)

    os.makedirs(args.output_dir, exist_ok=True)

    ytdlp_args = [
        "yt-dlp",
        args.url,
        "-f", args.format,
        "--force-ipv4",
        "--newline",
        "--no-warnings",
        "--progress-template", "download:%(progress._percent_str)s\t%(progress._speed_str)s\t%(progress._eta_str)s\t%(title)s\t%(duration_string)s",
        "--socket-timeout", "15",
        "--retries", "3",
        "--fragment-retries", "3",
        "-P", args.output_dir,
        "-o", "%(title)s.%(ext)s",
        "--restrict-filenames",
        "--merge-output-format", "mp4"
    ]

    if args.cookies:
        if args.cookies in ("chrome", "firefox"):
            ytdlp_args.extend(["--cookies-from-browser", args.cookies])
        elif os.path.exists(args.cookies):
            ytdlp_args.extend(["--cookies", args.cookies])

    try:
        proc = subprocess.Popen(
            ytdlp_args,
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
            text=True,
            bufsize=1
        )

        def read_stderr():
            for line in proc.stderr:
                stripped = line.strip()
                if stripped:
                    print(f"LOG\t{stripped}", flush=True)

        import threading
        t = threading.Thread(target=read_stderr, daemon=True)
        t.start()

        for line in proc.stdout:
            line = line.strip()
            if not line:
                continue
            if line.startswith("download:"):
                # Pass through tab-delimited progress event
                print(line, flush=True)
            elif "Destination:" in line or "Merging formats" in line:
                print(f"DEST\t{line}", flush=True)
            else:
                print(f"LOG\t{line}", flush=True)

        proc.wait()
        if proc.returncode != 0:
            print(f"ERROR\tyt-dlp exited with code {proc.returncode}", flush=True)
            sys.exit(proc.returncode)
        else:
            print("SUCCESS\tDownload completed", flush=True)

    except Exception as e:
        print(f"ERROR\t{str(e)}", flush=True)
        sys.exit(1)

if __name__ == "__main__":
    main()
