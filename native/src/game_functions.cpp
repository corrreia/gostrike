// game_functions.cpp - Common game function wrappers
// Inspired by CounterStrikeSharp's entity function call patterns
// Uses gamedata offsets for virtual function calls

#include "game_functions.h"
#include "gostrike.h"
#include "gameconfig.h"
#include "player_manager.h"
#include "utils.h"

#include <cstdio>
#include <cstring>

#ifndef USE_STUB_SDK
#include <entity2/entityinstance.h>
#include <funchook.h>
#endif

#include "schema.h"
#include "entity_system.h"
#include "go_bridge.h"

namespace gostrike {

// Cached gamedata offsets (resolved once at init)
static int s_offsetRespawn = -1;
static int s_offsetChangeTeam = -1;
static int s_offsetTeleport = -1;
static int s_offsetCommitSuicide = -1;
static int s_offsetSetModel = -1;
static int s_offsetRemoveWeapons = -1;

// CTakeDamageInfo field offsets (loaded from gamedata, with CSSharp-derived defaults)
static int s_offsetDamageAttacker = 0x0C;
static int s_offsetDamage = 0x50;
static int s_offsetDamageType = 0x60;

// Cached signature-based function pointers
typedef void (*SwitchTeamFn)(void*, int);
static SwitchTeamFn s_fnSwitchTeam = nullptr;

void GameFunctions_Initialize() {
    // Cache gamedata offsets
    s_offsetRespawn = g_gameConfig.GetOffset("CCSPlayerController_Respawn");
    s_offsetChangeTeam = g_gameConfig.GetOffset("CCSPlayerController_ChangeTeam");
    s_offsetTeleport = g_gameConfig.GetOffset("CBaseEntity_Teleport");
    s_offsetCommitSuicide = g_gameConfig.GetOffset("CBasePlayerPawn_CommitSuicide");
    s_offsetSetModel = -1; // SetModel uses signature, not offset
    s_offsetRemoveWeapons = g_gameConfig.GetOffset("CCSPlayer_ItemServices_RemoveWeapons");

    // CTakeDamageInfo offsets (fallback to defaults if not in gamedata)
    int val;
    val = g_gameConfig.GetOffset("CTakeDamageInfo_attacker");
    if (val >= 0) s_offsetDamageAttacker = val;
    val = g_gameConfig.GetOffset("CTakeDamageInfo_damage");
    if (val >= 0) s_offsetDamage = val;
    val = g_gameConfig.GetOffset("CTakeDamageInfo_damageType");
    if (val >= 0) s_offsetDamageType = val;

    // Resolve signature-based functions
    void* switchTeamAddr = g_gameConfig.ResolveSignature("CCSPlayerController_SwitchTeam");
    if (switchTeamAddr) {
        s_fnSwitchTeam = reinterpret_cast<SwitchTeamFn>(switchTeamAddr);
    }

    printf("[GoStrike] GameFunctions: initialized (respawn=%d, changeTeam=%d, teleport=%d, suicide=%d, removeWeapons=%d)\n",
           s_offsetRespawn, s_offsetChangeTeam, s_offsetTeleport, s_offsetCommitSuicide, s_offsetRemoveWeapons);
    printf("[GoStrike] GameFunctions: SwitchTeam=%p\n", (void*)s_fnSwitchTeam);
    printf("[GoStrike] GameFunctions: CTakeDamageInfo offsets (attacker=0x%X, damage=0x%X, damageType=0x%X)\n",
           s_offsetDamageAttacker, s_offsetDamage, s_offsetDamageType);
}

void GameFunc_Respawn(int32_t slot) {
#ifndef USE_STUB_SDK
    if (s_offsetRespawn < 0) {
        printf("[GoStrike] GameFunc_Respawn: offset not available\n");
        return;
    }

    void* controller = PlayerManager_GetController(slot);
    if (!controller) {
        printf("[GoStrike] GameFunc_Respawn: no controller for slot %d\n", slot);
        return;
    }

    CallVirtual<void>(controller, s_offsetRespawn);
#else
    (void)slot;
#endif
}

void GameFunc_ChangeTeam(int32_t slot, int32_t team) {
#ifndef USE_STUB_SDK
    if (s_offsetChangeTeam < 0) {
        printf("[GoStrike] GameFunc_ChangeTeam: offset not available\n");
        return;
    }

    void* controller = PlayerManager_GetController(slot);
    if (!controller) return;

    CallVirtual<void>(controller, s_offsetChangeTeam, team);
#else
    (void)slot;
    (void)team;
#endif
}

void GameFunc_SwitchTeam(int32_t slot, int32_t team) {
#ifndef USE_STUB_SDK
    if (!s_fnSwitchTeam) {
        // Fall back to ChangeTeam
        GameFunc_ChangeTeam(slot, team);
        return;
    }

    void* controller = PlayerManager_GetController(slot);
    if (!controller) return;

    s_fnSwitchTeam(controller, team);
#else
    (void)slot;
    (void)team;
#endif
}

void GameFunc_Slay(int32_t slot) {
#ifndef USE_STUB_SDK
    if (s_offsetCommitSuicide < 0) {
        printf("[GoStrike] GameFunc_Slay: offset not available\n");
        return;
    }

    void* pawn = PlayerManager_GetPawn(slot);
    if (!pawn) return;

    // CBasePlayerPawn::CommitSuicide(bool bExplode, bool bForce)
    CallVirtual<void>(pawn, s_offsetCommitSuicide, false, true);
#else
    (void)slot;
#endif
}

void GameFunc_Teleport(int32_t slot, gs_vector3_t* pos, gs_vector3_t* angles, gs_vector3_t* velocity) {
#ifndef USE_STUB_SDK
    if (s_offsetTeleport < 0) {
        printf("[GoStrike] GameFunc_Teleport: offset not available\n");
        return;
    }

    void* pawn = PlayerManager_GetPawn(slot);
    if (!pawn) return;

    // CBaseEntity::Teleport(const Vector* newPosition, const QAngle* newAngles, const Vector* newVelocity)
    // Pass nullptr for parameters we don't want to change
    void* pPos = pos ? reinterpret_cast<void*>(pos) : nullptr;
    void* pAngles = angles ? reinterpret_cast<void*>(angles) : nullptr;
    void* pVelocity = velocity ? reinterpret_cast<void*>(velocity) : nullptr;

    CallVirtual<void>(pawn, s_offsetTeleport, pPos, pAngles, pVelocity);
#else
    (void)slot;
    (void)pos;
    (void)angles;
    (void)velocity;
#endif
}

void GameFunc_SetModel(void* entity, const char* model) {
#ifndef USE_STUB_SDK
    if (!entity || !model) return;

    // Resolve SetModel signature if needed
    static void* s_fnSetModel = nullptr;
    static bool s_resolved = false;
    if (!s_resolved) {
        s_fnSetModel = g_gameConfig.ResolveSignature("CBaseModelEntity_SetModel");
        s_resolved = true;
    }

    if (!s_fnSetModel) {
        printf("[GoStrike] GameFunc_SetModel: function not resolved\n");
        return;
    }

    typedef void (*SetModelFn)(void*, const char*);
    reinterpret_cast<SetModelFn>(s_fnSetModel)(entity, model);
#else
    (void)entity;
    (void)model;
#endif
}

// ============================================================
// Weapon Management
// ============================================================

// GiveNamedItem signature (resolved from gamedata)
// CSSharp pattern: call on CCSPlayer_ItemServices, virtual offset or direct signature
typedef void (*GiveNamedItemFn)(void* itemServices, const char* item, void* unk1, void* unk2, void* unk3, void* unk4);
static GiveNamedItemFn s_fnGiveNamedItem = nullptr;
static bool s_giveNamedItemResolved = false;

void GameFunc_GiveNamedItem(int32_t slot, const char* itemName) {
#ifndef USE_STUB_SDK
    if (!itemName) return;

    // Lazy-resolve on first call
    if (!s_giveNamedItemResolved) {
        void* addr = g_gameConfig.ResolveSignature("GiveNamedItem");
        if (addr) {
            s_fnGiveNamedItem = reinterpret_cast<GiveNamedItemFn>(addr);
        }
        s_giveNamedItemResolved = true;
        printf("[GoStrike] GiveNamedItem resolved: %p\n", (void*)s_fnGiveNamedItem);
    }

    if (!s_fnGiveNamedItem) {
        printf("[GoStrike] GameFunc_GiveNamedItem: function not resolved\n");
        return;
    }

    void* pawn = PlayerManager_GetPawn(slot);
    if (!pawn) {
        printf("[GoStrike] GameFunc_GiveNamedItem: no pawn for slot %d\n", slot);
        return;
    }

    // Get CCSPlayer_ItemServices from pawn via schema: CCSPlayerPawnBase.m_pItemServices
    auto itemServicesKey = schema::GetOffset("CCSPlayerPawnBase", "m_pItemServices");
    if (itemServicesKey.offset <= 0) {
        printf("[GoStrike] GameFunc_GiveNamedItem: m_pItemServices offset not found\n");
        return;
    }

    void* itemServices = *reinterpret_cast<void**>(reinterpret_cast<uintptr_t>(pawn) + itemServicesKey.offset);
    if (!itemServices) {
        printf("[GoStrike] GameFunc_GiveNamedItem: itemServices is null for slot %d\n", slot);
        return;
    }

    // Prepend "weapon_" if not already present
    std::string fullName(itemName);
    if (fullName.find("weapon_") != 0 && fullName.find("item_") != 0) {
        fullName = "weapon_" + fullName;
    }

    s_fnGiveNamedItem(itemServices, fullName.c_str(), nullptr, nullptr, nullptr, nullptr);
#else
    (void)slot;
    (void)itemName;
#endif
}

void GameFunc_DropWeapons(int32_t slot) {
#ifndef USE_STUB_SDK
    if (s_offsetRemoveWeapons < 0) {
        printf("[GoStrike] GameFunc_DropWeapons: CCSPlayer_ItemServices_RemoveWeapons offset not available\n");
        return;
    }

    void* pawn = PlayerManager_GetPawn(slot);
    if (!pawn) {
        printf("[GoStrike] GameFunc_DropWeapons: no pawn for slot %d\n", slot);
        return;
    }

    // Get CCSPlayer_ItemServices from pawn via schema (same as GiveNamedItem)
    auto itemServicesKey = schema::GetOffset("CCSPlayerPawnBase", "m_pItemServices");
    if (itemServicesKey.offset <= 0) {
        printf("[GoStrike] GameFunc_DropWeapons: m_pItemServices offset not found\n");
        return;
    }

    void* itemServices = *reinterpret_cast<void**>(reinterpret_cast<uintptr_t>(pawn) + itemServicesKey.offset);
    if (!itemServices) {
        printf("[GoStrike] GameFunc_DropWeapons: itemServices is null for slot %d\n", slot);
        return;
    }

    CallVirtual<void>(itemServices, s_offsetRemoveWeapons);
#else
    (void)slot;
#endif
}

// ============================================================
// Damage Hook (funchook on CBaseEntity_TakeDamageOld)
// Inspired by CSSharp's damage hook approach
// ============================================================

#ifndef USE_STUB_SDK
// TakeDamageOld signature:
// void CBaseEntity::TakeDamageOld(CTakeDamageInfo* info)
typedef void (*TakeDamageOldFn)(void* entity, void* damageInfo);
static TakeDamageOldFn s_pOriginalTakeDamageOld = nullptr;
static funchook_t* s_pDamageHook = nullptr;

static void DetourTakeDamageOld(void* entity, void* damageInfo) {
    if (!entity || !damageInfo) {
        s_pOriginalTakeDamageOld(entity, damageInfo);
        return;
    }

    // Extract victim entity index
    int victimIndex = EntitySystem_GetEntityIndex(entity);

    // Extract attacker from CTakeDamageInfo (offset from gamedata)
    int attackerIndex = -1;
    uint32_t attackerHandle = *reinterpret_cast<uint32_t*>(
        reinterpret_cast<uintptr_t>(damageInfo) + s_offsetDamageAttacker);
    if (attackerHandle != 0xFFFFFFFF) {
        // CHandle: extract entity index from lower 15 bits
        uint32_t attackerEntIndex = attackerHandle & 0x7FFF;
        attackerIndex = static_cast<int>(attackerEntIndex);
    }

    // Extract damage amount (offset from gamedata)
    float damage = *reinterpret_cast<float*>(
        reinterpret_cast<uintptr_t>(damageInfo) + s_offsetDamage);

    // Extract damage type (offset from gamedata)
    int32_t damageType = *reinterpret_cast<int32_t*>(
        reinterpret_cast<uintptr_t>(damageInfo) + s_offsetDamageType);

    // Dispatch to Go
    gs_event_result_t result = GoBridge_OnTakeDamage(victimIndex, attackerIndex, damage, damageType);

    if (result >= GS_EVENT_HANDLED) {
        // Plugin wants to block this damage - skip the original
        return;
    }

    // Call original
    s_pOriginalTakeDamageOld(entity, damageInfo);
}
#endif

void GameFunc_InitDamageHook() {
#ifndef USE_STUB_SDK
    void* takeDamageAddr = g_gameConfig.ResolveSignature("CBaseEntity_TakeDamageOld");
    if (!takeDamageAddr) {
        printf("[GoStrike] GameFunc_InitDamageHook: CBaseEntity_TakeDamageOld signature not found\n");
        return;
    }

    printf("[GoStrike] CBaseEntity_TakeDamageOld found at %p\n", takeDamageAddr);

    s_pOriginalTakeDamageOld = reinterpret_cast<TakeDamageOldFn>(takeDamageAddr);
    s_pDamageHook = funchook_create();
    if (!s_pDamageHook) {
        printf("[GoStrike] GameFunc_InitDamageHook: funchook_create() failed\n");
        return;
    }

    int rv = funchook_prepare(s_pDamageHook, (void**)&s_pOriginalTakeDamageOld, (void*)&DetourTakeDamageOld);
    if (rv != 0) {
        printf("[GoStrike] GameFunc_InitDamageHook: funchook_prepare() failed: %s\n",
               funchook_error_message(s_pDamageHook));
        funchook_destroy(s_pDamageHook);
        s_pDamageHook = nullptr;
        s_pOriginalTakeDamageOld = nullptr;
        return;
    }

    rv = funchook_install(s_pDamageHook, 0);
    if (rv != 0) {
        printf("[GoStrike] GameFunc_InitDamageHook: funchook_install() failed: %s\n",
               funchook_error_message(s_pDamageHook));
        funchook_destroy(s_pDamageHook);
        s_pDamageHook = nullptr;
        s_pOriginalTakeDamageOld = nullptr;
        return;
    }

    printf("[GoStrike] TakeDamageOld hook installed successfully\n");
#endif
}

void GameFunc_ShutdownDamageHook() {
#ifndef USE_STUB_SDK
    if (s_pDamageHook) {
        funchook_uninstall(s_pDamageHook, 0);
        funchook_destroy(s_pDamageHook);
        s_pDamageHook = nullptr;
        s_pOriginalTakeDamageOld = nullptr;
        printf("[GoStrike] TakeDamageOld hook removed\n");
    }
#endif
}

} // namespace gostrike
