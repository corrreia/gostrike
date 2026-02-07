// entity_system.cpp - Entity lifecycle tracking and lookup
// Inspired by CounterStrikeSharp's entity_manager.cpp
// (https://github.com/roflmuffin/CounterStrikeSharp)

#include "entity_system.h"
#include "gostrike.h"
#include "gameconfig.h"
#include "go_bridge.h"

#include <cstdio>

#ifndef USE_STUB_SDK
#include <entity2/entitysystem.h>
#include <entity2/entityidentity.h>

// Forward declare the entity listener
class GoStrikeEntityListener : public IEntityListener {
public:
    void OnEntitySpawned(CEntityInstance* pEntity) override;
    void OnEntityCreated(CEntityInstance* pEntity) override;
    void OnEntityDeleted(CEntityInstance* pEntity) override;
    void OnEntityParentChanged(CEntityInstance* pEntity, CEntityInstance* pNewParent) override;
};

static GoStrikeEntityListener s_entityListener;
static CGameEntitySystem* s_pEntitySystem = nullptr;
#endif

namespace gostrike {

void EntitySystem_Initialize() {
#ifndef USE_STUB_SDK
    if (!gs_pGameResourceService) {
        printf("[GoStrike] EntitySystem: CGameResourceService not available\n");
        return;
    }

    // Get the entity system using the gamedata offset
    // IGameResourceService is only forward-declared in the SDK, so we use offset
    int offset = g_gameConfig.GetOffset("GameEntitySystem");
    if (offset < 0) {
        // Fallback to known offset for Linux
        offset = 80;
        printf("[GoStrike] EntitySystem: using fallback offset %d for GameEntitySystem\n", offset);
    }

    s_pEntitySystem = *reinterpret_cast<CGameEntitySystem**>(
        reinterpret_cast<uintptr_t>(gs_pGameResourceService) + offset);

    if (!s_pEntitySystem) {
        printf("[GoStrike] EntitySystem: CGameEntitySystem not available at offset %d\n", offset);
        return;
    }

    // Register our entity listener
    s_pEntitySystem->AddListenerEntity(&s_entityListener);
    printf("[GoStrike] EntitySystem: initialized at %p, entity listener registered\n", s_pEntitySystem);
#else
    printf("[GoStrike] EntitySystem: stub mode, no entity tracking\n");
#endif
}

void EntitySystem_Shutdown() {
#ifndef USE_STUB_SDK
    if (s_pEntitySystem) {
        s_pEntitySystem->RemoveListenerEntity(&s_entityListener);
        s_pEntitySystem = nullptr;
        printf("[GoStrike] EntitySystem: listener removed\n");
    }
#endif
}

void* EntitySystem_GetEntityByIndex(uint32_t index) {
#ifndef USE_STUB_SDK
    if (!s_pEntitySystem) return nullptr;
    CEntityInstance* ent = s_pEntitySystem->GetEntityInstance(CEntityIndex(static_cast<int>(index)));
    return static_cast<void*>(ent);
#else
    (void)index;
    return nullptr;
#endif
}

uint32_t EntitySystem_GetEntityIndex(void* entity) {
#ifndef USE_STUB_SDK
    if (!entity) return UINT32_MAX;
    auto* pEntity = static_cast<CEntityInstance*>(entity);
    if (!pEntity->m_pEntity) return UINT32_MAX;
    return pEntity->m_pEntity->m_EHandle.GetEntryIndex();
#else
    (void)entity;
    return UINT32_MAX;
#endif
}

const char* EntitySystem_GetEntityClassname(void* entity) {
#ifndef USE_STUB_SDK
    if (!entity) return nullptr;
    auto* pEntity = static_cast<CEntityInstance*>(entity);
    return pEntity->GetClassname();
#else
    (void)entity;
    return nullptr;
#endif
}

bool EntitySystem_IsEntityValid(void* entity) {
#ifndef USE_STUB_SDK
    if (!entity) return false;
    auto* pEntity = static_cast<CEntityInstance*>(entity);
    return pEntity->m_pEntity != nullptr;
#else
    (void)entity;
    return false;
#endif
}

} // namespace gostrike

// ============================================================
// Entity Listener Callbacks
// ============================================================

#ifndef USE_STUB_SDK
void GoStrikeEntityListener::OnEntitySpawned(CEntityInstance* pEntity) {
    if (!pEntity || !pEntity->m_pEntity) return;
    uint32_t index = pEntity->m_pEntity->m_EHandle.GetEntryIndex();
    const char* classname = pEntity->GetClassname();
    GoBridge_OnEntitySpawned(index, classname ? classname : "");
}

void GoStrikeEntityListener::OnEntityCreated(CEntityInstance* pEntity) {
    if (!pEntity || !pEntity->m_pEntity) return;
    uint32_t index = pEntity->m_pEntity->m_EHandle.GetEntryIndex();
    const char* classname = pEntity->GetClassname();
    GoBridge_OnEntityCreated(index, classname ? classname : "");
}

void GoStrikeEntityListener::OnEntityDeleted(CEntityInstance* pEntity) {
    if (!pEntity || !pEntity->m_pEntity) return;
    uint32_t index = pEntity->m_pEntity->m_EHandle.GetEntryIndex();
    GoBridge_OnEntityDeleted(index);
}

void GoStrikeEntityListener::OnEntityParentChanged(CEntityInstance* /*pEntity*/,
                                                     CEntityInstance* /*pNewParent*/) {
    // Not forwarded to Go for now
}
#endif
