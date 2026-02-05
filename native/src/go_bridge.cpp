// go_bridge.cpp - C++ to Go bridge implementation
// Loads the Go shared library and handles all communication with the Go runtime

#include "go_bridge.h"
#include "gostrike.h"
#include <dlfcn.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <unistd.h>

// ============================================================
// Go Library Handle and Function Pointers
// ============================================================

static void* g_goLib = nullptr;
static bool g_initialized = false;

// Function pointers to Go exports
static gs_error_t (*pfn_GoStrike_Init)(void) = nullptr;
static void (*pfn_GoStrike_Shutdown)(void) = nullptr;
static void (*pfn_GoStrike_OnTick)(float) = nullptr;
static gs_event_result_t (*pfn_GoStrike_OnEvent)(gs_event_t*, bool) = nullptr;
static void (*pfn_GoStrike_OnPlayerConnect)(gs_player_t*) = nullptr;
static void (*pfn_GoStrike_OnPlayerDisconnect)(int32_t, const char*) = nullptr;
static void (*pfn_GoStrike_OnMapChange)(const char*) = nullptr;
static bool (*pfn_GoStrike_OnChatMessage)(int32_t, const char*) = nullptr;
static char* (*pfn_GoStrike_GetLastError)(void) = nullptr;
static void (*pfn_GoStrike_ClearLastError)(void) = nullptr;
static int32_t (*pfn_GoStrike_GetABIVersion)(void) = nullptr;
static void (*pfn_GoStrike_RegisterCallbacks)(gs_callbacks_t*) = nullptr;

// ============================================================
// Callback Implementations (C++ -> Go calls these from Go)
// ============================================================

// Storage for player data returned to Go
static gs_player_t g_playerCache[64];
static char g_playerNames[64][GS_MAX_NAME_LEN];
static char g_playerIPs[64][64];
static char g_currentMap[GS_MAX_PATH_LEN] = "unknown";

// Log callback
static void CB_Log(int level, const char* tag, const char* msg) {
    const char* levelStr = "INFO";
    switch (level) {
        case GS_LOG_DEBUG:   levelStr = "DEBUG"; break;
        case GS_LOG_INFO:    levelStr = "INFO"; break;
        case GS_LOG_WARNING: levelStr = "WARN"; break;
        case GS_LOG_ERROR:   levelStr = "ERROR"; break;
    }
    printf("[%s][%s] %s\n", tag, levelStr, msg);
}

// Execute a server command
static void CB_ExecCommand(const char* cmd) {
    if (!cmd) return;
    
    // In actual implementation, use engine interface:
    // if (g_pEngineServer) {
    //     g_pEngineServer->ServerCommand(cmd);
    // }
    printf("[GoStrike] ExecCommand: %s\n", cmd);
}

// Reply to a command invoker
static void CB_ReplyToCommand(int32_t slot, const char* msg) {
    if (!msg) return;
    
    if (slot < 0) {
        // Server console
        printf("%s\n", msg);
    } else {
        // Would send to player in actual implementation
        printf("[To Player %d] %s\n", slot, msg);
    }
}

// Get player information by slot
static gs_player_t* CB_GetPlayer(int32_t slot) {
    if (slot < 0 || slot >= 64) {
        return nullptr;
    }
    
    // In actual implementation, fetch real player data:
    // CPlayerSlot playerSlot(slot);
    // CBaseEntity* player = ...;
    
    // For now, return cached/mock data
    gs_player_t* player = &g_playerCache[slot];
    
    // Check if slot is valid (would check actual player list)
    if (player->slot < 0) {
        return nullptr;
    }
    
    return player;
}

// Get number of connected players
static int32_t CB_GetPlayerCount() {
    // In actual implementation, iterate player list
    int32_t count = 0;
    for (int i = 0; i < 64; i++) {
        if (g_playerCache[i].slot >= 0) {
            count++;
        }
    }
    return count;
}

