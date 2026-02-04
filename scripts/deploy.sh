#!/bin/bash
# GoStrike Deploy Script
# Deploys GoStrike to a CS2 dedicated server

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"
BUILD_DIR="$PROJECT_DIR/build"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

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

# Default CS2 path
CS2_PATH="${CS2_PATH:-/opt/cs2-server/game/csgo}"

# Parse arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --path)
            CS2_PATH="$2"
            shift 2
            ;;
        --build)
            "$SCRIPT_DIR/build.sh"
            shift
            ;;
        --help|-h)
            echo "Usage: $0 [OPTIONS]"
            echo ""
            echo "Options:"
            echo "  --path PATH   CS2 server path (default: $CS2_PATH)"
            echo "  --build       Build before deploying"
            echo "  --help        Show this help"
            echo ""
            echo "Environment:"
            echo "  CS2_PATH      CS2 server game directory"
            exit 0
            ;;
        *)
            error "Unknown option: $1"
            ;;
    esac
done

# Validate paths
if [ ! -f "$BUILD_DIR/libgostrike_go.so" ]; then
    error "Go library not built. Run ./scripts/build.sh first"
fi

if [ ! -d "$CS2_PATH" ]; then
    error "CS2 path not found: $CS2_PATH"
fi

info "Deploying GoStrike to: $CS2_PATH"

# Create directories
info "Creating directories..."
mkdir -p "$CS2_PATH/addons/gostrike/bin"
mkdir -p "$CS2_PATH/addons/gostrike/configs"
mkdir -p "$CS2_PATH/addons/metamod"

# Copy Go library
info "Copying Go library..."
cp "$BUILD_DIR/libgostrike_go.so" "$CS2_PATH/addons/gostrike/bin/"

# Copy configs
info "Copying configurations..."
if [ -f "$PROJECT_DIR/configs/gostrike.json" ]; then
    cp "$PROJECT_DIR/configs/gostrike.json" "$CS2_PATH/addons/gostrike/configs/"
fi

# Copy native plugin if exists
if [ -f "$BUILD_DIR/native/gostrike.so" ]; then
    info "Copying native plugin..."
    cp "$BUILD_DIR/native/gostrike.so" "$CS2_PATH/addons/metamod/"
elif [ -f "$BUILD_DIR/gostrike.so" ]; then
    info "Copying native plugin..."
    cp "$BUILD_DIR/gostrike.so" "$CS2_PATH/addons/metamod/"
else
    warn "Native plugin not found, skipping"
fi

# Show deployment summary
info ""
info "Deployment complete!"
info ""
info "Deployed files:"
ls -la "$CS2_PATH/addons/gostrike/bin/" 2>/dev/null || true
ls -la "$CS2_PATH/addons/metamod/gostrike.so" 2>/dev/null || true
info ""
info "Next steps:"
info "1. Add 'gostrike' to your metaplugins.ini"
info "2. Restart your CS2 server"
info "3. Check console for '[GoStrike]' messages"
