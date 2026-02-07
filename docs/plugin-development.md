# GoStrike Plugin Development Guide

This guide covers how to build plugins for GoStrike, a Go-based CS2 server modding framework.

## Plugin Structure

Every plugin must implement the `plugin.Plugin` interface and register itself in an `init()` function.

```go
package myplugin

import (
    "github.com/corrreia/gostrike/pkg/gostrike"
    "github.com/corrreia/gostrike/pkg/plugin"
)

type MyPlugin struct {
    plugin.BasePlugin
    logger gostrike.Logger
}

func (p *MyPlugin) Slug() string        { return "myplugin" }
func (p *MyPlugin) Name() string        { return "My Plugin" }
func (p *MyPlugin) Version() string     { return "1.0.0" }
func (p *MyPlugin) Author() string      { return "You" }
func (p *MyPlugin) Description() string { return "A cool plugin" }

func (p *MyPlugin) Load(hotReload bool) error {
    p.logger = gostrike.GetLogger(p.Slug())
    p.logger.Info("Plugin loaded!")
    return nil
}

func (p *MyPlugin) Unload(hotReload bool) error {
    p.logger.Info("Plugin unloaded!")
    return nil
}

func init() {
    plugin.Register(&MyPlugin{})
}
```

Place your plugin in `plugins/<name>/` and it will be auto-discovered at build time.

### Slug

The `Slug()` method returns a unique identifier used for:
- HTTP route namespacing (`/api/plugins/<slug>/...`)
- Database isolation (`data/plugins/<slug>.db`)
- Config file path (`configs/plugins/<slug>.json`)
- Resource tracking and logging

## Chat Commands

Register chat commands that players trigger with `!` prefix in chat:

```go
gostrike.RegisterChatCommand(gostrike.ChatCommandInfo{
    Name:        "hello",
    Description: "Say hello",
    Flags:       gostrike.ChatCmdPublic,
    Callback: func(ctx *gostrike.CommandContext) error {
        ctx.Reply("Hello, %s!", ctx.Player.Name)
        return nil
    },
})
```

**CommandContext fields:**
- `ctx.Player` — the `*Player` who ran the command
- `ctx.Args` — string slice of arguments after the command name
- `ctx.Reply(format, args...)` — sends a chat message back to the player

**Flags:**
- `ChatCmdPublic` — anyone can use
- `ChatCmdAdmin` — requires admin permission

Unregister in `Unload()`:
```go
gostrike.UnregisterChatCommand("hello")
```

## Game Events

GoStrike hooks `IGameEventManager2::FireEvent` to intercept all Source 2 game events. You can register handlers for any event using the typed or generic API.

### Typed Event Handlers (Recommended)

```go
// Player death — typed wrapper with convenience methods
gostrike.RegisterPlayerDeathHandler(func(event *gostrike.PlayerDeathEvent) gostrike.EventResult {
    victim := event.Victim()       // *Player (or nil)
    attacker := event.Attacker()   // *Player (or nil)
    weapon := event.Weapon()       // string
    headshot := event.Headshot()   // bool

    fmt.Printf("%s killed %s with %s\n", attacker.Name, victim.Name, weapon)
    return gostrike.EventContinue
}, gostrike.HookPost)

// Round start
gostrike.RegisterRoundStartHandler(func(event *gostrike.RoundStartEvent) gostrike.EventResult {
    fmt.Printf("Round started! Time limit: %d\n", event.TimeLimit())
    return gostrike.EventContinue
}, gostrike.HookPost)

// Round end
gostrike.RegisterRoundEndHandler(func(event *gostrike.RoundEndEvent) gostrike.EventResult {
    fmt.Printf("Round ended! Winner: %s\n", event.Winner())
    return gostrike.EventContinue
}, gostrike.HookPost)

// Bomb planted
gostrike.RegisterBombPlantedHandler(func(event *gostrike.BombPlantedEvent) gostrike.EventResult {
    player := event.Player()
    site := event.Site() // 0=A, 1=B
    fmt.Printf("%s planted on site %d\n", player.Name, site)
    return gostrike.EventContinue
}, gostrike.HookPost)
```

### Generic Game Event Handler

For any event not covered by typed wrappers, use `RegisterGameEventHandler`. The `GameEvent` object provides direct native field access:

```go
gostrike.RegisterGameEventHandler("weapon_fire", func(event *gostrike.GameEvent) gostrike.EventResult {
    weapon := event.GetString("weapon")
    silenced := event.GetBool("silenced")

    // Modify fields in pre-hook
    if event.CanModify {
        event.SetString("weapon", "modified_weapon")
    }

    return gostrike.EventContinue
}, gostrike.HookPre) // HookPre to intercept before processing
```

