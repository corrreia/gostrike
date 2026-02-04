// Package bridge provides the CGO bridge between the C++ native plugin and Go runtime.
// This file contains Go functions that call back into C++ via the registered callbacks.
package bridge

/*
#cgo CFLAGS: -I../../native/include
#include "gostrike_abi.h"
#include <stdlib.h>

// Helper to call function pointers from Go
static inline void call_log(gs_callbacks_t* cb, int level, const char* tag, const char* msg) {
    if (cb && cb->log) {
        cb->log(level, tag, msg);
    }
}

static inline void call_exec_command(gs_callbacks_t* cb, const char* cmd) {
    if (cb && cb->exec_command) {
        cb->exec_command(cmd);
    }
}

static inline void call_reply(gs_callbacks_t* cb, int32_t slot, const char* msg) {
    if (cb && cb->reply_to_command) {
        cb->reply_to_command(slot, msg);
    }
}

static inline gs_player_t* call_get_player(gs_callbacks_t* cb, int32_t slot) {
    if (cb && cb->get_player) {
        return cb->get_player(slot);
    }
    return NULL;
}

static inline int32_t call_get_player_count(gs_callbacks_t* cb) {
    if (cb && cb->get_player_count) {
        return cb->get_player_count();
    }
    return 0;
}

static inline int32_t call_get_all_players(gs_callbacks_t* cb, int32_t* out_slots) {
    if (cb && cb->get_all_players) {
        return cb->get_all_players(out_slots);
    }
    return 0;
}

static inline void call_kick_player(gs_callbacks_t* cb, int32_t slot, const char* reason) {
    if (cb && cb->kick_player) {
        cb->kick_player(slot, reason);
    }
}

static inline const char* call_get_map_name(gs_callbacks_t* cb) {
    if (cb && cb->get_map_name) {
        return cb->get_map_name();
    }
    return "unknown";
}

static inline int32_t call_get_max_players(gs_callbacks_t* cb) {
    if (cb && cb->get_max_players) {
        return cb->get_max_players();
    }
    return 64;
}

static inline int32_t call_get_tick_rate(gs_callbacks_t* cb) {
    if (cb && cb->get_tick_rate) {
        return cb->get_tick_rate();
    }
    return 64;
}

static inline void call_send_chat(gs_callbacks_t* cb, int32_t slot, const char* msg) {
    if (cb && cb->send_chat) {
        cb->send_chat(slot, msg);
    }
}

static inline void call_send_center(gs_callbacks_t* cb, int32_t slot, const char* msg) {
    if (cb && cb->send_center) {
        cb->send_center(slot, msg);
    }
}
*/
import "C"
import (
	"fmt"
	"unsafe"
)

// ============================================================
// Logging
// ============================================================

// Log writes a message to the server console via C++
func Log(level int, tag, message string) {
	if callbacks == nil {
		// Fallback to stdout if callbacks not registered
		fmt.Printf("[%s] %s\n", tag, message)
		return
	}

	cTag := C.CString(tag)
	cMsg := C.CString(message)
	defer C.free(unsafe.Pointer(cTag))
	defer C.free(unsafe.Pointer(cMsg))

	C.call_log(callbacks, C.int(level), cTag, cMsg)
}

// LogDebug logs a debug message
func LogDebug(tag, format string, args ...interface{}) {
	Log(int(C.GS_LOG_DEBUG), tag, fmt.Sprintf(format, args...))
}

// LogInfo logs an info message
func LogInfo(tag, format string, args ...interface{}) {
	Log(int(C.GS_LOG_INFO), tag, fmt.Sprintf(format, args...))
}

// LogWarning logs a warning message
func LogWarning(tag, format string, args ...interface{}) {
	Log(int(C.GS_LOG_WARNING), tag, fmt.Sprintf(format, args...))
}

// LogError logs an error message
func LogError(tag, format string, args ...interface{}) {
	Log(int(C.GS_LOG_ERROR), tag, fmt.Sprintf(format, args...))
}

// ============================================================
// Command Execution
// ============================================================

// ExecuteServerCommand executes a command on the server console
func ExecuteServerCommand(cmd string) {
	if callbacks == nil {
		return
	}

	cCmd := C.CString(cmd)
	defer C.free(unsafe.Pointer(cCmd))

	C.call_exec_command(callbacks, cCmd)
}

// ReplyToCommand sends a reply message to a command invoker
// slot: -1 for server console, >= 0 for player slot
func ReplyToCommand(slot int, message string) {
	if callbacks == nil {
		return
	}

	cMsg := C.CString(message)
	defer C.free(unsafe.Pointer(cMsg))

	C.call_reply(callbacks, C.int32_t(slot), cMsg)
}

// ReplyToCommandf sends a formatted reply message
func ReplyToCommandf(slot int, format string, args ...interface{}) {
	ReplyToCommand(slot, fmt.Sprintf(format, args...))
}

// ============================================================
// Player Information
// ============================================================

