// Native selection contract only; no OS input state is changed.
#define wmain arion_control_main
#include "control.cpp"
#undef wmain

int wmain() {
    int observations=0;
    auto selected=[&](TF_INPUTPROCESSORPROFILE &profile) { ++observations; profile.dwProfileType=TF_PROFILETYPE_INPUTPROCESSOR; profile.langid=ArionLanguage; profile.clsid=ArionService; profile.guidProfile=ArionProfile; return S_OK; };
    if (confirmSelectionWith(S_FALSE,selected)!=HRESULT_FROM_WIN32(ERROR_NOT_READY) || observations) return 1;
    if (confirmSelectionWith(E_ACCESSDENIED,selected)!=E_ACCESSDENIED || observations) return 2;
    if (confirmSelectionWith(S_OK,selected)!=S_OK || observations!=1) return 3;
    if (confirmSelectionWith(S_OK,[](TF_INPUTPROCESSORPROFILE&) { return E_FAIL; })!=E_FAIL) return 4;
    if (confirmSelectionWith(S_OK,[](TF_INPUTPROCESSORPROFILE&) { return S_FALSE; })!=HRESULT_FROM_WIN32(ERROR_NOT_READY)) return 5;
    if (confirmSelectionWith(S_OK,[](TF_INPUTPROCESSORPROFILE&) { return S_OK; })!=HRESULT_FROM_WIN32(ERROR_RETRY)) return 6;
    for (int field=0;field<4;++field) {
        auto wrong=[&](TF_INPUTPROCESSORPROFILE &profile) {
            selected(profile);
            if (field==0) profile.dwProfileType=TF_PROFILETYPE_KEYBOARDLAYOUT;
            if (field==1) profile.langid=ArionLegacyLanguage;
            if (field==2) profile.clsid=GUID{};
            if (field==3) profile.guidProfile=GUID{};
            return S_OK;
        };
        if (confirmSelectionWith(S_OK,wrong)!=HRESULT_FROM_WIN32(ERROR_RETRY)) return 7+field;
    }
    std::cout<<"PASS: S_FALSE, native errors and unobserved/wrong profile are not selection success; exact owned profile required. No OS settings changed.\n";
    return 0;
}
