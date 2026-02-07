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

// === V2 Callback Helpers (Phase 1: Foundation) ===

// Schema
static inline int32_t call_schema_get_offset(gs_callbacks_t* cb, const char* class_name, const char* field_name, bool* is_networked) {
    if (cb && cb->schema_get_offset) {
        return cb->schema_get_offset(class_name, field_name, is_networked);
    }
    return 0;
}

static inline void call_schema_set_state_changed(gs_callbacks_t* cb, uintptr_t entity, const char* class_name, const char* field_name, int32_t offset) {
    if (cb && cb->schema_set_state_changed) {
        cb->schema_set_state_changed((void*)entity, class_name, field_name, offset);
    }
}

// Entity properties - accept uintptr_t to avoid Go unsafe.Pointer conversions
static inline int32_t call_entity_get_int(gs_callbacks_t* cb, uintptr_t entity, const char* class_name, const char* field_name) {
    if (cb && cb->entity_get_int) {
        return cb->entity_get_int((void*)entity, class_name, field_name);
    }
    return 0;
}

static inline void call_entity_set_int(gs_callbacks_t* cb, uintptr_t entity, const char* class_name, const char* field_name, int32_t value) {
    if (cb && cb->entity_set_int) {
        cb->entity_set_int((void*)entity, class_name, field_name, value);
    }
}

static inline float call_entity_get_float(gs_callbacks_t* cb, uintptr_t entity, const char* class_name, const char* field_name) {
    if (cb && cb->entity_get_float) {
        return cb->entity_get_float((void*)entity, class_name, field_name);
    }
    return 0.0f;
}

static inline void call_entity_set_float(gs_callbacks_t* cb, uintptr_t entity, const char* class_name, const char* field_name, float value) {
    if (cb && cb->entity_set_float) {
        cb->entity_set_float((void*)entity, class_name, field_name, value);
    }
}

static inline bool call_entity_get_bool(gs_callbacks_t* cb, uintptr_t entity, const char* class_name, const char* field_name) {
    if (cb && cb->entity_get_bool) {
        return cb->entity_get_bool((void*)entity, class_name, field_name);
    }
    return false;
}

static inline void call_entity_set_bool(gs_callbacks_t* cb, uintptr_t entity, const char* class_name, const char* field_name, bool value) {
    if (cb && cb->entity_set_bool) {
        cb->entity_set_bool((void*)entity, class_name, field_name, value);
    }
}

static inline int32_t call_entity_get_string(gs_callbacks_t* cb, uintptr_t entity, const char* class_name, const char* field_name, char* buf, int32_t buf_size) {
    if (cb && cb->entity_get_string) {
        return cb->entity_get_string((void*)entity, class_name, field_name, buf, buf_size);
    }
    return 0;
}

static inline void call_entity_get_vector(gs_callbacks_t* cb, uintptr_t entity, const char* class_name, const char* field_name, gs_vector3_t* out) {
    if (cb && cb->entity_get_vector) {
        cb->entity_get_vector((void*)entity, class_name, field_name, out);
    }
}

static inline void call_entity_set_vector(gs_callbacks_t* cb, uintptr_t entity, const char* class_name, const char* field_name, gs_vector3_t* value) {
    if (cb && cb->entity_set_vector) {
        cb->entity_set_vector((void*)entity, class_name, field_name, value);
    }
}

// Entity lookup - use uintptr_t to avoid Go unsafe.Pointer issues
static inline uintptr_t call_get_entity_by_index(gs_callbacks_t* cb, uint32_t index) {
    if (cb && cb->get_entity_by_index) {
        return (uintptr_t)cb->get_entity_by_index(index);
    }
    return 0;
}

static inline uint32_t call_get_entity_index(gs_callbacks_t* cb, uintptr_t entity) {
    if (cb && cb->get_entity_index) {
        return cb->get_entity_index((void*)entity);
    }
    return 0xFFFFFFFF;
}

static inline const char* call_get_entity_classname(gs_callbacks_t* cb, uintptr_t entity) {
    if (cb && cb->get_entity_classname) {
        return cb->get_entity_classname((void*)entity);
    }
    return NULL;
}

