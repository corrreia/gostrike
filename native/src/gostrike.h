// gostrike.h - GoStrike Metamod:Source Plugin Header
#ifndef GOSTRIKE_H
#define GOSTRIKE_H

// Platform compatibility for HL2SDK
#ifndef USE_STUB_SDK
    // Linux-specific defines required by HL2SDK
    #ifdef __linux__
        #include <strings.h>
        #define stricmp strcasecmp
        #define strnicmp strncasecmp
    #endif
#endif

// Use stub headers if full SDK not available
#ifdef USE_STUB_SDK
    #include "stub/ISmmPlugin.h"
    #include "stub/igameevents.h"
    #include "stub/iplayerinfo.h"
    #include "stub/sh_vector.h"
#else
    #include <ISmmPlugin.h>
    #include <igameevents.h>
    // Note: iplayerinfo.h is Source 1 - CS2 uses entity system instead
    #include <sh_vector.h>
#endif

#include "gostrike_abi.h"

// Forward declarations - only for types not typedef'd in SDK
class IGameEventManager2;
class IServerPluginHelpers;
class CGlobalVars;

// NOTE: INetworkMessages and IGameEventSystem are not used currently.
// CS2's UserMessage system requires complex protobuf integration.

// GoStrike plugin class implementing ISmmPlugin and IMetamodListener
class GoStrikePlugin : public ISmmPlugin, public IMetamodListener
{
public:
    // ISmmPlugin interface
    bool Load(PluginId id, ISmmAPI* ismm, char* error, size_t maxlen, bool late) override;
    bool Unload(char* error, size_t maxlen) override;
    void AllPluginsLoaded() override;
    bool Pause(char* error, size_t maxlen) override;
    bool Unpause(char* error, size_t maxlen) override;
    
    // Plugin metadata
    const char* GetAuthor() override;
    const char* GetName() override;
    const char* GetDescription() override;
    const char* GetURL() override;
    const char* GetLicense() override;
    const char* GetVersion() override;
    const char* GetDate() override;
    const char* GetLogTag() override;

public:
    // Game frame hook (called every tick)
    void Hook_GameFrame(bool simulating, bool bFirstTick, bool bLastTick);
    
    // Client connect/disconnect hooks
    bool Hook_ClientConnect(CPlayerSlot slot, const char* pszName, 
                           uint64_t xuid, const char* pszNetworkID, 
                           bool unk1, CBufferString* pRejectReason);
    void Hook_ClientDisconnect(CPlayerSlot slot, ENetworkDisconnectionReason reason,
                               const char* pszName, uint64_t xuid, const char* pszNetworkID);
    void Hook_ClientPutInServer(CPlayerSlot slot, char const* pszName, int type, uint64_t xuid);
    
    // Event handler
    void OnFireGameEvent(IGameEvent* event);

private:
    bool m_bLateLoad;
};

// Global plugin instance
extern GoStrikePlugin g_Plugin;

// Global engine interfaces
// Note: IVEngineServer and ISource2Server are typedef'd in eiface.h
#ifndef USE_STUB_SDK
extern IVEngineServer* g_pEngineServer;
extern ISource2Server* g_pSource2Server;
#else
extern void* g_pEngineServer;
extern void* g_pSource2Server;
#endif
extern IGameEventManager2* g_pGameEventManager;
extern CGlobalVars* g_pGlobals;

// Metamod globals
PLUGIN_GLOBALVARS();

#endif // GOSTRIKE_H
