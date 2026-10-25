#pragma once
#include <atomic>
inline std::atomic<long> objects{0};
// Declare first in objects whose members can execute module code while being
// destroyed. The final lease release follows all remaining member destructors.
struct CoreObjectLease {
    CoreObjectLease() { ++objects; }
    ~CoreObjectLease() { --objects; }
    CoreObjectLease(const CoreObjectLease&)=delete;
    CoreObjectLease &operator=(const CoreObjectLease&)=delete;
};
