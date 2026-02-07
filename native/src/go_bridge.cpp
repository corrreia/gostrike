// go_bridge.cpp - C++ to Go bridge implementation
// Loads the Go shared library and handles all communication with the Go runtime

#include "go_bridge.h"
#include "gostrike.h"
#include "schema.h"
#include "entity_system.h"
#include "gameconfig.h"
#include "convar_manager.h"
#include "player_manager.h"
#include "game_functions.h"
#include "chat_manager.h"
#include <dlfcn.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <string>
#include <unistd.h>

// NOTE: SDK includes for UserMessage/Chat functionality are not used currently.
// CS2's protobuf-based UserMessage system requires complex integration due to
// generated protobuf classes being marked 'final'. See CB_SendChat for details.

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

// V2: Entity lifecycle
static void (*pfn_GoStrike_OnEntityCreated)(uint32_t, const char*) = nullptr;
static void (*pfn_GoStrike_OnEntitySpawned)(uint32_t, const char*) = nullptr;
static void (*pfn_GoStrike_OnEntityDeleted)(uint32_t) = nullptr;

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
#ifndef USE_STUB_SDK
    if (gs_pEngineServer2) {
        std::string fullCmd(cmd);
        if (fullCmd.empty() || fullCmd.back() != '\n') fullCmd += '\n';
        gs_pEngineServer2->ServerCommand(fullCmd.c_str());
        return;
    }
#endif
    printf("[GoStrike] ExecCommand (no engine): %s\n", cmd);
}

// Reply to a command invoker
static void CB_ReplyToCommand(int32_t slot, const char* msg) {
    if (!msg) return;

    if (slot < 0) {
        // Server console
        printf("%s\n", msg);
    } else {
#ifndef USE_STUB_SDK
        // Use UTIL_ClientPrint (chat) via Phase 3 chat manager
        gostrike::ClientPrint(slot, GS_HUD_PRINTTALK, msg);
        return;
#endif
        printf("[To Player %d] %s\n", slot, msg);
    }
}

// Get player information by slot (thread-safe: returns cached data only)
static gs_player_t* CB_GetPlayer(int32_t slot) {
    if (slot < 0 || slot >= 64) {
        return nullptr;
    }

    gs_player_t* player = &g_playerCache[slot];

    // Check if slot is valid (populated on connect)
    if (player->slot < 0) {
        return nullptr;
    }

    return player;
}

#ifndef USE_STUB_SDK
// Refresh live player data from schema/entity system (game thread only)
static void RefreshPlayerCache() {
    // Don't access entity system until it's initialized
    if (!gostrike::EntitySystem_GetSystemPtr()) return;

    for (int i = 0; i < 64; i++) {
        if (g_playerCache[i].slot < 0) continue;

        void* controller = gostrike::PlayerManager_GetController(i);
        if (!controller) continue;

        // Read from controller
        auto aliveKey = gostrike::schema::GetOffset("CCSPlayerController", "m_bPawnIsAlive");
        if (aliveKey.offset > 0) {
            g_playerCache[i].is_alive = *reinterpret_cast<bool*>(reinterpret_cast<uintptr_t>(controller) + aliveKey.offset);
        }
        auto healthKey = gostrike::schema::GetOffset("CCSPlayerController", "m_iPawnHealth");
        if (healthKey.offset > 0) {
            g_playerCache[i].health = *reinterpret_cast<int32_t*>(reinterpret_cast<uintptr_t>(controller) + healthKey.offset);
        }
        auto teamKey = gostrike::schema::GetOffset("CBaseEntity", "m_iTeamNum");
        if (teamKey.offset > 0) {
            g_playerCache[i].team = *reinterpret_cast<int32_t*>(reinterpret_cast<uintptr_t>(controller) + teamKey.offset);
        }

        // Read from pawn (health, armor, position)
        void* pawn = gostrike::PlayerManager_GetPawn(i);
        if (pawn) {
            auto pawnHealthKey = gostrike::schema::GetOffset("CBaseEntity", "m_iHealth");
            if (pawnHealthKey.offset > 0) {
                g_playerCache[i].health = *reinterpret_cast<int32_t*>(reinterpret_cast<uintptr_t>(pawn) + pawnHealthKey.offset);
            }
            auto armorKey = gostrike::schema::GetOffset("CCSPlayerPawn", "m_ArmorValue");
            if (armorKey.offset > 0) {
                g_playerCache[i].armor = *reinterpret_cast<int32_t*>(reinterpret_cast<uintptr_t>(pawn) + armorKey.offset);
            }
            // Position from CGameSceneNode (CBodyComponent -> m_pSceneNode -> m_vecAbsOrigin)
            auto bodyKey = gostrike::schema::GetOffset("CBaseEntity", "m_CBodyComponent");
            if (bodyKey.offset > 0) {
                void* bodyComp = *reinterpret_cast<void**>(reinterpret_cast<uintptr_t>(pawn) + bodyKey.offset);
                if (bodyComp) {
                    auto sceneNodeKey = gostrike::schema::GetOffset("CBodyComponent", "m_pSceneNode");
                    if (sceneNodeKey.offset > 0) {
                        void* sceneNode = *reinterpret_cast<void**>(reinterpret_cast<uintptr_t>(bodyComp) + sceneNodeKey.offset);
                        if (sceneNode) {
                            auto posKey = gostrike::schema::GetOffset("CGameSceneNode", "m_vecAbsOrigin");
                            if (posKey.offset > 0) {
                                float* pos = reinterpret_cast<float*>(reinterpret_cast<uintptr_t>(sceneNode) + posKey.offset);
                                g_playerCache[i].position.x = pos[0];
                                g_playerCache[i].position.y = pos[1];
                                g_playerCache[i].position.z = pos[2];
                            }
                        }
                    }
                }
            }
        }
    }
}
#endif

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
#ifndef USE_STUB_SDK
    if (gs_pEngineServer2) {
        gs_pEngineServer2->DisconnectClient(CPlayerSlot(slot), (ENetworkDisconnectionReason)39, reason);
        return;
    }
