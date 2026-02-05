// Package manager provides plugin lifecycle management for GoStrike.
package manager

import (
	"encoding/json"
	"fmt"
	"os"
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
	PluginStateDisabled // Plugin is disabled via config
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
	case PluginStateDisabled:
		return "Disabled"
	default:
		return "Unknown"
	}
}

// PluginsConfig represents the plugins configuration file
type PluginsConfig struct {
	Plugins       map[string]PluginConfigEntry `json:"plugins"`
	AutoEnableNew bool                         `json:"auto_enable_new"`
}

// PluginConfigEntry represents a single plugin's configuration
type PluginConfigEntry struct {
	Enabled bool                   `json:"enabled"`
	Config  map[string]interface{} `json:"config"`
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
	plugins       []*pluginEntry
	pluginsMu     sync.RWMutex
	initialized   bool
	initMu        sync.Mutex
	pluginsConfig *PluginsConfig
	configPath    = "configs/plugins.json"
)

func init() {
	// Register with shared package
	shared.ManagerInit = Init
	shared.ManagerShutdown = Shutdown
}

// loadPluginsConfig loads the plugins configuration
func loadPluginsConfig() {
	data, err := os.ReadFile(configPath)
	if err != nil {
		// Config doesn't exist, use defaults (enable all)
		pluginsConfig = &PluginsConfig{
			Plugins:       make(map[string]PluginConfigEntry),
			AutoEnableNew: true,
		}
		return
	}

	var config PluginsConfig
	if err := json.Unmarshal(data, &config); err != nil {
		logError("PluginManager", fmt.Sprintf("Failed to parse plugins config: %v", err))
		pluginsConfig = &PluginsConfig{
			Plugins:       make(map[string]PluginConfigEntry),
			AutoEnableNew: true,
		}
		return
	}

	pluginsConfig = &config
}

// isPluginEnabled checks if a plugin is enabled in the config
func isPluginEnabled(name string) bool {
	if pluginsConfig == nil {
		return true // No config = enable all
	}

	entry, ok := pluginsConfig.Plugins[name]
	if !ok {
		// Plugin not in config, use auto_enable_new setting
		return pluginsConfig.AutoEnableNew
	}

	return entry.Enabled
}

// GetPluginConfig returns the config for a specific plugin
func GetPluginConfig(name string) map[string]interface{} {
	if pluginsConfig == nil {
		return nil
	}

	entry, ok := pluginsConfig.Plugins[name]
	if !ok {
		return nil
	}

	return entry.Config
}

// SetPluginEnabled sets whether a plugin is enabled (does not persist)
func SetPluginEnabled(name string, enabled bool) {
	if pluginsConfig == nil {
		pluginsConfig = &PluginsConfig{
			Plugins:       make(map[string]PluginConfigEntry),
			AutoEnableNew: true,
		}
	}

	entry := pluginsConfig.Plugins[name]
	entry.Enabled = enabled
	pluginsConfig.Plugins[name] = entry
}

func logInfo(tag, msg string) {
	shared.LogInfo(tag, msg)
}

func logError(tag, msg string) {
	shared.LogError(tag, msg)
}

