// gostrike.cpp - GoStrike Metamod:Source Plugin Implementation
// Architecture inspired by CounterStrikeSharp (https://github.com/roflmuffin/CounterStrikeSharp)
#include "gostrike.h"
#include "go_bridge.h"
#include "memory_module.h"
#include "gameconfig.h"
#include "schema.h"
#include "entity_system.h"
#include "convar_manager.h"
#include "game_functions.h"
#include "chat_manager.h"
#include <stdio.h>
#include <string.h>
#include <unistd.h>

// Plugin instance and Metamod exposure
GoStrikePlugin g_Plugin;
PLUGIN_EXPOSE(GoStrikePlugin, g_Plugin);

// Global engine interfaces (gs_ prefix to avoid SDK naming conflicts)
#ifndef USE_STUB_SDK
IVEngineServer2*       gs_pEngineServer2 = nullptr;
ISource2Server*        gs_pSource2Server = nullptr;
ICvar*                 gs_pCVar = nullptr;
IGameEventSystem*      gs_pGameEventSystem = nullptr;
CSchemaSystem*         gs_pSchemaSystem = nullptr;
INetworkMessages*      gs_pNetworkMessages = nullptr;
IServerGameClients*    gs_pServerGameClients = nullptr;
CGlobalVars*           gs_pGlobals = nullptr;
IGameResourceService*  gs_pGameResourceService = nullptr;
#else
void* gs_pEngineServer2 = nullptr;
void* gs_pSource2Server = nullptr;
void* gs_pCVar = nullptr;
void* gs_pGameEventSystem = nullptr;
void* gs_pSchemaSystem = nullptr;
void* gs_pNetworkMessages = nullptr;
void* gs_pServerGameClients = nullptr;
void* gs_pGlobals = nullptr;
void* gs_pGameResourceService = nullptr;
#endif

// Track if server is fully initialized (past AllPluginsLoaded)
static bool g_bServerFullyInitialized = false;

// Count how many times we've been loaded (for debugging)
static int g_loadCount = 0;

// Helper function for console output
static void ConPrint(const char* msg) {
    printf("%s", msg);
}

static void ConPrintf(const char* fmt, ...) {
    char buffer[1024];
    va_list args;
    va_start(args, fmt);
    vsnprintf(buffer, sizeof(buffer), fmt, args);
    va_end(args);
    ConPrint(buffer);
}

// ============================================================
// SourceHook Hook Declarations
// ============================================================

#ifndef USE_STUB_SDK
// Hook into IServerGameDLL::GameFrame (ISource2Server inherits from IServerGameDLL)
SH_DECL_HOOK3_void(IServerGameDLL, GameFrame, SH_NOATTRIB, 0, bool, bool, bool);

// Hook into IServerGameClients::ClientConnect
SH_DECL_HOOK6(IServerGameClients, ClientConnect, SH_NOATTRIB, 0, bool, CPlayerSlot, const char*, uint64, const char*, bool, CBufferString*);

// Hook into IServerGameClients::ClientDisconnect
SH_DECL_HOOK5_void(IServerGameClients, ClientDisconnect, SH_NOATTRIB, 0, CPlayerSlot, ENetworkDisconnectionReason, const char*, uint64, const char*);

// Hook into IServerGameClients::ClientPutInServer
SH_DECL_HOOK4_void(IServerGameClients, ClientPutInServer, SH_NOATTRIB, 0, CPlayerSlot, char const*, int, uint64);
#endif

// ============================================================
// ISmmPlugin Implementation
// ============================================================

