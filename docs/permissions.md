# GoStrike Permissions System

GoStrike uses a string-based permission system with dot notation, role-based access control, and wildcard matching. Permissions are stored in SQLite and managed via a REST API.

## Permission Format

Permissions use dot-separated strings:

```
gostrike.kick
gostrike.ban
example.give
example.hp
myplugin.admin.manage
```

Wildcards match any permission under a prefix:

| Permission | Matches |
|------------|---------|
| `*` | Everything (root) |
| `gostrike.*` | `gostrike.kick`, `gostrike.ban`, etc. |
| `example.*` | `example.give`, `example.hp`, etc. |

## Roles

Roles are named groups of permissions with an immunity level. Default roles seeded on first boot:

| Role | Immunity | Permissions |
|------|----------|-------------|
| `root` | 100 | `*` |
| `admin` | 80 | `gostrike.*` |
| `moderator` | 50 | `gostrike.kick`, `gostrike.ban`, `gostrike.slay`, `gostrike.chat` |
| `vip` | 10 | `gostrike.reservation` |

Create custom roles via the API.

## Players

Players are identified by SteamID64. Each player can have:
- **Roles** — inherits all permissions from assigned roles
- **Direct permissions** — granted individually
- **Immunity** — personal immunity level (combined with role immunity)
- **Expiration** — optional expiry timestamp (0 = never)

Effective permissions = direct permissions + all role permissions.
Effective immunity = max(player immunity, max(role immunities)).

## Immunity & Targeting

Players can only target other players with equal or lower immunity:

```go
player.CanTarget(otherPlayer) // true if player.immunity >= other.immunity
```

Root (`*`) always bypasses immunity checks.

## Storage

Permissions are stored in `data/permissions.db` (SQLite, WAL mode). The database is self-contained and does not depend on the optional database module.

Tables:
- `roles` — role definitions
- `role_permissions` — permissions assigned to roles
- `players` — player entries (by SteamID64)
- `player_roles` — role assignments
- `player_permissions` — direct permission grants

An in-memory cache is loaded from the database on startup for fast runtime permission checks.

## Plugin Integration

### Registering Permissions

Plugins should declare their permissions in `Load()`:

```go
func (p *MyPlugin) Load(hotReload bool) error {
    gostrike.RegisterPermission("myplugin.give", "Give weapons to self")
    gostrike.RegisterPermission("myplugin.admin", "Admin commands")
    // ...
}
```

Registered permissions appear in `GET /api/permissions/registered`.

### Chat Commands

Set the `Permission` field to require a permission:

```go
gostrike.RegisterChatCommand(gostrike.ChatCommandInfo{
    Name:       "give",
    Permission: "myplugin.give", // empty = public
    Callback:   myHandler,
})
```

### Checking Permissions in Code

```go
// On a player
player.HasPermission("myplugin.give")
player.IsAdmin()
player.CanTarget(otherPlayer)

// In a command context
ctx.HasPermission("myplugin.admin")
ctx.RequirePermission("myplugin.admin") // sends error if denied

// Standalone
gostrike.HasPermission(steamID, "myplugin.give")
gostrike.IsAdmin(steamID)
gostrike.CanTarget(sourceSteamID, targetSteamID)
```

## HTTP API

All endpoints are under `/api/permissions/`. No authentication required (by design).

### Roles

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/permissions/roles` | List all roles |
| `POST` | `/api/permissions/roles` | Create a role |
| `GET` | `/api/permissions/roles/{name}` | Get role by name |
| `PUT` | `/api/permissions/roles/{name}` | Update role |
| `DELETE` | `/api/permissions/roles/{name}` | Delete role |
| `POST` | `/api/permissions/role-permissions/{name}` | Add permission to role |
| `DELETE` | `/api/permissions/role-permissions/{name}` | Remove permission from role |

**Create role:**
```bash
curl -X POST http://localhost:8080/api/permissions/roles \
  -H "Content-Type: application/json" \
  -d '{"name": "helper", "display_name": "Helper", "immunity": 30}'
```

**Add permission to role:**
```bash
curl -X POST http://localhost:8080/api/permissions/role-permissions/helper \
  -H "Content-Type: application/json" \
  -d '{"permission": "gostrike.kick"}'
```

### Players

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/permissions/players` | List all players |
| `POST` | `/api/permissions/players` | Create/update player |
| `GET` | `/api/permissions/players/{steamid}` | Get player (includes effective perms) |
| `DELETE` | `/api/permissions/players/{steamid}` | Remove player |
| `POST` | `/api/permissions/player-roles/{steamid}` | Assign role to player |
| `DELETE` | `/api/permissions/player-roles/{steamid}` | Remove role from player |
| `POST` | `/api/permissions/player-permissions/{steamid}` | Grant direct permission |
| `DELETE` | `/api/permissions/player-permissions/{steamid}` | Revoke direct permission |

> **Note for JavaScript consumers:** SteamID64 values exceed `Number.MAX_SAFE_INTEGER` (2^53 - 1) and will lose precision if parsed as JavaScript numbers. Treat `steam_id` fields as strings in your client code (e.g., use `BigInt` or a JSON reviver). The API accepts both numeric and string-typed `steam_id` values.

**Add a player and assign admin role:**
```bash
# Create player entry
curl -X POST http://localhost:8080/api/permissions/players \
  -H "Content-Type: application/json" \
  -d '{"steam_id": "76561198012345678", "name": "MyAdmin", "immunity": 0}'

# Assign admin role
curl -X POST http://localhost:8080/api/permissions/player-roles/76561198012345678 \
  -H "Content-Type: application/json" \
  -d '{"role": "admin"}'
```

**Grant a direct permission:**
```bash
curl -X POST http://localhost:8080/api/permissions/player-permissions/76561198012345678 \
  -H "Content-Type: application/json" \
  -d '{"permission": "example.give"}'
```

### Utility

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/permissions/registered` | List all plugin-registered permissions |
| `POST` | `/api/permissions/check` | Check if a player has a permission |
| `POST` | `/api/permissions/reload` | Reload cache from database |

**Check a permission:**
```bash
curl -X POST http://localhost:8080/api/permissions/check \
  -H "Content-Type: application/json" \
  -d '{"steam_id": "76561198012345678", "permission": "gostrike.kick"}'
# → {"steam_id": 76561198012345678, "permission": "gostrike.kick", "has_permission": true}
```
