// Package plugin provides the plugin interface and registration for GoStrike.
package plugin

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"

	"github.com/corrreia/gostrike/internal/manager"
	"github.com/corrreia/gostrike/pkg/gostrike"
)

// slugRegex validates plugin slugs: must start with letter, contain only letters/numbers/underscores, 2-32 chars
var slugRegex = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_]{1,31}$`)

// reservedSlugs are slugs that cannot be used by plugins
var reservedSlugs = []string{"core", "system", "admin", "api", "gostrike", "internal", "plugin", "plugins"}

// ValidateSlug checks if a slug is valid for use as a plugin identifier
func ValidateSlug(slug string) error {
	if slug == "" {
		return fmt.Errorf("slug cannot be empty")
	}
	if !slugRegex.MatchString(slug) {
		return fmt.Errorf("invalid slug format '%s': must start with letter, contain only letters/numbers/underscores, 2-32 chars", slug)
	}
	slugLower := strings.ToLower(slug)
	for _, reserved := range reservedSlugs {
		if slugLower == reserved {
			return fmt.Errorf("slug '%s' is reserved", slug)
		}
	}
	return nil
}

// SanitizeSlug converts a name to a valid slug
// It lowercases, replaces spaces/special chars with underscores, and removes invalid characters
func SanitizeSlug(name string) string {
	var result strings.Builder
	prevUnderscore := false

	for i, r := range name {
		switch {
		case unicode.IsLetter(r):
			result.WriteRune(unicode.ToLower(r))
			prevUnderscore = false
		case unicode.IsDigit(r) && i > 0:
			result.WriteRune(r)
			prevUnderscore = false
		case (r == ' ' || r == '-' || r == '_') && !prevUnderscore && result.Len() > 0:
			result.WriteRune('_')
			prevUnderscore = true
		}
	}

	// Trim trailing underscore
	s := result.String()
	s = strings.TrimSuffix(s, "_")

	// Ensure minimum length
	if len(s) < 2 {
		s = s + "_plugin"
	}

	// Truncate if too long
	if len(s) > 32 {
		s = s[:32]
	}

	return s
}

// Plugin is the interface all GoStrike plugins must implement
type Plugin interface {
	// Slug returns the plugin's unique identifier
	// Must match: ^[a-zA-Z][a-zA-Z0-9_]{1,31}$
	// Used for namespacing HTTP routes, database, and resources
	Slug() string

	// Name returns the plugin's display name
	Name() string

	// Version returns the plugin's version string
	Version() string

	// Author returns the plugin author
	Author() string

	// Description returns a brief description
	Description() string

	// DefaultConfig returns the default configuration for the plugin
	// Return nil if the plugin does not need a config file
	// Config files are auto-generated at configs/plugins/[slug].json
	DefaultConfig() map[string]interface{}

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
	slug   string // cached slug
}

// Slug returns a sanitized version of the plugin name as the default slug
// Override this method in your plugin to provide a custom slug
func (p *BasePlugin) Slug() string {
	if p.slug == "" {
		p.slug = SanitizeSlug(p.Name())
	}
	return p.slug
}

// Name returns the plugin's display name
func (p *BasePlugin) Name() string { return "Unnamed Plugin" }

// Version returns the plugin's version string
func (p *BasePlugin) Version() string { return "0.0.0" }

// Author returns the plugin author
func (p *BasePlugin) Author() string { return "Unknown" }

// Description returns a brief description
func (p *BasePlugin) Description() string { return "" }

// DefaultConfig returns nil by default (no config file needed)
// Override this method to provide default configuration values
func (p *BasePlugin) DefaultConfig() map[string]interface{} { return nil }

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
