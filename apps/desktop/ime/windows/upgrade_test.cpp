// Callback-only protected-version handover. No real COM/registry mutation.
#define wmain arion_control_main
#include "control.cpp"
#undef wmain
#include <cstdlib>

static void require(bool ok,const char *message) { if (!ok) { std::cerr<<"Upgrade regression: "<<message<<'\n'; std::exit(1); } }
int wmain() {
    OwnedProfiles current;current.currentFound=true;
    OwnedProfiles legacy;legacy.legacyFound=true;
    OwnedProfiles both=current;both.legacyFound=true;
    TF_INPUTPROCESSORPROFILE active{};active.dwProfileType=TF_PROFILETYPE_INPUTPROCESSOR;active.clsid=ArionService;active.guidProfile=ArionProfile;active.langid=ArionLegacyLanguage;
    require(ownedActiveProfile(both,active) && !ownedActiveProfile(current,active),"mixed-catalog old-language selection remains visible and defers background approval");
    active.langid=ArionLanguage;require(ownedActiveProfile(both,active),"mixed-catalog current-language selection visible");
    for (const auto &owned:{current,legacy,both}) {
        require(upgradePreflightWith(owned,L"old\\ArionTSF.dll",L"OLD\\ArionTSF.dll",false)==S_OK,"known current/legacy profiles may switch immutable versions");
        require(upgradePreflightWith(owned,L"old",L"different",false)==HRESULT_FROM_WIN32(ERROR_REVISION_MISMATCH),"concurrent registration change refused");
        require(upgradePreflightWith(owned,L"old",L"old",true)==HRESULT_FROM_WIN32(ERROR_BUSY),"selected profile defers without destructive action");
        require(upgradePreflightWith(owned,L"old",L"old",true,true)==S_OK,"verified stable loader permits selected-profile core publication");
    }
    require(FAILED(upgradePreflightWith({},L"old",L"old",false)),"orphan COM registration requires recovery");
    require(FAILED(upgradePreflightWith(current,L"",L"",false)),"missing previous code is never guessed");
    UpgradeSnapshot snapshot;
    std::wstring restored=L"new core"; int erased=0;
    require(restoreCorePathWith(snapshot,[&](const std::wstring &value){restored=value;return S_OK;},[&]{restored.clear();++erased;return S_OK;})==S_OK && restored.empty() && erased==1,"direct legacy rollback restores absent CorePath");
    snapshot.corePathPresent=true;snapshot.corePath=L"previous core";
    require(restoreCorePathWith(snapshot,[&](const std::wstring &value){restored=value;return S_OK;},[&]{++erased;return S_OK;})==S_OK && restored==snapshot.corePath && erased==1,"loader rollback restores exact prior CorePath");
    require(restoreCorePathWith(snapshot,[](const std::wstring &){return E_ACCESSDENIED;},[]{return S_OK;})==E_ACCESSDENIED,"CorePath restore failure remains explicit");
    snapshot.path=L"direct legacy";
    require(snapshotIcon(snapshot)==snapshot.path,"direct registration ignores stale CorePath for icon restoration");
    snapshot.loader=true;
    require(snapshotIcon(snapshot)==snapshot.corePath,"loader registration uses versioned core icon");
    for (bool fail:{false,true}) {
        std::vector<int> calls;
        HRESULT hr=upgradeRegistrationModeWith(true,[&]{calls.push_back(0);return fail?E_ACCESSDENIED:S_OK;},[&]{calls.push_back(1);return S_OK;},[&]{calls.push_back(2);return S_OK;},[&]{calls.push_back(3);return S_OK;},[&]{calls.push_back(4);return S_OK;});
        require(hr==(fail?E_ACCESSDENIED:S_OK) && calls==(fail?std::vector<int>{0,4}:std::vector<int>{0}),"stable publication never touches active profile, categories or retirement; failure restores only publication");
    }
    for (int failed=-1;failed<4;++failed) {
        std::vector<int> calls;int visible=0;bool oldProfile=true;const std::wstring oldBytes=L"immutable old payload";std::wstring oldPayload=oldBytes;
        auto step=[&](int number) {calls.push_back(number);visible=number+1;if(number==3)oldProfile=false;return number==failed?E_ACCESSDENIED:S_OK;};
        int rollbacks=0;
        auto hr=upgradeRegistrationWith([&]{return step(0);},[&]{return step(1);},[&]{return step(2);},[&]{return step(3);},[&]{calls.push_back(4);++rollbacks;visible=0;oldProfile=true;return S_OK;});
        if (failed<0) require(hr==S_OK && calls==std::vector<int>({0,1,2,3}) && !oldProfile && !rollbacks,"publish complete new profile before retiring legacy registration");
        else {
            require(hr==E_ACCESSDENIED && rollbacks==1 && visible==0 && oldProfile,"every partially applied phase restores old registration");
            require(calls.size()==static_cast<size_t>(failed+2) && calls.back()==4,"failure stops later mutation and rolls back once");
        }
        require(oldPayload==oldBytes,"handover never mutates existing payload bytes");
    }
    int rolled=0;
    require(upgradeRegistrationWith([]{return S_FALSE;},[]{return S_OK;},[]{return S_OK;},[]{return S_OK;},[&]{++rolled;return S_OK;})==HRESULT_FROM_WIN32(ERROR_NOT_READY) && rolled==1,"unconfirmed phase is not successful installation");
    require(upgradeRegistrationWith([]{return E_ACCESSDENIED;},[]{return S_OK;},[]{return S_OK;},[]{return S_OK;},[]{return E_ABORT;})==E_ABORT,"rollback failure remains explicit");
    wchar_t directory[32768]{};require(GetCurrentDirectoryW(32768,directory)!=0,"owned fixture directory");
    const std::wstring intent=std::wstring(directory)+L"\\upgrade-intent-fixture",next(64,L'b'),old(64,L'a');
    std::wstring target,previous;
    require(readUpgradeIntent(intent,target,previous)==S_FALSE,"missing intent is distinct from corrupt receipt");
    require(writeUpgradeIntent(intent,next,old)==S_OK && readUpgradeIntent(intent,target,previous)==S_OK && target==next && previous==old,"durable intent roundtrip contains only reviewed version identities");
    HANDLE locked=CreateFileW(intent.c_str(),GENERIC_READ,FILE_SHARE_READ,nullptr,OPEN_EXISTING,0,nullptr);
    require(locked!=INVALID_HANDLE_VALUE,"reserve original intent against replacement");
    require(FAILED(writeUpgradeIntent(intent,old,next)),"failed atomic publication cannot replace original receipt");
    CloseHandle(locked);
    require(readUpgradeIntent(intent,target,previous)==S_OK && target==next && previous==old,"original receipt survives replacement failure");
    const auto temporary=intent+L".next";require(DeleteFileW(temporary.c_str()),"remove only owned unsuccessful next receipt");
    require(CreateHardLinkW(temporary.c_str(),intent.c_str(),nullptr),"create owned hardlink attack fixture");
    const auto before=fileHash(intent);
    require(FAILED(writeUpgradeIntent(intent,old,next)) && fileHash(intent)==before,"linked temporary is rejected before truncating existing bytes");
    require(DeleteFileW(temporary.c_str()),"remove only owned hardlink");
    HANDLE corrupt=CreateFileW(intent.c_str(),GENERIC_WRITE,0,nullptr,OPEN_EXISTING,0,nullptr);DWORD written=0;
    require(corrupt!=INVALID_HANDLE_VALUE && WriteFile(corrupt,"!",1,&written,nullptr) && written==1,"corrupt owned receipt fixture");CloseHandle(corrupt);
    require(FAILED(readUpgradeIntent(intent,target,previous)),"corrupt intent never becomes absent or a guessed build");
    require(clearUpgradeIntent(intent)==S_OK && clearUpgradeIntent(intent)==S_OK,"intent cleanup is exact and idempotent");
    std::cout<<"PASS: protected version switch, current/legacy migration, selected/concurrent deferral, exact phase order and all rollback failures. No registration changed.\n";
    return 0;
}
