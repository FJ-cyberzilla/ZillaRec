# ==============================================================================
#  ZILLA-REC® VCR MASTER CONTROLLER - MAKEFILE
# ==============================================================================

SHELL := /bin/bash
BINARY_NAME := zillarec
VERSION := 1.0.0

# ANSI Color Codes (Vintage Olive & Red Theme)
CYAN   := \033[1;36m
GREEN  := \033[1;32m
OLIVE  := \033[1;33m
RED    := \033[1;31m
MAGENTA:= \033[1;35m
RESET  := \033[0m
BOLD   := \033[1m

.PHONY: all help build build-linux install-wsl test diag update run server clean

all: help

help:
	@printf "$(CYAN)======================================================================$(RESET)\n"
	@printf "$(OLIVE)  ZILLA-REC(R) VCR MASTER CONTROLLER & COMMAND CENTER$(RESET)\n"
	@printf "$(CYAN)======================================================================$(RESET)\n"
	@printf "$(GREEN)  Available Makefile Targets:$(RESET)\n"
	@printf "    $(OLIVE)make build$(RESET)        - Build the native $(BINARY_NAME) binary\n"
	@printf "    $(OLIVE)make build-linux$(RESET)  - Cross-compile for Linux / WSL2\n"
	@printf "    $(OLIVE)make install-wsl$(RESET)  - Run PowerShell suite to deploy into WSL2\n"
	@printf "    $(OLIVE)make test$(RESET)         - Run all unit tests with verbose output\n"
	@printf "    $(OLIVE)make diag$(RESET)         - Run diagnostics (go vet, tests, yt-dlp check)\n"
	@printf "    $(OLIVE)make update$(RESET)       - Update yt-dlp and Go dependencies\n"
	@printf "    $(OLIVE)make run$(RESET)          - Run the CLI downloader application\n"
	@printf "    $(OLIVE)make server$(RESET)       - Start the Zilla-Rec web server interface\n"
	@printf "    $(OLIVE)make clean$(RESET)        - Clean build artifacts and temporary files\n"
	@printf "$(CYAN)======================================================================$(RESET)\n"

build:
	@printf "$(OLIVE)[BUILD]$(RESET) Compiling native Zilla-Rec binary...\n"
	@go build -v -o $(BINARY_NAME) .
	@printf "$(GREEN)[SUCCESS]$(RESET) Binary built successfully: $(BOLD)./$(BINARY_NAME)$(RESET)\n"

build-linux:
	@printf "$(OLIVE)[BUILD-LINUX]$(RESET) Cross-compiling Zilla-Rec for Linux/WSL2...\n"
	@GOOS=linux GOARCH=amd64 go build -v -o bin/$(BINARY_NAME)-linux .
	@printf "$(GREEN)[SUCCESS]$(RESET) Linux binary built: $(BOLD)bin/$(BINARY_NAME)-linux$(RESET)\n"

install-wsl: build-linux
	@printf "$(OLIVE)[INSTALL-WSL]$(RESET) Triggering PowerShell deployment suite...\n"
	@powershell -ExecutionPolicy Bypass -File install.ps1

test:
	@printf "$(OLIVE)[TEST]$(RESET) Running unit test suite...\n"
	@go test -v ./...

diag:
	@printf "$(OLIVE)[DIAG]$(RESET) Running comprehensive system diagnostics...\n"
	@printf "  1. Checking Go version...\n"
	@go version
	@printf "  2. Running 'go vet'...\n"
	@go vet ./...
	@printf "  3. Running unit test suite...\n"
	@go test -v ./...
	@printf "  4. Checking yt-dlp dependency in PATH...\n"
	@if command -v yt-dlp &> /dev/null; then \
		printf "     $(GREEN)✔ yt-dlp found ($(shell which yt-dlp))$(RESET)\n"; \
	else \
		printf "     $(RED)✖ yt-dlp not found in PATH! (Install via pip/pkg)$(RESET)\n"; \
	fi
	@printf "$(GREEN)[DIAG COMPLETE]$(RESET) All diagnostics passed successfully!\n"

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
	@go get -u ./...
	@go mod tidy
	@printf "$(GREEN)[UPDATE COMPLETE]$(RESET) All dependencies updated successfully!\n"

run: build
	@printf "$(OLIVE)[RUN]$(RESET) Starting Zilla-Rec CLI...\n"
	@./$(BINARY_NAME)

server: build
	@printf "$(OLIVE)[SERVER]$(RESET) Starting Zilla-Rec VCR web server mode...\n"
	@./$(BINARY_NAME) web

clean:
	@printf "$(RED)[CLEAN]$(RESET) Removing build artifacts and temp files...\n"
	@rm -f $(BINARY_NAME)
	@rm -rf bin/
	@go clean
	@printf "$(GREEN)[CLEAN COMPLETE]$(RESET)\n"
