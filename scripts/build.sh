#!/bin/bash
# Build script for Linux/macOS
# Usage: ./scripts/build.sh

set -e

echo "Building Ditto..."

VERSION="0.1.0"

# Build for current platform
go build -o Ditto -ldflags "-s -w" ./cmd/ditto

echo "Build complete: Ditto"

# Cross-compile for Windows
echo "Building for Windows..."
GOOS=windows GOARCH=amd64 go build -o Ditto.exe -ldflags "-s -w" ./cmd/ditto

# Cross-compile for Linux (if not on Linux)
if [ "$(uname)" != "Linux" ]; then
    echo "Building for Linux..."
    GOOS=linux GOARCH=amd64 go build -o Ditto-linux -ldflags "-s -w" ./cmd/ditto
fi

# Cross-compile for macOS (if not on macOS)
if [ "$(uname)" != "Darwin" ]; then
    echo "Building for macOS..."
    GOOS=darwin GOARCH=amd64 go build -o Ditto-macos -ldflags "-s -w" ./cmd/ditto
fi

echo ""
echo "All builds complete!"
ls -la Ditto*
