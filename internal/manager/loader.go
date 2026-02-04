// Package manager provides plugin lifecycle management for GoStrike.
// This file contains plugin loading utilities.
package manager

// LoadOrder represents plugin load order preferences
type LoadOrder int

const (
	LoadOrderNormal LoadOrder = iota
	LoadOrderEarly
	LoadOrderLate
)

// PluginDependency represents a plugin dependency
type PluginDependency struct {
	Name     string
	Version  string
	Optional bool
}

// DependentPlugin interface for plugins that declare dependencies
type DependentPlugin interface {
	Dependencies() []PluginDependency
}

// OrderedPlugin interface for plugins that specify load order
type OrderedPlugin interface {
	LoadOrder() LoadOrder
}

// ValidateDependencies checks if all required dependencies are loaded
func ValidateDependencies(plugin PluginInterface) []string {
	dep, ok := plugin.(DependentPlugin)
	if !ok {
		return nil // No dependencies declared
	}

	var missing []string
	for _, d := range dep.Dependencies() {
		if d.Optional {
			continue
		}

		info := GetPlugin(d.Name)
		if info == nil || info.State != PluginStateLoaded {
			missing = append(missing, d.Name)
		}
	}

	return missing
}

// GetLoadOrder returns the load order for a plugin
func GetLoadOrder(plugin PluginInterface) LoadOrder {
	ordered, ok := plugin.(OrderedPlugin)
	if !ok {
		return LoadOrderNormal
	}
	return ordered.LoadOrder()
}

// SortPluginsByLoadOrder sorts plugin entries by their load order
// This is a simple stable sort that groups by load order
func SortPluginsByLoadOrder() {
	pluginsMu.Lock()
	defer pluginsMu.Unlock()

	// Simple grouping: early, normal, late
	var early, normal, late []*pluginEntry

	for _, entry := range plugins {
		if entry.plugin == nil {
			normal = append(normal, entry)
			continue
		}

		switch GetLoadOrder(entry.plugin) {
		case LoadOrderEarly:
			early = append(early, entry)
		case LoadOrderLate:
			late = append(late, entry)
		default:
			normal = append(normal, entry)
		}
	}

	// Rebuild plugins slice
	plugins = make([]*pluginEntry, 0, len(early)+len(normal)+len(late))
	plugins = append(plugins, early...)
	plugins = append(plugins, normal...)
	plugins = append(plugins, late...)
}

// GetPluginByName returns a plugin instance by name (for inter-plugin communication)
func GetPluginByName(name string) PluginInterface {
	pluginsMu.RLock()
	defer pluginsMu.RUnlock()

	for _, entry := range plugins {
		if entry.info.Name == name && entry.info.State == PluginStateLoaded {
			return entry.plugin
		}
	}
	return nil
}
