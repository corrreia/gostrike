# GoStrike Makefile
# Build targets for Go runtime, native Metamod plugin, and Docker server management

.PHONY: all build go native go-host native-host native-stub native-proto native-clean native-dev \
        clean test fmt lint install submodules info \
        server-init server-start server-stop server-restart server-logs server-console server-shell server-status server-clean \
        metamod-install deploy setup dev help

# Paths - default to external/ submodules (can be overridden)
METAMOD_PATH ?= $(CURDIR)/external/metamod-source
HL2SDK_PATH ?= $(CURDIR)/external/hl2sdk-cs2
CS2_PATH ?= /opt/cs2-server/game/csgo

# Docker configuration
DOCKER_COMPOSE := docker compose -f docker/docker-compose.yml
DOCKER_DATA := docker/data/cs2
DOCKER_CONTAINER := gostrike-cs2

# Default target - use Docker build for GLIBC compatibility
all: build

# Initialize/update git submodules
submodules:
	git submodule update --init --recursive

# ==============================================================================
# Build Targets (Docker - Recommended)
# ==============================================================================
# These targets build inside Docker containers with the correct GLIBC version
# for compatibility with the CS2 server's Steam Runtime environment.

# Build both Go and native plugins in Docker with FULL SDK (RECOMMENDED)
build:
	@echo "Building GoStrike in Docker (GLIBC compatible, full SDK)..."
	./scripts/docker-build.sh
	@echo ""
	@echo "Build complete! Deploy with: make deploy"

# Build with stub SDK (faster, for development only)
build-stub:
	@echo "Building GoStrike in Docker (GLIBC compatible, stub SDK)..."
	./scripts/docker-build.sh --stub
	@echo ""
	@echo "Build complete! Deploy with: make deploy"

# Aliases for convenience - these all use Docker builds
go: build
native: build

# ==============================================================================
# Build Targets (Host - Advanced)
# ==============================================================================
# WARNING: Host builds may have GLIBC compatibility issues with the CS2 server.
# Only use these if you know your host GLIBC matches the server environment.

# Build Go shared library on host (may have GLIBC issues)
go-host:
	@echo "Building Go shared library on host..."
	@echo "WARNING: This may have GLIBC compatibility issues with CS2 server"
	mkdir -p build
	CGO_ENABLED=1 go build -buildmode=c-shared -o build/libgostrike_go.so ./cmd/gostrike
	@echo "Built: build/libgostrike_go.so"

# Build Go shared library with debug symbols on host
go-debug:
	@echo "Building Go shared library (debug) on host..."
	@echo "WARNING: This may have GLIBC compatibility issues with CS2 server"
	mkdir -p build
	CGO_ENABLED=1 go build -buildmode=c-shared -gcflags="all=-N -l" -o build/libgostrike_go.so ./cmd/gostrike
	@echo "Built: build/libgostrike_go.so (debug)"

# Build native Metamod plugin on host with full SDK (may have GLIBC issues)
native-host:
	@echo "Building native Metamod plugin on host..."
	@echo "WARNING: This may have GLIBC compatibility issues with CS2 server"
	mkdir -p build/native
	cd build/native && cmake ../../native \
		-DMETAMOD_PATH=$(METAMOD_PATH) \
		-DHL2SDK_PATH=$(HL2SDK_PATH) \
		-DCMAKE_BUILD_TYPE=Release
	$(MAKE) -C build/native
	@echo "Built: build/native/gostrike.so"

# Build native plugin with stub SDK on host (may have GLIBC issues)
native-stub:
	@echo "Building native Metamod plugin (stub SDK) on host..."
	@echo "WARNING: This may have GLIBC compatibility issues with CS2 server"
	mkdir -p build/native
	cd build/native && cmake ../../native \
		-DUSE_STUB_SDK=ON \
		-DCMAKE_BUILD_TYPE=Release
	$(MAKE) -C build/native
	@echo "Built: build/native/gostrike.so (stub)"

