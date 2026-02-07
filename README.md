# GoStrike

A Counter-Strike 2 server modding framework using Go as the plugin language/runtime.

GoStrike provides a thin Metamod:Source C++ plugin that embeds a Go runtime, allowing plugin authors to write server plugins in Go with access to game events, timers, player management, and HTTP APIs.

## TL;DR - Quick Start

```bash
# First-time setup (requires ~70GB disk space)
git clone --recursive https://github.com/corrreia/gostrike && cd gostrike
make setup                # Start CS2 download (~60GB)
make server-logs          # Wait for "VAC secure mode is activated"
make server-stop          # Stop server
make metamod-install      # Install Metamod:Source
make build deploy         # Build and deploy GoStrike
make server-start         # Start server with plugin

# Development cycle (after initial setup)
make dev                  # Build, deploy, restart - one command!
```

## Features

- **Go Plugin SDK**: Write CS2 server plugins in Go with a comprehensive API
- **Entity System**: Full Source 2 entity access via CSchemaSystem - read/write any entity property
- **Schema Code Generation**: `schemagen` tool generates typed Go wrappers for entity classes
- **GameData System**: Cross-update compatibility via signature scanning and offset resolution
- **ConVar System**: Read and write server ConVars programmatically
- **Player Pawn/Controller**: Access both CCSPlayerController and CCSPlayerPawn entities
- **Game Functions**: Respawn, slay, teleport, change team via native game functions
- **Event System**: Hook game events (player_connect, player_death, round_start, etc.)
- **Entity Lifecycle Events**: Track entity creation, spawning, and deletion
- **Chat Commands**: Register chat commands (`!command`) for player interaction
- **In-Game Messaging**: Proper UTIL_ClientPrint integration (chat, center, console, alert)
- **Menu System**: Chat-based numbered menus with timeout and selection handling
- **Target Patterns**: Resolve `@all`, `@alive`, `@ct`, `@t`, `@me`, `#slot`, name match
- **Localization**: JSON-based i18n with per-locale translations and placeholder support
- **HTTP API**: RESTful API for server/plugin management and external integrations
- **Timer System**: Schedule delayed and repeating callbacks
- **Player Management**: Access player information, kick players, send messages
- **Permissions Module**: Admin flags, groups, immunity, and command overrides
- **Database Module**: SQLite/MySQL abstraction with query builder
- **Plugin Dependencies**: Topological sort-based load ordering with dependency validation
- **Plugin Configuration**: Enable/disable plugins via config
- **Panic Recovery**: Go panics don't crash the server
- **Stable C ABI**: Versioned callback interface between C++ and Go

## Prerequisites

- **Docker** and **Docker Compose** (required for builds and server)
- **Git** (with submodule support)
- **~70GB disk space** (CS2 server is ~60GB)
- **Linux x86_64** host

That's it! All builds happen inside Docker containers, so you don't need Go, CMake, or any other build tools installed locally.

## First-Time Setup

### Step 1: Clone the Repository

```bash
git clone --recursive https://github.com/corrreia/gostrike
cd gostrike
```

If you already cloned without `--recursive`:
```bash
git submodule update --init --recursive
```

### Step 2: Download CS2 Server

```bash
make setup
```

This starts the CS2 dedicated server container, which will download ~60GB of game files. Monitor the progress:

```bash
make server-logs
```

**Wait for this message before continuing:**
```
VAC secure mode is activated
```

This typically takes 30-60 minutes depending on your internet speed.

### Step 3: Install Metamod and Build

Once CS2 is downloaded, stop the server and install the plugin framework:

```bash
make server-stop
make metamod-install
make build deploy
```

### Step 4: Start the Server

```bash
make server-start
```

### Step 5: Verify Installation

Open the server console and check that GoStrike loaded:

```bash
make server-console
```

Then type:
```
meta list
```

You should see:
```
Listing 1 plugin:
  [01] GoStrike (0.1.0) by GoStrike Team
```

Press `Ctrl+C` to detach from the console (the server keeps running).

### Step 6: Test the HTTP API

If the HTTP module is enabled (default: `configs/http.json`), you can check the API:

