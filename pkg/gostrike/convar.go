// Package gostrike provides the public SDK for GoStrike plugins.
// This file provides ConVar (console variable) access.
package gostrike

import (
	"github.com/corrreia/gostrike/internal/bridge"
)

// GetConVarInt reads an integer ConVar value
func GetConVarInt(name string) int {
	return int(bridge.ConVarGetInt(name))
}

// SetConVarInt writes an integer ConVar value
func SetConVarInt(name string, value int) {
	bridge.ConVarSetInt(name, int32(value))
}

// GetConVarFloat reads a float ConVar value
func GetConVarFloat(name string) float64 {
	return float64(bridge.ConVarGetFloat(name))
}

// SetConVarFloat writes a float ConVar value
func SetConVarFloat(name string, value float64) {
	bridge.ConVarSetFloat(name, float32(value))
}

// GetConVarString reads a string ConVar value
func GetConVarString(name string) string {
	return bridge.ConVarGetString(name)
}

// SetConVarString writes a string ConVar value
func SetConVarString(name string, value string) {
	bridge.ConVarSetString(name, value)
}

// GetConVarBool reads a boolean ConVar value
func GetConVarBool(name string) bool {
	return bridge.ConVarGetInt(name) != 0
}

// SetConVarBool writes a boolean ConVar value
func SetConVarBool(name string, value bool) {
	v := int32(0)
	if value {
		v = 1
	}
	bridge.ConVarSetInt(name, v)
}
