# GoStrike Architecture

This document describes the architecture of GoStrike, a Counter-Strike 2 server modding framework using Go.

## Overview

GoStrike consists of three main layers:

1. **Native Layer** - C++ Metamod plugin that hooks into CS2
2. **Bridge Layer** - CGO interface between C++ and Go
3. **Go Runtime** - Plugin system, modules, and SDK

```
┌─────────────────────────────────────────────────────────────────┐
│                     CS2 Dedicated Server                        │
├─────────────────────────────────────────────────────────────────┤
│                      Metamod:Source                             │
├─────────────────────────────────────────────────────────────────┤
│              GoStrike Native Plugin (gostrike.so)               │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │                    C ABI Bridge                          │   │
│  │  ┌─────────────────────────────────────────────────┐    │   │
│  │  │            Go Runtime (libgostrike_go.so)        │    │   │
│  │  │  ┌───────────────────────────────────────────┐  │    │   │
│  │  │  │              Core Modules                  │  │    │   │
│  │  │  │  ┌─────────┐ ┌──────┐ ┌──────────────┐   │  │    │   │
│  │  │  │  │Permissions│ │ HTTP │ │   Database   │   │  │    │   │
│  │  │  │  └─────────┘ └──────┘ └──────────────┘   │  │    │   │
│  │  │  ├───────────────────────────────────────────┤  │    │   │
│  │  │  │           Plugin Manager                   │  │    │   │
│  │  │  │  ┌────────┐ ┌────────┐ ┌────────────┐    │  │    │   │
│  │  │  │  │Plugin A│ │Plugin B│ │  Plugin C  │    │  │    │   │
│  │  │  │  └────────┘ └────────┘ └────────────┘    │  │    │   │
│  │  │  └───────────────────────────────────────────┘  │    │   │
│  │  └─────────────────────────────────────────────────┘    │   │
│  └─────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────┘
```

## Communication Architecture

GoStrike uses two main interfaces for communication:

### HTTP API (Primary Interface)

The HTTP module provides the primary interface for external communication with GoStrike:

- Server management and status
- Plugin management
- Configuration
- Integration with external tools

### Chat Commands (Plugin Interaction)

Plugins can register chat commands (prefixed with `!`) for player interaction:

- `!help` - Show available commands
- `!admin` - Admin actions (if authorized)
- Custom plugin commands

## Directory Structure

```
gostrike/
├── cmd/gostrike/           # Go c-shared entry point
│   └── main.go             # Imports bridge and plugins
├── configs/                # Configuration files
│   ├── gostrike.json       # Main configuration
│   ├── admins.json         # Admin permissions
│   ├── http.json           # HTTP server config
│   └── plugins.json        # Plugin enable/disable
├── docker/                 # Docker development environment
│   ├── docker-compose.yml  # CS2 server container
│   ├── Dockerfile          # Build container
│   └── scripts/            # Setup scripts
├── external/               # Git submodules (SDKs)
│   ├── hl2sdk-cs2/         # HL2SDK for CS2
│   └── metamod-source/     # Metamod:Source
├── internal/               # Internal implementation
│   ├── bridge/             # CGO exports and callbacks
│   │   ├── callbacks.go    # Go -> C++ callbacks
│   │   ├── exports.go      # C++ -> Go exports
│   │   └── types.go        # Type conversions
│   ├── manager/            # Plugin lifecycle management
│   │   ├── manager.go      # Plugin loading/unloading
│   │   └── registry.go     # Static plugin registration
│   ├── modules/            # Core modules
│   │   ├── module.go       # Module interface
│   │   ├── permissions/    # Permission system
│   │   ├── http/           # HTTP server
│   │   └── database/       # Database abstraction
│   ├── runtime/            # Runtime system
│   │   ├── chat.go         # Chat command system
│   │   ├── commands.go     # Version/plugin utilities
│   │   ├── dispatcher.go   # Event dispatch
│   │   ├── modules.go      # Module integration
│   │   ├── runtime.go      # Init/shutdown
│   │   └── timers.go       # Timer system
│   └── shared/             # Shared types
├── native/                 # C++ Metamod plugin
│   ├── CMakeLists.txt      # CMake build configuration
│   ├── include/            # Headers
│   │   ├── gostrike_abi.h  # C ABI definition
│   │   └── stub/           # Stub SDK headers (development)
│   ├── src/                # Source files
│   │   ├── go_bridge.cpp   # Go library loading + callbacks
│   │   ├── go_bridge.h     # Bridge header
│   │   ├── gostrike.cpp    # Metamod plugin implementation
│   │   └── gostrike.h      # Plugin header
│   ├── scripts/            # Build scripts
│   │   └── generate_protos.sh  # Protobuf header generator
│   └── generated/          # Generated protobuf headers (gitignored)
├── pkg/                    # Public SDK
│   ├── gostrike/           # Plugin SDK
│   │   ├── command.go      # Chat command registration
│   │   ├── database.go     # Database access
│   │   ├── event.go        # Event handlers
│   │   ├── http.go         # HTTP handlers
│   │   ├── permissions.go  # Permission checks
│   │   ├── player.go       # Player API
│   │   ├── server.go       # Server API
│   │   └── timer.go        # Timers
│   └── plugin/             # Plugin interface
├── plugins/                # Community plugins
│   └── example/            # Example plugin
└── scripts/                # Build scripts
```

