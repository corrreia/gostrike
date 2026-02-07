# GoStrike Architecture

This document describes the architecture of GoStrike, a Counter-Strike 2 server modding framework using Go.

## Overview

GoStrike consists of three main layers:

1. **Native Layer** - C++ Metamod plugin with schema, gamedata, and memory systems
2. **Bridge Layer** - Versioned C ABI between C++ and Go (CGO)
3. **Go Runtime** - Plugin system, core modules, and public SDK

```
┌─────────────────────────────────────────────────────────────────┐
│                     CS2 Dedicated Server                        │
├─────────────────────────────────────────────────────────────────┤
│                      Metamod:Source                             │
├─────────────────────────────────────────────────────────────────┤
│              GoStrike Native Plugin (gostrike.so)               │
│  ┌──────────┬────────────┬──────────────┬───────────────────┐  │
│  │  Schema  │  GameData  │    Memory    │  Game Functions   │  │
│  │  System  │  Resolver  │    Module    │  (via gamedata)   │  │
│  └──────────┴────────────┴──────────────┴───────────────────┘  │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │                 C ABI Bridge (Versioned)                 │   │
│  │  ┌─────────────────────────────────────────────────┐    │   │
│  │  │            Go Runtime (libgostrike_go.so)        │    │   │
│  │  │  ┌───────────────────────────────────────────┐  │    │   │
│  │  │  │              Core Modules                  │  │    │   │
│  │  │  │  ┌─────────┐ ┌──────┐ ┌──────────────┐   │  │    │   │
│  │  │  │  │Permissions│ │ HTTP │ │   Database   │   │  │    │   │
│  │  │  │  └─────────┘ └──────┘ └──────────────┘   │  │    │   │
│  │  │  ├───────────────────────────────────────────┤  │    │   │
│  │  │  │      Plugin Manager (Dependency Sort)      │  │    │   │
│  │  │  │  ┌────────┐ ┌────────┐ ┌────────────┐    │  │    │   │
│  │  │  │  │Plugin A│ │Plugin B│ │  Plugin C  │    │  │    │   │
│  │  │  │  └────────┘ └────────┘ └────────────┘    │  │    │   │
│  │  │  └───────────────────────────────────────────┘  │    │   │
│  │  └─────────────────────────────────────────────────┘    │   │
│  └─────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────┘
```

## Directory Structure

