# GoStrike Native Plugin CMakeLists.txt
# Builds the Metamod:Source plugin that hosts the Go runtime
#
# Architecture inspired by CounterStrikeSharp (https://github.com/roflmuffin/CounterStrikeSharp)
# Protobuf cmake approach based on Poggicek's work via CSSharp

cmake_minimum_required(VERSION 3.16)
project(gostrike VERSION 0.1.0 LANGUAGES CXX C)

# C++ standard
set(CMAKE_CXX_STANDARD 20)
set(CMAKE_CXX_STANDARD_REQUIRED ON)
set(CMAKE_POSITION_INDEPENDENT_CODE ON)

# SDK paths (can be overridden)
set(METAMOD_PATH "" CACHE PATH "Path to Metamod:Source")
set(HL2SDK_PATH "" CACHE PATH "Path to HL2SDK")

# Output name
set(PLUGIN_NAME "gostrike")

# Option to force stub SDK usage (for quick dev builds only)
option(USE_STUB_SDK "Use stub SDK headers instead of real SDKs" OFF)

# ============================================================
# funchook - function hooking library (for Host_Say detour)
# Same approach as CounterStrikeSharp
# ============================================================
set(FUNCHOOK_BUILD_TESTS OFF CACHE BOOL "Disable building tests for funchook." FORCE)
set(FUNCHOOK_BUILD_SHARED OFF CACHE BOOL "Disable shared funchook library." FORCE)
add_subdirectory(thirdparty/funchook)
set_property(TARGET funchook-static PROPERTY POSITION_INDEPENDENT_CODE ON)

# ============================================================
# Detect SDK availability
# ============================================================

set(HAS_FULL_SDK FALSE)
if(NOT USE_STUB_SDK)
    if(EXISTS "${METAMOD_PATH}/core/ISmmPlugin.h" AND EXISTS "${HL2SDK_PATH}/public/eiface.h")
        set(HAS_FULL_SDK TRUE)
    else()
        message(STATUS "SDK headers not found at specified paths, falling back to stub SDK")
        message(STATUS "  METAMOD_PATH=${METAMOD_PATH}")
        message(STATUS "  HL2SDK_PATH=${HL2SDK_PATH}")
    endif()
endif()

# ============================================================
# Protobuf generation (required for full SDK builds)
# ============================================================

if(HAS_FULL_SDK)
    # Use the SDK's bundled protoc binary
    set(PROTOC_EXECUTABLE ${HL2SDK_PATH}/devtools/bin/linux/protoc)

    # Proto files we need to compile (SDK common protos + usermessages for TextMsg)
    file(GLOB PROTO_COMMON "${HL2SDK_PATH}/common/*.proto")
    set(PROTO_TARGETS ${PROTO_COMMON}
        "${HL2SDK_PATH}/game/shared/usermessages.proto"
    )

    # Build output list
    set(PROTO_OUTPUT "")
    set(PROTO_INPUT "")
    foreach(PROTO_TARGET ${PROTO_TARGETS})
        get_filename_component(PROTO_FILENAME ${PROTO_TARGET} NAME_WLE)
        list(APPEND PROTO_OUTPUT
            ${CMAKE_CURRENT_BINARY_DIR}/protobuf/${PROTO_FILENAME}.pb.cc
            ${CMAKE_CURRENT_BINARY_DIR}/protobuf/${PROTO_FILENAME}.pb.h
        )
        list(APPEND PROTO_INPUT ${PROTO_FILENAME}.proto)
    endforeach()

    file(MAKE_DIRECTORY ${CMAKE_CURRENT_BINARY_DIR}/protobuf)

    # Generate protobuf sources
    add_custom_command(
        OUTPUT ${PROTO_OUTPUT}
        COMMAND "${PROTOC_EXECUTABLE}"
            -I ${HL2SDK_PATH}/thirdparty/protobuf-3.21.8/src
            --proto_path=${HL2SDK_PATH}/common
            --proto_path=${HL2SDK_PATH}/game/shared
            --cpp_out=${CMAKE_CURRENT_BINARY_DIR}/protobuf
            ${PROTO_INPUT}
        COMMENT "Generating protobuf files from HL2SDK protos"
        DEPENDS ${PROTO_TARGETS}
    )

    # Create static protobuf library
    add_library(GoStrikeProtobufs STATIC ${PROTO_OUTPUT})
    target_include_directories(GoStrikeProtobufs
        PUBLIC ${CMAKE_CURRENT_BINARY_DIR}/protobuf
        PUBLIC ${HL2SDK_PATH}/thirdparty/protobuf-3.21.8/src
    )
    target_link_libraries(GoStrikeProtobufs
        PUBLIC ${HL2SDK_PATH}/lib/linux64/release/libprotobuf.a
    )
    set_target_properties(GoStrikeProtobufs PROPERTIES LINKER_LANGUAGE CXX)
    # Suppress warnings in generated code
    target_compile_options(GoStrikeProtobufs PRIVATE -w)
