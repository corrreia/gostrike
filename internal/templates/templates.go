// Package templates provides server template management for GoStrike.
// Templates are named presets (plugin sets + convars + map) that can be
// applied at runtime to switch a server's purpose.
package templates

// Template represents a server configuration preset.
type Template struct {
	Name        string            `json:"name"`
	Description string            `json:"description,omitempty"`
	Extends     string            `json:"extends,omitempty"`
	Plugins     []string          `json:"plugins"`
	ConVars     map[string]string `json:"convars,omitempty"`
	Map         string            `json:"map,omitempty"`
}

// ResolvedTemplate is a fully-resolved template with inheritance merged.
type ResolvedTemplate struct {
	Name        string
	Description string
	Plugins     []string
	ConVars     map[string]string
	Map         string
	Chain       []string // inheritance chain for debugging
}
