#pragma once
#include <windows.h>
#include <vector>

inline bool tokenValue(HANDLE token, TOKEN_INFORMATION_CLASS kind, std::vector<BYTE> &value) {
    DWORD size = 0;
    GetTokenInformation(token, kind, nullptr, 0, &size);
    if (!size) return false;
    value.resize(size);
    return GetTokenInformation(token, kind, value.data(), size, &size) != FALSE;
}

inline bool tokenIntegrity(HANDLE token, DWORD &integrity) {
    std::vector<BYTE> label;
    if (!tokenValue(token, TokenIntegrityLevel, label)) return false;
    PSID sid = reinterpret_cast<TOKEN_MANDATORY_LABEL*>(label.data())->Label.Sid;
    if (!IsValidSid(sid) || !*GetSidSubAuthorityCount(sid)) return false;
    integrity = *GetSidSubAuthority(sid, *GetSidSubAuthorityCount(sid) - 1);
    return true;
}

// Local keyboard conversion runs inside the editor, including elevated editors.
// Sandboxed clients and system processes remain outside this desktop boundary.
inline bool keyboardToken(HANDLE token) {
    DWORD appContainer = 0, size = 0, integrity = 0;
    if (!GetTokenInformation(token, TokenIsAppContainer, &appContainer, sizeof(appContainer), &size) || appContainer) return false;
    return tokenIntegrity(token, integrity) && integrity >= SECURITY_MANDATORY_MEDIUM_RID && integrity < SECURITY_MANDATORY_SYSTEM_RID;
}

inline bool keyboardHost() {
    HANDLE token = nullptr;
    if (!OpenProcessToken(GetCurrentProcess(), TOKEN_QUERY, &token)) return false;
    bool allowed = keyboardToken(token);
    CloseHandle(token); return allowed;
}

// Voice IPC must not turn a lower-integrity process into an input source for
// an elevated editor. Its admission stays separate from local keyboard typing.
inline bool ordinaryToken(HANDLE token, DWORD &integrity) {
    TOKEN_ELEVATION elevation{}; DWORD size = 0;
    if (!GetTokenInformation(token, TokenElevation, &elevation, sizeof(elevation), &size) || elevation.TokenIsElevated) return false;
    return tokenIntegrity(token, integrity) && integrity >= SECURITY_MANDATORY_MEDIUM_RID && integrity < SECURITY_MANDATORY_HIGH_RID;
}

inline bool ordinaryHost() {
    HANDLE token = nullptr; DWORD integrity = 0;
    if (!OpenProcessToken(GetCurrentProcess(), TOKEN_QUERY, &token)) return false;
    bool allowed = ordinaryToken(token, integrity);
    CloseHandle(token); return allowed;
}
