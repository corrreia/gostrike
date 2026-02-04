# GoStrike Justfile
# Alternative task runner using just (https://github.com/casey/just)

# Default recipe
default: build

# Build Go shared library
build:
    @echo "Building Go shared library..."
    mkdir -p build
    CGO_ENABLED=1 go build -buildmode=c-shared -o build/libgostrike_go.so ./cmd/gostrike
    @echo "Built: build/libgostrike_go.so"

# Build with debug symbols
build-debug:
    @echo "Building Go shared library (debug)..."
    mkdir -p build
    CGO_ENABLED=1 go build -buildmode=c-shared -gcflags="all=-N -l" -o build/libgostrike_go.so ./cmd/gostrike
    @echo "Built: build/libgostrike_go.so (debug)"

# Build native plugin
build-native metamod_path="/opt/metamod-source" hl2sdk_path="/opt/hl2sdk-cs2":
    @echo "Building native Metamod plugin..."
    mkdir -p build/native
    cd build/native && cmake ../../native \
        -DMETAMOD_PATH={{metamod_path}} \
        -DHL2SDK_PATH={{hl2sdk_path}} \
        -DCMAKE_BUILD_TYPE=Release
    make -C build/native
    @echo "Built: build/native/gostrike.so"

# Clean build artifacts
clean:
    rm -rf build
    @echo "Cleaned build directory"

# Run tests
test:
    go test -v ./...

# Run tests with race detection
test-race:
    CGO_ENABLED=1 go test -race -v ./...

# Format code
fmt:
    go fmt ./...

# Lint code
lint:
    go vet ./...

# Install to server
install cs2_path="/opt/cs2-server/game/csgo": build
    @echo "Installing to {{cs2_path}}..."
    mkdir -p {{cs2_path}}/addons/gostrike/bin
    mkdir -p {{cs2_path}}/addons/gostrike/configs
    cp build/libgostrike_go.so {{cs2_path}}/addons/gostrike/bin/
    cp configs/gostrike.json {{cs2_path}}/addons/gostrike/configs/ || true
    @echo "Installation complete"

# Show project info
info:
    @echo "GoStrike Build Info"
    @echo "==================="
    @echo "Go version: $(go version)"
    @echo "Architecture: $(uname -m)"
    @go list -m

# Generate header from Go exports
gen-header: build
    @echo "Header generated at: build/libgostrike_go.h"

# Watch and rebuild on changes
watch:
    @echo "Watching for changes..."
    @which watchexec > /dev/null || (echo "Install watchexec: cargo install watchexec-cli" && exit 1)
    watchexec -e go -r just build

# Run all checks before commit
check: fmt lint test
    @echo "All checks passed!"
