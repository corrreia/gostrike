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
#endif

namespace gostrike {

// Cached gamedata offsets (resolved once at init)
static int s_offsetRespawn = -1;
static int s_offsetChangeTeam = -1;
static int s_offsetTeleport = -1;
static int s_offsetCommitSuicide = -1;
static int s_offsetSetModel = -1;

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

    // Resolve signature-based functions
    void* switchTeamAddr = g_gameConfig.ResolveSignature("CCSPlayerController_SwitchTeam");
    if (switchTeamAddr) {
        s_fnSwitchTeam = reinterpret_cast<SwitchTeamFn>(switchTeamAddr);
    }

    printf("[GoStrike] GameFunctions: initialized (respawn=%d, changeTeam=%d, teleport=%d, suicide=%d)\n",
           s_offsetRespawn, s_offsetChangeTeam, s_offsetTeleport, s_offsetCommitSuicide);
    printf("[GoStrike] GameFunctions: SwitchTeam=%p\n", (void*)s_fnSwitchTeam);
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

} // namespace gostrike
