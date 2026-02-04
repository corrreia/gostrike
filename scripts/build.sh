#!/bin/bash
# GoStrike Build Script
# Builds the Go shared library and optionally the native plugin

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"
BUILD_DIR="$PROJECT_DIR/build"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

error() {
    echo -e "${RED}[ERROR]${NC} $1"
    exit 1
}

# Parse arguments
BUILD_NATIVE=false
DEBUG=false

while [[ $# -gt 0 ]]; do
    case $1 in
        --native)
            BUILD_NATIVE=true
            shift
            ;;
        --debug)
            DEBUG=true
            shift
            ;;
        --help|-h)
            echo "Usage: $0 [OPTIONS]"
            echo ""
            echo "Options:"
            echo "  --native    Build native Metamod plugin (requires SDK)"
            echo "  --debug     Build with debug symbols"
            echo "  --help      Show this help"
            exit 0
            ;;
        *)
            error "Unknown option: $1"
            ;;
    esac
done

# Check Go version
GO_VERSION=$(go version | awk '{print $3}')
info "Go version: $GO_VERSION"

# Create build directory
mkdir -p "$BUILD_DIR"

# Build Go shared library
info "Building Go shared library..."

cd "$PROJECT_DIR"

GO_FLAGS="-buildmode=c-shared"
if [ "$DEBUG" = true ]; then
    GO_FLAGS="$GO_FLAGS -gcflags=all=-N\\ -l"
    info "Debug mode enabled"
fi

CGO_ENABLED=1 go build $GO_FLAGS \
    -o "$BUILD_DIR/libgostrike_go.so" \
    ./cmd/gostrike

info "Built: $BUILD_DIR/libgostrike_go.so"
info "Header: $BUILD_DIR/libgostrike_go.h"

# Build native plugin if requested
if [ "$BUILD_NATIVE" = true ]; then
    info "Building native Metamod plugin..."
    
    METAMOD_PATH="${METAMOD_PATH:-/opt/metamod-source}"
    HL2SDK_PATH="${HL2SDK_PATH:-/opt/hl2sdk-cs2}"
    
    if [ ! -d "$METAMOD_PATH" ]; then
        warn "Metamod not found at $METAMOD_PATH"
        warn "Set METAMOD_PATH environment variable"
    fi
    
    if [ ! -d "$HL2SDK_PATH" ]; then
        warn "HL2SDK not found at $HL2SDK_PATH"
        warn "Set HL2SDK_PATH environment variable"
    fi
    
    mkdir -p "$BUILD_DIR/native"
    cd "$BUILD_DIR/native"
    
    cmake "$PROJECT_DIR/native" \
        -DMETAMOD_PATH="$METAMOD_PATH" \
        -DHL2SDK_PATH="$HL2SDK_PATH" \
        -DCMAKE_BUILD_TYPE=$([ "$DEBUG" = true ] && echo Debug || echo Release)
    
    make -j$(nproc)
    
    info "Built: $BUILD_DIR/native/gostrike.so"
fi

info "Build complete!"