```bash
curl http://localhost:8080/health
curl http://localhost:8080/api/status
curl http://localhost:8080/api/plugins
```

## Communication Architecture

GoStrike uses two main interfaces:

### HTTP API (Primary Interface)

The HTTP module provides the main interface for managing GoStrike:

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/health` | GET | Health check |
| `/api/status` | GET | GoStrike runtime status |
| `/api/plugins` | GET | List loaded plugins |
| `/api/modules` | GET | List core modules |
| `/api/routes` | GET | List all API routes |

Configure in `configs/http.json`:

```json
{
  "enabled": true,
  "host": "0.0.0.0",
  "port": 8080,
  "enable_cors": true
}
```

### Chat Commands (Player Interaction)

Plugins register chat commands with the `!` prefix for player interaction:

```
!help     - Show available commands
!players  - List connected players
!info     - Show server info
```

## Development Workflow

Once set up, the development cycle is simple:

```bash
# Make changes to Go code in plugins/ or pkg/, then:
make dev    # Builds, deploys, and restarts server

# Or step by step:
make build           # Build Go library + native plugin
make deploy          # Copy to server volume
make server-restart  # Restart to reload plugin
```

View logs to see your plugin output:
```bash
make server-logs
```

## Useful Commands

| Command | Description |
|---------|-------------|
| `make build` | Build GoStrike (Go + native) in Docker |
| `make deploy` | Deploy to server volume |
| `make dev` | Build + deploy + restart (full dev cycle) |
| `make server-start` | Start the CS2 server |
| `make server-stop` | Stop the CS2 server |
| `make server-restart` | Restart (for plugin changes) |
| `make server-logs` | View server logs (follow mode) |
| `make server-console` | Attach to CS2 console |
| `make server-shell` | Bash shell into container |
| `make server-status` | Check server and plugin status |
| `make server-clean` | Delete CS2 data (~60GB) |
| `make help` | Show all available commands |

## Why Docker Builds?

GoStrike builds inside Docker containers to ensure **GLIBC compatibility**.

The CS2 dedicated server runs on Steam Runtime (based on older Debian), which uses GLIBC 2.31. If you build on a modern Linux host (Ubuntu 22.04+, Arch, etc.), your binaries will link against a newer GLIBC (2.34+) and fail to load on the server with errors like:

```
GLIBC_2.34 not found (required by gostrike.so)
```

The `make build` command uses:
- `golang:1.21-bullseye` for Go builds (GLIBC 2.31)
- `registry.gitlab.steamos.cloud/steamrt/sniper/sdk` for native builds (Steam Runtime compatible)

This is all handled automatically - just use `make build` and it works.

## Native Plugin Build (Advanced)

The native C++ plugin can be built locally for development:

```bash
# Build with stub SDK (development, no engine integration)
make native-stub

# Build with full HL2SDK (requires protobuf headers)
make native-proto    # Generate protobuf headers from SDK (one-time)
make native-host     # Build with full SDK
```

### Protobuf Requirements

The CS2 SDK requires generated protobuf headers. The SDK bundles protobuf 3.21.8, but these headers must be generated from `.proto` files:

```bash
# Generate protobuf headers using SDK's bundled protoc
./native/scripts/generate_protos.sh
```

This script builds `protoc` from the SDK's bundled protobuf source (to avoid version conflicts with system protobuf) and generates the required `.pb.h` files.

## Architecture

```
┌─────────────────────────────────────────────┐
│           CS2 Dedicated Server              │
├─────────────────────────────────────────────┤
│             Metamod:Source                  │
├─────────────────────────────────────────────┤
│      GoStrike Native Plugin (C++)           │
│  ┌─────────┬──────────┬─────────────────┐  │
│  │ Schema  │ GameData │  Memory Module  │  │
│  │ System  │ Resolver │  Scanner        │  │
│  └─────────┴──────────┴─────────────────┘  │
├─────────────────────────────────────────────┤
│       C ABI Bridge Layer (Versioned)        │
├─────────────────────────────────────────────┤
│      Go Runtime (libgostrike_go.so)         │
├─────────────────────────────────────────────┤
│              Core Modules                   │
│  ┌───────────┬────────┬────────────────┐   │
│  │Permissions│  HTTP  │    Database    │   │
│  └───────────┴────────┴────────────────┘   │
├─────────────────────────────────────────────┤
│     Plugin Manager (Dependency Sort)        │
├─────────────────────────────────────────────┤
│      Plugin A │ Plugin B │ Plugin C         │
└─────────────────────────────────────────────┘
```

For detailed architecture information, see [ARCHITECTURE.md](ARCHITECTURE.md).

## Core Modules

### HTTP Module

Embedded HTTP server for REST APIs and webhooks.

Configuration: `configs/http.json`

```go
// Register custom endpoints in your plugin
gostrike.RegisterGET("/api/myplugin/status", func(w http.ResponseWriter, r *http.Request) {
    gostrike.JSONSuccess(w, map[string]string{"status": "ok"})
})
```

### Permissions Module

String-based permissions with dot notation, role-based access control, and wildcard matching. Stored in SQLite (`data/permissions.db`), managed via REST API.

See [Permissions Documentation](docs/permissions.md) for full details.

```go
// Register plugin permissions in Load()
gostrike.RegisterPermission("myplugin.give", "Give weapons")