// Get all player slots
static int32_t CB_GetAllPlayers(int32_t* outSlots) {
    if (!outSlots) return 0;
    
    int32_t count = 0;
    for (int i = 0; i < 64; i++) {
        if (g_playerCache[i].slot >= 0) {
            outSlots[count++] = i;
        }
    }
    return count;
}

// Kick a player
static void CB_KickPlayer(int32_t slot, const char* reason) {
    if (slot < 0 || slot >= 64) return;
    
    // In actual implementation:
    // g_pEngineServer->DisconnectClient(CPlayerSlot(slot), reason);
    printf("[GoStrike] Kicking player %d: %s\n", slot, reason ? reason : "No reason");
}

// Get current map name
static const char* CB_GetMapName() {
    // In actual implementation:
    // return g_pGlobals->mapname.ToCStr();
    return g_currentMap;
}

// Get max players
static int32_t CB_GetMaxPlayers() {
    // In actual implementation:
    // return g_pGlobals->maxClients;
    return 64;
}

// Get tick rate
static int32_t CB_GetTickRate() {
    // CS2 typically runs at 64 or 128 tick
    return 64;
}

// Send chat message
static void CB_SendChat(int32_t slot, const char* msg) {
    if (!msg) return;
    
    // In actual implementation, use UserMessage or similar
    if (slot < 0) {
        printf("[Chat All] %s\n", msg);
    } else {
        printf("[Chat %d] %s\n", slot, msg);
    }
}

// Send center message
static void CB_SendCenter(int32_t slot, const char* msg) {
    if (!msg) return;
    
    // In actual implementation, use HudMessage or similar
    printf("[Center %d] %s\n", slot, msg);
}

// ============================================================
// Bridge Implementation
// ============================================================