#endif
    printf("[GoStrike] Kicking player %d: %s\n", slot, reason ? reason : "No reason");
}

// Get current map name
static const char* CB_GetMapName() {
#ifndef USE_STUB_SDK
    if (gs_pGlobals) {
        return gs_pGlobals->mapname.ToCStr();
    }
#endif
    return g_currentMap;
}

// Get max players
static int32_t CB_GetMaxPlayers() {
#ifndef USE_STUB_SDK
    if (gs_pGlobals) {
        return gs_pGlobals->maxClients;
    }
#endif
    return 64;
}

// Get tick rate
static int32_t CB_GetTickRate() {
#ifndef USE_STUB_SDK
    if (gs_pGlobals && gs_pGlobals->m_flIntervalPerTick > 0.0f) {
        return (int32_t)(1.0f / gs_pGlobals->m_flIntervalPerTick);
    }
#endif
    return 64;
}

// Send chat message - uses UTIL_ClientPrint via chat manager (Phase 3)
static void CB_SendChat(int32_t slot, const char* msg) {
    if (!msg) return;
#ifndef USE_STUB_SDK
    if (slot < 0) {
        gostrike::ClientPrintAll(GS_HUD_PRINTTALK, msg);
    } else {
        gostrike::ClientPrint(slot, GS_HUD_PRINTTALK, msg);
    }
    return;
#endif
    printf("[GoStrike Chat] %s\n", msg);
}

// Send center message - uses UTIL_ClientPrint via chat manager (Phase 3)
static void CB_SendCenter(int32_t slot, const char* msg) {
    if (!msg) return;
#ifndef USE_STUB_SDK
    if (slot < 0) {
        gostrike::ClientPrintAll(GS_HUD_PRINTCENTER, msg);
    } else {
        gostrike::ClientPrint(slot, GS_HUD_PRINTCENTER, msg);
    }
    return;
#endif
    printf("[GoStrike Center] %s\n", msg);
}

// ============================================================
// V2 Callback Implementations (Phase 1: Foundation)
// ============================================================

// Schema: get field offset
static int32_t CB_SchemaGetOffset(const char* className, const char* fieldName, bool* isNetworked) {
#ifndef USE_STUB_SDK
    auto key = gostrike::schema::GetOffset(className, fieldName);
    if (isNetworked) *isNetworked = key.networked;
    return key.offset;
#else
    (void)className; (void)fieldName;
    if (isNetworked) *isNetworked = false;
    return 0;
#endif
}

// Schema: set state changed
static void CB_SchemaSetStateChanged(void* entity, const char* className, const char* fieldName, int32_t offset) {
#ifndef USE_STUB_SDK
    gostrike::schema::SetStateChanged(entity, className, fieldName, offset);
#else
    (void)entity; (void)className; (void)fieldName; (void)offset;
#endif
}