// Check permissions
if player.HasPermission("myplugin.give") { ... }
if !ctx.RequirePermission("myplugin.admin") { return nil }

// Protect chat commands
gostrike.RegisterChatCommand(gostrike.ChatCommandInfo{
    Name:       "give",
    Permission: "myplugin.give", // empty = public
    Callback:   handler,
})
```

### Database Module

SQLite/MySQL abstraction with query builder (disabled by default).

```go
// Simple query
rows, _ := gostrike.Query("SELECT * FROM players WHERE steam_id = ?", steamID)

// Query builder
query, args := gostrike.Table("players").
    Select("name", "score").
    Where("team = ?", "CT").
    OrderBy("score", true).
    BuildSelect()
```

## Writing Plugins

Plugins are Go packages that implement the `plugin.Plugin` interface:

```go
package myplugin

import (
    "github.com/corrreia/gostrike/pkg/gostrike"
    "github.com/corrreia/gostrike/pkg/plugin"
)

type MyPlugin struct {
    plugin.BasePlugin
}

func (p *MyPlugin) Name() string        { return "My Plugin" }
func (p *MyPlugin) Version() string     { return "1.0.0" }
func (p *MyPlugin) Author() string      { return "Your Name" }
func (p *MyPlugin) Description() string { return "My awesome plugin" }