endif()

# ============================================================
# Source files
# ============================================================

set(SOURCES
    src/gostrike.cpp
    src/go_bridge.cpp
    src/memory_module.cpp
    src/gameconfig.cpp
    src/schema.cpp
    src/entity_system.cpp
    src/convar_manager.cpp
    src/player_manager.cpp
    src/game_functions.cpp
    src/chat_manager.cpp
)

# SDK source files needed for linking (same pattern as CSSharp)
if(HAS_FULL_SDK)
    list(APPEND SOURCES
        ${HL2SDK_PATH}/tier1/convar.cpp
        ${HL2SDK_PATH}/tier1/generichash.cpp
        ${HL2SDK_PATH}/tier1/keyvalues3.cpp
        ${HL2SDK_PATH}/entity2/entityidentity.cpp
        ${HL2SDK_PATH}/entity2/entitysystem.cpp
        ${HL2SDK_PATH}/entity2/entitykeyvalues.cpp
        ${METAMOD_PATH}/core/sourcehook/sourcehook.cpp
        ${METAMOD_PATH}/core/sourcehook/sourcehook_impl_chookidman.cpp
        ${METAMOD_PATH}/core/sourcehook/sourcehook_impl_chookmaninfo.cpp
        ${METAMOD_PATH}/core/sourcehook/sourcehook_impl_cvfnptr.cpp
        ${METAMOD_PATH}/core/sourcehook/sourcehook_impl_cproto.cpp
    )
endif()

set(HEADERS
    src/gostrike.h
    src/go_bridge.h
    src/memory_module.h
    src/gameconfig.h
    src/schema.h
    src/entity_system.h
    src/convar_manager.h
    src/player_manager.h
    src/game_functions.h
    src/chat_manager.h
    src/utils.h
    include/gostrike_abi.h
)

# Create shared library
add_library(${PLUGIN_NAME} SHARED ${SOURCES} ${HEADERS})

# ============================================================
# Include directories and defines
# ============================================================

# Always include our own headers
target_include_directories(${PLUGIN_NAME} PRIVATE
    ${CMAKE_CURRENT_SOURCE_DIR}/include
    ${CMAKE_CURRENT_SOURCE_DIR}/src
    ${CMAKE_CURRENT_SOURCE_DIR}/thirdparty
    ${CMAKE_CURRENT_SOURCE_DIR}/thirdparty/funchook/include
)

