// Package http provides an embedded HTTP server module for GoStrike.
// This file contains the HTTP router implementation.
package http

import (
	"net/http"
	"strings"
	"sync"
)

// Route represents a registered route
type Route struct {
	Method  string
	Path    string
	Handler http.HandlerFunc
}

// RouteInfo represents route information for listing
type RouteInfo struct {
	Method string `json:"method"`
	Path   string `json:"path"`
}

// Router is a simple HTTP router
type Router struct {
	mu          sync.RWMutex
	routes      map[string]map[string]http.HandlerFunc // method -> path -> handler
	notFound    http.HandlerFunc
	middlewares []func(http.Handler) http.Handler
}

// NewRouter creates a new router
func NewRouter() *Router {
	return &Router{
		routes: make(map[string]map[string]http.HandlerFunc),
		notFound: func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(`{"error": "not found"}`))
		},
	}
}

// ServeHTTP implements http.Handler
func (router *Router) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	router.mu.RLock()
	methodRoutes, ok := router.routes[r.Method]
	if !ok {
		router.mu.RUnlock()
		router.notFound(w, r)
		return
	}

	// Try exact match first
	handler, ok := methodRoutes[r.URL.Path]
	if ok {
		router.mu.RUnlock()
		handler(w, r)
		return
	}

	// Try pattern matching
	for pattern, h := range methodRoutes {
		if matchPath(pattern, r.URL.Path) {
			router.mu.RUnlock()
			h(w, r)
			return
		}
	}
	router.mu.RUnlock()

	router.notFound(w, r)
}

// HandleFunc registers a handler for a method and path
func (router *Router) HandleFunc(method, path string, handler http.HandlerFunc) {
	router.mu.Lock()
	defer router.mu.Unlock()

	method = strings.ToUpper(method)
	if router.routes[method] == nil {
		router.routes[method] = make(map[string]http.HandlerFunc)
	}
	router.routes[method][path] = handler
}

// Handle registers a handler for a method and path
func (router *Router) Handle(method, path string, handler http.Handler) {
	router.HandleFunc(method, path, handler.ServeHTTP)
}

// GET registers a GET handler
func (router *Router) GET(path string, handler http.HandlerFunc) {
	router.HandleFunc("GET", path, handler)
}

// POST registers a POST handler
func (router *Router) POST(path string, handler http.HandlerFunc) {
	router.HandleFunc("POST", path, handler)
}

// PUT registers a PUT handler
func (router *Router) PUT(path string, handler http.HandlerFunc) {
	router.HandleFunc("PUT", path, handler)
}

// DELETE registers a DELETE handler
func (router *Router) DELETE(path string, handler http.HandlerFunc) {
	router.HandleFunc("DELETE", path, handler)
}

// PATCH registers a PATCH handler
func (router *Router) PATCH(path string, handler http.HandlerFunc) {
	router.HandleFunc("PATCH", path, handler)
}

// SetNotFound sets the not found handler
func (router *Router) SetNotFound(handler http.HandlerFunc) {
	router.mu.Lock()
	defer router.mu.Unlock()
	router.notFound = handler
}

// GetRoutes returns all registered routes
func (router *Router) GetRoutes() []RouteInfo {
	router.mu.RLock()
	defer router.mu.RUnlock()

	var routes []RouteInfo
	for method, paths := range router.routes {
		for path := range paths {
			routes = append(routes, RouteInfo{
				Method: method,
				Path:   path,
			})
		}
	}
	return routes
}

// Use adds a middleware to the router
func (router *Router) Use(middleware func(http.Handler) http.Handler) {
	router.mu.Lock()
	defer router.mu.Unlock()
	router.middlewares = append(router.middlewares, middleware)
}

// Group creates a route group with a common prefix
func (router *Router) Group(prefix string) *RouteGroup {
	return &RouteGroup{
		router: router,
		prefix: prefix,
	}
}

// RouteGroup represents a group of routes with a common prefix
type RouteGroup struct {
	router *Router
	prefix string
}

// HandleFunc registers a handler with the group prefix
func (g *RouteGroup) HandleFunc(method, path string, handler http.HandlerFunc) {
	g.router.HandleFunc(method, g.prefix+path, handler)
}

// GET registers a GET handler
func (g *RouteGroup) GET(path string, handler http.HandlerFunc) {
	g.HandleFunc("GET", path, handler)
}

// POST registers a POST handler
func (g *RouteGroup) POST(path string, handler http.HandlerFunc) {
	g.HandleFunc("POST", path, handler)
}

// RemoveRoute removes a registered route
func (router *Router) RemoveRoute(method, path string) {
	router.mu.Lock()
	defer router.mu.Unlock()

	method = strings.ToUpper(method)
	if routes, ok := router.routes[method]; ok {
		delete(routes, path)
	}
}

// matchPath performs simple path matching with wildcards
func matchPath(pattern, path string) bool {
	// Simple exact match for now
	// TODO: Add support for path parameters like /api/player/:id
	if pattern == path {
		return true
	}

	// Check for wildcard suffix
	if strings.HasSuffix(pattern, "*") {
		prefix := strings.TrimSuffix(pattern, "*")
		return strings.HasPrefix(path, prefix)
	}

	return false
}
