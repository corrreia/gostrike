// chat_manager.cpp - In-game messaging and chat interception
// Outbound: TextMsg network messages via PostEventAbstract
// Inbound: Host_Say function hook via funchook
// Both approaches from CounterStrikeSharp (https://github.com/roflmuffin/CounterStrikeSharp)

#include "chat_manager.h"
#include "gostrike.h"
#include "gameconfig.h"
#include "go_bridge.h"

#include <cstdio>
#include <cstring>
#include <string>
#include <funchook.h>

#ifndef USE_STUB_SDK
#include <const.h>
#include <bitvec.h>
#include <networksystem/inetworkmessages.h>
#include <networksystem/netmessage.h>
#include <engine/igameeventsystem.h>
#include <entity2/entitysystem.h>
#include "usermessages.pb.h"
#endif

namespace gostrike {

// ============================================================
// TextMsg outbound messaging (same as before)
// ============================================================

#ifndef USE_STUB_SDK
static INetworkMessageInternal* s_pTextMsg = nullptr;
#endif

// ============================================================
// Host_Say hook via funchook (inspired by CSSharp's chat_manager.cpp)
// ============================================================

#ifndef USE_STUB_SDK
// Host_Say function signature from CSSharp:
// void Host_Say(CEntityInstance* pController, CCommand& args, bool teamonly, int unk1, const char* unk2)
typedef void (*HostSay)(CEntityInstance*, CCommand&, bool, int, const char*);
static HostSay s_pOriginalHostSay = nullptr;
static funchook_t* s_pFunchook = nullptr;

static void DetourHostSay(CEntityInstance* pController, CCommand& args, bool teamonly, int unk1, const char* unk2) {
    if (!pController || args.ArgC() < 2) {
        s_pOriginalHostSay(pController, args, teamonly, unk1, unk2);
        return;
    }

    // Get message text (args[1] is the chat message)
    const char* rawMsg = args.Arg(1);
    if (!rawMsg || rawMsg[0] == '\0') {
        s_pOriginalHostSay(pController, args, teamonly, unk1, unk2);
        return;
    }

    // Get player slot from entity index (same as CSSharp)
    int entityIndex = pController->GetEntityIndex().Get();
    int playerSlot = entityIndex - 1;

    // Strip surrounding quotes if present
    std::string msg(rawMsg);
    if (msg.size() >= 2 && msg.front() == '"' && msg.back() == '"') {
        msg = msg.substr(1, msg.size() - 2);
    }

    if (msg.empty()) {
        s_pOriginalHostSay(pController, args, teamonly, unk1, unk2);
        return;
    }

    // Dispatch to Go - returns true if the message was a command and should be suppressed
    bool handled = GoBridge_OnChatMessage(playerSlot, msg.c_str());

    if (!handled) {
        // Not a command - let the original Host_Say broadcast the message normally
        s_pOriginalHostSay(pController, args, teamonly, unk1, unk2);
    }
    // If handled (was a command like !hello), we suppress by not calling the original
}
#endif

// ============================================================
// Initialization / Shutdown
// ============================================================

void ChatManager_Initialize() {
#ifndef USE_STUB_SDK
    // Initialize TextMsg for outbound messaging
    if (gs_pNetworkMessages) {
        s_pTextMsg = gs_pNetworkMessages->FindNetworkMessagePartial("TextMsg");
        if (s_pTextMsg) {
            printf("[GoStrike] ChatManager: TextMsg network message found\n");
        } else {
            printf("[GoStrike] ChatManager: WARNING - TextMsg not found\n");
        }
    }

    // Hook Host_Say for inbound chat interception
    const char* hostSaySig = g_gameConfig.GetSignature("Host_Say");
    if (!hostSaySig) {
        printf("[GoStrike] ChatManager: WARNING - Host_Say signature not found in gamedata\n");
        printf("[GoStrike] ChatManager: Chat commands (!hello etc.) will not work\n");
    } else {
        void* hostSayAddr = g_gameConfig.ResolveSignature("Host_Say");
        if (!hostSayAddr) {
            printf("[GoStrike] ChatManager: WARNING - Host_Say signature scan failed\n");
            printf("[GoStrike] ChatManager: Chat commands (!hello etc.) will not work\n");
        } else {
            printf("[GoStrike] ChatManager: Host_Say found at %p\n", hostSayAddr);

            s_pOriginalHostSay = reinterpret_cast<HostSay>(hostSayAddr);
            s_pFunchook = funchook_create();
            if (!s_pFunchook) {
                printf("[GoStrike] ChatManager: ERROR - funchook_create() failed\n");
            } else {
                int rv = funchook_prepare(s_pFunchook, (void**)&s_pOriginalHostSay, (void*)&DetourHostSay);
                if (rv != 0) {
                    printf("[GoStrike] ChatManager: ERROR - funchook_prepare() failed: %s\n",
                           funchook_error_message(s_pFunchook));
                    funchook_destroy(s_pFunchook);
                    s_pFunchook = nullptr;
                    s_pOriginalHostSay = nullptr;
                } else {
                    rv = funchook_install(s_pFunchook, 0);
                    if (rv != 0) {
                        printf("[GoStrike] ChatManager: ERROR - funchook_install() failed: %s\n",
                               funchook_error_message(s_pFunchook));
                        funchook_destroy(s_pFunchook);
                        s_pFunchook = nullptr;
                        s_pOriginalHostSay = nullptr;
                    } else {
                        printf("[GoStrike] ChatManager: Host_Say hook installed successfully!\n");
                    }
                }
            }
        }
    }
#endif
    printf("[GoStrike] ChatManager: initialized\n");
}

void ChatManager_Shutdown() {
#ifndef USE_STUB_SDK
    if (s_pFunchook) {
        funchook_uninstall(s_pFunchook, 0);
        funchook_destroy(s_pFunchook);
        s_pFunchook = nullptr;
        s_pOriginalHostSay = nullptr;
        printf("[GoStrike] ChatManager: Host_Say hook removed\n");
    }
#endif
}

// ============================================================
// Outbound messaging
// ============================================================

void ClientPrint(int32_t slot, int dest, const char* msg) {
    if (!msg) return;

#ifndef USE_STUB_SDK
    if (s_pTextMsg && gs_pGameEventSystem) {
        CNetMessage* pMsg = s_pTextMsg->AllocateMessage();
        if (pMsg) {
            auto* data = pMsg->ToPB<CUserMessageTextMsg>();
            data->set_dest(dest);
            data->add_param(msg);

            CPlayerBitVec recipients;
            recipients.Set(slot);

            gs_pGameEventSystem->PostEventAbstract(
                CSplitScreenSlot(-1),
                false,
                ABSOLUTE_PLAYER_LIMIT,
                reinterpret_cast<const uint64*>(recipients.Base()),
                s_pTextMsg,
                data,
                0,
                NetChannelBufType_t::BUF_RELIABLE
            );

            delete data;
            return;
        }
    }
#endif
    // Fallback: log to console
    printf("[GoStrike] ClientPrint (slot=%d, dest=%d): %s\n", slot, dest, msg);
}

void ClientPrintAll(int dest, const char* msg) {
    if (!msg) return;

#ifndef USE_STUB_SDK
    if (s_pTextMsg && gs_pGameEventSystem) {
        CNetMessage* pMsg = s_pTextMsg->AllocateMessage();
        if (pMsg) {
            auto* data = pMsg->ToPB<CUserMessageTextMsg>();
            data->set_dest(dest);
            data->add_param(msg);

            // Post to all clients: nClientCount=-1, clients=NULL
            gs_pGameEventSystem->PostEventAbstract(
                CSplitScreenSlot(-1),
                false,
                -1,
                nullptr,
                s_pTextMsg,
                data,
                0,
                NetChannelBufType_t::BUF_RELIABLE
            );

            delete data;
            return;
        }
    }
#endif
    // Fallback: log to console
    printf("[GoStrike] ClientPrintAll (dest=%d): %s\n", dest, msg);
}

} // namespace gostrike
