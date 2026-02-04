// Package gostrike provides the public SDK for GoStrike plugins.
package gostrike

import (
	"github.com/corrreia/gostrike/internal/runtime"
)

// EventResult determines how the event chain continues
type EventResult int

const (
	EventContinue EventResult = iota // Allow event to proceed
	EventChanged                     // Event data was modified
	EventHandled                     // Stop processing, but allow event
	EventStop                        // Cancel the event entirely
)

// HookMode determines when the handler is called
type HookMode int

const (
	HookPre  HookMode = iota // Before the event is processed
	HookPost                 // After the event is processed
)

// Event is the base interface for all game events
type Event interface {
	Name() string
}

// ============================================================
// Event Types
// ============================================================

// PlayerConnectEvent fires when a player connects
type PlayerConnectEvent struct {
	Player *Player
}

func (e *PlayerConnectEvent) Name() string { return "player_connect" }

// PlayerDisconnectEvent fires when a player disconnects
type PlayerDisconnectEvent struct {
	Slot   int
	Reason string
}

func (e *PlayerDisconnectEvent) Name() string { return "player_disconnect" }

// PlayerDeathEvent fires when a player dies
type PlayerDeathEvent struct {
	Victim     *Player
	Attacker   *Player
	Weapon     string
	Headshot   bool
	Penetrated bool
}

func (e *PlayerDeathEvent) Name() string { return "player_death" }

// RoundStartEvent fires at round start
type RoundStartEvent struct {
	TimeLimit int
	FragLimit int
}

func (e *RoundStartEvent) Name() string { return "round_start" }

// RoundEndEvent fires at round end
type RoundEndEvent struct {
	Winner Team
	Reason int
}

func (e *RoundEndEvent) Name() string { return "round_end" }

// MapChangeEvent fires when the map changes
type MapChangeEvent struct {
	MapName string
}

func (e *MapChangeEvent) Name() string { return "map_change" }

// GenericEvent represents any game event
type GenericEvent struct {
	EventName string
	Data      map[string]interface{}
}

func (e *GenericEvent) Name() string { return e.EventName }

// ============================================================
// Event Handler Registration
// ============================================================

// EventHandlerFunc is the generic event handler type
type EventHandlerFunc func(event Event) EventResult

// PlayerConnectHandler handles player connect events
type PlayerConnectHandler func(event *PlayerConnectEvent) EventResult

// PlayerDisconnectHandler handles player disconnect events
type PlayerDisconnectHandler func(event *PlayerDisconnectEvent) EventResult

// GenericEventHandler handles any event type
type GenericEventHandler func(eventName string, event Event) EventResult

// RegisterPlayerConnectHandler registers a handler for player connect events
func RegisterPlayerConnectHandler(handler PlayerConnectHandler, mode HookMode) {
	runtime.RegisterPlayerConnectHandler(func(player *runtime.PlayerInfo) int {
		p := &Player{
			Slot:    player.Slot,
			UserID:  player.UserID,
			SteamID: player.SteamID,
			Name:    player.Name,
			IP:      player.IP,
			Team:    Team(player.Team),
			IsAlive: player.IsAlive,
			IsBot:   player.IsBot,
			Health:  player.Health,
			Armor:   player.Armor,
			Position: Vector3{
				X: player.PosX,
				Y: player.PosY,
				Z: player.PosZ,
			},
		}
		event := &PlayerConnectEvent{Player: p}
		return int(handler(event))
	}, mode == HookPost)
}

// RegisterPlayerDisconnectHandler registers a handler for player disconnect events
func RegisterPlayerDisconnectHandler(handler PlayerDisconnectHandler, mode HookMode) {
	runtime.RegisterPlayerDisconnectHandler(func(slot int, reason string) int {
		event := &PlayerDisconnectEvent{Slot: slot, Reason: reason}
		return int(handler(event))
	}, mode == HookPost)
}

// RegisterMapChangeHandler registers a handler for map change events
func RegisterMapChangeHandler(handler func(*MapChangeEvent) EventResult) {
	runtime.RegisterMapChangeHandler(func(mapName string) {
		event := &MapChangeEvent{MapName: mapName}
		handler(event)
	})
}

// RegisterGenericEventHandler registers a handler for any event by name
func RegisterGenericEventHandler(eventName string, handler GenericEventHandler, mode HookMode) {
	runtime.RegisterEventHandler(eventName, func(data map[string]interface{}) int {
		event := &GenericEvent{EventName: eventName, Data: data}
		return int(handler(eventName, event))
	}, mode == HookPost)
}

// UnregisterEventHandler removes a registered event handler
// Note: Currently handlers cannot be individually unregistered, they are cleared on unload
func UnregisterEventHandler(eventName string) {
	// TODO: Implement individual handler removal
}

// ============================================================
// Tick Handlers
// ============================================================

// TickHandler is called every server tick
type TickHandler func(deltaTime float64)

// RegisterTickHandler registers a function to be called every tick
func RegisterTickHandler(handler TickHandler) {
	runtime.RegisterTickHandler(func(dt float64) {
		handler(dt)
	})
}

// UnregisterTickHandler removes a tick handler
func UnregisterTickHandler(handler TickHandler) {
	// TODO: Implement
}
