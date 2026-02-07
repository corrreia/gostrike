// Package gostrike provides the public SDK for GoStrike plugins.
package gostrike

import (
	"fmt"

	"github.com/corrreia/gostrike/internal/bridge"
)

// Server represents the CS2 dedicated server instance
type Server struct{}

// global server instance
var server = &Server{}

// GetServer returns the server instance
func GetServer() *Server {
	return server
}

// GetMaxPlayers returns the server's max player count
func (s *Server) GetMaxPlayers() int {
	return bridge.GetMaxPlayers()
}

// GetMapName returns the current map name
func (s *Server) GetMapName() string {
	return bridge.GetMapName()
}

// GetTickRate returns the server tick rate
func (s *Server) GetTickRate() int {
	return bridge.GetTickRate()
}

// GetPlayers returns all connected players
func (s *Server) GetPlayers() []*Player {
	infos := bridge.GetAllPlayerInfos()
	if infos == nil {
		return nil
	}

	players := make([]*Player, len(infos))
	for i, info := range infos {
		players[i] = playerFromBridgeInfo(info)
	}
	return players
}

// GetPlayerBySlot returns a player by their slot index
func (s *Server) GetPlayerBySlot(slot int) *Player {
	info := bridge.GetPlayer(slot)
	if info == nil {
		return nil
	}
	return playerFromBridgeInfo(info)
}

// GetPlayerBySteamID returns a player by their Steam ID
func (s *Server) GetPlayerBySteamID(steamID uint64) *Player {
	for _, player := range s.GetPlayers() {
		if player.SteamID == steamID {
			return player
		}
	}
	return nil
}

// GetPlayerCount returns the number of connected players
func (s *Server) GetPlayerCount() int {
	return bridge.GetPlayerCount()
}

// ExecuteCommand executes a server console command
func (s *Server) ExecuteCommand(cmd string) {
	bridge.ExecuteServerCommand(cmd)
}

// PrintToConsole prints a message to the server console
func (s *Server) PrintToConsole(format string, args ...interface{}) {
	bridge.LogInfo("Server", format, args...)
}

// PrintToAll sends a chat message to all players via UTIL_ClientPrintAll
func (s *Server) PrintToAll(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	bridge.ClientPrintAll(bridge.HudPrintTalk, msg)
}

// PrintToCenterAll shows a centered HUD message to all players
func (s *Server) PrintToCenterAll(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	bridge.ClientPrintAll(bridge.HudPrintCenter, msg)
}

// PrintToConsoleAll sends a console message to all players
func (s *Server) PrintToConsoleAll(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	bridge.ClientPrintAll(bridge.HudPrintConsole, msg)
}
