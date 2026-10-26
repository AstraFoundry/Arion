#define WIN32_LEAN_AND_MEAN
#include <windows.h>
#include <msctf.h>
#include <sddl.h>
#include <shlobj.h>
#include <aclapi.h>
#include <string>
#include <vector>
#include <iostream>
#include <cstring>
#include "identity.h"
#include "wire.h"
#include "payload.h"
#include "keyboard_payload.h"
#include "loader.h"

static std::wstring classKey() { wchar_t id[40]; StringFromGUID2(ArionService, id, 40); return std::wstring(L"Software\\Classes\\CLSID\\") + id; }
using InstallLayoutOrTipFunction=BOOL(CALLBACK*)(LPCWSTR,DWORD);
static bool ownedLanguageProfile(const TF_INPUTPROCESSORPROFILE &profile,LANGID language) {
    return profile.dwProfileType==TF_PROFILETYPE_INPUTPROCESSOR && profile.langid==language && profile.clsid==ArionService && profile.guidProfile==ArionProfile;
}
struct OwnedProfiles {
    bool currentFound=false,legacyFound=false;
    TF_INPUTPROCESSORPROFILE current{},legacy{};
    bool registered() const { return currentFound || legacyFound; }
    bool upgradeRequired(bool hasComPath) const { return legacyFound || (registered()!=hasComPath) || (hasComPath && !currentFound); }
};
static bool ownedActiveProfile(const OwnedProfiles &owned,const TF_INPUTPROCESSORPROFILE &active) {
    return (owned.currentFound && ownedLanguageProfile(active,ArionLanguage)) || (owned.legacyFound && ownedLanguageProfile(active,ArionLegacyLanguage));
}
template<typename Next> static HRESULT observeOwnedProfilesWith(Next next,OwnedProfiles &owned) {
    owned={};
    while (true) {
        TF_INPUTPROCESSORPROFILE profile{};
        HRESULT hr=next(profile);
        if (hr==S_FALSE) return S_OK;
        if (hr!=S_OK) return FAILED(hr) ? hr : E_UNEXPECTED;
        if (ownedLanguageProfile(profile,ArionLanguage)) { owned.currentFound=true; owned.current=profile; }
        else if (ownedLanguageProfile(profile,ArionLegacyLanguage)) { owned.legacyFound=true; owned.legacy=profile; }
    }
}
static HRESULT observeOwnedProfiles(ITfInputProcessorProfileMgr *profiles,OwnedProfiles &owned) {
    IEnumTfInputProcessorProfiles *all=nullptr;
    // Language zero enumerates all installed languages, including the old
    // English voice profile that an 0804-only query cannot discover.
    HRESULT hr=profiles->EnumProfiles(0,&all);
    if (hr!=S_OK) return FAILED(hr) ? hr : E_UNEXPECTED;
    if (!all) return E_UNEXPECTED;
    hr=observeOwnedProfilesWith([&](TF_INPUTPROCESSORPROFILE &profile) {
        ULONG count=0;
        HRESULT result=all->Next(1,&profile,&count);
        if ((result==S_OK && count!=1) || (result==S_FALSE && count!=0)) return E_UNEXPECTED;
        return result;
    },owned);
    all->Release(); return hr;
}
static HRESULT currentProfileRequired(const OwnedProfiles &owned) {
    if (owned.legacyFound) return HRESULT_FROM_WIN32(ERROR_REVISION_MISMATCH);
    return owned.currentFound ? S_OK : HRESULT_FROM_WIN32(ERROR_NOT_FOUND);
}
static std::wstring userProfileString(LANGID language=ArionLanguage) {
    wchar_t languageText[10]{},service[40]{},profile[40]{};
    swprintf(languageText,10,L"0x%04x:",static_cast<unsigned>(language));
    StringFromGUID2(ArionService,service,40); StringFromGUID2(ArionProfile,profile,40);
    return std::wstring(languageText)+service+profile;
}
static HRESULT installUserProfileWith(InstallLayoutOrTipFunction install, bool remove = false,LANGID language=ArionLanguage) {
    if (!install) return E_POINTER;
    if (language!=ArionLanguage && language!=ArionLegacyLanguage) return E_INVALIDARG;
    if (!remove && language!=ArionLanguage) return HRESULT_FROM_WIN32(ERROR_REVISION_MISMATCH);
    // Exactly one owned TIP. Zero enables it; ILOT_UNINSTALL removes only it.
    return install(userProfileString(language).c_str(),remove ? 0x00000001 : 0) ? S_OK : E_FAIL;
}
static HRESULT installUserProfile(bool remove = false,LANGID language=ArionLanguage) {
    HMODULE input=LoadLibraryExW(L"input.dll",nullptr,LOAD_LIBRARY_SEARCH_SYSTEM32);
    if (!input) return HRESULT_FROM_WIN32(GetLastError());
    FARPROC address=GetProcAddress(input,"InstallLayoutOrTip");
    InstallLayoutOrTipFunction install=nullptr;
    static_assert(sizeof(install)==sizeof(address));
    std::memcpy(&install,&address,sizeof(install));
    HRESULT result=install ? installUserProfileWith(install,remove,language) : HRESULT_FROM_WIN32(ERROR_PROC_NOT_FOUND);
    FreeLibrary(input); return result;
}
static std::wstring registeredPath(HRESULT *observed=nullptr) {
    std::wstring path; HRESULT hr=arionLoader::readValue(nullptr,path);
    if (observed) *observed=hr;
    return hr==S_OK?path:L"";
}
static std::wstring loaderHash() {
#ifdef ARION_TSF_LOADER_SHA256
    const std::string hash=ARION_TSF_LOADER_SHA256;
    return std::wstring(hash.begin(),hash.end());
#else
    return {};
#endif
}
static HRESULT verifyLoader(const std::wstring &root,const std::wstring &path) {
    if (!arionLoader::loaderPath(root,path)) return E_ACCESSDENIED;
    const auto hash=loaderHash();
    if (!hashName(hash)) return HRESULT_FROM_WIN32(ERROR_NOT_SUPPORTED);
    HRESULT hr=arionLoader::protectedPath(root,path);
    return FAILED(hr)?hr:fileHash(path)==hash?S_OK:TRUST_E_BAD_DIGEST;
}
static HRESULT resolveRegistration(std::wstring &core,bool &loader) {
    loader=false; HRESULT hr=S_OK;
    auto path=registeredPath(&hr);
    if (hr!=S_OK) return hr;
    if (path.empty()) return E_ACCESSDENIED;
    std::wstring root; hr=arionLoader::machineRoot(root); if (FAILED(hr)) return hr;
    if (arionLoader::loaderPath(root,path)) {
        hr=verifyLoader(root,path); if (FAILED(hr)) return hr;
        hr=arionLoader::readValue(L"CorePath",core);
        if (hr!=S_OK) return FAILED(hr)?hr:HRESULT_FROM_WIN32(ERROR_INVALID_DATA);
        loader=true;
    } else core=path;
    std::wstring hash; return arionLoader::validateCore(root,core,hash);
}
static HRESULT setValue(const std::wstring &key, const wchar_t *name, const std::wstring &value) {
    HKEY handle; LSTATUS status = RegCreateKeyExW(HKEY_LOCAL_MACHINE, key.c_str(), 0, nullptr, 0, KEY_WRITE, nullptr, &handle, nullptr);
    if (status != ERROR_SUCCESS) return HRESULT_FROM_WIN32(status);
    status = RegSetValueExW(handle, name, 0, REG_SZ, reinterpret_cast<const BYTE*>(value.c_str()), static_cast<DWORD>((value.size() + 1) * sizeof(wchar_t)));
    RegCloseKey(handle); return HRESULT_FROM_WIN32(status);
}
// Windows SDK GUID_TFCAT_TIPCAP_SYSTRAYSUPPORT; older MinGW headers omit it.
static constexpr GUID ArionSystemTrayCategory={0x25504fb4,0x7bab,0x4bc1,{0x9c,0x69,0xcf,0x81,0x89,0x0f,0x0e,0xf5}};
template<typename Apply> static HRESULT changeManagedCategoriesWith(Apply apply,bool remove) {
    const GUID owned[]={GUID_TFCAT_TIP_KEYBOARD,ArionSystemTrayCategory};
    HRESULT result=S_OK;
    for (const auto &category:owned) {
        HRESULT hr=apply(category,remove);
        if (FAILED(hr) && SUCCEEDED(result)) result=hr;
        if (FAILED(hr) && !remove) break;
    }
    return result;
}
static HRESULT changeManagedCategories(bool remove) {
    ITfCategoryMgr *categories=nullptr;
    HRESULT hr=CoCreateInstance(CLSID_TF_CategoryMgr,nullptr,CLSCTX_INPROC_SERVER,IID_ITfCategoryMgr,reinterpret_cast<void**>(&categories));
    if (FAILED(hr)) return hr;
    hr=changeManagedCategoriesWith([&](REFGUID category,bool unregister) {
        return unregister ? categories->UnregisterCategory(ArionService,category,ArionService) : categories->RegisterCategory(ArionService,category,ArionService);
    },remove);
    categories->Release(); return hr;
}
template<typename Remove> static HRESULT disableOwnedProfilesWith(const OwnedProfiles &owned,Remove remove) {
    HRESULT result=S_OK;
    for (LANGID language:{ArionLanguage,ArionLegacyLanguage}) {
        if (language==ArionLanguage ? !owned.currentFound : !owned.legacyFound) continue;
        HRESULT hr=remove(language);
        if (FAILED(hr) && SUCCEEDED(result)) result=hr;
    }
    return result;
}
static HRESULT changeUserProfile(ITfInputProcessorProfileMgr *profiles,bool remove) {
    OwnedProfiles owned;
    HRESULT hr=observeOwnedProfiles(profiles,owned);
    if (FAILED(hr)) return hr;
    if (remove) return disableOwnedProfilesWith(owned,[](LANGID language) { return installUserProfile(true,language); });
    hr=currentProfileRequired(owned);
    return FAILED(hr) ? hr : installUserProfile();
}
template<typename Deactivate,typename Unregister> static HRESULT unregisterOwnedProfilesWith(const OwnedProfiles &owned,Deactivate deactivate,Unregister unregister) {
    for (LANGID language:{ArionLanguage,ArionLegacyLanguage}) {
        if (language==ArionLanguage ? !owned.currentFound : !owned.legacyFound) continue;
        HRESULT hr=deactivate(ArionService,language,ArionProfile);
        // S_FALSE means the exact profile was not active in this session.
        if (FAILED(hr)) return hr;
        hr=unregister(ArionService,language,ArionProfile,0);
        if (hr!=S_OK) return FAILED(hr) ? hr : HRESULT_FROM_WIN32(ERROR_NOT_READY);
    }
    return S_OK;
}
static HRESULT removeOwnedRegistration(ITfInputProcessorProfileMgr *profiles,const OwnedProfiles &owned) {
    HRESULT hr=unregisterOwnedProfilesWith(owned,
        [&](REFCLSID service,LANGID language,REFGUID profile) { return profiles->DeactivateProfile(TF_PROFILETYPE_INPUTPROCESSOR,language,service,profile,nullptr,TF_IPPMF_FORSESSION); },
        [&](REFCLSID service,LANGID language,REFGUID profile,DWORD flags) { return profiles->UnregisterProfile(service,language,profile,flags); });
    if (SUCCEEDED(hr)) hr=changeManagedCategories(true);
    if (SUCCEEDED(hr)) {
        LSTATUS result=RegDeleteTreeW(HKEY_LOCAL_MACHINE,classKey().c_str());
        if (result!=ERROR_SUCCESS && result!=ERROR_FILE_NOT_FOUND) hr=HRESULT_FROM_WIN32(result);
    }
    return hr;
}
template<typename Repair> static HRESULT repairExistingRegistrationWith(const std::wstring &existing,const std::wstring &dll,Repair repair) {
    if (_wcsicmp(existing.c_str(),dll.c_str())!=0) return HRESULT_FROM_WIN32(ERROR_REVISION_MISMATCH);
    return repair();
}
static HRESULT registrationPreflightWith(const OwnedProfiles &owned,const std::wstring &existing,const std::wstring &dll) {
    if (owned.upgradeRequired(!existing.empty())) return HRESULT_FROM_WIN32(ERROR_REVISION_MISMATCH);
    // Same-build repair is the only allowed operation over a registered DLL.
    return existing.empty() ? S_OK : repairExistingRegistrationWith(existing,dll,[] { return S_OK; });
}
static HRESULT registrationPreflight(ITfInputProcessorProfileMgr *profiles,const std::wstring &dll) {
    std::wstring existing; bool loader=false;
    HRESULT registry=resolveRegistration(existing,loader);
    if (FAILED(registry)) return registry;
    OwnedProfiles owned;
    HRESULT hr=observeOwnedProfiles(profiles,owned);
    return FAILED(hr) ? hr : registrationPreflightWith(owned,existing,dll);
}

