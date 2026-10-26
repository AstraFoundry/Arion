// Callback contracts: no COM registration or user configuration is changed.
#define wmain arion_control_main
#include "control.cpp"
#undef wmain

int wmain() {
    std::vector<GUID> seen; std::vector<bool> removal;
    auto record=[&](REFGUID category,bool remove) { seen.push_back(category); removal.push_back(remove); return S_OK; };
    auto correct=[&](bool remove) { return seen.size()==2 && seen[0]==GUID_TFCAT_TIP_KEYBOARD && seen[1]==ArionSystemTrayCategory && removal[0]==remove && removal[1]==remove; };
    if (changeManagedCategoriesWith(record,false)!=S_OK || !correct(false)) return 1;
    seen.clear(); removal.clear();
    if (changeManagedCategoriesWith(record,true)!=S_OK || !correct(true)) return 2;
    int attempts=0;
    auto fail=[&](REFGUID,bool) { ++attempts; return E_ACCESSDENIED; };
    if (changeManagedCategoriesWith(fail,false)!=E_ACCESSDENIED || attempts!=1) return 3;
    attempts=0;
    if (changeManagedCategoriesWith(fail,true)!=E_ACCESSDENIED || attempts!=2) return 4;
    int repairs=0; auto repair=[&] { ++repairs; return S_OK; };
    if (repairExistingRegistrationWith(L"C:\\Protected\\ArionTSF.dll",L"c:\\protected\\ariontsf.dll",repair)!=S_OK || repairs!=1) return 5;
    if (repairExistingRegistrationWith(L"C:\\Old\\ArionTSF.dll",L"C:\\New\\ArionTSF.dll",repair)!=HRESULT_FROM_WIN32(ERROR_REVISION_MISMATCH) || repairs!=1) return 6;
    if (repairExistingRegistrationWith(L"same",L"same",[] { return E_ACCESSDENIED; })!=E_ACCESSDENIED) return 7;
    std::cout<<"PASS: only keyboard/system-tray categories; symmetric removal; same-DLL repair; different DLL refused; native failures preserved. No registration changed.\n";
    return 0;
}
