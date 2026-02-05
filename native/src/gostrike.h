// gostrike.h - GoStrike Metamod:Source Plugin Header
#ifndef GOSTRIKE_H
#define GOSTRIKE_H

// Use stub headers if full SDK not available
#ifdef USE_STUB_SDK
    #include "stub/ISmmPlugin.h"
    #include "stub/igameevents.h"
    #include "stub/iplayerinfo.h"
    #include "stub/sh_vector.h"
#else
    #include <ISmmPlugin.h>
    #include <igameevents.h>
    #include <iplayerinfo.h>
    #include <sh_vector.h>
#endif

#include "gostrike_abi.h"

// Forward declarations
class IVEngineServer;
class IServerGameDLL;
class IGameEventManager2;
class IServerPluginHelpers;
class CGlobalVars;
class ISource2Server;

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
extern IVEngineServer* g_pEngineServer;
extern ISource2Server* g_pSource2Server;
extern IGameEventManager2* g_pGameEventManager;
extern CGlobalVars* g_pGlobals;

// Metamod globals
PLUGIN_GLOBALVARS();

#endif // GOSTRIKE_H