## C ABI Bridge

The bridge between C++ and Go uses a stable C ABI defined in `native/include/gostrike_abi.h`.

### Exports (Go → C++)

Functions exported from Go that C++ calls:

| Function | Description |
|----------|-------------|
| `GoStrike_Init()` | Initialize Go runtime |
| `GoStrike_Shutdown()` | Shutdown Go runtime |
| `GoStrike_OnTick(deltaTime)` | Called every server tick |
| `GoStrike_OnEvent(event, isPost)` | Game event dispatch |
| `GoStrike_OnChatMessage(slot, message)` | Chat message dispatch (for !commands) |
| `GoStrike_OnPlayerConnect(player)` | Player connect event |
| `GoStrike_OnPlayerDisconnect(slot, reason)` | Player disconnect event |
| `GoStrike_OnMapChange(mapName)` | Map change event |
| `GoStrike_RegisterCallbacks(callbacks)` | Register C++ callbacks |
| `GoStrike_GetABIVersion()` | Get ABI version for compatibility |

### Callbacks (C++ → Go)

C++ functions that Go can call:

| Callback | Description |
|----------|-------------|
| `log(level, tag, msg)` | Write to server log |
| `exec_command(cmd)` | Execute server command |
| `reply_to_command(slot, msg)` | Reply to command invoker |
| `get_player(slot)` | Get player information |
| `get_player_count()` | Get connected player count |
| `get_all_players(slots)` | Get all player slots |
| `kick_player(slot, reason)` | Kick a player |
| `get_map_name()` | Get current map |
| `get_max_players()` | Get max player slots |
| `get_tick_rate()` | Get server tick rate |
| `send_chat(slot, msg)` | Send chat message |
| `send_center(slot, msg)` | Send center message |

## Core Modules

### HTTP Module

Location: `internal/modules/http/`

The HTTP module provides the primary communication interface for GoStrike:

- **REST API**: Built-in endpoints for server/plugin management
- **Plugin Endpoints**: Plugins can register custom endpoints
- **Middleware**: CORS, rate limiting

Configuration: `configs/http.json`

```json
{
  "enabled": true,
  "host": "0.0.0.0",
  "port": 8080,
  "enable_cors": true,
  "cors_origins": "*",
  "rate_limit": 0
}
```

#### Built-in API Endpoints

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/health` | GET | Health check |
| `/api/status` | GET | GoStrike runtime status |
| `/api/plugins` | GET | List loaded plugins |
| `/api/modules` | GET | List core modules |
| `/api/routes` | GET | List all registered routes |

SDK Usage:

```go
// Register an endpoint
gostrike.RegisterGET("/api/myplugin/status", func(w http.ResponseWriter, r *http.Request) {
    gostrike.JSONSuccess(w, map[string]string{"status": "ok"})
})

// Route groups
api := gostrike.NewHTTPGroup("/api/myplugin")
api.GET("/players", handlePlayers)
api.POST("/kick", handleKick)
```

### Permissions Module

Location: `internal/modules/permissions/`

Provides admin authentication and authorization:

- **Admin Flags**: Granular permissions (kick, ban, slay, etc.)
- **Groups**: Permission groups (Admin, Moderator, VIP)
- **SteamID Mapping**: Link SteamIDs to permissions
- **Immunity**: Immunity levels for targeting

Configuration: `configs/admins.json`

```json
{
  "groups": [
    {"name": "admin", "flags": "abcdefghijklm", "immunity": 80}
  ],
  "admins": [
    {"steamid": "STEAM_0:0:12345", "groups": ["admin"]}
  ]
}
```

SDK Usage:

```go
// Check player permission
if player.HasPermission(gostrike.AdminKick) {
    // Can kick players
}

