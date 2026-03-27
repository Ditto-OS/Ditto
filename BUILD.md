# CI/CD & Build Guide

## Automated Builds (GitHub Actions)

Ditto uses GitHub Actions to automatically build tiny binaries for all platforms on every commit.

### What Happens on Push

```yaml
Push to main → GitHub Actions → 6 binaries built & tested
                                    ↓
                    ┌───────────────┼───────────────┐
                    ↓               ↓               ↓
            Windows (2)      macOS (2)       Linux (2)
            - amd64          - amd64         - amd64
            - arm64          - arm64         - arm64
```

### Workflow Jobs

| Job | Platform | Output |
|-----|----------|--------|
| `build-windows` | Windows Latest | `Ditto-windows-amd64.exe` |
| `build-macos-intel` | macOS 13 | `Ditto-macos-amd64` |
| `build-macos-arm` | macOS Latest | `Ditto-macos-arm64` |
| `build-linux` | Ubuntu Latest | `Ditto-linux-amd64` |
| `test` | Ubuntu Latest | Test results |
| `lint` | Ubuntu Latest | Code quality |

### On Tag Release (v*)

When you tag a release (`git tag v0.1.0`), additional jobs run:

| Job | Purpose |
|-----|---------|
| `build-cross` | Builds all 6 platforms |
| `create-release` | Creates GitHub Release with all binaries |

---

## Manual Builds

### Quick Build (Current Platform)

```bash
# Using Go directly
go build -o Ditto ./cmd/ditto

# Using Make
make build
```

### Build All Platforms

```bash
# Using the build script (recommended)
./scripts/build-all.sh        # Linux/macOS
.\scripts\build-all.ps1       # Windows

# Using Make
make build-all

# Using Go directly
GOOS=windows GOARCH=amd64 go build -o dist/Ditto-windows.exe ./cmd/ditto
GOOS=darwin GOARCH=amd64 go build -o dist/Ditto-macos ./cmd/ditto
GOOS=linux GOARCH=amd64 go build -o dist/Ditto-linux ./cmd/ditto
```

### Build with Make (All Targets)

```bash
make help              # Show all targets
make build             # Build for current platform
make build-all         # Build for all platforms
make build-tiny        # Build smallest binary
make test              # Run tests
make test-examples     # Test all example files
make lint              # Run code quality checks
make docker-build      # Build Docker image
make release           # Create release with goreleaser
make clean             # Clean build artifacts
```

---

## Build Output Sizes

Typical binary sizes after optimization (`-ldflags "-s -w"`):

| Platform | Architecture | Size |
|----------|--------------|------|
| Windows | amd64 | ~13 MB |
| Windows | arm64 | ~12 MB |
| macOS | amd64 | ~13 MB |
| macOS | arm64 | ~12 MB |
| Linux | amd64 | ~12 MB |
| Linux | arm64 | ~11 MB |

---

## Docker Builds

### Build Image

```bash
# Using Make
make docker-build

# Using Docker directly
docker build -t ditto:0.1.0 \
  --build-arg VERSION=0.1.0 \
  --build-arg COMMIT=$(git rev-parse --short HEAD) \
  .
```

### Run in Container

```bash
# Run help
docker run --rm ditto:0.1.0 help

# Run a script (mount current directory)
docker run --rm -v $(pwd):/data ditto:0.1.0 run /data/script.py

# Interactive mode
docker run --rm -it ditto:0.1.0
```

### Development Container

```bash
# Build dev image with debug symbols
docker build --target development -t ditto:dev .

# Run with volume mounts for live development
docker run --rm -it \
  -v $(pwd):/workspace \
  ditto:dev run examples/hello.py
```

---

## Professional Releases (GoReleaser)

[GoReleaser](https://goreleaser.com/) automates the entire release process:

### Setup

```bash
# Install goreleaser
go install github.com/goreleaser/goreleaser@latest

# Verify configuration
goreleaser check
```

### Create Release

```bash
# Dry run (local testing)
goreleaser release --snapshot --rm-dist

# Actual release (creates GitHub Release)
goreleaser release --rm-dist
```

### What GoReleaser Does

1. **Builds** binaries for all configured platforms
2. **Creates** archives (.tar.gz, .zip)
3. **Generates** checksums
4. **Creates** GitHub Release
5. **Publishes** to package managers:
   - Homebrew (macOS/Linux)
   - Scoop (Windows)
   - Docker Hub
   - GitHub Container Registry
6. **Signs** binaries (cosign)
7. **Generates** SBOMs

---

## GitHub Actions Workflow File

The workflow is configured in `.github/workflows/build.yml`:

```yaml
name: Build Ditto

on:
  push:
    branches: [main]
    tags: ['v*']
  pull_request:
    branches: [main]

jobs:
  build-windows:
    runs-on: windows-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.21'
      - run: go build -o Ditto.exe -ldflags "-s -w" ./cmd/ditto
      - uses: actions/upload-artifact@v4
        with:
          name: Ditto-windows
          path: Ditto.exe
```

---

## Build Flags Explained

| Flag | Purpose |
|------|---------|
| `-s` | Omit symbol table (smaller binary) |
| `-w` | Omit DWARF debug info (smaller binary) |
| `-trimpath` | Remove file system paths (reproducible builds) |
| `CGO_ENABLED=0` | Static binary, no C dependencies |
| `-ldflags "-X main.version=..."` | Embed version info |

### Example: Minimal Binary

```bash
# Smallest possible build
CGO_ENABLED=0 GOOS=linux go build \
  -trimpath \
  -ldflags "-s -w" \
  -o ditto \
  ./cmd/ditto
```

---

## Troubleshooting

### Build Fails on Windows

```powershell
# Ensure Go is in PATH
go version

# Clear Go cache
go clean -cache

# Rebuild
go build -v -a ./cmd/ditto
```

### Binary Too Large

```bash
# Check what's included
go build -ldflags="-s -w" ./cmd/ditto
upx --best Ditto  # Optional: compress further with UPX
```

### Cross-Compilation Issues

```bash
# Ensure proper toolchain
go install golang.org/dl/go1.21.0@latest
go1.21.0 download

# Set explicit build flags
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build ./cmd/ditto
```

---

## Binary Size Optimization Tips

1. **Strip symbols**: `-ldflags "-s -w"`
2. **Disable CGO**: `CGO_ENABLED=0`
3. **Trim paths**: `-trimpath`
4. **Use Alpine**: Build in `golang:alpine` container
5. **Compress**: Use `upx` for additional compression

```bash
# Ultimate tiny build
CGO_ENABLED=0 GOOS=linux go build \
  -trimpath \
  -ldflags "-s -w" \
  ./cmd/ditto

upx --best Ditto  # Reduces size by ~50%
```