static inline bool call_is_entity_valid(gs_callbacks_t* cb, uintptr_t entity) {
    if (cb && cb->is_entity_valid) {
        return cb->is_entity_valid((void*)entity);
    }
    return false;
}

// GameData - return uintptr_t to avoid Go unsafe.Pointer issues
static inline uintptr_t call_resolve_gamedata(gs_callbacks_t* cb, const char* name) {
    if (cb && cb->resolve_gamedata) {
        return (uintptr_t)cb->resolve_gamedata(name);
    }
    return 0;
}

static inline int32_t call_get_gamedata_offset(gs_callbacks_t* cb, const char* name) {
    if (cb && cb->get_gamedata_offset) {
        return cb->get_gamedata_offset(name);
    }
    return -1;
}

// === V3 Callback Helpers (Phase 2: Core Game Integration) ===

// ConVar
static inline int32_t call_convar_get_int(gs_callbacks_t* cb, const char* name) {
    if (cb && cb->convar_get_int) { return cb->convar_get_int(name); }
    return 0;
}

static inline void call_convar_set_int(gs_callbacks_t* cb, const char* name, int32_t value) {
    if (cb && cb->convar_set_int) { cb->convar_set_int(name, value); }
}

static inline float call_convar_get_float(gs_callbacks_t* cb, const char* name) {
    if (cb && cb->convar_get_float) { return cb->convar_get_float(name); }
    return 0.0f;
}

static inline void call_convar_set_float(gs_callbacks_t* cb, const char* name, float value) {
    if (cb && cb->convar_set_float) { cb->convar_set_float(name, value); }
}

static inline int32_t call_convar_get_string(gs_callbacks_t* cb, const char* name, char* buf, int32_t buf_size) {
    if (cb && cb->convar_get_string) { return cb->convar_get_string(name, buf, buf_size); }
    return 0;
}

static inline void call_convar_set_string(gs_callbacks_t* cb, const char* name, const char* value) {
    if (cb && cb->convar_set_string) { cb->convar_set_string(name, value); }
}

// Player entities - return uintptr_t to avoid Go unsafe.Pointer issues
static inline uintptr_t call_get_player_controller(gs_callbacks_t* cb, int32_t slot) {
    if (cb && cb->get_player_controller) { return (uintptr_t)cb->get_player_controller(slot); }
    return 0;
}

static inline uintptr_t call_get_player_pawn(gs_callbacks_t* cb, int32_t slot) {
    if (cb && cb->get_player_pawn) { return (uintptr_t)cb->get_player_pawn(slot); }
    return 0;
}

// Game functions
static inline void call_player_respawn(gs_callbacks_t* cb, int32_t slot) {
    if (cb && cb->player_respawn) { cb->player_respawn(slot); }
}

static inline void call_player_change_team(gs_callbacks_t* cb, int32_t slot, int32_t team) {
    if (cb && cb->player_change_team) { cb->player_change_team(slot, team); }
}

static inline void call_player_slay(gs_callbacks_t* cb, int32_t slot) {
    if (cb && cb->player_slay) { cb->player_slay(slot); }
}

static inline void call_player_teleport(gs_callbacks_t* cb, int32_t slot, gs_vector3_t* pos, gs_vector3_t* angles, gs_vector3_t* velocity) {
    if (cb && cb->player_teleport) { cb->player_teleport(slot, pos, angles, velocity); }
}

static inline void call_entity_set_model(gs_callbacks_t* cb, uintptr_t entity, const char* model) {
    if (cb && cb->entity_set_model) { cb->entity_set_model((void*)entity, model); }
}

// === V4 Callback Helpers (Phase 3: Communication) ===

static inline void call_client_print(gs_callbacks_t* cb, int32_t slot, int32_t dest, const char* msg) {
    if (cb && cb->client_print) { cb->client_print(slot, dest, msg); }
}

static inline void call_client_print_all(gs_callbacks_t* cb, int32_t dest, const char* msg) {
    if (cb && cb->client_print_all) { cb->client_print_all(dest, msg); }
}

