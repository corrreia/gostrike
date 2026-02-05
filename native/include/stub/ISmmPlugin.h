// Stub ISmmPlugin.h for compilation without full Metamod SDK
// This implements the minimum required Metamod:Source plugin interface
// Based on the official Metamod:Source ISmmPlugin.h

#ifndef ISMMPLUG_H
#define ISMMPLUG_H

#include <cstddef>
#include <cstdint>
#include <cstdarg>
#include <cstring>

// Metamod Plugin API version - must match what Metamod expects
// Current version is 17, interface name is "ISmmPlugin"
#define METAMOD_PLAPI_VERSION 17
#define METAMOD_PLAPI_NAME "ISmmPlugin"

// Interface return status (matches HL2SDK)
enum {
    META_IFACE_OK = 0,
    META_IFACE_FAILED = 1
};

// Symbol visibility macro for proper export
#if defined(__GNUC__)
    #define SMM_API extern "C" __attribute__((visibility("default")))
#else
    #define SMM_API extern "C" __declspec(dllexport)
#endif

// Forward declarations
class ISmmAPI;
class ISmmPlugin;
class IMetamodListener;

typedef int PluginId;

// Minimal IMetamodListener interface
class IMetamodListener {
public:
    virtual ~IMetamodListener() = default;
    
    virtual void OnPluginLoad(PluginId id) {}
    virtual void OnPluginUnload(PluginId id) {}
    virtual void OnPluginPause(PluginId id) {}
    virtual void OnPluginUnpause(PluginId id) {}
    virtual void OnLevelInit(char const* pMapName, char const* pMapEntities,
                            char const* pOldLevel, char const* pLandmarkName,
                            bool loadGame, bool background) {}
    virtual void OnLevelShutdown() {}
    virtual void* OnEngineQuery(const char* iface, int* ret) {
        if (ret) *ret = META_IFACE_FAILED;
        return nullptr;
    }
    virtual void* OnPhysicsQuery(const char* iface, int* ret) {
        if (ret) *ret = META_IFACE_FAILED;
        return nullptr;
    }
    virtual void* OnFileSystemQuery(const char* iface, int* ret) {
        if (ret) *ret = META_IFACE_FAILED;
        return nullptr;
    }
    virtual void* OnGameDLLQuery(const char* iface, int* ret) {
        if (ret) *ret = META_IFACE_FAILED;
        return nullptr;
    }
    virtual void* OnMetamodQuery(const char* iface, int* ret) {
        if (ret) *ret = META_IFACE_FAILED;
        return nullptr;
    }
};

// ISmmPlugin interface - the main plugin interface
// CRITICAL: GetApiVersion() MUST be the first virtual method (after destructor)
// as Metamod checks the vtable ordering
class ISmmPlugin {
public:
    /**
     * @brief Returns the plugin API version. This is the FIRST method called by Metamod.
     * @return Plugin API version (must return METAMOD_PLAPI_VERSION)
     */
    virtual int GetApiVersion() { return METAMOD_PLAPI_VERSION; }
    
    /**
     * @brief Virtual destructor
     */
    virtual ~ISmmPlugin() = default;
    
    /**
     * @brief Called when the plugin is loaded.
     * @param id Plugin ID assigned by Metamod
     * @param ismm Pointer to Metamod API
     * @param error Buffer for error message
     * @param maxlen Size of error buffer
     * @param late True if loaded after server start
     * @return True on success, false to reject load
     */
    virtual bool Load(PluginId id, ISmmAPI* ismm, char* error, size_t maxlen, bool late) = 0;
    
    /**
     * @brief Called when the plugin is unloaded.
     */
    virtual bool Unload(char* error, size_t maxlen) = 0;
    
    /**
     * @brief Called after all plugins are loaded.
     */
    virtual void AllPluginsLoaded() {}
    
    /**
     * @brief Called to query if plugin is still running properly.
     */
    virtual bool QueryRunning(char* error, size_t maxlen) { return true; }
    
    /**
     * @brief Called when plugin is paused.
     */
    virtual bool Pause(char* error, size_t maxlen) { return true; }
    
    /**
     * @brief Called when plugin is unpaused.
     */
    virtual bool Unpause(char* error, size_t maxlen) { return true; }
    
    // Plugin metadata - pure virtual, must be implemented
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
    virtual void* MetaFactory(const char* iface, int* ret, PluginId* id) { return nullptr; }
    virtual void LogMsg(ISmmPlugin* plugin, const char* msg, ...) {}
    virtual void ConPrint(const char* str) {}
    virtual void ConPrintf(const char* fmt, ...) {}
    virtual void Format(char* buffer, size_t maxlen, const char* fmt, ...) {}
};

// Global variables that Metamod expects plugins to have
// These are set by PLUGIN_SAVEVARS() in the Load() function
extern ISmmAPI* g_SMAPI;
extern ISmmPlugin* g_PLAPI;
extern PluginId g_PLID;
extern void* g_SHPtr;

// Macro to declare global variables in header
#define PLUGIN_GLOBALVARS() \
    extern ISmmAPI* g_SMAPI; \
    extern ISmmPlugin* g_PLAPI; \
    extern PluginId g_PLID; \
    extern void* g_SHPtr;

// Macro to save variables in Load() - must be called first in Load()
#define PLUGIN_SAVEVARS() \
    g_SMAPI = ismm; \
    g_PLAPI = static_cast<ISmmPlugin*>(this); \
    g_PLID = id;

// Main plugin exposure macro - creates the CreateInterface function
// that Metamod calls to get the plugin instance
#define PLUGIN_EXPOSE(name, var) \
    ISmmAPI* g_SMAPI = nullptr; \
    ISmmPlugin* g_PLAPI = nullptr; \
    PluginId g_PLID = 0; \
    void* g_SHPtr = nullptr; \
    SMM_API void* CreateInterface(const char* pName, int* pReturnCode) { \
        if (pName && strcmp(pName, METAMOD_PLAPI_NAME) == 0) { \
            if (pReturnCode) *pReturnCode = META_IFACE_OK; \
            return static_cast<ISmmPlugin*>(&var); \
        } \
        if (pReturnCode) *pReturnCode = META_IFACE_FAILED; \
        return nullptr; \
    }

// Logging macros (stub implementations)
#define META_LOG if (g_SMAPI) g_SMAPI->LogMsg
#define META_CONPRINT if (g_SMAPI) g_SMAPI->ConPrint
#define META_CONPRINTF if (g_SMAPI) g_SMAPI->ConPrintf

#endif // ISMMPLUG_H