**GameEvent methods:**
- `GetInt(key)`, `GetFloat(key)`, `GetBool(key)`, `GetString(key)`, `GetUint64(key)`
- `SetInt(key, val)`, `SetFloat(key, val)`, `SetBool(key, val)`, `SetString(key, val)` — pre-hook only

### EventResult Values

| Value | Meaning |
|-------|---------|
| `EventContinue` | Allow event to proceed normally |
| `EventChanged` | Event data was modified (pre-hook) |
| `EventHandled` | Stop processing further handlers |
| `EventStop` | Cancel the event entirely (pre-hook only) |

### HookMode

- `HookPre` — called before the event is processed; can modify or cancel
- `HookPost` — called after the event is processed; read-only, informational

### Other Event Types

```go
// Player connect/disconnect
gostrike.RegisterPlayerConnectHandler(func(e *gostrike.PlayerConnectEvent) gostrike.EventResult {
    fmt.Printf("Player connected: %s\n", e.Player.Name)
    return gostrike.EventContinue
}, gostrike.HookPost)

gostrike.RegisterPlayerDisconnectHandler(func(e *gostrike.PlayerDisconnectEvent) gostrike.EventResult {
    fmt.Printf("Slot %d disconnected: %s\n", e.Slot, e.Reason)
    return gostrike.EventContinue
}, gostrike.HookPost)

// Map change
gostrike.RegisterMapChangeHandler(func(e *gostrike.MapChangeEvent) gostrike.EventResult {
    fmt.Printf("Map changed to: %s\n", e.MapName)
    return gostrike.EventContinue
})
```

## Damage Hooks

GoStrike hooks `CBaseEntity::TakeDamageOld` to intercept all damage. You can log, modify, or block damage:

```go
gostrike.RegisterDamageHandler(func(info *gostrike.DamageInfo) gostrike.EventResult {
    // info.VictimIndex    — entity index of the victim
    // info.AttackerIndex  — entity index of the attacker (-1 if world)
    // info.Damage         — damage amount (float32)
    // info.DamageType     — damage type flags (int)

    // Block all fall damage (type 32)
    if info.DamageType == 32 {
        return gostrike.EventHandled // Skip the damage
    }

    return gostrike.EventContinue
})
```

## Player API

The `*Player` struct provides all player operations:

### Information
```go
player.Slot      // int — player slot
player.Name      // string — player name
player.SteamID   // uint64 — Steam ID
player.Team      // Team — TeamT, TeamCT, TeamSpectator, TeamUnassigned
player.IsAlive   // bool
player.IsBot     // bool
player.Health    // int
player.Armor     // int
player.Position  // Vector3{X, Y, Z}

player.Refresh()    // Update all fields from server
player.IsValid()    // Check if still connected
```

### Actions
```go
player.Respawn()                        // Respawn the player
player.Slay()                           // Kill the player immediately
player.Kick("reason")                   // Kick from server
player.ChangeTeam(gostrike.TeamCT)      // Change team
player.Teleport(&pos, &angles, &vel)    // Teleport (pass nil for unchanged)
```

### Weapons
```go
player.GiveWeapon("ak47")         // Give weapon (auto-prepends "weapon_" if needed)
player.GiveWeapon("weapon_awp")   // Also works with full name
player.DropWeapons()               // Drop all weapons
```

### Health & Armor
```go
player.SetHealth(500)       // Set health (via schema)
player.SetMaxHealth(500)    // Set max health (so HUD shows correctly)
player.SetArmor(100)        // Set armor value
```

### Communication
```go
player.PrintToChat("Hello %s!", player.Name)      // Chat message
player.PrintToCenter("Important message!")          // Center HUD
player.PrintToConsole("Debug info here")            // Console message
player.PrintToAlert("Alert!")                       // Alert HUD
```

### Entity Access
```go
controller := player.GetController()  // *Entity — CCSPlayerController
pawn := player.GetPawn()              // *Entity — CCSPlayerPawn (nil if dead)
```

## Server API

```go
server := gostrike.GetServer()
server.GetMapName()          // string
server.GetPlayerCount()      // int
server.GetMaxPlayers()       // int
server.GetTickRate()         // int
server.GetPlayers()          // []*Player
server.GetPlayerBySlot(0)    // *Player
server.PrintToAll("msg")     // Print to all players
server.ExecuteCommand("mp_restartgame 1")
```

## Entity System

### Finding Entities
```go
entity := gostrike.GetEntityByIndex(42)
entities := gostrike.FindEntitiesByClassName("cs_player_controller")
```

### Schema Properties (Raw)
```go
health, err := entity.GetPropInt("CBaseEntity", "m_iHealth")
entity.SetPropInt("CBaseEntity", "m_iHealth", 500)
entity.SetPropFloat("CBaseEntity", "m_flGravityScale", 0.5)
name := entity.GetPropString("CBasePlayerController", "m_iszPlayerName")
x, y, z := entity.GetPropVector("CBaseEntity", "m_vecAbsOrigin")
```

