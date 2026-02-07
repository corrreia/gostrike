package permissions

import (
	"strings"
	"sync"
	"time"
)

// cachedRole holds a role in memory.
type cachedRole struct {
	ID          int64
	Name        string
	DisplayName string
	Immunity    int
	Permissions map[string]bool
}

// cachedPlayer holds a player in memory.
type cachedPlayer struct {
	SteamID     uint64
	Name        string
	Immunity    int
	ExpiresAt   int64
	Roles       []int64         // role IDs
	Permissions map[string]bool // direct permissions
}

// registeredPerm tracks a plugin-declared permission.
type registeredPerm struct {
	Name        string
	Description string
}

// permCache is the in-memory permission cache.
type permCache struct {
	mu          sync.RWMutex
	roles       map[int64]*cachedRole   // by role ID
	rolesByName map[string]*cachedRole  // by role name
	players     map[uint64]*cachedPlayer
	registered  map[string]*registeredPerm // plugin-registered permissions
}

func newPermCache() *permCache {
	return &permCache{
		roles:       make(map[int64]*cachedRole),
		rolesByName: make(map[string]*cachedRole),
		players:     make(map[uint64]*cachedPlayer),
		registered:  make(map[string]*registeredPerm),
	}
}

// clear wipes the cache (not registered perms — those come from plugins).
func (c *permCache) clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.roles = make(map[int64]*cachedRole)
	c.rolesByName = make(map[string]*cachedRole)
	c.players = make(map[uint64]*cachedPlayer)
}

// loadFromDB populates the cache from the database.
func (c *permCache) loadFromDB(roles []dbRole, players []dbPlayer) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.roles = make(map[int64]*cachedRole, len(roles))
	c.rolesByName = make(map[string]*cachedRole, len(roles))
	c.players = make(map[uint64]*cachedPlayer, len(players))

	// Index roles
	for _, r := range roles {
		perms := make(map[string]bool, len(r.Permissions))
		for _, p := range r.Permissions {
			perms[p] = true
		}
		cr := &cachedRole{
			ID:          r.ID,
			Name:        r.Name,
			DisplayName: r.DisplayName,
			Immunity:    r.Immunity,
			Permissions: perms,
		}
		c.roles[r.ID] = cr
		c.rolesByName[r.Name] = cr
	}

	// Index players, resolving role names to IDs
	for _, p := range players {
		perms := make(map[string]bool, len(p.Permissions))
		for _, perm := range p.Permissions {
			perms[perm] = true
		}
		var roleIDs []int64
		for _, roleName := range p.Roles {
			if cr, ok := c.rolesByName[roleName]; ok {
				roleIDs = append(roleIDs, cr.ID)
			}
		}
		c.players[p.SteamID] = &cachedPlayer{
			SteamID:     p.SteamID,
			Name:        p.Name,
			Immunity:    p.Immunity,
			ExpiresAt:   p.ExpiresAt,
			Roles:       roleIDs,
			Permissions: perms,
		}
	}
}

// hasPermission checks if a steamID has the requested permission.
func (c *permCache) hasPermission(steamID uint64, want string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	player, ok := c.players[steamID]
	if !ok {
		return false
	}

	// Check expiration
	if player.ExpiresAt > 0 && time.Now().Unix() > player.ExpiresAt {
		return false
	}

	// Collect all permissions: direct + from roles
	// Check direct permissions first
	for perm := range player.Permissions {
		if matchPermission(perm, want) {
			return true
		}
	}

	// Check role permissions
	for _, roleID := range player.Roles {
		role, ok := c.roles[roleID]
		if !ok {
			continue
		}
		for perm := range role.Permissions {
			if matchPermission(perm, want) {
				return true
			}
		}
	}

	return false
}

// isAdmin checks if a steamID has any permissions at all.
func (c *permCache) isAdmin(steamID uint64) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	player, ok := c.players[steamID]
	if !ok {
		return false
	}

	if player.ExpiresAt > 0 && time.Now().Unix() > player.ExpiresAt {
		return false
	}

	// Has direct permissions?
	if len(player.Permissions) > 0 {
		return true
	}

	// Has any role?
	return len(player.Roles) > 0
}

// getImmunity returns the effective immunity for a player.
func (c *permCache) getImmunity(steamID uint64) int {
	c.mu.RLock()
	defer c.mu.RUnlock()

	player, ok := c.players[steamID]
	if !ok {
		return 0
	}

	if player.ExpiresAt > 0 && time.Now().Unix() > player.ExpiresAt {
		return 0
	}

	immunity := player.Immunity
	for _, roleID := range player.Roles {
		if role, ok := c.roles[roleID]; ok {
			if role.Immunity > immunity {
				immunity = role.Immunity
			}
		}
	}
	return immunity
}

// getEffectivePermissions returns all resolved permissions for a player.
func (c *permCache) getEffectivePermissions(steamID uint64) []string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	player, ok := c.players[steamID]
	if !ok {
		return nil
	}

	seen := make(map[string]bool)
	for perm := range player.Permissions {
		seen[perm] = true
	}
	for _, roleID := range player.Roles {
		if role, ok := c.roles[roleID]; ok {
			for perm := range role.Permissions {
				seen[perm] = true
			}
		}
	}

	result := make([]string, 0, len(seen))
	for perm := range seen {
		result = append(result, perm)
	}
	return result
}

// registerPermission records a plugin-declared permission.
func (c *permCache) registerPermission(name, description string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.registered[name] = &registeredPerm{Name: name, Description: description}
}

// getRegistered returns all plugin-registered permissions.
func (c *permCache) getRegistered() map[string]string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	result := make(map[string]string, len(c.registered))
	for name, rp := range c.registered {
		result[name] = rp.Description
	}
	return result
}

// matchPermission checks if a held permission grants the wanted permission.
//   - "*" matches everything
//   - "gostrike.*" matches "gostrike.kick", "gostrike.ban", etc.
//   - Exact match: "gostrike.kick" == "gostrike.kick"
func matchPermission(have, want string) bool {
	if have == "*" {
		return true
	}
	if have == want {
		return true
	}
	// Prefix wildcard: "gostrike.*" matches "gostrike.kick"
	if strings.HasSuffix(have, ".*") {
		prefix := strings.TrimSuffix(have, "*") // "gostrike."
		return strings.HasPrefix(want, prefix)
	}
	return false
}
