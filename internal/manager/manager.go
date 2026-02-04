// Package manager provides plugin lifecycle management for GoStrike.
package manager

import (
	"fmt"
	"sync"

	"github.com/corrreia/gostrike/internal/shared"
)

// PluginState represents the current state of a plugin
type PluginState int

const (
	PluginStateUnloaded PluginState = iota
	PluginStateLoading
	PluginStateLoaded
	PluginStateUnloading
	PluginStateFailed
)

// String returns the state name
func (s PluginState) String() string {
	switch s {
	case PluginStateUnloaded:
		return "Unloaded"
	case PluginStateLoading:
		return "Loading"
	case PluginStateLoaded:
		return "Loaded"
	case PluginStateUnloading:
		return "Unloading"
	case PluginStateFailed:
		return "Failed"
	default:
		return "Unknown"
	}
}

// PluginInfo contains plugin metadata
type PluginInfo struct {
	Name        string
	Version     string
	Author      string
	Description string
	State       PluginState
	LoadError   error
}

// PluginInterface is the interface that plugins must implement
type PluginInterface interface {
	Name() string
	Version() string
	Author() string
	Description() string
	Load(hotReload bool) error
	Unload(hotReload bool) error
}

// pluginEntry holds a registered plugin
type pluginEntry struct {
	plugin  PluginInterface
	factory func() interface{}
	info    PluginInfo
}

var (
	plugins     []*pluginEntry
	pluginsMu   sync.RWMutex
	initialized bool
	initMu      sync.Mutex
	logFunc     func(level int, tag, msg string)
)

func init() {
	// Register with shared package
	shared.ManagerInit = Init
	shared.ManagerShutdown = Shutdown
}

// SetLogFunc sets the logging function (called by bridge)
func SetLogFunc(fn func(level int, tag, msg string)) {
	logFunc = fn
}

func logInfo(tag, msg string) {
	if logFunc != nil {
		logFunc(1, tag, msg)
	}
}

func logError(tag, msg string) {
	if logFunc != nil {
		logFunc(3, tag, msg)
	}
}

// Init initializes the plugin manager
func Init() {
	initMu.Lock()
	defer initMu.Unlock()

	if initialized {
		return
	}

	logInfo("PluginManager", "Initializing plugin manager...")

	// Load all registered plugins
	loadAllPlugins(false)

	initialized = true
	logInfo("PluginManager", fmt.Sprintf("Plugin manager initialized with %d plugins", len(plugins)))
}

// Shutdown shuts down the plugin manager
func Shutdown() {
	initMu.Lock()
	defer initMu.Unlock()

	if !initialized {
		return
	}

	logInfo("PluginManager", "Shutting down plugin manager...")

	// Unload all plugins in reverse order
	unloadAllPlugins(false)

	initialized = false
	logInfo("PluginManager", "Plugin manager shutdown complete")
}

// RegisterPlugin registers a plugin instance
func RegisterPlugin(p interface{}) {
	pluginsMu.Lock()
	defer pluginsMu.Unlock()

	plugin, ok := p.(PluginInterface)
	if !ok {
		logError("PluginManager", "Invalid plugin type: does not implement PluginInterface")
		return
	}

	entry := &pluginEntry{
		plugin:  plugin,
		factory: nil,
		info: PluginInfo{
			Name:        plugin.Name(),
			Version:     plugin.Version(),
			Author:      plugin.Author(),
			Description: plugin.Description(),
			State:       PluginStateUnloaded,
		},
	}

	plugins = append(plugins, entry)
	logInfo("PluginManager", fmt.Sprintf("Registered plugin: %s v%s by %s",
		entry.info.Name, entry.info.Version, entry.info.Author))
}

// RegisterPluginFunc registers a plugin factory function
func RegisterPluginFunc(factory func() interface{}) {
	pluginsMu.Lock()
	defer pluginsMu.Unlock()

	entry := &pluginEntry{
		plugin:  nil,
		factory: factory,
		info: PluginInfo{
			Name:  "Unknown",
			State: PluginStateUnloaded,
		},
	}

	plugins = append(plugins, entry)
}

// loadAllPlugins loads all registered plugins
func loadAllPlugins(hotReload bool) {
	pluginsMu.Lock()
	defer pluginsMu.Unlock()

	for _, entry := range plugins {
		loadPluginEntry(entry, hotReload)
	}
}

