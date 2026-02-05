// go_bridge.h - C++ to Go bridge interface
#ifndef GO_BRIDGE_H
#define GO_BRIDGE_H

#include "gostrike_abi.h"

// Initialize the Go bridge (load Go shared library and initialize runtime)
// Returns true on success, false on failure
bool GoBridge_Init(void);

// Shutdown the Go bridge (shutdown runtime and unload library)
void GoBridge_Shutdown(void);

// Check if the Go bridge is initialized
bool GoBridge_IsInitialized(void);

// Register C++ callbacks with Go
void GoBridge_RegisterCallbacks(void);

// Dispatch a tick to Go
void GoBridge_OnTick(float deltaTime);

// Fire a game event to Go
// Returns the event result from Go handlers
gs_event_result_t GoBridge_FireEvent(const char* name, void* event, bool isPost);

// Notify Go of player connect
void GoBridge_OnPlayerConnect(gs_player_t* player);

// Notify Go of player disconnect
void GoBridge_OnPlayerDisconnect(int32_t slot, const char* reason);

// Notify Go of map change
void GoBridge_OnMapChange(const char* mapName);

// Process a chat message (check for !commands)
// Returns true if message was a command and should be suppressed
bool GoBridge_OnChatMessage(int32_t playerSlot, const char* message);

// Get the last error message from Go (caller must free)
char* GoBridge_GetLastError(void);

#endif // GO_BRIDGE_H
