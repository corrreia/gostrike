// gostrike.cpp - GoStrike Metamod:Source Plugin Implementation
#include "gostrike.h"
#include "go_bridge.h"
#include <stdio.h>
#include <string.h>

// Plugin instance and Metamod exposure
GoStrikePlugin g_Plugin;
PLUGIN_EXPOSE(GoStrikePlugin, g_Plugin);

// Global engine interfaces
IVEngineServer* g_pEngineServer = nullptr;
ISource2Server* g_pSource2Server = nullptr;
ICvar* g_pCVar = nullptr;
IGameEventManager2* g_pGameEventManager = nullptr;
CGlobalVars* g_pGlobals = nullptr;

// Hook declarations using SourceHook macros
// Note: Actual hook signatures depend on the CS2/Source2 SDK version
// These are simplified placeholders

// Helper function for console output
static void ConPrint(const char* msg) {
    // Use META_CONPRINTF if available, otherwise printf
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
// ISmmPlugin Implementation
// ============================================================

bool GoStrikePlugin::Load(PluginId id, ISmmAPI* ismm, char* error, size_t maxlen, bool late) {
    PLUGIN_SAVEVARS();
    
    m_bLateLoad = late;
    
    ConPrintf("[GoStrike] Loading plugin (late=%s)...\n", late ? "true" : "false");
    
    // Get engine interfaces
    // Note: Interface names and macros depend on the specific SDK version
    // These are examples - actual implementation needs proper SDK headers
    
    /*
    GET_V_IFACE_CURRENT(GetEngineFactory, g_pEngineServer, 
                        IVEngineServer, INTERFACEVERSION_VENGINESERVER);
    GET_V_IFACE_CURRENT(GetEngineFactory, g_pCVar, 
                        ICvar, CVAR_INTERFACE_VERSION);
    GET_V_IFACE_ANY(GetEngineFactory, g_pGameEventManager,
                    IGameEventManager2, INTERFACEVERSION_GAMEEVENTSMANAGER2);
    GET_V_IFACE_ANY(GetServerFactory, g_pSource2Server,
                    ISource2Server, SOURCE2SERVER_INTERFACE_VERSION);
    */
    
    // Initialize Go runtime
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
    
    // Register as Metamod listener for events
    if (ismm) {
        ismm->AddListener(this, this);
    }
    
    // Register console commands
    // META_REGCMD(gostrike_status);
    // META_REGCMD(gostrike_reload);
    
    // Hook game events if event manager is available
    /*
    if (g_pGameEventManager) {
        g_pGameEventManager->AddListener(this, "player_connect", true);
        g_pGameEventManager->AddListener(this, "player_disconnect", true);
        g_pGameEventManager->AddListener(this, "player_death", true);
        g_pGameEventManager->AddListener(this, "round_start", true);
        g_pGameEventManager->AddListener(this, "round_end", true);
    }
    */
    
    ConPrintf("[GoStrike] Plugin loaded successfully (version %s)\n", GOSTRIKE_VERSION);
    return true;
}

bool GoStrikePlugin::Unload(char* error, size_t maxlen) {
    ConPrintf("[GoStrike] Unloading plugin...\n");
    
    // Unhook game events
    /*
    if (g_pGameEventManager) {
        g_pGameEventManager->RemoveListener(this);
    }
    */
    
    // Unregister console commands
    // META_UNREGCMD(gostrike_status);
    // META_UNREGCMD(gostrike_reload);
    
    // Shutdown Go runtime
    GoBridge_Shutdown();
    
    ConPrintf("[GoStrike] Plugin unloaded\n");
    return true;
}

void GoStrikePlugin::AllPluginsLoaded() {
    ConPrintf("[GoStrike] All plugins loaded\n");
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
    return "GoStrike Team";
}

const char* GoStrikePlugin::GetName() {
    return "GoStrike";
}

const char* GoStrikePlugin::GetDescription() {
    return "Go-based CS2 modding framework";
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
    // Calculate delta time
    static float lastTime = 0.0f;
    float currentTime = 0.0f; // Would come from g_pGlobals->curtime
    float deltaTime = currentTime - lastTime;
    lastTime = currentTime;
    
    // Dispatch tick to Go
    GoBridge_OnTick(deltaTime);
    
    // Note: In actual implementation, use RETURN_META(MRES_IGNORED);
}

bool GoStrikePlugin::Hook_ClientConnect(CPlayerSlot slot, const char* pszName,
                                        uint64_t xuid, const char* pszNetworkID,
                                        bool unk1, CBufferString* pRejectReason) {
    ConPrintf("[GoStrike] Client connecting: %s (slot %d)\n", pszName, slot.Get());
    
    // Build player info
    gs_player_t player = {};
    player.slot = slot.Get();
    player.steam_id = xuid;
    player.name = const_cast<char*>(pszName);
    player.ip = const_cast<char*>(pszNetworkID);
    player.is_bot = false;
    player.is_alive = false;
    player.team = GS_TEAM_UNASSIGNED;
    
    // Notify Go
    GoBridge_OnPlayerConnect(&player);
    
    // Note: Return MRES_IGNORED to allow connection
    return true;
}

void GoStrikePlugin::Hook_ClientDisconnect(CPlayerSlot slot, ENetworkDisconnectionReason reason,
                                           const char* pszName, uint64_t xuid,
                                           const char* pszNetworkID) {
    ConPrintf("[GoStrike] Client disconnected: %s (slot %d)\n", pszName, slot.Get());
    
    // Notify Go
    GoBridge_OnPlayerDisconnect(slot.Get(), "disconnect");
}

void GoStrikePlugin::Hook_ClientPutInServer(CPlayerSlot slot, char const* pszName,
                                            int type, uint64_t xuid) {
    ConPrintf("[GoStrike] Client put in server: %s (slot %d)\n", pszName, slot.Get());
}

void GoStrikePlugin::OnFireGameEvent(IGameEvent* event) {
    if (!event) return;
    
    const char* name = event->GetName();
    if (!name) return;
    
    // Dispatch to Go
    GoBridge_FireEvent(name, event, false);
}

// ============================================================
// Console Commands
// ============================================================

// CON_COMMAND(gostrike_status, "Show GoStrike status") {
//     ConPrintf("[GoStrike] Status:\n");
//     ConPrintf("  Version: %s\n", GOSTRIKE_VERSION);
//     ConPrintf("  ABI Version: %d\n", GOSTRIKE_ABI_VERSION);
//     ConPrintf("  Go Runtime: %s\n", GoBridge_IsInitialized() ? "Running" : "Not Running");
// }

// CON_COMMAND(gostrike_test, "Test GoStrike command") {
//     gs_command_ctx_t ctx = {};
//     ctx.player_slot = -1;  // Server console
//     ctx.command = "gostrike_test";
//     ctx.args = args.ArgS();
//     ctx.argc = args.ArgC();
//     
//     GoBridge_OnCommand(&ctx);
// }
