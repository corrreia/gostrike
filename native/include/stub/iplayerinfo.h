// Stub iplayerinfo.h for compilation without full SDK
#ifndef IPLAYERINFO_H
#define IPLAYERINFO_H

#include <cstdint>

// Player slot wrapper
class CPlayerSlot {
public:
    CPlayerSlot() : m_slot(-1) {}
    CPlayerSlot(int slot) : m_slot(slot) {}
    int Get() const { return m_slot; }
private:
    int m_slot;
};

// Buffer string for reject reasons
class CBufferString {
public:
    void Set(const char* str) {}
};

// Network disconnect reason
enum ENetworkDisconnectionReason {
    NETWORK_DISCONNECT_INVALID = 0,
    NETWORK_DISCONNECT_SHUTDOWN,
    NETWORK_DISCONNECT_DISCONNECT_BY_USER,
    NETWORK_DISCONNECT_DISCONNECT_BY_SERVER,
    NETWORK_DISCONNECT_KICKED,
    NETWORK_DISCONNECT_BANADDED,
};

#endif // IPLAYERINFO_H
