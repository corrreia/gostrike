// Package example provides an example GoStrike plugin demonstrating
// how to use the SDK features: commands, events, timers, and logging.
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

	// Register commands
	p.registerCommands()

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

	// Stop timers
	if p.greetTimer != nil {
		p.greetTimer.Stop()
	}

	p.logger.Info("Example plugin unloaded")
	return nil
}

// registerCommands registers all plugin commands
func (p *ExamplePlugin) registerCommands() {
	// Simple hello command
	gostrike.RegisterCommand(gostrike.CommandInfo{
		Name:        "gs_hello",
		Description: "Say hello",
		Flags:       gostrike.CmdAll,
		Callback: func(ctx *gostrike.CommandContext) error {
			if ctx.Player != nil {
				ctx.Reply("Hello, %s!", ctx.Player.Name)
			} else {
				ctx.Reply("Hello from the server console!")
			}
			return nil
		},
	})

	// Player list command
	gostrike.RegisterCommand(gostrike.CommandInfo{
		Name:        "gs_players",
		Description: "List all connected players",
		Flags:       gostrike.CmdAll,
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

	// Server info command
	gostrike.RegisterCommand(gostrike.CommandInfo{
		Name:        "gs_info",
		Description: "Show server information",
		Flags:       gostrike.CmdAll,
		Callback: func(ctx *gostrike.CommandContext) error {
			server := gostrike.GetServer()
			ctx.Reply("=== Server Info ===")
			ctx.Reply("Map: %s", server.GetMapName())
			ctx.Reply("Players: %d/%d", server.GetPlayerCount(), server.GetMaxPlayers())
			ctx.Reply("Tick Rate: %d", server.GetTickRate())
			return nil
		},
	})

	// Slap command (admin only)
	gostrike.RegisterCommand(gostrike.CommandInfo{
		Name:        "gs_slap",
		Description: "Slap a player",
		Usage:       "<player> [damage]",
		MinArgs:     1,
		Flags:       gostrike.CmdServer | gostrike.CmdAdmin,
		Callback: func(ctx *gostrike.CommandContext) error {
			if !ctx.RequireFlag(gostrike.AdminSlay) {
				return nil
			}

			targetName := ctx.GetArg(0)
			damage := ctx.GetArgInt(1, 0)

			// Find player by name
			var target *gostrike.Player
			for _, p := range gostrike.GetServer().GetPlayers() {
				if p.Name == targetName {
					target = p
					break
				}
			}

			if target == nil {
				ctx.ReplyError("Player not found: %s", targetName)
				return nil
			}

			target.PrintToChat("You have been slapped for %d damage!", damage)
			ctx.Reply("Slapped %s for %d damage", target.Name, damage)
			p.logger.Info("Admin slapped %s for %d damage", target.Name, damage)

			return nil
		},
	})

	// Timer test command
	gostrike.RegisterCommand(gostrike.CommandInfo{
		Name:        "gs_timer",
		Description: "Test timer system",
		Usage:       "<seconds>",
		MinArgs:     1,
		Flags:       gostrike.CmdAll,
		Callback: func(ctx *gostrike.CommandContext) error {
			seconds := ctx.GetArgFloat(0, 5.0)
			if seconds <= 0 || seconds > 60 {
				ctx.ReplyError("Seconds must be between 0 and 60")
				return nil
			}

			ctx.Reply("Timer set for %.1f seconds...", seconds)

			gostrike.After(seconds, func() {
				if ctx.Player != nil {
					ctx.Player.PrintToChat("Timer finished!")
				}
				p.logger.Info("Timer callback executed after %.1f seconds", seconds)
			})

			return nil
		},
	})

	// Panic test command (for testing panic recovery)
	gostrike.RegisterCommand(gostrike.CommandInfo{
		Name:        "gs_panic",
		Description: "Test panic recovery (debug only)",
		Flags:       gostrike.CmdServer,
		Callback: func(ctx *gostrike.CommandContext) error {
			ctx.Reply("About to panic... (should be recovered)")
			panic("Intentional panic for testing")
		},
	})

	p.logger.Info("Registered %d commands", 6)
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