// In chat command handler
if !ctx.HasFlag(gostrike.AdminGeneric) {
    ctx.ReplyError("You do not have permission")
    return nil
}
```

### Database Module

Location: `internal/modules/database/`

Database abstraction supporting SQLite and MySQL:

- **Query Builder**: Type-safe query construction
- **Migrations**: Version-controlled schema changes
- **Connection Pooling**: Efficient connection management

Configuration: `configs/database.json` (optional)

SDK Usage:

```go
// Simple query
rows, err := gostrike.Query("SELECT * FROM players WHERE steam_id = ?", steamID)

// Query builder
query, args := gostrike.Table("players").
    Select("name", "score").
    Where("team = ?", "CT").
    OrderBy("score", true).
    Limit(10).
    BuildSelect()
```

## Plugin System

### Plugin Interface

Plugins implement the `plugin.Plugin` interface:

```go
type Plugin interface {
    Name() string
    Version() string
    Author() string
    Description() string
    Load(hotReload bool) error
    Unload(hotReload bool) error
}
```

### Plugin Lifecycle

```
┌───────────┐    Register    ┌──────────┐    Init   ┌─────────┐
│ Unloaded  │ ──────────────→│ Loading  │ ─────────→│ Loaded  │
└───────────┘                └──────────┘           └─────────┘
      ↑                           │                      │
      │                           │ Error                │ Unload
      │                           ↓                      │
      │                     ┌──────────┐                 │
      │                     │  Failed  │                 │
      │                     └──────────┘                 │
      │                                                  │
      └──────────────────────────────────────────────────┘
```

### Plugin Configuration

Plugins can be enabled/disabled via `configs/plugins.json`:

```json
{
  "plugins": {
    "Example Plugin": {"enabled": true},
    "Admin Tools": {"enabled": true}
  },
  "auto_enable_new": true
}
```

### Creating a Plugin

1. Create a new package in `plugins/`:

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

func (p *MyPlugin) Name() string        { return "My Plugin" }
func (p *MyPlugin) Version() string     { return "1.0.0" }
func (p *MyPlugin) Author() string      { return "Your Name" }
func (p *MyPlugin) Description() string { return "Description" }

func (p *MyPlugin) Load(hotReload bool) error {
    p.logger = gostrike.GetLogger("MyPlugin")
    
    // Register chat commands
    gostrike.RegisterChatCommand(gostrike.ChatCommandInfo{
        Name:        "hello",
        Description: "Say hello",
        Flags:       gostrike.ChatCmdPublic,
        Callback: func(ctx *gostrike.CommandContext) error {
            ctx.Reply("Hello, %s!", ctx.Player.Name)
            return nil
        },
    })
    
    return nil
}

func (p *MyPlugin) Unload(hotReload bool) error {
    gostrike.UnregisterChatCommand("hello")
    return nil
}

func init() {
    plugin.Register(&MyPlugin{})
}
```

2. Import in `cmd/gostrike/main.go`:

```go
import _ "github.com/corrreia/gostrike/plugins/myplugin"
```

3. Build and deploy: `make dev`

## Chat Command System

Plugins can register chat commands that players invoke with the `!` prefix.

### Registering Chat Commands

```go
gostrike.RegisterChatCommand(gostrike.ChatCommandInfo{
    Name:        "slap",
    Description: "Slap a player",
    Usage:       "<player> [damage]",
    MinArgs:     1,
    Flags:       gostrike.ChatCmdAdmin, // Requires admin permission
    Callback: func(ctx *gostrike.CommandContext) error {
        targetName := ctx.GetArg(0)
        damage := ctx.GetArgInt(1, 0)
        
        // Find and slap player...
        ctx.Reply("Slapped %s for %d damage", targetName, damage)
        return nil
    },
})
```

### Chat Command Flags

| Flag | Description |
|------|-------------|
| `ChatCmdPublic` | Anyone can use |
| `ChatCmdAdmin` | Requires admin permission |

### Command Context Methods

```go
ctx.Reply(format, args...)      // Send reply to player
ctx.ReplyError(format, args...) // Send error message
ctx.GetArg(index)               // Get argument by index
ctx.GetArgInt(index, default)   // Get int argument
ctx.GetArgFloat(index, default) // Get float argument
ctx.GetArgBool(index, default)  // Get bool argument
ctx.HasFlag(flag)               // Check admin permission
ctx.Player                      // Get player info
```

## Event System

### Event Types

- **Game Events**: `player_death`, `round_start`, etc.
- **Player Events**: Connect, disconnect
- **Map Events**: Map change

### Event Handlers

```go
// Typed event handler
gostrike.RegisterPlayerConnectHandler(func(event *gostrike.PlayerConnectEvent) gostrike.EventResult {
    event.Player.PrintToChat("Welcome!")
    return gostrike.EventContinue
}, gostrike.HookPost)

// Generic event handler
gostrike.RegisterGenericEventHandler("round_start", func(name string, event gostrike.Event) gostrike.EventResult {
    // Handle event
    return gostrike.EventContinue
}, gostrike.HookPost)
```

