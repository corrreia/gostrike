// Package runtime provides the internal runtime for GoStrike.
// This file contains the main event and command dispatch logic.
package runtime

import (
	"sync"
	"sync/atomic"
)

// ============================================================
// Handler Identity
// ============================================================

// HandlerID uniquely identifies a registered handler for later removal.
type HandlerID uint64

var nextHandlerID uint64 // atomic

func newHandlerID() HandlerID {
	return HandlerID(atomic.AddUint64(&nextHandlerID, 1))
}

// ============================================================
// Tick Dispatching
// ============================================================

type tickHandler func(deltaTime float64)

type tickHandlerEntry struct {
	id      HandlerID
	handler tickHandler
}

var (
	tickHandlers   []tickHandlerEntry
	tickHandlersMu sync.RWMutex
)

// RegisterTickHandler adds a tick handler and returns its HandlerID.
func RegisterTickHandler(handler tickHandler) HandlerID {
	id := newHandlerID()
	tickHandlersMu.Lock()
	defer tickHandlersMu.Unlock()
	tickHandlers = append(tickHandlers, tickHandlerEntry{id: id, handler: handler})
	return id
}

// UnregisterTickHandler removes a tick handler by ID.
func UnregisterTickHandler(id HandlerID) {
	tickHandlersMu.Lock()
	defer tickHandlersMu.Unlock()
	tickHandlers = removeEntry(tickHandlers, id)
}

