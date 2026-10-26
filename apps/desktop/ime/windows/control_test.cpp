// Read-only regression against an already registered Arion test profile.
// This executable never installs, enables, selects or removes an input source.
#define wmain arion_control_main
#include "control.cpp"
#undef wmain
#include <sstream>

int wmain() {
    HRESULT initialized=CoInitializeEx(nullptr,COINIT_APARTMENTTHREADED);
    if (FAILED(initialized)) return 1;
    ITfInputProcessorProfileMgr *modern=nullptr;
    HRESULT hr=CoCreateInstance(CLSID_TF_InputProcessorProfiles,nullptr,CLSCTX_INPROC_SERVER,IID_ITfInputProcessorProfileMgr,reinterpret_cast<void**>(&modern));
    bool current=false,old=false;
    TF_INPUTPROCESSORPROFILE catalog{},legacyCatalog{};
    IEnumTfInputProcessorProfiles *enumerator=nullptr;
    if (SUCCEEDED(hr)) hr=modern->EnumProfiles(0,&enumerator);
    if (SUCCEEDED(hr)) {
        ULONG fetched=0; TF_INPUTPROCESSORPROFILE item{};
        while ((hr=enumerator->Next(1,&item,&fetched))==S_OK && fetched==1) {
            if (item.dwProfileType!=TF_PROFILETYPE_INPUTPROCESSOR || item.clsid!=ArionService || item.guidProfile!=ArionProfile) continue;
            if (item.langid==ArionLanguage) { current=true; catalog=item; }
            if (item.langid==ArionLegacyLanguage) { old=true; legacyCatalog=item; }
        }
        enumerator->Release();
        if (hr==S_FALSE) hr=S_OK;
    }
    if (!current && old) catalog=legacyCatalog;
    BOOL enabled=FALSE;
    ITfInputProcessorProfiles *profiles=nullptr;
    if (SUCCEEDED(hr) && (current || old)) hr=modern->QueryInterface(IID_ITfInputProcessorProfiles,reinterpret_cast<void**>(&profiles));
    if (SUCCEEDED(hr) && profiles) { hr=profiles->IsEnabledLanguageProfile(ArionService,catalog.langid,ArionProfile,&enabled); profiles->Release(); }
    TF_INPUTPROCESSORPROFILE active{};
    HRESULT selected=SUCCEEDED(hr) ? modern->GetActiveProfile(GUID_TFCAT_TIP_KEYBOARD,&active) : hr;
    if (modern) modern->Release();
    CoUninitialize();
    if (FAILED(hr) || FAILED(selected)) { std::cerr<<"Native read-only profile observation failed.\n"; return 1; }
    bool hasComPath=!registeredPath().empty();
    if (!current && !old && !hasComPath) { std::cerr<<"An existing Arion profile or COM registration is required for this read-only regression.\n"; return 77; }
    std::ostringstream captured;
    auto output=std::cout.rdbuf(captured.rdbuf());
    wchar_t program[]=L"control-test",operation[]=L"query"; wchar_t *arguments[]={program,operation};
    int result=arion_control_main(2,arguments);
    std::cout.rdbuf(output);
    bool ready=enabled && (catalog.dwFlags&TF_IPP_FLAG_ENABLED)!=0;
    std::string expected=ready ? "\"enabled\":true" : "\"enabled\":false";
    std::wstring installed,build;bool loader=false;
    HRESULT payload=hasComPath?ownedPayload(installed,build,&loader):S_OK;
    bool upgrade=old || ((current || old)!=hasComPath) || (hasComPath && (!current || !loader));
    bool activeOwned=selected==S_OK && active.dwProfileType==TF_PROFILETYPE_INPUTPROCESSOR && ((current && active.langid==ArionLanguage) || (old && active.langid==ArionLegacyLanguage)) && active.clsid==ArionService && active.guidProfile==ArionProfile;
    if (FAILED(payload) || result!=0 || captured.str().find(expected)==std::string::npos || captured.str().find("\"registered\":true")==std::string::npos || captured.str().find(loader?"\"loader\":true":"\"loader\":false")==std::string::npos || captured.str().find(old ? "\"legacy\":true" : "\"legacy\":false")==std::string::npos || captured.str().find(upgrade ? "\"upgradeRequired\":true" : "\"upgradeRequired\":false")==std::string::npos || captured.str().find(activeOwned ? "\"selected\":true" : "\"selected\":false")==std::string::npos) { std::cerr<<"Query differs from independent current/legacy observation: "<<captured.str(); return 1; }
    std::cout<<"PASS: read-only query matches independent catalog/user/active state (0804="<<current<<", 0409="<<old<<", upgrade="<<upgrade<<").\n"<<captured.str();
    return 0;
}
