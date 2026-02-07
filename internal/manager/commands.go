package manager

import (
	"fmt"
	"strings"

	"github.com/corrreia/gostrike/internal/runtime"
	"github.com/corrreia/gostrike/internal/shared"
)

const pluginManagementPermission = "gostrike.admin.plugins"

// RegisterPluginCommands registers the !plugin chat command.
func RegisterPluginCommands() {
	err := runtime.RegisterChatCommand(runtime.ChatCommand{
		Name:        "plugin",
		Description: "Manage plugins at runtime",
		Usage:       "<list|load|unload|reload|info> [slug]",
		MinArgs:     1,
		Permission:  pluginManagementPermission,
		Handler:     handlePluginCommand,
	})
	if err != nil {
		shared.LogWarning("PluginManager", "Failed to register !plugin command: %v", err)
	}
}

func handlePluginCommand(slot int, args []string) bool {
	action := strings.ToLower(args[0])

	switch action {
	case "list":
		list := GetPluginList()
		reply(slot, "Plugins (%d):", len(list))
		for _, p := range list {
			state := p.State
			if p.Error != "" {
				state += " (" + p.Error + ")"
			}
			reply(slot, "  %s [%s] - %s v%s", p.Slug, state, p.Name, p.Version)
		}
		return true

	case "load":
		if len(args) < 2 {
			reply(slot, "Usage: !plugin load <slug>")
			return true
		}
		if err := LoadPlugin(args[1]); err != nil {
			reply(slot, "[ERROR] %v", err)
		} else {
			reply(slot, "Plugin %s loaded successfully", args[1])
		}
		return true

	case "unload":
		if len(args) < 2 {
			reply(slot, "Usage: !plugin unload <slug>")
			return true
		}
		if err := UnloadPlugin(args[1]); err != nil {
			reply(slot, "[ERROR] %v", err)
		} else {
			reply(slot, "Plugin %s unloaded successfully", args[1])
		}
		return true

	case "reload":
		if len(args) < 2 {
			reply(slot, "Usage: !plugin reload <slug>")
			return true
		}
		if err := ReloadPluginBySlug(args[1]); err != nil {
			reply(slot, "[ERROR] %v", err)
		} else {
			reply(slot, "Plugin %s reloaded successfully", args[1])
		}
		return true

	case "info":
		if len(args) < 2 {
			reply(slot, "Usage: !plugin info <slug>")
			return true
		}
		info := GetPluginBySlug(args[1])
		if info == nil {
			reply(slot, "[ERROR] Plugin not found: %s", args[1])
		} else {
			reply(slot, "%s v%s by %s", info.Name, info.Version, info.Author)
			reply(slot, "  Slug: %s | State: %s", info.Slug, info.State.String())
			if info.Description != "" {
				reply(slot, "  %s", info.Description)
			}
			if info.LoadError != nil {
				reply(slot, "  Error: %v", info.LoadError)
			}
		}
		return true

	default:
		reply(slot, "Unknown action: %s. Use: list, load, unload, reload, info", action)
		return true
	}
}

func reply(slot int, format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	// Use the bridge reply if available, otherwise log
	if shared.DispatchPlayerConnect != nil {
		// We can't directly call bridge from internal/manager to avoid cycles.
		// Use the replyFunc callback instead.
		if replyFunc != nil {
			replyFunc(slot, msg)
		}
	}
}

// replyFunc is set by the runtime/bridge initialization to provide chat replies.
var replyFunc func(slot int, message string)

// SetReplyFunc sets the function used to reply to chat commands.
func SetReplyFunc(fn func(slot int, message string)) {
	replyFunc = fn
}