// === V5 Callback Helpers (Game Events + Weapons) ===

// Game event field access - event is opaque IGameEvent*
static inline int32_t call_event_get_int(gs_callbacks_t* cb, uintptr_t event, const char* key) {
    if (cb && cb->event_get_int) { return cb->event_get_int((void*)event, key); }
    return 0;
}

static inline float call_event_get_float(gs_callbacks_t* cb, uintptr_t event, const char* key) {
    if (cb && cb->event_get_float) { return cb->event_get_float((void*)event, key); }
    return 0.0f;
}

static inline bool call_event_get_bool(gs_callbacks_t* cb, uintptr_t event, const char* key) {
    if (cb && cb->event_get_bool) { return cb->event_get_bool((void*)event, key); }
    return false;
}

static inline int32_t call_event_get_string(gs_callbacks_t* cb, uintptr_t event, const char* key, char* buf, int32_t buf_size) {
    if (cb && cb->event_get_string) { return cb->event_get_string((void*)event, key, buf, buf_size); }
    return 0;
}

static inline uint64_t call_event_get_uint64(gs_callbacks_t* cb, uintptr_t event, const char* key) {
    if (cb && cb->event_get_uint64) { return cb->event_get_uint64((void*)event, key); }
    return 0;
}

static inline void call_event_set_int(gs_callbacks_t* cb, uintptr_t event, const char* key, int32_t value) {
    if (cb && cb->event_set_int) { cb->event_set_int((void*)event, key, value); }
}

static inline void call_event_set_float(gs_callbacks_t* cb, uintptr_t event, const char* key, float value) {
    if (cb && cb->event_set_float) { cb->event_set_float((void*)event, key, value); }
}

static inline void call_event_set_bool(gs_callbacks_t* cb, uintptr_t event, const char* key, bool value) {
    if (cb && cb->event_set_bool) { cb->event_set_bool((void*)event, key, value); }
}

static inline void call_event_set_string(gs_callbacks_t* cb, uintptr_t event, const char* key, const char* value) {
    if (cb && cb->event_set_string) { cb->event_set_string((void*)event, key, value); }
}

// Weapon management
static inline void call_give_named_item(gs_callbacks_t* cb, int32_t slot, const char* item_name) {
    if (cb && cb->give_named_item) { cb->give_named_item(slot, item_name); }
}

