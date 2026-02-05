// Package modules provides the core module system for GoStrike.
// Modules are built-in features like permissions, HTTP server, and database.
package modules

import (
	"fmt"
	"sync"

	"github.com/corrreia/gostrike/internal/shared"
)

// ModuleState represents the state of a module
type ModuleState int

const (
	ModuleStateUnloaded ModuleState = iota
	ModuleStateLoading
	ModuleStateLoaded
	ModuleStateUnloading
	ModuleStateFailed
)

// String returns the string representation of the module state
func (s ModuleState) String() string {
	switch s {
	case ModuleStateUnloaded:
		return "Unloaded"
	case ModuleStateLoading:
		return "Loading"
	case ModuleStateLoaded:
		return "Loaded"
	case ModuleStateUnloading:
		return "Unloading"
	case ModuleStateFailed:
		return "Failed"
	default:
		return "Unknown"
	}
}

// Module is the interface that core modules must implement
type Module interface {
	// Name returns the module's display name
	Name() string

	// Version returns the module's version string
	Version() string

	// Priority returns the module's load priority (lower loads first)
	// Default modules should use 100, plugins use 1000+
	Priority() int

	// Init initializes the module
	Init() error

	// Shutdown shuts down the module
	Shutdown() error
}

// ModuleWithConfig is an optional interface for modules that have configuration
type ModuleWithConfig interface {
	Module
	// Configure is called with the module's configuration section
	Configure(config map[string]interface{}) error
}

// moduleEntry holds a registered module
type moduleEntry struct {
	module Module
	state  ModuleState
	err    error
}

var (
	registeredModules []*moduleEntry
	modulesMu         sync.RWMutex
	initialized       bool
	initMu            sync.Mutex
)

func logInfo(msg string) {
	shared.LogInfo("Modules", msg)
}

func logError(msg string) {
	shared.LogError("Modules", msg)
}

// Register registers a core module
// This should be called during init() of module packages
func Register(m Module) {
	modulesMu.Lock()
	defer modulesMu.Unlock()

	entry := &moduleEntry{
		module: m,
		state:  ModuleStateUnloaded,
	}

	// Insert in priority order
	inserted := false
	for i, e := range registeredModules {
		if m.Priority() < e.module.Priority() {
			registeredModules = append(registeredModules[:i], append([]*moduleEntry{entry}, registeredModules[i:]...)...)
			inserted = true
			break
		}
	}
	if !inserted {
		registeredModules = append(registeredModules, entry)
	}
}

// Init initializes all registered modules
func Init() error {
	shared.DebugLog("[GoStrike-Debug-Modules-Core] Init() called")
	initMu.Lock()
	defer initMu.Unlock()
	shared.DebugLog("[GoStrike-Debug-Modules-Core] Acquired initMu")

	if initialized {
		shared.DebugLog("[GoStrike-Debug-Modules-Core] Already initialized")
		return nil
	}

	logInfo("Initializing modules...")

	shared.DebugLog("[GoStrike-Debug-Modules-Core] Acquiring modulesMu...")
	modulesMu.Lock()
	defer modulesMu.Unlock()
	shared.DebugLog("[GoStrike-Debug-Modules-Core] modulesMu acquired, %d modules registered", len(registeredModules))

	for _, entry := range registeredModules {
		if entry.state == ModuleStateLoaded {
			continue
		}

		entry.state = ModuleStateLoading
		logInfo(fmt.Sprintf("Loading module: %s v%s", entry.module.Name(), entry.module.Version()))

		// Initialize module with panic recovery
		func() {
			defer func() {
				if r := recover(); r != nil {
					entry.state = ModuleStateFailed
					entry.err = fmt.Errorf("panic during init: %v", r)
					logError(fmt.Sprintf("Module %s panicked: %v", entry.module.Name(), r))
				}
			}()

			if err := entry.module.Init(); err != nil {
				entry.state = ModuleStateFailed
				entry.err = err
				logError(fmt.Sprintf("Module %s failed to load: %v", entry.module.Name(), err))
				return
			}

			entry.state = ModuleStateLoaded
			entry.err = nil

			logInfo(fmt.Sprintf("Module %s loaded successfully", entry.module.Name()))
		}()
	}

	initialized = true
	// Count loaded modules inline to avoid deadlock (we already hold modulesMu)
	loadedCount := 0
	for _, entry := range registeredModules {
		if entry.state == ModuleStateLoaded {
			loadedCount++
		}
	}
	logInfo(fmt.Sprintf("Modules initialized: %d loaded", loadedCount))
	shared.DebugLog("[GoStrike-Debug-Modules-Core] Init() returning nil")
	return nil
}

// Shutdown shuts down all modules in reverse order
func Shutdown() {
	initMu.Lock()
	defer initMu.Unlock()

	if !initialized {
		return
	}

	logInfo("Shutting down modules...")

	modulesMu.Lock()
	defer modulesMu.Unlock()

	for i := len(registeredModules) - 1; i >= 0; i-- {
		entry := registeredModules[i]
		if entry.state != ModuleStateLoaded {
			continue
		}

		entry.state = ModuleStateUnloading
		logInfo(fmt.Sprintf("Shutting down module: %s", entry.module.Name()))

		// Shutdown module with panic recovery
		func() {
			defer func() {
				if r := recover(); r != nil {
					logError(fmt.Sprintf("Module %s panicked during shutdown: %v", entry.module.Name(), r))
				}
			}()

			if err := entry.module.Shutdown(); err != nil {
				logError(fmt.Sprintf("Module %s failed to shutdown cleanly: %v", entry.module.Name(), err))
			}
		}()

		entry.state = ModuleStateUnloaded
	}

	initialized = false
	logInfo("Modules shutdown complete")
}

// Get returns a module by name if it's loaded
func Get(name string) Module {
	modulesMu.RLock()
	defer modulesMu.RUnlock()

	for _, entry := range registeredModules {
		if entry.module.Name() == name && entry.state == ModuleStateLoaded {
			return entry.module
		}
	}
	return nil
}

// GetAll returns information about all registered modules
func GetAll() []ModuleInfo {
	modulesMu.RLock()
	defer modulesMu.RUnlock()

	result := make([]ModuleInfo, len(registeredModules))
	for i, entry := range registeredModules {
		result[i] = ModuleInfo{
			Name:     entry.module.Name(),
			Version:  entry.module.Version(),
			Priority: entry.module.Priority(),
			State:    entry.state.String(),
		}
		if entry.err != nil {
			result[i].Error = entry.err.Error()
		}
	}
	return result
}

// GetLoadedCount returns the number of loaded modules
func GetLoadedCount() int {
	modulesMu.RLock()
	defer modulesMu.RUnlock()

	count := 0
	for _, entry := range registeredModules {
		if entry.state == ModuleStateLoaded {
			count++
		}
	}
	return count
}

// ModuleInfo contains module metadata for external use
type ModuleInfo struct {
	Name     string
	Version  string
	Priority int
	State    string
	Error    string
}

// IsInitialized returns true if modules have been initialized
func IsInitialized() bool {
	initMu.Lock()
	defer initMu.Unlock()
	return initialized
}
