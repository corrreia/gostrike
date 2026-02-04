// Package gostrike provides the public SDK for GoStrike plugins.
package gostrike

import (
	"strconv"
	"strings"

	"github.com/corrreia/gostrike/internal/bridge"
	"github.com/corrreia/gostrike/internal/runtime"
)

// CommandContext provides information about a command invocation
type CommandContext struct {
	Player    *Player  // nil if executed from server console
	Command   string   // The command name
	Args      []string // Arguments passed to the command
	ArgString string   // Full argument string
	slot      int      // Internal: player slot (-1 for console)
}

// Reply sends a response to the command invoker
func (ctx *CommandContext) Reply(format string, args ...interface{}) {
	bridge.ReplyToCommandf(ctx.slot, format, args...)
}

// ReplyError sends an error response (prefixed with ERROR:)
func (ctx *CommandContext) ReplyError(format string, args ...interface{}) {
	ctx.Reply("[ERROR] "+format, args...)
}

// GetArg returns an argument by index, or empty string if not present
func (ctx *CommandContext) GetArg(index int) string {
	if index < 0 || index >= len(ctx.Args) {
		return ""
	}
	return ctx.Args[index]
}

// GetArgInt returns an argument as an integer
func (ctx *CommandContext) GetArgInt(index int, defaultVal int) int {
	arg := ctx.GetArg(index)
	if arg == "" {
		return defaultVal
	}
	val, err := strconv.Atoi(arg)
	if err != nil {
		return defaultVal
	}
	return val
}

// GetArgFloat returns an argument as a float
func (ctx *CommandContext) GetArgFloat(index int, defaultVal float64) float64 {
	arg := ctx.GetArg(index)
	if arg == "" {
		return defaultVal
	}
	val, err := strconv.ParseFloat(arg, 64)
	if err != nil {
		return defaultVal
	}
	return val
}

// GetArgBool returns an argument as a boolean
func (ctx *CommandContext) GetArgBool(index int, defaultVal bool) bool {
	arg := strings.ToLower(ctx.GetArg(index))
	switch arg {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	default:
		return defaultVal
	}
}

// IsFromConsole returns true if the command was executed from the server console
func (ctx *CommandContext) IsFromConsole() bool {
	return ctx.Player == nil
}

// CommandFlags define command behavior and permissions
type CommandFlags int

const (
	CmdServer CommandFlags = 1 << iota // Executable from server console
	CmdClient                          // Executable from client console
	CmdChat                            // Executable from chat (! or /)
	CmdAdmin                           // Requires admin permission

	// Common combinations
	CmdAll = CmdServer | CmdClient | CmdChat
)

// CommandCallback is the handler signature for commands
type CommandCallback func(ctx *CommandContext) error

// CommandInfo defines a command's metadata
type CommandInfo struct {
	Name        string          // Command name (e.g., "gostrike_test")
	Description string          // Help text
	Usage       string          // Usage string (e.g., "<player> <reason>")
	MinArgs     int             // Minimum required arguments
	Flags       CommandFlags    // Permission flags
	Callback    CommandCallback // Handler function
}

// RegisterCommand registers a new server command
func RegisterCommand(info CommandInfo) error {
	// Wrap the callback for the runtime
	handler := func(cmdName, argString string, playerSlot int) bool {
		// Parse arguments
		args := parseArgs(argString)

		// Get player if not console
		var player *Player
		if playerSlot >= 0 {
			player = GetServer().GetPlayerBySlot(playerSlot)
		}

		// Check minimum arguments
		if len(args) < info.MinArgs {
			ctx := &CommandContext{
				Player:    player,
				Command:   cmdName,
				Args:      args,
				ArgString: argString,
				slot:      playerSlot,
			}
			ctx.ReplyError("Usage: %s %s", info.Name, info.Usage)
			return true
		}

		// Check flags
		if playerSlot < 0 && (info.Flags&CmdServer) == 0 {
			return false // Console not allowed
		}
		if playerSlot >= 0 && (info.Flags&(CmdClient|CmdChat)) == 0 {
			return false // Client not allowed
		}

		// Create context and call handler
		ctx := &CommandContext{
			Player:    player,
			Command:   cmdName,
			Args:      args,
			ArgString: argString,
			slot:      playerSlot,
		}

		if err := info.Callback(ctx); err != nil {
			ctx.ReplyError("%v", err)
		}

		return true
	}

	runtime.RegisterCommand(info.Name, info.Description, handler)
	return nil
}

// UnregisterCommand removes a registered command
func UnregisterCommand(name string) {
	runtime.UnregisterCommand(name)
}

// parseArgs splits an argument string into individual arguments
// Handles quoted strings
func parseArgs(argString string) []string {
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
				// End quote
				inQuote = false
				if current.Len() > 0 {
					args = append(args, current.String())
					current.Reset()
				}
			} else if !inQuote {
				// Start quote
				inQuote = true
				quoteChar = ch
			} else {
				// Quote char inside different quote
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

	// Add remaining
	if current.Len() > 0 {
		args = append(args, current.String())
	}

	return args
}
