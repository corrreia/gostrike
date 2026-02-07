// Package runtime provides the internal runtime for GoStrike.
// This file contains the main event and command dispatch logic.
package runtime

import (
	"sync"
)

// ============================================================
// Tick Dispatching
// ============================================================

type tickHandler func(deltaTime float64)

var (
	tickHandlers   []tickHandler
	tickHandlersMu sync.RWMutex
)

// RegisterTickHandler adds a tick handler
func RegisterTickHandler(handler tickHandler) {
	tickHandlersMu.Lock()
	defer tickHandlersMu.Unlock()
	tickHandlers = append(tickHandlers, handler)
}

// DispatchTick is called every server tick
func DispatchTick(deltaTime float64) {
	// Process timers first
	processTimers(deltaTime)

	// Then call tick handlers
	tickHandlersMu.RLock()
	handlers := tickHandlers
	tickHandlersMu.RUnlock()

	for _, handler := range handlers {
		handler(deltaTime)
	}
}

// ============================================================
// Event Dispatching
// ============================================================

// Event result constants
const (
	EventContinue = 0
	EventChanged  = 1
	EventHandled  = 2
	EventStop     = 3
)

// GameEventData is passed to game event handlers with access to native event fields
type GameEventData struct {
	Name      string
	NativePtr uintptr
	CanModify bool
}

type eventHandler func(data map[string]interface{}) int
type gameEventHandler func(event *GameEventData) int
type playerConnectHandler func(player *PlayerInfo) int
type playerDisconnectHandler func(slot int, reason string) int
type mapChangeHandler func(mapName string)
type entityCreatedHandler func(index uint32, classname string)
type entityDeletedHandler func(index uint32)
type damageHandler func(victimIdx, attackerIdx int, damage float32, damageType int) int

var (
	eventHandlers            = make(map[string][]eventHandler)
	gameEventHandlers        = make(map[string][]gameEventHandler)
	eventHandlersMu          sync.RWMutex
	playerConnectHandlers    []playerConnectHandler
	playerConnectHandlersMu  sync.RWMutex
	playerDisconnectHandlers []playerDisconnectHandler
	playerDisconnectMu       sync.RWMutex
	mapChangeHandlers        []mapChangeHandler
	mapChangeHandlersMu      sync.RWMutex
	entityCreatedHandlers    []entityCreatedHandler
	entityCreatedMu          sync.RWMutex
	entitySpawnedHandlers    []entityCreatedHandler
	entitySpawnedMu          sync.RWMutex
	entityDeletedHandlers    []entityDeletedHandler
	entityDeletedMu          sync.RWMutex
	damageHandlers           []damageHandler
	damageHandlersMu         sync.RWMutex
)

func initEvents() {
	eventHandlers = make(map[string][]eventHandler)
	gameEventHandlers = make(map[string][]gameEventHandler)
	playerConnectHandlers = nil
	playerDisconnectHandlers = nil
	mapChangeHandlers = nil
	entityCreatedHandlers = nil
	entitySpawnedHandlers = nil
	entityDeletedHandlers = nil
	damageHandlers = nil
}

func shutdownEvents() {
	eventHandlersMu.Lock()
	eventHandlers = make(map[string][]eventHandler)
	gameEventHandlers = make(map[string][]gameEventHandler)
	eventHandlersMu.Unlock()

	playerConnectHandlersMu.Lock()
	playerConnectHandlers = nil
	playerConnectHandlersMu.Unlock()

	playerDisconnectMu.Lock()
	playerDisconnectHandlers = nil
	playerDisconnectMu.Unlock()

	mapChangeHandlersMu.Lock()
	mapChangeHandlers = nil
	mapChangeHandlersMu.Unlock()

	entityCreatedMu.Lock()
	entityCreatedHandlers = nil
	entityCreatedMu.Unlock()

	entitySpawnedMu.Lock()
	entitySpawnedHandlers = nil
	entitySpawnedMu.Unlock()

	entityDeletedMu.Lock()
	entityDeletedHandlers = nil
	entityDeletedMu.Unlock()

	damageHandlersMu.Lock()
	damageHandlers = nil
	damageHandlersMu.Unlock()

	tickHandlersMu.Lock()
	tickHandlers = nil
	tickHandlersMu.Unlock()
}

// RegisterEventHandler registers a handler for a specific event
func RegisterEventHandler(eventName string, handler eventHandler, isPost bool) {
	eventHandlersMu.Lock()
	defer eventHandlersMu.Unlock()

	key := eventName
	if isPost {
		key = eventName + "_post"
	}

	eventHandlers[key] = append(eventHandlers[key], handler)
}

// RegisterPlayerConnectHandler registers a player connect handler
func RegisterPlayerConnectHandler(handler playerConnectHandler, isPost bool) {
	playerConnectHandlersMu.Lock()
	defer playerConnectHandlersMu.Unlock()
	playerConnectHandlers = append(playerConnectHandlers, handler)
}

// RegisterPlayerDisconnectHandler registers a player disconnect handler
func RegisterPlayerDisconnectHandler(handler playerDisconnectHandler, isPost bool) {
	playerDisconnectMu.Lock()
	defer playerDisconnectMu.Unlock()
	playerDisconnectHandlers = append(playerDisconnectHandlers, handler)
}

// RegisterMapChangeHandler registers a map change handler
func RegisterMapChangeHandler(handler mapChangeHandler) {
	mapChangeHandlersMu.Lock()
	defer mapChangeHandlersMu.Unlock()
	mapChangeHandlers = append(mapChangeHandlers, handler)
}