// Entity property read/write via schema offsets
static int32_t CB_EntityGetInt(void* entity, const char* className, const char* fieldName) {
#ifndef USE_STUB_SDK
    if (!entity || !className || !fieldName) return 0;
    auto key = gostrike::schema::GetOffset(className, fieldName);
    if (key.offset == 0) return 0;
    return *reinterpret_cast<int32_t*>(reinterpret_cast<uintptr_t>(entity) + key.offset);
#else
    (void)entity; (void)className; (void)fieldName;
    return 0;
#endif
}

static void CB_EntitySetInt(void* entity, const char* className, const char* fieldName, int32_t value) {
#ifndef USE_STUB_SDK
    if (!entity || !className || !fieldName) return;
    auto key = gostrike::schema::GetOffset(className, fieldName);
    if (key.offset == 0) return;
    *reinterpret_cast<int32_t*>(reinterpret_cast<uintptr_t>(entity) + key.offset) = value;
    if (key.networked) {
        gostrike::schema::SetStateChanged(entity, className, fieldName, key.offset);
    }
#else
    (void)entity; (void)className; (void)fieldName; (void)value;
#endif
}

static float CB_EntityGetFloat(void* entity, const char* className, const char* fieldName) {
#ifndef USE_STUB_SDK
    if (!entity || !className || !fieldName) return 0.0f;
    auto key = gostrike::schema::GetOffset(className, fieldName);
    if (key.offset == 0) return 0.0f;
    return *reinterpret_cast<float*>(reinterpret_cast<uintptr_t>(entity) + key.offset);
#else
    (void)entity; (void)className; (void)fieldName;
    return 0.0f;
#endif
}

static void CB_EntitySetFloat(void* entity, const char* className, const char* fieldName, float value) {
#ifndef USE_STUB_SDK
    if (!entity || !className || !fieldName) return;
    auto key = gostrike::schema::GetOffset(className, fieldName);
    if (key.offset == 0) return;
    *reinterpret_cast<float*>(reinterpret_cast<uintptr_t>(entity) + key.offset) = value;
    if (key.networked) {
        gostrike::schema::SetStateChanged(entity, className, fieldName, key.offset);
    }
#else
    (void)entity; (void)className; (void)fieldName; (void)value;
#endif
}

static bool CB_EntityGetBool(void* entity, const char* className, const char* fieldName) {
#ifndef USE_STUB_SDK
    if (!entity || !className || !fieldName) return false;
    auto key = gostrike::schema::GetOffset(className, fieldName);
    if (key.offset == 0) return false;
    return *reinterpret_cast<bool*>(reinterpret_cast<uintptr_t>(entity) + key.offset);
#else
    (void)entity; (void)className; (void)fieldName;
    return false;
#endif
}

static void CB_EntitySetBool(void* entity, const char* className, const char* fieldName, bool value) {
#ifndef USE_STUB_SDK
    if (!entity || !className || !fieldName) return;
    auto key = gostrike::schema::GetOffset(className, fieldName);
    if (key.offset == 0) return;
    *reinterpret_cast<bool*>(reinterpret_cast<uintptr_t>(entity) + key.offset) = value;
    if (key.networked) {
        gostrike::schema::SetStateChanged(entity, className, fieldName, key.offset);
    }
#else
    (void)entity; (void)className; (void)fieldName; (void)value;
#endif
}

static int32_t CB_EntityGetString(void* entity, const char* className, const char* fieldName, char* buf, int32_t bufSize) {
#ifndef USE_STUB_SDK
    if (!entity || !className || !fieldName || !buf || bufSize <= 0) return 0;
    auto key = gostrike::schema::GetOffset(className, fieldName);
    if (key.offset == 0) return 0;
    const char* str = reinterpret_cast<const char*>(reinterpret_cast<uintptr_t>(entity) + key.offset);
    int32_t len = static_cast<int32_t>(strlen(str));
    int32_t copy = (len < bufSize - 1) ? len : bufSize - 1;
    memcpy(buf, str, copy);
    buf[copy] = '\0';
    return copy;
#else
    (void)entity; (void)className; (void)fieldName; (void)buf; (void)bufSize;
    return 0;
#endif
}

static void CB_EntityGetVector(void* entity, const char* className, const char* fieldName, gs_vector3_t* out) {
#ifndef USE_STUB_SDK
    if (!entity || !className || !fieldName || !out) return;
    auto key = gostrike::schema::GetOffset(className, fieldName);
    if (key.offset == 0) { *out = {0, 0, 0}; return; }
    float* vec = reinterpret_cast<float*>(reinterpret_cast<uintptr_t>(entity) + key.offset);
    out->x = vec[0];
    out->y = vec[1];
    out->z = vec[2];
#else
    (void)entity; (void)className; (void)fieldName;
    if (out) *out = {0, 0, 0};
#endif
}

