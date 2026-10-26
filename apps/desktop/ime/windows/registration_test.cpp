// Regression contracts for the actual default-enable catalog failure.
#define wmain arion_control_main
#include "control.cpp"
#undef wmain

int wmain() {
    auto verify=[](REFCLSID service,LANGID language,REFGUID profile,const wchar_t *description,ULONG descriptionLength,const wchar_t *icon,ULONG iconLength,ULONG iconIndex,HKL substitute,DWORD preferred,BOOL enabled,DWORD flags)->HRESULT {
        return service==ArionService && language==0x0804 && profile==ArionProfile && std::wstring(description,descriptionLength)==L"Arion" && std::wstring(icon,iconLength)==L"C:\\Protected\\ArionTSF.dll" && iconIndex==0 && substitute==nullptr && preferred==0 && enabled==TRUE && flags==0 ? S_OK : E_INVALIDARG;
    };
    if (registerOwnedProfileWith(verify,L"C:\\Protected\\ArionTSF.dll")!=S_OK) return 1;
    if (registerOwnedProfileWith([](auto&&...)->HRESULT { return E_ACCESSDENIED; },L"owned")!=E_ACCESSDENIED) return 2;
    TF_INPUTPROCESSORPROFILE profile{};
    if (profileEnabled(profile,TRUE) || profileEnabled(profile,FALSE)) return 3;
    profile.dwFlags=TF_IPP_FLAG_ENABLED;
    if (!profileEnabled(profile,TRUE) || profileEnabled(profile,FALSE)) return 4;
    if (enableOwnedByDefaultWith([](REFCLSID service,LANGID language,REFGUID profile,BOOL enabled)->HRESULT { return service==ArionService && language==0x0804 && profile==ArionProfile && enabled==TRUE ? S_OK : E_INVALIDARG; })!=S_OK) return 5;
    if (enableOwnedByDefaultWith([](auto&&...)->HRESULT { return E_ACCESSDENIED; })!=E_ACCESSDENIED) return 6;
    std::cout<<"PASS: exact owned registration and repair enable catalog by default; query requires both user and catalog enabled; native failure preserved. No registration changed.\n";
    return 0;
}
