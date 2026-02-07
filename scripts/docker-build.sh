#!/bin/bash
# Build GoStrike inside Docker containers with correct GLIBC
# This ensures compatibility with the CS2 server Steam Runtime
#
# Architecture inspired by CounterStrikeSharp (https://github.com/roflmuffin/CounterStrikeSharp)

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"

# Parse arguments
USE_STUB_SDK=OFF
for arg in "$@"; do
    case $arg in
        --stub) USE_STUB_SDK=ON ;;
    esac
done

echo "Building GoStrike inside Docker..."
echo "Project directory: $PROJECT_DIR"
echo "SDK mode: $([ "$USE_STUB_SDK" = "ON" ] && echo "STUB" || echo "FULL")"

# ============================================================
# Step 1: Build Go shared library (GLIBC compatible)
# ============================================================
echo ""
echo "=== Step 1: Building Go shared library ==="
docker run --rm \
    -v "$PROJECT_DIR:/build" \
    -w /build \
    golang:1.24-bookworm \
    bash -c '
        rm -f build/libgostrike_go.so
        mkdir -p build
        CGO_ENABLED=1 go build -buildvcs=false -buildmode=c-shared -o build/libgostrike_go.so ./cmd/gostrike
        chown -R $(stat -c %u:%g /build) /build/build
        echo "Built: build/libgostrike_go.so"
        ls -la build/libgostrike_go.so
    '

# ============================================================
# Step 2: Build native Metamod plugin (Steam Runtime compatible)
# ============================================================
echo ""
echo "=== Step 2: Building native Metamod plugin ==="
docker run --rm \
    -v "$PROJECT_DIR:/build" \
    -w /build \
    registry.gitlab.steamos.cloud/steamrt/sniper/sdk:latest \
    bash -c "
        echo 'Installing build dependencies...'
        apt-get update -qq && apt-get install -y -qq cmake build-essential > /dev/null 2>&1

        echo 'Cleaning previous native build...'
        rm -rf build/native

        echo 'Initializing submodules (if needed)...'
        # Ensure submodules are available
        if [ ! -f external/metamod-source/core/ISmmPlugin.h ]; then
            echo 'ERROR: Metamod:Source submodule not initialized.'
            echo 'Run: git submodule update --init --recursive'
            exit 1
        fi

        echo 'Building native plugin...'
        mkdir -p build/native
        cd build/native

        cmake ../../native \\
            -DMETAMOD_PATH=/build/external/metamod-source \\
            -DHL2SDK_PATH=/build/external/hl2sdk-cs2 \\
            -DUSE_STUB_SDK=${USE_STUB_SDK} \\
            -DCMAKE_BUILD_TYPE=Release

        make -j\$(nproc)

        # Fix ownership
        chown -R \$(stat -c %u:%g /build) /build/build

        echo ''
        echo 'Native plugin build complete!'
        ls -la gostrike.so
    "

echo ""
echo "Build complete!"
echo "  Go library:    build/libgostrike_go.so"
echo "  Native plugin: build/native/gostrike.so"
echo ""
echo "Deploy with: make deploy"
