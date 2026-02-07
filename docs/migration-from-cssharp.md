# Migrating from CounterStrikeSharp to GoStrike

This guide helps CSSharp plugin developers get started with GoStrike. The APIs are intentionally similar since GoStrike's architecture is inspired by CSSharp, but there are important differences because GoStrike uses Go instead of C#.

## Key Differences

| Feature | CounterStrikeSharp (C#) | GoStrike (Go) |
|---------|------------------------|---------------|
| Language | C# (.NET 8) | Go 1.21+ |
| Plugin Format | .dll loaded by .NET runtime | Compiled into single .so |
| Entity Access | `player.PlayerPawn.Value.Health` | `pawn.GetPropInt("CCSPlayerPawn", "m_iHealth")` |
| Events | Attribute-based `[GameEventHandler]` | `gostrike.RegisterGenericEventHandler(...)` |
| Chat Commands | `[ConsoleCommand("css_hello")]` | `gostrike.RegisterChatCommand(...)` |
| Plugin Lifecycle | `OnLoad()/OnUnload()` | `Load()/Unload()` |
| Config | `configs/plugins/<name>/` | `configs/plugins/<slug>.json` |
| Admin Flags | `@css/kick`, `@css/ban` | Letter flags `c`, `d` or named `kick`, `ban` |

## Plugin Structure

### CSSharp

```csharp
using CounterStrikeSharp.API.Core;

public class MyPlugin : BasePlugin
{
    public override string ModuleName => "My Plugin";
    public override string ModuleVersion => "1.0.0";

    public override void Load(bool hotReload)
    {
        AddCommand("css_hello", "Says hello", OnHelloCommand);
        RegisterEventHandler<EventPlayerDeath>(OnPlayerDeath);
    }
}
```

### GoStrike

```go
package myplugin

import (
    "github.com/corrreia/gostrike/pkg/gostrike"
    "github.com/corrreia/gostrike/pkg/plugin"
)

type MyPlugin struct {
    plugin.BasePlugin
}

func (p *MyPlugin) Slug() string        { return "my_plugin" }
func (p *MyPlugin) Name() string        { return "My Plugin" }
func (p *MyPlugin) Version() string     { return "1.0.0" }
func (p *MyPlugin) Author() string      { return "Author" }
func (p *MyPlugin) Description() string { return "My plugin" }

func (p *MyPlugin) Load(hotReload bool) error {
    gostrike.RegisterChatCommand(gostrike.ChatCommandInfo{
        Name:     "hello",
        Callback: func(ctx *gostrike.CommandContext) error {
            ctx.Reply("Hello, %s!", ctx.Player.Name)
            return nil
        },
    })

    gostrike.RegisterGenericEventHandler("player_death",
        func(name string, event gostrike.Event) gostrike.EventResult {
            return gostrike.EventContinue
        }, gostrike.HookPost)

    return nil
}

func init() {
    plugin.Register(&MyPlugin{})
}
```

## Entity Property Access

### CSSharp - Strongly Typed Properties

```csharp
var health = player.PlayerPawn.Value.Health;
player.PlayerPawn.Value.Health = 200;
player.PlayerPawn.Value.ArmorValue = 100;
var isScoped = player.PlayerPawn.Value.IsScoped;
```

### GoStrike - Schema-Based Access

There are two approaches in GoStrike:

**Option A: Direct schema access (flexible, works for any field)**
```go
pawn := player.GetPawn()
health, _ := pawn.GetPropInt("CCSPlayerPawnBase", "m_iHealth")
pawn.SetPropInt("CCSPlayerPawnBase", "m_iHealth", 200)
pawn.SetPropInt("CCSPlayerPawnBase", "m_ArmorValue", 100)
isScoped, _ := pawn.GetPropBool("CCSPlayerPawnBase", "m_bIsScoped")
```

**Option B: Generated typed wrappers (type-safe, IDE-friendly)**
```go
import "github.com/corrreia/gostrike/pkg/gostrike/entities"

pawn := entities.NewCCSPlayerPawnBase(player.GetPawn())
health := pawn.Health()
pawn.SetHealth(200)
pawn.SetArmorValue(100)
isScoped := pawn.IsScoped()
```

## Events

### CSSharp

```csharp
[GameEventHandler]
public HookResult OnPlayerDeath(EventPlayerDeath @event, GameEventInfo info)
{
    var attacker = @event.Attacker;
    var victim = @event.Userid;
    return HookResult.Continue;
}
```

### GoStrike

```go
gostrike.RegisterGenericEventHandler("player_death",
    func(name string, event gostrike.Event) gostrike.EventResult {
        // Access event data via the event interface
        return gostrike.EventContinue
    }, gostrike.HookPost)
```

## Chat Commands

### CSSharp

```csharp
[ConsoleCommand("css_kick")]
[RequiresPermissions("@css/kick")]
public void OnKickCommand(CCSPlayerController? player, CommandInfo info)
{
    // ...
}
```

### GoStrike

```go
gostrike.RegisterChatCommand(gostrike.ChatCommandInfo{
    Name:        "kick",
    Description: "Kick a player",
    Flags:       gostrike.ChatCmdAdmin,
    Callback: func(ctx *gostrike.CommandContext) error {
        if !ctx.HasFlag(gostrike.AdminKick) {
            ctx.ReplyError("No permission")
            return nil
        }
        // ...
        return nil
    },
})
```

## Timers

### CSSharp

```csharp
AddTimer(5.0f, () => {
    Server.PrintToChatAll("5 seconds passed!");
});

AddTimer(10.0f, () => {
    // Repeating timer
}, TimerFlags.REPEAT);
```

### GoStrike

```go
// One-shot timer
gostrike.After(5.0, func() {
    gostrike.GetServer().PrintToAll("5 seconds passed!")
})

// Repeating timer
timer := gostrike.Every(10.0, func() {
    // Runs every 10 seconds
})
timer.Stop() // Stop when done
```

## Player Operations

### CSSharp

```csharp
player.Respawn();
player.ChangeTeam(CsTeam.CounterTerrorist);
player.CommitSuicide(false, false);
player.GiveNamedItem("weapon_ak47");
```

### GoStrike

```go
player.Respawn()
player.ChangeTeam(gostrike.TeamCT)
player.Slay()
// GiveNamedItem requires gamedata resolution (Phase 2)
```

## ConVars

### CSSharp

```csharp
var roundTime = ConVar.Find("mp_roundtime");
float value = roundTime.GetPrimitiveValue<float>();
```

### GoStrike

```go
roundTime := gostrike.GetConVarFloat("mp_roundtime")
gostrike.SetConVarFloat("mp_roundtime", 3.0)
```

## Server Commands

### CSSharp

```csharp
Server.ExecuteCommand("changelevel de_dust2");
```

### GoStrike

```go
gostrike.GetServer().ExecuteCommand("changelevel de_dust2")
```

## Messaging

### CSSharp

```csharp
player.PrintToChat("Hello!");
player.PrintToCenter("Center text!");
Server.PrintToChatAll("Broadcast!");
```

### GoStrike

```go
player.PrintToChat("Hello!")
player.PrintToCenter("Center text!")
gostrike.GetServer().PrintToAll("Broadcast!")
```

## Target Patterns

Both CSSharp and GoStrike support target patterns for admin commands:

| Pattern | Description |
|---------|-------------|
| `@all` | All players |
| `@alive` | Alive players |
| `@dead` | Dead players |
| `@ct` | Counter-Terrorists |
| `@t` | Terrorists |
| `@me` | The caller |
| `@!me` | Everyone except caller |
| `@random` | Random alive player |
| `@bot` | All bots |
| `#<slot>` | Player by slot number |
| `<name>` | Partial name match |

```go
targets := gostrike.ResolveTarget(caller, "@alive")
for _, target := range targets {
    target.PrintToChat("You are alive!")
}
```

## Menus

### CSSharp

```csharp
var menu = new ChatMenu("Select Weapon");
menu.AddMenuOption("AK-47", (player, option) => {
    player.GiveNamedItem("weapon_ak47");
});
MenuManager.OpenChatMenu(player, menu);
```

### GoStrike

```go
menu := gostrike.NewMenu("Select Weapon")
menu.AddItem("AK-47", func(p *gostrike.Player) {
    // Give weapon
})
menu.AddItem("M4A4", func(p *gostrike.Player) {
    // Give weapon
})
menu.Display(player)
```

## Localization

### CSSharp

```csharp
Localizer["welcome.message", player.PlayerName]
```

### GoStrike

```go
localizer := gostrike.NewLocalizer("my_plugin")
localizer.LoadLangDir("configs/lang/")
msg := localizer.ForPlayer(player, "welcome.message", player.Name)
```

## Admin Permissions

GoStrike uses letter-based flags compatible with SourceMod:

| Letter | CSSharp Flag | GoStrike Constant | Permission |
|--------|-------------|-------------------|------------|
| a | `@css/reservation` | `FlagReservation` | Slot reservation |
| b | `@css/generic` | `FlagGeneric` | Generic admin |
| c | `@css/kick` | `FlagKick` | Kick players |
| d | `@css/ban` | `FlagBan` | Ban players |
| f | `@css/slay` | `FlagSlay` | Slay players |
| g | `@css/changemap` | `FlagChangelevel` | Change map |
| z | `@css/root` | `FlagRoot` | Full access |

Admin overrides can be configured in `configs/admin_overrides.json`:
```json
{
  "command_overrides": {
    "kick": "c",
    "ban": "d"
  }
}
```

## Building and Deploying

GoStrike plugins are compiled into the main binary (no hot-loading of individual .dll files):

```bash
# Add your plugin import to cmd/gostrike/main.go
import _ "github.com/corrreia/gostrike/plugins/myplugin"

# Build and deploy
make dev
```

Plugin enable/disable is controlled via `configs/plugins.json` without recompiling.

## Plugin Dependencies

GoStrike supports declaring plugin dependencies for load ordering:

```go
func (p *MyPlugin) Dependencies() []manager.PluginDependency {
    return []manager.PluginDependency{
        {Name: "Core Plugin", Optional: false},
        {Name: "Stats Plugin", Optional: true},
    }
}
```

Required dependencies are validated before loading and plugins are topologically sorted.
