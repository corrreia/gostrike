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
set(HAS_FULL_SDK FALSE)
if(EXISTS "${METAMOD_PATH}/core/ISmmPlugin.h" AND 
   EXISTS "${HL2SDK_PATH}/public/eiface.h")
    # Check if protobuf headers exist (required for CS2)
    if(EXISTS "${HL2SDK_PATH}/common/network_connection.pb.h")
        set(HAS_FULL_SDK TRUE)
    endif()
endif()

if(HAS_FULL_SDK AND NOT USE_STUB_SDK)
    message(STATUS "Found complete SDK setup")
    message(STATUS "  Metamod:Source at ${METAMOD_PATH}")
    message(STATUS "  HL2SDK at ${HL2SDK_PATH}")
    
    target_include_directories(${PLUGIN_NAME} PRIVATE
        ${METAMOD_PATH}/core
        ${METAMOD_PATH}/core/sourcehook
        ${HL2SDK_PATH}/public
        ${HL2SDK_PATH}/public/tier0
        ${HL2SDK_PATH}/public/tier1
        ${HL2SDK_PATH}/public/entity2
        ${HL2SDK_PATH}/public/mathlib
        ${HL2SDK_PATH}/public/vstdlib
        ${HL2SDK_PATH}/game/shared
        ${HL2SDK_PATH}/game/server
        ${HL2SDK_PATH}/common
    )
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
        target_link_options(${PLUGIN_NAME} PRIVATE
            -Wl,--no-undefined
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
