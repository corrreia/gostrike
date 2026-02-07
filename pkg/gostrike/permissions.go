package gostrike

import (
	"github.com/corrreia/gostrike/internal/modules/permissions"
	"github.com/corrreia/gostrike/internal/scope"
)

// ============================================================
// Permission checking — string-based
// ============================================================

// HasPermission checks if a steamID has a specific permission string.
// Supports exact match ("gostrike.kick") and wildcards ("gostrike.*", "*").
func HasPermission(steamID uint64, permission string) bool {
	pm := permissions.Get()
	if pm != nil {
		return pm.HasPermission(steamID, permission)
	}
	return false
}

// IsAdmin checks if a steamID has any permissions at all.
func IsAdmin(steamID uint64) bool {
	pm := permissions.Get()
	if pm != nil {
		return pm.IsAdmin(steamID)
	}
	return false
}

// GetImmunity returns the effective immunity level for a steamID.
func GetImmunity(steamID uint64) int {
	pm := permissions.Get()
	if pm != nil {
		return pm.GetImmunity(steamID)
	}
	return 0
}

// CanTarget checks if source can target destination based on immunity.
func CanTarget(sourceSteamID, targetSteamID uint64) bool {
	pm := permissions.Get()
	if pm != nil {
		return pm.CanTarget(sourceSteamID, targetSteamID)
	}
	return true
}

// RegisterPermission declares a permission so it shows up in the API registry.
// Plugins should call this in Load() for every permission they use.
func RegisterPermission(name, description string) {
	pm := permissions.Get()
	if pm != nil {
		pm.RegisterPermission(name, description)
		if s := scope.GetActive(); s != nil {
			s.TrackPermission(name)
		}
	}
}

// ============================================================
// Player helpers
// ============================================================

// HasPermission checks if the player has a specific permission.
func (p *Player) HasPermission(permission string) bool {
	if p == nil {
		return false
	}
	return HasPermission(p.SteamID, permission)
}

// IsAdmin checks if the player has any permissions.
func (p *Player) IsAdmin() bool {
	if p == nil {
		return false
	}
	return IsAdmin(p.SteamID)
}

// GetImmunity returns the player's effective immunity level.
func (p *Player) GetImmunity() int {
	if p == nil {
		return 0
	}
	return GetImmunity(p.SteamID)
}

// CanTarget checks if this player can target another player.
func (p *Player) CanTarget(target *Player) bool {
	if p == nil || target == nil {
		return false
	}
	return CanTarget(p.SteamID, target.SteamID)
}

// ============================================================
// CommandContext helpers
// ============================================================

// HasPermission checks if the command invoker has a specific permission.
func (ctx *CommandContext) HasPermission(permission string) bool {
	if ctx.Player == nil {
		return false
	}
	return HasPermission(ctx.Player.SteamID, permission)
}

// IsAdmin checks if the command invoker is an admin.
func (ctx *CommandContext) IsAdmin() bool {
	if ctx.Player == nil {
		return false
	}
	return IsAdmin(ctx.Player.SteamID)
}

// RequirePermission checks permission and sends error if not authorized.
// Returns true if authorized, false otherwise.
func (ctx *CommandContext) RequirePermission(permission string) bool {
	if ctx.HasPermission(permission) {
		return true
	}
	ctx.ReplyError("You do not have permission to use this command")
	return false
}
