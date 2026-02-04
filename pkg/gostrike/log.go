// Package gostrike provides the public SDK for GoStrike plugins.
package gostrike

import (
	"github.com/corrreia/gostrike/internal/bridge"
)

// LogLevel represents logging severity
type LogLevel int

const (
	LogDebug LogLevel = iota
	LogInfo
	LogWarning
	LogError
)

// Logger provides structured logging for plugins
type Logger interface {
	Debug(format string, args ...interface{})
	Info(format string, args ...interface{})
	Warning(format string, args ...interface{})
	Error(format string, args ...interface{})

	WithField(key string, value interface{}) Logger
	WithFields(fields map[string]interface{}) Logger
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

func (l *logger) formatMessage(format string, args ...interface{}) string {
	return format
}

func (l *logger) Debug(format string, args ...interface{}) {
	bridge.LogDebug(l.tag, format, args...)
}

func (l *logger) Info(format string, args ...interface{}) {
	bridge.LogInfo(l.tag, format, args...)
}

func (l *logger) Warning(format string, args ...interface{}) {
	bridge.LogWarning(l.tag, format, args...)
}

func (l *logger) Error(format string, args ...interface{}) {
	bridge.LogError(l.tag, format, args...)
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

// Package-level logging functions

// Debug logs a debug message
func Debug(tag, format string, args ...interface{}) {
	bridge.LogDebug(tag, format, args...)
}

// Info logs an info message
func Info(tag, format string, args ...interface{}) {
	bridge.LogInfo(tag, format, args...)
}

// Warning logs a warning message
func Warning(tag, format string, args ...interface{}) {
	bridge.LogWarning(tag, format, args...)
}

// Error logs an error message
func Error(tag, format string, args ...interface{}) {
	bridge.LogError(tag, format, args...)
}
