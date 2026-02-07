// game_functions.h - Common game function wrappers
// Inspired by CounterStrikeSharp's entity/function call patterns

#ifndef GOSTRIKE_GAME_FUNCTIONS_H
#define GOSTRIKE_GAME_FUNCTIONS_H

#include "gostrike_abi.h"
#include <cstdint>

namespace gostrike {

// Initialize game function pointers from gamedata
void GameFunctions_Initialize();

// Player actions
void GameFunc_Respawn(int32_t slot);
void GameFunc_ChangeTeam(int32_t slot, int32_t team);
void GameFunc_SwitchTeam(int32_t slot, int32_t team);
void GameFunc_Slay(int32_t slot);
void GameFunc_Teleport(int32_t slot, gs_vector3_t* pos, gs_vector3_t* angles, gs_vector3_t* velocity);

// Entity actions
void GameFunc_SetModel(void* entity, const char* model);

} // namespace gostrike

#endif // GOSTRIKE_GAME_FUNCTIONS_H