// Helper macro to load a symbol from the Go library
#define LOAD_GO_SYMBOL(name) \
    do { \
        pfn_##name = (decltype(pfn_##name))dlsym(g_goLib, #name); \
        if (!pfn_##name) { \
            printf("[GoStrike] Failed to load symbol: %s (%s)\n", #name, dlerror()); \
            return false; \
        } \
    } while(0)

// Try multiple paths to find the Go library
static const char* FindGoLibrary() {
    // Possible paths for the Go library
    // The CS2 server working directory is typically /home/steam/cs2-dedicated/game/csgo/
    static const char* paths[] = {
        // Relative paths (most common)
        "addons/gostrike/bin/libgostrike_go.so",
        "./addons/gostrike/bin/libgostrike_go.so",
        "../addons/gostrike/bin/libgostrike_go.so",
        // Absolute path for CS2 dedicated server (Docker)
        "/home/steam/cs2-dedicated/game/csgo/addons/gostrike/bin/libgostrike_go.so",
        // Alternative absolute paths
        "/opt/cs2-server/game/csgo/addons/gostrike/bin/libgostrike_go.so",
        "./csgo/addons/gostrike/bin/libgostrike_go.so",
        "./game/csgo/addons/gostrike/bin/libgostrike_go.so",
        "./libgostrike_go.so",  // Current directory fallback
        nullptr
    };
    
    for (int i = 0; paths[i] != nullptr; i++) {
        if (access(paths[i], F_OK) == 0) {
            return paths[i];
        }
    }
    
    // If not found, return the default path (will fail with a clear error)
    return "addons/gostrike/bin/libgostrike_go.so";
}

bool GoBridge_Init() {
    if (g_initialized) {
        printf("[GoStrike] Go bridge already initialized\n");
        return true;
    }
    
    // Initialize player cache to invalid
    for (int i = 0; i < 64; i++) {
        g_playerCache[i].slot = -1;
    }
    
    // Find and load the Go shared library
    const char* libPath = FindGoLibrary();
    printf("[GoStrike] Loading Go library from: %s\n", libPath);
    
    g_goLib = dlopen(libPath, RTLD_NOW | RTLD_GLOBAL);
    if (!g_goLib) {
        printf("[GoStrike] Failed to load Go library: %s\n", dlerror());
        return false;
    }
    
    printf("[GoStrike] Go library loaded successfully\n");
    
    // Load all Go function symbols
    LOAD_GO_SYMBOL(GoStrike_Init);
    LOAD_GO_SYMBOL(GoStrike_Shutdown);
    LOAD_GO_SYMBOL(GoStrike_OnTick);
    LOAD_GO_SYMBOL(GoStrike_OnEvent);
    LOAD_GO_SYMBOL(GoStrike_OnPlayerConnect);
    LOAD_GO_SYMBOL(GoStrike_OnPlayerDisconnect);
    LOAD_GO_SYMBOL(GoStrike_OnMapChange);
    LOAD_GO_SYMBOL(GoStrike_OnChatMessage);
    LOAD_GO_SYMBOL(GoStrike_GetLastError);
    LOAD_GO_SYMBOL(GoStrike_ClearLastError);
    LOAD_GO_SYMBOL(GoStrike_GetABIVersion);
    LOAD_GO_SYMBOL(GoStrike_RegisterCallbacks);
    
    printf("[GoStrike] All Go symbols loaded\n");
    
    // Check ABI version compatibility
    int32_t goAbiVersion = pfn_GoStrike_GetABIVersion();
    if (goAbiVersion != GOSTRIKE_ABI_VERSION) {
        printf("[GoStrike] ABI version mismatch! C++: %d, Go: %d\n",
               GOSTRIKE_ABI_VERSION, goAbiVersion);
        dlclose(g_goLib);
        g_goLib = nullptr;
        return false;
    }
    
    printf("[GoStrike] ABI version check passed (version %d)\n", goAbiVersion);
    
    // Initialize Go runtime
    gs_error_t err = pfn_GoStrike_Init();
    if (err != GS_OK) {
        char* errMsg = pfn_GoStrike_GetLastError();
        printf("[GoStrike] Go initialization failed (code %d): %s\n",
               err, errMsg ? errMsg : "unknown error");
        if (errMsg) {
            free(errMsg);
        }
        dlclose(g_goLib);
        g_goLib = nullptr;
        return false;
    }
    
    g_initialized = true;
    printf("[GoStrike] Go runtime initialized\n");
    
    return true;
}

void GoBridge_RegisterCallbacks() {
    if (!g_initialized || !pfn_GoStrike_RegisterCallbacks) {
        return;
    }
    
    static gs_callbacks_t callbacks = {
        // Logging
        .log = CB_Log,
        
        // Commands
        .exec_command = CB_ExecCommand,
        .reply_to_command = CB_ReplyToCommand,
        
        // Players
        .get_player = CB_GetPlayer,
        .get_player_count = CB_GetPlayerCount,
        .get_all_players = CB_GetAllPlayers,
        .kick_player = CB_KickPlayer,
        
        // Server info
        .get_map_name = CB_GetMapName,
        .get_max_players = CB_GetMaxPlayers,
        .get_tick_rate = CB_GetTickRate,
        
        // Messaging
        .send_chat = CB_SendChat,
        .send_center = CB_SendCenter,
    };
    
    pfn_GoStrike_RegisterCallbacks(&callbacks);
    printf("[GoStrike] Callbacks registered with Go runtime\n");
}

void GoBridge_Shutdown() {
    if (!g_initialized) {
        return;
    }
    
    printf("[GoStrike] Shutting down Go bridge...\n");
    
    // Shutdown Go runtime
    if (pfn_GoStrike_Shutdown) {
        pfn_GoStrike_Shutdown();
    }
    
    // Unload Go library
    if (g_goLib) {
        dlclose(g_goLib);
        g_goLib = nullptr;
    }
    
    // Clear function pointers
    pfn_GoStrike_Init = nullptr;
    pfn_GoStrike_Shutdown = nullptr;
    pfn_GoStrike_OnTick = nullptr;
    pfn_GoStrike_OnEvent = nullptr;
    pfn_GoStrike_OnPlayerConnect = nullptr;
    pfn_GoStrike_OnPlayerDisconnect = nullptr;
    pfn_GoStrike_OnMapChange = nullptr;
    pfn_GoStrike_OnChatMessage = nullptr;
    pfn_GoStrike_GetLastError = nullptr;
    pfn_GoStrike_ClearLastError = nullptr;
    pfn_GoStrike_GetABIVersion = nullptr;
    pfn_GoStrike_RegisterCallbacks = nullptr;
    
    g_initialized = false;
    printf("[GoStrike] Go bridge shutdown complete\n");
}

bool GoBridge_IsInitialized() {
    return g_initialized;
}

void GoBridge_OnTick(float deltaTime) {
    if (!g_initialized || !pfn_GoStrike_OnTick) {
        return;
    }
    pfn_GoStrike_OnTick(deltaTime);
}

gs_event_result_t GoBridge_FireEvent(const char* name, void* event, bool isPost) {
    if (!g_initialized || !pfn_GoStrike_OnEvent || !name) {
        return GS_EVENT_CONTINUE;
    }
    
    gs_event_t gsEvent = {};
    gsEvent.name = name;
    gsEvent.name_len = (uint32_t)strlen(name);
    gsEvent.native_event = event;
    gsEvent.can_modify = !isPost;
    
    return pfn_GoStrike_OnEvent(&gsEvent, isPost);
}

void GoBridge_OnPlayerConnect(gs_player_t* player) {
    if (!g_initialized || !pfn_GoStrike_OnPlayerConnect || !player) {
        return;
    }
    
    // Cache player data
    int slot = player->slot;
    if (slot >= 0 && slot < 64) {
        memcpy(&g_playerCache[slot], player, sizeof(gs_player_t));
        
        // Copy strings to our buffers
        if (player->name) {
            strncpy(g_playerNames[slot], player->name, GS_MAX_NAME_LEN - 1);
            g_playerNames[slot][GS_MAX_NAME_LEN - 1] = '\0';
            g_playerCache[slot].name = g_playerNames[slot];
        }
        if (player->ip) {
            strncpy(g_playerIPs[slot], player->ip, 63);
            g_playerIPs[slot][63] = '\0';
            g_playerCache[slot].ip = g_playerIPs[slot];
        }
    }
    
    pfn_GoStrike_OnPlayerConnect(player);
}

void GoBridge_OnPlayerDisconnect(int32_t slot, const char* reason) {
    if (!g_initialized || !pfn_GoStrike_OnPlayerDisconnect) {
        return;
    }
    
    // Clear player cache
    if (slot >= 0 && slot < 64) {
        g_playerCache[slot].slot = -1;
    }
    
    pfn_GoStrike_OnPlayerDisconnect(slot, reason ? reason : "disconnect");
}

void GoBridge_OnMapChange(const char* mapName) {
    if (!mapName) {
        return;
    }
    
    // Update cached map name
    strncpy(g_currentMap, mapName, GS_MAX_PATH_LEN - 1);
    g_currentMap[GS_MAX_PATH_LEN - 1] = '\0';
    
    if (!g_initialized || !pfn_GoStrike_OnMapChange) {
        return;
    }
    
    pfn_GoStrike_OnMapChange(mapName);
}

char* GoBridge_GetLastError() {
    if (!g_initialized || !pfn_GoStrike_GetLastError) {
        return nullptr;
    }
    return pfn_GoStrike_GetLastError();
}

bool GoBridge_OnChatMessage(int32_t playerSlot, const char* message) {
    if (!g_initialized || !pfn_GoStrike_OnChatMessage || !message) {
        return false;
    }
    return pfn_GoStrike_OnChatMessage(playerSlot, message);
}
