// Package gostrike provides the public SDK for GoStrike plugins.
package gostrike

// AdminFlags define admin permission levels
type AdminFlags int

const (
	AdminNone        AdminFlags = 0
	AdminReservation AdminFlags = 1 << iota // Reserved slot access
	AdminGeneric                            // Generic admin (minimal)
	AdminKick                               // Kick players
	AdminBan                                // Ban players
	AdminUnban                              // Unban players
	AdminSlay                               // Slay players
	AdminChangeMap                          // Change map
	AdminCvar                               // Change cvars
	AdminConfig                             // Execute configs
	AdminChat                               // Admin chat commands
	AdminVote                               // Start votes
	AdminPassword                           // Set password
	AdminRCon                               // RCON access
	AdminCheats                             // Cheat commands
	AdminRoot                               // All permissions (superadmin)
)

// Admin represents an admin user
type Admin struct {
	SteamID uint64
	Name    string
	Flags   AdminFlags
}

// adminStore holds registered admins
var adminStore = make(map[uint64]*Admin)

// RegisterAdmin adds an admin to the system
func RegisterAdmin(steamID uint64, name string, flags AdminFlags) {
	adminStore[steamID] = &Admin{
		SteamID: steamID,
		Name:    name,
		Flags:   flags,
	}
}

// UnregisterAdmin removes an admin from the system
func UnregisterAdmin(steamID uint64) {
	delete(adminStore, steamID)
}

// GetAdmin retrieves an admin by Steam ID
func GetAdmin(steamID uint64) *Admin {
	return adminStore[steamID]
}

// GetAllAdmins returns all registered admins
func GetAllAdmins() []*Admin {
	admins := make([]*Admin, 0, len(adminStore))
	for _, admin := range adminStore {
		admins = append(admins, admin)
	}
	return admins
}

// ClearAdmins removes all registered admins
func ClearAdmins() {
	adminStore = make(map[uint64]*Admin)
}

// HasFlag checks if the admin has a specific flag
func (a *Admin) HasFlag(flag AdminFlags) bool {
	if a == nil {
		return false
	}
	// Root has all permissions
	if a.Flags&AdminRoot != 0 {
		return true
	}
	return a.Flags&flag != 0
}

// HasAnyFlag checks if the admin has any of the specified flags
func (a *Admin) HasAnyFlag(flags AdminFlags) bool {
	if a == nil {
		return false
	}
	if a.Flags&AdminRoot != 0 {
		return true
	}
	return a.Flags&flags != 0
}

// HasAllFlags checks if the admin has all of the specified flags
func (a *Admin) HasAllFlags(flags AdminFlags) bool {
	if a == nil {
		return false
	}
	if a.Flags&AdminRoot != 0 {
		return true
	}
	return a.Flags&flags == flags
}

// IsRoot checks if the admin has root (superadmin) access
func (a *Admin) IsRoot() bool {
	return a != nil && a.Flags&AdminRoot != 0
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
	admin := GetPlayerAdmin(player)
	return admin != nil && admin.HasFlag(flag)
}

// PlayerIsAdmin checks if a player is an admin (has any admin flags)
func PlayerIsAdmin(player *Player) bool {
	return GetPlayerAdmin(player) != nil
}

// CommandContext permission helpers

// IsAdmin checks if the command invoker is an admin
func (ctx *CommandContext) IsAdmin() bool {
	if ctx.IsFromConsole() {
		return true // Console is always admin
	}
	return PlayerIsAdmin(ctx.Player)
}

// HasFlag checks if the command invoker has a specific admin flag
func (ctx *CommandContext) HasFlag(flag AdminFlags) bool {
	if ctx.IsFromConsole() {
		return true // Console has all flags
	}
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
