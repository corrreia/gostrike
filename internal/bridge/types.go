// Package bridge provides the CGO bridge between the C++ native plugin and Go runtime.
// This file contains type definitions shared between the bridge and other packages.
package bridge

// Log levels matching C++ gs_log_level_t
const (
	LogLevelDebug   = 0
	LogLevelInfo    = 1
	LogLevelWarning = 2
	LogLevelError   = 3
)

// Event results matching C++ gs_event_result_t
const (
	EventContinue = 0 // Allow event to proceed normally
	EventChanged  = 1 // Event data was modified
	EventHandled  = 2 // Stop processing, but allow event
	EventStop     = 3 // Cancel the event entirely
)

// Team IDs matching C++ gs_team_t
const (
	TeamUnassigned = 0
	TeamSpectator  = 1
	TeamT          = 2
	TeamCT         = 3
)
