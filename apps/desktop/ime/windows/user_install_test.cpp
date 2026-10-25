// Contract test only: callbacks record the requested TIP without OS changes.
#define wmain arion_control_main
#include "control.cpp"
#undef wmain

static std::wstring requested;
static const wchar_t *owned=L"0x0804:{0CF9199C-BC90-44F0-9331-474FA6742271}{78FBF1A9-3B48-4B37-A52C-B73D9D312263}";
static DWORD flags=0xffffffff;
static BOOL CALLBACK accept(LPCWSTR profile,DWORD value) { requested=profile; flags=value; return TRUE; }
static BOOL CALLBACK refuse(LPCWSTR,DWORD) { return FALSE; }
int wmain() {
    if (installUserProfileWith(accept)!=S_OK || flags!=0 || requested!=owned) { std::cerr<<"Unexpected user-profile installation request.\n"; return 1; }
    if (installUserProfileWith(accept,true)!=S_OK || flags!=0x00000001 || requested!=owned) { std::cerr<<"Unexpected user-profile removal request.\n"; return 1; }
    if (installUserProfileWith(refuse)!=E_FAIL || installUserProfileWith(refuse,true)!=E_FAIL || installUserProfileWith(nullptr)!=E_POINTER) { std::cerr<<"Failed installation request was not preserved.\n"; return 1; }
    if (installUserProfileWith(accept,true,ArionLegacyLanguage)!=S_OK || flags!=0x00000001 || requested!=userProfileString(ArionLegacyLanguage)) return 2;
    requested.clear(); flags=0xffffffff;
    if (installUserProfileWith(accept,false,ArionLegacyLanguage)!=HRESULT_FROM_WIN32(ERROR_REVISION_MISMATCH) || !requested.empty() || flags!=0xffffffff) return 3;
    if (installUserProfileWith(accept,true,0x0411)!=E_INVALIDARG || !requested.empty()) return 4;
    std::cout<<"PASS: exact owned TIP, install zero flags, uninstall one flag, native failure preserved; no input configuration changed.\n";
    return 0;
}
