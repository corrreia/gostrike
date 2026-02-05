// Package http provides an embedded HTTP server module for GoStrike.
// It allows plugins to register endpoints and provides built-in API endpoints.
package http

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/corrreia/gostrike/internal/modules"
	"github.com/corrreia/gostrike/internal/shared"
)

// Register the HTTP module at init time
func init() {
	modules.Register(New())
}

// Module implements the HTTP server module
type Module struct {
	mu         sync.RWMutex
	server     *http.Server
	router     *Router
	config     *Config
	configPath string
	running    bool
}

// Config represents the HTTP module configuration
type Config struct {
	Enabled     bool   `json:"enabled"`
	Host        string `json:"host"`
	Port        int    `json:"port"`
	EnableCORS  bool   `json:"enable_cors"`
	CORSOrigins string `json:"cors_origins"`
	RateLimit   int    `json:"rate_limit"` // Requests per minute, 0 = disabled
}

// DefaultConfig returns the default configuration
func DefaultConfig() *Config {
	return &Config{
		Enabled:     true,
		Host:        "0.0.0.0",
		Port:        8080,
		EnableCORS:  true,
		CORSOrigins: "*",
		RateLimit:   0, // No rate limit by default
	}
}

// instance is the singleton instance
var instance *Module

// New creates a new HTTP module
func New() *Module {
	if instance != nil {
		return instance
	}
	instance = &Module{
		router:     NewRouter(),
		config:     DefaultConfig(),
		configPath: "configs/http.json",
	}
	return instance
}

// Get returns the singleton instance
func Get() *Module {
	return instance
}

// Name returns the module name
func (m *Module) Name() string {
	return "HTTP"
}

// Version returns the module version
func (m *Module) Version() string {
	return "1.0.0"
}

// Priority returns the module load priority
func (m *Module) Priority() int {
	return 50 // Load after permissions
}

// Init initializes the HTTP module
func (m *Module) Init() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Load config
	if err := m.loadConfig(); err != nil {
		// Use default config
		m.config = DefaultConfig()
	}

	// Register built-in routes
	m.registerBuiltinRoutes()

	// Start server if enabled
	if m.config.Enabled {
		return m.startServerLocked()
	}

	return nil
}

// Shutdown shuts down the HTTP module
func (m *Module) Shutdown() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.server != nil && m.running {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := m.server.Shutdown(ctx); err != nil {
			return fmt.Errorf("server shutdown error: %w", err)
		}
		m.running = false
	}

	return nil
}

// loadConfig loads the configuration
func (m *Module) loadConfig() error {
	// Try multiple paths for the config file
	configPaths := []string{
		"/home/steam/cs2-dedicated/game/csgo/addons/gostrike/configs/http.json",
		"addons/gostrike/configs/http.json",
		"configs/http.json",
	}

	var data []byte
	var err error
	for _, path := range configPaths {
		data, err = os.ReadFile(path)
		if err == nil {
			m.configPath = path
			break
		}
	}

	if err != nil {
		shared.LogDebug("HTTP", "Config not found, using defaults")
		m.config = DefaultConfig()
		return nil
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		shared.LogWarning("HTTP", "Failed to parse config: %v, using defaults", err)
		m.config = DefaultConfig()
		return nil
	}

	m.config = &config
	shared.LogInfo("HTTP", "Loaded config from %s", m.configPath)
	return nil
}

// startServerLocked starts the HTTP server (must be called with lock held)
func (m *Module) startServerLocked() error {
	if m.running {
		return nil
	}

	addr := fmt.Sprintf("%s:%d", m.config.Host, m.config.Port)

	// Create handler with middleware chain
	var handler http.Handler = m.router

	if m.config.EnableCORS {
		handler = &corsMiddleware{
			handler: handler,
			origins: m.config.CORSOrigins,
		}
	}

	m.server = &http.Server{
		Addr:         addr,
		Handler:      handler,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		if err := m.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			// Log error but don't crash
		}
	}()

	m.running = true
	return nil
}

// registerBuiltinRoutes registers the built-in API routes
func (m *Module) registerBuiltinRoutes() {
	// Health check
	m.router.HandleFunc("GET", "/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "ok",
			"time":   time.Now().Unix(),
		})
	})

	// API status - shows GoStrike runtime status
	m.router.HandleFunc("GET", "/api/status", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		// Get module info
		moduleList := modules.GetAll()

		json.NewEncoder(w).Encode(map[string]interface{}{
			"version":       "0.1.0",
			"abi_version":   1,
			"status":        "running",
			"modules_count": len(moduleList),
		})
	})

	// API plugins list - shows loaded plugins
	m.router.HandleFunc("GET", "/api/plugins", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		// Get plugin list via callback (set during initialization)
		plugins := getPluginList()

		json.NewEncoder(w).Encode(map[string]interface{}{
			"count":   len(plugins),
			"plugins": plugins,
		})
	})

	// API modules list - shows core modules
	m.router.HandleFunc("GET", "/api/modules", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		moduleList := modules.GetAll()
		result := make([]map[string]interface{}, len(moduleList))

		for i, mod := range moduleList {
			result[i] = map[string]interface{}{
				"name":     mod.Name,
				"version":  mod.Version,
				"priority": mod.Priority,
				"state":    mod.State,
			}
			if mod.Error != "" {
				result[i]["error"] = mod.Error
			}
		}

		json.NewEncoder(w).Encode(map[string]interface{}{
			"count":   len(moduleList),
			"modules": result,
		})
	})

	// API routes list
	m.router.HandleFunc("GET", "/api/routes", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		routes := m.router.GetRoutes()
		json.NewEncoder(w).Encode(map[string]interface{}{
			"count":  len(routes),
			"routes": routes,
		})
	})
}

// Plugin list callback - set during initialization
var (
	pluginListCallback func() []map[string]interface{}
	pluginListMu       sync.RWMutex
)

// SetPluginListCallback sets the callback function to get plugin list
func SetPluginListCallback(fn func() []map[string]interface{}) {
	pluginListMu.Lock()
	defer pluginListMu.Unlock()
	pluginListCallback = fn
}

func getPluginList() []map[string]interface{} {
	pluginListMu.RLock()
	defer pluginListMu.RUnlock()

	if pluginListCallback != nil {
		return pluginListCallback()
	}
	return []map[string]interface{}{}
}

// ============================================================
// Public API
// ============================================================

// IsRunning returns true if the HTTP server is running
func (m *Module) IsRunning() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.running
}

// GetAddress returns the server address
func (m *Module) GetAddress() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if !m.running {
		return ""
	}
	return fmt.Sprintf("%s:%d", m.config.Host, m.config.Port)
}

// RegisterHandler registers a new HTTP handler for plugins
func (m *Module) RegisterHandler(method, path string, handler http.HandlerFunc) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.router.HandleFunc(method, path, handler)
}

// RegisterHandlerFunc registers a handler with a custom function signature
func (m *Module) RegisterHandlerFunc(method, path string, handler func(w http.ResponseWriter, r *http.Request)) {
	m.RegisterHandler(method, path, handler)
}

// Start starts the HTTP server (if configured)
func (m *Module) Start() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.config.Enabled {
		return fmt.Errorf("HTTP server is disabled in config")
	}

	return m.startServerLocked()
}

// Stop stops the HTTP server
func (m *Module) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.server != nil && m.running {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := m.server.Shutdown(ctx); err != nil {
			return err
		}
		m.running = false
	}

	return nil
}

// SetConfig sets the configuration
func (m *Module) SetConfig(config *Config) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.config = config
}

// GetConfig returns a copy of the configuration
func (m *Module) GetConfig() Config {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return *m.config
}
