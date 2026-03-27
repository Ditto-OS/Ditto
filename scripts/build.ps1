# Build script for Windows
# Usage: .\scripts\build.ps1

$ErrorActionPreference = "Stop"

Write-Host "Building Ditto for Windows..."

# Get version
$VERSION = "0.1.0"

# Build for Windows
go build -o Ditto.exe -ldflags "-s -w" ./cmd/ditto

Write-Host "Build complete: Ditto.exe"

# Cross-compile for Linux
Write-Host "`nBuilding for Linux..."
$env:GOOS = "linux"
$env:GOARCH = "amd64"
go build -o Ditto-linux -ldflags "-s -w" ./cmd/ditto

# Cross-compile for macOS
Write-Host "Building for macOS..."
$env:GOOS = "darwin"
$env:GOARCH = "amd64"
go build -o Ditto-macos -ldflags "-s -w" ./cmd/ditto

# Reset environment
$env:GOOS = ""
$env:GOARCH = ""

Write-Host "`nAll builds complete!"
Write-Host "  - Ditto.exe (Windows)"
Write-Host "  - Ditto-linux (Linux)"
Write-Host "  - Ditto-macos (macOS)"
