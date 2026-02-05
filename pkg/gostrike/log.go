// Package gostrike provides the public SDK for GoStrike plugins.
package gostrike

import (
	"fmt"

	"github.com/corrreia/gostrike/internal/shared"
)

// LogLevel represents logging severity
type LogLevel = shared.LogLevel

// Log level constants - re-exported for plugin use
const (
	LogDebug   = shared.LogLevelDebug
	LogInfo    = shared.LogLevelInfo
	LogWarning = shared.LogLevelWarning
	LogError   = shared.LogLevelError
)

// Logger provides structured logging for plugins
type Logger interface {
	// Basic logging methods
	Debug(format string, args ...interface{})
	Info(format string, args ...interface{})
	Warning(format string, args ...interface{})
	Error(format string, args ...interface{})

	// Log at a specific level
	Log(level LogLevel, format string, args ...interface{})

	// Check if a level would be logged (useful for expensive operations)
	IsDebugEnabled() bool
	IsLevelEnabled(level LogLevel) bool

	// Structured logging with fields
	WithField(key string, value interface{}) Logger
	WithFields(fields map[string]interface{}) Logger

	// Get the tag/name of this logger
	Tag() string
}

// logger implements Logger
type logger struct {
	tag    string
	fields map[string]interface{}
}

// GetLogger returns a logger for the given plugin name
func GetLogger(pluginName string) Logger {
	return &logger{
		tag:    pluginName,
		fields: make(map[string]interface{}),
	}
}

// Tag returns the logger's tag
func (l *logger) Tag() string {
	return l.tag
}

// formatWithFields formats a message with any attached fields
func (l *logger) formatWithFields(format string, args ...interface{}) string {
	msg := fmt.Sprintf(format, args...)
	if len(l.fields) == 0 {
		return msg
	}
	// Append fields to message
	fieldStr := ""
	for k, v := range l.fields {
		if fieldStr != "" {
			fieldStr += ", "
		}
		fieldStr += fmt.Sprintf("%s=%v", k, v)
	}
	return fmt.Sprintf("%s {%s}", msg, fieldStr)
}

// Log logs at the specified level
func (l *logger) Log(level LogLevel, format string, args ...interface{}) {
	shared.Log(level, l.tag, l.formatWithFields(format, args...))
}

func (l *logger) Debug(format string, args ...interface{}) {
	l.Log(LogDebug, format, args...)
}

func (l *logger) Info(format string, args ...interface{}) {
	l.Log(LogInfo, format, args...)
}

func (l *logger) Warning(format string, args ...interface{}) {
	l.Log(LogWarning, format, args...)
}

func (l *logger) Error(format string, args ...interface{}) {
	l.Log(LogError, format, args...)
}

// IsDebugEnabled returns true if debug logging is enabled
func (l *logger) IsDebugEnabled() bool {
	return shared.ShouldLog(LogDebug)
}

// IsLevelEnabled returns true if the given level would be logged
func (l *logger) IsLevelEnabled(level LogLevel) bool {
	return shared.ShouldLog(level)
}

func (l *logger) WithField(key string, value interface{}) Logger {
	newFields := make(map[string]interface{})
	for k, v := range l.fields {
		newFields[k] = v
	}
	newFields[key] = value
	return &logger{
		tag:    l.tag,
		fields: newFields,
	}
}

func (l *logger) WithFields(fields map[string]interface{}) Logger {
	newFields := make(map[string]interface{})
	for k, v := range l.fields {
		newFields[k] = v
	}
	for k, v := range fields {
		newFields[k] = v
	}
	return &logger{
		tag:    l.tag,
		fields: newFields,
	}
}

// ============================================================
// Package-level logging functions
// ============================================================

// Log logs a message at the specified level with a custom tag
func Log(level LogLevel, tag, format string, args ...interface{}) {
	shared.Log(level, tag, format, args...)
}

// Debug logs a debug message
func Debug(tag, format string, args ...interface{}) {
	shared.LogDebug(tag, format, args...)
}

// Info logs an info message
func Info(tag, format string, args ...interface{}) {
	shared.LogInfo(tag, format, args...)
}

// Warning logs a warning message
func Warning(tag, format string, args ...interface{}) {
	shared.LogWarning(tag, format, args...)
}

// Error logs an error message
func Error(tag, format string, args ...interface{}) {
	shared.LogError(tag, format, args...)
}

// ============================================================
// Log Level Utilities
// ============================================================

// GetLogLevel returns the current log level
func GetLogLevel() LogLevel {
	return shared.GetLogLevel()
}

// SetLogLevel sets the log level (primarily for testing)
func SetLogLevel(level LogLevel) {
	shared.SetLogLevel(level)
}

// IsDebugEnabled returns true if debug logging is enabled globally
func IsDebugEnabled() bool {
	return shared.ShouldLog(LogDebug)
}

// ParseLogLevel converts a string to LogLevel
func ParseLogLevel(s string) LogLevel {
	return shared.ParseLogLevel(s)
}