bool GoStrikePlugin::Load(PluginId id, ISmmAPI* ismm, char* error, size_t maxlen, bool late) {
    PLUGIN_SAVEVARS();

    m_bLateLoad = late;
    g_loadCount++;

    ConPrintf("[GoStrike] Loading plugin (attempt=%d, late=%s, goInitialized=%s)...\n",
              g_loadCount,
              late ? "true" : "false",
              GoBridge_IsInitialized() ? "true" : "false");

    if (late || g_loadCount > 1) {
        g_bServerFullyInitialized = true;
        ConPrintf("[GoStrike] Server marked as fully initialized (late=%s, loadCount=%d)\n",
                  late ? "true" : "false", g_loadCount);
    }

    // ============================================================
    // Acquire Engine Interfaces
    // ============================================================

#ifndef USE_STUB_SDK
    GET_V_IFACE_CURRENT(GetEngineFactory, gs_pEngineServer2,
                        IVEngineServer2, SOURCE2ENGINETOSERVER_INTERFACE_VERSION);

    GET_V_IFACE_CURRENT(GetEngineFactory, gs_pCVar,
                        ICvar, CVAR_INTERFACE_VERSION);

    GET_V_IFACE_CURRENT(GetEngineFactory, gs_pSchemaSystem,
                        CSchemaSystem, SCHEMASYSTEM_INTERFACE_VERSION);

    GET_V_IFACE_CURRENT(GetEngineFactory, gs_pGameEventSystem,
                        IGameEventSystem, GAMEEVENTSYSTEM_INTERFACE_VERSION);

    GET_V_IFACE_CURRENT(GetEngineFactory, gs_pNetworkMessages,
                        INetworkMessages, NETWORKMESSAGES_INTERFACE_VERSION);

    GET_V_IFACE_ANY(GetServerFactory, gs_pSource2Server,
                    ISource2Server, SOURCE2SERVER_INTERFACE_VERSION);

    GET_V_IFACE_ANY(GetServerFactory, gs_pServerGameClients,
                    IServerGameClients, INTERFACEVERSION_SERVERGAMECLIENTS);

    GET_V_IFACE_ANY(GetEngineFactory, gs_pGameResourceService,
                    IGameResourceService, GAMERESOURCESERVICESERVER_INTERFACE_VERSION);

    ConPrintf("[GoStrike] All engine interfaces acquired successfully\n");
    ConPrintf("[GoStrike]   SchemaSystem: %p\n", gs_pSchemaSystem);
    ConPrintf("[GoStrike]   GameEventSystem: %p\n", gs_pGameEventSystem);
    ConPrintf("[GoStrike]   NetworkMessages: %p\n", gs_pNetworkMessages);
    ConPrintf("[GoStrike]   CVar: %p\n", gs_pCVar);

    // ============================================================
    // Register SourceHook Hooks
    // ============================================================

    // GameFrame - called every server tick (ISource2Server inherits IServerGameDLL)
    SH_ADD_HOOK_MEMFUNC(IServerGameDLL, GameFrame, gs_pSource2Server, &g_Plugin, &GoStrikePlugin::Hook_GameFrame, true);

    // Client connect/disconnect
    SH_ADD_HOOK_MEMFUNC(IServerGameClients, ClientConnect, gs_pServerGameClients, &g_Plugin, &GoStrikePlugin::Hook_ClientConnect, false);
    SH_ADD_HOOK_MEMFUNC(IServerGameClients, ClientDisconnect, gs_pServerGameClients, &g_Plugin, &GoStrikePlugin::Hook_ClientDisconnect, true);
    SH_ADD_HOOK_MEMFUNC(IServerGameClients, ClientPutInServer, gs_pServerGameClients, &g_Plugin, &GoStrikePlugin::Hook_ClientPutInServer, true);

    ConPrintf("[GoStrike] SourceHook hooks registered\n");

    // ============================================================
    // Initialize Phase 1 Systems (Memory, GameData, Schema, Entities)
    // ============================================================

    // Discover loaded game modules
    gostrike::modules::InitializeAll();

    // Load gamedata configuration
    {
        char gamedataPath[512];
        // Try to find gamedata.json relative to the plugin
        // CS2 working directory is typically game/csgo/
        const char* paths[] = {
            "addons/gostrike/configs/gamedata/gamedata.json",
            "./addons/gostrike/configs/gamedata/gamedata.json",
            nullptr
        };
        bool loaded = false;
        for (int i = 0; paths[i]; i++) {
            if (access(paths[i], F_OK) == 0) {
                loaded = gostrike::g_gameConfig.Init(paths[i]);
                break;
            }
        }
        if (!loaded) {
            ConPrintf("[GoStrike] WARNING: gamedata.json not found, some features may not work\n");
        }
    }

    // Initialize schema system
    gostrike::schema::Initialize();

    // Initialize ConVar manager
    gostrike::ConVarManager_Initialize();

    // Note: Entity system and game functions are initialized in AllPluginsLoaded()
    // because CGameEntitySystem may not be ready during Load()

#else
    ConPrintf("[GoStrike] Stub SDK mode - engine interfaces not available\n");
#endif

    // ============================================================
    // Initialize Go Runtime
    // ============================================================

    if (!GoBridge_Init()) {
        if (error && maxlen > 0) {
            snprintf(error, maxlen, "Failed to initialize Go runtime");
        }
        ConPrintf("[GoStrike] ERROR: Failed to initialize Go runtime\n");
        return false;
    }

    // Register C++ callbacks with Go
    GoBridge_RegisterCallbacks();

    ConPrintf("[GoStrike] Go runtime initialized successfully\n");

    // Register as Metamod listener
    if (ismm) {
        ismm->AddListener(this, this);
    }

    ConPrintf("[GoStrike] Plugin loaded successfully (version %s)\n", GOSTRIKE_VERSION);
    return true;
}

bool GoStrikePlugin::Unload(char* error, size_t maxlen) {
    ConPrintf("[GoStrike] Unloading plugin...\n");

    if (!g_bServerFullyInitialized) {
        ConPrintf("[GoStrike] Early unload cycle detected - refusing to unload\n");
        if (error && maxlen > 0) {
            snprintf(error, maxlen, "Cannot unload during server initialization");
        }
        return false;
    }

#ifndef USE_STUB_SDK
    // Shutdown entity system
    gostrike::EntitySystem_Shutdown();

    // Remove SourceHook hooks
    SH_REMOVE_HOOK_MEMFUNC(IServerGameDLL, GameFrame, gs_pSource2Server, &g_Plugin, &GoStrikePlugin::Hook_GameFrame, true);
    SH_REMOVE_HOOK_MEMFUNC(IServerGameClients, ClientConnect, gs_pServerGameClients, &g_Plugin, &GoStrikePlugin::Hook_ClientConnect, false);
    SH_REMOVE_HOOK_MEMFUNC(IServerGameClients, ClientDisconnect, gs_pServerGameClients, &g_Plugin, &GoStrikePlugin::Hook_ClientDisconnect, true);
    SH_REMOVE_HOOK_MEMFUNC(IServerGameClients, ClientPutInServer, gs_pServerGameClients, &g_Plugin, &GoStrikePlugin::Hook_ClientPutInServer, true);
    ConPrintf("[GoStrike] SourceHook hooks removed\n");
#endif

    // Shutdown Go runtime
    GoBridge_Shutdown();

    ConPrintf("[GoStrike] Plugin unloaded\n");
    return true;
}