# Generate protobuf headers from SDK (required for full SDK build)
native-proto:
	@echo "Generating protobuf headers from SDK..."
	@echo "This builds protoc from SDK's bundled protobuf 3.21.8 (first run only)"
	./native/scripts/generate_protos.sh
	@echo ""
	@echo "Protobuf headers generated in native/generated/"

# Clean native build and generated protos
native-clean:
	rm -rf build/native
	rm -rf native/build
	rm -rf native/protobuf-build
	rm -rf native/generated
	@echo "Cleaned native build artifacts"

# Quick native development build (stub SDK, faster)
native-dev: native-stub
	@if [ -d "$(DOCKER_DATA)/game/csgo/addons/gostrike" ]; then \
		echo "Deploying native plugin..."; \
		cp build/native/gostrike.so $(DOCKER_DATA)/game/csgo/addons/gostrike/; \
		echo "Deployed. Restart server with: make server-restart"; \
	else \
		echo "Server not set up. Run 'make setup' first."; \
	fi

# Clean build artifacts
clean:
	rm -rf build
	@echo "Cleaned build directory"

# Go packages to test/lint (excludes external/ and docker/)
GO_PACKAGES := ./cmd/... ./internal/... ./pkg/...

# Run tests
test:
	go test -v $(GO_PACKAGES)

# Run tests with race detection
test-race:
	CGO_ENABLED=1 go test -race -v $(GO_PACKAGES)

# Format code
fmt:
	go fmt $(GO_PACKAGES)

# Lint code
lint:
	go vet $(GO_PACKAGES)

# Install to CS2 server
install: go
	@echo "Installing to $(CS2_PATH)..."
	mkdir -p $(CS2_PATH)/addons/gostrike/bin
	mkdir -p $(CS2_PATH)/addons/gostrike/configs
	cp build/libgostrike_go.so $(CS2_PATH)/addons/gostrike/bin/
	-cp configs/gostrike.json $(CS2_PATH)/addons/gostrike/configs/
	@echo "Installation complete"

# Show project info
info:
	@echo "GoStrike Build Info"
	@echo "==================="
	@echo "Go version: $$(go version)"
	@echo "Architecture: $$(uname -m)"
	@go list -m

# Run all checks before commit
check: fmt lint test
	@echo "All checks passed!"

# ==============================================================================
# Docker Server Management
# ==============================================================================

# Initialize CS2 server (first-time download, ~60GB)
server-init:
	@echo "Starting CS2 server for initial download..."
	@echo "This will download ~60GB of game files. Please wait."
	@echo ""
	@echo "Creating data directory with correct permissions..."
	mkdir -p $(DOCKER_DATA)
	chown -R 1000:1000 $(DOCKER_DATA) 2>/dev/null || sudo chown -R 1000:1000 $(DOCKER_DATA) || true
	$(DOCKER_COMPOSE) up -d cs2-server
	@echo ""
	@echo "Server starting. Monitor progress with: make server-logs"
	@echo "Wait for 'VAC secure mode is activated' message before proceeding."

# Start CS2 server
server-start:
	@echo "Starting CS2 server..."
	$(DOCKER_COMPOSE) up -d cs2-server
	@echo "Server started. View logs with: make server-logs"

# Stop CS2 server
server-stop:
	@echo "Stopping CS2 server..."
	$(DOCKER_COMPOSE) down
	@echo "Server stopped."

# Restart CS2 server (for plugin changes)
server-restart:
	@echo "Restarting CS2 server..."
	$(DOCKER_COMPOSE) restart cs2-server
	@echo "Server restarted. View logs with: make server-logs"

# View server logs (follow mode)
server-logs:
	$(DOCKER_COMPOSE) logs -f cs2-server

# Attach to CS2 server console (Ctrl+C detaches without stopping)
server-console:
	@echo "Attaching to CS2 console... (Ctrl+C to detach)"
	docker attach $(DOCKER_CONTAINER)

