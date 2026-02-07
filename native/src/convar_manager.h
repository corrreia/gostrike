// convar_manager.h - ConVar read/write via ICvar
// Inspired by CounterStrikeSharp's con_command_manager.h

#ifndef GOSTRIKE_CONVAR_MANAGER_H
#define GOSTRIKE_CONVAR_MANAGER_H

#include <cstdint>

namespace gostrike {

// Initialize the ConVar manager
void ConVarManager_Initialize();

// ConVar read/write operations
int32_t ConVar_GetInt(const char* name);
void    ConVar_SetInt(const char* name, int32_t value);
float   ConVar_GetFloat(const char* name);
void    ConVar_SetFloat(const char* name, float value);
int32_t ConVar_GetString(const char* name, char* buf, int32_t buf_size);
void    ConVar_SetString(const char* name, const char* value);
bool    ConVar_GetBool(const char* name);
void    ConVar_SetBool(const char* name, bool value);

} // namespace gostrike

#endif // GOSTRIKE_CONVAR_MANAGER_H
