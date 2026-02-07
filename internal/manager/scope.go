package manager

import (
	"sync"

	"github.com/corrreia/gostrike/internal/scope"
)

// routeKey identifies an HTTP route for cleanup.
type routeKey struct {
	Method string
	Path   string
}

// ScopeRemovers holds function pointers for cleaning up tracked resources.
type ScopeRemovers struct {
	UnregisterChatCommand func(name string)
	UnregisterHandler     func(id uint64)
	StopTimer             func(id uint64)
	RemoveHTTPRoute       func(method, path string)
	UnregisterPermission  func(name string)
	UnsubscribeEvent      func(id uint64)
	UnregisterService     func(name string)
}

// PluginScope implements scope.Tracker and accumulates all resources
// registered by a plugin during its Load() call and runtime.
type PluginScope struct {
	mu           sync.Mutex
	slug         string
	chatCommands []string
	handlerIDs   []uint64
	timerIDs     []uint64
	httpRoutes   []routeKey
	permissions  []string
	eventSubs    []uint64
	services     []string
}

// Compile-time check.
var _ scope.Tracker = (*PluginScope)(nil)

func newPluginScope(slug string) *PluginScope {
	return &PluginScope{slug: slug}
}

func (s *PluginScope) TrackChatCommand(name string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.chatCommands = append(s.chatCommands, name)
}

func (s *PluginScope) TrackHandler(id uint64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.handlerIDs = append(s.handlerIDs, id)
}

func (s *PluginScope) TrackTimer(id uint64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.timerIDs = append(s.timerIDs, id)
}

func (s *PluginScope) TrackHTTPRoute(method, path string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.httpRoutes = append(s.httpRoutes, routeKey{Method: method, Path: path})
}

func (s *PluginScope) TrackPermission(name string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.permissions = append(s.permissions, name)
}

func (s *PluginScope) TrackEventSub(id uint64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.eventSubs = append(s.eventSubs, id)
}

func (s *PluginScope) TrackService(name string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.services = append(s.services, name)
}

// Cleanup iterates all tracked resources and removes them using the provided removers.
func (s *PluginScope) Cleanup(r ScopeRemovers) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if r.UnregisterChatCommand != nil {
		for _, name := range s.chatCommands {
			r.UnregisterChatCommand(name)
		}
	}
	if r.UnregisterHandler != nil {
		for _, id := range s.handlerIDs {
			r.UnregisterHandler(id)
		}
	}
	if r.StopTimer != nil {
		for _, id := range s.timerIDs {
			r.StopTimer(id)
		}
	}
	if r.RemoveHTTPRoute != nil {
		for _, rk := range s.httpRoutes {
			r.RemoveHTTPRoute(rk.Method, rk.Path)
		}
	}
	if r.UnregisterPermission != nil {
		for _, name := range s.permissions {
			r.UnregisterPermission(name)
		}
	}
	if r.UnsubscribeEvent != nil {
		for _, id := range s.eventSubs {
			r.UnsubscribeEvent(id)
		}
	}
	if r.UnregisterService != nil {
		for _, name := range s.services {
			r.UnregisterService(name)
		}
	}

	// Clear all slices
	s.chatCommands = nil
	s.handlerIDs = nil
	s.timerIDs = nil
	s.httpRoutes = nil
	s.permissions = nil
	s.eventSubs = nil
	s.services = nil
}
