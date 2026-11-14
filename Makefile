# ==============================================================================
#  ZILLA-REC® VCR MASTER CONTROLLER - MAKEFILE
# ==============================================================================

SHELL       := /bin/bash
BINARY_NAME := zillarec
VERSION     := 1.0.2
BIN_DIR     := bin
GO          ?= go

# Strip symbols + DWARF for smaller release binaries
LDFLAGS := -s -w

# ANSI Color Codes (Vintage Olive & Red Theme)
CYAN    := \033[1;36m
GREEN   := \033[1;32m
OLIVE   := \033[1;33m
RED     := \033[1;31m
MAGENTA := \033[1;35m
RESET   := \033[0m
BOLD    := \033[1m

.PHONY: all help build build-native build-linux build-android build-darwin \
        build-windows build-all install-wsl test diag update run server \
        checksums release clean

all: help

# ------------------------------------------------------------------------------
#  HELP
# ------------------------------------------------------------------------------
help:
	@printf "$(CYAN)======================================================================$(RESET)\n"
	@printf "$(OLIVE)  ZILLA-REC(R) VCR MASTER CONTROLLER & COMMAND CENTER$(RESET)\n"
	@printf "$(CYAN)======================================================================$(RESET)\n"
	@printf "$(GREEN)  Available Makefile Targets:$(RESET)\n"
	@printf "    $(OLIVE)make build$(RESET)          - Build native $(BINARY_NAME) (this machine)\n"
	@printf "    $(OLIVE)make build-linux$(RESET)    - Static Linux x86-64 binary\n"
	@printf "    $(OLIVE)make build-android$(RESET)  - Android/Termux arm64 binary\n"
	@printf "    $(OLIVE)make build-darwin$(RESET)   - macOS arm64 binary\n"
	@printf "    $(OLIVE)make build-windows$(RESET)  - Windows x86-64 .exe\n"
	@printf "    $(OLIVE)make build-all$(RESET)      - All platform binaries\n"
	@printf "    $(OLIVE)make install-wsl$(RESET)    - Run PowerShell suite to deploy into WSL2\n"
	@printf "    $(OLIVE)make test$(RESET)           - Run all unit tests (verbose)\n"
	@printf "    $(OLIVE)make diag$(RESET)           - Diagnostics (go vet, tests, yt-dlp check)\n"
	@printf "    $(OLIVE)make update$(RESET)         - Update yt-dlp and Go dependencies\n"
	@printf "    $(OLIVE)make run$(RESET)            - Run the CLI downloader\n"
	@printf "    $(OLIVE)make server$(RESET)         - Start the Zilla-Rec web server\n"
	@printf "    $(OLIVE)make checksums$(RESET)      - SHA-256 files for all binaries in $(BIN_DIR)/\n"
	@printf "    $(OLIVE)make release$(RESET)        - Upload linux+android binaries to GitHub Release v$(VERSION)\n"
	@printf "    $(OLIVE)make clean$(RESET)          - Remove build artifacts\n"
	@printf "$(CYAN)======================================================================$(RESET)\n"

# ------------------------------------------------------------------------------
#  BUILD
# ------------------------------------------------------------------------------
$(BIN_DIR):
	@mkdir -p $(BIN_DIR)

build: $(BIN_DIR)
	@printf "$(OLIVE)[BUILD]$(RESET) Compiling native $(BINARY_NAME)...\n"
	@$(GO) build -v -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/$(BINARY_NAME) .
	@printf "$(GREEN)[SUCCESS]$(RESET) $(BOLD)$(BIN_DIR)/$(BINARY_NAME)$(RESET)\n"

build-native: build

build-linux: $(BIN_DIR)
	@printf "$(OLIVE)[BUILD-LINUX]$(RESET) Static Linux x86-64 binary...\n"
	@CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
		$(GO) build -v -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/$(BINARY_NAME)-linux-amd64 .
	@printf "$(GREEN)[SUCCESS]$(RESET) $(BOLD)$(BIN_DIR)/$(BINARY_NAME)-linux-amd64$(RESET)\n"

build-android: $(BIN_DIR)
	@printf "$(OLIVE)[BUILD-ANDROID]$(RESET) Android/Termux arm64 binary...\n"
	@CGO_ENABLED=0 GOOS=android GOARCH=arm64 \
		$(GO) build -v -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/$(BINARY_NAME)-android-arm64 .
	@printf "$(GREEN)[SUCCESS]$(RESET) $(BOLD)$(BIN_DIR)/$(BINARY_NAME)-android-arm64$(RESET)\n"

build-darwin: $(BIN_DIR)
	@printf "$(OLIVE)[BUILD-DARWIN]$(RESET) macOS arm64 binary...\n"
	@CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 \
		$(GO) build -v -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/$(BINARY_NAME)-darwin-arm64 .
	@printf "$(GREEN)[SUCCESS]$(RESET) $(BOLD)$(BIN_DIR)/$(BINARY_NAME)-darwin-arm64$(RESET)\n"

build-windows: $(BIN_DIR)
	@printf "$(OLIVE)[BUILD-WINDOWS]$(RESET) Windows x86-64 binary...\n"
	@CGO_ENABLED=0 GOOS=windows GOARCH=amd64 \
		$(GO) build -v -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/$(BINARY_NAME)-windows-amd64.exe .
	@printf "$(GREEN)[SUCCESS]$(RESET) $(BOLD)$(BIN_DIR)/$(BINARY_NAME)-windows-amd64.exe$(RESET)\n"

build-all: build-linux build-android build-darwin build-windows
	@printf "$(GREEN)[BUILD-ALL COMPLETE]$(RESET) All platform binaries in $(BOLD)$(BIN_DIR)/$(RESET)\n"

# ------------------------------------------------------------------------------
#  DEPLOY / INSTALL
# ------------------------------------------------------------------------------
install-wsl: build-linux
	@printf "$(OLIVE)[INSTALL-WSL]$(RESET) Triggering PowerShell deployment suite...\n"
	@powershell -ExecutionPolicy Bypass -File install.ps1

# ------------------------------------------------------------------------------
#  TEST / DIAG
# ------------------------------------------------------------------------------
test:
	@printf "$(OLIVE)[TEST]$(RESET) Running unit test suite...\n"
	@$(GO) test -v ./...

diag:
	@printf "$(OLIVE)[DIAG]$(RESET) Running comprehensive system diagnostics...\n"
	@printf "  1. Go version...\n"
	@$(GO) version
	@printf "  2. go vet...\n"
	@$(GO) vet ./...
	@printf "  3. Unit tests...\n"
	@$(GO) test -v ./...
	@printf "  4. yt-dlp in PATH...\n"
	@if command -v yt-dlp &> /dev/null; then \
		printf "     $(GREEN)✔ yt-dlp found ($$(command -v yt-dlp))$(RESET)\n"; \
	else \
		printf "     $(RED)✖ yt-dlp not found in PATH! (Install via pip/pkg)$(RESET)\n"; \
	fi
	@printf "$(GREEN)[DIAG COMPLETE]$(RESET)\n"

# ------------------------------------------------------------------------------
#  UPDATE
# ------------------------------------------------------------------------------
update:
	@printf "$(OLIVE)[UPDATE]$(RESET) Updating yt-dlp and Go dependencies...\n"
	@if command -v pip3 &> /dev/null; then \
		pip3 install --upgrade yt-dlp || pip install --upgrade yt-dlp; \
	elif command -v pip &> /dev/null; then \
		pip install --upgrade yt-dlp; \
	elif command -v pkg &> /dev/null; then \
		pkg update && pkg upgrade yt-dlp; \
	else \
		printf "$(RED)[WARNING]$(RESET) No package manager found for updating yt-dlp.\n"; \
	fi
	@printf "  Updating Go modules...\n"
	@$(GO) get -u ./...
	@$(GO) mod tidy
	@printf "$(GREEN)[UPDATE COMPLETE]$(RESET)\n"

# ------------------------------------------------------------------------------
#  RUN
# ------------------------------------------------------------------------------
run: build
	@printf "$(OLIVE)[RUN]$(RESET) Starting Zilla-Rec CLI...\n"
	@./$(BIN_DIR)/$(BINARY_NAME)

server: build
	@printf "$(OLIVE)[SERVER]$(RESET) Starting Zilla-Rec web server...\n"
	@./$(BIN_DIR)/$(BINARY_NAME) web

# ------------------------------------------------------------------------------
#  RELEASE
# ------------------------------------------------------------------------------
checksums:
	@printf "$(OLIVE)[CHECKSUMS]$(RESET) Generating SHA-256 sidecars...\n"
	@cd $(BIN_DIR) && for f in $(BINARY_NAME)-*; do \
		case "$$f" in \
			*.sha256) continue ;; \
		esac; \
		sha256sum "$$f" > "$$f.sha256" && \
		printf "  $(GREEN)✔$(RESET) $$f.sha256\n"; \
	done; true

release: build-linux build-android checksums
	@printf "$(OLIVE)[RELEASE]$(RESET) Uploading to GitHub Release v$(VERSION)...\n"
	@gh release view v$(VERSION) >/dev/null 2>&1 || \
		gh release create v$(VERSION) --title "$(BINARY_NAME) v$(VERSION)" --notes "Release v$(VERSION)"
	@gh release upload v$(VERSION) \
		$(BIN_DIR)/$(BINARY_NAME)-linux-amd64 \
		$(BIN_DIR)/$(BINARY_NAME)-linux-amd64.sha256 \
		$(BIN_DIR)/$(BINARY_NAME)-android-arm64 \
		$(BIN_DIR)/$(BINARY_NAME)-android-arm64.sha256 \
		--clobber
	@printf "$(GREEN)[RELEASE COMPLETE]$(RESET) https://github.com/FJ-cyberzilla/ZillaRec/releases/tag/v$(VERSION)\n"

# ------------------------------------------------------------------------------
#  CLEAN
# ------------------------------------------------------------------------------
clean:
	@printf "$(RED)[CLEAN]$(RESET) Removing build artifacts...\n"
	@rm -rf $(BIN_DIR)
	@$(GO) clean
	@printf "$(GREEN)[CLEAN COMPLETE]$(RESET)\n"