// Init initializes the plugin manager
func Init() {
	shared.DebugLog("[GoStrike-Debug-Manager] Init() called")
	initMu.Lock()
	defer initMu.Unlock()
	shared.DebugLog("[GoStrike-Debug-Manager] Acquired initMu")

	if initialized {
		shared.DebugLog("[GoStrike-Debug-Manager] Already initialized")
		return
	}

	logInfo("PluginManager", "Initializing plugin manager...")

	// Load plugins configuration
	shared.DebugLog("[GoStrike-Debug-Manager] Calling loadPluginsConfig()...")
	loadPluginsConfig()
	shared.DebugLog("[GoStrike-Debug-Manager] loadPluginsConfig() done")

	// Load all registered plugins
	shared.DebugLog("[GoStrike-Debug-Manager] Calling loadAllPlugins(), %d plugins registered...", len(plugins))
	loadAllPlugins(false)
	shared.DebugLog("[GoStrike-Debug-Manager] loadAllPlugins() done")

	initialized = true
	logInfo("PluginManager", fmt.Sprintf("Plugin manager initialized with %d plugins", len(plugins)))
	shared.DebugLog("[GoStrike-Debug-Manager] Init() completed")
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
	shared.DebugLog("[GoStrike-Debug-Manager] loadAllPlugins() acquiring pluginsMu...")
	pluginsMu.Lock()
	defer pluginsMu.Unlock()
	shared.DebugLog("[GoStrike-Debug-Manager] loadAllPlugins() pluginsMu acquired")

	for i, entry := range plugins {
		shared.DebugLog("[GoStrike-Debug-Manager] Loading plugin %d: %s", i, entry.info.Name)
		loadPluginEntry(entry, hotReload)
		shared.DebugLog("[GoStrike-Debug-Manager] Plugin %d loaded", i)
	}
	shared.DebugLog("[GoStrike-Debug-Manager] loadAllPlugins() all plugins loaded")
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
	shared.DebugLog("[GoStrike-Debug-Manager] loadPluginEntry() for %s, state=%d", entry.info.Name, entry.info.State)
	if entry.info.State == PluginStateLoaded {
		shared.DebugLog("[GoStrike-Debug-Manager] Plugin already loaded, skipping")
		return
	}

	// If we have a factory, use it to create the plugin first to get the name
	if entry.plugin == nil && entry.factory != nil {
		shared.DebugLog("[GoStrike-Debug-Manager] Calling factory function...")
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
		shared.DebugLog("[GoStrike-Debug-Manager] Factory created plugin: %s", entry.info.Name)
	}

	if entry.plugin == nil {
		entry.info.State = PluginStateFailed
		entry.info.LoadError = fmt.Errorf("no plugin instance")
		shared.DebugLog("[GoStrike-Debug-Manager] No plugin instance")
		return
	}

	// Check if plugin is enabled in config
	shared.DebugLog("[GoStrike-Debug-Manager] Checking if %s is enabled...", entry.info.Name)
	if !isPluginEnabled(entry.info.Name) {
		entry.info.State = PluginStateDisabled
		logInfo("PluginManager", fmt.Sprintf("Plugin %s is disabled in config", entry.info.Name))
		shared.DebugLog("[GoStrike-Debug-Manager] Plugin is disabled")
		return
	}
	shared.DebugLog("[GoStrike-Debug-Manager] Plugin is enabled")

	entry.info.State = PluginStateLoading
	logInfo("PluginManager", fmt.Sprintf("Loading plugin: %s v%s", entry.info.Name, entry.info.Version))

	// Call plugin's Load method with panic recovery
	shared.DebugLog("[GoStrike-Debug-Manager] Calling plugin.Load() for %s...", entry.info.Name)
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
			shared.DebugLog("[GoStrike-Debug-Manager] Plugin %s Load() failed: %v", entry.info.Name, err)
			return
		}
		shared.DebugLog("[GoStrike-Debug-Manager] Plugin %s Load() succeeded", entry.info.Name)

		entry.info.State = PluginStateLoaded
		entry.info.LoadError = nil
		logInfo("PluginManager", fmt.Sprintf("Plugin %s loaded successfully", entry.info.Name))
	}()
	shared.DebugLog("[GoStrike-Debug-Manager] loadPluginEntry() completed for %s", entry.info.Name)
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

// PluginListItem is a simplified plugin info for the runtime
type PluginListItem struct {
	Name        string
	Version     string
	Author      string
	Description string
	State       string
	Error       string
}

// GetPluginList returns a list of all plugins for the runtime
func GetPluginList() []PluginListItem {
	pluginsMu.RLock()
	defer pluginsMu.RUnlock()

	result := make([]PluginListItem, len(plugins))
	for i, entry := range plugins {
		result[i] = PluginListItem{
			Name:        entry.info.Name,
			Version:     entry.info.Version,
			Author:      entry.info.Author,
			Description: entry.info.Description,
			State:       entry.info.State.String(),
		}
		if entry.info.LoadError != nil {
			result[i].Error = entry.info.LoadError.Error()
		}
	}
	return result
}

// GetPluginListItemByName returns a specific plugin by name for the runtime gs command
func GetPluginListItemByName(name string) *PluginListItem {
	pluginsMu.RLock()
	defer pluginsMu.RUnlock()

	for _, entry := range plugins {
		if entry.info.Name == name {
			item := PluginListItem{
				Name:        entry.info.Name,
				Version:     entry.info.Version,
				Author:      entry.info.Author,
				Description: entry.info.Description,
				State:       entry.info.State.String(),
			}
			if entry.info.LoadError != nil {
				item.Error = entry.info.LoadError.Error()
			}
			return &item
		}
	}
	return nil
}