// DispatchTick is called every server tick
func DispatchTick(deltaTime float64) {
	// Process timers first
	processTimers(deltaTime)

	// Then call tick handlers
	tickHandlersMu.RLock()
	handlers := tickHandlers
	tickHandlersMu.RUnlock()

	for _, entry := range handlers {
		entry.handler(deltaTime)
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

type eventHandlerEntry struct {
	id      HandlerID
	handler eventHandler
}

type gameEventHandlerEntry struct {
	id      HandlerID
	handler gameEventHandler
}

type playerConnectHandlerEntry struct {
	id      HandlerID
	handler playerConnectHandler
}

type playerDisconnectHandlerEntry struct {
	id      HandlerID
	handler playerDisconnectHandler
}

type mapChangeHandlerEntry struct {
	id      HandlerID
	handler mapChangeHandler
}

type entityCreatedHandlerEntry struct {
	id      HandlerID
	handler entityCreatedHandler
}

type entityDeletedHandlerEntry struct {
	id      HandlerID
	handler entityDeletedHandler
}

type damageHandlerEntry struct {
	id      HandlerID
	handler damageHandler
}

var (
	eventHandlers            = make(map[string][]eventHandlerEntry)
	gameEventHandlers        = make(map[string][]gameEventHandlerEntry)
	eventHandlersMu          sync.RWMutex
	playerConnectHandlers    []playerConnectHandlerEntry
	playerConnectHandlersMu  sync.RWMutex
	playerDisconnectHandlers []playerDisconnectHandlerEntry
	playerDisconnectMu       sync.RWMutex
	mapChangeHandlers        []mapChangeHandlerEntry
	mapChangeHandlersMu      sync.RWMutex
	entityCreatedHandlers    []entityCreatedHandlerEntry
	entityCreatedMu          sync.RWMutex
	entitySpawnedHandlers    []entityCreatedHandlerEntry
	entitySpawnedMu          sync.RWMutex
	entityDeletedHandlers    []entityDeletedHandlerEntry
	entityDeletedMu          sync.RWMutex
	damageHandlers           []damageHandlerEntry
	damageHandlersMu         sync.RWMutex
)

func initEvents() {
	eventHandlers = make(map[string][]eventHandlerEntry)
	gameEventHandlers = make(map[string][]gameEventHandlerEntry)
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
	eventHandlers = make(map[string][]eventHandlerEntry)
	gameEventHandlers = make(map[string][]gameEventHandlerEntry)
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

// ============================================================
// Registration Functions (all return HandlerID)
// ============================================================

// RegisterEventHandler registers a handler for a specific event
func RegisterEventHandler(eventName string, handler eventHandler, isPost bool) HandlerID {
	id := newHandlerID()
	eventHandlersMu.Lock()
	defer eventHandlersMu.Unlock()

	key := eventName
	if isPost {
		key = eventName + "_post"
	}

	eventHandlers[key] = append(eventHandlers[key], eventHandlerEntry{id: id, handler: handler})
	return id
}

// RegisterGameEventHandler registers a handler for native game events with field access
func RegisterGameEventHandler(eventName string, handler gameEventHandler, isPost bool) HandlerID {
	id := newHandlerID()
	eventHandlersMu.Lock()
	defer eventHandlersMu.Unlock()

	key := eventName
	if isPost {
		key = eventName + "_post"
	}

	gameEventHandlers[key] = append(gameEventHandlers[key], gameEventHandlerEntry{id: id, handler: handler})
	return id
}

// RegisterPlayerConnectHandler registers a player connect handler
func RegisterPlayerConnectHandler(handler playerConnectHandler, isPost bool) HandlerID {
	id := newHandlerID()
	playerConnectHandlersMu.Lock()
	defer playerConnectHandlersMu.Unlock()
	playerConnectHandlers = append(playerConnectHandlers, playerConnectHandlerEntry{id: id, handler: handler})
	return id
}

// RegisterPlayerDisconnectHandler registers a player disconnect handler
func RegisterPlayerDisconnectHandler(handler playerDisconnectHandler, isPost bool) HandlerID {
	id := newHandlerID()
	playerDisconnectMu.Lock()
	defer playerDisconnectMu.Unlock()
	playerDisconnectHandlers = append(playerDisconnectHandlers, playerDisconnectHandlerEntry{id: id, handler: handler})
	return id
}

// RegisterMapChangeHandler registers a map change handler
func RegisterMapChangeHandler(handler mapChangeHandler) HandlerID {
	id := newHandlerID()
	mapChangeHandlersMu.Lock()
	defer mapChangeHandlersMu.Unlock()
	mapChangeHandlers = append(mapChangeHandlers, mapChangeHandlerEntry{id: id, handler: handler})
	return id
}

// RegisterDamageHandler registers a handler for damage events
func RegisterDamageHandler(handler damageHandler) HandlerID {
	id := newHandlerID()
	damageHandlersMu.Lock()
	defer damageHandlersMu.Unlock()
	damageHandlers = append(damageHandlers, damageHandlerEntry{id: id, handler: handler})
	return id
}

// RegisterEntityCreatedHandler registers a handler called when an entity is created
func RegisterEntityCreatedHandler(handler entityCreatedHandler) HandlerID {
	id := newHandlerID()
	entityCreatedMu.Lock()
	defer entityCreatedMu.Unlock()
	entityCreatedHandlers = append(entityCreatedHandlers, entityCreatedHandlerEntry{id: id, handler: handler})
	return id
}

// RegisterEntitySpawnedHandler registers a handler called when an entity is spawned
func RegisterEntitySpawnedHandler(handler entityCreatedHandler) HandlerID {
	id := newHandlerID()
	entitySpawnedMu.Lock()
	defer entitySpawnedMu.Unlock()
	entitySpawnedHandlers = append(entitySpawnedHandlers, entityCreatedHandlerEntry{id: id, handler: handler})
	return id
}

// RegisterEntityDeletedHandler registers a handler called when an entity is deleted
func RegisterEntityDeletedHandler(handler entityDeletedHandler) HandlerID {
	id := newHandlerID()
	entityDeletedMu.Lock()
	defer entityDeletedMu.Unlock()
	entityDeletedHandlers = append(entityDeletedHandlers, entityDeletedHandlerEntry{id: id, handler: handler})
	return id
}

// ============================================================
// Unregistration Functions
// ============================================================

// UnregisterEventHandler removes an event handler by ID.
func UnregisterEventHandler(id HandlerID) {
	eventHandlersMu.Lock()
	defer eventHandlersMu.Unlock()
	for key, entries := range eventHandlers {
		eventHandlers[key] = removeEntry(entries, id)
	}
}

// UnregisterGameEventHandler removes a game event handler by ID.
func UnregisterGameEventHandler(id HandlerID) {
	eventHandlersMu.Lock()
	defer eventHandlersMu.Unlock()
	for key, entries := range gameEventHandlers {
		gameEventHandlers[key] = removeEntry(entries, id)
	}
}

// UnregisterPlayerConnectHandler removes a player connect handler by ID.
func UnregisterPlayerConnectHandler(id HandlerID) {
	playerConnectHandlersMu.Lock()
	defer playerConnectHandlersMu.Unlock()
	playerConnectHandlers = removeEntry(playerConnectHandlers, id)
}

// UnregisterPlayerDisconnectHandler removes a player disconnect handler by ID.
func UnregisterPlayerDisconnectHandler(id HandlerID) {
	playerDisconnectMu.Lock()
	defer playerDisconnectMu.Unlock()
	playerDisconnectHandlers = removeEntry(playerDisconnectHandlers, id)
}

// UnregisterMapChangeHandler removes a map change handler by ID.
func UnregisterMapChangeHandler(id HandlerID) {
	mapChangeHandlersMu.Lock()
	defer mapChangeHandlersMu.Unlock()
	mapChangeHandlers = removeEntry(mapChangeHandlers, id)
}

// UnregisterDamageHandler removes a damage handler by ID.
func UnregisterDamageHandler(id HandlerID) {
	damageHandlersMu.Lock()
	defer damageHandlersMu.Unlock()
	damageHandlers = removeEntry(damageHandlers, id)
}

// UnregisterEntityCreatedHandler removes an entity created handler by ID.
func UnregisterEntityCreatedHandler(id HandlerID) {
	entityCreatedMu.Lock()
	defer entityCreatedMu.Unlock()
	entityCreatedHandlers = removeEntry(entityCreatedHandlers, id)
}

// UnregisterEntitySpawnedHandler removes an entity spawned handler by ID.
func UnregisterEntitySpawnedHandler(id HandlerID) {
	entitySpawnedMu.Lock()
	defer entitySpawnedMu.Unlock()
	entitySpawnedHandlers = removeEntry(entitySpawnedHandlers, id)
}

// UnregisterEntityDeletedHandler removes an entity deleted handler by ID.
func UnregisterEntityDeletedHandler(id HandlerID) {
	entityDeletedMu.Lock()
	defer entityDeletedMu.Unlock()
	entityDeletedHandlers = removeEntry(entityDeletedHandlers, id)
}

// UnregisterHandler removes a handler by ID from whichever handler list it belongs to.
func UnregisterHandler(id HandlerID) {
	UnregisterTickHandler(id)
	UnregisterEventHandler(id)
	UnregisterGameEventHandler(id)
	UnregisterPlayerConnectHandler(id)
	UnregisterPlayerDisconnectHandler(id)
	UnregisterMapChangeHandler(id)
	UnregisterDamageHandler(id)
	UnregisterEntityCreatedHandler(id)
	UnregisterEntitySpawnedHandler(id)
	UnregisterEntityDeletedHandler(id)
}

// UnregisterHandlers removes multiple handlers by their IDs.
func UnregisterHandlers(ids []HandlerID) {
	for _, id := range ids {
		UnregisterHandler(id)
	}
}

// ============================================================
// Generic entry removal helper
// ============================================================

// idEntry is a constraint for any entry type that has an ID field.
type idEntry interface {
	getID() HandlerID
}

func (e tickHandlerEntry) getID() HandlerID             { return e.id }
func (e eventHandlerEntry) getID() HandlerID            { return e.id }
func (e gameEventHandlerEntry) getID() HandlerID        { return e.id }
func (e playerConnectHandlerEntry) getID() HandlerID    { return e.id }
func (e playerDisconnectHandlerEntry) getID() HandlerID { return e.id }
func (e mapChangeHandlerEntry) getID() HandlerID        { return e.id }
func (e entityCreatedHandlerEntry) getID() HandlerID    { return e.id }
func (e entityDeletedHandlerEntry) getID() HandlerID    { return e.id }
func (e damageHandlerEntry) getID() HandlerID           { return e.id }

// removeEntry removes an entry with the given ID using swap-delete.
func removeEntry[T idEntry](entries []T, id HandlerID) []T {
	for i, entry := range entries {
		if entry.getID() == id {
			last := len(entries) - 1
			entries[i] = entries[last]
			return entries[:last]
		}
	}
	return entries
}

// ============================================================
// Dispatch Functions
// ============================================================

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
		for _, entry := range newHandlers {
			r := entry.handler(eventData)
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

		for _, entry := range oldHandlers {
			r := entry.handler(data)
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
	for _, entry := range handlers {
		r := entry.handler(victimIdx, attackerIdx, damage, damageType)
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

	for _, entry := range handlers {
		entry.handler(player)
	}
}

// DispatchPlayerDisconnect dispatches a player disconnect event
func DispatchPlayerDisconnect(slot int, reason string) {
	playerDisconnectMu.RLock()
	handlers := playerDisconnectHandlers
	playerDisconnectMu.RUnlock()

	for _, entry := range handlers {
		entry.handler(slot, reason)
	}
}

// DispatchMapChange dispatches a map change event
func DispatchMapChange(mapName string) {
	mapChangeHandlersMu.RLock()
	handlers := mapChangeHandlers
	mapChangeHandlersMu.RUnlock()

	for _, entry := range handlers {
		entry.handler(mapName)
	}
}

// ============================================================
// Entity Lifecycle Dispatching
// ============================================================

// DispatchEntityCreated dispatches an entity created event
func DispatchEntityCreated(index uint32, classname string) {
	entityCreatedMu.RLock()
	handlers := entityCreatedHandlers
	entityCreatedMu.RUnlock()

	for _, entry := range handlers {
		entry.handler(index, classname)
	}
}

// DispatchEntitySpawned dispatches an entity spawned event
func DispatchEntitySpawned(index uint32, classname string) {
	entitySpawnedMu.RLock()
	handlers := entitySpawnedHandlers
	entitySpawnedMu.RUnlock()

	for _, entry := range handlers {
		entry.handler(index, classname)
	}
}

// DispatchEntityDeleted dispatches an entity deleted event
func DispatchEntityDeleted(index uint32) {
	entityDeletedMu.RLock()
	handlers := entityDeletedHandlers
	entityDeletedMu.RUnlock()

	for _, entry := range handlers {
		entry.handler(index)
	}
}
