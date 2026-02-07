# Example Plugin

A reference GoStrike plugin demonstrating the full SDK feature set.

## Chat Commands

| Command | Description |
|---------|-------------|
| `!hello` | Greets the player and records it in the database |
| `!players` | Lists all connected players with team and alive status |
| `!info` | Shows server info (map, player count, tick rate) |
| `!health` | Reads player health via the schema system |
| `!entities` | Counts player controller entities and shows schema offsets |
| `!respawn` | Respawns the player using native game functions |
| `!roundtime` | Reads `mp_roundtime` and `mp_freezetime` ConVars |
| `!pawninfo` | Shows pawn data using generated typed entity accessors |

## HTTP API

All routes are automatically namespaced under `/api/plugins/example/`.

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/plugins/example/status` | GET | Plugin status and player count |
| `/api/plugins/example/players` | GET | List connected players as JSON |
| `/api/plugins/example/greet` | POST | Broadcast a message to all players |
| `/api/plugins/example/say` | POST | Send a chat message in-game |

Example:

```bash
curl http://localhost:8080/api/plugins/example/status
curl -X POST http://localhost:8080/api/plugins/example/say \
  -H "Content-Type: application/json" \
  -d '{"message": "Hello from the API!"}'
```

## Features Demonstrated

- **Plugin lifecycle** - `Load()` / `Unload()` with hot-reload support
- **Plugin slug** - Unique identifier for HTTP route and database namespacing
- **Plugin config** - Auto-generated at `configs/plugins/example.json` from `DefaultConfig()`
- **Chat commands** - Registration, arguments, reply helpers
- **Event handlers** - Player connect/disconnect, map change, round start/end
- **Entity lifecycle** - Entity spawned/deleted events
- **Schema access** - Read entity properties by class + field name
- **Typed entities** - Generated `CCSPlayerPawnBase`, `CBaseEntity` wrappers
- **ConVar access** - Read server ConVars
- **Game functions** - Player respawn via native game functions
- **Timers** - Repeating timer (60s debug log)
- **HTTP API** - REST endpoints with JSON request/response
- **Database** - Isolated per-plugin SQLite database
- **Logging** - Structured logger with slug-based tag

## Configuration

The plugin auto-generates `configs/plugins/example.json` on first load:

```json
{
    "welcome_message": "Welcome to the server!",
    "max_greetings": 100,
    "features": {
        "auto_greet": true,
        "log_connects": true,
        "track_players": true
    }
}
```

## Using as a Template

To create your own plugin based on this example:

1. Copy `plugins/example/` to `plugins/yourplugin/`
2. Change the package name, slug, and plugin metadata
3. Add your import to `cmd/gostrike/main.go`:
   ```go
   import _ "github.com/corrreia/gostrike/plugins/yourplugin"
   ```
4. Build and test: `make dev`
