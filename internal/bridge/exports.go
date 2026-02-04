// Package bridge provides the CGO bridge between the C++ native plugin and Go runtime.
// This file contains all functions exported to C++ via CGO.
package bridge

/*
#cgo CFLAGS: -I../../native/include
#include "gostrike_abi.h"
#include <stdlib.h>
#include <string.h>

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
	"runtime/debug"
	"sync"

	"github.com/corrreia/gostrike/internal/manager"
	"github.com/corrreia/gostrike/internal/runtime"
	"github.com/corrreia/gostrike/internal/shared"
)

// ============================================================
// Global State
// ============================================================

var (
	initialized bool
	initMu      sync.Mutex
	lastError   string
	lastErrorMu sync.Mutex
	callbacks   *C.gs_callbacks_t
)

// ============================================================
// Error Handling
// ============================================================

// setLastError stores an error message for later retrieval by C++
func setLastError(format string, args ...interface{}) {
	lastErrorMu.Lock()
	lastError = fmt.Sprintf(format, args...)
	lastErrorMu.Unlock()
}

// clearLastError clears the last error
func clearLastError() {
	lastErrorMu.Lock()
	lastError = ""
	lastErrorMu.Unlock()
}

// ============================================================
// Panic Recovery
// ============================================================

// safeCall wraps a function with panic recovery
// Returns GS_OK on success, GS_ERR_PANIC if a panic occurred
func safeCall(fn func()) (err C.gs_error_t) {
	defer func() {
		if r := recover(); r != nil {
			stack := string(debug.Stack())
			setLastError("panic: %v\n%s", r, stack)
			err = C.GS_ERR_PANIC

			// Log the panic
			logError("PANIC", fmt.Sprintf("Recovered from panic: %v", r))
		}
	}()
	fn()
	return C.GS_OK
}

// safeCallBool wraps a function returning bool with panic recovery
func safeCallBool(fn func() bool, defaultVal bool) bool {
	result := defaultVal
	defer func() {
		if r := recover(); r != nil {
			stack := string(debug.Stack())
			setLastError("panic: %v\n%s", r, stack)
			logError("PANIC", fmt.Sprintf("Recovered from panic: %v", r))
			result = defaultVal
		}
	}()
	result = fn()
	return result
}

// safeCallInt wraps a function returning int with panic recovery
func safeCallInt(fn func() C.gs_event_result_t, defaultVal C.gs_event_result_t) C.gs_event_result_t {
	result := defaultVal
	defer func() {
		if r := recover(); r != nil {
			stack := string(debug.Stack())
			setLastError("panic: %v\n%s", r, stack)
			logError("PANIC", fmt.Sprintf("Recovered from panic: %v", r))
			result = defaultVal
		}
	}()
	result = fn()
	return result
}

// ============================================================
// Logging Helpers
// ============================================================

func logDebug(tag, msg string) {
	Log(int(C.GS_LOG_DEBUG), tag, msg)
}

func logInfo(tag, msg string) {
	Log(int(C.GS_LOG_INFO), tag, msg)
}

func logWarning(tag, msg string) {
	Log(int(C.GS_LOG_WARNING), tag, msg)
}

func logError(tag, msg string) {
	Log(int(C.GS_LOG_ERROR), tag, msg)
}

// ============================================================
// Exported Functions (called by C++)
// ============================================================

//export GoStrike_Init
func GoStrike_Init() C.gs_error_t {
	initMu.Lock()
	defer initMu.Unlock()

	if initialized {
		return C.GS_OK
	}

	err := safeCall(func() {
		// Set up callback functions for other packages
		runtime.SetReplyFunc(ReplyToCommand)
		runtime.SetPanicLogger(func(context string, panicVal interface{}, stack string) {
			logError("PANIC", fmt.Sprintf("Panic in %s: %v\n%s", context, panicVal, stack))
		})
		manager.SetLogFunc(func(level int, tag, msg string) {
			Log(level, tag, msg)
		})

		// Initialize the runtime dispatcher
		runtime.Init()

		// Initialize the plugin manager
		manager.Init()

		initialized = true
	})

	if err != C.GS_OK {
		return err
	}

	logInfo("GoStrike", "Go runtime initialized successfully")
	return C.GS_OK
}

//export GoStrike_Shutdown
func GoStrike_Shutdown() {
	initMu.Lock()
	defer initMu.Unlock()

	if !initialized {
		return
	}

	_ = safeCall(func() {
		logInfo("GoStrike", "Shutting down Go runtime...")

		// Shutdown plugin manager first
		manager.Shutdown()

		// Shutdown runtime dispatcher
		runtime.Shutdown()

		initialized = false
	})

	logInfo("GoStrike", "Go runtime shutdown complete")
}

//export GoStrike_OnTick
func GoStrike_OnTick(deltaTime C.float) {
	if !initialized {
		return
	}

	_ = safeCall(func() {
		runtime.DispatchTick(float64(deltaTime))
	})
}

//export GoStrike_OnEvent
func GoStrike_OnEvent(event *C.gs_event_t, isPost C.bool) C.gs_event_result_t {
	if !initialized || event == nil {
		return C.GS_EVENT_CONTINUE
	}

	return safeCallInt(func() C.gs_event_result_t {
		eventName := C.GoStringN(event.name, C.int(event.name_len))
		result := runtime.DispatchEvent(eventName, uintptr(event.native_event), bool(isPost))
		return C.gs_event_result_t(result)
	}, C.GS_EVENT_CONTINUE)
}

//export GoStrike_OnCommand
func GoStrike_OnCommand(ctx *C.gs_command_ctx_t) C.bool {
	if !initialized || ctx == nil {
		return C.bool(false)
	}

	return C.bool(safeCallBool(func() bool {
		command := C.GoString(ctx.command)
		args := C.GoString(ctx.args)
		playerSlot := int(ctx.player_slot)

		return runtime.DispatchCommand(command, args, playerSlot)
	}, false))
}

//export GoStrike_OnPlayerConnect
func GoStrike_OnPlayerConnect(player *C.gs_player_t) {
	if !initialized || player == nil {
		return
	}

	_ = safeCall(func() {
		// Convert C player to Go player
		goPlayer := convertCPlayer(player)
		runtime.DispatchPlayerConnect((*runtime.PlayerInfo)(goPlayer))
	})
}

//export GoStrike_OnPlayerDisconnect
func GoStrike_OnPlayerDisconnect(slot C.int32_t, reason *C.char) {
	if !initialized {
		return
	}

	_ = safeCall(func() {
		goReason := ""
		if reason != nil {
			goReason = C.GoString(reason)
		}
		runtime.DispatchPlayerDisconnect(int(slot), goReason)
	})
}

//export GoStrike_OnMapChange
func GoStrike_OnMapChange(mapName *C.char) {
	if !initialized || mapName == nil {
		return
	}

	_ = safeCall(func() {
		goMapName := C.GoString(mapName)
		runtime.DispatchMapChange(goMapName)
	})
}

//export GoStrike_GetLastError
func GoStrike_GetLastError() *C.char {
	lastErrorMu.Lock()
	defer lastErrorMu.Unlock()

	if lastError == "" {
		return nil
	}

	// Caller must free this memory
	return C.CString(lastError)
}

//export GoStrike_ClearLastError
func GoStrike_ClearLastError() {
	clearLastError()
}

//export GoStrike_GetABIVersion
func GoStrike_GetABIVersion() C.int32_t {
	return C.GOSTRIKE_ABI_VERSION
}

//export GoStrike_RegisterCallbacks
func GoStrike_RegisterCallbacks(cb *C.gs_callbacks_t) {
	callbacks = cb
	logInfo("GoStrike", "C++ callbacks registered")
}

// ============================================================
// Helper Functions
// ============================================================

// convertCPlayer converts a C gs_player_t to a Go-friendly structure
func convertCPlayer(p *C.gs_player_t) *shared.PlayerInfo {
	if p == nil {
		return nil
	}

	player := &shared.PlayerInfo{
		Slot:    int(p.slot),
		UserID:  int(p.user_id),
		SteamID: uint64(p.steam_id),
		Team:    int(p.team),
		IsAlive: bool(p.is_alive),
		IsBot:   bool(p.is_bot),
		Health:  int(p.health),
		Armor:   int(p.armor),
		PosX:    float64(p.position.x),
		PosY:    float64(p.position.y),
		PosZ:    float64(p.position.z),
	}

	if p.name != nil {
		player.Name = C.GoString(p.name)
	}
	if p.ip != nil {
		player.IP = C.GoString(p.ip)
	}

	return player
}
