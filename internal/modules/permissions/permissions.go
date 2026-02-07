// Package permissions provides the permissions system for GoStrike.
// It handles admin flags, groups, and SteamID-based permission checks.
package permissions

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// findConfigFile searches for a config file in known paths
func findConfigFile(filename string) string {
	paths := []string{
		"csgo/addons/gostrike/configs/" + filename,
		"/home/steam/cs2-dedicated/game/csgo/addons/gostrike/configs/" + filename,
		"addons/gostrike/configs/" + filename,
		"configs/" + filename,
	}
	for _, p := range paths {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	// Return the most likely path even if not found (will error on load)
	return paths[0]
}

// OverridesConfig represents the admin_overrides.json file
type OverridesConfig struct {
	CommandOverrides map[string]string `json:"command_overrides"`
}

// Module implements the permissions module
type Module struct {
	mu             sync.RWMutex
	cache          *AdminCache
	configPath     string
	overridesPath  string
	overrides      *OverridesConfig
	loaded         bool
}

// Config represents the permissions configuration file
type Config struct {
	Groups []GroupConfig `json:"groups"`
	Admins []AdminConfig `json:"admins"`
}

// GroupConfig represents a group in the config file
type GroupConfig struct {
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
	Flags       string `json:"flags"`
	Immunity    int    `json:"immunity"`
}

// AdminConfig represents an admin in the config file
type AdminConfig struct {
	SteamID  string   `json:"steamid"`
	Name     string   `json:"name"`
	Flags    string   `json:"flags"`
	Groups   []string `json:"groups"`
	Immunity int      `json:"immunity"`
	Expires  int64    `json:"expires"`
	Comment  string   `json:"comment"`
}

// instance is the singleton instance
var instance *Module

// New creates a new permissions module
func New() *Module {
	if instance != nil {
		return instance
	}
	instance = &Module{
		cache:         NewAdminCache(),
		configPath:    findConfigFile("admins.json"),
		overridesPath: findConfigFile("admin_overrides.json"),
	}
	return instance
}

// Get returns the singleton instance
func Get() *Module {
	return instance
}

// Name returns the module name
func (m *Module) Name() string {
	return "Permissions"
}

// Version returns the module version
func (m *Module) Version() string {
	return "1.0.0"
}

// Priority returns the module load priority
func (m *Module) Priority() int {
	return 10 // Load early
}

// Init initializes the permissions module
func (m *Module) Init() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.loaded {
		return nil
	}

	// Try to load config
	if err := m.loadConfigLocked(); err != nil {
		// Create default config if not found
		if os.IsNotExist(err) {
			if err := m.createDefaultConfig(); err != nil {
				return fmt.Errorf("failed to create default config: %w", err)
			}
			// Try loading again
			if err := m.loadConfigLocked(); err != nil {
				return fmt.Errorf("failed to load config after creation: %w", err)
			}
		} else {
			return fmt.Errorf("failed to load config: %w", err)
		}
	}

	// Load overrides (optional - don't fail if not found)
	m.loadOverridesLocked()

	m.loaded = true
	return nil
}

// Shutdown shuts down the permissions module
func (m *Module) Shutdown() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.cache.Clear()
	m.loaded = false
	return nil
}

// loadConfigLocked loads the configuration (must be called with lock held)
func (m *Module) loadConfigLocked() error {
	data, err := os.ReadFile(m.configPath)
	if err != nil {
		return err
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("failed to parse config: %w", err)
	}

	// Clear existing cache
	m.cache.Clear()

	// Load groups
	for _, gc := range config.Groups {
		group := &Group{
			Name:        gc.Name,
			DisplayName: gc.DisplayName,
			Flags:       ParseFlags(gc.Flags),
			FlagsStr:    gc.Flags,
			Immunity:    gc.Immunity,
		}
		m.cache.AddGroup(group)
	}

	// Load admins
	for _, ac := range config.Admins {
		steamID, err := ParseSteamID(ac.SteamID)
		if err != nil {
			continue // Skip invalid entries
		}

		admin := &Admin{
			SteamID:    steamID,
			SteamIDStr: ac.SteamID,
			Name:       ac.Name,
			Flags:      ParseFlags(ac.Flags),
			FlagsStr:   ac.Flags,
			Groups:     ac.Groups,
			Immunity:   ac.Immunity,
			ExpireTime: ac.Expires,
			Comment:    ac.Comment,
		}
		m.cache.AddAdmin(admin)
	}

	return nil
}

