// Package runtime provides the internal runtime for GoStrike.
// This file contains version information and plugin utilities.
package runtime

import (
	"fmt"
)

// Version information (set at build time or here)
const (
	Version    = "0.1.0"
	ABIVersion = 1
)

// PluginListItem represents a plugin in the list
type PluginListItem struct {
	Name        string
	Version     string
	Author      string
	Description string
	State       string
	Error       string
}

// Plugin list functions - these will be set by the manager package
var (
	getPluginListFunc   func() []PluginListItem
	getPluginByNameFunc func(name string) *PluginListItem
	reloadPluginFunc    func(name string) error
)

// SetPluginListFunc sets the function to get plugin list
func SetPluginListFunc(fn func() []PluginListItem) {
	getPluginListFunc = fn
}

// SetPluginByNameFunc sets the function to get plugin by name
func SetPluginByNameFunc(fn func(name string) *PluginListItem) {
	getPluginByNameFunc = fn
}

// SetReloadPluginFunc sets the function to reload a plugin
func SetReloadPluginFunc(fn func(name string) error) {
	reloadPluginFunc = fn
}

// GetPluginList returns the list of loaded plugins
func GetPluginList() []PluginListItem {
	if getPluginListFunc != nil {
		return getPluginListFunc()
	}
	return nil
}

// GetPluginByName returns a plugin by name
func GetPluginByName(name string) *PluginListItem {
	if getPluginByNameFunc != nil {
		return getPluginByNameFunc(name)
	}
	return nil
}

// ReloadPlugin reloads a plugin by name
func ReloadPlugin(name string) error {
	if reloadPluginFunc != nil {
		return reloadPluginFunc(name)
	}
	return fmt.Errorf("reload not available")
}
