// Package shared provides shared types and function registrations
// to avoid import cycles between bridge, runtime, and manager packages.
package shared

// PlayerInfo contains player information
type PlayerInfo struct {
	Slot    int
	UserID  int
	SteamID uint64
	Name    string
	IP      string
	Team    int
	IsAlive bool
	IsBot   bool
	Health  int
	Armor   int
	PosX    float64
	PosY    float64
	PosZ    float64
}

// InitFunc is the type for initialization functions
type InitFunc func()

// ShutdownFunc is the type for shutdown functions
type ShutdownFunc func()

// TickFunc is the type for tick dispatch functions
type TickFunc func(deltaTime float64)

// EventFunc is the type for event dispatch functions
type EventFunc func(eventName string, nativeEvent uintptr, isPost bool) int

// CommandFunc is the type for command dispatch functions
type CommandFunc func(command, args string, playerSlot int) bool

// PlayerConnectFunc is the type for player connect dispatch functions
type PlayerConnectFunc func(player *PlayerInfo)

// PlayerDisconnectFunc is the type for player disconnect dispatch functions
type PlayerDisconnectFunc func(slot int, reason string)

// MapChangeFunc is the type for map change dispatch functions
type MapChangeFunc func(mapName string)

// Dispatch functions - set by runtime package
var (
	RuntimeInit              InitFunc
	RuntimeShutdown          ShutdownFunc
	DispatchTick             TickFunc
	DispatchEvent            EventFunc
	DispatchCommand          CommandFunc
	DispatchPlayerConnect    PlayerConnectFunc
	DispatchPlayerDisconnect PlayerDisconnectFunc
	DispatchMapChange        MapChangeFunc
)

// Manager functions - set by manager package
var (
	ManagerInit     InitFunc
	ManagerShutdown ShutdownFunc
)