```
gostrike/
├── cmd/
│   ├── gostrike/               # Go c-shared entry point (main.go)
│   └── schemagen/              # Entity code generator tool
├── configs/                    # Configuration files
│   ├── gostrike.json           # Main config (log level, etc.)
│   ├── http.json               # HTTP server config
│   ├── plugins.json            # Plugin enable/disable
│   ├── gamedata/               # GameData signatures and offsets
│   │   └── gamedata.json       # Function signatures (derived from CSSharp)
│   └── schema/                 # Entity schema definitions
│       └── cs2_schema.json     # CS2 entity class/field definitions
├── docker/                     # Docker development environment
│   ├── docker-compose.yml      # CS2 server container
│   └── scripts/                # Server setup scripts
├── docs/                       # Documentation
│   ├── permissions.md          # Permissions system & API reference
│   └── plugin-development.md   # Plugin development guide
├── external/                   # Git submodules
│   ├── hl2sdk-cs2/             # HL2SDK for CS2
│   └── metamod-source/         # Metamod:Source
├── internal/                   # Internal implementation (not public API)
│   ├── bridge/                 # CGO exports and callbacks
│   │   ├── callbacks.go        # C++ -> Go callback wrappers (V1-V4)
│   │   ├── exports.go          # Go -> C++ exported functions
│   │   └── types.go            # Type conversions (PlayerInfo, etc.)
│   ├── manager/                # Plugin lifecycle management
│   │   ├── manager.go          # Loading, unloading, config
│   │   ├── loader.go           # Load ordering and topological sort
│   │   └── registry.go         # Static plugin registration
│   ├── modules/                # Core modules
│   │   ├── module.go           # Module interface
│   │   ├── permissions/        # Admin flags, groups, overrides
│   │   ├── http/               # Embedded HTTP server
│   │   └── database/           # SQLite/MySQL abstraction
│   ├── runtime/                # Runtime dispatch
│   │   ├── dispatcher.go       # Event, entity, and command dispatch
│   │   ├── chat.go             # Chat command system
│   │   ├── timers.go           # Timer system
│   │   └── runtime.go          # Init/shutdown orchestration
│   └── shared/                 # Shared types between packages
├── native/                     # C++ Metamod plugin
│   ├── CMakeLists.txt          # CMake build config
│   ├── include/
│   │   ├── gostrike_abi.h      # C ABI definition (V1-V4)
│   │   └── stub/               # Stub SDK headers (dev builds)
│   ├── src/
│   │   ├── gostrike.cpp/h      # Metamod plugin entry point
│   │   ├── go_bridge.cpp/h     # Go library loading + callback impl
│   │   ├── schema.cpp/h        # CSchemaSystem field resolution
│   │   ├── gameconfig.cpp/h    # GameData JSON loading
│   │   ├── memory_module.cpp/h # Module discovery + sig scanning
│   │   ├── entity_system.cpp/h # Entity lifecycle (IEntityListener)
│   │   ├── player_manager.cpp/h # Controller/pawn resolution
│   │   ├── convar_manager.cpp/h # ConVar read/write via ICvar
│   │   ├── game_functions.cpp/h # Respawn, slay, teleport, etc.
│   │   ├── chat_manager.cpp/h  # UTIL_ClientPrint resolution
│   │   └── utils.h             # CallVirtual<T> template
│   └── scripts/
│       └── generate_protos.sh  # Protobuf header generator
├── pkg/                        # Public SDK (plugin-facing API)
│   ├── gostrike/               # Core SDK types and functions
│   │   ├── player.go           # Player API (pawn, controller, game funcs)
│   │   ├── server.go           # Server API (commands, messaging)
│   │   ├── entity.go           # Entity system + schema property access
│   │   ├── convar.go           # ConVar read/write
│   │   ├── command.go          # Chat command registration
│   │   ├── event.go            # Event handler registration
│   │   ├── timer.go            # Timer API
│   │   ├── menu.go             # Chat-based menu system
│   │   ├── target.go           # Target pattern resolution (@all, @ct, etc.)
│   │   ├── i18n.go             # Localization / translation
│   │   ├── http.go             # HTTP endpoint registration
│   │   ├── database.go         # Database access
│   │   ├── permissions.go      # Permission checks
│   │   └── entities/           # Generated typed entity wrappers
│   │       └── generated.go    # Auto-generated by schemagen
│   └── plugin/                 # Plugin interface
│       └── plugin.go           # Plugin, BasePlugin, Register()
├── plugins/                    # Plugins
│   └── example/                # Example plugin (reference impl)
├── scripts/                    # Build scripts
│   └── docker-build.sh         # Docker-based build
├── ARCHITECTURE.md             # This file
├── CREDITS.md                  # Attribution (CSSharp, etc.)
└── Makefile                    # Build, deploy, server management
```

## C ABI Bridge

The bridge between C++ and Go uses a stable, versioned C ABI defined in `native/include/gostrike_abi.h`. Entity pointers are passed as opaque `uintptr_t` values and never dereferenced on the Go side.

### Exports (Go functions called by C++)

| Function | Description |
|----------|-------------|
| `GoStrike_Init()` | Initialize Go runtime |
| `GoStrike_Shutdown()` | Shutdown Go runtime |
| `GoStrike_OnTick(deltaTime)` | Called every server tick |
| `GoStrike_OnEvent(event, isPost)` | Game event dispatch |
| `GoStrike_OnChatMessage(slot, msg)` | Chat message dispatch (for `!commands`) |
| `GoStrike_OnPlayerConnect(player)` | Player connect event |
| `GoStrike_OnPlayerDisconnect(slot, reason)` | Player disconnect event |
| `GoStrike_OnMapChange(mapName)` | Map change event |
| `GoStrike_OnEntityCreated(index, classname)` | Entity created |
| `GoStrike_OnEntitySpawned(index, classname)` | Entity spawned |
| `GoStrike_OnEntityDeleted(index)` | Entity deleted |
| `GoStrike_RegisterCallbacks(callbacks)` | Register C++ callback table |

