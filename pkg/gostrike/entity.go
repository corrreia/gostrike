// Package gostrike provides the public SDK for GoStrike plugins.
// This file provides the Entity type with Source 2 schema property access.
package gostrike

import (
	"fmt"

	"github.com/corrreia/gostrike/internal/bridge"
	"github.com/corrreia/gostrike/internal/runtime"
)

// Entity represents a Source 2 entity with schema property access.
// The underlying pointer is opaque and never dereferenced in Go.
type Entity struct {
	Index     uint32
	ClassName string
	ptr       uintptr // opaque C++ pointer, never dereferenced in Go
}

// Ptr returns the opaque entity pointer for internal use.
func (e *Entity) Ptr() uintptr {
	return e.ptr
}

// IsValid returns true if the entity is still valid in the game.
func (e *Entity) IsValid() bool {
	if e.ptr == 0 {
		return false
	}
	return bridge.IsEntityValid(e.ptr)
}

// Refresh updates the entity's cached fields from the engine.
func (e *Entity) Refresh() bool {
	if e.ptr == 0 {
		return false
	}
	if !bridge.IsEntityValid(e.ptr) {
		return false
	}
	cn := bridge.GetEntityClassname(e.ptr)
	if cn != "" {
		e.ClassName = cn
	}
	e.Index = bridge.GetEntityIndex(e.ptr)
	return true
}

// ============================================================
// Schema Property Access
// ============================================================

// GetPropInt reads an int32 property via schema.
func (e *Entity) GetPropInt(className, fieldName string) (int32, error) {
	if e.ptr == 0 {
		return 0, fmt.Errorf("entity pointer is nil")
	}
	return bridge.EntityGetInt(e.ptr, className, fieldName), nil
}

// SetPropInt writes an int32 property via schema.
// Automatically calls SetStateChanged for networked fields.
func (e *Entity) SetPropInt(className, fieldName string, value int32) error {
	if e.ptr == 0 {
		return fmt.Errorf("entity pointer is nil")
	}
	bridge.EntitySetInt(e.ptr, className, fieldName, value)
	return nil
}

// GetPropFloat reads a float32 property via schema.
func (e *Entity) GetPropFloat(className, fieldName string) (float32, error) {
	if e.ptr == 0 {
		return 0, fmt.Errorf("entity pointer is nil")
	}
	return bridge.EntityGetFloat(e.ptr, className, fieldName), nil
}

// SetPropFloat writes a float32 property via schema.
func (e *Entity) SetPropFloat(className, fieldName string, value float32) error {
	if e.ptr == 0 {
		return fmt.Errorf("entity pointer is nil")
	}
	bridge.EntitySetFloat(e.ptr, className, fieldName, value)
	return nil
}

// GetPropBool reads a bool property via schema.
func (e *Entity) GetPropBool(className, fieldName string) (bool, error) {
	if e.ptr == 0 {
		return false, fmt.Errorf("entity pointer is nil")
	}
	return bridge.EntityGetBool(e.ptr, className, fieldName), nil
}

// SetPropBool writes a bool property via schema.
func (e *Entity) SetPropBool(className, fieldName string, value bool) error {
	if e.ptr == 0 {
		return fmt.Errorf("entity pointer is nil")
	}
	bridge.EntitySetBool(e.ptr, className, fieldName, value)
	return nil
}

// GetPropString reads a string property via schema.
func (e *Entity) GetPropString(className, fieldName string) (string, error) {
	if e.ptr == 0 {
		return "", fmt.Errorf("entity pointer is nil")
	}
	return bridge.EntityGetString(e.ptr, className, fieldName), nil
}

// GetPropVector reads a Vector3 property via schema.
func (e *Entity) GetPropVector(className, fieldName string) (Vector3, error) {
	if e.ptr == 0 {
		return Vector3{}, fmt.Errorf("entity pointer is nil")
	}
	x, y, z := bridge.EntityGetVector(e.ptr, className, fieldName)
	return Vector3{X: float64(x), Y: float64(y), Z: float64(z)}, nil
}

// SetPropVector writes a Vector3 property via schema.
func (e *Entity) SetPropVector(className, fieldName string, v Vector3) error {
	if e.ptr == 0 {
		return fmt.Errorf("entity pointer is nil")
	}
	bridge.EntitySetVector(e.ptr, className, fieldName, float32(v.X), float32(v.Y), float32(v.Z))
	return nil
}

// ============================================================
// Entity Lookup
// ============================================================

// GetEntityByIndex returns an entity by its entity index.
// Returns nil if the entity doesn't exist.
func GetEntityByIndex(index uint32) *Entity {
	ptr := bridge.GetEntityByIndex(index)
	if ptr == 0 {
		return nil
	}

	classname := bridge.GetEntityClassname(ptr)
	return &Entity{
		Index:     index,
		ClassName: classname,
		ptr:       ptr,
	}
}

// FindEntitiesByClassName iterates all entity indices and returns
// entities matching the given classname.
func FindEntitiesByClassName(className string) []*Entity {
	var entities []*Entity
	// CS2 max entities is typically 16384
	for i := uint32(0); i < 16384; i++ {
		ptr := bridge.GetEntityByIndex(i)
		if ptr == 0 {
			continue
		}
		cn := bridge.GetEntityClassname(ptr)
		if cn == className {
			entities = append(entities, &Entity{
				Index:     i,
				ClassName: cn,
				ptr:       ptr,
			})
		}
	}
	return entities
}

// ============================================================
// Entity Lifecycle Events
// ============================================================

// EntityCreatedHandler is called when an entity is created
type EntityCreatedHandler func(entity *Entity)

// EntitySpawnedHandler is called when an entity is spawned
type EntitySpawnedHandler func(entity *Entity)

// EntityDeletedHandler is called when an entity is deleted
type EntityDeletedHandler func(index uint32)

// RegisterEntityCreatedHandler registers a handler for entity creation events
func RegisterEntityCreatedHandler(handler EntityCreatedHandler) {
	runtime.RegisterEntityCreatedHandler(func(index uint32, classname string) {
		ptr := bridge.GetEntityByIndex(index)
		entity := &Entity{
			Index:     index,
			ClassName: classname,
			ptr:       ptr,
		}
		handler(entity)
	})
}

// RegisterEntitySpawnedHandler registers a handler for entity spawn events
func RegisterEntitySpawnedHandler(handler EntitySpawnedHandler) {
	runtime.RegisterEntitySpawnedHandler(func(index uint32, classname string) {
		ptr := bridge.GetEntityByIndex(index)
		entity := &Entity{
			Index:     index,
			ClassName: classname,
			ptr:       ptr,
		}
		handler(entity)
	})
}

// RegisterEntityDeletedHandler registers a handler for entity deletion events
func RegisterEntityDeletedHandler(handler EntityDeletedHandler) {
	runtime.RegisterEntityDeletedHandler(func(index uint32) {
		handler(index)
	})
}

// ============================================================
// Schema Utility
// ============================================================

// GetSchemaOffset returns the byte offset of a schema field.
// This is useful for plugins that want to do manual memory manipulation.
func GetSchemaOffset(className, fieldName string) (offset int32, networked bool) {
	return bridge.SchemaGetOffset(className, fieldName)
}

// ============================================================
// GameData Utility
// ============================================================

// ResolveGamedata resolves a gamedata entry to a memory address.
// Returns 0 if not found.
func ResolveGamedata(name string) uintptr {
	return bridge.ResolveGamedata(name)
}

// GetGamedataOffset returns a gamedata offset by name.
// Returns -1 if not found.
func GetGamedataOffset(name string) int32 {
	return bridge.GetGamedataOffset(name)
}