if(HAS_FULL_SDK)
    message(STATUS "Building with FULL SDK")
    message(STATUS "  Metamod:Source at ${METAMOD_PATH}")
    message(STATUS "  HL2SDK at ${HL2SDK_PATH}")

    target_include_directories(${PLUGIN_NAME} PRIVATE
        # SDK protobuf (must come before system protobuf)
        ${HL2SDK_PATH}/thirdparty/protobuf-3.21.8/src
        # Generated protobuf headers
        ${CMAKE_CURRENT_BINARY_DIR}/protobuf
        # HL2SDK headers
        ${HL2SDK_PATH}
        ${HL2SDK_PATH}/common
        ${HL2SDK_PATH}/game/shared
        ${HL2SDK_PATH}/game/server
        ${HL2SDK_PATH}/public
        ${HL2SDK_PATH}/public/engine
        ${HL2SDK_PATH}/public/mathlib
        ${HL2SDK_PATH}/public/tier0
        ${HL2SDK_PATH}/public/tier1
        ${HL2SDK_PATH}/public/entity2
        ${HL2SDK_PATH}/public/game/server
        ${HL2SDK_PATH}/public/schemasystem
        ${HL2SDK_PATH}/public/networksystem
        ${HL2SDK_PATH}/public/vstdlib
        # Metamod headers
        ${METAMOD_PATH}/core
        ${METAMOD_PATH}/core/sourcehook
    )

    target_compile_definitions(${PLUGIN_NAME} PRIVATE
        HAVE_METAMOD=1
        HAVE_HL2SDK=1
        META_IS_SOURCE2=1
        _LINUX
        POSIX
        LINUX
        GNUC
        COMPILER_GCC
        PLATFORM_64BITS
        _FILE_OFFSET_BITS=64
        _GLIBCXX_USE_CXX11_ABI=0
    )

    # Link SDK libraries
    target_link_libraries(${PLUGIN_NAME} PRIVATE
        ${HL2SDK_PATH}/lib/linux64/libtier0.so
        ${HL2SDK_PATH}/lib/linux64/tier1.a
        ${HL2SDK_PATH}/lib/linux64/interfaces.a
        ${HL2SDK_PATH}/lib/linux64/mathlib.a
        GoStrikeProtobufs
    )
else()
    message(STATUS "Building with STUB SDK (development build)")

    target_include_directories(${PLUGIN_NAME} PRIVATE
        ${CMAKE_CURRENT_SOURCE_DIR}/include/stub
    )
    target_compile_definitions(${PLUGIN_NAME} PRIVATE
        USE_STUB_SDK=1
    )
endif()

# ============================================================
# Compiler flags (matching CSSharp's linux.base.cmake)
# ============================================================

if(CMAKE_CXX_COMPILER_ID MATCHES "GNU|Clang")
    target_compile_options(${PLUGIN_NAME} PRIVATE
        -Wall
        -Wno-uninitialized
        -Wno-switch
        -Wno-unused
        -Wno-non-virtual-dtor
        -Wno-overloaded-virtual
        -Wno-conversion-null
        -Wno-write-strings
        -Wno-invalid-offsetof
        -Wno-reorder
        -fno-strict-aliasing
        -fno-threadsafe-statics
        -fvisibility=default
        -mfpmath=sse
        -msse
    )

    # String comparison compatibility macros
    add_definitions(
        -Dstricmp=strcasecmp
        -D_stricmp=strcasecmp
        -D_strnicmp=strncasecmp
        -Dstrnicmp=strncasecmp
        -D_snprintf=snprintf
        -D_vsnprintf=vsnprintf
        -D_alloca=alloca
        -Dstrcmpi=strcasecmp
    )

    if(CMAKE_SYSTEM_NAME STREQUAL "Linux")
        # Exclude protobuf symbols from export
        target_link_options(${PLUGIN_NAME} PRIVATE
            -Wl,--exclude-libs=libprotobuf.a
        )
    endif()
endif()

# ============================================================
# Common settings
# ============================================================

target_compile_definitions(${PLUGIN_NAME} PRIVATE
    GOSTRIKE_VERSION="${PROJECT_VERSION}"
    SOURCE_ENGINE=28
)

# Link system libraries
target_link_libraries(${PLUGIN_NAME} PRIVATE
    dl
    pthread
    funchook-static
)

# Output settings
set_target_properties(${PLUGIN_NAME} PROPERTIES
    PREFIX ""
    SUFFIX ".so"
    OUTPUT_NAME "${PLUGIN_NAME}"
    LIBRARY_OUTPUT_DIRECTORY "${CMAKE_BINARY_DIR}"
)

# Install target
install(TARGETS ${PLUGIN_NAME}
    LIBRARY DESTINATION lib
)

# Print configuration summary
message(STATUS "")
message(STATUS "GoStrike Native Plugin Configuration:")
message(STATUS "  Version:     ${PROJECT_VERSION}")
message(STATUS "  Plugin name: ${PLUGIN_NAME}.so")
message(STATUS "  SDK mode:    ${HAS_FULL_SDK}")
message(STATUS "  Metamod:     ${METAMOD_PATH}")
message(STATUS "  HL2SDK:      ${HL2SDK_PATH}")
message(STATUS "")
