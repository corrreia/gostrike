// Package shared provides shared types and function registrations
// to avoid import cycles between bridge, runtime, and manager packages.
package shared

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
)

// ============================================================
// Log Level System
// ============================================================

// LogLevel represents logging severity
type LogLevel int

const (
	LogLevelDebug LogLevel = iota
	LogLevelInfo
	LogLevelWarning
	LogLevelError
	LogLevelNone // Disables all logging
)

// String returns the string representation of the log level
func (l LogLevel) String() string {
	switch l {
	case LogLevelDebug:
		return "debug"
	case LogLevelInfo:
		return "info"
	case LogLevelWarning:
		return "warning"
	case LogLevelError:
		return "error"
	case LogLevelNone:
		return "none"
	default:
		return "unknown"
	}
}

// ParseLogLevel converts a string to LogLevel
func ParseLogLevel(s string) LogLevel {
	switch strings.ToLower(s) {
	case "debug":
		return LogLevelDebug
	case "info":
		return LogLevelInfo
	case "warn", "warning":
		return LogLevelWarning
	case "error":
		return LogLevelError
	case "none", "off":
		return LogLevelNone
	default:
		return LogLevelInfo
	}
}

var (
	currentLogLevel LogLevel = LogLevelInfo
	logMu           sync.RWMutex
	configLoaded    bool

	// LogCallback is called for all log messages (set by bridge to forward to C++)
	LogCallback func(level LogLevel, tag, message string)
)

// Config represents the main gostrike.json configuration
type Config struct {
	Version  string `json:"version"`
	LogLevel string `json:"log_level"`
}

// LoadConfig loads the gostrike.json configuration
// This should be called early, before other initialization
func LoadConfig(configPath string) error {
	logMu.Lock()
	defer logMu.Unlock()

	if configLoaded {
		return nil
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		// Config file not found is not an error - use defaults
		configLoaded = true
		return nil
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return fmt.Errorf("failed to parse config: %w", err)
	}

	currentLogLevel = ParseLogLevel(cfg.LogLevel)
	configLoaded = true
	return nil
}

// SetLogLevel sets the current log level
func SetLogLevel(level LogLevel) {
	logMu.Lock()
	defer logMu.Unlock()
	currentLogLevel = level
}

// GetLogLevel returns the current log level
func GetLogLevel() LogLevel {
	logMu.RLock()
	defer logMu.RUnlock()
	return currentLogLevel
}

// ShouldLog returns true if the given level should be logged
func ShouldLog(level LogLevel) bool {
	logMu.RLock()
	defer logMu.RUnlock()
	return level >= currentLogLevel
}

// SetLogCallback sets the callback for log messages
func SetLogCallback(cb func(level LogLevel, tag, message string)) {
	logMu.Lock()
	defer logMu.Unlock()
	LogCallback = cb
}

// Log writes a log message at the specified level
// Output format: [GoStrike:tag] LEVEL: message
func Log(level LogLevel, tag, format string, args ...interface{}) {
	if !ShouldLog(level) {
		return
	}

	message := fmt.Sprintf(format, args...)

	// Format tag with GoStrike prefix
	formattedTag := fmt.Sprintf("GoStrike:%s", tag)

	// If we have a callback (bridge to C++), use it
	logMu.RLock()
	cb := LogCallback
	logMu.RUnlock()

	if cb != nil {
		cb(level, formattedTag, message)
	} else {
		// Fallback to stdout
		levelStr := strings.ToUpper(level.String())
		fmt.Fprintf(os.Stdout, "[%s] %s: %s\n", formattedTag, levelStr, message)
	}
}

// Convenience logging functions
func LogDebug(tag, format string, args ...interface{}) {
	Log(LogLevelDebug, tag, format, args...)
}

func LogInfo(tag, format string, args ...interface{}) {
	Log(LogLevelInfo, tag, format, args...)
}

func LogWarning(tag, format string, args ...interface{}) {
	Log(LogLevelWarning, tag, format, args...)
}

func LogError(tag, format string, args ...interface{}) {
	Log(LogLevelError, tag, format, args...)
}

// DebugLog is a convenience function for internal debug logging
func DebugLog(format string, args ...interface{}) {
	Log(LogLevelDebug, "GoStrike", format, args...)
}

// ============================================================
// Player Info
// ============================================================

// PlayerInfo contains player information
type PlayerInfo struct {
	Slot    int
	UserID  int
	SteamID uint64
	Name    string
	IP      string
	Team    int
	IsAlive bool
	IsBot   bool
	Health  int
	Armor   int
	PosX    float64
	PosY    float64
	PosZ    float64
}

// InitFunc is the type for initialization functions
type InitFunc func()

// ShutdownFunc is the type for shutdown functions
type ShutdownFunc func()

// TickFunc is the type for tick dispatch functions
type TickFunc func(deltaTime float64)

// EventFunc is the type for event dispatch functions
type EventFunc func(eventName string, nativeEvent uintptr, isPost bool) int

// PlayerConnectFunc is the type for player connect dispatch functions
type PlayerConnectFunc func(player *PlayerInfo)

// PlayerDisconnectFunc is the type for player disconnect dispatch functions
type PlayerDisconnectFunc func(slot int, reason string)

// MapChangeFunc is the type for map change dispatch functions
type MapChangeFunc func(mapName string)

// Dispatch functions - set by runtime package
var (
	RuntimeInit              InitFunc
	RuntimeShutdown          ShutdownFunc
	DispatchTick             TickFunc
	DispatchEvent            EventFunc
	DispatchPlayerConnect    PlayerConnectFunc
	DispatchPlayerDisconnect PlayerDisconnectFunc
	DispatchMapChange        MapChangeFunc
)

// Manager functions - set by manager package
var (
	ManagerInit     InitFunc
	ManagerShutdown ShutdownFunc
)
