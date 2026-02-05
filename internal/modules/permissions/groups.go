// Package permissions provides the permissions system for GoStrike.
// This file defines permission groups and admin entries.
package permissions

import (
	"fmt"
	"sync"
)

// Group represents a permission group
type Group struct {
	Name        string    `json:"name"`
	DisplayName string    `json:"display_name"`
	Flags       AdminFlag `json:"-"`
	FlagsStr    string    `json:"flags"`    // Stored as string in JSON
	Immunity    int       `json:"immunity"` // Immunity level (higher = more immune)
}

// Admin represents an admin entry
type Admin struct {
	SteamID    uint64    `json:"steam_id"`
	SteamIDStr string    `json:"steamid"`  // String format for JSON
	Name       string    `json:"name"`     // Display name (optional)
	Flags      AdminFlag `json:"-"`        // Direct flags
	FlagsStr   string    `json:"flags"`    // Stored as string in JSON
	Groups     []string  `json:"groups"`   // Group memberships
	Immunity   int       `json:"immunity"` // Personal immunity override
	ExpireTime int64     `json:"expires"`  // Unix timestamp, 0 = never
	Comment    string    `json:"comment"`  // Admin comment
}

// EffectiveFlags returns the combined flags from direct assignment and groups
func (a *Admin) EffectiveFlags(groups map[string]*Group) AdminFlag {
	result := a.Flags
	for _, groupName := range a.Groups {
		if group, ok := groups[groupName]; ok {
			result |= group.Flags
		}
	}
	return result
}

// EffectiveImmunity returns the highest immunity from direct assignment or groups
func (a *Admin) EffectiveImmunity(groups map[string]*Group) int {
	result := a.Immunity
	for _, groupName := range a.Groups {
		if group, ok := groups[groupName]; ok {
			if group.Immunity > result {
				result = group.Immunity
			}
		}
	}
	return result
}

// AdminCache provides fast lookup of admin permissions by various keys
type AdminCache struct {
	mu      sync.RWMutex
	bySteam map[uint64]*Admin
	groups  map[string]*Group
}

// NewAdminCache creates a new admin cache
func NewAdminCache() *AdminCache {
	return &AdminCache{
		bySteam: make(map[uint64]*Admin),
		groups:  make(map[string]*Group),
	}
}

// Clear clears the cache
func (c *AdminCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.bySteam = make(map[uint64]*Admin)
	c.groups = make(map[string]*Group)
}

// AddGroup adds a group to the cache
func (c *AdminCache) AddGroup(group *Group) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.groups[group.Name] = group
}

// GetGroup retrieves a group by name
func (c *AdminCache) GetGroup(name string) *Group {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.groups[name]
}

// GetGroups returns all groups
func (c *AdminCache) GetGroups() map[string]*Group {
	c.mu.RLock()
	defer c.mu.RUnlock()

	result := make(map[string]*Group, len(c.groups))
	for k, v := range c.groups {
		result[k] = v
	}
	return result
}

// AddAdmin adds an admin to the cache
func (c *AdminCache) AddAdmin(admin *Admin) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.bySteam[admin.SteamID] = admin
}

// GetAdmin retrieves an admin by SteamID
func (c *AdminCache) GetAdmin(steamID uint64) *Admin {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.bySteam[steamID]
}

// GetAdminFlags returns the effective flags for a SteamID
func (c *AdminCache) GetAdminFlags(steamID uint64) AdminFlag {
	c.mu.RLock()
	defer c.mu.RUnlock()

	admin, ok := c.bySteam[steamID]
	if !ok {
		return FlagNone
	}
	return admin.EffectiveFlags(c.groups)
}

// GetAdminImmunity returns the effective immunity for a SteamID
func (c *AdminCache) GetAdminImmunity(steamID uint64) int {
	c.mu.RLock()
	defer c.mu.RUnlock()

	admin, ok := c.bySteam[steamID]
	if !ok {
		return 0
	}
	return admin.EffectiveImmunity(c.groups)
}

// HasFlag checks if a SteamID has a specific flag
func (c *AdminCache) HasFlag(steamID uint64, flag AdminFlag) bool {
	return c.GetAdminFlags(steamID).Has(flag)
}

// GetAllAdmins returns all admins
func (c *AdminCache) GetAllAdmins() []*Admin {
	c.mu.RLock()
	defer c.mu.RUnlock()

	result := make([]*Admin, 0, len(c.bySteam))
	for _, admin := range c.bySteam {
		result = append(result, admin)
	}
	return result
}

// Count returns the number of admins and groups
func (c *AdminCache) Count() (admins int, groups int) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.bySteam), len(c.groups)
}

// ParseSteamID parses various SteamID formats
// Supports: STEAM_X:Y:Z, [U:1:123456], 76561198012345678
func ParseSteamID(s string) (uint64, error) {
	// Check for SteamID64 format (17-digit number starting with 765)
	if len(s) == 17 && s[0] == '7' && s[1] == '6' && s[2] == '5' {
		var id uint64
		for _, c := range s {
			if c < '0' || c > '9' {
				return 0, fmt.Errorf("invalid steamid64: %s", s)
			}
			id = id*10 + uint64(c-'0')
		}
		return id, nil
	}

	// Check for STEAM_X:Y:Z format
	if len(s) > 8 && s[0:6] == "STEAM_" {
		// Parse STEAM_X:Y:Z
		var x, y, z uint64
		_, err := fmt.Sscanf(s, "STEAM_%d:%d:%d", &x, &y, &z)
		if err != nil {
			return 0, fmt.Errorf("invalid steam_id format: %s", s)
		}
		// Convert to SteamID64
		return 76561197960265728 + z*2 + y, nil
	}

	// Check for [U:1:Z] format
	if len(s) > 5 && s[0] == '[' && s[1] == 'U' && s[2] == ':' {
		var universe, z uint64
		_, err := fmt.Sscanf(s, "[U:%d:%d]", &universe, &z)
		if err != nil {
			return 0, fmt.Errorf("invalid steam3id format: %s", s)
		}
		return 76561197960265728 + z, nil
	}

	// Try parsing as raw number
	var id uint64
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0, fmt.Errorf("invalid steamid: %s", s)
		}
		id = id*10 + uint64(c-'0')
	}
	return id, nil
}

// FormatSteamID64 formats a SteamID64 to string
func FormatSteamID64(id uint64) string {
	return fmt.Sprintf("%d", id)
}

// FormatSteamID2 formats a SteamID64 to STEAM_X:Y:Z format
func FormatSteamID2(id uint64) string {
	if id < 76561197960265728 {
		return fmt.Sprintf("STEAM_0:0:%d", id)
	}
	w := id - 76561197960265728
	y := w % 2
	z := w / 2
	return fmt.Sprintf("STEAM_0:%d:%d", y, z)
}

// FormatSteamID3 formats a SteamID64 to [U:1:Z] format
func FormatSteamID3(id uint64) string {
	if id < 76561197960265728 {
		return fmt.Sprintf("[U:1:%d]", id)
	}
	z := id - 76561197960265728
	return fmt.Sprintf("[U:1:%d]", z)
}