static void CB_EntitySetVector(void* entity, const char* className, const char* fieldName, gs_vector3_t* value) {
#ifndef USE_STUB_SDK
    if (!entity || !className || !fieldName || !value) return;
    auto key = gostrike::schema::GetOffset(className, fieldName);
    if (key.offset == 0) return;
    float* vec = reinterpret_cast<float*>(reinterpret_cast<uintptr_t>(entity) + key.offset);
    vec[0] = value->x;
    vec[1] = value->y;
    vec[2] = value->z;
    if (key.networked) {
        gostrike::schema::SetStateChanged(entity, className, fieldName, key.offset);
    }
#else
    (void)entity; (void)className; (void)fieldName; (void)value;
#endif
}

// Entity lookup callbacks
static void* CB_GetEntityByIndex(uint32_t index) {
    return gostrike::EntitySystem_GetEntityByIndex(index);
}

static uint32_t CB_GetEntityIndex(void* entity) {
    return gostrike::EntitySystem_GetEntityIndex(entity);
}

static const char* CB_GetEntityClassname(void* entity) {
    return gostrike::EntitySystem_GetEntityClassname(entity);
}

static bool CB_IsEntityValid(void* entity) {
    return gostrike::EntitySystem_IsEntityValid(entity);
}

// GameData callbacks
static void* CB_ResolveGamedata(const char* name) {
    if (!name) return nullptr;
    return gostrike::g_gameConfig.ResolveSignature(name);
}

static int32_t CB_GetGamedataOffset(const char* name) {
    if (!name) return -1;
    return gostrike::g_gameConfig.GetOffset(name);
}

// ============================================================
// V3 Callbacks: ConVar Operations
// ============================================================

static int32_t CB_ConVarGetInt(const char* name) {
    return gostrike::ConVar_GetInt(name);
}

static void CB_ConVarSetInt(const char* name, int32_t value) {
    gostrike::ConVar_SetInt(name, value);
}

static float CB_ConVarGetFloat(const char* name) {
    return gostrike::ConVar_GetFloat(name);
}

static void CB_ConVarSetFloat(const char* name, float value) {
    gostrike::ConVar_SetFloat(name, value);
}

static int32_t CB_ConVarGetString(const char* name, char* buf, int32_t buf_size) {
    return gostrike::ConVar_GetString(name, buf, buf_size);
}

static void CB_ConVarSetString(const char* name, const char* value) {
    gostrike::ConVar_SetString(name, value);
}

// ============================================================
// V3 Callbacks: Player Pawn/Controller
// ============================================================

static void* CB_GetPlayerController(int32_t slot) {
    return gostrike::PlayerManager_GetController(slot);
}

static void* CB_GetPlayerPawn(int32_t slot) {
    return gostrike::PlayerManager_GetPawn(slot);
}

// ============================================================
// V3 Callbacks: Game Functions
// ============================================================

static void CB_PlayerRespawn(int32_t slot) {
    gostrike::GameFunc_Respawn(slot);
}

static void CB_PlayerChangeTeam(int32_t slot, int32_t team) {
    gostrike::GameFunc_ChangeTeam(slot, team);
}

static void CB_PlayerSlay(int32_t slot) {
    gostrike::GameFunc_Slay(slot);
}

static void CB_PlayerTeleport(int32_t slot, gs_vector3_t* pos, gs_vector3_t* angles, gs_vector3_t* velocity) {
    gostrike::GameFunc_Teleport(slot, pos, angles, velocity);
}

static void CB_EntitySetModel(void* entity, const char* model) {
    gostrike::GameFunc_SetModel(entity, model);
}

// ============================================================
// V4 Callbacks: Communication
// ============================================================

static void CB_ClientPrint(int32_t slot, int32_t dest, const char* msg) {
    gostrike::ClientPrint(slot, dest, msg);
}

