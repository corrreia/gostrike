// gameconfig.cpp - GameData configuration system
// Inspired by CounterStrikeSharp's gameconfig.cpp
// (https://github.com/roflmuffin/CounterStrikeSharp)

#include "gameconfig.h"
#include "memory_module.h"

#include <cstdio>
#include <fstream>
#include <nlohmann/json.hpp>

using json = nlohmann::json;

namespace gostrike {

GameConfig g_gameConfig;

bool GameConfig::Init(const char* path, char* error, int errorSize) {
    if (!path) {
        if (error) snprintf(error, errorSize, "GameConfig: null path");
        return false;
    }

    m_path = path;
    printf("[GoStrike] Loading gamedata from: %s\n", path);

    std::ifstream file(path);
    if (!file.is_open()) {
        if (error) snprintf(error, errorSize, "GameConfig: could not open %s", path);
        printf("[GoStrike] ERROR: Could not open gamedata file: %s\n", path);
        return false;
    }

    json data;
    try {
        data = json::parse(file);
    } catch (const json::parse_error& e) {
        if (error) snprintf(error, errorSize, "GameConfig: JSON parse error: %s", e.what());
        printf("[GoStrike] ERROR: JSON parse error in %s: %s\n", path, e.what());
        return false;
    }

    int sigCount = 0, offsetCount = 0;

    for (auto& [key, value] : data.items()) {
        // Parse signatures
        if (value.contains("signatures")) {
            auto& sig = value["signatures"];
            if (sig.contains("library")) {
                m_libraries[key] = sig["library"].get<std::string>();
            }
            // Use linux signatures (we only target Linux)
            if (sig.contains("linux")) {
                std::string sigStr = sig["linux"].get<std::string>();
                m_signatures[key] = sigStr;
                sigCount++;
            }
        }

        // Parse offsets
        if (value.contains("offsets")) {
            auto& off = value["offsets"];
            if (off.contains("linux") && off["linux"].is_number_integer()) {
                m_offsets[key] = off["linux"].get<int>();
                offsetCount++;
            }
        }
    }

    printf("[GoStrike] GameData loaded: %d signatures, %d offsets\n", sigCount, offsetCount);
    return true;
}

const char* GameConfig::GetLibrary(const std::string& name) const {
    auto it = m_libraries.find(name);
    if (it == m_libraries.end()) return nullptr;
    return it->second.c_str();
}

const char* GameConfig::GetSignature(const std::string& name) const {
    auto it = m_signatures.find(name);
    if (it == m_signatures.end()) return nullptr;
    return it->second.c_str();
}

int GameConfig::GetOffset(const std::string& name) const {
    auto it = m_offsets.find(name);
    if (it == m_offsets.end()) return -1;
    return it->second;
}

bool GameConfig::IsSymbol(const char* sig) {
    return sig && sig[0] == '@';
}

Module* GameConfig::GetModule(const std::string& name) const {
    const char* lib = GetLibrary(name);
    if (!lib) return nullptr;

    if (strcmp(lib, "server") == 0) return &modules::server;
    if (strcmp(lib, "engine") == 0) return &modules::engine;
    if (strcmp(lib, "tier0") == 0) return &modules::tier0;

    return nullptr;
}

void* GameConfig::ResolveSignature(const std::string& name) {
    // Check cache first
    auto cached = m_addressCache.find(name);
    if (cached != m_addressCache.end()) {
        return cached->second;
    }

    // Get the module for this entry
    Module* module = GetModule(name);
    if (!module || !module->IsInitialized()) {
        printf("[GoStrike] GameData: module not found for '%s'\n", name.c_str());
        return nullptr;
    }

    // Get the signature/symbol string
    const char* sig = GetSignature(name);
    if (!sig) {
        printf("[GoStrike] GameData: no signature for '%s'\n", name.c_str());
        return nullptr;
    }

    void* addr = nullptr;

    if (IsSymbol(sig)) {
        // Symbol lookup: strip the @ prefix
        addr = module->FindSymbol(sig + 1);
        if (!addr) {
            printf("[GoStrike] GameData: symbol not found for '%s': %s\n", name.c_str(), sig + 1);
        }
    } else {
        // Signature scan
        addr = module->FindSignature(sig);
        if (!addr) {
            printf("[GoStrike] GameData: signature scan failed for '%s'\n", name.c_str());
        }
    }

    // Cache the result (even if nullptr, to avoid repeated scans)
    m_addressCache[name] = addr;

    if (addr) {
        printf("[GoStrike] GameData: resolved '%s' -> %p\n", name.c_str(), addr);
    }

    return addr;
}

} // namespace gostrike