// RegisterGameEventHandler registers a handler for native game events with field access
func RegisterGameEventHandler(eventName string, handler gameEventHandler, isPost bool) {
	eventHandlersMu.Lock()
	defer eventHandlersMu.Unlock()

	key := eventName
	if isPost {
		key = eventName + "_post"
	}

	gameEventHandlers[key] = append(gameEventHandlers[key], handler)
}

// RegisterDamageHandler registers a handler for damage events
func RegisterDamageHandler(handler damageHandler) {
	damageHandlersMu.Lock()
	defer damageHandlersMu.Unlock()
	damageHandlers = append(damageHandlers, handler)
}

// DispatchEvent dispatches an event to registered handlers
func DispatchEvent(eventName string, nativeEvent uintptr, isPost bool) int {
	eventHandlersMu.RLock()
	key := eventName
	if isPost {
		key = eventName + "_post"
	}
	oldHandlers := eventHandlers[key]
	newHandlers := gameEventHandlers[key]
	eventHandlersMu.RUnlock()

	if len(oldHandlers) == 0 && len(newHandlers) == 0 {
		return EventContinue
	}

	result := EventContinue

	// Dispatch to new-style game event handlers (native field access)
	if len(newHandlers) > 0 {
		eventData := &GameEventData{
			Name:      eventName,
			NativePtr: nativeEvent,
			CanModify: !isPost,
		}
		for _, handler := range newHandlers {
			r := handler(eventData)
			if r > result {
				result = r
			}
			if result >= EventStop {
				return result
			}
		}
	}

	// Dispatch to old-style map-based handlers (backward compat)
	if len(oldHandlers) > 0 {
		data := make(map[string]interface{})
		data["_name"] = eventName
		data["_native"] = nativeEvent

		for _, handler := range oldHandlers {
			r := handler(data)
			if r > result {
				result = r
			}
			if result >= EventStop {
				break
			}
		}
	}

	return result
}

// DispatchTakeDamage dispatches a damage event to registered handlers
func DispatchTakeDamage(victimIdx, attackerIdx int, damage float32, damageType int) int {
	damageHandlersMu.RLock()
	handlers := damageHandlers
	damageHandlersMu.RUnlock()

	if len(handlers) == 0 {
		return EventContinue
	}

	result := EventContinue
	for _, handler := range handlers {
		r := handler(victimIdx, attackerIdx, damage, damageType)
		if r > result {
			result = r
		}
		if result >= EventStop {
			break
		}
	}

	return result
}

// DispatchPlayerConnect dispatches a player connect event
func DispatchPlayerConnect(player *PlayerInfo) {
	if player == nil {
		return
	}

	playerConnectHandlersMu.RLock()
	handlers := playerConnectHandlers
	playerConnectHandlersMu.RUnlock()

	for _, handler := range handlers {
		handler(player)
	}
}

// DispatchPlayerDisconnect dispatches a player disconnect event
func DispatchPlayerDisconnect(slot int, reason string) {
	playerDisconnectMu.RLock()
	handlers := playerDisconnectHandlers
	playerDisconnectMu.RUnlock()

	for _, handler := range handlers {
		handler(slot, reason)
	}
}

// DispatchMapChange dispatches a map change event
func DispatchMapChange(mapName string) {
	mapChangeHandlersMu.RLock()
	handlers := mapChangeHandlers
	mapChangeHandlersMu.RUnlock()

	for _, handler := range handlers {
		handler(mapName)
	}
}

// ============================================================
// Entity Lifecycle Dispatching
// ============================================================

// RegisterEntityCreatedHandler registers a handler called when an entity is created
func RegisterEntityCreatedHandler(handler entityCreatedHandler) {
	entityCreatedMu.Lock()
	defer entityCreatedMu.Unlock()
	entityCreatedHandlers = append(entityCreatedHandlers, handler)
}

// RegisterEntitySpawnedHandler registers a handler called when an entity is spawned
func RegisterEntitySpawnedHandler(handler entityCreatedHandler) {
	entitySpawnedMu.Lock()
	defer entitySpawnedMu.Unlock()
	entitySpawnedHandlers = append(entitySpawnedHandlers, handler)
}

// RegisterEntityDeletedHandler registers a handler called when an entity is deleted
func RegisterEntityDeletedHandler(handler entityDeletedHandler) {
	entityDeletedMu.Lock()
	defer entityDeletedMu.Unlock()
	entityDeletedHandlers = append(entityDeletedHandlers, handler)
}

// DispatchEntityCreated dispatches an entity created event
func DispatchEntityCreated(index uint32, classname string) {
	entityCreatedMu.RLock()
	handlers := entityCreatedHandlers
	entityCreatedMu.RUnlock()

	for _, handler := range handlers {
		handler(index, classname)
	}
}

// DispatchEntitySpawned dispatches an entity spawned event
func DispatchEntitySpawned(index uint32, classname string) {
	entitySpawnedMu.RLock()
	handlers := entitySpawnedHandlers
	entitySpawnedMu.RUnlock()

	for _, handler := range handlers {
		handler(index, classname)
	}
}

// DispatchEntityDeleted dispatches an entity deleted event
func DispatchEntityDeleted(index uint32) {
	entityDeletedMu.RLock()
	handlers := entityDeletedHandlers
	entityDeletedMu.RUnlock()

	for _, handler := range handlers {
		handler(index)
	}
}
