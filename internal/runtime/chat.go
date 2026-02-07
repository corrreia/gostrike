// Package runtime provides the internal runtime for GoStrike.
// This file contains the chat command system for !-prefixed commands.
package runtime

import (
	"fmt"
	"strings"
	"sync"

	"github.com/corrreia/gostrike/internal/shared"
)

// ChatCommand represents a registered chat command
type ChatCommand struct {
	Name        string
	Description string
	Usage       string
	MinArgs     int
	Permission  string // Required permission (empty = public)
	Handler     ChatCommandHandler
}

// ChatCommandHandler is the handler signature for chat commands
type ChatCommandHandler func(slot int, args []string) bool

var (
	chatCommands   = make(map[string]*ChatCommand)
	chatCommandsMu sync.RWMutex
)

// RegisterChatCommand registers a new chat command (! prefix)
// Returns an error if a command with the same name is already registered
func RegisterChatCommand(cmd ChatCommand) error {
	chatCommandsMu.Lock()
	defer chatCommandsMu.Unlock()

	// Normalize command name (lowercase, no ! prefix)
	name := strings.ToLower(strings.TrimPrefix(cmd.Name, "!"))

	// Check for collision
	if _, exists := chatCommands[name]; exists {
		return fmt.Errorf("chat command '!%s' is already registered", name)
	}

	cmd.Name = name
	chatCommands[name] = &cmd

	shared.LogDebug("ChatCmd", "Registered chat command: !%s", name)
	return nil
}

// UnregisterChatCommand removes a chat command
func UnregisterChatCommand(name string) {
	chatCommandsMu.Lock()
	defer chatCommandsMu.Unlock()

	name = strings.ToLower(strings.TrimPrefix(name, "!"))
	delete(chatCommands, name)
}

// ChatCommandExists checks if a chat command is already registered
func ChatCommandExists(name string) bool {
	chatCommandsMu.RLock()
	defer chatCommandsMu.RUnlock()

	name = strings.ToLower(strings.TrimPrefix(name, "!"))
	_, exists := chatCommands[name]
	return exists
}

// GetChatCommands returns all registered chat commands
func GetChatCommands() map[string]*ChatCommand {
	chatCommandsMu.RLock()
	defer chatCommandsMu.RUnlock()

	result := make(map[string]*ChatCommand)
	for name, cmd := range chatCommands {
		result[name] = cmd
	}
	return result
}

// DispatchChatCommand handles an incoming chat message and checks for commands
// Returns true if a command was processed (suppress the chat message)
func DispatchChatCommand(playerSlot int, message string) bool {
	// Must start with !
	if !strings.HasPrefix(message, "!") {
		return false
	}

	// Parse command and args
	message = strings.TrimPrefix(message, "!")
	parts := strings.SplitN(strings.TrimSpace(message), " ", 2)
	if len(parts) == 0 || parts[0] == "" {
		return false
	}

	cmdName := strings.ToLower(parts[0])
	argString := ""
	if len(parts) > 1 {
		argString = parts[1]
	}

	// Look up command
	chatCommandsMu.RLock()
	cmd, ok := chatCommands[cmdName]
	chatCommandsMu.RUnlock()

	if !ok {
		shared.LogDebug("ChatCmd", "Unknown chat command: !%s (from slot %d)", cmdName, playerSlot)
		return false
	}

	// Parse args
	args := parseChatArgs(argString)

	// Check minimum args
	if len(args) < cmd.MinArgs {
		// Could send a usage message here
		shared.LogDebug("ChatCmd", "Not enough args for !%s: got %d, need %d", cmdName, len(args), cmd.MinArgs)
		return true // Still consume the command
	}

	// Permission checks are handled by the SDK layer (pkg/gostrike/command.go)

	// Execute handler
	shared.LogDebug("ChatCmd", "Executing chat command: !%s (slot %d, args: %v)", cmdName, playerSlot, args)
	return cmd.Handler(playerSlot, args)
}

// parseChatArgs splits an argument string, handling quoted strings
func parseChatArgs(argString string) []string {
	if argString == "" {
		return nil
	}

	var args []string
	var current strings.Builder
	inQuote := false
	quoteChar := rune(0)

	for _, ch := range argString {
		switch {
		case ch == '"' || ch == '\'':
			if inQuote && ch == quoteChar {
				inQuote = false
				if current.Len() > 0 {
					args = append(args, current.String())
					current.Reset()
				}
			} else if !inQuote {
				inQuote = true
				quoteChar = ch
			} else {
				current.WriteRune(ch)
			}
		case ch == ' ' || ch == '\t':
			if inQuote {
				current.WriteRune(ch)
			} else if current.Len() > 0 {
				args = append(args, current.String())
				current.Reset()
			}
		default:
			current.WriteRune(ch)
		}
	}

	if current.Len() > 0 {
		args = append(args, current.String())
	}

	return args
}

// initChatCommands initializes the chat command system
func initChatCommands() {
	chatCommands = make(map[string]*ChatCommand)
	shared.LogDebug("ChatCmd", "Chat command system initialized")
}

// shutdownChatCommands cleans up the chat command system
func shutdownChatCommands() {
	chatCommandsMu.Lock()
	chatCommands = make(map[string]*ChatCommand)
	chatCommandsMu.Unlock()
}