static HRESULT upgradePreflightWith(const OwnedProfiles &owned,const std::wstring &existing,const std::wstring &expected,bool selected,bool loader=false) {
    if (!owned.registered() || existing.empty() || _wcsicmp(existing.c_str(),expected.c_str())!=0) return HRESULT_FROM_WIN32(ERROR_REVISION_MISMATCH);
    return selected && !loader ? HRESULT_FROM_WIN32(ERROR_BUSY) : S_OK;
}

// Every callback is exercised without COM registration by upgrade_test.cpp.
// Rollback is required even when publish fails after a partial registry write.
template<typename Publish,typename Register,typename Categories,typename Retire,typename Rollback>
static HRESULT upgradeRegistrationWith(Publish publish,Register current,Categories categories,Retire retire,Rollback rollback) {
    HRESULT hr=publish();
    if (hr==S_OK) hr=current();
    if (hr==S_OK) hr=categories();
    if (hr==S_OK) hr=retire();
    if (hr==S_OK) return S_OK;
    if (SUCCEEDED(hr)) hr=HRESULT_FROM_WIN32(ERROR_NOT_READY);
    HRESULT restored=rollback();
    if (FAILED(restored)) {
        std::cerr<<"TSF upgrade rollback failed: 0x"<<std::hex<<static_cast<unsigned long>(restored)<<"; original failure: 0x"<<static_cast<unsigned long>(hr)<<"\n";
        return restored;
    }
    return hr;
}
template<typename Publish,typename Register,typename Categories,typename Retire,typename Rollback>
static HRESULT upgradeRegistrationModeWith(bool stable,Publish publish,Register current,Categories categories,Retire retire,Rollback rollback) {
    return upgradeRegistrationWith(publish,[&] { return stable?S_OK:current(); },[&] { return stable?S_OK:categories(); },[&] { return stable?S_OK:retire(); },rollback);
}
template<typename Register> static HRESULT registerOwnedProfileWith(Register apply,const std::wstring &dll) {
    return apply(ArionService,ArionLanguage,ArionProfile,ArionName,static_cast<ULONG>(wcslen(ArionName)),dll.c_str(),static_cast<ULONG>(dll.size()),0,nullptr,0,TRUE,0);
}
template<typename Enable> static HRESULT enableOwnedByDefaultWith(Enable apply) {
    return apply(ArionService,ArionLanguage,ArionProfile,TRUE);
}
static HRESULT repairProfile(ITfInputProcessorProfileMgr *profiles) {
    OwnedProfiles owned;
    HRESULT hr=observeOwnedProfiles(profiles,owned);
    if (FAILED(hr)) return hr;
    hr=currentProfileRequired(owned);
    if (FAILED(hr)) return hr;
    hr=changeManagedCategories(false);
    if (FAILED(hr)) return hr;
    ITfInputProcessorProfiles *legacy=nullptr;
    hr=profiles->QueryInterface(IID_ITfInputProcessorProfiles,reinterpret_cast<void**>(&legacy));
    if (FAILED(hr)) return hr;
    hr=enableOwnedByDefaultWith([&](auto&&... arguments) { return legacy->EnableLanguageProfileByDefault(arguments...); });
    legacy->Release(); return hr;
}
static bool profileEnabled(const TF_INPUTPROCESSORPROFILE &profile,BOOL userEnabled) {
    return userEnabled && (profile.dwFlags&TF_IPP_FLAG_ENABLED)!=0;
}
static bool ownedProfile(const TF_INPUTPROCESSORPROFILE &profile) {
    return ownedLanguageProfile(profile,ArionLanguage);
}
template<typename Observe> static HRESULT confirmSelectionWith(HRESULT requested,Observe observe) {
    if (requested!=S_OK) return FAILED(requested) ? requested : HRESULT_FROM_WIN32(ERROR_NOT_READY);
    TF_INPUTPROCESSORPROFILE active{};
    HRESULT hr=observe(active);
    if (hr!=S_OK) return FAILED(hr) ? hr : HRESULT_FROM_WIN32(ERROR_NOT_READY);
    return ownedProfile(active) ? S_OK : HRESULT_FROM_WIN32(ERROR_RETRY);
}
static HRESULT selectProfile(ITfInputProcessorProfileMgr *profiles) {
    OwnedProfiles owned;
    HRESULT observed=observeOwnedProfiles(profiles,owned);
    if (FAILED(observed)) return observed;
    observed=currentProfileRequired(owned);
    if (FAILED(observed)) return observed;
    HRESULT requested=profiles->ActivateProfile(TF_PROFILETYPE_INPUTPROCESSOR,ArionLanguage,ArionService,ArionProfile,nullptr,TF_IPPMF_FORSESSION|TF_IPPMF_DONTCARECURRENTINPUTLANGUAGE);
    HRESULT result=confirmSelectionWith(requested,[&](TF_INPUTPROCESSORPROFILE &active) { return profiles->GetActiveProfile(GUID_TFCAT_TIP_KEYBOARD,&active); });
    if (result!=S_OK) std::cerr<<"TSF selection request HRESULT=0x"<<std::hex<<static_cast<unsigned long>(requested)<<"; confirmed selection HRESULT=0x"<<static_cast<unsigned long>(result)<<"\n";
    return result;
}
static HRESULT registration(ITfInputProcessorProfileMgr *profiles, const std::wstring &dll) {
    HRESULT observed=registrationPreflight(profiles,dll);
    if (FAILED(observed)) return observed;
    HRESULT registry=S_OK;
    std::wstring existing; bool loader=false;
    registry=resolveRegistration(existing,loader);
    if (FAILED(registry)) return registry;
    // Existing in-process services can remain loaded in arbitrary editors.
    // Refuse replacement until a host-handover transaction is implemented.
    if (!existing.empty()) return loader?repairExistingRegistrationWith(existing,dll,[&] { return repairProfile(profiles); }):HRESULT_FROM_WIN32(ERROR_REVISION_MISMATCH);
    std::wstring root; HRESULT hr=arionLoader::machineRoot(root);
    if (FAILED(hr)) return hr;
    hr = setValue(classKey(), nullptr, ArionName);
    if (SUCCEEDED(hr)) hr = setValue(classKey() + L"\\InprocServer32", L"CorePath", dll);
    if (SUCCEEDED(hr)) hr = setValue(classKey() + L"\\InprocServer32", L"ThreadingModel", L"Apartment");
    if (SUCCEEDED(hr)) hr = setValue(classKey() + L"\\InprocServer32", nullptr, root+L"\\ArionTSFLoader.dll");
    if (FAILED(hr)) { RegDeleteTreeW(HKEY_LOCAL_MACHINE,classKey().c_str()); return hr; }
    hr=registerOwnedProfileWith([&](auto&&... arguments) { return profiles->RegisterProfile(arguments...); },dll);
    if (FAILED(hr)) { RegDeleteTreeW(HKEY_LOCAL_MACHINE, classKey().c_str()); return hr; }
    hr=changeManagedCategories(false);
    if (FAILED(hr)) { changeManagedCategories(true); profiles->UnregisterProfile(ArionService, ArionLanguage, ArionProfile, 0); RegDeleteTreeW(HKEY_LOCAL_MACHINE, classKey().c_str()); }
    return hr;
}
static HRESULT machineRoot(std::wstring &root) {
    wchar_t *folder = nullptr;
    HRESULT hr = SHGetKnownFolderPath(FOLDERID_ProgramFiles, KF_FLAG_DEFAULT, nullptr, &folder);
    if (FAILED(hr)) return hr;
    root = std::wstring(folder) + L"\\Arion\\InputMethod"; CoTaskMemFree(folder); return S_OK;
}
static HRESULT protectedDirectory(const std::wstring &path) {
    // Only Administrators and SYSTEM can replace code loaded into other apps.
    PSECURITY_DESCRIPTOR descriptor = nullptr;
    if (!ConvertStringSecurityDescriptorToSecurityDescriptorW(L"O:BAG:BAD:P(A;OICI;FA;;;SY)(A;OICI;FA;;;BA)(A;OICI;GRGX;;;BU)", SDDL_REVISION_1, &descriptor, nullptr)) return HRESULT_FROM_WIN32(GetLastError());
    SECURITY_ATTRIBUTES attributes{sizeof(SECURITY_ATTRIBUTES), descriptor, FALSE};
    bool created = CreateDirectoryW(path.c_str(), &attributes) != FALSE;
    DWORD error = GetLastError();
    LocalFree(descriptor);
    if (!created && error != ERROR_ALREADY_EXISTS) return HRESULT_FROM_WIN32(error);
    DWORD flags = GetFileAttributesW(path.c_str());
    if (flags == INVALID_FILE_ATTRIBUTES || !(flags & FILE_ATTRIBUTE_DIRECTORY) || (flags & FILE_ATTRIBUTE_REPARSE_POINT)) return E_ACCESSDENIED;
    // Refuse existing writable destinations rather than trusting their ACL.
    PACL dacl = nullptr; PSID owner = nullptr; PSECURITY_DESCRIPTOR security = nullptr;
    DWORD status = GetNamedSecurityInfoW(const_cast<wchar_t*>(path.c_str()), SE_FILE_OBJECT, OWNER_SECURITY_INFORMATION | DACL_SECURITY_INFORMATION, &owner, nullptr, &dacl, nullptr, &security);
    if (status != ERROR_SUCCESS || !dacl) { if (security) LocalFree(security); return E_ACCESSDENIED; }
    BYTE admin[SECURITY_MAX_SID_SIZE], system[SECURITY_MAX_SID_SIZE]; DWORD adminSize = sizeof(admin), systemSize = sizeof(system);
    CreateWellKnownSid(WinBuiltinAdministratorsSid, nullptr, admin, &adminSize); CreateWellKnownSid(WinLocalSystemSid, nullptr, system, &systemSize);
    bool safe = owner && (EqualSid(owner, admin) || EqualSid(owner, system));
    for (DWORD i = 0; i < dacl->AceCount; ++i) {
        void *raw = nullptr; if (!GetAce(dacl, i, &raw)) { safe = false; break; }
        auto ace = static_cast<ACCESS_ALLOWED_ACE*>(raw);
        if (ace->Header.AceType == ACCESS_DENIED_ACE_TYPE) continue;
        if (ace->Header.AceType != ACCESS_ALLOWED_ACE_TYPE) { safe = false; break; }
        if ((ace->Mask & (FILE_WRITE_DATA | FILE_APPEND_DATA | FILE_DELETE_CHILD | FILE_WRITE_EA | FILE_WRITE_ATTRIBUTES | DELETE | WRITE_DAC | WRITE_OWNER | GENERIC_WRITE | GENERIC_ALL)) && !EqualSid(&ace->SidStart, admin) && !EqualSid(&ace->SidStart, system)) { safe = false; break; }
    }
    LocalFree(security); return safe ? S_OK : E_ACCESSDENIED;
}
struct MachineMaintenance {
    HANDLE handle=INVALID_HANDLE_VALUE;
    ~MachineMaintenance() { if (handle!=INVALID_HANDLE_VALUE) CloseHandle(handle); }
    HRESULT acquire(const std::wstring &root) {
        HRESULT hr=protectedDirectory(root.substr(0,root.find_last_of(L'\\')));
        if (SUCCEEDED(hr)) hr=protectedDirectory(root);
        if (FAILED(hr)) return hr;
        handle=CreateFileW((root+L"\\.maintenance.lock").c_str(),GENERIC_READ|GENERIC_WRITE,0,nullptr,OPEN_ALWAYS,FILE_FLAG_OPEN_REPARSE_POINT,nullptr);
        if (handle==INVALID_HANDLE_VALUE) return HRESULT_FROM_WIN32(GetLastError());
        BY_HANDLE_FILE_INFORMATION info{};
        return GetFileInformationByHandle(handle,&info) && !(info.dwFileAttributes&(FILE_ATTRIBUTE_DIRECTORY|FILE_ATTRIBUTE_REPARSE_POINT)) && info.nNumberOfLinks==1 ? S_OK : E_ACCESSDENIED;
    }
};
static HRESULT readUpgradeIntent(const std::wstring &path,std::wstring &target,std::wstring &previous) {
    HANDLE file=CreateFileW(path.c_str(),GENERIC_READ,FILE_SHARE_READ|FILE_SHARE_WRITE|FILE_SHARE_DELETE,nullptr,OPEN_EXISTING,FILE_FLAG_OPEN_REPARSE_POINT,nullptr);
    if (file==INVALID_HANDLE_VALUE) {
        DWORD error=GetLastError();
        return error==ERROR_FILE_NOT_FOUND || error==ERROR_PATH_NOT_FOUND ? S_FALSE : HRESULT_FROM_WIN32(error);
    }
    BY_HANDLE_FILE_INFORMATION info{}; char text[130]{}; DWORD count=0;
    bool valid=GetFileInformationByHandle(file,&info) && !(info.dwFileAttributes&(FILE_ATTRIBUTE_DIRECTORY|FILE_ATTRIBUTE_REPARSE_POINT)) && info.nNumberOfLinks==1 && !info.nFileSizeHigh && info.nFileSizeLow==sizeof(text) && ReadFile(file,text,sizeof(text),&count,nullptr) && count==sizeof(text);
    CloseHandle(file);
    if (!valid || text[64]!='\n' || text[129]!='\n') return TRUST_E_BAD_DIGEST;
    std::wstring next(text,text+64),old(text+65,text+129);
    if (!hashName(next) || !hashName(old)) return TRUST_E_BAD_DIGEST;
    target=std::move(next); previous=std::move(old); return S_OK;
}
static HRESULT writeUpgradeIntent(const std::wstring &path,const std::wstring &target,const std::wstring &previous) {
    if (!hashName(target) || !hashName(previous)) return E_INVALIDARG;
    const auto temporary=path+L".next";
    HANDLE file=CreateFileW(temporary.c_str(),GENERIC_WRITE,0,nullptr,OPEN_ALWAYS,FILE_FLAG_OPEN_REPARSE_POINT,nullptr);
    if (file==INVALID_HANDLE_VALUE) return HRESULT_FROM_WIN32(GetLastError());
    BY_HANDLE_FILE_INFORMATION info{};
    std::string text(target.begin(),target.end()); text+='\n'; text.append(previous.begin(),previous.end()); text+='\n'; DWORD count=0;
    LARGE_INTEGER start{};
    bool valid=GetFileInformationByHandle(file,&info) && !(info.dwFileAttributes&(FILE_ATTRIBUTE_DIRECTORY|FILE_ATTRIBUTE_REPARSE_POINT)) && info.nNumberOfLinks==1 && SetFilePointerEx(file,start,nullptr,FILE_BEGIN) && SetEndOfFile(file) && WriteFile(file,text.data(),static_cast<DWORD>(text.size()),&count,nullptr) && count==text.size() && FlushFileBuffers(file);
    CloseHandle(file);
    if (!valid) return E_ACCESSDENIED;
    return MoveFileExW(temporary.c_str(),path.c_str(),MOVEFILE_REPLACE_EXISTING|MOVEFILE_WRITE_THROUGH) ? S_OK : HRESULT_FROM_WIN32(GetLastError());
}
static HRESULT clearUpgradeIntent(const std::wstring &path) {
    if (DeleteFileW(path.c_str())) return S_OK;
    DWORD error=GetLastError(); return error==ERROR_FILE_NOT_FOUND || error==ERROR_PATH_NOT_FOUND ? S_OK : HRESULT_FROM_WIN32(error);
}
static std::wstring keyboardManifestHash() {
    const std::string hash=ARION_KEYBOARD_MANIFEST_SHA256;
    return std::wstring(hash.begin(),hash.end());
}
static HRESULT copyProtectedFile(const std::wstring &source,const std::wstring &destination,const std::wstring &hash) {
    DWORD flags=GetFileAttributesW(destination.c_str());
    if (flags!=INVALID_FILE_ATTRIBUTES && (flags&(FILE_ATTRIBUTE_REPARSE_POINT|FILE_ATTRIBUTE_DIRECTORY))) return E_ACCESSDENIED;
    if (flags==INVALID_FILE_ATTRIBUTES && !CopyFileW(source.c_str(),destination.c_str(),TRUE)) return HRESULT_FROM_WIN32(GetLastError());
    // CopyFile can preserve source security. Apply the protected payload ACL
    // explicitly before any host can load the copied code.
    PSECURITY_DESCRIPTOR descriptor=nullptr;
    if (!ConvertStringSecurityDescriptorToSecurityDescriptorW(L"O:BAG:BAD:P(A;;FA;;;SY)(A;;FA;;;BA)(A;;GRGX;;;BU)",SDDL_REVISION_1,&descriptor,nullptr)) return HRESULT_FROM_WIN32(GetLastError());
    BOOL secured=SetFileSecurityW(destination.c_str(),OWNER_SECURITY_INFORMATION|GROUP_SECURITY_INFORMATION|DACL_SECURITY_INFORMATION|PROTECTED_DACL_SECURITY_INFORMATION,descriptor);
    DWORD error=GetLastError(); LocalFree(descriptor);
    if (!secured) return HRESULT_FROM_WIN32(error);
    return fileHash(destination)==hash ? S_OK : TRUST_E_BAD_DIGEST;
}
static HRESULT installKeyboardPayload(const std::wstring &source,const std::wstring &destination,const std::wstring &manifestHash) {
    std::vector<KeyboardPayloadFile> files;
    HRESULT hr=verifyKeyboardPayload(source,manifestHash,files);
    if (FAILED(hr)) return hr;
    hr=protectedDirectory(destination);
    if (FAILED(hr)) return hr;
    for (const auto &file:files) {
        size_t position=0;
        while ((position=file.relative.find(L'/',position))!=std::wstring::npos) {
            hr=protectedDirectory(keyboardNativePath(destination,file.relative.substr(0,position)));
            if (FAILED(hr)) return hr;
            ++position;
        }
        hr=copyProtectedFile(keyboardNativePath(source,file.relative),keyboardNativePath(destination,file.relative),file.sha256);
        if (FAILED(hr)) return hr;
    }
    hr=copyProtectedFile(source+L"\\runtime.json",destination+L"\\runtime.json",manifestHash);
    if (FAILED(hr)) return hr;
    return verifyKeyboardPayload(destination,manifestHash,files);
}
static HRESULT removeKeyboardPayload(const std::wstring &installed,const std::vector<KeyboardPayloadFile> &files,std::vector<HANDLE> &handles) {
    const auto version=installed.substr(0,installed.find_last_of(L'\\'));
    const auto root=version+L"\\keyboard";
    std::set<std::wstring> directories;
    for (const auto &file:files) {
        if (!DeleteFileW(keyboardNativePath(root,file.relative).c_str())) return HRESULT_FROM_WIN32(GetLastError());
        size_t position=0;
        while ((position=file.relative.find(L'/',position))!=std::wstring::npos) {
            directories.insert(keyboardNativePath(root,file.relative.substr(0,position)));
            ++position;
        }
    }
    if (!files.empty() && !DeleteFileW((root+L"\\runtime.json").c_str())) return HRESULT_FROM_WIN32(GetLastError());
    if (!DeleteFileW(installed.c_str())) return HRESULT_FROM_WIN32(GetLastError());
    for (HANDLE handle:handles) CloseHandle(handle);
    handles.clear();
    for (auto it=directories.rbegin();it!=directories.rend();++it) {
        if (!RemoveDirectoryW(it->c_str())) return HRESULT_FROM_WIN32(GetLastError());
    }
    if (!files.empty() && !RemoveDirectoryW(root.c_str())) return HRESULT_FROM_WIN32(GetLastError());
    return RemoveDirectoryW(version.c_str()) ? S_OK : HRESULT_FROM_WIN32(GetLastError());
}
struct PayloadRemovalGuard {
    std::vector<HANDLE> handles;
    std::vector<KeyboardPayloadFile> files;
    ~PayloadRemovalGuard() { for (HANDLE handle:handles) CloseHandle(handle); }
    HRESULT reserve(const std::wstring &path) {
        // DELETE alone can succeed for a mapped image even though deletion
        // later fails. Write access rejects existing image sections and pins
        // the file against new image mappings until unregister completes.
        HANDLE handle=CreateFileW(path.c_str(),GENERIC_WRITE|DELETE|FILE_READ_ATTRIBUTES,FILE_SHARE_READ|FILE_SHARE_DELETE,nullptr,OPEN_EXISTING,FILE_FLAG_OPEN_REPARSE_POINT,nullptr);
        if (handle==INVALID_HANDLE_VALUE) return HRESULT_FROM_WIN32(GetLastError());
        BY_HANDLE_FILE_INFORMATION info{};
        if (!GetFileInformationByHandle(handle,&info) || (info.dwFileAttributes&(FILE_ATTRIBUTE_DIRECTORY|FILE_ATTRIBUTE_REPARSE_POINT))) { CloseHandle(handle); return E_ACCESSDENIED; }
        handles.push_back(handle); return S_OK;
    }
    HRESULT prepare(const std::wstring &installed) {
        const auto version=installed.substr(0,installed.find_last_of(L'\\'));
        const auto root=version+L"\\keyboard";
        // Keep delete access reserved until unregister/cleanup finishes. A
        // loaded module must fail here, while registration is still intact.
        DWORD flags=GetFileAttributesW(root.c_str());
        if (flags==INVALID_FILE_ATTRIBUTES) {
            DWORD error=GetLastError();
            return error==ERROR_FILE_NOT_FOUND || error==ERROR_PATH_NOT_FOUND ? reserve(installed) : HRESULT_FROM_WIN32(error);
        }
        HRESULT hr=verifyKeyboardPayload(root,fileHash(root+L"\\runtime.json"),files);
        if (FAILED(hr)) return hr;
        hr=reserve(installed);
        if (FAILED(hr)) return hr;
        for (const auto &file:files) {
            hr=reserve(keyboardNativePath(root,file.relative));
            if (FAILED(hr)) return hr;
        }
        return reserve(root+L"\\runtime.json");
    }
};
static HRESULT verifySourcePayload(const std::wstring &source,const std::wstring &expected,std::wstring &keyboardSource) {
    if (!hashName(expected) || fileHash(source) != expected) return TRUST_E_BAD_DIGEST;
    const auto sourceDirectory=source.substr(0,source.find_last_of(L"\\/"));
    const auto expectedLoader=loaderHash();
    if (!hashName(expectedLoader) || fileHash(sourceDirectory+L"\\ArionTSFLoader.dll")!=expectedLoader) return TRUST_E_BAD_DIGEST;
    keyboardSource=sourceDirectory.substr(0,sourceDirectory.find_last_of(L"\\/"))+L"\\keyboard";
    std::vector<KeyboardPayloadFile> keyboardFiles;
    return verifyKeyboardPayload(keyboardSource,keyboardManifestHash(),keyboardFiles);
}
static HRESULT install(ITfInputProcessorProfileMgr *profiles, const std::wstring &source, const std::wstring &expected) {
    std::wstring keyboardSource;
    HRESULT checked=verifySourcePayload(source,expected,keyboardSource);
    if (FAILED(checked)) return checked;
    std::wstring root; HRESULT hr = machineRoot(root); if (FAILED(hr)) return hr;
    // Observe migration/build compatibility before creating even the first
    // protected directory or copying code. Registration repeats the check.
    hr=registrationPreflight(profiles,root+L"\\"+expected+L"\\ArionTSF.dll");
    if (FAILED(hr)) return hr;
    MachineMaintenance maintenance; hr=maintenance.acquire(root); if (FAILED(hr)) return hr;
    hr=registrationPreflight(profiles,root+L"\\"+expected+L"\\ArionTSF.dll");
    if (FAILED(hr)) return hr;
    hr = protectedDirectory(root.substr(0, root.find_last_of(L'\\')));
    if (SUCCEEDED(hr)) hr = protectedDirectory(root);
    auto version = root + L"\\" + expected;
    if (SUCCEEDED(hr)) hr = protectedDirectory(version);
    if (FAILED(hr)) return hr;
    hr=copyProtectedFile(source.substr(0,source.find_last_of(L"\\/"))+L"\\ArionTSFLoader.dll",root+L"\\ArionTSFLoader.dll",loaderHash());
    if (FAILED(hr)) return hr;
    auto destination = version + L"\\ArionTSF.dll";
    hr=installKeyboardPayload(keyboardSource,version+L"\\keyboard",keyboardManifestHash());
    if (SUCCEEDED(hr)) hr=copyProtectedFile(source,destination,expected);
    if (FAILED(hr)) return hr;
    hr=registration(profiles, destination);
    // A new explicit install also completes stale maintenance metadata from
    // a previously removed installation, after its new registration is valid.
    if (hr==S_OK) hr=clearUpgradeIntent(root+L"\\.upgrade-pending");
    return hr;
}
static HRESULT ownedPayload(std::wstring &path, std::wstring &build,bool *loaderReady=nullptr) {
    bool loader=false; HRESULT hr=resolveRegistration(path,loader);
    if (hr!=S_OK) return hr;
    if (loaderReady) *loaderReady=loader;
    std::wstring root; hr=machineRoot(root); if (FAILED(hr)) return hr;
    return arionLoader::corePath(root,path,build)?S_OK:E_ACCESSDENIED;
}
struct UpgradeSnapshot {
    OwnedProfiles profiles;
    std::wstring path,corePath,name,threading,currentDescription,legacyDescription;
    bool corePathPresent=false,loader=false;
    bool keyboardCategory=false,trayCategory=false;
};
static HRESULT readClassValue(const std::wstring &key,const wchar_t *name,std::wstring &value) {
    wchar_t text[32768]{}; DWORD size=sizeof(text);
    LSTATUS result=RegGetValueW(HKEY_LOCAL_MACHINE,key.c_str(),name,RRF_RT_REG_SZ,nullptr,text,&size);
    if (result!=ERROR_SUCCESS) return HRESULT_FROM_WIN32(result);
    value=text; return S_OK;
}
static HRESULT captureUpgrade(ITfInputProcessorProfileMgr *profiles,UpgradeSnapshot &snapshot) {
    HRESULT hr=observeOwnedProfiles(profiles,snapshot.profiles);
    if (SUCCEEDED(hr)) { snapshot.path=registeredPath(&hr); if (hr==S_FALSE) hr=HRESULT_FROM_WIN32(ERROR_NOT_FOUND); }
    if (SUCCEEDED(hr)) { std::wstring root; hr=machineRoot(root); if (SUCCEEDED(hr)) snapshot.loader=arionLoader::loaderPath(root,snapshot.path); }
    if (SUCCEEDED(hr)) {
        hr=arionLoader::readValue(L"CorePath",snapshot.corePath);
        snapshot.corePathPresent=hr==S_OK;
        if (hr==S_FALSE) hr=S_OK;
    }
    if (SUCCEEDED(hr)) hr=readClassValue(classKey(),nullptr,snapshot.name);
    if (SUCCEEDED(hr)) hr=readClassValue(classKey()+L"\\InprocServer32",L"ThreadingModel",snapshot.threading);
    if (FAILED(hr)) return hr;
    if (_wcsicmp(snapshot.threading.c_str(),L"Apartment")!=0) return E_INVALIDARG;
    ITfInputProcessorProfiles *legacy=nullptr;
    hr=profiles->QueryInterface(IID_ITfInputProcessorProfiles,reinterpret_cast<void**>(&legacy));
    if (FAILED(hr)) return hr;
    for (LANGID language:{ArionLanguage,ArionLegacyLanguage}) {
        if (language==ArionLanguage ? !snapshot.profiles.currentFound : !snapshot.profiles.legacyFound) continue;
        BSTR description=nullptr;
        hr=legacy->GetLanguageProfileDescription(ArionService,language,ArionProfile,&description);
        if (hr==S_OK && description) (language==ArionLanguage?snapshot.currentDescription:snapshot.legacyDescription).assign(description,SysStringLen(description));
        else if (SUCCEEDED(hr)) hr=E_UNEXPECTED;
        SysFreeString(description);
        if (FAILED(hr)) break;
    }
    legacy->Release();
    if (FAILED(hr)) return hr;
    ITfCategoryMgr *categories=nullptr; IEnumGUID *all=nullptr;
    hr=CoCreateInstance(CLSID_TF_CategoryMgr,nullptr,CLSCTX_INPROC_SERVER,IID_ITfCategoryMgr,reinterpret_cast<void**>(&categories));
    if (FAILED(hr)) return hr;
    hr=categories->EnumCategoriesInItem(ArionService,&all);
    if (hr==S_OK && all) {
        GUID value{}; ULONG fetched=0;
        while ((hr=all->Next(1,&value,&fetched))==S_OK) {
            if (fetched!=1) { hr=E_UNEXPECTED; break; }
            snapshot.keyboardCategory=snapshot.keyboardCategory || value==GUID_TFCAT_TIP_KEYBOARD;
            snapshot.trayCategory=snapshot.trayCategory || value==ArionSystemTrayCategory;
        }
        if (hr==S_FALSE) hr=S_OK;
        all->Release();
    } else if (SUCCEEDED(hr)) hr=E_UNEXPECTED;
    categories->Release(); return hr;
}
static HRESULT upgradeIdle(ITfInputProcessorProfileMgr *profiles,const std::wstring &expected) {
    OwnedProfiles owned; HRESULT hr=observeOwnedProfiles(profiles,owned);
    if (FAILED(hr)) return hr;
    std::wstring existing; bool loader=false;
    HRESULT registry=resolveRegistration(existing,loader);
    if (FAILED(registry)) return registry;
    TF_INPUTPROCESSORPROFILE active{};
    hr=profiles->GetActiveProfile(GUID_TFCAT_TIP_KEYBOARD,&active);
    if (hr!=S_OK && hr!=S_FALSE) return FAILED(hr)?hr:E_UNEXPECTED;
    const bool selected=hr==S_OK && ownedActiveProfile(owned,active);
    return upgradePreflightWith(owned,existing,expected,selected,loader);
}
template<typename Set,typename Erase> static HRESULT restoreCorePathWith(const UpgradeSnapshot &snapshot,Set set,Erase erase) {
    return snapshot.corePathPresent?set(snapshot.corePath):erase();
}
static const std::wstring &snapshotIcon(const UpgradeSnapshot &snapshot) { return snapshot.loader?snapshot.corePath:snapshot.path; }
static HRESULT restoreUpgrade(ITfInputProcessorProfileMgr *profiles,const UpgradeSnapshot &snapshot) {
    HRESULT result=S_OK;
    auto remember=[&](HRESULT hr) { if (FAILED(hr) && SUCCEEDED(result)) result=hr; };
    remember(setValue(classKey(),nullptr,snapshot.name));
    remember(setValue(classKey()+L"\\InprocServer32",nullptr,snapshot.path));
    remember(setValue(classKey()+L"\\InprocServer32",L"ThreadingModel",snapshot.threading));
    remember(restoreCorePathWith(snapshot,[&](const std::wstring &value) { return setValue(classKey()+L"\\InprocServer32",L"CorePath",value); },[] {
        HKEY key=nullptr;
        LSTATUS status=RegOpenKeyExW(HKEY_LOCAL_MACHINE,(classKey()+L"\\InprocServer32").c_str(),0,KEY_SET_VALUE|KEY_WOW64_64KEY,&key);
        if (status!=ERROR_SUCCESS) return HRESULT_FROM_WIN32(status);
        status=RegDeleteValueW(key,L"CorePath"); RegCloseKey(key);
        return status==ERROR_FILE_NOT_FOUND?S_OK:HRESULT_FROM_WIN32(status);
    }));
    for (LANGID language:{ArionLanguage,ArionLegacyLanguage}) {
        const bool existed=language==ArionLanguage?snapshot.profiles.currentFound:snapshot.profiles.legacyFound;
        if (!existed) {
            if (language==ArionLanguage) remember(profiles->UnregisterProfile(ArionService,language,ArionProfile,0));
            continue;
        }
        const auto &description=language==ArionLanguage?snapshot.currentDescription:snapshot.legacyDescription;
        const auto &profile=language==ArionLanguage?snapshot.profiles.current:snapshot.profiles.legacy;
        const auto &icon=snapshotIcon(snapshot);
        remember(profiles->RegisterProfile(ArionService,language,ArionProfile,description.c_str(),static_cast<ULONG>(description.size()),icon.c_str(),static_cast<ULONG>(icon.size()),0,profile.hklSubstitute,0,(profile.dwFlags&TF_IPP_FLAG_ENABLED)!=0,0));
    }
    ITfCategoryMgr *categories=nullptr;
    HRESULT hr=CoCreateInstance(CLSID_TF_CategoryMgr,nullptr,CLSCTX_INPROC_SERVER,IID_ITfCategoryMgr,reinterpret_cast<void**>(&categories));
    remember(hr);
    if (SUCCEEDED(hr)) {
        for (const auto &category:{GUID_TFCAT_TIP_KEYBOARD,ArionSystemTrayCategory}) {
            const bool existed=category==GUID_TFCAT_TIP_KEYBOARD?snapshot.keyboardCategory:snapshot.trayCategory;
            remember(existed?categories->RegisterCategory(ArionService,category,ArionService):categories->UnregisterCategory(ArionService,category,ArionService));
        }
        categories->Release();
    }
    return result;
}
static HRESULT upgrade(ITfInputProcessorProfileMgr *profiles,const std::wstring &source,const std::wstring &expected,const std::wstring &previous) {
    std::wstring keyboardSource;
    HRESULT hr=verifySourcePayload(source,expected,keyboardSource);
    if (FAILED(hr) || !hashName(previous)) return FAILED(hr)?hr:E_INVALIDARG;
    std::wstring root; hr=machineRoot(root); if (FAILED(hr)) return hr;
    MachineMaintenance maintenance; hr=maintenance.acquire(root); if (FAILED(hr)) return hr;
    UpgradeSnapshot snapshot; std::wstring observed,previousCore;
    hr=ownedPayload(previousCore,observed);
    if (hr!=S_OK) return FAILED(hr)?hr:HRESULT_FROM_WIN32(ERROR_NOT_FOUND);
    if (observed!=previous) return HRESULT_FROM_WIN32(ERROR_REVISION_MISMATCH);
    hr=upgradeIdle(profiles,previousCore);
    if (SUCCEEDED(hr)) hr=captureUpgrade(profiles,snapshot);
    if (FAILED(hr)) return hr;
    const auto version=root+L"\\"+expected,destination=version+L"\\ArionTSF.dll";
    hr=protectedDirectory(version);
    if (SUCCEEDED(hr)) hr=copyProtectedFile(source.substr(0,source.find_last_of(L"\\/"))+L"\\ArionTSFLoader.dll",root+L"\\ArionTSFLoader.dll",loaderHash());
    if (SUCCEEDED(hr)) hr=installKeyboardPayload(keyboardSource,version+L"\\keyboard",keyboardManifestHash());
    if (SUCCEEDED(hr)) hr=copyProtectedFile(source,destination,expected);
    if (FAILED(hr)) return hr;
    // Recheck after staging. Never overwrite or delete the old closure: an
    // arbitrary editor may retain its COM factory and native dependencies.
    hr=upgradeIdle(profiles,previousCore);
    if (FAILED(hr)) return hr;
    const auto intent=root+L"\\.upgrade-pending";
    hr=writeUpgradeIntent(intent,expected,previous);
    if (FAILED(hr)) return hr;
    bool restored=false;
    const bool stable=snapshot.loader && snapshot.profiles.currentFound && !snapshot.profiles.legacyFound;
    hr=upgradeRegistrationModeWith(stable,[&] {
        if (stable) return setValue(classKey()+L"\\InprocServer32",L"CorePath",destination);
        HRESULT publish=setValue(classKey(),nullptr,ArionName);
        if (SUCCEEDED(publish)) publish=setValue(classKey()+L"\\InprocServer32",L"ThreadingModel",L"Apartment");
        // A valid CorePath is installed before migrating the COM default. Once
        // the stable default exists, updates change only this single value.
        if (SUCCEEDED(publish)) publish=setValue(classKey()+L"\\InprocServer32",L"CorePath",destination);
        if (SUCCEEDED(publish)) publish=setValue(classKey()+L"\\InprocServer32",nullptr,root+L"\\ArionTSFLoader.dll");
        return publish;
    },[&] { return registerOwnedProfileWith([&](auto&&... arguments) { return profiles->RegisterProfile(arguments...); },destination); },
    [] { return changeManagedCategories(false); },[&] {
        return snapshot.profiles.legacyFound?profiles->UnregisterProfile(ArionService,ArionLegacyLanguage,ArionProfile,0):S_OK;
    },[&] {
        HRESULT result=stable?setValue(classKey()+L"\\InprocServer32",L"CorePath",snapshot.corePath):restoreUpgrade(profiles,snapshot);
        restored=SUCCEEDED(result); return result;
    });
    if (hr==S_OK || restored) {
        HRESULT cleared=clearUpgradeIntent(intent);
        if (FAILED(cleared)) return cleared;
    }
    return hr;
}
int wmain(int count, wchar_t **args) {
    if (count < 2) return 2;
    const std::wstring operation=args[1];
    if (operation==L"verify") {
        if (count!=4) return 2;
        std::wstring keyboardSource;
        HRESULT checked=verifySourcePayload(args[2],args[3],keyboardSource);
        if (FAILED(checked)) std::cerr<<"TSF payload verification failed: 0x"<<std::hex<<static_cast<unsigned long>(checked)<<"\n";
        return SUCCEEDED(checked) ? 0 : 1;
    }
    HRESULT hr = CoInitializeEx(nullptr, COINIT_APARTMENTTHREADED);
    if (FAILED(hr)) return 3;
    ITfInputProcessorProfileMgr *profiles = nullptr;
    hr = CoCreateInstance(CLSID_TF_InputProcessorProfiles, nullptr, CLSCTX_INPROC_SERVER, IID_ITfInputProcessorProfileMgr, reinterpret_cast<void**>(&profiles));
    if (FAILED(hr)) { CoUninitialize(); return 4; }
    MachineMaintenance maintenance;
    if ((operation==L"remove" || operation==L"repair") && !registeredPath().empty()) {
        std::wstring root; hr=machineRoot(root);
        if (SUCCEEDED(hr)) hr=maintenance.acquire(root);
        if (FAILED(hr)) { profiles->Release(); CoUninitialize(); return 1; }
    }
    if (operation == L"install" && count == 4) hr = install(profiles, args[2], args[3]);
    else if (operation == L"update" && count == 5) hr=upgrade(profiles,args[2],args[3],args[4]);
    else if (operation == L"repair" && count == 2) {
        std::wstring installed,build;
        hr=ownedPayload(installed,build);
        if (hr==S_OK) hr=repairProfile(profiles);
        else if (hr==S_FALSE) hr=HRESULT_FROM_WIN32(ERROR_NOT_FOUND);
    }
    else if (operation == L"enable") hr=changeUserProfile(profiles,false);
    else if (operation == L"disable-legacy" && count==2) hr=installUserProfile(true,ArionLegacyLanguage);
    else if (operation == L"disable") hr=changeUserProfile(profiles,true);
    else if (operation == L"select") hr=selectProfile(profiles);
    else if (operation == L"remove") {
        OwnedProfiles owned;
        hr=observeOwnedProfiles(profiles,owned);
        if (FAILED(hr)) { profiles->Release(); CoUninitialize(); return 1; }
        HRESULT registry=S_OK;
        registeredPath(&registry);
        if (FAILED(registry)) { profiles->Release(); CoUninitialize(); return 1; }
        std::wstring installed, build;
        hr = ownedPayload(installed, build);
        if (hr==S_FALSE && owned.registered()) {
            // A dangling owned profile has no payload to replace or delete.
            // Remove only its exact known language/GUID registrations.
            hr=removeOwnedRegistration(profiles,owned);
            profiles->Release(); CoUninitialize(); return SUCCEEDED(hr) ? 0 : 1;
        }
        if (hr != S_OK) { profiles->Release(); CoUninitialize(); return hr == S_FALSE ? 0 : 1; }
        PayloadRemovalGuard removal;
        std::wstring root; hr=machineRoot(root);
        const auto loader=root+L"\\ArionTSFLoader.dll";
        const bool hasLoader=arionLoader::loaderPath(root,registeredPath());
        if (SUCCEEDED(hr) && hasLoader) hr=removal.reserve(loader);
        if (SUCCEEDED(hr)) hr=removal.prepare(installed);
        if (SUCCEEDED(hr)) hr=removeOwnedRegistration(profiles,owned);
        if (SUCCEEDED(hr)) {
            hr=removeKeyboardPayload(installed,removal.files,removal.handles);
            if (SUCCEEDED(hr) && hasLoader && !DeleteFileW(loader.c_str())) hr=HRESULT_FROM_WIN32(GetLastError());
        }
    } else if (operation == L"query") {
        OwnedProfiles owned;
        HRESULT query=observeOwnedProfiles(profiles,owned);
        TF_INPUTPROCESSORPROFILE active{};
        HRESULT selected = profiles->GetActiveProfile(GUID_TFCAT_TIP_KEYBOARD, &active);
        BOOL enabled = FALSE;
        HRESULT enablement = S_OK;
        const auto &profile=owned.currentFound ? owned.current : owned.legacy;
        if (owned.registered()) {
            ITfInputProcessorProfiles *legacy = nullptr;
            enablement = profiles->QueryInterface(IID_ITfInputProcessorProfiles,reinterpret_cast<void**>(&legacy));
            if (SUCCEEDED(enablement)) { enablement=legacy->IsEnabledLanguageProfile(ArionService,profile.langid,ArionProfile,&enabled); legacy->Release(); }
        }
        bool current=selected==S_OK && ownedActiveProfile(owned,active);
        HRESULT registry=S_OK;
        bool hasComPath=!registeredPath(&registry).empty();
        bool registered=owned.registered() || hasComPath;
        std::wstring installed, build; bool loader=false;
        hr = FAILED(query) ? query : FAILED(registry) ? registry : FAILED(enablement) ? enablement : FAILED(selected) ? selected : registered ? ownedPayload(installed,build,&loader) : S_OK;
        bool pending=false;
        if (SUCCEEDED(hr) && registered) {
            std::wstring root,target,previous;
            HRESULT intent=machineRoot(root);
            if (SUCCEEDED(intent)) intent=readUpgradeIntent(root+L"\\.upgrade-pending",target,previous);
            if (intent==S_OK) {
                pending=true;
                if (build!=target && build!=previous) hr=HRESULT_FROM_WIN32(ERROR_REVISION_MISMATCH);
            } else if (FAILED(intent)) hr=intent;
        }
        if (SUCCEEDED(hr)) std::cout << "{\"registered\":" << (registered ? "true" : "false") << ",\"enabled\":" << (profileEnabled(profile,enabled) ? "true" : "false") << ",\"selected\":" << (current ? "true" : "false") << ",\"build\":" << quoted(build) << ",\"legacy\":" << (owned.legacyFound ? "true" : "false") << ",\"loader\":" << (loader?"true":"false") << ",\"upgradeRequired\":" << (owned.upgradeRequired(hasComPath) || pending || (hasComPath && !loader) ? "true" : "false") << "}\n";
    } else hr = E_INVALIDARG;
    profiles->Release(); CoUninitialize();
    if (FAILED(hr)) { std::cerr << "TSF operation failed: 0x" << std::hex << static_cast<unsigned long>(hr) << "\n"; return hr==HRESULT_FROM_WIN32(ERROR_BUSY)?75:1; }
    return 0;
}
