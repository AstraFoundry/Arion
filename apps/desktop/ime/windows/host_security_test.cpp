#define WIN32_LEAN_AND_MEAN
#include <windows.h>
#include <iostream>
#include "host_security.h"

static bool require(bool value, const char *name) {
    if (!value) std::cerr << "FAIL: " << name << std::endl;
    return value;
}

int main() {
    DWORD integrity = 0;
    if (!require(!keyboardToken(nullptr) && !ordinaryToken(nullptr, integrity), "unknown tokens are refused")) return 1;
    HANDLE token = nullptr;
    if (!OpenProcessToken(GetCurrentProcess(), TOKEN_QUERY | TOKEN_DUPLICATE, &token)) return 2;
    TOKEN_ELEVATION elevation{}; DWORD size = 0;
    if (!GetTokenInformation(token, TokenElevation, &elevation, sizeof(elevation), &size) || !tokenIntegrity(token, integrity)) return 2;
    if (!require(keyboardHost() && keyboardToken(token), "current desktop token supports local keyboard input")) return 1;
    if (elevation.TokenIsElevated && !require(!ordinaryHost() && !ordinaryToken(token, integrity), "elevated keyboard host cannot use ordinary voice IPC")) return 1;
    if (!elevation.TokenIsElevated && !require(ordinaryHost(), "ordinary desktop voice admission remains available")) return 1;
    // Exercise real kernel token labels, not a synthetic integrity classifier.
    for (DWORD level : {SECURITY_MANDATORY_LOW_RID, SECURITY_MANDATORY_MEDIUM_RID}) {
        HANDLE copy = nullptr; PSID sid = nullptr;
        if (!DuplicateTokenEx(token, TOKEN_QUERY | TOKEN_ADJUST_DEFAULT, nullptr, SecurityImpersonation, TokenPrimary, &copy)) return 2;
        SID_IDENTIFIER_AUTHORITY authority = SECURITY_MANDATORY_LABEL_AUTHORITY;
        if (!AllocateAndInitializeSid(&authority, 1, level, 0,0,0,0,0,0,0, &sid)) return 2;
        TOKEN_MANDATORY_LABEL label{}; label.Label.Sid = sid; label.Label.Attributes = SE_GROUP_INTEGRITY;
        bool changed = SetTokenInformation(copy, TokenIntegrityLevel, &label, sizeof(label) + GetLengthSid(sid)) != FALSE;
        bool allowed = keyboardToken(copy);
        bool voice = ordinaryToken(copy, integrity);
        FreeSid(sid); CloseHandle(copy);
        if (!require(changed, "test token integrity changes successfully")) return 1;
        if (!require(allowed == (level == SECURITY_MANDATORY_MEDIUM_RID), "low token refused and medium token accepted for keyboard")) return 1;
        if (level == SECURITY_MANDATORY_LOW_RID && !require(!voice, "low token cannot use voice IPC")) return 1;
    }
    CloseHandle(token);
    std::cout << "{\"passed\":true,\"elevated\":" << (elevation.TokenIsElevated ? "true" : "false") << "}" << std::endl;
    return 0;
}
