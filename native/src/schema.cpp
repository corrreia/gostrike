// schema.cpp - Source 2 Schema System interface
// Inspired by CounterStrikeSharp's schema.cpp
// (https://github.com/roflmuffin/CounterStrikeSharp)

#include "schema.h"
#include "gostrike.h"

#include <cstdio>
#include <cstring>
#include <unordered_map>
#include <string>

#ifndef USE_STUB_SDK
#include <schemasystem/schemasystem.h>
#include <schemasystem/schematypes.h>
#include <entity2/entityinstance.h>
#include <entity2/entityidentity.h>
#endif

namespace gostrike {
namespace schema {

// ============================================================
// FNV-1a hash for fast cache lookups
// ============================================================

static uint32_t FnvHash(const char* str) {
    uint32_t hash = 0x811c9dc5;
    while (*str) {
        hash ^= static_cast<uint8_t>(*str++);
        hash *= 0x01000193;
    }
    return hash;
}

// Combined key hash for (className, fieldName) pair
static uint64_t CombinedHash(const char* className, const char* fieldName) {
    uint64_t h1 = FnvHash(className);
    uint64_t h2 = FnvHash(fieldName);
    return (h1 << 32) | h2;
}

// ============================================================
// Cache
// ============================================================

static std::unordered_map<uint64_t, SchemaKey> s_cache;

// ============================================================
// Schema Lookup
// ============================================================

#ifndef USE_STUB_SDK
static bool IsFieldNetworked(SchemaClassFieldData_t& field) {
    for (int i = 0; i < field.m_nStaticMetadataCount; i++) {
        if (field.m_pStaticMetadata && field.m_pStaticMetadata[i].m_pszName) {
            if (strcmp(field.m_pStaticMetadata[i].m_pszName, "MNetworkEnable") == 0) {
                return true;
            }
        }
    }
    return false;
}
#endif

void Initialize() {
    s_cache.clear();
    printf("[GoStrike] Schema system initialized (cache cleared)\n");
}

SchemaKey GetOffset(const char* className, const char* fieldName) {
    if (!className || !fieldName) return {0, false};

    uint64_t key = CombinedHash(className, fieldName);

    // Check cache
    auto it = s_cache.find(key);
    if (it != s_cache.end()) {
        return it->second;
    }

#ifndef USE_STUB_SDK
    if (!gs_pSchemaSystem) {
        printf("[GoStrike] Schema: CSchemaSystem not available\n");
        return {0, false};
    }

    // Find type scope for the server module
    CSchemaSystemTypeScope* pScope = gs_pSchemaSystem->FindTypeScopeForModule("server.dll");
    if (!pScope) {
        // Try alternative name
        pScope = gs_pSchemaSystem->FindTypeScopeForModule("libserver.so");
    }
    if (!pScope) {
        printf("[GoStrike] Schema: could not find type scope for server module\n");
        return {0, false};
    }

    // Find the class
    SchemaClassInfoData_t* pClassInfo = pScope->FindDeclaredClass(className).Get();
    if (!pClassInfo) {
        printf("[GoStrike] Schema: class '%s' not found\n", className);
        s_cache[key] = {0, false};
        return {0, false};
    }

    // Search fields
    int fieldCount = pClassInfo->m_nFieldCount;
    SchemaClassFieldData_t* fields = pClassInfo->m_pFields;

    for (int i = 0; i < fieldCount; i++) {
        if (fields[i].m_pszName && strcmp(fields[i].m_pszName, fieldName) == 0) {
            SchemaKey result = {
                fields[i].m_nSingleInheritanceOffset,
                IsFieldNetworked(fields[i])
            };
            s_cache[key] = result;
            return result;
        }
    }

    // Not found in this class, check base classes
    if (pClassInfo->m_pBaseClasses) {
        SchemaClassInfoData_t* baseClass = pClassInfo->m_pBaseClasses->m_pClass;
        if (baseClass && baseClass->m_pFields) {
            int baseFieldCount = baseClass->m_nFieldCount;
            SchemaClassFieldData_t* baseFields = baseClass->m_pFields;
            for (int i = 0; i < baseFieldCount; i++) {
                if (baseFields[i].m_pszName && strcmp(baseFields[i].m_pszName, fieldName) == 0) {
                    SchemaKey result = {
                        baseFields[i].m_nSingleInheritanceOffset,
                        IsFieldNetworked(baseFields[i])
                    };
                    s_cache[key] = result;
                    return result;
                }
            }
        }
    }

    printf("[GoStrike] Schema: field '%s::%s' not found\n", className, fieldName);
#endif

    s_cache[key] = {0, false};
    return {0, false};
}

void SetStateChanged(void* entity, const char* className, const char* fieldName, int32_t fieldOffset) {
    if (!entity) return;

#ifndef USE_STUB_SDK
    // Notify the engine that a networked property changed
    auto* pEntity = static_cast<CEntityInstance*>(entity);
    NetworkStateChangedData data(static_cast<uint32_t>(fieldOffset));
    pEntity->NetworkStateChanged(data);
#else
    (void)className;
    (void)fieldName;
    (void)fieldOffset;
#endif
}

} // namespace schema
} // namespace gostrike
