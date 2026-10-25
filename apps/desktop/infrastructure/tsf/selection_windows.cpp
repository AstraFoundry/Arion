#include <windows.h>
#include <msctf.h>
#include "../../ime/windows/identity.h"

extern "C" long arion_tsf_selected(int *selected) {
    *selected = 0;
    HRESULT initialized = CoInitializeEx(nullptr, COINIT_APARTMENTTHREADED);
    if (FAILED(initialized) && initialized != RPC_E_CHANGED_MODE) return initialized;
    ITfInputProcessorProfileMgr *profiles = nullptr;
    HRESULT hr = CoCreateInstance(CLSID_TF_InputProcessorProfiles, nullptr, CLSCTX_INPROC_SERVER, IID_ITfInputProcessorProfileMgr, reinterpret_cast<void**>(&profiles));
    if (SUCCEEDED(hr)) {
        TF_INPUTPROCESSORPROFILE active{};
        hr = profiles->GetActiveProfile(GUID_TFCAT_TIP_KEYBOARD, &active);
        if (hr == S_OK) *selected = active.clsid == ArionService && active.guidProfile == ArionProfile;
        profiles->Release();
    }
    if (SUCCEEDED(initialized)) CoUninitialize();
    return hr;
}
