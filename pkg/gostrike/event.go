// Package gostrike provides the public SDK for GoStrike plugins.
package gostrike

import (
	"github.com/corrreia/gostrike/internal/bridge"
	"github.com/corrreia/gostrike/internal/runtime"
	"github.com/corrreia/gostrike/internal/scope"
)

// HandlerID uniquely identifies a registered handler for later removal.
type HandlerID = runtime.HandlerID

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
// GameEvent — wraps a native IGameEvent* with field access
// ============================================================

// GameEvent provides access to native game event fields.
// In pre-hook mode, fields can be modified; in post-hook mode they are read-only.
type GameEvent struct {
	name      string
	nativePtr uintptr
	canModify bool
}

func (e *GameEvent) Name() string { return e.name }

// GetInt reads an int32 field from the event
func (e *GameEvent) GetInt(key string) int32 {
	return bridge.EventGetInt(e.nativePtr, key)
}

// GetFloat reads a float field from the event
func (e *GameEvent) GetFloat(key string) float32 {
	return bridge.EventGetFloat(e.nativePtr, key)
}

// GetBool reads a bool field from the event
func (e *GameEvent) GetBool(key string) bool {
	return bridge.EventGetBool(e.nativePtr, key)
}

// GetString reads a string field from the event
func (e *GameEvent) GetString(key string) string {
	return bridge.EventGetString(e.nativePtr, key)
}

// GetUint64 reads a uint64 field from the event
func (e *GameEvent) GetUint64(key string) uint64 {
	return bridge.EventGetUint64(e.nativePtr, key)
}

// SetInt writes an int32 field (pre-hook only)
func (e *GameEvent) SetInt(key string, value int32) {
	if e.canModify {
		bridge.EventSetInt(e.nativePtr, key, value)
	}
}

// SetFloat writes a float field (pre-hook only)
func (e *GameEvent) SetFloat(key string, value float32) {
	if e.canModify {
		bridge.EventSetFloat(e.nativePtr, key, value)
	}
}

// SetBool writes a bool field (pre-hook only)
func (e *GameEvent) SetBool(key string, value bool) {
	if e.canModify {
		bridge.EventSetBool(e.nativePtr, key, value)
	}
}

// SetString writes a string field (pre-hook only)
func (e *GameEvent) SetString(key string, value string) {
	if e.canModify {
		bridge.EventSetString(e.nativePtr, key, value)
	}
}

// ============================================================
// Typed Event Wrappers
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

// PlayerDeathEvent wraps a native player_death event with typed field access
type PlayerDeathEvent struct {
	*GameEvent
}

// Victim returns the player who died
func (e *PlayerDeathEvent) Victim() *Player {
	// userid in Source 2 contains the entity index in the lower bits
	slot := int(e.GetInt("userid") & 0xFF)
	return GetServer().GetPlayerBySlot(slot)
}

// Attacker returns the player who killed
func (e *PlayerDeathEvent) Attacker() *Player {
	slot := int(e.GetInt("attacker") & 0xFF)
	return GetServer().GetPlayerBySlot(slot)
}

// Weapon returns the weapon name used for the kill
func (e *PlayerDeathEvent) Weapon() string {
	return e.GetString("weapon")
}

// Headshot returns whether the kill was a headshot
func (e *PlayerDeathEvent) Headshot() bool {
	return e.GetBool("headshot")
}

// RoundStartEvent wraps a native round_start event
type RoundStartEvent struct {
	*GameEvent
}

// TimeLimit returns the round time limit
func (e *RoundStartEvent) TimeLimit() int32 {
	return e.GetInt("timelimit")
}

// FragLimit returns the frag limit
func (e *RoundStartEvent) FragLimit() int32 {
	return e.GetInt("fraglimit")
}

// RoundEndEvent wraps a native round_end event
type RoundEndEvent struct {
	*GameEvent
}

// Winner returns the winning team
func (e *RoundEndEvent) Winner() Team {
	return Team(e.GetInt("winner"))
}

// Reason returns the round end reason
func (e *RoundEndEvent) Reason() int32 {
	return e.GetInt("reason")
}

// Message returns the round end message
func (e *RoundEndEvent) Message() string {
	return e.GetString("message")
}

// BombPlantedEvent wraps a native bomb_planted event
type BombPlantedEvent struct {
	*GameEvent
}

// Player returns the player who planted the bomb
func (e *BombPlantedEvent) Player() *Player {
	slot := int(e.GetInt("userid") & 0xFF)
	return GetServer().GetPlayerBySlot(slot)
}

// Site returns the bomb site (0=A, 1=B)
func (e *BombPlantedEvent) Site() int32 {
	return e.GetInt("site")
}

// MapChangeEvent fires when the map changes
type MapChangeEvent struct {
	MapName string
}

func (e *MapChangeEvent) Name() string { return "map_change" }

// GenericEvent represents any game event (deprecated — use GameEvent instead)
type GenericEvent struct {
	EventName string
	Data      map[string]interface{}
}

func (e *GenericEvent) Name() string { return e.EventName }

// DamageInfo contains information about a damage event
type DamageInfo struct {
	VictimIndex   int
	AttackerIndex int
	Damage        float32
	DamageType    int
}

// ============================================================
// Event Handler Registration
// ============================================================

// EventHandlerFunc is the generic event handler type
type EventHandlerFunc func(event Event) EventResult

// PlayerConnectHandler handles player connect events
type PlayerConnectHandler func(event *PlayerConnectEvent) EventResult

// PlayerDisconnectHandler handles player disconnect events
type PlayerDisconnectHandler func(event *PlayerDisconnectEvent) EventResult

