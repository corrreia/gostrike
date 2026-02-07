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

#ifndef USE_STUB_SDK
#include "usermessages.pb.h"
#endif
#include <string.h>
#include <string>
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
IGameEventManager2*    gs_pGameEventManager = nullptr;
CSchemaSystem*         gs_pSchemaSystem = nullptr;
INetworkMessages*      gs_pNetworkMessages = nullptr;
IServerGameClients*    gs_pServerGameClients = nullptr;
CGlobalVars*           gs_pGlobals = nullptr;
IGameResourceService*  gs_pGameResourceService = nullptr;
INetworkServerService* gs_pNetworkServerService = nullptr;
#else
void* gs_pEngineServer2 = nullptr;
void* gs_pSource2Server = nullptr;
void* gs_pCVar = nullptr;
void* gs_pGameEventSystem = nullptr;
void* gs_pGameEventManager = nullptr;
void* gs_pSchemaSystem = nullptr;
void* gs_pNetworkMessages = nullptr;
void* gs_pServerGameClients = nullptr;
void* gs_pGlobals = nullptr;
void* gs_pGameResourceService = nullptr;
void* gs_pNetworkServerService = nullptr;
#endif

// Provide the GameEntitySystem() function that the SDK's entity2 code expects.
// This is defined in libserver.so at runtime, but since we link entitysystem.cpp
// from the SDK statically, we need our own definition (same approach as CSSharp).
#ifndef USE_STUB_SDK
static CGameEntitySystem* s_pGameEntitySystem = nullptr;
CGameEntitySystem* GameEntitySystem() { return s_pGameEntitySystem; }
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

// Hook into IGameEventManager2::FireEvent (game events like player_death, round_start)
// Approach from CSSharp: hook LoadEventsFromFile to capture the IGameEventManager2 instance,
// then hook FireEvent for pre/post game event dispatch
SH_DECL_HOOK2(IGameEventManager2, FireEvent, SH_NOATTRIB, 0, bool, IGameEvent*, bool);
SH_DECL_HOOK2(IGameEventManager2, LoadEventsFromFile, SH_NOATTRIB, 0, int, const char*, bool);

static int g_iLoadEventsFromFileHookId = 0;
static bool g_bFireEventHooked = false;

// Note: Chat interception now uses funchook on Host_Say (see chat_manager.cpp)
// instead of ClientCommand hook, which doesn't fire for say commands in Source 2
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

    GET_V_IFACE_ANY(GetEngineFactory, gs_pNetworkServerService,
                    INetworkServerService, NETWORKSERVERSERVICE_INTERFACE_VERSION);

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
    // Hook IGameEventManager2 to capture game events
    // ============================================================
    // CSSharp approach: find the CGameEventManager vtable in libserver.so,
    // hook LoadEventsFromFile to capture the runtime IGameEventManager2 instance,
    // then hook FireEvent for pre/post dispatch to Go plugins.

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
            "csgo/addons/gostrike/configs/gamedata/gamedata.json",
            "addons/gostrike/configs/gamedata/gamedata.json",
            "./csgo/addons/gostrike/configs/gamedata/gamedata.json",
            "/home/steam/cs2-dedicated/game/csgo/addons/gostrike/configs/gamedata/gamedata.json",
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

    // Hook LoadEventsFromFile on CGameEventManager vtable to capture the runtime instance
    // (same approach as CSSharp - we need the instance pointer before we can hook FireEvent)
    if (gostrike::modules::server.IsInitialized()) {
        // The vtable symbol for CGameEventManager in ELF: _ZTV20CGameEventManager
        void* pVTable = gostrike::modules::server.FindSymbol("_ZTV20CGameEventManager");
        if (pVTable) {
            // Skip past the RTTI offset and typeinfo pointer (2 pointers)
            auto* pVTableStart = reinterpret_cast<IGameEventManager2*>(
                reinterpret_cast<uintptr_t>(pVTable) + 2 * sizeof(void*));
            g_iLoadEventsFromFileHookId = SH_ADD_DVPHOOK(IGameEventManager2, LoadEventsFromFile,
                pVTableStart, SH_MEMBER(&g_Plugin, &GoStrikePlugin::Hook_LoadEventsFromFile), false);
            ConPrintf("[GoStrike] CGameEventManager vtable found, LoadEventsFromFile hooked\n");
        } else {
            ConPrintf("[GoStrike] WARNING: CGameEventManager vtable not found - game events will not work\n");
        }
    }

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
    // Shutdown damage hook
    gostrike::GameFunc_ShutdownDamageHook();

    // Shutdown chat manager (unhook Host_Say)
    gostrike::ChatManager_Shutdown();

    // Shutdown entity system
    gostrike::EntitySystem_Shutdown();

    // Remove FireEvent hooks
    if (g_bFireEventHooked && gs_pGameEventManager) {
        SH_REMOVE_HOOK_MEMFUNC(IGameEventManager2, FireEvent, gs_pGameEventManager,
            &g_Plugin, &GoStrikePlugin::Hook_FireEvent, false);
        SH_REMOVE_HOOK_MEMFUNC(IGameEventManager2, FireEvent, gs_pGameEventManager,
            &g_Plugin, &GoStrikePlugin::Hook_FireEventPost, true);
        g_bFireEventHooked = false;
        ConPrintf("[GoStrike] FireEvent hooks removed\n");
    }

    // Remove LoadEventsFromFile vtable hook
    if (g_iLoadEventsFromFileHookId) {
        SH_REMOVE_HOOK_ID(g_iLoadEventsFromFileHookId);
        g_iLoadEventsFromFileHookId = 0;
    }

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
    // Acquire CGlobalVars from network server service
    if (gs_pNetworkServerService) {
        auto* gameServer = gs_pNetworkServerService->GetIGameServer();
        if (gameServer) {
            gs_pGlobals = gameServer->GetGlobals();
            ConPrintf("[GoStrike] CGlobalVars acquired: %p\n", gs_pGlobals);
        }
    }

    // Initialize entity system now that everything is ready
    // Set the global GameEntitySystem() pointer before initializing
    // (entity_system.cpp and SDK's entitysystem.cpp both need it)
    gostrike::EntitySystem_Initialize();
    s_pGameEntitySystem = static_cast<CGameEntitySystem*>(gostrike::EntitySystem_GetSystemPtr());

    // Initialize game function pointers from gamedata
    gostrike::GameFunctions_Initialize();

    // Initialize damage hook (funchook on CBaseEntity_TakeDamageOld)
    gostrike::GameFunc_InitDamageHook();

    // Initialize chat manager (TextMsg outbound + Host_Say hook for inbound)
    gostrike::ChatManager_Initialize();

    // Hook FireEvent on IGameEventManager2 if we captured the instance
    if (gs_pGameEventManager && !g_bFireEventHooked) {
        SH_ADD_HOOK_MEMFUNC(IGameEventManager2, FireEvent, gs_pGameEventManager,
            &g_Plugin, &GoStrikePlugin::Hook_FireEvent, false);
        SH_ADD_HOOK_MEMFUNC(IGameEventManager2, FireEvent, gs_pGameEventManager,
            &g_Plugin, &GoStrikePlugin::Hook_FireEventPost, true);
        g_bFireEventHooked = true;
        ConPrintf("[GoStrike] FireEvent hooks installed on IGameEventManager2\n");
    } else if (!gs_pGameEventManager) {
        ConPrintf("[GoStrike] WARNING: IGameEventManager2 not yet captured - game events may not work\n");
        ConPrintf("[GoStrike] Events will be hooked when LoadEventsFromFile is called\n");
    }
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

