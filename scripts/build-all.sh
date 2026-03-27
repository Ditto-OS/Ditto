#!/bin/bash
# Cross-platform build script for Ditto
# Builds tiny optimized binaries for Windows, macOS, and Linux

set -e

VERSION=${VERSION:-"0.1.0"}
OUTPUT_DIR=${OUTPUT_DIR:-"dist"}

echo "🔨 Building Ditto v${VERSION}..."
echo "📁 Output directory: ${OUTPUT_DIR}"

# Create output directory
mkdir -p "${OUTPUT_DIR}"

# Build flags for tiny binaries
LDFLAGS="-s -w -X main.version=${VERSION}"

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "Building for Windows..."
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

GOOS=windows GOARCH=amd64 go build -o "${OUTPUT_DIR}/Ditto-windows-amd64.exe" -ldflags "${LDFLAGS}" ./cmd/ditto
echo "✅ Ditto-windows-amd64.exe"

GOOS=windows GOARCH=arm64 go build -o "${OUTPUT_DIR}/Ditto-windows-arm64.exe" -ldflags "${LDFLAGS}" ./cmd/ditto
echo "✅ Ditto-windows-arm64.exe"

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "Building for macOS..."
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

GOOS=darwin GOARCH=amd64 go build -o "${OUTPUT_DIR}/Ditto-macos-amd64" -ldflags "${LDFLAGS}" ./cmd/ditto
echo "✅ Ditto-macos-amd64 (Intel)"

GOOS=darwin GOARCH=arm64 go build -o "${OUTPUT_DIR}/Ditto-macos-arm64" -ldflags "${LDFLAGS}" ./cmd/ditto
echo "✅ Ditto-macos-arm64 (Apple Silicon)"

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "Building for Linux..."
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

GOOS=linux GOARCH=amd64 go build -o "${OUTPUT_DIR}/Ditto-linux-amd64" -ldflags "${LDFLAGS}" ./cmd/ditto
echo "✅ Ditto-linux-amd64"

GOOS=linux GOARCH=arm64 go build -o "${OUTPUT_DIR}/Ditto-linux-arm64" -ldflags "${LDFLAGS}" ./cmd/ditto
echo "✅ Ditto-linux-arm64"

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "Generating checksums..."
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

cd "${OUTPUT_DIR}"
sha256sum * > checksums.txt 2>/dev/null || shasum -a 256 * > checksums.txt
cd ..

echo "✅ checksums.txt"

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "Build Summary"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

ls -lh "${OUTPUT_DIR}"

echo ""
echo "🎉 Build complete! All binaries in ./${OUTPUT_DIR}/"
echo ""
