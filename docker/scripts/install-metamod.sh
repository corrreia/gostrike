#!/bin/bash
# GoStrike - Metamod:Source Installation Script
# Downloads and installs Metamod:Source to the CS2 server

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

info() { echo -e "${GREEN}[INFO]${NC} $1"; }
warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
error() { echo -e "${RED}[ERROR]${NC} $1"; exit 1; }
step() { echo -e "${BLUE}[STEP]${NC} $1"; }

# Configuration
METAMOD_VERSION="${METAMOD_VERSION:-2.0}"
METAMOD_BASE_URL="https://mms.alliedmods.net/mmsdrop/${METAMOD_VERSION}"
INSTALL_PATH=""

usage() {
    echo "Usage: $0 [OPTIONS]"
    echo ""
    echo "Options:"
    echo "  --path PATH    Path to CS2 data directory (e.g., docker/data/cs2)"
    echo "  --check        Only check if Metamod is installed"
    echo "  --force        Reinstall even if already installed"
    echo "  --help         Show this help"
    echo ""
    echo "Environment:"
    echo "  METAMOD_VERSION  Metamod version branch (default: 2.0)"
}

# Parse arguments
CHECK_ONLY=false
FORCE=false
while [[ $# -gt 0 ]]; do
    case $1 in
        --path)
            INSTALL_PATH="$2/game/csgo"
            shift 2
            ;;
        --check)
            CHECK_ONLY=true
            shift
            ;;
        --force)
            FORCE=true
            shift
            ;;
        --help|-h)
            usage
            exit 0
            ;;
        *)
            error "Unknown option: $1"
            ;;
    esac
done

# Validate path
if [ -z "${INSTALL_PATH}" ]; then
    error "Please specify --path to CS2 data directory"
fi

if [ ! -d "${INSTALL_PATH}" ]; then
    error "CS2 not installed at ${INSTALL_PATH}. Run 'make server-init' and wait for download to complete."
fi

# Check if Metamod is already installed
check_metamod() {
    if [ -f "${INSTALL_PATH}/addons/metamod.vdf" ]; then
        return 0
    fi
    return 1
}

if [ "$CHECK_ONLY" = true ]; then
    if check_metamod; then
        info "Metamod is installed"
        exit 0
    else
        warn "Metamod is NOT installed"
        exit 1
    fi
fi

# Install Metamod
step "Installing Metamod:Source ${METAMOD_VERSION}"

# Check if already installed
if check_metamod && [ "$FORCE" != true ]; then
    warn "Metamod already installed. Use --force to reinstall."
    exit 0
fi

# Create temp directory
TEMP_DIR=$(mktemp -d)
cleanup_temp() {
    rm -rf "${TEMP_DIR}"
}
trap cleanup_temp EXIT

# Get latest version filename
step "Fetching latest Metamod version..."
METAMOD_FILE=$(curl -sSL "${METAMOD_BASE_URL}/mmsource-latest-linux") || error "Failed to get latest version"
info "Latest version: ${METAMOD_FILE}"

# Download Metamod
step "Downloading ${METAMOD_FILE}..."
curl -sSL "${METAMOD_BASE_URL}/${METAMOD_FILE}" -o "${TEMP_DIR}/metamod.tar.gz" || error "Failed to download Metamod"

# Verify download
if ! file "${TEMP_DIR}/metamod.tar.gz" | grep -q "gzip"; then
    error "Downloaded file is not a valid gzip archive"
fi

# Extract (--no-same-owner to avoid permission issues)
step "Extracting..."
tar --no-same-owner -xzf "${TEMP_DIR}/metamod.tar.gz" -C "${TEMP_DIR}"

# Copy files
step "Installing to ${INSTALL_PATH}/addons..."
mkdir -p "${INSTALL_PATH}/addons"
cp -r "${TEMP_DIR}/addons/." "${INSTALL_PATH}/addons/"

# Create metaplugins.ini with GoStrike entry
step "Configuring metaplugins.ini..."
METAPLUGINS_INI="${INSTALL_PATH}/addons/metamod/metaplugins.ini"

cat > "${METAPLUGINS_INI}" << 'EOF'
; Metamod:Source Plugin List
; Format: <path to plugin> or addons/metamod/<plugin>.vdf
;
; GoStrike - Go Plugin Framework for CS2
addons/metamod/gostrike.vdf
EOF

# Create GoStrike VDF file in metamod directory (correct location for CS2)
step "Creating GoStrike VDF..."
GOSTRIKE_DIR="${INSTALL_PATH}/addons/gostrike"
mkdir -p "${GOSTRIKE_DIR}"
mkdir -p "${GOSTRIKE_DIR}/bin"

# VDF file goes in metamod directory, path without .so extension
cat > "${INSTALL_PATH}/addons/metamod/gostrike.vdf" << 'EOF'
"Metamod Plugin"
{
	"alias"		"gostrike"
	"file"		"addons/gostrike/gostrike"
}
EOF

# Patch gameinfo.gi to load Metamod (if not already patched)
step "Patching gameinfo.gi..."
GAMEINFO_PATH="${INSTALL_PATH}/gameinfo.gi"

if [ -f "${GAMEINFO_PATH}" ]; then
    # Check if already patched
    if grep -q "metamod" "${GAMEINFO_PATH}" 2>/dev/null; then
        info "gameinfo.gi already patched for Metamod"
    else
        # Backup original
        cp "${GAMEINFO_PATH}" "${GAMEINFO_PATH}.backup"
        
        # Add Metamod entry after "Game_LowViolence" line
        sed -i '/Game_LowViolence/a\			Game	csgo/addons/metamod' "${GAMEINFO_PATH}"
        info "Patched gameinfo.gi"
    fi
else
    warn "gameinfo.gi not found - CS2 may not be fully installed yet"
fi

# Set permissions
step "Setting permissions..."
chown -R 1000:1000 "${INSTALL_PATH}/addons" 2>/dev/null || true

echo ""
info "=========================================="
info "Metamod:Source installed successfully!"
info "=========================================="
info ""
info "Next steps:"
info "  1. Build GoStrike:  make go native"
info "  2. Deploy plugin:   make deploy"
info "  3. Start server:    make server-start"
