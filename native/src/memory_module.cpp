// memory_module.cpp - Module discovery and signature scanning
// Inspired by CounterStrikeSharp's memory_module.cpp
// (https://github.com/roflmuffin/CounterStrikeSharp)

#include "memory_module.h"

#include <cstring>
#include <cstdio>
#include <algorithm>
#include <dlfcn.h>
#include <link.h>
#include <elf.h>

namespace gostrike {

// ============================================================
// Module Discovery via dl_iterate_phdr
// ============================================================

struct ModuleSearchCtx {
    const char* targetName;
    uint8_t* base;
    size_t size;
    char path[512];
    bool found;
};

static int DlIterateCallback(struct dl_phdr_info* info, size_t /*size*/, void* data) {
    auto* ctx = static_cast<ModuleSearchCtx*>(data);

    if (!info->dlpi_name || info->dlpi_name[0] == '\0') {
        return 0; // Skip unnamed entries
    }

    // Check if this module matches our target name
    const char* modulePath = info->dlpi_name;
    const char* slash = strrchr(modulePath, '/');
    const char* baseName = slash ? slash + 1 : modulePath;

    if (strstr(baseName, ctx->targetName) == nullptr) {
        return 0; // Not the module we're looking for
    }

    // Calculate module base and size from program headers
    uintptr_t minAddr = UINTPTR_MAX;
    uintptr_t maxAddr = 0;

    for (int i = 0; i < info->dlpi_phnum; i++) {
        const auto& phdr = info->dlpi_phdr[i];
        if (phdr.p_type == PT_LOAD) {
            uintptr_t segStart = info->dlpi_addr + phdr.p_vaddr;
            uintptr_t segEnd = segStart + phdr.p_memsz;
            if (segStart < minAddr) minAddr = segStart;
            if (segEnd > maxAddr) maxAddr = segEnd;
        }
    }

    if (minAddr < maxAddr) {
        ctx->base = reinterpret_cast<uint8_t*>(minAddr);
        ctx->size = maxAddr - minAddr;
        strncpy(ctx->path, modulePath, sizeof(ctx->path) - 1);
        ctx->path[sizeof(ctx->path) - 1] = '\0';
        ctx->found = true;
        return 1; // Stop iteration
    }

    return 0;
}

bool Module::Initialize(const char* moduleName) {
    if (!moduleName) return false;

    ModuleSearchCtx ctx = {};
    ctx.targetName = moduleName;
    ctx.found = false;

    dl_iterate_phdr(DlIterateCallback, &ctx);

    if (!ctx.found) {
        printf("[GoStrike] Module not found: %s\n", moduleName);
        return false;
    }

    m_name = moduleName;
    m_path = ctx.path;
    m_base = ctx.base;
    m_size = ctx.size;

    // Open handle for dlsym lookups
    m_dlHandle = dlopen(ctx.path, RTLD_NOW | RTLD_NOLOAD);

    printf("[GoStrike] Module found: %s at %p (size: %zu, path: %s)\n",
           moduleName, m_base, m_size, m_path.c_str());
    return true;
}

// ============================================================
// Signature Parsing
// ============================================================

std::vector<int16_t> Module::ParseSignature(const char* sig) {
    std::vector<int16_t> bytes;
    const char* p = sig;

    while (*p) {
        // Skip whitespace
        while (*p == ' ') p++;
        if (*p == '\0') break;

        // Wildcard
        if (*p == '?' || (*p == '2' && *(p + 1) == 'A') || (*p == '\\' && *(p + 1) == 'x' && *(p + 2) == '2' && *(p + 3) == 'A')) {
            bytes.push_back(-1);
            if (*p == '?') {
                p++;
                if (*p == '?') p++; // Skip ?? format
            } else if (*p == '2') {
                p += 2; // Skip 2A
            } else {
                p += 4; // Skip \x2A
            }
        } else {
            // Hex byte
            char hi = *p++;
            char lo = (*p && *p != ' ') ? *p++ : '0';

            auto hexDigit = [](char c) -> int16_t {
                if (c >= '0' && c <= '9') return c - '0';
                if (c >= 'A' && c <= 'F') return c - 'A' + 10;
                if (c >= 'a' && c <= 'f') return c - 'a' + 10;
                return 0;
            };

            bytes.push_back((hexDigit(hi) << 4) | hexDigit(lo));
        }
    }

    return bytes;
}

// ============================================================
// Signature Scanning
// ============================================================

void* Module::FindSignature(const char* signature) const {
    if (!m_base || m_size == 0 || !signature) return nullptr;

    auto sigBytes = ParseSignature(signature);
    if (sigBytes.empty()) return nullptr;

    size_t sigLen = sigBytes.size();
    uint8_t* end = m_base + m_size - sigLen;

    for (uint8_t* current = m_base; current <= end; current++) {
        // Quick check: if first byte is not wildcard, skip non-matching
        if (sigBytes[0] != -1 && *current != static_cast<uint8_t>(sigBytes[0])) {
            continue;
        }

        // Full comparison
        bool match = true;
        for (size_t i = 0; i < sigLen; i++) {
            if (sigBytes[i] != -1 && current[i] != static_cast<uint8_t>(sigBytes[i])) {
                match = false;
                break;
            }
        }

        if (match) {
            return current;
        }
    }

    return nullptr;
}

// ============================================================
// Symbol Lookup
// ============================================================

void* Module::FindSymbol(const char* symbolName) const {
    if (!symbolName) return nullptr;

    // Try dlsym with the module handle first
    if (m_dlHandle) {
        void* addr = dlsym(m_dlHandle, symbolName);
        if (addr) return addr;
    }

    // Fallback: try global search
    void* addr = dlsym(RTLD_DEFAULT, symbolName);
    return addr;
}

// ============================================================
// Well-known modules
// ============================================================

namespace modules {
    Module server;
    Module engine;
    Module tier0;

    bool InitializeAll() {
        bool ok = true;

        if (!server.Initialize("libserver.so")) {
            printf("[GoStrike] WARNING: Could not find libserver.so\n");
            ok = false;
        }

        if (!engine.Initialize("libengine2.so")) {
            printf("[GoStrike] WARNING: Could not find libengine2.so\n");
            ok = false;
        }

        if (!tier0.Initialize("libtier0.so")) {
            printf("[GoStrike] WARNING: Could not find libtier0.so\n");
            ok = false;
        }

        return ok;
    }
}

} // namespace gostrike
