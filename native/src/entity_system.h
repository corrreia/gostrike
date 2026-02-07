// entity_system.h - Entity lifecycle tracking and lookup
// Inspired by CounterStrikeSharp's entity_manager.h
// (https://github.com/roflmuffin/CounterStrikeSharp)

#ifndef GOSTRIKE_ENTITY_SYSTEM_H
#define GOSTRIKE_ENTITY_SYSTEM_H

#include <cstdint>

namespace gostrike {

// Initialize entity system. Must be called after CGameResourceService is available.
void EntitySystem_Initialize();

// Shutdown entity system.
void EntitySystem_Shutdown();

// Entity lookup functions
void* EntitySystem_GetEntityByIndex(uint32_t index);
uint32_t EntitySystem_GetEntityIndex(void* entity);
const char* EntitySystem_GetEntityClassname(void* entity);
bool EntitySystem_IsEntityValid(void* entity);

} // namespace gostrike

#endif // GOSTRIKE_ENTITY_SYSTEM_H
