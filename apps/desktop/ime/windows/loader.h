#pragma once
#include <windows.h>
#include <shlobj.h>
#include <knownfolders.h>
#include <aclapi.h>
#include <sddl.h>
#include <string>
#include "identity.h"
#include "payload.h"

namespace arionLoader {
inline std::wstring registryKey() {
    wchar_t id[40]{}; StringFromGUID2(ArionService,id,40);
    return std::wstring(L"Software\\Classes\\CLSID\\")+id+L"\\InprocServer32";
}
inline HRESULT machineRoot(std::wstring &root) {
    wchar_t *folder=nullptr;
    HRESULT hr=SHGetKnownFolderPath(FOLDERID_ProgramFiles,KF_FLAG_DEFAULT,nullptr,&folder);
    if (FAILED(hr)) return hr;
    root=std::wstring(folder)+L"\\Arion\\InputMethod"; CoTaskMemFree(folder); return S_OK;
}
inline HRESULT readValue(const wchar_t *name,std::wstring &value) {
    wchar_t text[32768]{}; DWORD size=sizeof(text);
    LSTATUS status=RegGetValueW(HKEY_LOCAL_MACHINE,registryKey().c_str(),name,RRF_RT_REG_SZ|RRF_SUBKEY_WOW6464KEY,nullptr,text,&size);
    if (status==ERROR_FILE_NOT_FOUND || status==ERROR_PATH_NOT_FOUND) return S_FALSE;
    if (status!=ERROR_SUCCESS) return HRESULT_FROM_WIN32(status);
    if (size<sizeof(wchar_t) || size%sizeof(wchar_t) || text[size/sizeof(wchar_t)-1]!=0 || wcslen(text)+1!=size/sizeof(wchar_t)) return E_INVALIDARG;
    value=text; return S_OK;
}
inline bool corePath(const std::wstring &root,const std::wstring &path,std::wstring &hash) {
    if (path.size()!=root.size()+1+64+13 || _wcsnicmp(path.c_str(),root.c_str(),root.size()) || path[root.size()]!=L'\\') return false;
    const auto suffix=path.substr(root.size()+1);
    if (suffix.substr(64)!=L"\\ArionTSF.dll" || !hashName(suffix.substr(0,64))) return false;
    hash=suffix.substr(0,64); return true;
}
inline bool loaderPath(const std::wstring &root,const std::wstring &path) {
    return _wcsicmp(path.c_str(),(root+L"\\ArionTSFLoader.dll").c_str())==0;
}
inline HRESULT protectedEntry(const std::wstring &path,bool directory) {
    HANDLE file=CreateFileW(path.c_str(),READ_CONTROL|FILE_READ_ATTRIBUTES,FILE_SHARE_READ|FILE_SHARE_WRITE,nullptr,OPEN_EXISTING,FILE_FLAG_OPEN_REPARSE_POINT|FILE_FLAG_BACKUP_SEMANTICS,nullptr);
    if (file==INVALID_HANDLE_VALUE) return HRESULT_FROM_WIN32(GetLastError());
    BY_HANDLE_FILE_INFORMATION info{};
    bool safe=GetFileInformationByHandle(file,&info) && !(info.dwFileAttributes&FILE_ATTRIBUTE_REPARSE_POINT) && !!(info.dwFileAttributes&FILE_ATTRIBUTE_DIRECTORY)==directory && (directory || info.nNumberOfLinks==1);
    PACL dacl=nullptr; PSID owner=nullptr; PSECURITY_DESCRIPTOR descriptor=nullptr;
    DWORD status=GetSecurityInfo(file,SE_FILE_OBJECT,OWNER_SECURITY_INFORMATION|DACL_SECURITY_INFORMATION,&owner,nullptr,&dacl,nullptr,&descriptor);
    BYTE admin[SECURITY_MAX_SID_SIZE],system[SECURITY_MAX_SID_SIZE]; DWORD a=sizeof(admin),s=sizeof(system); PSID installer=nullptr;
    safe=safe && CreateWellKnownSid(WinBuiltinAdministratorsSid,nullptr,admin,&a) && CreateWellKnownSid(WinLocalSystemSid,nullptr,system,&s) && ConvertStringSidToSidW(L"S-1-5-80-956008885-3418522649-1831038044-1853292631-2271478464",&installer);
    auto trusted=[&](PSID sid) { return sid && (EqualSid(sid,admin) || EqualSid(sid,system) || (installer && EqualSid(sid,installer))); };
    safe=safe && status==ERROR_SUCCESS && dacl && trusted(owner);
    for (DWORD i=0;safe && i<dacl->AceCount;++i) {
        void *raw=nullptr; if (!GetAce(dacl,i,&raw)) { safe=false; break; }
        auto ace=static_cast<ACCESS_ALLOWED_ACE*>(raw);
        if (ace->Header.AceFlags&INHERIT_ONLY_ACE) continue;
        if (ace->Header.AceType==ACCESS_DENIED_ACE_TYPE) continue;
        if (ace->Header.AceType!=ACCESS_ALLOWED_ACE_TYPE) { safe=false; break; }
        if ((ace->Mask&(FILE_WRITE_DATA|FILE_APPEND_DATA|FILE_DELETE_CHILD|FILE_WRITE_EA|FILE_WRITE_ATTRIBUTES|DELETE|WRITE_DAC|WRITE_OWNER|GENERIC_WRITE|GENERIC_ALL)) && !trusted(&ace->SidStart)) safe=false;
    }
    if (installer) LocalFree(installer);
    if (descriptor) LocalFree(descriptor);
    CloseHandle(file); return safe?S_OK:E_ACCESSDENIED;
}
inline HRESULT protectedPath(const std::wstring &root,const std::wstring &path) {
    std::wstring hash;
    if (!loaderPath(root,path) && !corePath(root,path,hash)) return E_ACCESSDENIED;
    // Check Program Files and every descendant; checking the leaf alone permits
    // a writable or reparsed parent to redirect a privileged code load.
    const auto programFiles=root.substr(0,root.size()-wcslen(L"\\Arion\\InputMethod"));
    size_t position=programFiles.size();
    while (true) {
        HRESULT hr=protectedEntry(path.substr(0,position),true);
        if (FAILED(hr)) return hr;
        position=path.find(L'\\',position+1);
        if (position==std::wstring::npos) break;
    }
    return protectedEntry(path,false);
}
inline HRESULT validateCore(const std::wstring &root,const std::wstring &path,std::wstring &hash) {
    if (!corePath(root,path,hash)) return E_ACCESSDENIED;
    HRESULT hr=protectedPath(root,path);
    if (FAILED(hr)) return hr;
    return fileHash(path)==hash?S_OK:TRUST_E_BAD_DIGEST;
}
}
