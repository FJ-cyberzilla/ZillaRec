# Architecture, Performance & Code Quality Recommendations

This document outlines comprehensive architectural recommendations and best practices implemented across the `yt-termux` project, evaluated through four core engineering lenses:
1. Go Project Layout & Packaging
2. Concurrency & Performance
3. Thermo-Nuclear Code Quality & Abstraction
4. Correctness & Robustness

---

## 1. Go Project Layout & Packaging
- **Domain-Driven Separation:** The codebase cleanly separates concerns into dedicated packages:
  - `downloader`: Subprocess execution, media parsing, UI rendering, and worker pool orchestration.
  - `omnicharge`: Security sandboxing, checksum verification, memory tracking, and master control coordination.
  - `storage`: Secure path resolution and workspace directory management.
  - `config`: Persistent application settings and validation.
  - `web`: Embedded HTTP server and Server-Sent Events (SSE) telemetry.
- **Dependency Direction:** Dependencies flow cleanly inwards and downwards, avoiding circular imports.

## 2. Concurrency & Performance
- **Lifecycle Management:** Background worker pools and HTTP servers utilize `context.WithCancel` and `sync.WaitGroup` to ensure clean shutdown paths without goroutine leaks.
- **Lock-Free State Tracking:** High-frequency metrics (active job counts, state flags) leverage `atomic.Int32` and `atomic.Bool` to eliminate lock contention on hot paths.
- **Resource Pooling & Limits:** Worker pools restrict concurrent heavy yt-dlp subprocesses, protecting system memory and CPU headroom on mobile/Termux environments.

## 3. Thermo-Nuclear Code Quality & Abstraction
- **Modular Function Decomposition:** Monolithic extraction loops (such as subprocess stdout line processing) have been decomposed into pure functions (e.g., `handleLine`, `parseProgressTemplate`) enabling comprehensive unit test coverage.
- **Single Responsibility Principle:** Separation of UI rendering (`ui.go`, `ui_progress.go`), process management (`proc_unix.go`, `proc_windows.go`), and engine coordination (`engine.go`).

## 4. Correctness & Robustness
- **IPC Protocol Compatibility:** Robust parsing of worker.py / yt-dlp output streams handling progress (`download:`), destination paths (`Destination:`), merging (`[Merger]`), and error states.
- **Security Sandboxing:** Strict path sandboxing in `storage` and `omnicharge/security.go` to prevent directory traversal vulnerabilities.
- **Exhaustive Test Coverage:** Unit tests validating progress parsing, path deduplication, subprocess line handling, and secure path resolution.