# Open shell in server container
server-shell:
	docker exec -it $(DOCKER_CONTAINER) bash

# Check server status
server-status:
	@echo "=== Docker Container Status ==="
	@docker ps -a --filter "name=$(DOCKER_CONTAINER)" --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}" 2>/dev/null || echo "Container not found"
	@echo ""
	@echo "=== Data Directory Status ==="
	@if [ -d "$(DOCKER_DATA)" ]; then \
		echo "Path: $(DOCKER_DATA)"; \
		echo "Size: $$(du -sh $(DOCKER_DATA) 2>/dev/null | cut -f1 || echo 'unknown')"; \
	else \
		echo "Data directory not created yet"; \
	fi
	@echo ""
	@echo "=== GoStrike Installation Check ==="
	@if [ -f "$(DOCKER_DATA)/game/csgo/addons/metamod.vdf" ]; then echo "[OK] Metamod installed"; else echo "[--] Metamod NOT installed"; fi
	@if [ -f "$(DOCKER_DATA)/game/csgo/addons/gostrike/gostrike.so" ]; then echo "[OK] GoStrike native plugin"; else echo "[--] GoStrike native plugin NOT found"; fi
	@if [ -f "$(DOCKER_DATA)/game/csgo/addons/gostrike/bin/libgostrike_go.so" ]; then echo "[OK] GoStrike Go library"; else echo "[--] GoStrike Go library NOT found"; fi
	@if [ -f "$(DOCKER_DATA)/game/csgo/gameinfo.gi" ]; then echo "[OK] CS2 installed"; else echo "[--] CS2 NOT installed (still downloading?)"; fi

# Delete CS2 server data (WARNING: deletes ~60GB of data)
server-clean:
	@echo "WARNING: This will delete the CS2 server data (~60GB)!"
	@echo "Stopping containers first..."
	-$(DOCKER_COMPOSE) down 2>/dev/null || true
	@echo "Removing data directory..."
	rm -rf $(DOCKER_DATA) && echo "Data deleted." || echo "Could not delete (try with sudo)"

# ==============================================================================
# Plugin Installation
# ==============================================================================

# Install Metamod:Source to the server data directory
metamod-install:
	@echo "Installing Metamod:Source..."
	./docker/scripts/install-metamod.sh --path $(DOCKER_DATA)

# Deploy GoStrike to the server data directory
deploy:
	@echo "Deploying GoStrike to server..."
	@echo "Creating directories..."
	@mkdir -p $(DOCKER_DATA)/game/csgo/addons/gostrike/bin
	@mkdir -p $(DOCKER_DATA)/game/csgo/addons/gostrike/configs
	@if [ -f build/libgostrike_go.so ]; then \
		echo "Copying Go library..."; \
		cp build/libgostrike_go.so $(DOCKER_DATA)/game/csgo/addons/gostrike/bin/; \
	else \
		echo "ERROR: Go library not found. Run 'make build' first."; \
		exit 1; \
	fi
	@echo "Copying config..."
	@cp configs/gostrike.json $(DOCKER_DATA)/game/csgo/addons/gostrike/configs/ 2>/dev/null || true
	@if [ -f build/native/gostrike.so ]; then \
		echo "Copying native plugin..."; \
		cp build/native/gostrike.so $(DOCKER_DATA)/game/csgo/addons/gostrike/; \
	else \
		echo "ERROR: Native plugin not found. Run 'make build' first."; \
		exit 1; \
	fi
	@chown -R 1000:1000 $(DOCKER_DATA)/game/csgo/addons/gostrike 2>/dev/null || true
	@echo "Done!"
	@echo ""
	@echo "Deployed. Restart server with: make server-restart"

# ==============================================================================
# Combined Workflows
# ==============================================================================

