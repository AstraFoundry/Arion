// Callback-only migration contracts. No OS registration or selection changes.
#define wmain arion_control_main
#include "control.cpp"
#undef wmain
#include <cstdlib>

static void require(bool value,const char *message) {
    if (!value) { std::cerr<<"Migration regression failed: "<<message<<'\n'; std::exit(1); }
}
static TF_INPUTPROCESSORPROFILE ownedRow(LANGID language) {
    TF_INPUTPROCESSORPROFILE value{};
    value.dwProfileType=TF_PROFILETYPE_INPUTPROCESSOR;
    value.langid=language; value.clsid=ArionService; value.guidProfile=ArionProfile;
    return value;
}
static OwnedProfiles observe(const std::vector<TF_INPUTPROCESSORPROFILE> &rows) {
    OwnedProfiles result; size_t index=0;
    require(observeOwnedProfilesWith([&](TF_INPUTPROCESSORPROFILE &row) { if (index==rows.size()) return S_FALSE; row=rows[index++]; return S_OK; },result)==S_OK,"enumerate exact owned rows");
    return result;
}
int wmain() {
    require(ArionLanguage==0x0804 && ArionLegacyLanguage==0x0409 && std::wstring(ArionName)==L"Arion","Chinese Pinyin identity");
    require(userProfileString()==L"0x0804:{0CF9199C-BC90-44F0-9331-474FA6742271}{78FBF1A9-3B48-4B37-A52C-B73D9D312263}","stable current CLSID and profile GUID");
    require(userProfileString(ArionLegacyLanguage)==L"0x0409:{0CF9199C-BC90-44F0-9331-474FA6742271}{78FBF1A9-3B48-4B37-A52C-B73D9D312263}","precise legacy TIP identity");
    auto unrelated=ownedRow(ArionLanguage); unrelated.clsid=GUID{};
    auto foreignProfile=ownedRow(ArionLanguage); foreignProfile.guidProfile=GUID{};
    auto layout=ownedRow(ArionLegacyLanguage); layout.dwProfileType=TF_PROFILETYPE_KEYBOARDLAYOUT;
    auto otherLanguage=ownedRow(0x0411);
    auto absent=observe({unrelated,foreignProfile,layout,otherLanguage});
    require(!absent.registered() && !absent.upgradeRequired(false),"other input methods are not owned");
    require(absent.upgradeRequired(true),"dangling COM registration is not absent");
    auto legacy=observe({unrelated,ownedRow(ArionLegacyLanguage),foreignProfile});
    require(legacy.registered() && legacy.legacyFound && !legacy.currentFound && legacy.upgradeRequired(true),"0409 discovered when 0804 is absent");
    require(currentProfileRequired(legacy)==HRESULT_FROM_WIN32(ERROR_REVISION_MISMATCH),"legacy enable/select/repair rejected");
    auto current=observe({ownedRow(ArionLanguage)});
    require(currentProfileRequired(current)==S_OK && !current.upgradeRequired(true),"current registration may be enabled");
    require(current.upgradeRequired(false),"dangling current profile requires explicit recovery");
    auto both=observe({ownedRow(ArionLanguage),ownedRow(ArionLegacyLanguage)});
    require(both.upgradeRequired(true) && currentProfileRequired(both)==HRESULT_FROM_WIN32(ERROR_REVISION_MISMATCH),"mixed-language registration never treated as ready");
    require(registrationPreflightWith(absent,L"",L"new")==S_OK,"fresh install may create its first protected payload");
    require(registrationPreflightWith(current,L"same",L"SAME")==S_OK,"same DLL current-profile repair permitted");
    for (const auto &state:{legacy,both}) {
        require(registrationPreflightWith(state,L"same",L"same")==HRESULT_FROM_WIN32(ERROR_REVISION_MISMATCH),"legacy is refused before copying even the same DLL");
        require(registrationPreflightWith(state,L"old",L"new")==HRESULT_FROM_WIN32(ERROR_REVISION_MISMATCH),"legacy replacement is refused before payload creation");
    }
    require(registrationPreflightWith(current,L"old",L"new")==HRESULT_FROM_WIN32(ERROR_REVISION_MISMATCH),"loaded current DLL cannot be replaced");
    require(registrationPreflightWith(current,L"",L"new")==HRESULT_FROM_WIN32(ERROR_REVISION_MISMATCH),"dangling current profile cannot be silently replaced");
    require(registrationPreflightWith(absent,L"old",L"new")==HRESULT_FROM_WIN32(ERROR_REVISION_MISMATCH),"orphan COM registration requires explicit removal");
    OwnedProfiles failed;
    require(observeOwnedProfilesWith([](TF_INPUTPROCESSORPROFILE&)->HRESULT { return E_ACCESSDENIED; },failed)==E_ACCESSDENIED,"enumeration failure never becomes absence");
    require(observeOwnedProfilesWith([](TF_INPUTPROCESSORPROFILE&)->HRESULT { return 2; },failed)==E_UNEXPECTED,"unexpected native enumeration result refused");
    std::vector<LANGID> deactivated,unregistered;
    auto deactivate=[&](REFCLSID service,LANGID language,REFGUID profile) { require(service==ArionService && profile==ArionProfile,"only owned deactivation"); deactivated.push_back(language); return S_FALSE; };
    auto unregister=[&](REFCLSID service,LANGID language,REFGUID profile,DWORD flags) { require(service==ArionService && profile==ArionProfile && flags==0,"precise profile removal without ALLPROFILES"); unregistered.push_back(language); return S_OK; };
    require(unregisterOwnedProfilesWith(legacy,deactivate,unregister)==S_OK && deactivated==std::vector<LANGID>{ArionLegacyLanguage} && unregistered==deactivated,"remove legacy language exactly");
    deactivated.clear(); unregistered.clear();
    require(unregisterOwnedProfilesWith(both,deactivate,unregister)==S_OK && unregistered==std::vector<LANGID>({ArionLanguage,ArionLegacyLanguage}) && deactivated==unregistered,"remove both known owned profiles only");
    require(unregisterOwnedProfilesWith(current,deactivate,[](auto&&...)->HRESULT { return E_ACCESSDENIED; })==E_ACCESSDENIED,"removal failure preserved");
    require(unregisterOwnedProfilesWith(current,deactivate,[](auto&&...)->HRESULT { return S_FALSE; })==HRESULT_FROM_WIN32(ERROR_NOT_READY),"unconfirmed unregister does not permit cleanup");
    std::vector<LANGID> disabled;
    require(disableOwnedProfilesWith(legacy,[&](LANGID language) { disabled.push_back(language); return S_OK; })==S_OK && disabled==std::vector<LANGID>{ArionLegacyLanguage},"per-user disable targets installed legacy language");
    require(disableOwnedProfilesWith(both,[](LANGID)->HRESULT { return E_ACCESSDENIED; })==E_ACCESSDENIED,"per-user failure preserved");
    std::cout<<"PASS: stable identity, dual-language discovery, migration refusal, exact legacy/current removal and unrelated-profile preservation. No OS registration changed.\n";
    return 0;
}