### Callbacks (C++ functions called by Go)

Callbacks are organized by ABI version. Each version extends the `gs_callbacks_t` struct.

**V1 - Core:**

| Callback | Description |
|----------|-------------|
| `log(level, tag, msg)` | Write to server log |
| `exec_command(cmd)` | Execute server command |
| `get_player(slot)` | Get player information |
| `kick_player(slot, reason)` | Kick a player |
| `get_map_name()` | Get current map name |
| `get_max_players()` | Max player count |
| `get_tick_rate()` | Server tick rate |

**V2 - Schema & Entities:**

| Callback | Description |
|----------|-------------|
| `schema_get_offset(class, field)` | Get schema field offset |
| `entity_get_int/float/bool/string/vector(...)` | Read entity property |
| `entity_set_int/float/bool/vector(...)` | Write entity property |
| `get_entity_by_index(index)` | Look up entity by index |
| `get_entity_classname(entity)` | Get entity class name |
| `resolve_gamedata(name)` | Resolve gamedata signature |
| `get_gamedata_offset(name)` | Get gamedata offset |

**V3 - ConVar & Game Functions:**

| Callback | Description |
|----------|-------------|
| `convar_get/set_int/float/string(name, ...)` | ConVar read/write |
| `get_player_controller(slot)` | Get CCSPlayerController entity |
| `get_player_pawn(slot)` | Get CCSPlayerPawn entity |
| `player_respawn(slot)` | Respawn player |
| `player_change_team(slot, team)` | Change team |
| `player_slay(slot)` | Kill player |
| `player_teleport(slot, pos, angles, vel)` | Teleport player |

**V4 - Communication:**

| Callback | Description |
|----------|-------------|
| `client_print(slot, dest, msg)` | UTIL_ClientPrint (per-player) |
| `client_print_all(dest, msg)` | UTIL_ClientPrintAll (broadcast) |

### CGO Pattern

C inline helpers in `callbacks.go` accept `uintptr_t` (not `void*`) to avoid Go vet warnings about `unsafe.Pointer`. The C helpers cast to `void*` internally:

```c
// In callbacks.go CGO preamble
static inline int32_t call_entity_get_int(gs_entity_get_int_t fn, uintptr_t ent, ...) {
    return fn((void*)ent, ...);
}
```

```go
// Go side - never touches unsafe.Pointer for entity pointers
result := C.call_entity_get_int(callbacks.entity_get_int, C.uintptr_t(entityPtr), ...)
```

## Native Layer (C++)

### Schema System (`schema.cpp`)

Accesses Source 2's `CSchemaSystem` to resolve entity property offsets at runtime. Field offsets are cached using FNV-1a hashing. When writing to networked fields, `SetStateChanged()` is called to notify the engine.

### GameData System (`gameconfig.cpp`)

Loads function signatures and offsets from `configs/gamedata/gamedata.json`. On startup, signatures are scanned in the appropriate game module (`libserver.so`, `libengine2.so`) and the resulting addresses are cached. This provides cross-update compatibility - when a game update changes addresses, only the gamedata JSON needs updating.

### Memory Module (`memory_module.cpp`)

Discovers loaded game modules via `dl_iterate_phdr()` on Linux. Provides byte-pattern signature scanning with wildcard support and ELF symbol table lookup.

### Game Functions (`game_functions.cpp`)

Wraps common game operations (respawn, slay, teleport, change team) by resolving function addresses from gamedata and calling them via `CallVirtual<T>` (vtable offset) or direct function pointers.

### Player Manager (`player_manager.cpp`)

Handles the CS2 dual-entity player model:
- **CCSPlayerController** (persistent, entity index = slot + 1) - holds SteamID, name, money, scores
- **CCSPlayerPawn** (physical, resolved via `m_hPlayerPawn` CHandle) - holds health, position, weapons

### Chat Manager (`chat_manager.cpp`)

Resolves `UTIL_ClientPrint` and `UTIL_ClientPrintAll` from gamedata for proper in-game messaging (chat, center, console, alert HUD destinations).

## Plugin System

### Plugin Interface

