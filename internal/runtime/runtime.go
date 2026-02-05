// Package runtime provides the internal runtime for GoStrike,
// handling event dispatching, command routing, and timer management.
package runtime

import (
	"sync"

	"github.com/corrreia/gostrike/internal/shared"
)

// PlayerInfo contains player information
type PlayerInfo = shared.PlayerInfo

var (
	initialized bool
	initMu      sync.Mutex
)

func init() {
	// Register dispatch functions with shared package
	shared.RuntimeInit = Init
	shared.RuntimeShutdown = Shutdown
	shared.DispatchTick = DispatchTick
	shared.DispatchEvent = dispatchEventWrapper
	shared.DispatchPlayerConnect = DispatchPlayerConnect
	shared.DispatchPlayerDisconnect = DispatchPlayerDisconnect
	shared.DispatchMapChange = DispatchMapChange
}

// dispatchEventWrapper wraps the internal dispatch function
func dispatchEventWrapper(eventName string, nativeEvent uintptr, isPost bool) int {
	return DispatchEvent(eventName, nativeEvent, isPost)
}

// Init initializes the runtime
func Init() {
	shared.DebugLog("[GoStrike-Debug-Runtime] Init() called")
	initMu.Lock()
	defer initMu.Unlock()
	shared.DebugLog("[GoStrike-Debug-Runtime] Acquired init mutex")

	if initialized {
		shared.DebugLog("[GoStrike-Debug-Runtime] Already initialized")
		return
	}

	// Initialize subsystems
	shared.DebugLog("[GoStrike-Debug-Runtime] initTimers...")
	initTimers()
	shared.DebugLog("[GoStrike-Debug-Runtime] initChatCommands...")
	initChatCommands()
	shared.DebugLog("[GoStrike-Debug-Runtime] initEvents...")
	initEvents()
	shared.DebugLog("[GoStrike-Debug-Runtime] initModules...")
	initModules()
	shared.DebugLog("[GoStrike-Debug-Runtime] All init functions completed")

	initialized = true
}

// Shutdown shuts down the runtime
func Shutdown() {
	initMu.Lock()
	defer initMu.Unlock()

	if !initialized {
		return
	}

	// Shutdown subsystems in reverse order
	shutdownModules()
	shutdownTimers()
	shutdownChatCommands()
	shutdownEvents()

	initialized = false
}

// IsInitialized returns true if the runtime is initialized
func IsInitialized() bool {
	initMu.Lock()
	defer initMu.Unlock()
	return initialized
}
