#pragma once
#include <windows.h>
#include <msctf.h>
#include <sddl.h>
#include <string>
#include <vector>

#ifndef __ITfTextInputProcessorEx_INTERFACE_DEFINED__
inline constexpr IID IID_ITfTextInputProcessorEx = {0x6e4e2102,0xf9cd,0x433d,{0xb4,0x96,0x30,0x3c,0xe0,0x3a,0x65,0x07}};
struct ITfTextInputProcessorEx : ITfTextInputProcessor {
    virtual HRESULT STDMETHODCALLTYPE ActivateEx(ITfThreadMgr*, TfClientId, DWORD) = 0;
};
#endif
#ifndef TF_CLIENTID_NULL
#define TF_CLIENTID_NULL ((TfClientId)0)
#endif
#ifndef TF_IPPMF_FORSESSION
#define TF_IPPMF_FORSESSION 0x20000000
#define TF_IPPMF_DONTCARECURRENTINPUTLANGUAGE 0x00000004
#define TF_IPP_FLAG_ENABLED 0x00000002
#endif

// Stable identities shared by the DLL and its registration helper.
inline constexpr GUID ArionService = {0x0cf9199c,0xbc90,0x44f0,{0x93,0x31,0x47,0x4f,0xa6,0x74,0x22,0x71}};
inline constexpr GUID ArionProfile = {0x78fbf1a9,0x3b48,0x4b37,{0xa5,0x2c,0xb7,0x3d,0x9d,0x31,0x22,0x63}};
inline constexpr LANGID ArionLanguage = 0x0804;
inline constexpr LANGID ArionLegacyLanguage = 0x0409;
inline constexpr wchar_t ArionName[] = L"Arion";
inline constexpr wchar_t ArionClass[] = L"Arion.Voice.TSF.Dispatch";

inline std::wstring userPipe() {
    HANDLE token = nullptr;
    if (!OpenProcessToken(GetCurrentProcess(), TOKEN_QUERY, &token)) return {};
    DWORD size = 0;
    GetTokenInformation(token, TokenUser, nullptr, 0, &size);
    std::vector<BYTE> buffer(size);
    bool ok = GetTokenInformation(token, TokenUser, buffer.data(), size, &size) != FALSE;
    CloseHandle(token);
    if (!ok) return {};
    wchar_t *sid = nullptr;
    if (!ConvertSidToStringSidW(reinterpret_cast<TOKEN_USER*>(buffer.data())->User.Sid, &sid)) return {};
    std::wstring result = L"\\\\.\\pipe\\Arion.Voice.TSF.";
    result += sid;
    LocalFree(sid);
    return result;
}
