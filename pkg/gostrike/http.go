// Package gostrike provides the public SDK for GoStrike plugins.
// This file provides HTTP server access for plugins.
package gostrike

import (
	"encoding/json"
	"net/http"

	httpmod "github.com/corrreia/gostrike/internal/modules/http"
)

// HTTPHandler is the function signature for HTTP handlers
type HTTPHandler func(w http.ResponseWriter, r *http.Request)

// HTTPContext provides context for HTTP handlers
type HTTPContext struct {
	Request  *http.Request
	Response http.ResponseWriter
}

// GetHTTP returns the HTTP module instance
func GetHTTP() *httpmod.Module {
	return httpmod.Get()
}

// RegisterHTTPHandler registers an HTTP handler for plugins
// method: HTTP method (GET, POST, PUT, DELETE, etc.)
// path: URL path (e.g., "/api/myplugin/status")
// handler: The handler function
func RegisterHTTPHandler(method, path string, handler HTTPHandler) {
	mod := httpmod.Get()
	if mod != nil {
		mod.RegisterHandler(method, path, http.HandlerFunc(handler))
	}
}

// RegisterGET registers a GET handler
func RegisterGET(path string, handler HTTPHandler) {
	RegisterHTTPHandler("GET", path, handler)
}

// RegisterPOST registers a POST handler
func RegisterPOST(path string, handler HTTPHandler) {
	RegisterHTTPHandler("POST", path, handler)
}

// RegisterPUT registers a PUT handler
func RegisterPUT(path string, handler HTTPHandler) {
	RegisterHTTPHandler("PUT", path, handler)
}

// RegisterDELETE registers a DELETE handler
func RegisterDELETE(path string, handler HTTPHandler) {
	RegisterHTTPHandler("DELETE", path, handler)
}

// IsHTTPEnabled returns true if the HTTP server is running
func IsHTTPEnabled() bool {
	mod := httpmod.Get()
	return mod != nil && mod.IsRunning()
}

// GetHTTPAddress returns the HTTP server address (e.g., "127.0.0.1:8080")
func GetHTTPAddress() string {
	mod := httpmod.Get()
	if mod != nil {
		return mod.GetAddress()
	}
	return ""
}

// JSON response helpers

// JSONResponse writes a JSON response
func JSONResponse(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if data != nil {
		json.NewEncoder(w).Encode(data)
	}
}

// JSONError writes a JSON error response
func JSONError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"error":   http.StatusText(status),
		"message": message,
	})
}

// JSONSuccess writes a JSON success response
func JSONSuccess(w http.ResponseWriter, data interface{}) {
	JSONResponse(w, http.StatusOK, data)
}

// ReadJSON reads JSON from request body into a struct
func ReadJSON(r *http.Request, v interface{}) error {
	return json.NewDecoder(r.Body).Decode(v)
}

// HTTPRouteGroup represents a group of routes with a common prefix
type HTTPRouteGroup struct {
	prefix string
}

// NewHTTPGroup creates a new route group
// Deprecated: Plugins should use NewPluginHTTPGroup for automatic namespacing
func NewHTTPGroup(prefix string) *HTTPRouteGroup {
	return &HTTPRouteGroup{prefix: prefix}
}

// GET registers a GET handler
func (g *HTTPRouteGroup) GET(path string, handler HTTPHandler) {
	RegisterGET(g.prefix+path, handler)
}

// POST registers a POST handler
func (g *HTTPRouteGroup) POST(path string, handler HTTPHandler) {
	RegisterPOST(g.prefix+path, handler)
}

// PUT registers a PUT handler
func (g *HTTPRouteGroup) PUT(path string, handler HTTPHandler) {
	RegisterPUT(g.prefix+path, handler)
}

// DELETE registers a DELETE handler
func (g *HTTPRouteGroup) DELETE(path string, handler HTTPHandler) {
	RegisterDELETE(g.prefix+path, handler)
}

// PluginHTTPGroup provides namespaced HTTP routes for plugins
// All routes are automatically prefixed with /api/plugins/[slug]/
type PluginHTTPGroup struct {
	slug   string
	prefix string
}

// NewPluginHTTPGroup creates a route group for a plugin with automatic namespacing
// Routes will be prefixed with /api/plugins/[slug]
// Example: api.GET("/status", handler) registers GET /api/plugins/[slug]/status
func NewPluginHTTPGroup(slug string) *PluginHTTPGroup {
	return &PluginHTTPGroup{
		slug:   slug,
		prefix: "/api/plugins/" + slug,
	}
}

// Slug returns the plugin's slug
func (g *PluginHTTPGroup) Slug() string {
	return g.slug
}

// Prefix returns the full route prefix (/api/plugins/[slug])
func (g *PluginHTTPGroup) Prefix() string {
	return g.prefix
}

// Handle registers a handler for any HTTP method
func (g *PluginHTTPGroup) Handle(method, path string, handler HTTPHandler) {
	RegisterHTTPHandler(method, g.prefix+path, handler)
}

// GET registers a GET handler
func (g *PluginHTTPGroup) GET(path string, handler HTTPHandler) {
	RegisterGET(g.prefix+path, handler)
}

// POST registers a POST handler
func (g *PluginHTTPGroup) POST(path string, handler HTTPHandler) {
	RegisterPOST(g.prefix+path, handler)
}

// PUT registers a PUT handler
func (g *PluginHTTPGroup) PUT(path string, handler HTTPHandler) {
	RegisterPUT(g.prefix+path, handler)
}

// DELETE registers a DELETE handler
func (g *PluginHTTPGroup) DELETE(path string, handler HTTPHandler) {
	RegisterDELETE(g.prefix+path, handler)
}

// PATCH registers a PATCH handler
func (g *PluginHTTPGroup) PATCH(path string, handler HTTPHandler) {
	RegisterHTTPHandler("PATCH", g.prefix+path, handler)
}

// Group creates a sub-group under the plugin's namespace
// Example: api.Group("/users").GET("/:id", handler) registers GET /api/plugins/[slug]/users/:id
func (g *PluginHTTPGroup) Group(subprefix string) *HTTPRouteGroup {
	return &HTTPRouteGroup{prefix: g.prefix + subprefix}
}
