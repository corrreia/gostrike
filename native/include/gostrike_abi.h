// gostrike_abi.h - Stable C ABI between C++ and Go
// This header defines the interface between the native Metamod plugin and the Go runtime.
// Both sides must use identical definitions for all types and functions.

#ifndef GOSTRIKE_ABI_H
#define GOSTRIKE_ABI_H

#include <stdint.h>
#include <stdbool.h>

#ifdef __cplusplus
extern "C" {
#endif

// ============================================================
// Version and Constants
// ============================================================

// Version for ABI compatibility checks
// Increment this when making breaking changes to the ABI
#define GOSTRIKE_ABI_VERSION 1

// GoStrike version string
#define GOSTRIKE_VERSION "0.1.0"

// Maximum string lengths for safety
#define GS_MAX_NAME_LEN 128
#define GS_MAX_PATH_LEN 512
#define GS_MAX_CMD_LEN 512
#define GS_MAX_MSG_LEN 1024

// ============================================================
// Error Codes
// ============================================================

typedef enum {
    GS_OK = 0,
    GS_ERR_INIT_FAILED = -1,
    GS_ERR_PANIC = -2,
    GS_ERR_NOT_FOUND = -3,
    GS_ERR_INVALID_ARG = -4,
    GS_ERR_ALREADY_EXISTS = -5,
    GS_ERR_NOT_INITIALIZED = -6,
} gs_error_t;

// ============================================================
// Result Types
// ============================================================

// Result type for operations that can fail
typedef struct {
    gs_error_t code;
    char* error_message;  // NULL if no error, caller must free
} gs_result_t;

// String with explicit length (for binary safety)
typedef struct {
    const char* data;
    uint32_t    len;
} gs_string_t;

// ============================================================
// Game Data Types
// ============================================================

// Team identifiers
typedef enum {
    GS_TEAM_UNASSIGNED = 0,
    GS_TEAM_SPECTATOR = 1,
    GS_TEAM_T = 2,
    GS_TEAM_CT = 3,
} gs_team_t;

// 3D Vector
typedef struct {
    float x;
    float y;
    float z;
} gs_vector3_t;

// Player information passed to Go
typedef struct {
    int32_t     slot;       // Player slot index (0-63)
    int32_t     user_id;    // Unique ID for this session
    uint64_t    steam_id;   // Steam ID (64-bit)
    char*       name;       // Player name, UTF-8, null-terminated
    char*       ip;         // IP address, null-terminated
    int32_t     team;       // Team (gs_team_t)
    bool        is_alive;   // Is the player alive
    bool        is_bot;     // Is this a bot
    int32_t     health;     // Current health
    int32_t     armor;      // Current armor
    gs_vector3_t position;  // World position
} gs_player_t;

// Event data passed to Go
typedef struct {
    const char* name;           // Event name (null-terminated)
    uint32_t    name_len;       // Length of event name
    void*       native_event;   // Opaque pointer to IGameEvent
    bool        can_modify;     // true for pre-hooks
} gs_event_t;

// ============================================================
// Event Results
// ============================================================

// Event result from Go handler
typedef enum {
    GS_EVENT_CONTINUE = 0,  // Allow event to proceed normally
    GS_EVENT_CHANGED = 1,   // Event data was modified
    GS_EVENT_HANDLED = 2,   // Stop processing, but allow event
    GS_EVENT_STOP = 3,      // Cancel the event entirely
} gs_event_result_t;

// ============================================================
// Log Levels
// ============================================================

typedef enum {
    GS_LOG_DEBUG = 0,
    GS_LOG_INFO = 1,
    GS_LOG_WARNING = 2,
    GS_LOG_ERROR = 3,
} gs_log_level_t;

// ============================================================
// Functions EXPORTED by Go (called by C++)
// ============================================================

// Initialize the Go runtime. Must be called once at plugin load.
// Returns GS_OK on success, error code on failure.
gs_error_t GoStrike_Init(void);

// Shutdown the Go runtime. Called at plugin unload.
void GoStrike_Shutdown(void);

// Called every server frame/tick
// delta_time: Time since last tick in seconds
void GoStrike_OnTick(float delta_time);

// Dispatch a game event to Go handlers
// event: Pointer to event data (valid only for duration of call)
// is_post: true if this is a post-hook (after engine processing)
// Returns the combined result from all handlers
gs_event_result_t GoStrike_OnEvent(gs_event_t* event, bool is_post);

// Called when a player connects
void GoStrike_OnPlayerConnect(gs_player_t* player);

// Called when a player disconnects
// Note: reason is non-const because Go CGO exports don't support const
void GoStrike_OnPlayerDisconnect(int32_t slot, char* reason);

// Called when the map changes
// Note: map_name is non-const because Go CGO exports don't support const
void GoStrike_OnMapChange(char* map_name);

// Called when a player sends a chat message
// Returns true if the message was a command and should be suppressed
// message: The chat message text
bool GoStrike_OnChatMessage(int32_t player_slot, char* message);

// === V2: Entity lifecycle events (called by C++ entity listener) ===
void GoStrike_OnEntityCreated(uint32_t index, char* classname);
void GoStrike_OnEntitySpawned(uint32_t index, char* classname);
void GoStrike_OnEntityDeleted(uint32_t index);

// Get the last error message (for debugging)
// Returns NULL if no error. Caller must free the returned string.
char* GoStrike_GetLastError(void);

// Clear the last error
void GoStrike_ClearLastError(void);

// Check ABI version compatibility
int32_t GoStrike_GetABIVersion(void);

// ============================================================
// Callback Function Types (C++ implementations called by Go)
// ============================================================

// Logging callback
typedef void (*gs_log_callback_t)(int level, const char* tag, const char* msg);

// Execute a server command
typedef void (*gs_exec_command_t)(const char* cmd);

// Send a reply to a command invoker
// slot: -1 for server console, >= 0 for player slot
typedef void (*gs_reply_callback_t)(int32_t slot, const char* msg);

// Get player information by slot
// Returns NULL if player doesn't exist. Memory owned by C++, valid until next call.
typedef gs_player_t* (*gs_get_player_t)(int32_t slot);

// Get the number of connected players
typedef int32_t (*gs_get_player_count_t)(void);

// Get all player slots (returns array of slot indices)
// out_slots: Array to fill with player slots (must be at least 64 elements)
// Returns the number of players
typedef int32_t (*gs_get_all_players_t)(int32_t* out_slots);

// Kick a player from the server
typedef void (*gs_kick_player_t)(int32_t slot, const char* reason);

// Get the current map name
// Returns pointer to static buffer, valid until map change
typedef const char* (*gs_get_map_name_t)(void);

// Get the maximum number of players
typedef int32_t (*gs_get_max_players_t)(void);

// Get the server tick rate
typedef int32_t (*gs_get_tick_rate_t)(void);

// Send a chat message to a player
// slot: -1 for all players, >= 0 for specific player
typedef void (*gs_send_chat_t)(int32_t slot, const char* msg);

// Send a center message to a player
typedef void (*gs_send_center_t)(int32_t slot, const char* msg);

// ============================================================
// V2 Callback Types (Phase 1: Foundation)
// ============================================================

// Schema: get field offset for a class member
// Returns offset in bytes, or 0 if not found. Sets is_networked output.
typedef int32_t (*gs_schema_get_offset_t)(const char* class_name, const char* field_name, bool* is_networked);

// Schema: notify engine that a networked field changed
typedef void (*gs_schema_set_state_changed_t)(void* entity, const char* class_name, const char* field_name, int32_t offset);

// Entity property read/write (entity_ptr is opaque, never dereferenced by Go)
typedef int32_t (*gs_entity_get_int_t)(void* entity, const char* class_name, const char* field_name);
typedef void (*gs_entity_set_int_t)(void* entity, const char* class_name, const char* field_name, int32_t value);
typedef float (*gs_entity_get_float_t)(void* entity, const char* class_name, const char* field_name);
typedef void (*gs_entity_set_float_t)(void* entity, const char* class_name, const char* field_name, float value);
typedef bool (*gs_entity_get_bool_t)(void* entity, const char* class_name, const char* field_name);
typedef void (*gs_entity_set_bool_t)(void* entity, const char* class_name, const char* field_name, bool value);
typedef int32_t (*gs_entity_get_string_t)(void* entity, const char* class_name, const char* field_name, char* buf, int32_t buf_size);
typedef void (*gs_entity_get_vector_t)(void* entity, const char* class_name, const char* field_name, gs_vector3_t* out);
typedef void (*gs_entity_set_vector_t)(void* entity, const char* class_name, const char* field_name, gs_vector3_t* value);

// Entity lookup
typedef void* (*gs_get_entity_by_index_t)(uint32_t index);
typedef uint32_t (*gs_get_entity_index_t)(void* entity);
typedef const char* (*gs_get_entity_classname_t)(void* entity);
typedef bool (*gs_is_entity_valid_t)(void* entity);

// GameData: resolve a signature name to an address
typedef void* (*gs_resolve_gamedata_t)(const char* name);
// GameData: get an offset by name
typedef int32_t (*gs_get_gamedata_offset_t)(const char* name);

// ============================================================
// Callback Registry
// ============================================================

// Callback registry passed to Go at init
typedef struct {
    // === V1 (existing) ===
    // Logging
    gs_log_callback_t       log;

    // Commands
    gs_exec_command_t       exec_command;
    gs_reply_callback_t     reply_to_command;

    // Players
    gs_get_player_t         get_player;
    gs_get_player_count_t   get_player_count;
    gs_get_all_players_t    get_all_players;
    gs_kick_player_t        kick_player;

    // Server info
    gs_get_map_name_t       get_map_name;
    gs_get_max_players_t    get_max_players;
    gs_get_tick_rate_t      get_tick_rate;

    // Messaging
    gs_send_chat_t          send_chat;
    gs_send_center_t        send_center;

    // === V2 (Phase 1: Foundation) ===
    // Schema
    gs_schema_get_offset_t          schema_get_offset;
    gs_schema_set_state_changed_t   schema_set_state_changed;

    // Entity properties
    gs_entity_get_int_t     entity_get_int;
    gs_entity_set_int_t     entity_set_int;
    gs_entity_get_float_t   entity_get_float;
    gs_entity_set_float_t   entity_set_float;
    gs_entity_get_bool_t    entity_get_bool;
    gs_entity_set_bool_t    entity_set_bool;
    gs_entity_get_string_t  entity_get_string;
    gs_entity_get_vector_t  entity_get_vector;
    gs_entity_set_vector_t  entity_set_vector;

    // Entity lookup
    gs_get_entity_by_index_t    get_entity_by_index;
    gs_get_entity_index_t       get_entity_index;
    gs_get_entity_classname_t   get_entity_classname;
    gs_is_entity_valid_t        is_entity_valid;

    // GameData
    gs_resolve_gamedata_t       resolve_gamedata;
    gs_get_gamedata_offset_t    get_gamedata_offset;
} gs_callbacks_t;

// Register callbacks from C++ to Go
// Must be called immediately after GoStrike_Init
void GoStrike_RegisterCallbacks(gs_callbacks_t* callbacks);

// ============================================================
// Memory Ownership Rules
// ============================================================
//
// 1. Strings passed FROM C++ TO Go:
//    - C++ owns the memory
//    - Go must copy if it needs to retain the string
//    - Valid only for the duration of the call
//
// 2. Strings passed FROM Go TO C++:
//    - Allocated with malloc() by Go
//    - C++ must call free() when done
//    - Includes: error messages, GoStrike_GetLastError result
//
// 3. Structs (gs_player_t, gs_event_t, etc.):
//    - Passed by pointer, owned by caller
//    - Valid only for duration of the call
//    - Nested strings follow rule #1
//
// 4. Opaque handles (void*):
//    - Must not be dereferenced by the other side
//    - Lifetime managed by the creating side
//
// 5. Arrays:
//    - Caller allocates and owns the array
//    - Callee fills in the data
//    - Size must be communicated separately

#ifdef __cplusplus
}
#endif

#endif // GOSTRIKE_ABI_H
