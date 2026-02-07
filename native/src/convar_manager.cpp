// convar_manager.cpp - ConVar read/write via ICvar
// Inspired by CounterStrikeSharp's con_command_manager.cpp

#include "convar_manager.h"
#include "gostrike.h"

#include <cstdio>
#include <cstring>

#ifndef USE_STUB_SDK
#include <tier1/convar.h>
#include <icvar.h>
#endif

namespace gostrike {

void ConVarManager_Initialize() {
#ifndef USE_STUB_SDK
    if (!gs_pCVar) {
        printf("[GoStrike] ConVarManager: ICvar not available\n");
        return;
    }
    printf("[GoStrike] ConVarManager: initialized\n");
#else
    printf("[GoStrike] ConVarManager: stub mode\n");
#endif
}

int32_t ConVar_GetInt(const char* name) {
#ifndef USE_STUB_SDK
    if (!gs_pCVar || !name) return 0;

    ConVarRef ref = gs_pCVar->FindConVar(name);
    if (!ref.IsValidRef()) return 0;

    ConVarRefAbstract aref(ref);
    return aref.GetInt();
#else
    (void)name;
    return 0;
#endif
}

void ConVar_SetInt(const char* name, int32_t value) {
#ifndef USE_STUB_SDK
    if (!gs_pCVar || !name) return;

    ConVarRef ref = gs_pCVar->FindConVar(name);
    if (!ref.IsValidRef()) return;

    ConVarRefAbstract aref(ref);
    aref.SetInt(value);
#else
    (void)name;
    (void)value;
#endif
}

float ConVar_GetFloat(const char* name) {
#ifndef USE_STUB_SDK
    if (!gs_pCVar || !name) return 0.0f;

    ConVarRef ref = gs_pCVar->FindConVar(name);
    if (!ref.IsValidRef()) return 0.0f;

    ConVarRefAbstract aref(ref);
    return aref.GetFloat();
#else
    (void)name;
    return 0.0f;
#endif
}

void ConVar_SetFloat(const char* name, float value) {
#ifndef USE_STUB_SDK
    if (!gs_pCVar || !name) return;

    ConVarRef ref = gs_pCVar->FindConVar(name);
    if (!ref.IsValidRef()) return;

    ConVarRefAbstract aref(ref);
    aref.SetFloat(value);
#else
    (void)name;
    (void)value;
#endif
}

int32_t ConVar_GetString(const char* name, char* buf, int32_t buf_size) {
#ifndef USE_STUB_SDK
    if (!gs_pCVar || !name || !buf || buf_size <= 0) return 0;

    ConVarRef ref = gs_pCVar->FindConVar(name);
    if (!ref.IsValidRef()) return 0;

    ConVarRefAbstract aref(ref);
    CBufferString bufStr;
    aref.GetValueAsString(bufStr);

    const char* str = bufStr.Get();
    if (!str) return 0;

    int len = static_cast<int>(strlen(str));
    if (len >= buf_size) len = buf_size - 1;
    memcpy(buf, str, len);
    buf[len] = '\0';
    return len;
#else
    (void)name;
    (void)buf;
    (void)buf_size;
    return 0;
#endif
}

void ConVar_SetString(const char* name, const char* value) {
#ifndef USE_STUB_SDK
    if (!gs_pCVar || !name || !value) return;

    ConVarRef ref = gs_pCVar->FindConVar(name);
    if (!ref.IsValidRef()) return;

    ConVarRefAbstract aref(ref);
    aref.SetString(CUtlString(value));
#else
    (void)name;
    (void)value;
#endif
}

bool ConVar_GetBool(const char* name) {
#ifndef USE_STUB_SDK
    if (!gs_pCVar || !name) return false;

    ConVarRef ref = gs_pCVar->FindConVar(name);
    if (!ref.IsValidRef()) return false;

    ConVarRefAbstract aref(ref);
    return aref.GetBool();
#else
    (void)name;
    return false;
#endif
}

void ConVar_SetBool(const char* name, bool value) {
#ifndef USE_STUB_SDK
    if (!gs_pCVar || !name) return;

    ConVarRef ref = gs_pCVar->FindConVar(name);
    if (!ref.IsValidRef()) return;

    ConVarRefAbstract aref(ref);
    aref.SetBool(value);
#else
    (void)name;
    (void)value;
#endif
}

} // namespace gostrike
