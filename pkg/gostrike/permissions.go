// Package gostrike provides the public SDK for GoStrike plugins.
package gostrike

import (
	"github.com/corrreia/gostrike/internal/modules/permissions"
)

// AdminFlags define admin permission levels
// These are re-exported from the internal permissions module
type AdminFlags = permissions.AdminFlag

// Admin flag constants - re-export from permissions module
const (
	AdminNone        AdminFlags = permissions.FlagNone
	AdminReservation AdminFlags = permissions.FlagReservation
	AdminKick        AdminFlags = permissions.FlagKick
	AdminBan         AdminFlags = permissions.FlagBan
	AdminUnban       AdminFlags = permissions.FlagUnban
	AdminSlay        AdminFlags = permissions.FlagSlay
	AdminChangeMap   AdminFlags = permissions.FlagChangelevel
	AdminCvar        AdminFlags = permissions.FlagCvar
	AdminConfig      AdminFlags = permissions.FlagConfig
	AdminChat        AdminFlags = permissions.FlagChat
	AdminVote        AdminFlags = permissions.FlagVote
	AdminPassword    AdminFlags = permissions.FlagPassword
	AdminRCon        AdminFlags = permissions.FlagRcon
	AdminCheats      AdminFlags = permissions.FlagCheats
	AdminRoot        AdminFlags = permissions.FlagRoot

	// Convenience aliases
	AdminGeneric = permissions.FlagGeneric
	AdminFull    = permissions.FlagFull
)

// Admin represents an admin user (wraps internal type)
type Admin struct {
	SteamID  uint64
	Name     string
	Flags    AdminFlags
	Groups   []string
	Immunity int
}

// GetPermissions returns the permissions module instance
func GetPermissions() *permissions.Module {
	return permissions.Get()
}

// GetAdmin retrieves an admin by Steam ID
// Uses the permissions module if available, falls back to local store
func GetAdmin(steamID uint64) *Admin {
	pm := permissions.Get()
	if pm != nil {
		internal := pm.GetAdmin(steamID)
		if internal != nil {
			return &Admin{
				SteamID:  internal.SteamID,
				Name:     internal.Name,
				Flags:    internal.EffectiveFlags(pm.GetAllGroups()),
				Groups:   internal.Groups,
				Immunity: internal.EffectiveImmunity(pm.GetAllGroups()),
			}
		}
	}
	return nil
}

// GetAllAdmins returns all registered admins
func GetAllAdmins() []*Admin {
	pm := permissions.Get()
	if pm == nil {
		return nil
	}

	internal := pm.GetAllAdmins()
	groups := pm.GetAllGroups()
	result := make([]*Admin, len(internal))
	for i, a := range internal {
		result[i] = &Admin{
			SteamID:  a.SteamID,
			Name:     a.Name,
			Flags:    a.EffectiveFlags(groups),
			Groups:   a.Groups,
			Immunity: a.EffectiveImmunity(groups),
		}
	}
	return result
}

// HasFlag checks if a SteamID has a specific flag
func HasFlag(steamID uint64, flag AdminFlags) bool {
	pm := permissions.Get()
	if pm != nil {
		return pm.HasFlag(steamID, flag)
	}
	return false
}

// GetImmunity returns the immunity level for a SteamID
func GetImmunity(steamID uint64) int {
	pm := permissions.Get()
	if pm != nil {
		return pm.GetImmunity(steamID)
	}
	return 0
}

// CanTarget checks if source can target destination based on immunity
func CanTarget(sourceSteamID, targetSteamID uint64) bool {
	pm := permissions.Get()
	if pm != nil {
		return pm.CanTarget(sourceSteamID, targetSteamID)
	}
	return true
}

// Admin methods

// HasFlag checks if the admin has a specific flag
func (a *Admin) HasFlag(flag AdminFlags) bool {
	if a == nil {
		return false
	}
	return a.Flags.Has(flag)
}

// HasAnyFlag checks if the admin has any of the specified flags
func (a *Admin) HasAnyFlag(flags AdminFlags) bool {
	if a == nil {
		return false
	}
	return a.Flags.HasAny(flags)
}

// HasAllFlags checks if the admin has all of the specified flags
func (a *Admin) HasAllFlags(flags AdminFlags) bool {
	if a == nil {
		return false
	}
	return a.Flags.Has(flags)
}

// IsRoot checks if the admin has root (superadmin) access
func (a *Admin) IsRoot() bool {
	return a != nil && a.Flags.Has(AdminRoot)
}

// Player permission helpers

// GetPlayerAdmin returns the Admin for a player, or nil if not an admin
func GetPlayerAdmin(player *Player) *Admin {
	if player == nil {
		return nil
	}
	return GetAdmin(player.SteamID)
}

// PlayerHasFlag checks if a player has a specific admin flag
func PlayerHasFlag(player *Player, flag AdminFlags) bool {
	if player == nil {
		return false
	}
	return HasFlag(player.SteamID, flag)
}

// PlayerIsAdmin checks if a player is an admin (has any admin flags)
func PlayerIsAdmin(player *Player) bool {
	admin := GetPlayerAdmin(player)
	return admin != nil && admin.Flags != AdminNone
}

// Player.HasPermission checks if the player has a specific permission
func (p *Player) HasPermission(flag AdminFlags) bool {
	if p == nil {
		return false
	}
	return HasFlag(p.SteamID, flag)
}

// Player.IsAdmin checks if the player is an admin
func (p *Player) IsAdmin() bool {
	return GetPlayerAdmin(p) != nil
}

// Player.GetImmunity returns the player's immunity level
func (p *Player) GetImmunity() int {
	if p == nil {
		return 0
	}
	return GetImmunity(p.SteamID)
}

// Player.CanTarget checks if this player can target another player
func (p *Player) CanTarget(target *Player) bool {
	if p == nil || target == nil {
		return false
	}
	return CanTarget(p.SteamID, target.SteamID)
}

// CommandContext permission helpers

// IsAdmin checks if the command invoker is an admin
func (ctx *CommandContext) IsAdmin() bool {
	return PlayerIsAdmin(ctx.Player)
}

// HasFlag checks if the command invoker has a specific admin flag
func (ctx *CommandContext) HasFlag(flag AdminFlags) bool {
	return PlayerHasFlag(ctx.Player, flag)
}

// RequireFlag checks permission and sends error if not authorized
// Returns true if authorized, false otherwise
func (ctx *CommandContext) RequireFlag(flag AdminFlags) bool {
	if ctx.HasFlag(flag) {
		return true
	}
	ctx.ReplyError("You do not have permission to use this command")
	return false
}