#ifndef USE_STUB_SDK
    // Lazily acquire CGlobalVars on first available tick
    if (!gs_pGlobals && gs_pNetworkServerService) {
        auto* gameServer = gs_pNetworkServerService->GetIGameServer();
        if (gameServer) {
            gs_pGlobals = gameServer->GetGlobals();
            if (gs_pGlobals) {
                ConPrintf("[GoStrike] CGlobalVars acquired: %p\n", gs_pGlobals);
            }
        }
    }
#endif

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

    GoBridge_RefreshPlayerCache();

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

// ============================================================
// Game Event Hooks (IGameEventManager2::FireEvent)
// Inspired by CounterStrikeSharp's event_manager.cpp
// ============================================================

#ifndef USE_STUB_SDK
int GoStrikePlugin::Hook_LoadEventsFromFile(const char* filename, bool bSearchAll) {
    // Capture the IGameEventManager2 runtime instance via META_IFACEPTR
    // (same approach as CSSharp - the vtable hook fires when any instance calls this method)
    if (!gs_pGameEventManager) {
        gs_pGameEventManager = META_IFACEPTR(IGameEventManager2);
        ConPrintf("[GoStrike] IGameEventManager2 captured: %p\n", gs_pGameEventManager);

        // Now install FireEvent hooks
        if (!g_bFireEventHooked) {
            SH_ADD_HOOK_MEMFUNC(IGameEventManager2, FireEvent, gs_pGameEventManager,
                &g_Plugin, &GoStrikePlugin::Hook_FireEvent, false);
            SH_ADD_HOOK_MEMFUNC(IGameEventManager2, FireEvent, gs_pGameEventManager,
                &g_Plugin, &GoStrikePlugin::Hook_FireEventPost, true);
            g_bFireEventHooked = true;
            ConPrintf("[GoStrike] FireEvent hooks installed on IGameEventManager2\n");
        }
    }
    RETURN_META_VALUE(MRES_IGNORED, 0);
}

bool GoStrikePlugin::Hook_FireEvent(IGameEvent* pEvent, bool bDontBroadcast) {
    if (!pEvent) {
        RETURN_META_VALUE(MRES_IGNORED, false);
    }

    const char* eventName = pEvent->GetName();

    // Dispatch to Go (pre-hook: plugins can block or modify the event)
    gs_event_result_t result = GoBridge_FireEvent(eventName, pEvent, false);

    if (result >= GS_EVENT_HANDLED) {
        // Plugin wants to suppress this event
        RETURN_META_VALUE(MRES_SUPERCEDE, false);
    }

    RETURN_META_VALUE(MRES_IGNORED, true);
}

bool GoStrikePlugin::Hook_FireEventPost(IGameEvent* pEvent, bool bDontBroadcast) {
    if (!pEvent) {
        RETURN_META_VALUE(MRES_IGNORED, false);
    }

    const char* eventName = pEvent->GetName();

    // Dispatch to Go (post-hook: informational only, can't modify)
    GoBridge_FireEvent(eventName, pEvent, true);

    RETURN_META_VALUE(MRES_IGNORED, true);
}
#else
int GoStrikePlugin::Hook_LoadEventsFromFile(const char*, bool) { return 0; }
bool GoStrikePlugin::Hook_FireEvent(IGameEvent*, bool) { return true; }
bool GoStrikePlugin::Hook_FireEventPost(IGameEvent*, bool) { return true; }
#endif

// Note: Chat interception is handled by Host_Say funchook in chat_manager.cpp
// ClientCommand hook was removed as it doesn't fire for say commands in Source 2
