package gostrike

import (
	"strconv"
	"strings"

	"github.com/corrreia/gostrike/internal/bridge"
	"github.com/corrreia/gostrike/internal/runtime"
	"github.com/corrreia/gostrike/internal/scope"
)

// CommandContext provides information about a chat command invocation
type CommandContext struct {
	Player    *Player  // The player who executed the command
	Command   string   // The command name (without ! prefix)
	Args      []string // Arguments passed to the command
	ArgString string   // Full argument string
	slot      int      // Internal: player slot
}

// Reply sends a response to the command invoker via chat
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

// ChatCommandCallback is the handler signature for chat commands
type ChatCommandCallback func(ctx *CommandContext) error

// ChatCommandInfo defines a chat command's metadata
type ChatCommandInfo struct {
	Name        string              // Command name (without ! prefix)
	Description string              // Help text
	Usage       string              // Usage string (e.g., "<player> <reason>")
	MinArgs     int                 // Minimum required arguments
	Permission  string              // Required permission (empty = public)
	Callback    ChatCommandCallback // Handler function
}

// RegisterChatCommand registers a new chat command (!command)
// Returns an error if a command with the same name is already registered
func RegisterChatCommand(info ChatCommandInfo) error {
	// Wrap the callback for the runtime
	handler := func(slot int, args []string) bool {
		// Get player
		player := GetServer().GetPlayerBySlot(slot)
		if player == nil {
			return false
		}

		// Build argument string for context
		argString := ""
		if len(args) > 0 {
			argString = strings.Join(args, " ")
		}

		// Create context
		ctx := &CommandContext{
			Player:    player,
			Command:   info.Name,
			Args:      args,
			ArgString: argString,
			slot:      slot,
		}

		// Check minimum arguments
		if len(args) < info.MinArgs {
			ctx.ReplyError("Usage: !%s %s", info.Name, info.Usage)
			return true
		}

		// Check permission
		if info.Permission != "" {
			if !HasPermission(player.SteamID, info.Permission) {
				ctx.ReplyError("You do not have permission to use this command")
				return true
			}
		}

		// Execute callback
		if err := info.Callback(ctx); err != nil {
			ctx.ReplyError("%v", err)
		}

		return true
	}

	// Register with runtime
	err := runtime.RegisterChatCommand(runtime.ChatCommand{
		Name:        info.Name,
		Description: info.Description,
		Usage:       info.Usage,
		MinArgs:     info.MinArgs,
		Permission:  info.Permission,
		Handler:     handler,
	})
	if err == nil {
		if s := scope.GetActive(); s != nil {
			s.TrackChatCommand(info.Name)
		}
	}
	return err
}

// ChatCommandExists checks if a chat command is already registered
func ChatCommandExists(name string) bool {
	return runtime.ChatCommandExists(name)
}

// UnregisterChatCommand removes a chat command
func UnregisterChatCommand(name string) {
	runtime.UnregisterChatCommand(name)
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
