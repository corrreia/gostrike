#!/bin/bash
# Generate protobuf headers from HL2SDK .proto files
# 
# IMPORTANT: The headers MUST be generated with protobuf 3.21.8 (the version
# bundled with HL2SDK). System protobuf versions (especially v4+/v5+/v33+)
# generate incompatible code.
#
# This script will build protoc from SDK's bundled source if needed.

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
NATIVE_DIR="$(dirname "$SCRIPT_DIR")"
HL2SDK_PATH="${HL2SDK_PATH:-$NATIVE_DIR/../external/hl2sdk-cs2}"
PROTOBUF_SRC="$HL2SDK_PATH/thirdparty/protobuf-3.21.8"

# Output directory for generated files
OUTPUT_DIR="$NATIVE_DIR/generated"
mkdir -p "$OUTPUT_DIR"

# Build directory for protoc
PROTOC_BUILD_DIR="$NATIVE_DIR/protobuf-build"
PROTOC_BIN="$PROTOC_BUILD_DIR/protoc"

echo "GoStrike Protobuf Header Generator"
echo "==================================="
echo "  HL2SDK path: $HL2SDK_PATH"
echo "  Output dir:  $OUTPUT_DIR"
echo ""

# Check if SDK protobuf source exists
if [ ! -d "$PROTOBUF_SRC" ]; then
    echo "ERROR: SDK protobuf source not found at $PROTOBUF_SRC"
    exit 1
fi

# Build protoc from SDK's bundled protobuf if not already built
if [ ! -x "$PROTOC_BIN" ]; then
    echo "Building protoc from SDK's protobuf 3.21.8..."
    echo "  This may take a few minutes on first run."
    echo ""
    
    mkdir -p "$PROTOC_BUILD_DIR"
    cd "$PROTOC_BUILD_DIR"
    
    # Configure and build protoc
    cmake "$PROTOBUF_SRC/cmake" \
        -Dprotobuf_BUILD_TESTS=OFF \
        -Dprotobuf_BUILD_SHARED_LIBS=OFF \
        -DCMAKE_BUILD_TYPE=Release \
        -Dprotobuf_BUILD_PROTOC_BINARIES=ON \
        2>&1 | tail -20
    
    make -j$(nproc) protoc 2>&1 | tail -20
    
    if [ ! -x "$PROTOC_BIN" ]; then
        echo "ERROR: Failed to build protoc"
        exit 1
    fi
    
    echo ""
    echo "Successfully built protoc from SDK source"
    cd "$NATIVE_DIR"
fi

PROTOC_VERSION=$("$PROTOC_BIN" --version)
echo "Using: $PROTOC_VERSION"
echo ""

# Proto source directories
PROTO_COMMON="$HL2SDK_PATH/common"
PROTO_SHARED="$HL2SDK_PATH/game/shared"
PROTO_GOOGLE="$PROTOBUF_SRC/src"  # For google/protobuf/descriptor.proto

# Check if proto files exist
if [ ! -f "$PROTO_COMMON/network_connection.proto" ]; then
    echo "ERROR: network_connection.proto not found at $PROTO_COMMON"
    exit 1
fi

# Common protoc arguments - include paths for all dependencies
PROTOC_ARGS="-I$PROTO_GOOGLE -I$PROTO_COMMON -I$PROTO_SHARED"

# Generate protobuf headers
echo "Compiling network_connection.proto..."
"$PROTOC_BIN" --cpp_out="$OUTPUT_DIR" $PROTOC_ARGS \
    "$PROTO_COMMON/network_connection.proto"

echo "Compiling networkbasetypes.proto..."
"$PROTOC_BIN" --cpp_out="$OUTPUT_DIR" $PROTOC_ARGS \
    "$PROTO_COMMON/networkbasetypes.proto"

echo "Compiling source2_steam_stats.proto..."
"$PROTOC_BIN" --cpp_out="$OUTPUT_DIR" $PROTOC_ARGS \
    "$PROTO_COMMON/source2_steam_stats.proto"

echo "Compiling netmessages.proto..."
"$PROTOC_BIN" --cpp_out="$OUTPUT_DIR" $PROTOC_ARGS \
    "$PROTO_COMMON/netmessages.proto"

echo "Compiling engine_gcmessages.proto..."
"$PROTOC_BIN" --cpp_out="$OUTPUT_DIR" $PROTOC_ARGS \
    "$PROTO_COMMON/engine_gcmessages.proto"

# Now compile game/shared protos (depend on common)
if [ -f "$PROTO_SHARED/usermessages.proto" ]; then
    echo "Compiling usermessages.proto..."
    "$PROTOC_BIN" --cpp_out="$OUTPUT_DIR" $PROTOC_ARGS \
        "$PROTO_SHARED/usermessages.proto"
fi

echo ""
echo "Generated files:"
ls -la "$OUTPUT_DIR"/*.pb.h 2>/dev/null | head -10 || echo "  No .pb.h files generated"
echo ""
TOTAL_H=$(ls -1 "$OUTPUT_DIR"/*.pb.h 2>/dev/null | wc -l)
TOTAL_CC=$(ls -1 "$OUTPUT_DIR"/*.pb.cc 2>/dev/null | wc -l)
echo "  Total: $TOTAL_H .pb.h files, $TOTAL_CC .pb.cc files"

echo ""
echo "Done! Now rebuild the native plugin with:"
echo "  cd $NATIVE_DIR && rm -rf build && mkdir build && cd build"
echo "  cmake -DHL2SDK_PATH=$HL2SDK_PATH -DMETAMOD_PATH=\$METAMOD_PATH .."
echo "  make"
