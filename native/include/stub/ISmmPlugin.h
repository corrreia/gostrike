// Stub ISmmPlugin.h for compilation without full Metamod SDK
// This allows the plugin to compile for development/testing
// For actual deployment, use the real Metamod:Source SDK

#ifndef ISMMPLUG_H
#define ISMMPLUG_H

#include <cstddef>
#include <cstdint>
#include <cstdarg>

// Forward declarations
class ISmmAPI;
class ISmmPlugin;
class IMetamodListener;

typedef int PluginId;

// Minimal IMetamodListener interface (empty for stubs)
class IMetamodListener {
public:
    virtual ~IMetamodListener() = default;
};

// Minimal ISmmPlugin interface
class ISmmPlugin {
public:
    virtual ~ISmmPlugin() = default;
    virtual bool Load(PluginId id, ISmmAPI* ismm, char* error, size_t maxlen, bool late) = 0;
    virtual bool Unload(char* error, size_t maxlen) = 0;
    virtual void AllPluginsLoaded() {}
    virtual bool Pause(char* error, size_t maxlen) { return true; }
    virtual bool Unpause(char* error, size_t maxlen) { return true; }
    
    virtual const char* GetAuthor() = 0;
    virtual const char* GetName() = 0;
    virtual const char* GetDescription() = 0;
    virtual const char* GetURL() = 0;
    virtual const char* GetLicense() = 0;
    virtual const char* GetVersion() = 0;
    virtual const char* GetDate() = 0;
    virtual const char* GetLogTag() = 0;
};

// Minimal ISmmAPI interface
class ISmmAPI {
public:
    virtual ~ISmmAPI() = default;
    virtual void AddListener(ISmmPlugin* plugin, IMetamodListener* listener) {}
};

// Stub macros
#define PLUGIN_EXPOSE(name, var) \
    extern "C" void* CreateInterface(const char* pName, int* pReturnCode) { \
        if (pReturnCode) *pReturnCode = 0; \
        return &var; \
    }

#define PLUGIN_SAVEVARS()
#define PLUGIN_GLOBALVARS()

#endif // ISMMPLUG_H
