// memory_module.h - Module discovery and signature scanning
// Inspired by CounterStrikeSharp's memory_module.h
// (https://github.com/roflmuffin/CounterStrikeSharp)

#ifndef GOSTRIKE_MEMORY_MODULE_H
#define GOSTRIKE_MEMORY_MODULE_H

#include <cstdint>
#include <cstddef>
#include <string>
#include <vector>

namespace gostrike {

class Module {
public:
    Module() = default;

    // Initialize by finding a loaded module by name (e.g. "libserver.so")
    bool Initialize(const char* moduleName);

    // Scan for a byte signature with wildcards
    // Signature format: "55 48 89 E5 ?? 48 89" where ?? is wildcard
    void* FindSignature(const char* signature) const;

    // Find an exported symbol by name
    void* FindSymbol(const char* symbolName) const;

    bool IsInitialized() const { return m_base != nullptr; }
    const char* GetName() const { return m_name.c_str(); }
    const char* GetPath() const { return m_path.c_str(); }
    uint8_t* GetBase() const { return m_base; }
    size_t GetSize() const { return m_size; }

private:
    // Parse hex signature string into byte vector (-1 = wildcard)
    static std::vector<int16_t> ParseSignature(const char* sig);

    std::string m_name;
    std::string m_path;
    uint8_t* m_base = nullptr;
    size_t m_size = 0;
    void* m_dlHandle = nullptr;
};

// Pre-initialized well-known modules
namespace modules {
    extern Module server;
    extern Module engine;
    extern Module tier0;

    // Initialize all known modules. Call once at plugin startup.
    bool InitializeAll();
}

} // namespace gostrike

#endif // GOSTRIKE_MEMORY_MODULE_H
