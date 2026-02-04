# GoStrike Test Environment Dockerfile
# Creates a CS2 dedicated server with GoStrike installed for testing

# ==================================================
# Builder stage - compile Go library
# ==================================================
FROM golang:1.21-bookworm AS builder

# Install build dependencies
RUN apt-get update && apt-get install -y \
    git \
    && rm -rf /var/lib/apt/lists/*

# Copy source code
WORKDIR /build
COPY . .

# Build Go shared library
RUN CGO_ENABLED=1 go build -buildmode=c-shared \
    -o /build/output/libgostrike_go.so \
    ./cmd/gostrike

# ==================================================
# Runtime image - CS2 dedicated server
# ==================================================
FROM joedwards32/cs2

# Switch to root for setup
USER root

# Create GoStrike directories
RUN mkdir -p /home/steam/cs2-dedicated/game/csgo/addons/gostrike/bin \
    && mkdir -p /home/steam/cs2-dedicated/game/csgo/addons/gostrike/configs \
    && mkdir -p /home/steam/cs2-dedicated/game/csgo/addons/metamod \
    && chown -R steam:steam /home/steam/cs2-dedicated/game/csgo/addons

# Copy GoStrike Go library
COPY --from=builder --chown=steam:steam /build/output/libgostrike_go.so \
    /home/steam/cs2-dedicated/game/csgo/addons/gostrike/bin/

# Copy configuration
COPY --chown=steam:steam configs/gostrike.json \
    /home/steam/cs2-dedicated/game/csgo/addons/gostrike/configs/

# Note: The native gostrike.so plugin would be copied here when ready:
# COPY --chown=steam:steam build/gostrike.so \
#     /home/steam/cs2-dedicated/game/csgo/addons/metamod/

# Switch back to steam user
USER steam

# Don't override WORKDIR or ENTRYPOINT - let base image handle it
