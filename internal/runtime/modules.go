// Package runtime provides the internal runtime for GoStrike.
// This file provides module integration for the runtime.
package runtime

import (
	"github.com/corrreia/gostrike/internal/modules"
	"github.com/corrreia/gostrike/internal/shared"
)

// Re-export types from modules package for convenience
type (
	// Module is the interface that core modules must implement
	Module = modules.Module
	// ModuleInfo contains module metadata
	ModuleInfo = modules.ModuleInfo
	// ModuleState represents the state of a module
	ModuleState = modules.ModuleState
)

// Re-export constants
const (
	ModuleStateUnloaded  = modules.ModuleStateUnloaded
	ModuleStateLoading   = modules.ModuleStateLoading
	ModuleStateLoaded    = modules.ModuleStateLoaded
	ModuleStateUnloading = modules.ModuleStateUnloading
	ModuleStateFailed    = modules.ModuleStateFailed
)

// RegisterModule registers a core module
func RegisterModule(m Module) {
	modules.Register(m)
}

// initModules initializes all registered modules
func initModules() {
	shared.DebugLog("[GoStrike-Debug-Modules] initModules() called")

	// Initialize modules (will do nothing if no modules registered)
	// Modules must be explicitly registered via RegisterModule()
	shared.DebugLog("[GoStrike-Debug-Modules] Calling modules.Init()...")
	if err := modules.Init(); err != nil {
		shared.DebugLog("[GoStrike-Debug-Modules] modules.Init() error: %v", err)
		// Log error but don't fail - modules are optional
	}
	shared.DebugLog("[GoStrike-Debug-Modules] modules.Init() completed")
}

// shutdownModules shuts down all modules in reverse order
func shutdownModules() {
	modules.Shutdown()
}

// GetModules returns information about all registered modules
func GetModules() []ModuleInfo {
	return modules.GetAll()
}

// GetModule returns a specific module by name
func GetModule(name string) Module {
	return modules.Get(name)
}