```go
type Plugin interface {
    Slug() string                           // Unique ID (namespacing)
    Name() string                           // Display name
    Version() string
    Author() string
    Description() string
    DefaultConfig() map[string]interface{}  // Auto-generated config
    Load(hotReload bool) error
    Unload(hotReload bool) error
}
```

### Plugin Lifecycle

```
Register() ──→ SortByDependencies() ──→ ValidateDeps()
                                              │
                                      ┌───────┴───────┐
                                      │               │
                                      ▼               ▼
                               ┌──────────┐    ┌──────────┐
                               │ Loading  │    │  Failed  │
                               └────┬─────┘    └──────────┘
                                    │
                            ┌───────┴───────┐
                            │               │
                            ▼               ▼
                     ┌──────────┐    ┌──────────┐
                     │  Loaded  │    │  Failed  │
                     └────┬─────┘    └──────────┘
                          │
                          │ Unload()
                          ▼
                     ┌──────────┐
                     │ Unloaded │
                     └──────────┘
```

Plugins are sorted by load order (Early/Normal/Late) and then topologically by declared dependencies before loading.

### Plugin Features

- **Slug-based namespacing** - HTTP routes (`/api/plugins/<slug>/`), database files, config files
- **Auto-config** - `DefaultConfig()` generates `configs/plugins/<slug>.json` on first load
- **Dependencies** - Declare required/optional dependencies; validated before load
- **Hot reload** - `Load(hotReload=true)` / `Unload(hotReload=true)`
- **Panic recovery** - Plugin panics are caught and don't crash the server

## Entity System

### Schema Property Access

Plugins can read/write any entity property by class and field name:

```go
entity := gostrike.GetEntityByIndex(42)
health, _ := entity.GetPropInt("CBaseEntity", "m_iHealth")
entity.SetPropInt("CBaseEntity", "m_iHealth", 200)
```

### Generated Typed Wrappers

The `schemagen` tool generates typed Go wrappers from `configs/schema/cs2_schema.json`:

```bash
go run ./cmd/schemagen
```

This produces `pkg/gostrike/entities/generated.go` with types like:

```go
pawn := entities.NewCCSPlayerPawnBase(player.GetPawn())
health := pawn.Health()
pawn.SetArmorValue(100)
```

### Entity Lifecycle

Entity creation, spawning, and deletion events are dispatched to Go handlers via the `IEntityListener` interface in C++.

## Core Modules

### Permissions

String-based permissions with dot notation, role-based access control, and wildcard matching.

- Storage: `data/permissions.db` (SQLite, self-contained)
- API: `/api/permissions/*` (roles, players, permissions CRUD)
- See: `docs/permissions.md`

### HTTP

Embedded HTTP server with REST API, CORS, and plugin route namespacing.

- Config: `configs/http.json`
- Built-in: `/health`, `/api/status`, `/api/plugins`, `/api/modules`, `/api/routes`

### Database

SQLite/MySQL abstraction with query builder. Per-plugin isolated databases at `data/plugins/<slug>.db`.

## Data Flow

```
CS2 Game Event
      │
      ▼
Metamod:Source (SourceHook)
      │
      ▼
gostrike.cpp (C++ hooks)
      │
      ▼
go_bridge.cpp (dlsym'd Go exports)
      │
      ▼
exports.go (CGO exports)
      │
      ▼
runtime/dispatcher.go
      │
      ├──→ Core Modules (permissions, http, database)
      │
      └──→ Plugin Handlers (events, commands, entities)
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
      C++ implementation (schema, gamedata, game functions)
              │
              ▼
      CS2 Server (effect applied)
```

## Build System

All production builds use Docker for GLIBC 2.31 compatibility with CS2's Steam Runtime.

| Command | What it does |
|---------|-------------|
| `make build` | Build Go library + native plugin in Docker |
| `make deploy` | Copy binaries to CS2 server volume |
| `make dev` | Build + deploy + restart server |
| `make setup` | First-time CS2 server download |

The build produces two binaries:
- `build/libgostrike_go.so` - Go runtime (built in `golang:1.21-bullseye`)
- `build/native/gostrike.so` - C++ Metamod plugin (built in Steam Runtime SDK)

See the [Makefile](Makefile) and `make help` for all targets.
