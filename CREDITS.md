# Credits

## CounterStrikeSharp

GoStrike's architecture is heavily inspired by [CounterStrikeSharp](https://github.com/roflmuffin/CounterStrikeSharp) by [roflmuffin](https://github.com/roflmuffin).

Specifically, the following systems are derived from or inspired by CSSharp:

- **GameData format and signature database** - GoStrike uses the same `gamedata.json` format for cross-update compatibility
- **Schema system approach** - Entity property access via Source 2's CSchemaSystem interface
- **Memory module scanning patterns** - Module discovery and byte signature scanning
- **Admin permission flag conventions** - SourceMod-compatible flag system
- **Event hook architecture** - Pre/post hook modes with event result control
- **Build system patterns** - CMake setup with HL2SDK and Metamod:Source integration, protobuf generation
- **SourceHook usage patterns** - Interface acquisition and function hooking approach

The `gamedata.json` file shipped with GoStrike is derived from CounterStrikeSharp's gamedata with modifications for GoStrike's architecture.

CounterStrikeSharp is licensed under the [GNU General Public License v3.0](https://github.com/roflmuffin/CounterStrikeSharp/blob/main/LICENSE).

## HL2SDK

GoStrike uses the [HL2SDK](https://github.com/alliedmodders/hl2sdk) (CS2 branch) by AlliedModders for Source 2 engine integration.

## Metamod:Source

GoStrike uses [Metamod:Source](https://github.com/alliedmodders/metamod-source) by AlliedModders as its plugin framework.

## Protobuf Generation

The CMake protobuf generation approach is based on work by [Poggicek](https://github.com/Poggicek/StickerInspect), as used in CounterStrikeSharp.
