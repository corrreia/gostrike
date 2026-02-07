// gostrike.h - GoStrike Metamod:Source Plugin Header
// Architecture inspired by CounterStrikeSharp (https://github.com/roflmuffin/CounterStrikeSharp)
#ifndef GOSTRIKE_H
#define GOSTRIKE_H

// Platform compatibility
#ifdef __linux__
    #include <strings.h>
#endif

#ifdef USE_STUB_SDK
    #include "stub/ISmmPlugin.h"
    #include "stub/igameevents.h"
    #include "stub/iplayerinfo.h"
    #include "stub/sh_vector.h"
#else
    // Metamod:Source headers
    #include <ISmmPlugin.h>
    #include <sh_vector.h>
    #include <sourcehook.h>
    #include <sourcehook_impl.h>

    // HL2SDK headers
    #include <eiface.h>
    #include <igameevents.h>
    #include <iserver.h>
    #include <entity2/entitysystem.h>
    #include <entity2/entityidentity.h>
    #include <schemasystem/schemasystem.h>
    #include <icvar.h>
    #include <tier1/convar.h>
    #include <playerslot.h>
    #include <igameeventsystem.h>
    #include <networksystem/inetworkmessages.h>
    #include <networksystem/netmessage.h>
    #include <inetchannel.h>
    #include <irecipientfilter.h>
    #include <iserver.h>
#endif

#include "gostrike_abi.h"

// Forward declarations
class IGameEventManager2;
class CGlobalVars;

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
    // NOTE: uint64 not uint64_t - must match SDK types exactly
    bool Hook_ClientConnect(CPlayerSlot slot, const char* pszName,
                           uint64 xuid, const char* pszNetworkID,
                           bool unk1, CBufferString* pRejectReason);
    void Hook_ClientDisconnect(CPlayerSlot slot, ENetworkDisconnectionReason reason,
                               const char* pszName, uint64 xuid, const char* pszNetworkID);
    void Hook_ClientPutInServer(CPlayerSlot slot, char const* pszName, int type, uint64 xuid);

    // Note: Chat interception uses funchook on Host_Say (see chat_manager.cpp)

private:
    bool m_bLateLoad;
};

// Global plugin instance
extern GoStrikePlugin g_Plugin;

// Global engine interfaces (using gs_ prefix to avoid SDK conflicts)
#ifndef USE_STUB_SDK
extern IVEngineServer2*        gs_pEngineServer2;
extern ISource2Server*         gs_pSource2Server;
extern ICvar*                  gs_pCVar;
extern IGameEventSystem*       gs_pGameEventSystem;
extern CSchemaSystem*          gs_pSchemaSystem;
extern INetworkMessages*       gs_pNetworkMessages;
extern IServerGameClients*     gs_pServerGameClients;
extern CGlobalVars*            gs_pGlobals;
extern IGameResourceService*   gs_pGameResourceService;
extern INetworkServerService*  gs_pNetworkServerService;
#else
extern void* gs_pEngineServer2;
extern void* gs_pSource2Server;
extern void* gs_pCVar;
extern void* gs_pGameEventSystem;
extern void* gs_pSchemaSystem;
extern void* gs_pNetworkMessages;
extern void* gs_pServerGameClients;
extern void* gs_pGlobals;
extern void* gs_pGameResourceService;
extern void* gs_pNetworkServerService;
#endif

// Metamod globals
PLUGIN_GLOBALVARS();

#endif // GOSTRIKE_H
