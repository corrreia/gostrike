// Package scope provides a tracker interface for plugin resource registration.
// It breaks the import cycle between pkg/gostrike and internal/manager.
package scope

import "sync"

// Tracker is implemented by PluginScope and records every resource a plugin
// registers so they can be cleaned up on unload.
type Tracker interface {
	TrackChatCommand(name string)
	TrackHandler(id uint64)
	TrackTimer(id uint64)
	TrackHTTPRoute(method, path string)
	TrackPermission(name string)
	TrackEventSub(id uint64)
	TrackService(name string)
}

var (
	mu     sync.Mutex
	active Tracker

	registryMu sync.RWMutex
	registry   = make(map[string]Tracker) // slug -> tracker
)

// SetActive sets the currently-active scope tracker.
// Called by the manager before invoking plugin.Load().
func SetActive(t Tracker) {
	mu.Lock()
	defer mu.Unlock()
	active = t
}

// ClearActive clears the active scope (called via defer after Load).
func ClearActive() {
	mu.Lock()
	defer mu.Unlock()
	active = nil
}

// GetActive returns the currently-active scope tracker, or nil.
func GetActive() Tracker {
	mu.Lock()
	defer mu.Unlock()
	return active
}

// Register adds a named tracker to the global registry.
func Register(slug string, t Tracker) {
	registryMu.Lock()
	defer registryMu.Unlock()
	registry[slug] = t
}

// Get returns a tracker by slug.
func Get(slug string) Tracker {
	registryMu.RLock()
	defer registryMu.RUnlock()
	return registry[slug]
}

// Unregister removes a tracker from the global registry.
func Unregister(slug string) {
	registryMu.Lock()
	defer registryMu.Unlock()
	delete(registry, slug)
}