# Full first-time setup
setup: server-init
	@echo ""
	@echo "=========================================="
	@echo "  CS2 Server Initializing"
	@echo "=========================================="
	@echo ""
	@echo "The server is now downloading CS2 (~60GB)."
	@echo ""
	@echo "Next steps:"
	@echo "  1. Watch logs:           make server-logs"
	@echo "  2. Wait for download to complete (look for 'VAC secure mode')"
	@echo "  3. Stop the server:      make server-stop"
	@echo "  4. Install Metamod:      make metamod-install"
	@echo "  5. Build GoStrike:       make build"
	@echo "  6. Deploy plugin:        make deploy"
	@echo "  7. Start server:         make server-start"
	@echo ""
	@echo "Or after download completes, run: make server-stop metamod-install build deploy server-start"

# Development workflow: build, deploy, restart
dev: build deploy server-restart
	@echo ""
	@echo "Development cycle complete!"
	@echo "View logs with: make server-logs"

# Quick rebuild and deploy (no restart)
quick: build deploy
	@echo "Built and deployed. Restart server to apply changes: make server-restart"

# Clean everything including Docker volume
clean-all: clean server-clean
	@echo "All cleaned."

# Help
help:
	@echo "GoStrike Makefile"
	@echo "================="
	@echo ""
	@echo "Build Targets (Docker - Recommended):"
	@echo "  make build         - Build Go + native plugin in Docker (GLIBC compatible)"
	@echo "  make go            - Alias for 'make build'"
	@echo "  make native        - Alias for 'make build'"
	@echo "  make all           - Alias for 'make build'"
	@echo ""
	@echo "Build Targets (Host - Advanced):"
	@echo "  make go-host       - Build Go library on host (may have GLIBC issues)"
	@echo "  make native-host   - Build native plugin with full SDK on host"
	@echo "  make native-stub   - Build native plugin with stub SDK on host"
	@echo "  make native-proto  - Generate protobuf headers from SDK"
	@echo "  make native-dev    - Build stub + deploy (quick native dev)"
	@echo "  make native-clean  - Clean native build and generated files"
	@echo ""
	@echo "Other:"
	@echo "  make clean         - Remove build artifacts"
	@echo "  make clean-all     - Remove build + Docker volume"
	@echo "  make test          - Run tests"
	@echo "  make submodules    - Initialize git submodules"
	@echo ""
	@echo "Docker Server Management:"
	@echo "  make server-init   - Start CS2 server (first-time download)"
	@echo "  make server-start  - Start the server"
	@echo "  make server-stop   - Stop the server"
	@echo "  make server-restart- Restart for plugin changes"
	@echo "  make server-logs   - View server logs (follow)"
	@echo "  make server-console- Attach to CS2 console"
	@echo "  make server-shell  - Bash shell into container"
	@echo "  make server-status - Check server and plugin status"
	@echo "  make server-clean  - Delete CS2 data (~60GB)"
	@echo ""
	@echo "Plugin Installation:"
	@echo "  make metamod-install - Install Metamod:Source to volume"
	@echo "  make deploy          - Deploy GoStrike to server"
	@echo ""
	@echo "Workflows:"
	@echo "  make setup         - First-time setup (starts CS2 download)"
	@echo "  make dev           - Build + deploy + restart (dev loop)"
	@echo "  make quick         - Build + deploy (no restart)"
	@echo ""
	@echo "First-time setup:"
	@echo "  1. make setup                    # Start CS2 download"
	@echo "  2. make server-logs              # Wait for 'VAC secure mode'"
	@echo "  3. make server-stop              # Stop server"
	@echo "  4. make metamod-install          # Install Metamod"
	@echo "  5. make build deploy             # Build and deploy GoStrike"
	@echo "  6. make server-start             # Start server with plugin"
	@echo ""
	@echo "Development cycle:"
	@echo "  make dev                         # Build, deploy, restart"
	@echo ""
	@echo "Variables:"
	@echo "  DOCKER_DATA=$(DOCKER_DATA)"
	@echo "  DOCKER_CONTAINER=$(DOCKER_CONTAINER)"
