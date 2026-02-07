// schema.h - Source 2 Schema System interface
// Resolves entity field offsets via CSchemaSystem at runtime.
// Inspired by CounterStrikeSharp's schema.h
// (https://github.com/roflmuffin/CounterStrikeSharp)

#ifndef GOSTRIKE_SCHEMA_H
#define GOSTRIKE_SCHEMA_H

#include <cstdint>

namespace gostrike {
namespace schema {

// Cached schema field info
struct SchemaKey {
    int32_t offset;
    bool networked;
};

// Initialize the schema system. Must be called after CSchemaSystem is available.
void Initialize();

// Get the offset and networked status for a class field.
// Returns {0, false} if not found.
// Results are cached for subsequent lookups.
SchemaKey GetOffset(const char* className, const char* fieldName);

// Notify the engine that a networked field has changed.
// entity: pointer to the entity (CBaseEntity*)
// className: schema class name
// fieldName: field name
// fieldOffset: byte offset of the field within the entity
void SetStateChanged(void* entity, const char* className, const char* fieldName, int32_t fieldOffset);

} // namespace schema
} // namespace gostrike

#endif // GOSTRIKE_SCHEMA_H
