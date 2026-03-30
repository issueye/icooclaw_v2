#!/bin/bash

# Icooclaw Build Script for Linux
# Usage: ./build.sh [build|clean|test|install]

set -e

BINARY_NAME="icooclaw"
VERSION="dev"

# Get git version
if command -v git &> /dev/null; then
    VERSION=$(git describe --tags --always --dirty 2>/dev/null || echo "dev")
fi

TARGET="${1:-build}"

case "$TARGET" in
    clean)
        echo "Cleaning..."
        rm -rf bin
        rm -f "${BINARY_NAME}"
        echo "Done."
        ;;
    test)
        echo "Running tests..."
        go test -v ./...
        ;;
    install)
        echo "Installing..."
        go install -ldflags "-s -w -X github.com/icooclaw/icooclaw/cmd/icooclaw/commands.version=${VERSION}" ./cmd/icooclaw
        echo "Done."
        ;;
    build)
        echo "Building ${BINARY_NAME} v${VERSION} for Linux..."
        mkdir -p bin
        
        # Build for Linux AMD64
        GOOS=linux GOARCH=amd64 go build \
            -ldflags "-s -w -X github.com/icooclaw/icooclaw/cmd/icooclaw/commands.version=${VERSION}" \
            -o "bin/${BINARY_NAME}" \
            ./cmd/icooclaw
        
        echo "Build OK: bin/${BINARY_NAME}"
        echo "To run: ./bin/${BINARY_NAME}"
        ;;
    *)
        echo "Usage: $0 [build|clean|test|install]"
        exit 1
        ;;
esac
