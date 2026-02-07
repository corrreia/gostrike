package ipc

import (
	"fmt"
	"sync"
)

type serviceEntry struct {
	name    string
	slug    string // owning plugin slug
	service interface{}
}

var (
	servicesMu sync.RWMutex
	services   = make(map[string]*serviceEntry)
)

// RegisterService registers a named service. Returns an error if already registered.
func RegisterService(name string, service interface{}) error {
	return RegisterServiceForPlugin("", name, service)
}

// RegisterServiceForPlugin registers a named service with an owning plugin slug.
func RegisterServiceForPlugin(slug, name string, service interface{}) error {
	servicesMu.Lock()
	defer servicesMu.Unlock()

	if _, exists := services[name]; exists {
		return fmt.Errorf("service '%s' is already registered", name)
	}

	services[name] = &serviceEntry{name: name, slug: slug, service: service}
	return nil
}

// UnregisterService removes a named service.
func UnregisterService(name string) {
	servicesMu.Lock()
	defer servicesMu.Unlock()
	delete(services, name)
}

// GetService returns a registered service, or nil if not found.
func GetService(name string) interface{} {
	servicesMu.RLock()
	defer servicesMu.RUnlock()

	if e, ok := services[name]; ok {
		return e.service
	}
	return nil
}

// UnregisterAllForPlugin removes all services owned by a plugin slug.
func UnregisterAllForPlugin(slug string) {
	servicesMu.Lock()
	defer servicesMu.Unlock()

	for name, e := range services {
		if e.slug == slug {
			delete(services, name)
		}
	}
}
