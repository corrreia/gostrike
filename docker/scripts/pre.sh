#!/bin/bash
# GoStrike Pre-Start Hook
# This script is SOURCED by the container entrypoint before CS2 server starts
# It verifies that Metamod and GoStrike are properly installed
#
# NOTE: Do NOT use 'exit' or 'set -e' - this script is sourced!

# Paths
CS2_PATH="/home/steam/cs2-dedicated"
CSGO_PATH="${CS2_PATH}/game/csgo"
ADDONS_PATH="${CSGO_PATH}/addons"
METAMOD_PATH="${ADDONS_PATH}/metamod"
GOSTRIKE_PATH="${ADDONS_PATH}/gostrike"

# Colors (may not work in all terminals)
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
BLUE='\033[0;34m'
NC='\033[0m'

echo ""
echo -e "${BLUE}=========================================${NC}"
echo -e "${BLUE}  GoStrike Pre-Start Check${NC}"
echo -e "${BLUE}=========================================${NC}"
echo ""

# Track status
WARNINGS=0
ERRORS=0

check_pass() {
    echo -e "  ${GREEN}[OK]${NC} $1"
}

check_warn() {
    echo -e "  ${YELLOW}[WARN]${NC} $1"
    ((WARNINGS++)) || true
}

check_fail() {
    echo -e "  ${RED}[FAIL]${NC} $1"
    ((ERRORS++)) || true
}

# Check Metamod installation
echo "Checking Metamod:Source..."
if [ -f "${ADDONS_PATH}/metamod.vdf" ]; then
    check_pass "metamod.vdf exists"
else
    check_fail "metamod.vdf not found - Metamod not installed"
    check_warn "Run 'make metamod-install' on the host"
fi

if [ -f "${METAMOD_PATH}/bin/linux64/server.so" ]; then
    check_pass "Metamod binary exists"
else
    check_fail "Metamod binary not found"
fi

if [ -f "${METAMOD_PATH}/metaplugins.ini" ]; then
    check_pass "metaplugins.ini exists"
    if grep -q "gostrike" "${METAMOD_PATH}/metaplugins.ini" 2>/dev/null; then
        check_pass "GoStrike registered in metaplugins.ini"
    else
        check_warn "GoStrike not in metaplugins.ini"
    fi
else
    check_warn "metaplugins.ini not found"
fi

# Check gameinfo.gi patch
echo ""
echo "Checking gameinfo.gi..."
if [ -f "${CSGO_PATH}/gameinfo.gi" ]; then
    if grep -q "addons/metamod" "${CSGO_PATH}/gameinfo.gi" 2>/dev/null; then
        check_pass "gameinfo.gi patched for Metamod"
    else
        check_fail "gameinfo.gi NOT patched - Metamod won't load"
        check_warn "Run 'make metamod-install' on the host"
    fi
else
    check_warn "gameinfo.gi not found (CS2 still downloading?)"
fi

# Check GoStrike installation
echo ""
echo "Checking GoStrike..."
# VDF file should be in metamod directory, not gostrike directory
if [ -f "${METAMOD_PATH}/gostrike.vdf" ]; then
    check_pass "gostrike.vdf exists (in metamod/)"
else
    check_warn "gostrike.vdf not found in metamod/"
fi

if [ -f "${GOSTRIKE_PATH}/gostrike.so" ]; then
    check_pass "gostrike.so (native plugin) exists"
else
    check_fail "gostrike.so not found - plugin won't load"
    check_warn "Run 'make deploy' on the host"
fi

if [ -f "${GOSTRIKE_PATH}/bin/libgostrike_go.so" ]; then
    check_pass "libgostrike_go.so (Go runtime) exists"
else
    check_fail "libgostrike_go.so not found"
    check_warn "Run 'make build deploy' on the host"
fi

if [ -f "${GOSTRIKE_PATH}/configs/gostrike.json" ]; then
    check_pass "gostrike.json config exists"
else
    check_warn "gostrike.json not found (using defaults)"
fi

# Summary
echo ""
echo -e "${BLUE}=========================================${NC}"
if [ $ERRORS -gt 0 ]; then
    echo -e "${RED}  Status: $ERRORS errors, $WARNINGS warnings${NC}"
    echo -e "${RED}  GoStrike may not load correctly!${NC}"
elif [ $WARNINGS -gt 0 ]; then
    echo -e "${YELLOW}  Status: $WARNINGS warnings${NC}"
    echo -e "${YELLOW}  Server starting with warnings...${NC}"
else
    echo -e "${GREEN}  Status: All checks passed!${NC}"
    echo -e "${GREEN}  GoStrike ready to load${NC}"
fi
echo -e "${BLUE}=========================================${NC}"
echo ""

# Don't block server startup even if there are errors
# The user may be intentionally running without plugins
# NOTE: Do NOT use 'exit' here - this script is sourced by the entrypoint
#       and 'exit' would kill the parent script before CS2 starts!
