# GoStrike Smoke Test Checklist

This document provides a checklist for verifying GoStrike functionality after building and deploying.

## Prerequisites

- [ ] Go 1.21+ installed
- [ ] CS2 dedicated server running
- [ ] Metamod:Source installed
- [ ] GoStrike built (`make go`)
- [ ] GoStrike deployed to server

## Build Verification

### Go Library Build

```bash
# Build the Go library
make go

# Verify output exists
ls -la build/libgostrike_go.so
ls -la build/libgostrike_go.h

# Check exported symbols
nm -D build/libgostrike_go.so | grep GoStrike
```

Expected symbols:

- [ ] `GoStrike_Init`
- [ ] `GoStrike_Shutdown`
- [ ] `GoStrike_OnTick`
- [ ] `GoStrike_OnEvent`
- [ ] `GoStrike_OnCommand`
- [ ] `GoStrike_GetABIVersion`
- [ ] `GoStrike_RegisterCallbacks`

## Server Startup

### 1. Server Starts Without Crash

- [ ] Server launches without crash
- [ ] Console shows no segmentation faults
- [ ] Console shows no missing symbol errors

### 2. Plugin Loads Successfully

Check for these console messages:

- [ ] `[GoStrike] Loading plugin...`
- [ ] `[GoStrike] Go runtime initialized successfully`
- [ ] `[GoStrike] Plugin loaded successfully`

### 3. Example Plugin Loads

- [ ] `[Example][INFO] Loading example plugin`
- [ ] `[Example][INFO] Example plugin loaded successfully!`
- [ ] `[Example][INFO] Registered 6 commands`
- [ ] `[Example][INFO] Registered event handlers`

## Command Tests

### gostrike_status

```
gostrike_status
```

Expected output:

- [ ] Version information displayed
- [ ] ABI version shown
- [ ] Runtime status: Running
- [ ] Map name displayed
- [ ] Player count displayed

### gostrike_test

```
gostrike_test hello world
```

Expected output:

- [ ] `GoStrike test command executed!`
- [ ] Arguments echoed back
- [ ] Player slot displayed

### gs_hello

```
gs_hello
```

Expected output:

- [ ] Hello message displayed

### gs_info

```
gs_info
```

Expected output:

- [ ] Server info displayed
- [ ] Map name
- [ ] Player count
- [ ] Tick rate

### gs_players

```
gs_players
```

Expected output:

- [ ] Player list displayed (or "No players" message)

### gs_timer

```
gs_timer 5
```

Expected output:

- [ ] "Timer set for 5.0 seconds..."
- [ ] After 5 seconds: "Timer finished!"

## Event Tests

### Player Connect Event

1. Add a bot or connect a player:

```
bot_add_t
```

Expected output:

- [ ] `[Example][INFO] Player connected: <name>`
- [ ] Player receives welcome message (if applicable)

### Player Disconnect Event

1. Kick a bot:

```
bot_kick
```

Expected output:

- [ ] `[Example][INFO] Player disconnected: slot X, reason: <reason>`

### Map Change Event

1. Change map:

```
changelevel de_mirage
```

Expected output:

- [ ] `[Example][INFO] Map changed to: de_mirage`

## Panic Recovery Test

### Test Panic Recovery

```
gs_panic
```

Expected output:

- [ ] "About to panic..." message
- [ ] Panic should be recovered
- [ ] Server does NOT crash
- [ ] Error logged with stack trace

## Timer System Test

### One-shot Timer

```
gs_timer 3
```

- [ ] Timer fires after 3 seconds
- [ ] Only fires once

### Verify Timer Count

After setting multiple timers, check that they all execute and clean up properly.

## Performance Checks

### Tick Rate

- [ ] Server maintains expected tick rate
- [ ] No significant FPS drops from GoStrike
- [ ] CPU usage reasonable

### Memory

- [ ] No memory leaks over time
- [ ] Memory usage stable

## Error Handling

### Invalid Commands

```
gs_slap
```

Expected:

- [ ] Error message about missing arguments
- [ ] Usage information displayed

### Permission Denied (if applicable)

When non-admin tries admin command:

- [ ] Permission denied message
- [ ] Command not executed

## Cleanup Test

### Server Shutdown

1. Stop the server gracefully
2. Check console output:

- [ ] `[GoStrike] Unloading plugin...`
- [ ] `[Example][INFO] Unloading example plugin`
- [ ] `[GoStrike] Plugin unloaded`
- [ ] No crash during shutdown

## Quick Test Script

```bash
#!/bin/bash
# Quick smoke test commands - run these in server console

# Status check
echo "Testing status..."
gostrike_status

# Test command
echo "Testing commands..."
gostrike_test "hello world"
gs_hello
gs_info
gs_players

# Timer test
echo "Testing timer (wait 5 seconds)..."
gs_timer 5

# Add and remove bot to test events
echo "Testing events..."
bot_add_t
sleep 2
bot_kick

# Panic recovery test
echo "Testing panic recovery..."
gs_panic

echo "Smoke test complete!"
```

## Troubleshooting

### Library Not Found

If you see "Failed to load Go library":

1. Check library path in go_bridge.cpp
2. Verify libgostrike_go.so exists in expected location
3. Check file permissions

### Symbol Not Found

If you see "Failed to load symbol":

1. Verify Go build completed successfully
2. Check `nm -D` output for exported symbols
3. Ensure //export comments are present in Go code

### Panic on Startup

If the server crashes on startup:

1. Check for nil pointer dereferences
2. Verify callbacks are registered before use
3. Check ABI version compatibility

### Commands Not Working

If commands don't respond:

1. Verify command registration in console
2. Check command handler is called
3. Verify reply_to_command callback is working
