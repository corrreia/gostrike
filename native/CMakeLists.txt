# GoStrike Native Plugin CMakeLists.txt
# Builds the Metamod:Source plugin that hosts the Go runtime

cmake_minimum_required(VERSION 3.16)
project(gostrike VERSION 0.1.0 LANGUAGES CXX C)

# C++ standard
set(CMAKE_CXX_STANDARD 17)
set(CMAKE_CXX_STANDARD_REQUIRED ON)
set(CMAKE_POSITION_INDEPENDENT_CODE ON)

# SDK paths (can be overridden)
set(METAMOD_PATH "/opt/metamod-source" CACHE PATH "Path to Metamod:Source")
set(HL2SDK_PATH "/opt/hl2sdk-cs2" CACHE PATH "Path to HL2SDK")

# Output name
set(PLUGIN_NAME "gostrike")

# Source files
set(SOURCES
    src/gostrike.cpp
    src/go_bridge.cpp
)

# NOTE: Generated protobuf sources (*.pb.cc) are NOT included here.
# They require matching protobuf library version - system protobuf often
# doesn't match the SDK's bundled version (3.21.8), causing conflicts.
# For chat functionality, we fall back to console output until this is resolved.

# Header files
set(HEADERS
    src/gostrike.h
    src/go_bridge.h
    include/gostrike_abi.h
)

# Create shared library
add_library(${PLUGIN_NAME} SHARED ${SOURCES} ${HEADERS})

# Include directories
target_include_directories(${PLUGIN_NAME} PRIVATE
    ${CMAKE_CURRENT_SOURCE_DIR}/include
    ${CMAKE_CURRENT_SOURCE_DIR}/src
)

# Option to force stub SDK usage
option(USE_STUB_SDK "Use stub SDK headers instead of real SDKs" OFF)

# Check for protobuf headers (required for HL2SDK CS2)
# The CS2 SDK's eiface.h requires network_connection.pb.h which must be generated
set(HAS_FULL_SDK FALSE)
set(HAS_PROTOBUF_HEADERS FALSE)

if(EXISTS "${METAMOD_PATH}/core/ISmmPlugin.h" AND 
   EXISTS "${HL2SDK_PATH}/public/eiface.h")
    # Check if SDK proto files exist
    if(EXISTS "${HL2SDK_PATH}/common/network_connection.proto")
        # Check for generated protobuf headers - REQUIRED for CS2 SDK
        # The eiface.h header includes network_connection.pb.h
        if(EXISTS "${CMAKE_CURRENT_SOURCE_DIR}/generated/network_connection.pb.h")
            set(HAS_FULL_SDK TRUE)
            message(STATUS "  Found generated network protobuf headers")
            
            # Check for usermessages.pb.h (needed for chat functionality)
            if(EXISTS "${CMAKE_CURRENT_SOURCE_DIR}/generated/usermessages.pb.h")
                set(HAS_PROTOBUF_HEADERS TRUE)
                message(STATUS "  Found generated usermessages protobuf headers")
            else()
                message(STATUS "  usermessages.pb.h not generated - chat will fall back to console")
            endif()
        else()
            message(STATUS "  Generated protobuf headers not found")
            message(STATUS "  Run: ./scripts/generate_protos.sh")
            message(STATUS "  Falling back to stub SDK")
        endif()
    endif()
endif()

if(HAS_FULL_SDK AND NOT USE_STUB_SDK)
    message(STATUS "Found complete SDK setup")
    message(STATUS "  Metamod:Source at ${METAMOD_PATH}")
    message(STATUS "  HL2SDK at ${HL2SDK_PATH}")
    
    # NOTE: We do NOT link system protobuf - it conflicts with SDK's bundled version.
    # The SDK uses protobuf 3.21.8, while system may have a different version.
    # Chat functionality will fall back to console output until protobuf is resolved.
    
    # IMPORTANT: Include SDK's protobuf BEFORE generated headers to avoid version conflicts
    target_include_directories(${PLUGIN_NAME} PRIVATE
        ${HL2SDK_PATH}/thirdparty/protobuf-3.21.8/src  # SDK's bundled protobuf (must be first)
        ${CMAKE_CURRENT_SOURCE_DIR}/generated          # Generated protobuf headers
        ${METAMOD_PATH}/core
        ${METAMOD_PATH}/core/sourcehook
        ${HL2SDK_PATH}/public
        ${HL2SDK_PATH}/public/tier0
        ${HL2SDK_PATH}/public/tier1
        ${HL2SDK_PATH}/public/entity2
        ${HL2SDK_PATH}/public/mathlib
        ${HL2SDK_PATH}/public/vstdlib
        ${HL2SDK_PATH}/public/networksystem  # For inetworkmessages.h
        ${HL2SDK_PATH}/public/engine         # For igameeventsystem.h
        ${HL2SDK_PATH}/public/game/server    # For iplayerinfo.h
        ${HL2SDK_PATH}/game/shared
        ${HL2SDK_PATH}/game/server
        ${HL2SDK_PATH}/common
    )
    
    # NOTE: Protobuf headers are generated but not currently used.
    # CS2's protobuf UserMessage system requires complex integration.
    # See go_bridge.cpp CB_SendChat for details.
    
    target_compile_definitions(${PLUGIN_NAME} PRIVATE 
        HAVE_METAMOD=1
        HAVE_HL2SDK=1
        META_IS_SOURCE2=1
    )
else()
    message(STATUS "Using stub SDK headers (development build)")
    message(STATUS "  For production, you need the full SDK with protobuf headers")
    
    target_include_directories(${PLUGIN_NAME} PRIVATE
        ${CMAKE_CURRENT_SOURCE_DIR}/include/stub
    )
    target_compile_definitions(${PLUGIN_NAME} PRIVATE 
        USE_STUB_SDK=1
    )
endif()

# Compiler flags
if(CMAKE_CXX_COMPILER_ID MATCHES "GNU|Clang")
    target_compile_options(${PLUGIN_NAME} PRIVATE
        -Wall
        -Wno-unused-function
        -Wno-unused-variable
        -fvisibility=hidden
        -fno-strict-aliasing
    )
    
    # Linux-specific
    if(CMAKE_SYSTEM_NAME STREQUAL "Linux")
        target_compile_definitions(${PLUGIN_NAME} PRIVATE
            POSIX
            LINUX
            GNUC
            _LINUX
            COMPILER_GCC
        )
        # Allow undefined symbols - they'll be resolved by the game engine at load time
        # The SDK uses templates that reference engine functions like CBufferString::Format
        target_link_options(${PLUGIN_NAME} PRIVATE
            -Wl,--unresolved-symbols=ignore-in-object-files
        )
    endif()
endif()

# Preprocessor definitions
target_compile_definitions(${PLUGIN_NAME} PRIVATE
    GOSTRIKE_VERSION="${PROJECT_VERSION}"
    SOURCE_ENGINE=28  # CS2 engine version
)

# Link libraries
target_link_libraries(${PLUGIN_NAME} PRIVATE
    dl      # For dlopen/dlsym
    pthread # For threading
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
message(STATUS "  Metamod:     ${METAMOD_PATH}")
message(STATUS "  HL2SDK:      ${HL2SDK_PATH}")
message(STATUS "")
