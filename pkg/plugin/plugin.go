// Package plugin provides the plugin interface and registration for GoStrike.
package plugin

import (
	"github.com/corrreia/gostrike/internal/manager"
	"github.com/corrreia/gostrike/pkg/gostrike"
)

// Plugin is the interface all GoStrike plugins must implement
type Plugin interface {
	// Name returns the plugin's display name
	Name() string

	// Version returns the plugin's version string
	Version() string

	// Author returns the plugin author
	Author() string

	// Description returns a brief description
	Description() string

	// Load is called when the plugin is loaded
	// hotReload is true if this is a reload, not initial load
	Load(hotReload bool) error

	// Unload is called when the plugin is being unloaded
	// hotReload is true if plugin will be reloaded
	Unload(hotReload bool) error
}

// BasePlugin provides a default implementation of Plugin
// Embed this in your plugin struct to provide default implementations
type BasePlugin struct {
	logger gostrike.Logger
	config *gostrike.Config
}

// Name returns the plugin's display name
func (p *BasePlugin) Name() string { return "Unnamed Plugin" }

// Version returns the plugin's version string
func (p *BasePlugin) Version() string { return "0.0.0" }

// Author returns the plugin author
func (p *BasePlugin) Author() string { return "Unknown" }

// Description returns a brief description
func (p *BasePlugin) Description() string { return "" }

// Load is called when the plugin is loaded
func (p *BasePlugin) Load(hotReload bool) error { return nil }

// Unload is called when the plugin is being unloaded
func (p *BasePlugin) Unload(hotReload bool) error { return nil }

// GetLogger returns the plugin's logger
func (p *BasePlugin) GetLogger() gostrike.Logger {
	if p.logger == nil {
		p.logger = gostrike.GetLogger(p.Name())
	}
	return p.logger
}

// SetLogger sets the plugin's logger
func (p *BasePlugin) SetLogger(logger gostrike.Logger) {
	p.logger = logger
}

// GetConfig returns the plugin's configuration
func (p *BasePlugin) GetConfig() *gostrike.Config {
	return p.config
}

// SetConfig sets the plugin's configuration
func (p *BasePlugin) SetConfig(config *gostrike.Config) {
	p.config = config
}

// Register adds a plugin to the framework
// This should be called in the plugin's init() function
func Register(p Plugin) {
	manager.RegisterPlugin(p)
}

// RegisterFunc registers a plugin using a factory function
// The factory is called when the plugin needs to be instantiated
func RegisterFunc(factory func() Plugin) {
	manager.RegisterPluginFunc(func() interface{} {
		return factory()
	})
}
