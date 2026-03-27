# Cross-platform build script for Ditto
# Builds tiny optimized binaries for Windows, macOS, and Linux

$ErrorActionPreference = "Stop"

$VERSION = if ($env:VERSION) { $env:VERSION } else { "0.1.0" }
$OUTPUT_DIR = if ($env:OUTPUT_DIR) { $env:OUTPUT_DIR } else { "dist" }

Write-Host "🔨 Building Ditto v${VERSION}..."
Write-Host "📁 Output directory: ${OUTPUT_DIR}"
Write-Host ""

# Create output directory
New-Item -ItemType Directory -Force -Path $OUTPUT_DIR | Out-Null

# Build flags for tiny binaries
$LDFLAGS = "-s -w -X main.version=${VERSION}"

Write-Host "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
Write-Host "Building for Windows..."
Write-Host "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

$env:GOOS = "windows"
$env:GOARCH = "amd64"
go build -o "${OUTPUT_DIR}/Ditto-windows-amd64.exe" -ldflags "${LDFLAGS}" ./cmd/ditto
Write-Host "✅ Ditto-windows-amd64.exe"

$env:GOOS = "windows"
$env:GOARCH = "arm64"
go build -o "${OUTPUT_DIR}/Ditto-windows-arm64.exe" -ldflags "${LDFLAGS}" ./cmd/ditto
Write-Host "✅ Ditto-windows-arm64.exe"

Write-Host ""
Write-Host "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
Write-Host "Building for macOS..."
Write-Host "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

$env:GOOS = "darwin"
$env:GOARCH = "amd64"
go build -o "${OUTPUT_DIR}/Ditto-macos-amd64" -ldflags "${LDFLAGS}" ./cmd/ditto
Write-Host "✅ Ditto-macos-amd64 (Intel)"

$env:GOOS = "darwin"
$env:GOARCH = "arm64"
go build -o "${OUTPUT_DIR}/Ditto-macos-arm64" -ldflags "${LDFLAGS}" ./cmd/ditto
Write-Host "✅ Ditto-macos-arm64 (Apple Silicon)"

Write-Host ""
Write-Host "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
Write-Host "Building for Linux..."
Write-Host "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

$env:GOOS = "linux"
$env:GOARCH = "amd64"
go build -o "${OUTPUT_DIR}/Ditto-linux-amd64" -ldflags "${LDFLAGS}" ./cmd/ditto
Write-Host "✅ Ditto-linux-amd64"

$env:GOOS = "linux"
$env:GOARCH = "arm64"
go build -o "${OUTPUT_DIR}/Ditto-linux-arm64" -ldflags "${LDFLAGS}" ./cmd/ditto
Write-Host "✅ Ditto-linux-arm64"

# Reset environment
$env:GOOS = ""
$env:GOARCH = ""

Write-Host ""
Write-Host "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
Write-Host "Build Summary"
Write-Host "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
Write-Host ""

Get-ChildItem $OUTPUT_DIR | Format-Table Name, Length -AutoSize

Write-Host ""
Write-Host "🎉 Build complete! All binaries in ./${OUTPUT_DIR}/"
Write-Host ""