static inline void call_player_drop_weapons(gs_callbacks_t* cb, int32_t slot) {
    if (cb && cb->player_drop_weapons) { cb->player_drop_weapons(slot); }
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

// ============================================================
// V2: Schema System
// ============================================================

// SchemaGetOffset returns the byte offset of a class field.
// Returns (offset, isNetworked). Offset is 0 if not found.
func SchemaGetOffset(className, fieldName string) (int32, bool) {
	if callbacks == nil {
		return 0, false
	}

	cClass := C.CString(className)
	cField := C.CString(fieldName)
	defer C.free(unsafe.Pointer(cClass))
	defer C.free(unsafe.Pointer(cField))

	var networked C.bool
	offset := C.call_schema_get_offset(callbacks, cClass, cField, &networked)
	return int32(offset), bool(networked)
}

// SchemaSetStateChanged notifies the engine that a networked field changed
func SchemaSetStateChanged(entityPtr uintptr, className, fieldName string, offset int32) {
	if callbacks == nil {
		return
	}

	cClass := C.CString(className)
	cField := C.CString(fieldName)
	defer C.free(unsafe.Pointer(cClass))
	defer C.free(unsafe.Pointer(cField))

	C.call_schema_set_state_changed(callbacks, C.uintptr_t(entityPtr), cClass, cField, C.int32_t(offset))
}

// ============================================================
// V2: Entity Properties
// ============================================================

// EntityGetInt reads an int32 property from an entity via schema
func EntityGetInt(entityPtr uintptr, className, fieldName string) int32 {
	if callbacks == nil {
		return 0
	}

	cClass := C.CString(className)
	cField := C.CString(fieldName)
	defer C.free(unsafe.Pointer(cClass))
	defer C.free(unsafe.Pointer(cField))

	return int32(C.call_entity_get_int(callbacks, C.uintptr_t(entityPtr), cClass, cField))
}

// EntitySetInt writes an int32 property on an entity via schema
func EntitySetInt(entityPtr uintptr, className, fieldName string, value int32) {
	if callbacks == nil {
		return
	}

	cClass := C.CString(className)
	cField := C.CString(fieldName)
	defer C.free(unsafe.Pointer(cClass))
	defer C.free(unsafe.Pointer(cField))

	C.call_entity_set_int(callbacks, C.uintptr_t(entityPtr), cClass, cField, C.int32_t(value))
}

// EntityGetFloat reads a float property from an entity via schema
func EntityGetFloat(entityPtr uintptr, className, fieldName string) float32 {
	if callbacks == nil {
		return 0
	}

	cClass := C.CString(className)
	cField := C.CString(fieldName)
	defer C.free(unsafe.Pointer(cClass))
	defer C.free(unsafe.Pointer(cField))

	return float32(C.call_entity_get_float(callbacks, C.uintptr_t(entityPtr), cClass, cField))
}

// EntitySetFloat writes a float property on an entity via schema
func EntitySetFloat(entityPtr uintptr, className, fieldName string, value float32) {
	if callbacks == nil {
		return
	}

	cClass := C.CString(className)
	cField := C.CString(fieldName)
	defer C.free(unsafe.Pointer(cClass))
	defer C.free(unsafe.Pointer(cField))

	C.call_entity_set_float(callbacks, C.uintptr_t(entityPtr), cClass, cField, C.float(value))
}

// EntityGetBool reads a bool property from an entity via schema
func EntityGetBool(entityPtr uintptr, className, fieldName string) bool {
	if callbacks == nil {
		return false
	}

	cClass := C.CString(className)
	cField := C.CString(fieldName)
	defer C.free(unsafe.Pointer(cClass))
	defer C.free(unsafe.Pointer(cField))

	return bool(C.call_entity_get_bool(callbacks, C.uintptr_t(entityPtr), cClass, cField))
}

// EntitySetBool writes a bool property on an entity via schema
func EntitySetBool(entityPtr uintptr, className, fieldName string, value bool) {
	if callbacks == nil {
		return
	}

	cClass := C.CString(className)
	cField := C.CString(fieldName)
	defer C.free(unsafe.Pointer(cClass))
	defer C.free(unsafe.Pointer(cField))

	C.call_entity_set_bool(callbacks, C.uintptr_t(entityPtr), cClass, cField, C.bool(value))
}

// EntityGetString reads a string property from an entity via schema
func EntityGetString(entityPtr uintptr, className, fieldName string) string {
	if callbacks == nil {
		return ""
	}

	cClass := C.CString(className)
	cField := C.CString(fieldName)
	defer C.free(unsafe.Pointer(cClass))
	defer C.free(unsafe.Pointer(cField))

	var buf [1024]C.char
	length := C.call_entity_get_string(callbacks, C.uintptr_t(entityPtr), cClass, cField, &buf[0], 1024)
	if length <= 0 {
		return ""
	}
	return C.GoStringN(&buf[0], length)
}

// EntityGetVector reads a Vector3 property from an entity via schema
func EntityGetVector(entityPtr uintptr, className, fieldName string) (float32, float32, float32) {
	if callbacks == nil {
		return 0, 0, 0
	}

	cClass := C.CString(className)
	cField := C.CString(fieldName)
	defer C.free(unsafe.Pointer(cClass))
	defer C.free(unsafe.Pointer(cField))

	var vec C.gs_vector3_t
	C.call_entity_get_vector(callbacks, C.uintptr_t(entityPtr), cClass, cField, &vec)
	return float32(vec.x), float32(vec.y), float32(vec.z)
}

// EntitySetVector writes a Vector3 property on an entity via schema
func EntitySetVector(entityPtr uintptr, className, fieldName string, x, y, z float32) {
	if callbacks == nil {
		return
	}

	cClass := C.CString(className)
	cField := C.CString(fieldName)
	defer C.free(unsafe.Pointer(cClass))
	defer C.free(unsafe.Pointer(cField))

	vec := C.gs_vector3_t{x: C.float(x), y: C.float(y), z: C.float(z)}
	C.call_entity_set_vector(callbacks, C.uintptr_t(entityPtr), cClass, cField, &vec)
}

// ============================================================
// V2: Entity Lookup
// ============================================================

// GetEntityByIndex returns an opaque entity pointer by entity index.
// Returns 0 if entity not found.
func GetEntityByIndex(index uint32) uintptr {
	if callbacks == nil {
		return 0
	}
	return uintptr(C.call_get_entity_by_index(callbacks, C.uint32_t(index)))
}

// GetEntityIndex returns the entity index from an opaque entity pointer.
// Returns 0xFFFFFFFF if invalid.
func GetEntityIndex(entityPtr uintptr) uint32 {
	if callbacks == nil {
		return 0xFFFFFFFF
	}
	return uint32(C.call_get_entity_index(callbacks, C.uintptr_t(entityPtr)))
}

// GetEntityClassname returns the classname of an entity.
func GetEntityClassname(entityPtr uintptr) string {
	if callbacks == nil {
		return ""
	}
	cName := C.call_get_entity_classname(callbacks, C.uintptr_t(entityPtr))
	if cName == nil {
		return ""
	}
	return C.GoString(cName)
}

// IsEntityValid returns true if the entity pointer is valid.
func IsEntityValid(entityPtr uintptr) bool {
	if callbacks == nil {
		return false
	}
	return bool(C.call_is_entity_valid(callbacks, C.uintptr_t(entityPtr)))
}

// ============================================================
// V2: GameData
// ============================================================

// ResolveGamedata resolves a gamedata entry name to a memory address.
// Returns 0 if not found.
func ResolveGamedata(name string) uintptr {
	if callbacks == nil {
		return 0
	}

	cName := C.CString(name)
	defer C.free(unsafe.Pointer(cName))

	return uintptr(C.call_resolve_gamedata(callbacks, cName))
}

// GetGamedataOffset returns a gamedata offset by name.
// Returns -1 if not found.
func GetGamedataOffset(name string) int32 {
	if callbacks == nil {
		return -1
	}

	cName := C.CString(name)
	defer C.free(unsafe.Pointer(cName))

	return int32(C.call_get_gamedata_offset(callbacks, cName))
}

// ============================================================
// V3: ConVar System
// ============================================================

// ConVarGetInt reads an integer ConVar value
func ConVarGetInt(name string) int32 {
	if callbacks == nil {
		return 0
	}
	cName := C.CString(name)
	defer C.free(unsafe.Pointer(cName))
	return int32(C.call_convar_get_int(callbacks, cName))
}

// ConVarSetInt writes an integer ConVar value
func ConVarSetInt(name string, value int32) {
	if callbacks == nil {
		return
	}
	cName := C.CString(name)
	defer C.free(unsafe.Pointer(cName))
	C.call_convar_set_int(callbacks, cName, C.int32_t(value))
}

// ConVarGetFloat reads a float ConVar value
func ConVarGetFloat(name string) float32 {
	if callbacks == nil {
		return 0
	}
	cName := C.CString(name)
	defer C.free(unsafe.Pointer(cName))
	return float32(C.call_convar_get_float(callbacks, cName))
}

// ConVarSetFloat writes a float ConVar value
func ConVarSetFloat(name string, value float32) {
	if callbacks == nil {
		return
	}
	cName := C.CString(name)
	defer C.free(unsafe.Pointer(cName))
	C.call_convar_set_float(callbacks, cName, C.float(value))
}

// ConVarGetString reads a string ConVar value
func ConVarGetString(name string) string {
	if callbacks == nil {
		return ""
	}
	cName := C.CString(name)
	defer C.free(unsafe.Pointer(cName))

	var buf [1024]C.char
	length := C.call_convar_get_string(callbacks, cName, &buf[0], 1024)
	if length <= 0 {
		return ""
	}
	return C.GoStringN(&buf[0], length)
}

// ConVarSetString writes a string ConVar value
func ConVarSetString(name string, value string) {
	if callbacks == nil {
		return
	}
	cName := C.CString(name)
	cValue := C.CString(value)
	defer C.free(unsafe.Pointer(cName))
	defer C.free(unsafe.Pointer(cValue))
	C.call_convar_set_string(callbacks, cName, cValue)
}

// ============================================================
// V3: Player Pawn/Controller
// ============================================================

// GetPlayerController returns the CCSPlayerController entity pointer for a player slot.
// Returns 0 if not found.
func GetPlayerController(slot int) uintptr {
	if callbacks == nil {
		return 0
	}
	return uintptr(C.call_get_player_controller(callbacks, C.int32_t(slot)))
}

// GetPlayerPawn returns the CCSPlayerPawn entity pointer for a player slot.
// Returns 0 if not found (dead/spectating/disconnected).
func GetPlayerPawn(slot int) uintptr {
	if callbacks == nil {
		return 0
	}
	return uintptr(C.call_get_player_pawn(callbacks, C.int32_t(slot)))
}

// ============================================================
// V3: Game Functions
// ============================================================

// PlayerRespawn respawns a player
func PlayerRespawn(slot int) {
	if callbacks == nil {
		return
	}
	C.call_player_respawn(callbacks, C.int32_t(slot))
}

// PlayerChangeTeam changes a player's team
func PlayerChangeTeam(slot int, team int) {
	if callbacks == nil {
		return
	}
	C.call_player_change_team(callbacks, C.int32_t(slot), C.int32_t(team))
}

// PlayerSlay kills a player
func PlayerSlay(slot int) {
	if callbacks == nil {
		return
	}
	C.call_player_slay(callbacks, C.int32_t(slot))
}

// PlayerTeleport teleports a player
func PlayerTeleport(slot int, pos, angles, velocity *[3]float32) {
	if callbacks == nil {
		return
	}

	var cPos, cAngles, cVelocity *C.gs_vector3_t

	if pos != nil {
		p := C.gs_vector3_t{x: C.float(pos[0]), y: C.float(pos[1]), z: C.float(pos[2])}
		cPos = &p
	}
	if angles != nil {
		a := C.gs_vector3_t{x: C.float(angles[0]), y: C.float(angles[1]), z: C.float(angles[2])}
		cAngles = &a
	}
	if velocity != nil {
		v := C.gs_vector3_t{x: C.float(velocity[0]), y: C.float(velocity[1]), z: C.float(velocity[2])}
		cVelocity = &v
	}

	C.call_player_teleport(callbacks, C.int32_t(slot), cPos, cAngles, cVelocity)
}

// EntitySetModel sets the model on an entity
func EntitySetModel(entityPtr uintptr, model string) {
	if callbacks == nil {
		return
	}
	cModel := C.CString(model)
	defer C.free(unsafe.Pointer(cModel))
	C.call_entity_set_model(callbacks, C.uintptr_t(entityPtr), cModel)
}

// ============================================================
// V4: Communication (Proper In-Game Messaging)
// ============================================================

// Message destination constants
const (
	HudPrintNotify  = 1
	HudPrintConsole = 2
	HudPrintTalk    = 3
	HudPrintCenter  = 4
	HudPrintAlert   = 5
)

// ClientPrint sends a message to a specific player via engine UTIL_ClientPrint
func ClientPrint(slot int, dest int, message string) {
	if callbacks == nil {
		return
	}
	cMsg := C.CString(message)
	defer C.free(unsafe.Pointer(cMsg))
	C.call_client_print(callbacks, C.int32_t(slot), C.int32_t(dest), cMsg)
}

// ClientPrintAll sends a message to all players via engine UTIL_ClientPrintAll
func ClientPrintAll(dest int, message string) {
	if callbacks == nil {
		return
	}
	cMsg := C.CString(message)
	defer C.free(unsafe.Pointer(cMsg))
	C.call_client_print_all(callbacks, C.int32_t(dest), cMsg)
}

// ============================================================
// V5: Game Event Field Access
// ============================================================

// EventGetInt reads an int32 field from a native IGameEvent
func EventGetInt(eventPtr uintptr, key string) int32 {
	if callbacks == nil {
		return 0
	}
	cKey := C.CString(key)
	defer C.free(unsafe.Pointer(cKey))
	return int32(C.call_event_get_int(callbacks, C.uintptr_t(eventPtr), cKey))
}

// EventGetFloat reads a float field from a native IGameEvent
func EventGetFloat(eventPtr uintptr, key string) float32 {
	if callbacks == nil {
		return 0
	}
	cKey := C.CString(key)
	defer C.free(unsafe.Pointer(cKey))
	return float32(C.call_event_get_float(callbacks, C.uintptr_t(eventPtr), cKey))
}

// EventGetBool reads a bool field from a native IGameEvent
func EventGetBool(eventPtr uintptr, key string) bool {
	if callbacks == nil {
		return false
	}
	cKey := C.CString(key)
	defer C.free(unsafe.Pointer(cKey))
	return bool(C.call_event_get_bool(callbacks, C.uintptr_t(eventPtr), cKey))
}

// EventGetString reads a string field from a native IGameEvent
func EventGetString(eventPtr uintptr, key string) string {
	if callbacks == nil {
		return ""
	}
	cKey := C.CString(key)
	defer C.free(unsafe.Pointer(cKey))

	var buf [1024]C.char
	length := C.call_event_get_string(callbacks, C.uintptr_t(eventPtr), cKey, &buf[0], 1024)
	if length <= 0 {
		return ""
	}
	return C.GoStringN(&buf[0], length)
}

// EventGetUint64 reads a uint64 field from a native IGameEvent
func EventGetUint64(eventPtr uintptr, key string) uint64 {
	if callbacks == nil {
		return 0
	}
	cKey := C.CString(key)
	defer C.free(unsafe.Pointer(cKey))
	return uint64(C.call_event_get_uint64(callbacks, C.uintptr_t(eventPtr), cKey))
}

// EventSetInt writes an int32 field on a native IGameEvent (pre-hook only)
func EventSetInt(eventPtr uintptr, key string, value int32) {
	if callbacks == nil {
		return
	}
	cKey := C.CString(key)
	defer C.free(unsafe.Pointer(cKey))
	C.call_event_set_int(callbacks, C.uintptr_t(eventPtr), cKey, C.int32_t(value))
}

// EventSetFloat writes a float field on a native IGameEvent (pre-hook only)
func EventSetFloat(eventPtr uintptr, key string, value float32) {
	if callbacks == nil {
		return
	}
	cKey := C.CString(key)
	defer C.free(unsafe.Pointer(cKey))
	C.call_event_set_float(callbacks, C.uintptr_t(eventPtr), cKey, C.float(value))
}

// EventSetBool writes a bool field on a native IGameEvent (pre-hook only)
func EventSetBool(eventPtr uintptr, key string, value bool) {
	if callbacks == nil {
		return
	}
	cKey := C.CString(key)
	defer C.free(unsafe.Pointer(cKey))
	C.call_event_set_bool(callbacks, C.uintptr_t(eventPtr), cKey, C.bool(value))
}

// EventSetString writes a string field on a native IGameEvent (pre-hook only)
func EventSetString(eventPtr uintptr, key string, value string) {
	if callbacks == nil {
		return
	}
	cKey := C.CString(key)
	cVal := C.CString(value)
	defer C.free(unsafe.Pointer(cKey))
	defer C.free(unsafe.Pointer(cVal))
	C.call_event_set_string(callbacks, C.uintptr_t(eventPtr), cKey, cVal)
}

// ============================================================
// V5: Weapon Management
// ============================================================

// GiveNamedItem gives a weapon/item to a player by slot
func GiveNamedItem(slot int, itemName string) {
	if callbacks == nil {
		return
	}
	cName := C.CString(itemName)
	defer C.free(unsafe.Pointer(cName))
	C.call_give_named_item(callbacks, C.int32_t(slot), cName)
}

// PlayerDropWeapons drops all weapons for a player
func PlayerDropWeapons(slot int) {
	if callbacks == nil {
		return
	}
	C.call_player_drop_weapons(callbacks, C.int32_t(slot))
}