func (p *MyPlugin) Load(hotReload bool) error {
    // Register a chat command (!hello)
    gostrike.RegisterChatCommand(gostrike.ChatCommandInfo{
        Name:        "hello",
        Description: "Say hello",
        Flags:       gostrike.ChatCmdPublic,
        Callback: func(ctx *gostrike.CommandContext) error {
            ctx.Reply("Hello, %s!", ctx.Player.Name)
            return nil
        },
    })

    // Register an event handler
    gostrike.RegisterPlayerConnectHandler(func(e *gostrike.PlayerConnectEvent) gostrike.EventResult {
        gostrike.GetLogger("MyPlugin").Info("Player connected: %s", e.Player.Name)
        return gostrike.EventContinue
    }, gostrike.HookPost)

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

To include your plugin, add an import in `cmd/gostrike/main.go`:

```go
import _ "path/to/myplugin"
```

Then rebuild: `make dev`

## Project Structure

```
gostrike/
├── cmd/
│   ├── gostrike/           # c-shared entry point
│   └── schemagen/          # Entity code generator tool
├── configs/                # Configuration files
│   ├── gostrike.json       # Main configuration
│   ├── http.json           # HTTP server config
│   ├── plugins.json        # Plugin enable/disable
│   ├── gamedata/           # GameData signatures/offsets
│   └── schema/             # Entity schema definitions
├── docker/                 # Docker development environment
│   ├── docker-compose.yml
│   ├── data/               # CS2 server data (gitignored)
│   └── scripts/            # Server setup scripts
├── docs/                   # Documentation
│   ├── permissions.md      # Permissions system & API reference
│   └── plugin-development.md
├── external/               # Git submodules (SDKs)
├── internal/               # Internal implementation
│   ├── bridge/             # CGO exports and callbacks
│   ├── manager/            # Plugin lifecycle and dependencies
│   ├── modules/            # Core modules
│   │   ├── permissions/    # String-based permissions (SQLite + cache)
│   │   ├── http/           # HTTP server
│   │   └── database/       # Database abstraction
│   └── runtime/            # Event/command/entity dispatch
├── native/                 # C++ Metamod plugin
│   ├── include/            # Headers (ABI, stubs)
│   └── src/                # Source files
├── pkg/                    # Go SDK (public API)
│   ├── gostrike/           # Core types and functions
│   │   └── entities/       # Generated typed entity wrappers
│   └── plugin/             # Plugin interface
├── plugins/                # Community plugins
│   └── example/            # Example plugin
├── scripts/                # Build scripts
│   └── docker-build.sh     # Docker build script
├── ARCHITECTURE.md         # Detailed architecture docs
├── CREDITS.md              # Attribution and credits
└── tests/                  # Test suites
```

## Plugin Configuration

Enable/disable plugins without recompiling via `configs/plugins.json`:

```json
{
  "plugins": {
    "Example Plugin": {"enabled": true},
    "My Custom Plugin": {"enabled": false}
  },
  "auto_enable_new": true
}
```

Plugin names must match the `Name()` method of the plugin.

## Troubleshooting

### `meta list` shows `<NOFILE>` or `<FAILED>`

1. Check that GoStrike is built and deployed:
   ```bash
   make server-status
   ```

2. If files are missing, rebuild and deploy:
   ```bash
   make build deploy server-restart
   ```

3. Check server logs for specific errors:
   ```bash
   make server-logs
   ```

### `GLIBC_2.34 not found` error

You built on the host instead of in Docker. Always use:
```bash
make build    # NOT make go-host or make native-host
```

### HTTP API not responding

Check if HTTP module is enabled in `configs/http.json`:
```json
{
  "enabled": true,
  "port": 8080
}
```

### Server won't start / container keeps restarting

Check if CS2 finished downloading:
```bash
make server-status
```

If CS2 isn't fully installed, wait for the download to complete.

### Permission errors

The Docker container runs as UID 1000. If you have permission issues:
```bash
sudo chown -R 1000:1000 docker/data/cs2
```

## Manual Installation (Production)

For production servers without Docker:

1. Install Metamod:Source on your CS2 server
2. Build GoStrike on a compatible system (or use Docker builds)
3. Copy files to your CS2 server:
   ```
   csgo/addons/metamod/gostrike.vdf
   csgo/addons/gostrike/gostrike.so
   csgo/addons/gostrike/bin/libgostrike_go.so
   csgo/addons/gostrike/configs/gostrike.json
   csgo/addons/gostrike/configs/http.json
   ```
4. Add to `csgo/addons/metamod/metaplugins.ini`:
   ```
   addons/metamod/gostrike.vdf
   ```
5. Ensure `gameinfo.gi` has Metamod entry
6. Restart your server

## Contributing Plugins

GoStrike uses a PR-based plugin contribution model. To add your plugin:

1. **Fork** this repository
2. **Create** your plugin in `plugins/yourplugin/`
3. **Add** the import to `cmd/gostrike/main.go`:
   ```go
   import _ "github.com/corrreia/gostrike/plugins/yourplugin"
   ```
4. **Test** locally with `make dev`
5. **Submit** a pull request

### Plugin Guidelines

- Follow Go best practices and naming conventions
- Include a clear description in your plugin's `Description()` method
- Handle errors gracefully and log appropriately
- Clean up resources in `Unload()` (stop timers, close connections, unregister commands)
- Use the permissions module for admin chat commands
- Document your plugin's chat commands and features

See [plugins/example](plugins/example/) for a reference implementation.

## License

MIT License - see LICENSE file for details.

## Acknowledgments

- [CounterStrikeSharp](https://github.com/roflmuffin/CounterStrikeSharp) - Architecture inspiration
- [Metamod:Source](https://www.metamodsource.net/) - Plugin loading framework
- [AlliedModders](https://alliedmods.net/) - Community resources
