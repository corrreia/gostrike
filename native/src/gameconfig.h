// gameconfig.h - GameData configuration system
// Loads function signatures, offsets, and patches from JSON files.
// Inspired by CounterStrikeSharp's gameconfig.h
// (https://github.com/roflmuffin/CounterStrikeSharp)

#ifndef GOSTRIKE_GAMECONFIG_H
#define GOSTRIKE_GAMECONFIG_H

#include <cstdint>
#include <string>
#include <unordered_map>

namespace gostrike {

class Module; // forward declare

class GameConfig {
public:
    GameConfig() = default;

    // Initialize from a JSON file path
    // Returns true on success, sets error message on failure.
    bool Init(const char* path, char* error = nullptr, int errorSize = 0);

    // Get the library name for a gamedata entry (e.g. "server", "engine")
    const char* GetLibrary(const std::string& name) const;

    // Get the raw signature string for a gamedata entry
    const char* GetSignature(const std::string& name) const;

    // Get the offset for a gamedata entry. Returns -1 if not found.
    int GetOffset(const std::string& name) const;

    // Resolve a gamedata entry to a memory address.
    // Uses the signature to scan the appropriate module.
    // Returns nullptr if not found.
    void* ResolveSignature(const std::string& name);

    // Get the module for a gamedata entry's library
    Module* GetModule(const std::string& name) const;

    // Check if a signature is actually a symbol (starts with @)
    static bool IsSymbol(const char* sig);

private:
    std::string m_path;
    std::unordered_map<std::string, std::string> m_signatures; // name -> signature
    std::unordered_map<std::string, std::string> m_libraries;  // name -> library
    std::unordered_map<std::string, int> m_offsets;            // name -> offset
    std::unordered_map<std::string, void*> m_addressCache;     // name -> resolved addr
};

// Global game config instance
extern GameConfig g_gameConfig;

} // namespace gostrike

#endif // GOSTRIKE_GAMECONFIG_H
