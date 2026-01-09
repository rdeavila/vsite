# videogen - Makefile
# Static HTML video gallery generator

# Binary name
BINARY_NAME := vsite

# Version from main.go
VERSION := 1.4.0

# Build directory
BUILD_DIR := build

# Go parameters
GOCMD := go
GOBUILD := $(GOCMD) build
GOCLEAN := $(GOCMD) clean
GOTEST := $(GOCMD) test
GOGET := $(GOCMD) get
GOMOD := $(GOCMD) mod

# Build flags for optimization
# -s: Omit symbol table and debug info
# -w: Omit DWARF symbol table
# -trimpath: Remove file system paths from binary
LDFLAGS := -ldflags="-s -w -X main.version=$(VERSION)"
GCFLAGS := -gcflags=all=-l
BUILDFLAGS := -trimpath $(LDFLAGS)

# Platforms for cross-compilation
PLATFORMS := linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64

# Default target
.DEFAULT_GOAL := build

# Phony targets
.PHONY: all build build-all clean test deps install uninstall help serve

# Build for current platform with optimizations
build:
	@echo "Building $(BINARY_NAME)..."
	CGO_ENABLED=0 $(GOBUILD) $(BUILDFLAGS) -o $(BINARY_NAME) .
	@echo "Done! Binary: ./$(BINARY_NAME)"
	@ls -lh $(BINARY_NAME)

# Build with debug symbols (for development)
build-debug:
	@echo "Building $(BINARY_NAME) with debug symbols..."
	$(GOBUILD) -o $(BINARY_NAME) .
	@echo "Done! Binary: ./$(BINARY_NAME)"

# Build for all platforms
build-all: clean
	@mkdir -p $(BUILD_DIR)
	@echo "Building for all platforms..."
	@for platform in $(PLATFORMS); do \
		GOOS=$${platform%/*} GOARCH=$${platform#*/} \
		CGO_ENABLED=0 $(GOBUILD) $(BUILDFLAGS) \
		-o $(BUILD_DIR)/$(BINARY_NAME)-$${platform%/*}-$${platform#*/}$$(if [ "$${platform%/*}" = "windows" ]; then echo ".exe"; fi) . ; \
		echo "  Built: $(BINARY_NAME)-$${platform%/*}-$${platform#*/}" ; \
	done
	@echo "Done! Binaries in $(BUILD_DIR)/"
	@ls -lh $(BUILD_DIR)/

# Build for Linux amd64
build-linux:
	@echo "Building for Linux amd64..."
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GOBUILD) $(BUILDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 .
	@echo "Done! Binary: $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64"

# Build for macOS (Apple Silicon)
build-darwin:
	@echo "Building for macOS arm64..."
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 $(GOBUILD) $(BUILDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 .
	@echo "Done! Binary: $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64"

# Build for Windows
build-windows:
	@echo "Building for Windows amd64..."
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 $(GOBUILD) $(BUILDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe .
	@echo "Done! Binary: $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe"

# Clean build artifacts
clean:
	@echo "Cleaning..."
	$(GOCLEAN)
	rm -f $(BINARY_NAME)
	rm -rf $(BUILD_DIR)
	@echo "Done!"

# Run tests
test:
	@echo "Running tests..."
	$(GOTEST) -v ./...

# Download dependencies
deps:
	@echo "Downloading dependencies..."
	$(GOMOD) download
	$(GOMOD) tidy
	@echo "Done!"

# Install to /usr/local/bin
install: build
	@echo "Installing to /usr/local/bin/$(BINARY_NAME)..."
	sudo cp $(BINARY_NAME) /usr/local/bin/$(BINARY_NAME)
	sudo chmod +x /usr/local/bin/$(BINARY_NAME)
	@echo "Done! Run '$(BINARY_NAME) --help' to get started"

# Install to user's local bin
install-user: build
	@echo "Installing to ~/.local/bin/$(BINARY_NAME)..."
	@mkdir -p ~/.local/bin
	cp $(BINARY_NAME) ~/.local/bin/$(BINARY_NAME)
	chmod +x ~/.local/bin/$(BINARY_NAME)
	@echo "Done! Make sure ~/.local/bin is in your PATH"

# Uninstall from /usr/local/bin
uninstall:
	@echo "Uninstalling from /usr/local/bin/$(BINARY_NAME)..."
	sudo rm -f /usr/local/bin/$(BINARY_NAME)
	@echo "Done!"

# Show binary info
info: build
	@echo ""
	@echo "Binary information:"
	@file $(BINARY_NAME)
	@echo ""
	@echo "Size:"
	@ls -lh $(BINARY_NAME)
	@echo ""
	@echo "Version:"
	@./$(BINARY_NAME) --version

# Compress binary with UPX (if available)
compress: build
	@if command -v upx >/dev/null 2>&1; then \
		echo "Compressing with UPX..."; \
		upx --best --lzma $(BINARY_NAME); \
		ls -lh $(BINARY_NAME); \
	else \
		echo "UPX not found. Install with: sudo apt install upx"; \
	fi

# Start HTTP server with range request support (for video seeking)
serve:
	@echo "Starting HTTP server with range request support..."
	@echo "Server will be available at http://localhost:8000"
	@echo "Press Ctrl+C to stop"
	@echo ""
	npx -y http-server -p 8000 -c-1

# Help
help:
	@echo "videogen - Build targets"
	@echo ""
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@echo "  build         Build optimized binary for current platform (default)"
	@echo "  build-debug   Build with debug symbols"
	@echo "  build-all     Build for all platforms (linux, darwin, windows)"
	@echo "  build-linux   Build for Linux amd64"
	@echo "  build-darwin  Build for macOS arm64"
	@echo "  build-windows Build for Windows amd64"
	@echo "  clean         Remove build artifacts"
	@echo "  test          Run tests"
	@echo "  deps          Download and tidy dependencies"
	@echo "  install       Install to /usr/local/bin (requires sudo)"
	@echo "  install-user  Install to ~/.local/bin"
	@echo "  uninstall     Remove from /usr/local/bin"
	@echo "  info          Show binary information"
	@echo "  compress      Compress binary with UPX"
	@echo "  serve         Start HTTP server with range request support"
	@echo "  help          Show this help"
