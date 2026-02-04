# GoStrike Test Environment Dockerfile
# Creates a CS2 dedicated server with GoStrike installed for testing

FROM cm2network/steamcmd:root AS builder

# Install build dependencies
RUN apt-get update && apt-get install -y \
    golang-go \
    git \
    cmake \
    build-essential \
    && rm -rf /var/lib/apt/lists/*

# Set up Go environment
ENV GOPATH=/go
ENV PATH=$PATH:/usr/local/go/bin:$GOPATH/bin

# Copy source code
WORKDIR /build
COPY . .

# Build Go shared library
RUN CGO_ENABLED=1 go build -buildmode=c-shared \
    -o /build/output/libgostrike_go.so \
    ./cmd/gostrike

# ==================================================
# Runtime image
# ==================================================
FROM cm2network/steamcmd:root

# Install CS2 dedicated server
USER steam
RUN /home/steam/steamcmd/steamcmd.sh \
    +force_install_dir /home/steam/cs2 \
    +login anonymous \
    +app_update 730 validate \
    +quit

USER root

# Create GoStrike directories
RUN mkdir -p /home/steam/cs2/game/csgo/addons/gostrike/bin \
    && mkdir -p /home/steam/cs2/game/csgo/addons/gostrike/configs \
    && chown -R steam:steam /home/steam/cs2/game/csgo/addons

# Copy GoStrike files
COPY --from=builder --chown=steam:steam /build/output/libgostrike_go.so \
    /home/steam/cs2/game/csgo/addons/gostrike/bin/

COPY --chown=steam:steam configs/gostrike.json \
    /home/steam/cs2/game/csgo/addons/gostrike/configs/

# Note: Native plugin would be copied here when built
# COPY --chown=steam:steam native/build/gostrike.so \
#     /home/steam/cs2/game/csgo/addons/metamod/

# Set up server configuration
WORKDIR /home/steam/cs2

# Expose ports
EXPOSE 27015/tcp
EXPOSE 27015/udp
EXPOSE 27020/udp

USER steam

# Default command - start CS2 server
CMD ["./game/bin/linuxsteamrt64/cs2", \
    "-dedicated", \
    "-console", \
    "-usercon", \
    "+game_type", "0", \
    "+game_mode", "1", \
    "+mapgroup", "mg_active", \
    "+map", "de_dust2"]