static void CB_ClientPrintAll(int32_t dest, const char* msg) {
    gostrike::ClientPrintAll(dest, msg);
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

    // V2 symbols (optional - may not exist in older Go builds)
    pfn_GoStrike_OnEntityCreated = (decltype(pfn_GoStrike_OnEntityCreated))dlsym(g_goLib, "GoStrike_OnEntityCreated");
    pfn_GoStrike_OnEntitySpawned = (decltype(pfn_GoStrike_OnEntitySpawned))dlsym(g_goLib, "GoStrike_OnEntitySpawned");
    pfn_GoStrike_OnEntityDeleted = (decltype(pfn_GoStrike_OnEntityDeleted))dlsym(g_goLib, "GoStrike_OnEntityDeleted");

    printf("[GoStrike] All Go symbols loaded\n");
    if (pfn_GoStrike_OnEntityCreated) {
        printf("[GoStrike] V2 entity lifecycle symbols available\n");
    }
    
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
    
    static gs_callbacks_t callbacks = {};

    // V1 callbacks
    callbacks.log = CB_Log;
    callbacks.exec_command = CB_ExecCommand;
    callbacks.reply_to_command = CB_ReplyToCommand;
    callbacks.get_player = CB_GetPlayer;
    callbacks.get_player_count = CB_GetPlayerCount;
    callbacks.get_all_players = CB_GetAllPlayers;
    callbacks.kick_player = CB_KickPlayer;
    callbacks.get_map_name = CB_GetMapName;
    callbacks.get_max_players = CB_GetMaxPlayers;
    callbacks.get_tick_rate = CB_GetTickRate;
    callbacks.send_chat = CB_SendChat;
    callbacks.send_center = CB_SendCenter;

    // V2 callbacks (Phase 1: Foundation)
    callbacks.schema_get_offset = CB_SchemaGetOffset;
    callbacks.schema_set_state_changed = CB_SchemaSetStateChanged;
    callbacks.entity_get_int = CB_EntityGetInt;
    callbacks.entity_set_int = CB_EntitySetInt;
    callbacks.entity_get_float = CB_EntityGetFloat;
    callbacks.entity_set_float = CB_EntitySetFloat;
    callbacks.entity_get_bool = CB_EntityGetBool;
    callbacks.entity_set_bool = CB_EntitySetBool;
    callbacks.entity_get_string = CB_EntityGetString;
    callbacks.entity_get_vector = CB_EntityGetVector;
    callbacks.entity_set_vector = CB_EntitySetVector;
    callbacks.get_entity_by_index = CB_GetEntityByIndex;
    callbacks.get_entity_index = CB_GetEntityIndex;
    callbacks.get_entity_classname = CB_GetEntityClassname;
    callbacks.is_entity_valid = CB_IsEntityValid;
    callbacks.resolve_gamedata = CB_ResolveGamedata;
    callbacks.get_gamedata_offset = CB_GetGamedataOffset;

    // === V3 (Phase 2: Core Game Integration) ===
    callbacks.convar_get_int = CB_ConVarGetInt;
    callbacks.convar_set_int = CB_ConVarSetInt;
    callbacks.convar_get_float = CB_ConVarGetFloat;
    callbacks.convar_set_float = CB_ConVarSetFloat;
    callbacks.convar_get_string = CB_ConVarGetString;
    callbacks.convar_set_string = CB_ConVarSetString;
    callbacks.get_player_controller = CB_GetPlayerController;
    callbacks.get_player_pawn = CB_GetPlayerPawn;
    callbacks.player_respawn = CB_PlayerRespawn;
    callbacks.player_change_team = CB_PlayerChangeTeam;
    callbacks.player_slay = CB_PlayerSlay;
    callbacks.player_teleport = CB_PlayerTeleport;
    callbacks.entity_set_model = CB_EntitySetModel;

    // === V4 (Phase 3: Communication) ===
    callbacks.client_print = CB_ClientPrint;
    callbacks.client_print_all = CB_ClientPrintAll;

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

// ============================================================
// Entity Lifecycle Forwarding to Go
// ============================================================

void GoBridge_OnEntityCreated(uint32_t index, const char* classname) {
    if (!g_initialized || !pfn_GoStrike_OnEntityCreated) return;
    pfn_GoStrike_OnEntityCreated(index, classname);
}

void GoBridge_OnEntitySpawned(uint32_t index, const char* classname) {
    if (!g_initialized || !pfn_GoStrike_OnEntitySpawned) return;
    pfn_GoStrike_OnEntitySpawned(index, classname);
}

void GoBridge_OnEntityDeleted(uint32_t index) {
    if (!g_initialized || !pfn_GoStrike_OnEntityDeleted) return;
    pfn_GoStrike_OnEntityDeleted(index);
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

void GoBridge_RefreshPlayerCache() {
#ifndef USE_STUB_SDK
    RefreshPlayerCache();
#endif
}
