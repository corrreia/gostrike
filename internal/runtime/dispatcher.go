// Package runtime provides the internal runtime for GoStrike.
// This file contains the main event and command dispatch logic.
package runtime

import (
	"fmt"
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

type eventHandler func(data map[string]interface{}) int
type playerConnectHandler func(player *PlayerInfo) int
type playerDisconnectHandler func(slot int, reason string) int
type mapChangeHandler func(mapName string)

var (
	eventHandlers            = make(map[string][]eventHandler)
	eventHandlersMu          sync.RWMutex
	playerConnectHandlers    []playerConnectHandler
	playerConnectHandlersMu  sync.RWMutex
	playerDisconnectHandlers []playerDisconnectHandler
	playerDisconnectMu       sync.RWMutex
	mapChangeHandlers        []mapChangeHandler
	mapChangeHandlersMu      sync.RWMutex
)

func initEvents() {
	eventHandlers = make(map[string][]eventHandler)
	playerConnectHandlers = nil
	playerDisconnectHandlers = nil
	mapChangeHandlers = nil
}

func shutdownEvents() {
	eventHandlersMu.Lock()
	eventHandlers = make(map[string][]eventHandler)
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

// DispatchEvent dispatches an event to registered handlers
func DispatchEvent(eventName string, nativeEvent uintptr, isPost bool) int {
	eventHandlersMu.RLock()
	key := eventName
	if isPost {
		key = eventName + "_post"
	}
	handlers := eventHandlers[key]
	eventHandlersMu.RUnlock()

	if len(handlers) == 0 {
		return EventContinue
	}

	// Create event data map (would be populated from native event)
	data := make(map[string]interface{})
	data["_name"] = eventName
	data["_native"] = nativeEvent

	result := EventContinue
	for _, handler := range handlers {
		r := handler(data)
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
// Command Dispatching
// ============================================================

type commandHandler func(cmdName, argString string, playerSlot int) bool

type commandInfo struct {
	name        string
	description string
	handler     commandHandler
}

var (
	commands   = make(map[string]*commandInfo)
	commandsMu sync.RWMutex
)

func initCommands() {
	commands = make(map[string]*commandInfo)

	// Register built-in commands
	RegisterCommand("gostrike_status", "Show GoStrike status", handleStatusCommand)
	RegisterCommand("gostrike_plugins", "List loaded plugins", handlePluginsCommand)
	RegisterCommand("gostrike_test", "Test command", handleTestCommand)
}

// replyToCommand is set by bridge package to enable replies
var replyToCommand func(slot int, msg string)

// SetReplyFunc sets the reply function (called by bridge)
func SetReplyFunc(fn func(slot int, msg string)) {
	replyToCommand = fn
}

func reply(slot int, format string, args ...interface{}) {
	if replyToCommand != nil {
		replyToCommand(slot, fmt.Sprintf(format, args...))
	}
}

func shutdownCommands() {
	commandsMu.Lock()
	commands = make(map[string]*commandInfo)
	commandsMu.Unlock()
}

// RegisterCommand registers a new command
func RegisterCommand(name, description string, handler commandHandler) {
	commandsMu.Lock()
	defer commandsMu.Unlock()

	commands[name] = &commandInfo{
		name:        name,
		description: description,
		handler:     handler,
	}
}

// UnregisterCommand removes a command
func UnregisterCommand(name string) {
	commandsMu.Lock()
	defer commandsMu.Unlock()
	delete(commands, name)
}

// DispatchCommand dispatches a command to the appropriate handler
func DispatchCommand(cmdName, argString string, playerSlot int) bool {
	commandsMu.RLock()
	cmd, ok := commands[cmdName]
	commandsMu.RUnlock()

	if !ok {
		return false
	}

	return cmd.handler(cmdName, argString, playerSlot)
}

// GetCommands returns all registered commands
func GetCommands() map[string]string {
	commandsMu.RLock()
	defer commandsMu.RUnlock()

	result := make(map[string]string)
	for name, info := range commands {
		result[name] = info.description
	}
	return result
}

// Built-in command handlers

func handleStatusCommand(cmdName, argString string, playerSlot int) bool {
	reply(playerSlot, "=== GoStrike Status ===")
	reply(playerSlot, "Version: 0.1.0")
	reply(playerSlot, "ABI Version: 1")
	reply(playerSlot, "Runtime: Running")
	return true
}

func handlePluginsCommand(cmdName, argString string, playerSlot int) bool {
	reply(playerSlot, "=== GoStrike Plugins ===")
	reply(playerSlot, "(Plugin list will be shown here)")
	return true
}

func handleTestCommand(cmdName, argString string, playerSlot int) bool {
	reply(playerSlot, "GoStrike test command executed!")
	reply(playerSlot, "Arguments: %s", argString)
	reply(playerSlot, "Player slot: %d", playerSlot)
	return true
}