void GoStrikePlugin::AllPluginsLoaded() {
    ConPrintf("[GoStrike] All plugins loaded - server fully initialized\n");
    g_bServerFullyInitialized = true;

#ifndef USE_STUB_SDK
    // Initialize entity system now that everything is ready
    gostrike::EntitySystem_Initialize();

    // Initialize game function pointers from gamedata
    gostrike::GameFunctions_Initialize();

    // Initialize chat manager (UTIL_ClientPrint resolution)
    gostrike::ChatManager_Initialize();
#endif
}

bool GoStrikePlugin::Pause(char* error, size_t maxlen) {
    ConPrintf("[GoStrike] Plugin paused\n");
    return true;
}

bool GoStrikePlugin::Unpause(char* error, size_t maxlen) {
    ConPrintf("[GoStrike] Plugin unpaused\n");
    return true;
}

// ============================================================
// Plugin Metadata
// ============================================================

const char* GoStrikePlugin::GetAuthor() {
    return "corrreia";
}

const char* GoStrikePlugin::GetName() {
    return "GoStrike";
}

const char* GoStrikePlugin::GetDescription() {
    return "GoStrike - Go-based CS2 modding framework (inspired by CounterStrikeSharp)";
}

const char* GoStrikePlugin::GetURL() {
    return "https://github.com/corrreia/gostrike";
}

const char* GoStrikePlugin::GetLicense() {
    return "MIT";
}

const char* GoStrikePlugin::GetVersion() {
    return GOSTRIKE_VERSION;
}

const char* GoStrikePlugin::GetDate() {
    return __DATE__;
}

const char* GoStrikePlugin::GetLogTag() {
    return "GOSTRIKE";
}

// ============================================================
// Game Hooks
// ============================================================

void GoStrikePlugin::Hook_GameFrame(bool simulating, bool bFirstTick, bool bLastTick) {
    if (!g_bServerFullyInitialized) {
        g_bServerFullyInitialized = true;
        ConPrintf("[GoStrike] Server fully initialized (first game frame)\n");
    }

    // Calculate delta time from globals
    static float lastTime = 0.0f;
    float currentTime = 0.0f;

#ifndef USE_STUB_SDK
    if (gs_pGlobals) {
        currentTime = gs_pGlobals->curtime;
    }
#endif

    float deltaTime = currentTime - lastTime;
    if (deltaTime < 0.0f) deltaTime = 0.0f;  // Handle map change time resets
    lastTime = currentTime;

    // Dispatch tick to Go
    GoBridge_OnTick(deltaTime);

    RETURN_META(MRES_IGNORED);
}

bool GoStrikePlugin::Hook_ClientConnect(CPlayerSlot slot, const char* pszName,
                                        uint64 xuid, const char* pszNetworkID,
                                        bool unk1, CBufferString* pRejectReason) {
    ConPrintf("[GoStrike] Client connecting: %s (slot %d)\n", pszName, slot.Get());

    gs_player_t player = {};
    player.slot = slot.Get();
    player.steam_id = xuid;
    player.name = const_cast<char*>(pszName);
    player.ip = const_cast<char*>(pszNetworkID);
    player.is_bot = false;
    player.is_alive = false;
    player.team = GS_TEAM_UNASSIGNED;

    GoBridge_OnPlayerConnect(&player);

    RETURN_META_VALUE(MRES_IGNORED, true);
}

void GoStrikePlugin::Hook_ClientDisconnect(CPlayerSlot slot, ENetworkDisconnectionReason reason,
                                           const char* pszName, uint64 xuid,
                                           const char* pszNetworkID) {
    ConPrintf("[GoStrike] Client disconnected: %s (slot %d)\n", pszName, slot.Get());

    GoBridge_OnPlayerDisconnect(slot.Get(), "disconnect");

    RETURN_META(MRES_IGNORED);
}

void GoStrikePlugin::Hook_ClientPutInServer(CPlayerSlot slot, char const* pszName,
                                            int type, uint64 xuid) {
    ConPrintf("[GoStrike] Client put in server: %s (slot %d)\n", pszName, slot.Get());

    RETURN_META(MRES_IGNORED);
}

void GoStrikePlugin::OnFireGameEvent(IGameEvent* event) {
    if (!event) return;

    const char* name = event->GetName();
    if (!name) return;

    GoBridge_FireEvent(name, event, false);
}
