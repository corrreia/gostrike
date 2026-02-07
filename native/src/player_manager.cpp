// player_manager.cpp - Player pawn/controller entity tracking
// Inspired by CounterStrikeSharp's player_manager.cpp

#include "player_manager.h"
#include "gostrike.h"
#include "schema.h"
#include "entity_system.h"

#include <cstdio>

#ifndef USE_STUB_SDK
#include <entity2/entitysystem.h>
#include <entity2/entityidentity.h>
#include <entity2/entityinstance.h>
#endif

namespace gostrike {

void* PlayerManager_GetController(int32_t slot) {
#ifndef USE_STUB_SDK
    // In CS2, player controllers use entity indices starting at 1
    // slot 0 = entity index 1, slot 1 = entity index 2, etc.
    uint32_t entityIndex = static_cast<uint32_t>(slot + 1);

    void* entity = EntitySystem_GetEntityByIndex(entityIndex);
    if (!entity) return nullptr;

    // Verify the entity is actually a player controller
    const char* classname = EntitySystem_GetEntityClassname(entity);
    if (!classname) return nullptr;

    // Accept both CS2 controller classnames
    if (strcmp(classname, "cs_player_controller") != 0 &&
        strcmp(classname, "player_controller") != 0) {
        return nullptr;
    }

    return entity;
#else
    (void)slot;
    return nullptr;
#endif
}

void* PlayerManager_GetPawn(int32_t slot) {
#ifndef USE_STUB_SDK
    void* controller = PlayerManager_GetController(slot);
    if (!controller) return nullptr;

    // Use schema to get the pawn handle from the controller
    // CCSPlayerController::m_hPlayerPawn is a CHandle<CCSPlayerPawn>
    // CHandle is a 32-bit value at the field offset
    schema::SchemaKey key = schema::GetOffset("CCSPlayerController", "m_hPlayerPawn");
    if (key.offset == 0) {
        // Try the base class field
        key = schema::GetOffset("CBasePlayerController", "m_hPawn");
    }
    if (key.offset == 0) return nullptr;

    // Read the handle value (CHandle is uint32_t internally)
    uint32_t handleValue = *reinterpret_cast<uint32_t*>(
        reinterpret_cast<uintptr_t>(controller) + key.offset);

    // Check for invalid handle (0xFFFFFFFF)
    if (handleValue == 0xFFFFFFFF) return nullptr;

    // Extract entity index from handle (lower bits)
    // CHandle stores: entry_index in lower 15 bits, serial in upper bits
    uint32_t entryIndex = handleValue & 0x7FFF;

    // Look up the pawn entity
    void* pawn = EntitySystem_GetEntityByIndex(entryIndex);
    return pawn;
#else
    (void)slot;
    return nullptr;
#endif
}

} // namespace gostrike