// unloadAllPlugins unloads all plugins in reverse order
func unloadAllPlugins(hotReload bool) {
	pluginsMu.Lock()
	defer pluginsMu.Unlock()

	for i := len(plugins) - 1; i >= 0; i-- {
		unloadPluginEntry(plugins[i], hotReload)
	}
}

// loadPluginEntry loads a single plugin
func loadPluginEntry(entry *pluginEntry, hotReload bool) {
	if entry.info.State == PluginStateLoaded {
		return
	}

	entry.info.State = PluginStateLoading

	// If we have a factory, use it to create the plugin
	if entry.plugin == nil && entry.factory != nil {
		p := entry.factory()
		plugin, ok := p.(PluginInterface)
		if !ok {
			entry.info.State = PluginStateFailed
			entry.info.LoadError = fmt.Errorf("factory did not return a valid plugin")
			logError("PluginManager", "Plugin factory failed: invalid type")
			return
		}
		entry.plugin = plugin
		entry.info.Name = plugin.Name()
		entry.info.Version = plugin.Version()
		entry.info.Author = plugin.Author()
		entry.info.Description = plugin.Description()
	}

	if entry.plugin == nil {
		entry.info.State = PluginStateFailed
		entry.info.LoadError = fmt.Errorf("no plugin instance")
		return
	}

	logInfo("PluginManager", fmt.Sprintf("Loading plugin: %s v%s", entry.info.Name, entry.info.Version))

	// Call plugin's Load method with panic recovery
	func() {
		defer func() {
			if r := recover(); r != nil {
				entry.info.State = PluginStateFailed
				entry.info.LoadError = fmt.Errorf("panic during load: %v", r)
				logError("PluginManager", fmt.Sprintf("Plugin %s panicked during load: %v",
					entry.info.Name, r))
			}
		}()

		if err := entry.plugin.Load(hotReload); err != nil {
			entry.info.State = PluginStateFailed
			entry.info.LoadError = err
			logError("PluginManager", fmt.Sprintf("Plugin %s failed to load: %v",
				entry.info.Name, err))
			return
		}

		entry.info.State = PluginStateLoaded
		entry.info.LoadError = nil
		logInfo("PluginManager", fmt.Sprintf("Plugin %s loaded successfully", entry.info.Name))
	}()
}

// unloadPluginEntry unloads a single plugin
func unloadPluginEntry(entry *pluginEntry, hotReload bool) {
	if entry.info.State != PluginStateLoaded {
		return
	}

	if entry.plugin == nil {
		return
	}

	entry.info.State = PluginStateUnloading
	logInfo("PluginManager", fmt.Sprintf("Unloading plugin: %s", entry.info.Name))

	// Call plugin's Unload method with panic recovery
	func() {
		defer func() {
			if r := recover(); r != nil {
				logError("PluginManager", fmt.Sprintf("Plugin %s panicked during unload: %v",
					entry.info.Name, r))
			}
		}()

		if err := entry.plugin.Unload(hotReload); err != nil {
			logError("PluginManager", fmt.Sprintf("Plugin %s failed to unload cleanly: %v",
				entry.info.Name, err))
		}
	}()

	entry.info.State = PluginStateUnloaded
	logInfo("PluginManager", fmt.Sprintf("Plugin %s unloaded", entry.info.Name))
}

// GetPlugins returns information about all registered plugins
func GetPlugins() []PluginInfo {
	pluginsMu.RLock()
	defer pluginsMu.RUnlock()

	result := make([]PluginInfo, len(plugins))
	for i, entry := range plugins {
		result[i] = entry.info
	}
	return result
}

// GetPlugin returns information about a specific plugin by name
func GetPlugin(name string) *PluginInfo {
	pluginsMu.RLock()
	defer pluginsMu.RUnlock()

	for _, entry := range plugins {
		if entry.info.Name == name {
			info := entry.info
			return &info
		}
	}
	return nil
}

// ReloadPlugin unloads and reloads a specific plugin
func ReloadPlugin(name string) error {
	pluginsMu.Lock()
	defer pluginsMu.Unlock()

	for _, entry := range plugins {
		if entry.info.Name == name {
			unloadPluginEntry(entry, true)
			loadPluginEntry(entry, true)
			return nil
		}
	}

	return fmt.Errorf("plugin not found: %s", name)
}

// GetLoadedCount returns the number of successfully loaded plugins
func GetLoadedCount() int {
	pluginsMu.RLock()
	defer pluginsMu.RUnlock()

	count := 0
	for _, entry := range plugins {
		if entry.info.State == PluginStateLoaded {
			count++
		}
	}
	return count
}
