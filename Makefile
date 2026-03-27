# Makefile for Ditto
# Universal build system

.PHONY: all build clean test release help

# Variables
VERSION ?= 0.1.0
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
DATE ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS ?= -s -w -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)

# Go parameters
GOCMD ?= go
GOBUILD ?= $(GOCMD) build
GOTEST ?= $(GOCMD) test
GOGET ?= $(GOCMD) get
GOMOD ?= $(GOCMD) mod
GOFMT ?= gofmt

# Binary name
BINARY_NAME ?= Ditto
OUTPUT_DIR ?= dist

# Default target
all: build

# ─────────────────────────────────────────────────────────────
# Build
# ─────────────────────────────────────────────────────────────

build: ## Build for current platform
	@echo "🔨 Building $(BINARY_NAME)..."
	$(GOBUILD) -o $(BINARY_NAME) -ldflags "$(LDFLAGS)" ./cmd/ditto
	@echo "✅ Build complete: $(BINARY_NAME)"

build-debug: ## Build with debug symbols
	@echo "🔨 Building $(BINARY_NAME) (debug)..."
	$(GOBUILD) -race -gcflags="all=-N -l" -o $(BINARY_NAME)-debug ./cmd/ditto
	@echo "✅ Debug build complete: $(BINARY_NAME)-debug"

build-all: ## Build for all platforms
	@echo "🔨 Building for all platforms..."
	@mkdir -p $(OUTPUT_DIR)
	
	# Windows
	@echo "  → Windows amd64"
	GOOS=windows GOARCH=amd64 $(GOBUILD) -o $(OUTPUT_DIR)/$(BINARY_NAME)-windows-amd64.exe -ldflags "$(LDFLAGS)" ./cmd/ditto
	@echo "  → Windows arm64"
	GOOS=windows GOARCH=arm64 $(GOBUILD) -o $(OUTPUT_DIR)/$(BINARY_NAME)-windows-arm64.exe -ldflags "$(LDFLAGS)" ./cmd/ditto
	
	# macOS
	@echo "  → macOS amd64"
	GOOS=darwin GOARCH=amd64 $(GOBUILD) -o $(OUTPUT_DIR)/$(BINARY_NAME)-macos-amd64 -ldflags "$(LDFLAGS)" ./cmd/ditto
	@echo "  → macOS arm64"
	GOOS=darwin GOARCH=arm64 $(GOBUILD) -o $(OUTPUT_DIR)/$(BINARY_NAME)-macos-arm64 -ldflags "$(LDFLAGS)" ./cmd/ditto
	
	# Linux
	@echo "  → Linux amd64"
	GOOS=linux GOARCH=amd64 $(GOBUILD) -o $(OUTPUT_DIR)/$(BINARY_NAME)-linux-amd64 -ldflags "$(LDFLAGS)" ./cmd/ditto
	@echo "  → Linux arm64"
	GOOS=linux GOARCH=arm64 $(GOBUILD) -o $(OUTPUT_DIR)/$(BINARY_NAME)-linux-arm64 -ldflags "$(LDFLAGS)" ./cmd/ditto
	
	@echo "✅ All builds complete in $(OUTPUT_DIR)/"

build-tiny: ## Build smallest possible binary
	@echo "🔨 Building tiny $(BINARY_NAME)..."
	CGO_ENABLED=0 $(GOBUILD) -trimpath -ldflags "-s -w $(LDFLAGS)" -o $(BINARY_NAME)-tiny ./cmd/ditto
	@ls -lh $(BINARY_NAME)-tiny

# ─────────────────────────────────────────────────────────────
# Test
# ─────────────────────────────────────────────────────────────

test: ## Run tests
	@echo "🧪 Running tests..."
	$(GOTEST) -v -race -cover ./...

test-coverage: ## Run tests with coverage report
	@echo "🧪 Running tests with coverage..."
	$(GOTEST) -v -race -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "📊 Coverage report: coverage.html"

test-examples: ## Test all example files
	@echo "🧪 Testing examples..."
	./$(BINARY_NAME) run examples/hello.py
	./$(BINARY_NAME) run examples/hello.js
	./$(BINARY_NAME) run examples/hello.lua
	./$(BINARY_NAME) run examples/query.sql
	@echo "✅ All examples passed"

# ─────────────────────────────────────────────────────────────
# Code Quality
# ─────────────────────────────────────────────────────────────

fmt: ## Format all Go files
	@echo "📝 Formatting code..."
	$(GOFMT) -l -w -s .
	@echo "✅ Formatting complete"

vet: ## Run go vet
	@echo "🔍 Running go vet..."
	$(GOCMD) vet ./...
	@echo "✅ Vet complete"

lint: fmt vet ## Run all linting checks
	@echo "✅ Linting complete"

tidy: ## Tidy go modules
	@echo "📦 Tidying modules..."
	$(GOMOD) tidy
	$(GOMOD) verify

# ─────────────────────────────────────────────────────────────
# Docker
# ─────────────────────────────────────────────────────────────

docker-build: ## Build Docker image
	@echo "🐳 Building Docker image..."
	docker build -t ditto:$(VERSION) --build-arg VERSION=$(VERSION) --build-arg COMMIT=$(COMMIT) .
	@echo "✅ Docker image built: ditto:$(VERSION)"

docker-run: ## Run Ditto in Docker
	@echo "🐳 Running Ditto in Docker..."
	docker run --rm -it ditto:$(VERSION) $(CMD)

docker-test: ## Test Docker image
	@echo "🐳 Testing Docker image..."
	docker run --rm ditto:$(VERSION) version
	docker run --rm ditto:$(VERSION) languages

# ─────────────────────────────────────────────────────────────
# Release
# ─────────────────────────────────────────────────────────────

release: ## Create a release using goreleaser
	@echo "🚀 Creating release..."
	goreleaser release --rm-dist --snapshot

release-dry: ## Dry run release
	@echo "🚀 Dry run release..."
	goreleaser release --rm-dist --snapshot --skip-publish

checksum: ## Generate checksums
	@echo "📋 Generating checksums..."
	cd $(OUTPUT_DIR) && sha256sum * > checksums.txt
	@echo "✅ Checksums generated: $(OUTPUT_DIR)/checksums.txt"

# ─────────────────────────────────────────────────────────────
# Cleanup
# ─────────────────────────────────────────────────────────────

clean: ## Clean build artifacts
	@echo "🧹 Cleaning..."
	rm -rf $(OUTPUT_DIR)
	rm -f $(BINARY_NAME)
	rm -f $(BINARY_NAME)-debug
	rm -f $(BINARY_NAME)-tiny
	rm -f coverage.out coverage.html
	@echo "✅ Clean complete"

distclean: clean ## Clean everything including cached modules
	@echo "🧹 Deep cleaning..."
	$(GOCMD) clean -cache -modcache
	@echo "✅ Deep clean complete"

# ─────────────────────────────────────────────────────────────
# Help
# ─────────────────────────────────────────────────────────────

help: ## Show this help message
	@echo "Ditto Build System"
	@echo "=================="
	@echo ""
	@echo "Usage: make [target]"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'
	@echo ""

# Default version info
version:
	@echo "Ditto Build System v$(VERSION)"
	@echo "Commit: $(COMMIT)"
	@echo "Date: $(DATE)"
