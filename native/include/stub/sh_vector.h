// Stub sh_vector.h - SourceHook vector compatibility
#ifndef SH_VECTOR_H
#define SH_VECTOR_H

#include <vector>

namespace SourceHook {
    template<typename T>
    using CVector = std::vector<T>;
}

#endif // SH_VECTOR_H
