// chat_manager.cpp - In-game messaging and chat interception
// Inspired by CounterStrikeSharp's chat manager and UTIL_ClientPrint

#include "chat_manager.h"
#include "gostrike.h"
#include "gameconfig.h"
#include "player_manager.h"

#include <cstdio>

#ifndef USE_STUB_SDK
#include <entity2/entityinstance.h>
#include <playerslot.h>
#endif

namespace gostrike {

// Function pointer types matching engine signatures
// void UTIL_ClientPrint(CBasePlayerController* player, int msg_dest, const char* msg_name, ...)
typedef void (*ClientPrintFn)(void* player, int msg_dest, const char* msg_name,
                              const char* param1, const char* param2,
                              const char* param3, const char* param4);

// void UTIL_ClientPrintAll(int msg_dest, const char* msg_name, ...)
typedef void (*ClientPrintAllFn)(int msg_dest, const char* msg_name,
                                  const char* param1, const char* param2,
                                  const char* param3, const char* param4);

static ClientPrintFn s_fnClientPrint = nullptr;
static ClientPrintAllFn s_fnClientPrintAll = nullptr;

void ChatManager_Initialize() {
    // Resolve UTIL_ClientPrint from gamedata
    void* addr = g_gameConfig.ResolveSignature("ClientPrint");
    if (addr) {
        s_fnClientPrint = reinterpret_cast<ClientPrintFn>(addr);
        printf("[GoStrike] ChatManager: ClientPrint resolved at %p\n", addr);
    } else {
        printf("[GoStrike] ChatManager: WARNING - ClientPrint not found\n");
    }

    // Resolve UTIL_ClientPrintAll from gamedata
    addr = g_gameConfig.ResolveSignature("UTIL_ClientPrintAll");
    if (addr) {
        s_fnClientPrintAll = reinterpret_cast<ClientPrintAllFn>(addr);
        printf("[GoStrike] ChatManager: UTIL_ClientPrintAll resolved at %p\n", addr);
    } else {
        printf("[GoStrike] ChatManager: WARNING - UTIL_ClientPrintAll not found\n");
    }

    printf("[GoStrike] ChatManager: initialized\n");
}

void ClientPrint(int32_t slot, int dest, const char* msg) {
#ifndef USE_STUB_SDK
    if (!msg) return;

    if (s_fnClientPrint) {
        // Get the player controller for this slot
        void* controller = PlayerManager_GetController(slot);
        if (!controller) {
            printf("[GoStrike] ClientPrint: no controller for slot %d\n", slot);
            return;
        }
        s_fnClientPrint(controller, dest, msg, nullptr, nullptr, nullptr, nullptr);
    } else {
        // Fallback: just log it
        printf("[GoStrike] ClientPrint (slot=%d, dest=%d): %s\n", slot, dest, msg);
    }
#else
    printf("[GoStrike] ClientPrint (slot=%d, dest=%d): %s\n", slot, dest, msg ? msg : "(null)");
#endif
}

void ClientPrintAll(int dest, const char* msg) {
#ifndef USE_STUB_SDK
    if (!msg) return;

    if (s_fnClientPrintAll) {
        s_fnClientPrintAll(dest, msg, nullptr, nullptr, nullptr, nullptr);
    } else {
        // Fallback: just log it
        printf("[GoStrike] ClientPrintAll (dest=%d): %s\n", dest, msg);
    }
#else
    printf("[GoStrike] ClientPrintAll (dest=%d): %s\n", dest, msg ? msg : "(null)");
#endif
}

} // namespace gostrike
