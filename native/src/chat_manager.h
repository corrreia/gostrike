// chat_manager.h - In-game messaging and chat interception
// Chat interception via Host_Say hook inspired by CounterStrikeSharp
// (https://github.com/roflmuffin/CounterStrikeSharp)

#ifndef GOSTRIKE_CHAT_MANAGER_H
#define GOSTRIKE_CHAT_MANAGER_H

#include <cstdint>

// HUD message destinations (matching engine constants)
#define GS_HUD_PRINTNOTIFY  1
#define GS_HUD_PRINTCONSOLE 2
#define GS_HUD_PRINTTALK    3
#define GS_HUD_PRINTCENTER  4
#define GS_HUD_PRINTALERT   5

namespace gostrike {

// Initialize the chat manager:
// - Resolves TextMsg network message for outbound messaging
// - Hooks Host_Say via funchook for inbound chat interception
void ChatManager_Initialize();

// Shutdown the chat manager (unhook Host_Say)
void ChatManager_Shutdown();

// Send a message to a specific player using the engine's messaging system
// slot: player slot (0-63)
// dest: GS_HUD_PRINTTALK, GS_HUD_PRINTCENTER, etc.
// msg: message text
void ClientPrint(int32_t slot, int dest, const char* msg);

// Send a message to all players
// dest: GS_HUD_PRINTTALK, GS_HUD_PRINTCENTER, etc.
// msg: message text
void ClientPrintAll(int dest, const char* msg);

} // namespace gostrike

#endif // GOSTRIKE_CHAT_MANAGER_H
