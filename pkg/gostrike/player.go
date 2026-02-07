// Package gostrike provides the public SDK for GoStrike plugins.
package gostrike

import (
	"fmt"

	"github.com/corrreia/gostrike/internal/bridge"
	"github.com/corrreia/gostrike/internal/shared"
)

// Team represents a CS2 team
type Team int

const (
	TeamUnassigned Team = 0
	TeamSpectator  Team = 1
	TeamT          Team = 2
	TeamCT         Team = 3
)

// String returns the team name
func (t Team) String() string {
	switch t {
	case TeamUnassigned:
		return "Unassigned"
	case TeamSpectator:
		return "Spectator"
	case TeamT:
		return "Terrorist"
	case TeamCT:
		return "Counter-Terrorist"
	default:
		return "Unknown"
	}
}

// Vector3 represents a 3D position
type Vector3 struct {
	X float64
	Y float64
	Z float64
}

// Player represents a connected player
type Player struct {
	Slot     int
	UserID   int
	SteamID  uint64
	Name     string
	IP       string
	Team     Team
	IsAlive  bool
	IsBot    bool
	Health   int
	Armor    int
	Position Vector3
}

// playerFromInfo converts a shared.PlayerInfo to a Player
func playerFromInfo(info *shared.PlayerInfo) *Player {
	if info == nil {
		return nil
	}
	return &Player{
		Slot:    info.Slot,
		UserID:  info.UserID,
		SteamID: info.SteamID,
		Name:    info.Name,
		IP:      info.IP,
		Team:    Team(info.Team),
		IsAlive: info.IsAlive,
		IsBot:   info.IsBot,
		Health:  info.Health,
		Armor:   info.Armor,
		Position: Vector3{
			X: info.PosX,
			Y: info.PosY,
			Z: info.PosZ,
		},
	}
}

// playerFromBridgeInfo converts a bridge.PlayerInfo to a Player
func playerFromBridgeInfo(info *bridge.PlayerInfo) *Player {
	if info == nil {
		return nil
	}
	return &Player{
		Slot:    info.Slot,
		UserID:  info.UserID,
		SteamID: info.SteamID,
		Name:    info.Name,
		IP:      info.IP,
		Team:    Team(info.Team),
		IsAlive: info.IsAlive,
		IsBot:   info.IsBot,
		Health:  info.Health,
		Armor:   info.Armor,
		Position: Vector3{
			X: info.PosX,
			Y: info.PosY,
			Z: info.PosZ,
		},
	}
}

// Refresh updates the player's information from the server
func (p *Player) Refresh() bool {
	info := bridge.GetPlayer(p.Slot)
	if info == nil {
		return false
	}

	p.UserID = info.UserID
	p.SteamID = info.SteamID
	p.Name = info.Name
	p.IP = info.IP
	p.Team = Team(info.Team)
	p.IsAlive = info.IsAlive
	p.IsBot = info.IsBot
	p.Health = info.Health
	p.Armor = info.Armor
	p.Position.X = info.PosX
	p.Position.Y = info.PosY
	p.Position.Z = info.PosZ

	return true
}

// Kick removes the player from the server
func (p *Player) Kick(reason string) {
	bridge.KickPlayer(p.Slot, reason)
}

// PrintToChat sends a chat message to this player via UTIL_ClientPrint
func (p *Player) PrintToChat(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	bridge.ClientPrint(p.Slot, bridge.HudPrintTalk, msg)
}

// PrintToCenter shows a centered HUD message via UTIL_ClientPrint
func (p *Player) PrintToCenter(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	bridge.ClientPrint(p.Slot, bridge.HudPrintCenter, msg)
}

// PrintToConsole sends a console message to this player
func (p *Player) PrintToConsole(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	bridge.ClientPrint(p.Slot, bridge.HudPrintConsole, msg)
}

// PrintToAlert shows an alert HUD message to this player
func (p *Player) PrintToAlert(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	bridge.ClientPrint(p.Slot, bridge.HudPrintAlert, msg)
}

// ExecuteClientCommand executes a command as this player
// Note: This requires additional native support
func (p *Player) ExecuteClientCommand(cmd string) {
	// Would need native support to execute client commands
	bridge.LogWarning("Player", "ExecuteClientCommand not yet implemented")
}

