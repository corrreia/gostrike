# GoStrike

A Counter-Strike 2 server modding framework using Go as the plugin language/runtime.

GoStrike provides a thin Metamod:Source C++ plugin that embeds a Go runtime, allowing plugin authors to write server plugins in Go with access to game events, commands, timers, and player management.

## Features

- **Go Plugin SDK**: Write CS2 server plugins in Go
- **Event System**: Hook game events (player_connect, player_death, round_start, etc.)
- **Command System**: Register server and chat commands
- **Timer System**: Schedule delayed and repeating callbacks
- **Player Management**: Access player information, kick players, send messages
- **Panic Recovery**: Go panics don't crash the server
- **Stable C ABI**: Clean boundary between C++ and Go

## Architecture

```
┌─────────────────────────────────────────┐
│         CS2 Dedicated Server            │
├─────────────────────────────────────────┤
│           Metamod:Source                │
├─────────────────────────────────────────┤
│    GoStrike Native Plugin (C++)         │
├─────────────────────────────────────────┤
│         C ABI Bridge Layer              │
├─────────────────────────────────────────┤
│    Go Runtime (libgostrike_go.so)       │
├─────────────────────────────────────────┤
│         Plugin Manager                  │
├─────────────────────────────────────────┤
│    Plugin A │ Plugin B │ Plugin C       │
└─────────────────────────────────────────┘
```

## Requirements

- Linux (x86_64)
- Go 1.21+
- CMake 3.16+
- GCC/Clang with C++17 support

The required SDKs (Metamod:Source and HL2SDK) are included as git submodules.

## Building

### Prerequisites

```bash
# Ubuntu/Debian
sudo apt-get install build-essential cmake golang-go git

# Arch Linux
sudo pacman -S base-devel cmake go git
```

### Build

```bash
# Clone the repository with submodules
git clone --recursive https://github.com/corrreia/gostrike
cd gostrike

# Or if already cloned, initialize submodules
git submodule update --init --recursive

# Build Go shared library
make go

# Build native Metamod plugin (SDKs fetched automatically if needed)
make native

# Build everything
make go native
```

**Note:** The CS2 HL2SDK requires protobuf-generated headers. For development, the build uses stub headers. For production deployment, you may need the full SDK with protobuf - see [CounterStrikeSharp](https://github.com/roflmuffin/CounterStrikeSharp) for reference.

### Output

After building:

- `build/libgostrike_go.so` - Go runtime library
- `build/gostrike.so` - Metamod plugin

## Installation

1. Install Metamod:Source on your CS2 server
2. Copy `gostrike.so` to `csgo/addons/metamod/`
3. Create `csgo/addons/gostrike/bin/` and copy `libgostrike_go.so` there
4. Add GoStrike to your `metaplugins.ini`:
   ```
   gostrike
   ```
5. Restart your server

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
    // Register a command
    gostrike.RegisterCommand(gostrike.CommandInfo{
        Name:        "mycommand",
        Description: "My custom command",
        Callback: func(ctx *gostrike.CommandContext) error {
            ctx.Reply("Hello from Go!")
            return nil
        },
    })

    // Register an event handler
    gostrike.RegisterEventHandler(func(e *gostrike.PlayerConnectEvent) gostrike.EventResult {
        gostrike.GetLogger("MyPlugin").Info("Player connected: %s", e.Name)
        return gostrike.EventContinue
    }, gostrike.HookPost)

    return nil
}

func (p *MyPlugin) Unload(hotReload bool) error {
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

Then rebuild the Go library.

## Project Structure

```
gostrike/
├── native/              # C++ Metamod plugin
│   ├── src/            # Source files
│   └── include/        # Headers (including gostrike_abi.h)
├── pkg/                # Go SDK (public API)
│   ├── gostrike/       # Core types and functions
│   └── plugin/         # Plugin interface
├── internal/           # Internal implementation
│   ├── bridge/         # CGO exports and callbacks
│   ├── runtime/        # Event/command dispatch
│   └── manager/        # Plugin lifecycle
├── cmd/gostrike/       # c-shared entry point
├── plugins/            # Example plugins
├── configs/            # Configuration files
├── docker/             # Test environment
└── tests/              # Test suites
```

## License

MIT License - see LICENSE file for details.

## Acknowledgments

- [CounterStrikeSharp](https://github.com/roflmuffin/CounterStrikeSharp) - Architecture inspiration
- [Metamod:Source](https://www.metamodsource.net/) - Plugin loading framework
- [AlliedModders](https://alliedmods.net/) - Community resources
