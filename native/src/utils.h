// utils.h - Utility macros and templates for GoStrike
// Inspired by CounterStrikeSharp's virtual function call patterns

#ifndef GOSTRIKE_UTILS_H
#define GOSTRIKE_UTILS_H

#include <cstdint>

namespace gostrike {

// Call a virtual function by vtable index
// Works on both Linux and Windows (no __thiscall needed on Linux)
template <typename T, typename... Args>
inline T CallVirtual(void* instance, int index, Args... args) {
    using Fn = T(*)(void*, Args...);
    auto vtable = *reinterpret_cast<void***>(instance);
    return reinterpret_cast<Fn>(vtable[index])(instance, args...);
}

} // namespace gostrike

#endif // GOSTRIKE_UTILS_H
