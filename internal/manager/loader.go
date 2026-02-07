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

// SortPluginsByLoadOrder sorts plugin entries by load order and dependency graph.
// It first groups by LoadOrder (early/normal/late), then within each group
// performs a topological sort based on declared dependencies.
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

	// Topological sort within each group
	early = topologicalSort(early)
	normal = topologicalSort(normal)
	late = topologicalSort(late)

	// Rebuild plugins slice
	plugins = make([]*pluginEntry, 0, len(early)+len(normal)+len(late))
	plugins = append(plugins, early...)
	plugins = append(plugins, normal...)
	plugins = append(plugins, late...)
}

// topologicalSort sorts plugin entries respecting dependency order.
// Plugins with dependencies load after their dependencies.
// If a cycle is detected or a dependency is missing, plugins load in original order.
func topologicalSort(entries []*pluginEntry) []*pluginEntry {
	if len(entries) <= 1 {
		return entries
	}

	// Build name -> entry index map
	nameToIdx := make(map[string]int)
	for i, entry := range entries {
		nameToIdx[entry.info.Name] = i
	}

	// Build adjacency list (dependency -> dependents)
	// and in-degree count
	n := len(entries)
	inDegree := make([]int, n)
	adj := make([][]int, n)

	for i, entry := range entries {
		if entry.plugin == nil {
			continue
		}
		dep, ok := entry.plugin.(DependentPlugin)
		if !ok {
			continue
		}
		for _, d := range dep.Dependencies() {
			if d.Optional {
				continue // Optional deps don't affect ordering
			}
			depIdx, exists := nameToIdx[d.Name]
			if !exists {
				continue // Dependency not in this group, skip
			}
			// depIdx must load before i
			adj[depIdx] = append(adj[depIdx], i)
			inDegree[i]++
		}
	}

	// Kahn's algorithm
	var queue []int
	for i := 0; i < n; i++ {
		if inDegree[i] == 0 {
			queue = append(queue, i)
		}
	}

	var sorted []*pluginEntry
	for len(queue) > 0 {
		idx := queue[0]
		queue = queue[1:]
		sorted = append(sorted, entries[idx])
		for _, next := range adj[idx] {
			inDegree[next]--
			if inDegree[next] == 0 {
				queue = append(queue, next)
			}
		}
	}

	// If not all entries were sorted, there's a cycle - return original order
	if len(sorted) != n {
		return entries
	}

	return sorted
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
