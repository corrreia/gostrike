// player_manager.h - Player pawn/controller entity tracking
// Inspired by CounterStrikeSharp's player_manager.h

#ifndef GOSTRIKE_PLAYER_MANAGER_H
#define GOSTRIKE_PLAYER_MANAGER_H

#include <cstdint>

namespace gostrike {

// Get the CCSPlayerController entity for a player slot
// Returns nullptr if not found
void* PlayerManager_GetController(int32_t slot);

// Get the CCSPlayerPawn entity for a player slot
// Follows controller -> m_hPlayerPawn handle
// Returns nullptr if not found or player has no pawn (dead/spectating)
void* PlayerManager_GetPawn(int32_t slot);

} // namespace gostrike

#endif // GOSTRIKE_PLAYER_MANAGER_H