// createDefaultConfig creates a default configuration file
func (m *Module) createDefaultConfig() error {
	config := Config{
		Groups: []GroupConfig{
			{
				Name:        "admin",
				DisplayName: "Administrator",
				Flags:       "abcdefghij",
				Immunity:    100,
			},
			{
				Name:        "moderator",
				DisplayName: "Moderator",
				Flags:       "bcfj",
				Immunity:    50,
			},
			{
				Name:        "vip",
				DisplayName: "VIP",
				Flags:       "a",
				Immunity:    10,
			},
		},
		Admins: []AdminConfig{
			{
				SteamID:  "STEAM_0:0:12345",
				Name:     "Example Admin",
				Groups:   []string{"admin"},
				Immunity: 100,
				Comment:  "Example admin entry - replace with real SteamID",
			},
		},
	}

	// Ensure directory exists
	dir := filepath.Dir(m.configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(m.configPath, data, 0644)
}

// loadOverridesLocked loads admin overrides (must be called with lock held)
func (m *Module) loadOverridesLocked() {
	data, err := os.ReadFile(m.overridesPath)
	if err != nil {
		// Overrides file is optional
		m.overrides = &OverridesConfig{
			CommandOverrides: make(map[string]string),
		}
		return
	}

	var config OverridesConfig
	if err := json.Unmarshal(data, &config); err != nil {
		m.overrides = &OverridesConfig{
			CommandOverrides: make(map[string]string),
		}
		return
	}

	if config.CommandOverrides == nil {
		config.CommandOverrides = make(map[string]string)
	}
	m.overrides = &config
}

// Reload reloads the configuration
func (m *Module) Reload() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.loadOverridesLocked()
	return m.loadConfigLocked()
}

// SetConfigPath sets the configuration file path
func (m *Module) SetConfigPath(path string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.configPath = path
}

// ============================================================
// Permission Check Functions
// ============================================================

// HasFlag checks if a SteamID has a specific flag
func (m *Module) HasFlag(steamID uint64, flag AdminFlag) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	admin := m.cache.GetAdmin(steamID)
	if admin == nil {
		return false
	}

	// Check expiration
	if admin.ExpireTime > 0 && time.Now().Unix() > admin.ExpireTime {
		return false
	}

	return admin.EffectiveFlags(m.cache.groups).Has(flag)
}

// HasAnyFlag checks if a SteamID has any of the specified flags
func (m *Module) HasAnyFlag(steamID uint64, flags AdminFlag) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	admin := m.cache.GetAdmin(steamID)
	if admin == nil {
		return false
	}

	// Check expiration
	if admin.ExpireTime > 0 && time.Now().Unix() > admin.ExpireTime {
		return false
	}

	return admin.EffectiveFlags(m.cache.groups).HasAny(flags)
}

// GetFlags returns all flags for a SteamID
func (m *Module) GetFlags(steamID uint64) AdminFlag {
	m.mu.RLock()
	defer m.mu.RUnlock()

	admin := m.cache.GetAdmin(steamID)
	if admin == nil {
		return FlagNone
	}

	// Check expiration
	if admin.ExpireTime > 0 && time.Now().Unix() > admin.ExpireTime {
		return FlagNone
	}

	return admin.EffectiveFlags(m.cache.groups)
}

// GetImmunity returns the immunity level for a SteamID
func (m *Module) GetImmunity(steamID uint64) int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	admin := m.cache.GetAdmin(steamID)
	if admin == nil {
		return 0
	}

	// Check expiration
	if admin.ExpireTime > 0 && time.Now().Unix() > admin.ExpireTime {
		return 0
	}

	return admin.EffectiveImmunity(m.cache.groups)
}

// IsAdmin checks if a SteamID is an admin (has any flags)
func (m *Module) IsAdmin(steamID uint64) bool {
	return m.GetFlags(steamID) != FlagNone
}

// GetCommandFlag returns the required flag for a command, checking overrides first.
// If the command has an override, the override flag is returned.
// Otherwise, the defaultFlag is returned.
func (m *Module) GetCommandFlag(command string, defaultFlag AdminFlag) AdminFlag {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.overrides == nil {
		return defaultFlag
	}

	flagStr, ok := m.overrides.CommandOverrides[command]
	if !ok {
		return defaultFlag
	}

	overrideFlag := ParseFlags(flagStr)
	if overrideFlag == FlagNone {
		return defaultFlag
	}

	return overrideFlag
}

// CanTarget checks if source can target destination based on immunity
func (m *Module) CanTarget(sourceSteamID, targetSteamID uint64) bool {
	sourceImmunity := m.GetImmunity(sourceSteamID)
	targetImmunity := m.GetImmunity(targetSteamID)

	// Root can target anyone
	if m.HasFlag(sourceSteamID, FlagRoot) {
		return true
	}

	// Can target if source immunity >= target immunity
	return sourceImmunity >= targetImmunity
}

// ============================================================
// Admin Management Functions
// ============================================================

// GetAdmin returns admin info by SteamID
func (m *Module) GetAdmin(steamID uint64) *Admin {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.cache.GetAdmin(steamID)
}

// GetAllAdmins returns all admins
func (m *Module) GetAllAdmins() []*Admin {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.cache.GetAllAdmins()
}

// GetGroup returns a group by name
func (m *Module) GetGroup(name string) *Group {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.cache.GetGroup(name)
}

// GetAllGroups returns all groups
func (m *Module) GetAllGroups() map[string]*Group {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.cache.GetGroups()
}

// GetStats returns statistics about loaded admins and groups
func (m *Module) GetStats() (admins int, groups int) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.cache.Count()
}