### Event Results

| Result | Description |
|--------|-------------|
| `EventContinue` | Continue processing |
| `EventChanged` | Event data was modified |
| `EventHandled` | Event was handled, skip remaining handlers |
| `EventStop` | Stop the event entirely |

## Timer System

```go
// One-shot timer
gostrike.After(5.0, func() {
    // Called after 5 seconds
})

// Repeating timer
timer := gostrike.Every(60.0, func() {
    // Called every 60 seconds
})

// Stop timer
timer.Stop()
```

## Build System

All builds use Docker for GLIBC compatibility:

```bash
make build       # Build Go library + native plugin
make deploy      # Copy to server volume
make dev         # Build + deploy + restart
```

### Build Process

1. **Go Library** (`golang:1.21-bullseye`):
   - Compiles `cmd/gostrike/main.go`
   - Produces `libgostrike_go.so` (c-shared)
   - GLIBC 2.31 compatible

2. **Native Plugin** (`steamrt/sniper/sdk`):
   - Compiles `native/src/*.cpp`
   - Produces `gostrike.so`
   - Steam Runtime compatible

### Native Plugin Build Modes

The native plugin supports two build modes:

#### Stub SDK (Development)

Uses minimal stub headers for development without full SDK:

```bash
make native-stub
```

Features:
- Fast compilation
- No SDK dependencies
- All engine interactions are stubbed (console output only)

#### Full SDK (Production)

Uses HL2SDK-CS2 and Metamod:Source for full engine integration:

```bash
make native-proto    # Generate protobuf headers (one-time)
make native-host     # Build with full SDK
```

Requirements:
- Generated protobuf headers (`native/generated/*.pb.h`)
- HL2SDK-CS2 in `external/hl2sdk-cs2/`
- Metamod:Source in `external/metamod-source/`

### Protobuf Header Generation

CS2's SDK requires protobuf headers generated from `.proto` files. The SDK bundles protobuf 3.21.8, which may conflict with system protobuf versions.

The `generate_protos.sh` script handles this:

```bash
./native/scripts/generate_protos.sh
```

This script:
1. Builds `protoc` from SDK's bundled protobuf 3.21.8 source
2. Generates `.pb.h` and `.pb.cc` files in `native/generated/`
3. Ensures version compatibility with SDK headers

Generated files:
- `network_connection.pb.h` - Network connection types
- `networkbasetypes.pb.h` - Base network types
- `netmessages.pb.h` - Network messages
- `usermessages.pb.h` - User messages (chat, etc.)
- `source2_steam_stats.pb.h` - Steam stats

### CS2 UserMessage System Limitations

CS2's UserMessage system (used for chat, center messages, etc.) has integration challenges:

1. **Protobuf classes are `final`**: Generated classes like `CUserMessageTextMsg` are marked `final`, preventing the standard `CNetMessagePB<T>` template inheritance pattern.

2. **Interface complexity**: Sending messages requires:
   - `INetworkMessages` interface for message allocation
   - `IGameEventSystem` interface for posting messages
   - Correct protobuf field population

**Current Status**: Chat functions (`PrintToAll`, `PrintToChat`) output to server console. Full in-game chat requires reimplementing the message allocation pattern used by CounterStrikeSharp.

## Configuration Files

| File | Description |
|------|-------------|
| `configs/gostrike.json` | Main configuration (log_level, etc.) |
| `configs/admins.json` | Admin permissions |
| `configs/plugins.json` | Plugin enable/disable |
| `configs/http.json` | HTTP server config |
| `configs/database.json` | Database config (optional) |

## Data Flow

```
CS2 Server Event
        │
        ▼
Metamod:Source (hooks)
        │
        ▼
gostrike.cpp (C++ plugin)
        │
        ▼
go_bridge.cpp (dlopen/dlsym)
        │
        ▼
exports.go (CGO exports)
        │
        ▼
runtime/dispatcher.go
        │
        ├──→ Core Modules (permissions, http, database)
        │
        └──→ Plugin Handlers
                │
                ▼
        Plugin Business Logic
                │
                ▼
        callbacks.go (CGO callbacks)
                │
                ▼
        go_bridge.cpp (function pointers)
                │
                ▼
        gostrike.cpp (execute action)
                │
                ▼
        CS2 Server (effect applied)
```

## Contributing Plugins

1. Fork the repository
2. Create your plugin in `plugins/yourplugin/`
3. Add import to `cmd/gostrike/main.go`
4. Test locally with `make dev`
5. Submit a pull request

See the [example plugin](plugins/example/example.go) for reference.
