#pragma once
#include <windows.h>
#include <wincrypt.h>
#include <string>

inline std::wstring fileHash(const std::wstring &path) {
    HANDLE file = CreateFileW(path.c_str(), GENERIC_READ, FILE_SHARE_READ, nullptr, OPEN_EXISTING, FILE_FLAG_OPEN_REPARSE_POINT, nullptr);
    if (file == INVALID_HANDLE_VALUE) return {};
    BY_HANDLE_FILE_INFORMATION info{};
    bool ok = GetFileInformationByHandle(file, &info) && !(info.dwFileAttributes & (FILE_ATTRIBUTE_DIRECTORY | FILE_ATTRIBUTE_REPARSE_POINT));
    HCRYPTPROV provider = 0; HCRYPTHASH hash = 0;
    if (ok) ok = CryptAcquireContextW(&provider, nullptr, nullptr, PROV_RSA_AES, CRYPT_VERIFYCONTEXT) && CryptCreateHash(provider, CALG_SHA_256, 0, 0, &hash);
    BYTE buffer[16384]; DWORD count;
    while (ok) { if (!ReadFile(file, buffer, sizeof(buffer), &count, nullptr)) { ok = false; break; } if (!count) break; ok = CryptHashData(hash, buffer, count, 0) != FALSE; }
    BYTE digest[32]; DWORD size = 32;
    if (ok) ok = CryptGetHashParam(hash, HP_HASHVAL, digest, &size, 0) != FALSE;
    if (hash) CryptDestroyHash(hash);
    if (provider) CryptReleaseContext(provider, 0);
    CloseHandle(file);
    if (!ok) return {};
    std::wstring result; for (BYTE byte : digest) { const wchar_t *hex = L"0123456789abcdef"; result += hex[byte >> 4]; result += hex[byte & 15]; } return result;
}

inline bool hashName(const std::wstring &value) {
    if (value.size() != 64) return false;
    for (wchar_t c : value) if (!((c >= L'0' && c <= L'9') || (c >= L'a' && c <= L'f'))) return false;
    return true;
}