// GenericEventHandler handles any event type (deprecated — use GameEventHandler)
type GenericEventHandler func(eventName string, event Event) EventResult

// GameEventHandler handles native game events with field access
type GameEventHandler func(event *GameEvent) EventResult

// DamageHandler handles damage events
type DamageHandler func(info *DamageInfo) EventResult

// RegisterPlayerConnectHandler registers a handler for player connect events
func RegisterPlayerConnectHandler(handler PlayerConnectHandler, mode HookMode) HandlerID {
	id := runtime.RegisterPlayerConnectHandler(func(player *runtime.PlayerInfo) int {
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
	if s := scope.GetActive(); s != nil {
		s.TrackHandler(uint64(id))
	}
	return id
}

// RegisterPlayerDisconnectHandler registers a handler for player disconnect events
func RegisterPlayerDisconnectHandler(handler PlayerDisconnectHandler, mode HookMode) HandlerID {
	id := runtime.RegisterPlayerDisconnectHandler(func(slot int, reason string) int {
		event := &PlayerDisconnectEvent{Slot: slot, Reason: reason}
		return int(handler(event))
	}, mode == HookPost)
	if s := scope.GetActive(); s != nil {
		s.TrackHandler(uint64(id))
	}
	return id
}

// RegisterMapChangeHandler registers a handler for map change events
func RegisterMapChangeHandler(handler func(*MapChangeEvent) EventResult) HandlerID {
	id := runtime.RegisterMapChangeHandler(func(mapName string) {
		event := &MapChangeEvent{MapName: mapName}
		handler(event)
	})
	if s := scope.GetActive(); s != nil {
		s.TrackHandler(uint64(id))
	}
	return id
}

// RegisterGenericEventHandler registers a handler for any event by name (deprecated)
func RegisterGenericEventHandler(eventName string, handler GenericEventHandler, mode HookMode) HandlerID {
	id := runtime.RegisterEventHandler(eventName, func(data map[string]interface{}) int {
		event := &GenericEvent{EventName: eventName, Data: data}
		return int(handler(eventName, event))
	}, mode == HookPost)
	if s := scope.GetActive(); s != nil {
		s.TrackHandler(uint64(id))
	}
	return id
}

// RegisterGameEventHandler registers a handler for a native game event with field access.
// Use this for events like "player_death", "round_start", "bomb_planted", etc.
func RegisterGameEventHandler(eventName string, handler GameEventHandler, mode HookMode) HandlerID {
	id := runtime.RegisterGameEventHandler(eventName, func(event *runtime.GameEventData) int {
		ge := &GameEvent{
			name:      event.Name,
			nativePtr: event.NativePtr,
			canModify: event.CanModify,
		}
		return int(handler(ge))
	}, mode == HookPost)
	if s := scope.GetActive(); s != nil {
		s.TrackHandler(uint64(id))
	}
	return id
}

// RegisterPlayerDeathHandler registers a typed handler for player_death events
func RegisterPlayerDeathHandler(handler func(event *PlayerDeathEvent) EventResult, mode HookMode) HandlerID {
	return RegisterGameEventHandler("player_death", func(event *GameEvent) EventResult {
		return handler(&PlayerDeathEvent{GameEvent: event})
	}, mode)
}

// RegisterRoundStartHandler registers a typed handler for round_start events
func RegisterRoundStartHandler(handler func(event *RoundStartEvent) EventResult, mode HookMode) HandlerID {
	return RegisterGameEventHandler("round_start", func(event *GameEvent) EventResult {
		return handler(&RoundStartEvent{GameEvent: event})
	}, mode)
}

// RegisterRoundEndHandler registers a typed handler for round_end events
func RegisterRoundEndHandler(handler func(event *RoundEndEvent) EventResult, mode HookMode) HandlerID {
	return RegisterGameEventHandler("round_end", func(event *GameEvent) EventResult {
		return handler(&RoundEndEvent{GameEvent: event})
	}, mode)
}

// RegisterBombPlantedHandler registers a typed handler for bomb_planted events
func RegisterBombPlantedHandler(handler func(event *BombPlantedEvent) EventResult, mode HookMode) HandlerID {
	return RegisterGameEventHandler("bomb_planted", func(event *GameEvent) EventResult {
		return handler(&BombPlantedEvent{GameEvent: event})
	}, mode)
}

// RegisterDamageHandler registers a handler for damage events (TakeDamage hook)
func RegisterDamageHandler(handler DamageHandler) HandlerID {
	id := runtime.RegisterDamageHandler(func(victimIdx, attackerIdx int, damage float32, damageType int) int {
		info := &DamageInfo{
			VictimIndex:   victimIdx,
			AttackerIndex: attackerIdx,
			Damage:        damage,
			DamageType:    damageType,
		}
		return int(handler(info))
	})
	if s := scope.GetActive(); s != nil {
		s.TrackHandler(uint64(id))
	}
	return id
}

// UnregisterHandlerByID removes a registered handler by its ID
func UnregisterHandlerByID(id HandlerID) {
	runtime.UnregisterHandler(id)
}

// ============================================================
// Tick Handlers
// ============================================================

// TickHandler is called every server tick
type TickHandler func(deltaTime float64)

// RegisterTickHandler registers a function to be called every tick
func RegisterTickHandler(handler TickHandler) HandlerID {
	id := runtime.RegisterTickHandler(func(dt float64) {
		handler(dt)
	})
	if s := scope.GetActive(); s != nil {
		s.TrackHandler(uint64(id))
	}
	return id
}

