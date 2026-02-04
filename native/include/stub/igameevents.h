// Stub igameevents.h for compilation without full SDK
#ifndef IGAMEEVENTS_H
#define IGAMEEVENTS_H

class IGameEvent {
public:
    virtual ~IGameEvent() = default;
    virtual const char* GetName() const = 0;
    virtual int GetInt(const char* key, int defaultValue = 0) { return defaultValue; }
    virtual float GetFloat(const char* key, float defaultValue = 0.0f) { return defaultValue; }
    virtual const char* GetString(const char* key, const char* defaultValue = "") { return defaultValue; }
    virtual bool GetBool(const char* key, bool defaultValue = false) { return defaultValue; }
};

class IGameEventListener2 {
public:
    virtual ~IGameEventListener2() = default;
    virtual void FireGameEvent(IGameEvent* event) = 0;
};

class IGameEventManager2 {
public:
    virtual ~IGameEventManager2() = default;
    virtual bool AddListener(IGameEventListener2* listener, const char* name, bool serverSide) { return true; }
    virtual void RemoveListener(IGameEventListener2* listener) {}
};

#endif // IGAMEEVENTS_H
