#!/bin/bash
# Build GoStrike native plugin inside Docker container with correct GLIBC
# This ensures compatibility with the CS2 server Steam Runtime

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"

echo "Building GoStrike native plugin inside Docker..."
echo "Project directory: $PROJECT_DIR"

# Use the Steam Runtime Sniper SDK for building
# This has the correct GLIBC version
# Build Go library inside container with compatible GLIBC (Debian Bullseye = GLIBC 2.31)
echo "Building Go shared library in container (GLIBC compatible)..."
docker run --rm \
    -v "$PROJECT_DIR:/build" \
    -w /build \
    golang:1.21-bullseye \
    bash -c '
        rm -f build/libgostrike_go.so
        mkdir -p build
        CGO_ENABLED=1 go build -buildvcs=false -buildmode=c-shared -o build/libgostrike_go.so ./cmd/gostrike
        chown -R $(stat -c %u:%g /build) /build/build
        echo "Built: build/libgostrike_go.so"
        ls -la build/libgostrike_go.so
    '

# Build native plugin inside Steam Runtime container for GLIBC compatibility
docker run --rm \
    -v "$PROJECT_DIR:/build" \
    -w /build \
    registry.gitlab.steamos.cloud/steamrt/sniper/sdk:latest \
    bash -c '
        echo "Installing build dependencies..."
        apt-get update -qq && apt-get install -y -qq cmake build-essential

        echo "Cleaning previous native build..."
        rm -rf build/native

        echo "Building native plugin with stub SDK..."
        mkdir -p build/native
        cd build/native
        cmake ../../native -DUSE_STUB_SDK=ON -DCMAKE_BUILD_TYPE=Release
        make -j$(nproc)
        
        # Fix ownership so host user can access files
        chown -R $(stat -c %u:%g /build) /build/build
        
        echo ""
        echo "Native plugin build complete!"
        ls -la gostrike.so
    '

echo ""
echo "Native plugin built: build/native/gostrike.so"
echo "Now run: make deploy server-restart"
