// Package example provides an example GoStrike plugin demonstrating
// how to use the SDK features: chat commands, events, timers, HTTP API,
// database, and logging.
package example

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/corrreia/gostrike/pkg/gostrike"
	"github.com/corrreia/gostrike/pkg/gostrike/entities"
	"github.com/corrreia/gostrike/pkg/plugin"
)

// ExamplePlugin demonstrates GoStrike SDK usage
type ExamplePlugin struct {
	plugin.BasePlugin
	logger      gostrike.Logger
	playerCount int
	greetTimer  *gostrike.Timer
	db          *gostrike.PluginDB // Plugin's isolated database
}

// Slug returns the plugin's unique identifier
// This is used for namespacing HTTP routes (/api/plugins/example/...),
// database isolation (data/plugins/example.db), and resource tracking
func (p *ExamplePlugin) Slug() string {
	return "example"
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

// DefaultConfig returns the default configuration for this plugin
// This will auto-generate configs/plugins/example.json if it doesn't exist
func (p *ExamplePlugin) DefaultConfig() map[string]interface{} {
	return map[string]interface{}{
		"welcome_message": "Welcome to the server!",
		"max_greetings":   100,
		"features": map[string]interface{}{
			"auto_greet":    true,
			"log_connects":  true,
			"track_players": true,
		},
	}
}

// Load is called when the plugin is loaded
func (p *ExamplePlugin) Load(hotReload bool) error {
	// Use slug for logger tag - this ensures consistent [GoStrike:example] prefix
	p.logger = gostrike.GetLogger(p.Slug())
	p.logger.Info("Loading example plugin (hotReload=%v)", hotReload)

	// Load plugin config (auto-generated at configs/plugins/example.json)
	config := gostrike.GetPluginConfigOrDefault(p.Slug())
	welcomeMsg := config.GetString("welcome_message", "Welcome!")
	autoGreet := config.GetBool("features.auto_greet", true)
	p.logger.Info("Config loaded: welcome_message=%s, auto_greet=%v", welcomeMsg, autoGreet)

	// Initialize plugin database (isolated per-plugin)
	// Database file will be created at: data/plugins/example.db
	if err := p.initDatabase(); err != nil {
		p.logger.Error("Failed to initialize database: %v", err)
		// Continue loading even if database fails - it's optional
	}

	// Register HTTP API endpoints
	// Routes will be prefixed with /api/plugins/example/
	p.registerHTTPHandlers()

	// Register chat commands
	// Commands now return errors on collision
	if err := p.registerChatCommands(); err != nil {
		return fmt.Errorf("failed to register chat commands: %w", err)
	}

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
	gostrike.UnregisterChatCommand("health")
	gostrike.UnregisterChatCommand("entities")
	gostrike.UnregisterChatCommand("respawn")
	gostrike.UnregisterChatCommand("roundtime")
	gostrike.UnregisterChatCommand("pawninfo")
	gostrike.UnregisterChatCommand("give")
	gostrike.UnregisterChatCommand("hp")
	gostrike.UnregisterChatCommand("armor")

	// Stop timers
	if p.greetTimer != nil {
		p.greetTimer.Stop()
	}

	// Close plugin database
	if p.db != nil {
		p.db.Close()
	}

	p.logger.Info("Example plugin unloaded")
	return nil
}

// initDatabase initializes the plugin's isolated database
func (p *ExamplePlugin) initDatabase() error {
	var err error
	p.db, err = gostrike.GetPluginDB(p.Slug())
	if err != nil {
		return err
	}

	// Create plugin-specific tables
	// Note: These tables are isolated to this plugin's database file
	_, err = p.db.Exec(`
		CREATE TABLE IF NOT EXISTS greetings (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			player_name TEXT NOT NULL,
			steam_id INTEGER NOT NULL,
			greeted_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create greetings table: %w", err)
	}

	p.logger.Info("Database initialized at: %s", p.db.Path())
	return nil
}

// registerHTTPHandlers registers HTTP API endpoints
// All routes are automatically namespaced under /api/plugins/example/
func (p *ExamplePlugin) registerHTTPHandlers() {
	// Create a plugin HTTP group - all routes prefixed with /api/plugins/example
	api := gostrike.NewPluginHTTPGroup(p.Slug())

	// GET /api/plugins/example/status
	api.GET("/status", func(w http.ResponseWriter, r *http.Request) {
		gostrike.JSONSuccess(w, map[string]interface{}{
			"plugin":       p.Name(),
			"version":      p.Version(),
			"slug":         p.Slug(),
			"player_count": p.playerCount,
		})
	})

	// GET /api/plugins/example/players
	api.GET("/players", func(w http.ResponseWriter, r *http.Request) {
		players := gostrike.GetServer().GetPlayers()
		gostrike.JSONSuccess(w, map[string]interface{}{
			"count":   len(players),
			"players": players,
		})
	})

	// POST /api/plugins/example/greet
	api.POST("/greet", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Message string `json:"message"`
		}
		if err := gostrike.ReadJSON(r, &req); err != nil {
			gostrike.JSONError(w, http.StatusBadRequest, "Invalid JSON")
			return
		}

		// Broadcast message to all players
		gostrike.GetServer().PrintToAll("[Example] %s", req.Message)

		gostrike.JSONSuccess(w, map[string]interface{}{
			"success": true,
			"message": req.Message,
		})
	})

	// POST /api/plugins/example/say
	// Sends a message to the game chat
	// Example: curl -X POST http://localhost:8080/api/plugins/example/say -H "Content-Type: application/json" -d '{"message": "Hello from API!"}'
	api.POST("/say", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Message string `json:"message"`
			Prefix  string `json:"prefix,omitempty"` // Optional prefix, defaults to "[Server]"
		}
		if err := gostrike.ReadJSON(r, &req); err != nil {
			gostrike.JSONError(w, http.StatusBadRequest, "Invalid JSON body")
			return
		}

		if req.Message == "" {
			gostrike.JSONError(w, http.StatusBadRequest, "Message cannot be empty")
			return
		}

		// Use default prefix if not provided
		prefix := req.Prefix
		if prefix == "" {
			prefix = "[Server]"
		}

		// Send message to game chat
		gostrike.GetServer().PrintToAll("%s %s", prefix, req.Message)
		p.logger.Info("API sent chat message: %s %s", prefix, req.Message)

		gostrike.JSONSuccess(w, map[string]interface{}{
			"success": true,
			"message": req.Message,
			"prefix":  prefix,
		})
	})

	p.logger.Info("Registered HTTP endpoints at %s/*", api.Prefix())
}

// registerChatCommands registers all chat commands (! prefix)
// Returns an error if any command registration fails (e.g., collision)
func (p *ExamplePlugin) registerChatCommands() error {
	// Simple hello command - !hello
	if err := gostrike.RegisterChatCommand(gostrike.ChatCommandInfo{
		Name:        "hello",
		Description: "Say hello",
		Flags:       gostrike.ChatCmdPublic,
		Callback: func(ctx *gostrike.CommandContext) error {
			ctx.Reply("Hello, %s!", ctx.Player.Name)

			// Record greeting in plugin database (if available)
			if p.db != nil {
				_, err := p.db.Exec(
					"INSERT INTO greetings (player_name, steam_id) VALUES (?, ?)",
					ctx.Player.Name, ctx.Player.SteamID,
				)
				if err != nil {
					p.logger.Error("Failed to record greeting: %v", err)
				}
			}

			return nil
		},
	}); err != nil {
		return fmt.Errorf("failed to register !hello: %w", err)
	}

	// Player list command - !players
	if err := gostrike.RegisterChatCommand(gostrike.ChatCommandInfo{
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
	}); err != nil {
		return fmt.Errorf("failed to register !players: %w", err)
	}

	// Server info command - !info
	if err := gostrike.RegisterChatCommand(gostrike.ChatCommandInfo{
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
	}); err != nil {
		return fmt.Errorf("failed to register !info: %w", err)
	}

	// Health command - !health (demonstrates Phase 1 entity/schema access)
	if err := gostrike.RegisterChatCommand(gostrike.ChatCommandInfo{
		Name:        "health",
		Description: "Show your current health via schema system",
		Flags:       gostrike.ChatCmdPublic,
		Callback: func(ctx *gostrike.CommandContext) error {
			// Get the player's entity by slot index
			entity := gostrike.GetEntityByIndex(uint32(ctx.Player.Slot))
			if entity == nil || !entity.IsValid() {
				ctx.Reply("Could not find your entity!")
				return nil
			}

			// Read health via schema system
			health, err := entity.GetPropInt("CBaseEntity", "m_iHealth")
			if err != nil {
				ctx.Reply("Could not read health: %v", err)
				return nil
			}

			ctx.Reply("Your health: %d", health)
			return nil
		},
	}); err != nil {
		return fmt.Errorf("failed to register !health: %w", err)
	}

	// Entities command - !entities (demonstrates entity iteration)
	if err := gostrike.RegisterChatCommand(gostrike.ChatCommandInfo{
		Name:        "entities",
		Description: "Count entities by type",
		Flags:       gostrike.ChatCmdPublic,
		Callback: func(ctx *gostrike.CommandContext) error {
			controllers := gostrike.FindEntitiesByClassName("cs_player_controller")
			ctx.Reply("Player controllers: %d", len(controllers))

			// Show schema offset for a well-known field
			offset, networked := gostrike.GetSchemaOffset("CBaseEntity", "m_iHealth")
			ctx.Reply("CBaseEntity::m_iHealth offset=%d, networked=%v", offset, networked)
			return nil
		},
	}); err != nil {
		return fmt.Errorf("failed to register !entities: %w", err)
	}

	// Respawn command - !respawn (demonstrates Phase 2 game functions)
	if err := gostrike.RegisterChatCommand(gostrike.ChatCommandInfo{
		Name:        "respawn",
		Description: "Respawn yourself",
		Flags:       gostrike.ChatCmdPublic,
		Callback: func(ctx *gostrike.CommandContext) error {
			ctx.Player.Respawn()
			ctx.Reply("Respawning!")
			return nil
		},
	}); err != nil {
		return fmt.Errorf("failed to register !respawn: %w", err)
	}

	// ConVar command - !roundtime (demonstrates Phase 2 ConVar access)
	if err := gostrike.RegisterChatCommand(gostrike.ChatCommandInfo{
		Name:        "roundtime",
		Description: "Show current round time",
		Flags:       gostrike.ChatCmdPublic,
		Callback: func(ctx *gostrike.CommandContext) error {
			roundTime := gostrike.GetConVarFloat("mp_roundtime")
			freezeTime := gostrike.GetConVarInt("mp_freezetime")
			ctx.Reply("Round time: %.1f min, Freeze time: %d sec", roundTime, freezeTime)
			return nil
		},
	}); err != nil {
		return fmt.Errorf("failed to register !roundtime: %w", err)
	}

	// Typed entity access - !pawninfo (demonstrates Phase 5 generated wrappers)
	if err := gostrike.RegisterChatCommand(gostrike.ChatCommandInfo{
		Name:        "pawninfo",
		Description: "Show pawn info using generated typed entity accessors",
		Flags:       gostrike.ChatCmdPublic,
		Callback: func(ctx *gostrike.CommandContext) error {
			pawn := entities.NewCCSPlayerPawnBase(ctx.Player.GetPawn())
			if pawn == nil {
				ctx.Reply("No pawn available (dead or spectating?)")
				return nil
			}

			// Read helmet from controller (m_bHasHelmet is on CCSPlayer_ItemServices sub-object,
			// but CCSPlayerController has a cached m_bPawnHasHelmet)
			ctrl := entities.NewCCSPlayerController(ctx.Player.GetController())
			hasHelmet := false
			if ctrl != nil {
				hasHelmet = ctrl.PawnHasHelmet()
			}

			base := entities.NewCBaseEntity(ctx.Player.GetPawn())
			ctx.Reply("=== Pawn Info (typed) ===")
			ctx.Reply("Health: %d/%d", base.Health(), base.MaxHealth())
			ctx.Reply("Armor: %d, Helmet: %v", pawn.ArmorValue(), hasHelmet)
			ctx.Reply("Scoped: %v, Walking: %v", pawn.IsScoped(), pawn.IsWalking())
			ctx.Reply("In buy zone: %v", pawn.InBuyZone())
			return nil
		},
	}); err != nil {
		return fmt.Errorf("failed to register !pawninfo: %w", err)
	}

	// Give weapon command - !give <weapon> (demonstrates weapon management)
	if err := gostrike.RegisterChatCommand(gostrike.ChatCommandInfo{
		Name:        "give",
		Description: "Give yourself a weapon (e.g. !give ak47)",
		Flags:       gostrike.ChatCmdPublic,
		Callback: func(ctx *gostrike.CommandContext) error {
			if len(ctx.Args) < 1 {
				ctx.Reply("Usage: !give <weapon_name>")
				ctx.Reply("Examples: !give ak47, !give awp, !give deagle")
				return nil
			}
			weapon := strings.Join(ctx.Args, "_")
			ctx.Player.GiveWeapon(weapon)
			ctx.Reply("Gave you %s!", weapon)
			return nil
		},
	}); err != nil {
		return fmt.Errorf("failed to register !give: %w", err)
	}

	// Set health command - !hp <amount> (demonstrates player convenience methods)
	if err := gostrike.RegisterChatCommand(gostrike.ChatCommandInfo{
		Name:        "hp",
		Description: "Set your health (e.g. !hp 500)",
		Flags:       gostrike.ChatCmdPublic,
		Callback: func(ctx *gostrike.CommandContext) error {
			if len(ctx.Args) < 1 {
				ctx.Reply("Usage: !hp <amount>")
				return nil
			}
			amount, err := strconv.Atoi(ctx.Args[0])
			if err != nil || amount < 1 {
				ctx.Reply("Invalid health amount")
				return nil
			}
			ctx.Player.SetHealth(amount)
			if amount > 100 {
				ctx.Player.SetMaxHealth(amount)
			}
			ctx.Reply("Health set to %d!", amount)
			return nil
		},
	}); err != nil {
		return fmt.Errorf("failed to register !hp: %w", err)
	}

	// Set armor command - !armor <amount> (demonstrates player convenience methods)
	if err := gostrike.RegisterChatCommand(gostrike.ChatCommandInfo{
		Name:        "armor",
		Description: "Set your armor (e.g. !armor 100)",
		Flags:       gostrike.ChatCmdPublic,
		Callback: func(ctx *gostrike.CommandContext) error {
			if len(ctx.Args) < 1 {
				ctx.Reply("Usage: !armor <amount>")
				return nil
			}
			amount, err := strconv.Atoi(ctx.Args[0])
			if err != nil || amount < 0 {
				ctx.Reply("Invalid armor amount")
				return nil
			}
			ctx.Player.SetArmor(amount)
			ctx.Reply("Armor set to %d!", amount)
			return nil
		},
	}); err != nil {
		return fmt.Errorf("failed to register !armor: %w", err)
	}

	p.logger.Info("Registered 11 chat commands: !hello, !players, !info, !health, !entities, !respawn, !roundtime, !pawninfo, !give, !hp, !armor")
	return nil
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

	// Player death handler (demonstrates native GameEvent with typed wrappers)
	gostrike.RegisterPlayerDeathHandler(func(event *gostrike.PlayerDeathEvent) gostrike.EventResult {
		victim := event.Victim()
		attacker := event.Attacker()

		victimName := "unknown"
		attackerName := "world"
		if victim != nil {
			victimName = victim.Name
		}
		if attacker != nil {
			attackerName = attacker.Name
		}

		hsText := ""
		if event.Headshot() {
			hsText = " (headshot)"
		}

		p.logger.Info("Kill: %s killed %s with %s%s", attackerName, victimName, event.Weapon(), hsText)
		return gostrike.EventContinue
	}, gostrike.HookPost)

	// Round start handler (demonstrates typed round event)
	gostrike.RegisterRoundStartHandler(func(event *gostrike.RoundStartEvent) gostrike.EventResult {
		p.logger.Info("Round started! Time limit: %d, Frag limit: %d", event.TimeLimit(), event.FragLimit())
		return gostrike.EventContinue
	}, gostrike.HookPost)

	// Round end handler
	gostrike.RegisterRoundEndHandler(func(event *gostrike.RoundEndEvent) gostrike.EventResult {
		p.logger.Info("Round ended! Winner: %s, Reason: %d", event.Winner(), event.Reason())
		return gostrike.EventContinue
	}, gostrike.HookPost)

	// Damage handler (demonstrates TakeDamage hook)
	gostrike.RegisterDamageHandler(func(info *gostrike.DamageInfo) gostrike.EventResult {
		// Log damage events (high-volume, use debug level)
		p.logger.Debug("Damage: entity %d took %.1f damage from entity %d (type: %d)",
			info.VictimIndex, info.Damage, info.AttackerIndex, info.DamageType)
		return gostrike.EventContinue
	})

	// Entity lifecycle handlers (Phase 1)
	gostrike.RegisterEntitySpawnedHandler(func(entity *gostrike.Entity) {
		// Log player controllers being spawned
		if entity.ClassName == "cs_player_controller" {
			p.logger.Debug("Player controller spawned: index=%d", entity.Index)
		}
	})

	gostrike.RegisterEntityDeletedHandler(func(index uint32) {
		p.logger.Debug("Entity deleted: index=%d", index)
	})

	p.logger.Info("Registered event handlers (connect, disconnect, map_change, player_death, round_start, round_end, damage)")
}

// init registers the plugin with GoStrike
func init() {
	plugin.Register(&ExamplePlugin{})
}