// GetPosition returns the player's current position
func (p *Player) GetPosition() Vector3 {
	// Refresh to get current position
	p.Refresh()
	return p.Position
}

// IsValid returns true if the player is still connected
func (p *Player) IsValid() bool {
	info := bridge.GetPlayer(p.Slot)
	return info != nil && info.SteamID == p.SteamID
}

// IsInTeam checks if the player is on a specific team
func (p *Player) IsInTeam(team Team) bool {
	p.Refresh()
	return p.Team == team
}

// IsTerrorist returns true if the player is on the Terrorist team
func (p *Player) IsTerrorist() bool {
	return p.IsInTeam(TeamT)
}

// IsCounterTerrorist returns true if the player is on the CT team
func (p *Player) IsCounterTerrorist() bool {
	return p.IsInTeam(TeamCT)
}

// ============================================================
// Pawn/Controller Entity Access
// ============================================================

// GetController returns the CCSPlayerController entity for this player.
// Returns nil if no controller is found.
func (p *Player) GetController() *Entity {
	ptr := bridge.GetPlayerController(p.Slot)
	if ptr == 0 {
		return nil
	}
	classname := bridge.GetEntityClassname(ptr)
	index := bridge.GetEntityIndex(ptr)
	return &Entity{
		Index:     index,
		ClassName: classname,
		ptr:       ptr,
	}
}

// GetPawn returns the CCSPlayerPawn entity for this player.
// Returns nil if the player has no pawn (dead, spectating, disconnected).
func (p *Player) GetPawn() *Entity {
	ptr := bridge.GetPlayerPawn(p.Slot)
	if ptr == 0 {
		return nil
	}
	classname := bridge.GetEntityClassname(ptr)
	index := bridge.GetEntityIndex(ptr)
	return &Entity{
		Index:     index,
		ClassName: classname,
		ptr:       ptr,
	}
}

// ============================================================
// Game Functions
// ============================================================

// Respawn respawns this player
func (p *Player) Respawn() {
	bridge.PlayerRespawn(p.Slot)
}

// ChangeTeam changes this player's team
func (p *Player) ChangeTeam(team Team) {
	bridge.PlayerChangeTeam(p.Slot, int(team))
}

// Slay kills this player immediately
func (p *Player) Slay() {
	bridge.PlayerSlay(p.Slot)
}

// Teleport moves this player to the specified position
// Pass nil for any parameter you don't want to change.
func (p *Player) Teleport(pos *Vector3, angles *Vector3, velocity *Vector3) {
	var pPos, pAngles, pVelocity *[3]float32

	if pos != nil {
		arr := [3]float32{float32(pos.X), float32(pos.Y), float32(pos.Z)}
		pPos = &arr
	}
	if angles != nil {
		arr := [3]float32{float32(angles.X), float32(angles.Y), float32(angles.Z)}
		pAngles = &arr
	}
	if velocity != nil {
		arr := [3]float32{float32(velocity.X), float32(velocity.Y), float32(velocity.Z)}
		pVelocity = &arr
	}

	bridge.PlayerTeleport(p.Slot, pPos, pAngles, pVelocity)
}

// GiveWeapon gives a weapon or item to this player.
// Accepts names with or without the "weapon_" prefix (e.g. "ak47" or "weapon_ak47").
func (p *Player) GiveWeapon(name string) {
	bridge.GiveNamedItem(p.Slot, name)
}

// DropWeapons drops all weapons this player is carrying
func (p *Player) DropWeapons() {
	bridge.PlayerDropWeapons(p.Slot)
}

// SetHealth sets this player's health via schema
func (p *Player) SetHealth(health int) {
	pawn := p.GetPawn()
	if pawn != nil {
		pawn.SetPropInt("CBaseEntity", "m_iHealth", int32(health))
	}
}

// SetMaxHealth sets this player's max health via schema
func (p *Player) SetMaxHealth(health int) {
	pawn := p.GetPawn()
	if pawn != nil {
		pawn.SetPropInt("CBaseEntity", "m_iMaxHealth", int32(health))
	}
}

// SetArmor sets this player's armor value via schema
func (p *Player) SetArmor(armor int) {
	pawn := p.GetPawn()
	if pawn != nil {
		pawn.SetPropInt("CCSPlayerPawnBase", "m_ArmorValue", int32(armor))
	}
}
