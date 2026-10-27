#pragma once
#include "json.h"
#include <aclapi.h>
#include <sddl.h>
#include <objbase.h>
#include <filesystem>
struct KeyboardOptions { bool fullWidth=false,englishPunctuation=false; };
enum class KeyboardOption { FullWidth,EnglishPunctuation };
namespace keyboardOptionsDetail {
class Handle {
public:
    HANDLE value=INVALID_HANDLE_VALUE;
    explicit Handle(HANDLE handle=INVALID_HANDLE_VALUE):value(handle) {}
    ~Handle() {if (value!=INVALID_HANDLE_VALUE) CloseHandle(value);}
    Handle(const Handle &)=delete;
    Handle &operator=(const Handle &)=delete;
};
class Security {
    std::vector<unsigned char> user_;
    unsigned char system_[SECURITY_MAX_SID_SIZE]{},administrators_[SECURITY_MAX_SID_SIZE]{};
    PSECURITY_DESCRIPTOR descriptor_=nullptr;
public:
    SECURITY_ATTRIBUTES attributes{};
    bool ready=false;
    Security() {
        HANDLE raw=nullptr;if (!OpenProcessToken(GetCurrentProcess(),TOKEN_QUERY,&raw)) return;Handle token(raw);
        DWORD count=0;GetTokenInformation(raw,TokenUser,nullptr,0,&count);user_.resize(count);
        if (!count || !GetTokenInformation(raw,TokenUser,user_.data(),count,&count)) return;
        count=sizeof(system_);if (!CreateWellKnownSid(WinLocalSystemSid,nullptr,system_,&count)) return;
        count=sizeof(administrators_);if (!CreateWellKnownSid(WinBuiltinAdministratorsSid,nullptr,administrators_,&count)) return;
        wchar_t *sid=nullptr;if (!ConvertSidToStringSidW(user(),&sid)) return;
        std::wstring policy=L"O:"+std::wstring(sid)+L"D:P(A;;FA;;;SY)(A;;FA;;;"+std::wstring(sid)+L")";LocalFree(sid);
        if (!ConvertStringSecurityDescriptorToSecurityDescriptorW(policy.c_str(),SDDL_REVISION_1,&descriptor_,nullptr)) return;
        attributes={sizeof(attributes),descriptor_,FALSE};ready=true;
    }
    ~Security() {if (descriptor_) LocalFree(descriptor_);}
    PSID user() const {return user_.empty()?nullptr:reinterpret_cast<const TOKEN_USER*>(user_.data())->User.Sid;}
    bool trusted(PSID sid) const {return EqualSid(sid,user()) || EqualSid(sid,const_cast<unsigned char*>(system_)) || EqualSid(sid,const_cast<unsigned char*>(administrators_));}
    bool valid(HANDLE file,bool directory) const {
        BY_HANDLE_FILE_INFORMATION info{};
        if (!ready || !GetFileInformationByHandle(file,&info) || (info.dwFileAttributes&FILE_ATTRIBUTE_REPARSE_POINT) ||
            ((info.dwFileAttributes&FILE_ATTRIBUTE_DIRECTORY)!=0)!=directory || (!directory && info.nNumberOfLinks!=1)) return false;
        PSID owner=nullptr;PACL acl=nullptr;PSECURITY_DESCRIPTOR security=nullptr;
        if (GetSecurityInfo(file,SE_FILE_OBJECT,OWNER_SECURITY_INFORMATION|DACL_SECURITY_INFORMATION,&owner,nullptr,&acl,nullptr,&security)!=ERROR_SUCCESS) return false;
        bool ok=owner && EqualSid(owner,user()) && acl;
        constexpr DWORD write=GENERIC_WRITE|GENERIC_ALL|FILE_WRITE_DATA|FILE_APPEND_DATA|FILE_WRITE_EA|FILE_WRITE_ATTRIBUTES|FILE_DELETE_CHILD|DELETE|WRITE_DAC|WRITE_OWNER;
        for (DWORD i=0;ok && i<acl->AceCount;++i) {
            void *raw=nullptr;if (!GetAce(acl,i,&raw)) {ok=false;break;}auto header=static_cast<ACE_HEADER*>(raw);
            if (header->AceFlags&INHERIT_ONLY_ACE) continue;
            if (header->AceType==ACCESS_ALLOWED_ACE_TYPE) {auto ace=static_cast<ACCESS_ALLOWED_ACE*>(raw);if ((ace->Mask&write) && !trusted(&ace->SidStart)) ok=false;}
            else if (header->AceType!=ACCESS_DENIED_ACE_TYPE) ok=false;
        }
        LocalFree(security);return ok;
    }
};
inline Security &security() {static Security value;return value;}
inline HANDLE parent(const std::wstring &path) {
    auto directory=std::filesystem::path(path).parent_path();if (directory.empty() || path.find(L'\0')!=std::wstring::npos) return INVALID_HANDLE_VALUE;
    HANDLE handle=CreateFileW(directory.c_str(),FILE_READ_ATTRIBUTES|READ_CONTROL,FILE_SHARE_READ|FILE_SHARE_WRITE,nullptr,OPEN_EXISTING,FILE_FLAG_BACKUP_SEMANTICS|FILE_FLAG_OPEN_REPARSE_POINT,nullptr);
    if (handle!=INVALID_HANDLE_VALUE && !security().valid(handle,true)) {CloseHandle(handle);return INVALID_HANDLE_VALUE;}return handle;
}
inline bool read(const std::wstring &path,KeyboardOptions &output) {
    Handle file(CreateFileW(path.c_str(),GENERIC_READ|READ_CONTROL,FILE_SHARE_READ|FILE_SHARE_DELETE,nullptr,OPEN_EXISTING,FILE_FLAG_OPEN_REPARSE_POINT,nullptr));
    KeyboardOptions result;
    if (file.value==INVALID_HANDLE_VALUE) {if (GetLastError()!=ERROR_FILE_NOT_FOUND) return false;output=result;return true;}
    LARGE_INTEGER size{};if (!security().valid(file.value,false) || !GetFileSizeEx(file.value,&size) || size.QuadPart<0 || size.QuadPart>65536) return false;
    std::string text(static_cast<size_t>(size.QuadPart),'\0');DWORD count=0;
    if (!ReadFile(file.value,text.data(),static_cast<DWORD>(text.size()),&count,nullptr) || count!=text.size()) return false;
    Json value;
    // Absent/unknown settings retain the same defaults as the macOS adapter.
    // Filesystem/identity failures do not produce a replacement in-memory mode.
    if (JsonReader(text).read(value)) {
        const Json *field=value.get(L"fullWidth");if (field && field->type==Json::Boolean) result.fullWidth=field->boolean;
        field=value.get(L"englishPunctuation");if (field && field->type==Json::Boolean) result.englishPunctuation=field->boolean;
    }
    output=result;return true;
}
inline bool save(const std::wstring &path,const KeyboardOptions &options) {
    GUID id{};wchar_t name[40]{};if (FAILED(CoCreateGuid(&id)) || !StringFromGUID2(id,name,40)) return false;
    auto temporary=(std::filesystem::path(path).parent_path()/(L".options-"+std::wstring(name)+L".tmp")).wstring();
    std::string text=std::string("{\"fullWidth\":")+(options.fullWidth?"true":"false")+",\"englishPunctuation\":"+(options.englishPunctuation?"true":"false")+"}\n";
    bool ok=false;
    {
        Handle file(CreateFileW(temporary.c_str(),GENERIC_WRITE,0,&security().attributes,CREATE_NEW,FILE_ATTRIBUTE_NORMAL|FILE_FLAG_OPEN_REPARSE_POINT,nullptr));
        if (file.value!=INVALID_HANDLE_VALUE) {DWORD count=0;ok=WriteFile(file.value,text.data(),static_cast<DWORD>(text.size()),&count,nullptr) && count==text.size() && FlushFileBuffers(file.value);}
    }
    if (ok) ok=MoveFileExW(temporary.c_str(),path.c_str(),MOVEFILE_REPLACE_EXISTING|MOVEFILE_WRITE_THROUGH)!=FALSE;
    if (!ok) DeleteFileW(temporary.c_str());
    return ok;
}
}
inline bool prepareKeyboardDirectory(const std::wstring &path) {
    using namespace keyboardOptionsDetail;
    if (!security().ready) return false;
    std::error_code error;
    std::filesystem::create_directories(std::filesystem::path(path).parent_path(),error);
    if (error || (!CreateDirectoryW(path.c_str(),&security().attributes) && GetLastError()!=ERROR_ALREADY_EXISTS)) return false;
    Handle directory(CreateFileW(path.c_str(),FILE_READ_ATTRIBUTES|READ_CONTROL,FILE_SHARE_READ|FILE_SHARE_WRITE,nullptr,OPEN_EXISTING,FILE_FLAG_BACKUP_SEMANTICS|FILE_FLAG_OPEN_REPARSE_POINT,nullptr));
    return directory.value!=INVALID_HANDLE_VALUE && security().valid(directory.value,true);
}
inline bool readKeyboardOptions(const std::wstring &path,KeyboardOptions &result) {
    keyboardOptionsDetail::Handle directory(keyboardOptionsDetail::parent(path));
    return directory.value!=INVALID_HANDLE_VALUE && keyboardOptionsDetail::read(path,result);
}
inline KeyboardOptions readKeyboardOptions(const std::wstring &path) {KeyboardOptions result;readKeyboardOptions(path,result);return result;}
inline bool toggleKeyboardOption(const std::wstring &path,KeyboardOption field,KeyboardOptions &persisted) {
    using namespace keyboardOptionsDetail;
    Handle directory(parent(path));if (directory.value==INVALID_HANDLE_VALUE) return false;
    Handle held(CreateFileW((path+L".lock").c_str(),GENERIC_READ|GENERIC_WRITE|READ_CONTROL,FILE_SHARE_READ|FILE_SHARE_WRITE,&security().attributes,OPEN_ALWAYS,FILE_ATTRIBUTE_NORMAL|FILE_FLAG_OPEN_REPARSE_POINT,nullptr));
    if (held.value==INVALID_HANDLE_VALUE || !security().valid(held.value,false)) return false;
    OVERLAPPED operation{};ULONGLONG deadline=GetTickCount64()+1000;bool acquired=false;
    do {acquired=LockFileEx(held.value,LOCKFILE_FAIL_IMMEDIATELY|LOCKFILE_EXCLUSIVE_LOCK,0,1,0,&operation)!=FALSE;if (acquired || GetLastError()!=ERROR_LOCK_VIOLATION || GetTickCount64()>=deadline) break;Sleep(1);} while (true);
    if (!acquired) return false;
    KeyboardOptions next;bool ok=read(path,next);
    if (ok) {if (field==KeyboardOption::FullWidth) next.fullWidth=!next.fullWidth;else if (field==KeyboardOption::EnglishPunctuation) next.englishPunctuation=!next.englishPunctuation;else ok=false;}
    if (ok) ok=save(path,next);
    UnlockFileEx(held.value,0,1,0,&operation);
    if (ok) persisted=next;
    return ok;
}