// PlayerInfo contains player information retrieved from C++
// This is an alias to the shared type
type PlayerInfo = struct {
	Slot    int
	UserID  int
	SteamID uint64
	Name    string
	IP      string
	Team    int
	IsAlive bool
	IsBot   bool
	Health  int
	Armor   int
	PosX    float64
	PosY    float64
	PosZ    float64
}

// GetPlayer retrieves player information by slot
// Returns nil if the player doesn't exist
func GetPlayer(slot int) *PlayerInfo {
	if callbacks == nil {
		return nil
	}

	cPlayer := C.call_get_player(callbacks, C.int32_t(slot))
	if cPlayer == nil {
		return nil
	}

	player := &PlayerInfo{
		Slot:    int(cPlayer.slot),
		UserID:  int(cPlayer.user_id),
		SteamID: uint64(cPlayer.steam_id),
		Team:    int(cPlayer.team),
		IsAlive: bool(cPlayer.is_alive),
		IsBot:   bool(cPlayer.is_bot),
		Health:  int(cPlayer.health),
		Armor:   int(cPlayer.armor),
		PosX:    float64(cPlayer.position.x),
		PosY:    float64(cPlayer.position.y),
		PosZ:    float64(cPlayer.position.z),
	}

	if cPlayer.name != nil {
		player.Name = C.GoString(cPlayer.name)
	}
	if cPlayer.ip != nil {
		player.IP = C.GoString(cPlayer.ip)
	}

	return player
}

// GetPlayerCount returns the number of connected players
func GetPlayerCount() int {
	if callbacks == nil {
		return 0
	}
	return int(C.call_get_player_count(callbacks))
}

// GetAllPlayers returns all connected player slots
func GetAllPlayers() []int {
	if callbacks == nil {
		return nil
	}

	// Allocate array for slots
	var slots [64]C.int32_t
	count := C.call_get_all_players(callbacks, &slots[0])

	if count <= 0 {
		return nil
	}

	result := make([]int, int(count))
	for i := 0; i < int(count); i++ {
		result[i] = int(slots[i])
	}
	return result
}

// GetAllPlayerInfos returns PlayerInfo for all connected players
func GetAllPlayerInfos() []*PlayerInfo {
	slots := GetAllPlayers()
	if len(slots) == 0 {
		return nil
	}

	players := make([]*PlayerInfo, 0, len(slots))
	for _, slot := range slots {
		if player := GetPlayer(slot); player != nil {
			players = append(players, player)
		}
	}
	return players
}

// KickPlayer removes a player from the server
func KickPlayer(slot int, reason string) {
	if callbacks == nil {
		return
	}

	cReason := C.CString(reason)
	defer C.free(unsafe.Pointer(cReason))

	C.call_kick_player(callbacks, C.int32_t(slot), cReason)
}

// ============================================================
// Server Information
// ============================================================

// GetMapName returns the current map name
func GetMapName() string {
	if callbacks == nil {
		return "unknown"
	}

	cName := C.call_get_map_name(callbacks)
	if cName == nil {
		return "unknown"
	}
	return C.GoString(cName)
}

// GetMaxPlayers returns the maximum number of players
func GetMaxPlayers() int {
	if callbacks == nil {
		return 64
	}
	return int(C.call_get_max_players(callbacks))
}

// GetTickRate returns the server tick rate
func GetTickRate() int {
	if callbacks == nil {
		return 64
	}
	return int(C.call_get_tick_rate(callbacks))
}

// ============================================================
// Messaging
// ============================================================

// SendChat sends a chat message to a player or all players
// slot: -1 for all players, >= 0 for specific player
func SendChat(slot int, message string) {
	if callbacks == nil {
		return
	}

	cMsg := C.CString(message)
	defer C.free(unsafe.Pointer(cMsg))

	C.call_send_chat(callbacks, C.int32_t(slot), cMsg)
}

// SendChatf sends a formatted chat message
func SendChatf(slot int, format string, args ...interface{}) {
	SendChat(slot, fmt.Sprintf(format, args...))
}

// SendChatAll sends a chat message to all players
func SendChatAll(message string) {
	SendChat(-1, message)
}

// SendChatAllf sends a formatted chat message to all players
func SendChatAllf(format string, args ...interface{}) {
	SendChat(-1, fmt.Sprintf(format, args...))
}

// SendCenter sends a center message to a player
func SendCenter(slot int, message string) {
	if callbacks == nil {
		return
	}

	cMsg := C.CString(message)
	defer C.free(unsafe.Pointer(cMsg))

	C.call_send_center(callbacks, C.int32_t(slot), cMsg)
}

// SendCenterf sends a formatted center message
func SendCenterf(slot int, format string, args ...interface{}) {
	SendCenter(slot, fmt.Sprintf(format, args...))
}

// ============================================================
// Callback Status
// ============================================================

// IsCallbacksRegistered returns true if C++ callbacks are registered
func IsCallbacksRegistered() bool {
	return callbacks != nil
}