### Generated Typed Wrappers

GoStrike generates typed entity wrappers from the CS2 schema:

```go
import "github.com/corrreia/gostrike/pkg/gostrike/entities"

pawn := entities.NewCCSPlayerPawnBase(player.GetPawn())
pawn.ArmorValue()   // int32
pawn.IsScoped()     // bool
pawn.IsWalking()    // bool
pawn.InBuyZone()    // bool

ctrl := entities.NewCCSPlayerController(player.GetController())
ctrl.PawnHasHelmet()  // bool

base := entities.NewCBaseEntity(player.GetPawn())
base.Health()       // int32
base.MaxHealth()    // int32
```

### Entity Lifecycle Events
```go
gostrike.RegisterEntitySpawnedHandler(func(entity *gostrike.Entity) {
    if entity.ClassName == "cs_player_controller" {
        fmt.Printf("Player controller spawned: %d\n", entity.Index)
    }
})

gostrike.RegisterEntityDeletedHandler(func(index uint32) {
    fmt.Printf("Entity %d deleted\n", index)
})
```

## ConVars

```go
// Read
roundTime := gostrike.GetConVarFloat("mp_roundtime")
freezeTime := gostrike.GetConVarInt("mp_freezetime")
hostname := gostrike.GetConVarString("hostname")

// Write
gostrike.SetConVarInt("mp_freezetime", 5)
gostrike.SetConVarFloat("mp_roundtime", 2.5)
gostrike.SetConVarString("hostname", "My Server")
```

## Timers

```go
// One-shot timer (seconds)
gostrike.After(5.0, func() {
    fmt.Println("5 seconds elapsed!")
})

// Repeating timer
timer := gostrike.Every(30.0, func() {
    fmt.Println("Tick every 30 seconds")
})

// Stop a repeating timer
timer.Stop()
```

## Database

Each plugin gets an isolated SQLite database:

```go
db, err := gostrike.GetPluginDB(p.Slug())
// Creates: data/plugins/<slug>.db

// Create tables
db.Exec(`CREATE TABLE IF NOT EXISTS stats (
    steam_id INTEGER PRIMARY KEY,
    kills INTEGER DEFAULT 0,
    deaths INTEGER DEFAULT 0
)`)

// Query
rows, _ := db.Query("SELECT kills, deaths FROM stats WHERE steam_id = ?", steamID)
defer rows.Close()

// Insert/Update
db.Exec("INSERT OR REPLACE INTO stats (steam_id, kills) VALUES (?, ?)", steamID, kills)
```

## HTTP API

Plugins can register HTTP endpoints, automatically namespaced:

```go
api := gostrike.NewPluginHTTPGroup(p.Slug())
// All routes prefixed with /api/plugins/<slug>/

api.GET("/status", func(w http.ResponseWriter, r *http.Request) {
    gostrike.JSONSuccess(w, map[string]interface{}{
        "status": "running",
    })
})

api.POST("/action", func(w http.ResponseWriter, r *http.Request) {
    var req struct {
        Message string `json:"message"`
    }
    if err := gostrike.ReadJSON(r, &req); err != nil {
        gostrike.JSONError(w, http.StatusBadRequest, "Invalid JSON")
        return
    }
    // Handle request...
    gostrike.JSONSuccess(w, map[string]interface{}{"ok": true})
})
```

## Configuration

Plugins get auto-managed JSON config files:

```go
// Define defaults in your plugin
func (p *MyPlugin) DefaultConfig() map[string]interface{} {
    return map[string]interface{}{
        "welcome_message": "Welcome!",
        "max_hp":          100,
    }
}

// Load in Load()
config := gostrike.GetPluginConfigOrDefault(p.Slug())
// Creates configs/plugins/<slug>.json if not exists

msg := config.GetString("welcome_message", "fallback")
maxHP := config.GetInt("max_hp", 100)
enabled := config.GetBool("features.enabled", true)
```

## Logging

```go
logger := gostrike.GetLogger("myplugin")
logger.Debug("debug info: %v", value)
logger.Info("something happened")
logger.Warning("careful: %s", msg)
logger.Error("failed: %v", err)
```

Log output goes to the CS2 server console with `[GoStrike:myplugin]` prefix.

## Tick Handlers

Called every server tick (~64Hz):

```go
gostrike.RegisterTickHandler(func(deltaTime float64) {
    // Called every tick
    // deltaTime is seconds since last tick
})
```

## Complete Example

See `plugins/example/example.go` for a full working plugin demonstrating all features including chat commands, game events, weapon management, health/armor modification, database, HTTP API, timers, and entity access.
