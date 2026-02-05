// Package example provides an example GoStrike plugin demonstrating
// how to use the SDK features: chat commands, events, timers, and logging.
package example

import (
	"github.com/corrreia/gostrike/pkg/gostrike"
	"github.com/corrreia/gostrike/pkg/plugin"
)

// ExamplePlugin demonstrates GoStrike SDK usage
type ExamplePlugin struct {
	plugin.BasePlugin
	logger      gostrike.Logger
	playerCount int
	greetTimer  *gostrike.Timer
}

// Name returns the plugin name
func (p *ExamplePlugin) Name() string {
	return "Example Plugin"
}

// Version returns the plugin version
func (p *ExamplePlugin) Version() string {
	return "1.0.0"
}

// Author returns the plugin author
func (p *ExamplePlugin) Author() string {
	return "GoStrike Team"
}

// Description returns the plugin description
func (p *ExamplePlugin) Description() string {
	return "An example plugin demonstrating GoStrike SDK features"
}

// Load is called when the plugin is loaded
func (p *ExamplePlugin) Load(hotReload bool) error {
	p.logger = gostrike.GetLogger("Example")
	p.logger.Info("Loading example plugin (hotReload=%v)", hotReload)

	// Register chat commands
	p.registerChatCommands()

	// Register event handlers
	p.registerEventHandlers()

	// Start a repeating timer
	p.greetTimer = gostrike.Every(60.0, func() {
		p.logger.Debug("Timer fired! Current player count: %d", p.playerCount)
	})

	p.logger.Info("Example plugin loaded successfully!")
	return nil
}

// Unload is called when the plugin is unloaded
func (p *ExamplePlugin) Unload(hotReload bool) error {
	p.logger.Info("Unloading example plugin (hotReload=%v)", hotReload)

	// Unregister chat commands
	gostrike.UnregisterChatCommand("hello")
	gostrike.UnregisterChatCommand("players")
	gostrike.UnregisterChatCommand("info")

	// Stop timers
	if p.greetTimer != nil {
		p.greetTimer.Stop()
	}

	p.logger.Info("Example plugin unloaded")
	return nil
}

// registerChatCommands registers all chat commands (! prefix)
func (p *ExamplePlugin) registerChatCommands() {
	// Simple hello command - !hello
	gostrike.RegisterChatCommand(gostrike.ChatCommandInfo{
		Name:        "hello",
		Description: "Say hello",
		Flags:       gostrike.ChatCmdPublic,
		Callback: func(ctx *gostrike.CommandContext) error {
			ctx.Reply("Hello, %s!", ctx.Player.Name)
			return nil
		},
	})

	// Player list command - !players
	gostrike.RegisterChatCommand(gostrike.ChatCommandInfo{
		Name:        "players",
		Description: "List all connected players",
		Flags:       gostrike.ChatCmdPublic,
		Callback: func(ctx *gostrike.CommandContext) error {
			players := gostrike.GetServer().GetPlayers()
			if len(players) == 0 {
				ctx.Reply("No players connected")
				return nil
			}

			ctx.Reply("=== Connected Players (%d) ===", len(players))
			for _, player := range players {
				status := "Dead"
				if player.IsAlive {
					status = "Alive"
				}
				ctx.Reply("[%d] %s - %s (%s)", player.Slot, player.Name, player.Team, status)
			}
			return nil
		},
	})

	// Server info command - !info
	gostrike.RegisterChatCommand(gostrike.ChatCommandInfo{
		Name:        "info",
		Description: "Show server information",
		Flags:       gostrike.ChatCmdPublic,
		Callback: func(ctx *gostrike.CommandContext) error {
			server := gostrike.GetServer()
			ctx.Reply("=== Server Info ===")
			ctx.Reply("Map: %s", server.GetMapName())
			ctx.Reply("Players: %d/%d", server.GetPlayerCount(), server.GetMaxPlayers())
			ctx.Reply("Tick Rate: %d", server.GetTickRate())
			return nil
		},
	})

	p.logger.Info("Registered 3 chat commands: !hello, !players, !info")
}

// registerEventHandlers registers all event handlers
func (p *ExamplePlugin) registerEventHandlers() {
	// Player connect handler
	gostrike.RegisterPlayerConnectHandler(func(event *gostrike.PlayerConnectEvent) gostrike.EventResult {
		p.playerCount++
		p.logger.Info("Player connected: %s (steamid: %d)", event.Player.Name, event.Player.SteamID)

		// Welcome message (delayed slightly)
		player := event.Player
		gostrike.After(2.0, func() {
			player.PrintToChat("Welcome to the server, %s!", player.Name)
			player.PrintToCenter("Welcome!")
		})

		return gostrike.EventContinue
	}, gostrike.HookPost)

	// Player disconnect handler
	gostrike.RegisterPlayerDisconnectHandler(func(event *gostrike.PlayerDisconnectEvent) gostrike.EventResult {
		p.playerCount--
		p.logger.Info("Player disconnected: slot %d, reason: %s", event.Slot, event.Reason)
		return gostrike.EventContinue
	}, gostrike.HookPost)

	// Map change handler
	gostrike.RegisterMapChangeHandler(func(event *gostrike.MapChangeEvent) gostrike.EventResult {
		p.logger.Info("Map changed to: %s", event.MapName)
		p.playerCount = 0 // Reset player count on map change
		return gostrike.EventContinue
	})

	// Generic event handler for round_start
	gostrike.RegisterGenericEventHandler("round_start", func(eventName string, event gostrike.Event) gostrike.EventResult {
		p.logger.Info("Round started!")
		return gostrike.EventContinue
	}, gostrike.HookPost)

	// Generic event handler for round_end
	gostrike.RegisterGenericEventHandler("round_end", func(eventName string, event gostrike.Event) gostrike.EventResult {
		p.logger.Info("Round ended!")
		return gostrike.EventContinue
	}, gostrike.HookPost)

	p.logger.Info("Registered event handlers")
}

// init registers the plugin with GoStrike
func init() {
	plugin.Register(&ExamplePlugin{})
}
