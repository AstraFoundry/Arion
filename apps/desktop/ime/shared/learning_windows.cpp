#ifdef _WIN32
#define WIN32_LEAN_AND_MEAN
#include <windows.h>
#include <objbase.h>
#include <sddl.h>
#include <wincrypt.h>
#include "learning.h"
#include <algorithm>
#include <filesystem>
#include <cstring>

namespace arion::ime {
namespace {
bool failed(LearningError &error, int code, const char *operation) { error = {code,operation}; return false; }
std::wstring wide(const std::string &text) {
    if (text.size() > static_cast<size_t>(INT_MAX)) return {};
    int size = MultiByteToWideChar(CP_UTF8,MB_ERR_INVALID_CHARS,text.data(),static_cast<int>(text.size()),nullptr,0);
    if (!size) return {};
    std::wstring value(static_cast<size_t>(size),L'\0');
    if (!MultiByteToWideChar(CP_UTF8,MB_ERR_INVALID_CHARS,text.data(),static_cast<int>(text.size()),value.data(),size)) return {};
    return value;
}
class FileLocks final : public LearningLock {
public:
    std::vector<std::pair<HANDLE,DWORD>> files;
    ~FileLocks() override { for (auto [handle,offset] : files) { OVERLAPPED operation{}; operation.Offset=offset; UnlockFileEx(handle,0,1,0,&operation); CloseHandle(handle); } }
};
class WindowsProvider final : public LearningProvider {
    std::string directory_;
    PSECURITY_DESCRIPTOR descriptor_ = nullptr;
    SECURITY_ATTRIBUTES attributes_{};
    bool hold(const std::string &file, bool exclusive, FileLocks &held, LearningError &error, DWORD offset=0, bool wait=false) {
        auto path = wide(file); if (path.empty()) return failed(error,9,"lock path");
        HANDLE handle = CreateFileW(path.c_str(),GENERIC_READ|GENERIC_WRITE,FILE_SHARE_READ|FILE_SHARE_WRITE,&attributes_,OPEN_ALWAYS,FILE_ATTRIBUTE_NORMAL|FILE_FLAG_OPEN_REPARSE_POINT,nullptr);
        if (handle == INVALID_HANDLE_VALUE) return failed(error,8,"lock open");
        BY_HANDLE_FILE_INFORMATION info{}; OVERLAPPED operation{};
        operation.Offset=offset;
        bool ok = GetFileInformationByHandle(handle,&info) && !(info.dwFileAttributes & (FILE_ATTRIBUTE_REPARSE_POINT|FILE_ATTRIBUTE_DIRECTORY));
        if (ok) {
            ULONGLONG deadline=GetTickCount64()+(wait?2000:0);
            do {
                ok=LockFileEx(handle,LOCKFILE_FAIL_IMMEDIATELY|(exclusive?LOCKFILE_EXCLUSIVE_LOCK:0),0,1,0,&operation)!=FALSE;
                if (ok || GetLastError()!=ERROR_LOCK_VIOLATION || GetTickCount64()>=deadline) break;
                Sleep(1);
            } while (true);
        }
        if (!ok) { CloseHandle(handle); return failed(error,8,"lock acquisition"); }
        held.files.emplace_back(handle,offset); return true;
    }
public:
    explicit WindowsProvider(std::string directory):directory_(std::move(directory)) {}
    ~WindowsProvider() override { if (descriptor_) LocalFree(descriptor_); }
    bool initialize(LearningError &error) {
        HANDLE token = nullptr; DWORD count = 0;
        if (!OpenProcessToken(GetCurrentProcess(),TOKEN_QUERY,&token)) return failed(error,1,"user identity");
        GetTokenInformation(token,TokenUser,nullptr,0,&count); std::vector<uint8_t> user(count);
        bool ok = count && GetTokenInformation(token,TokenUser,user.data(),count,&count);
        wchar_t *sid = nullptr;
        if (ok) ok = ConvertSidToStringSidW(reinterpret_cast<TOKEN_USER*>(user.data())->User.Sid,&sid);
        CloseHandle(token); if (!ok) return failed(error,1,"user identity");
        std::wstring sddl = L"D:P(A;OICI;FA;;;SY)(A;OICI;FA;;;" + std::wstring(sid) + L")"; LocalFree(sid);
        if (!ConvertStringSecurityDescriptorToSecurityDescriptorW(sddl.c_str(),SDDL_REVISION_1,&descriptor_,nullptr)) return failed(error,1,"private file policy");
        attributes_ = {sizeof(attributes_),descriptor_,FALSE};
        std::error_code ec; auto directory = std::filesystem::u8path(directory_);
        std::filesystem::create_directories(directory.parent_path(),ec); if (ec) return failed(error,5,"vault parent");
        auto path = wide(directory_); if (path.empty()) return failed(error,9,"vault path");
        if (!CreateDirectoryW(path.c_str(),&attributes_) && GetLastError()!=ERROR_ALREADY_EXISTS) return failed(error,5,"vault create");
        HANDLE handle = CreateFileW(path.c_str(),FILE_READ_ATTRIBUTES,FILE_SHARE_READ|FILE_SHARE_WRITE|FILE_SHARE_DELETE,nullptr,OPEN_EXISTING,FILE_FLAG_BACKUP_SEMANTICS|FILE_FLAG_OPEN_REPARSE_POINT,nullptr);
        if (handle == INVALID_HANDLE_VALUE) return failed(error,5,"vault open");
        BY_HANDLE_FILE_INFORMATION info{}; ok = GetFileInformationByHandle(handle,&info) && (info.dwFileAttributes & FILE_ATTRIBUTE_DIRECTORY) && !(info.dwFileAttributes & FILE_ATTRIBUTE_REPARSE_POINT);
        CloseHandle(handle); return ok || failed(error,5,"vault type");
    }
    const std::string &directory() const override { return directory_; }
    bool cooperative() const override { return true; }
    bool lease(const std::string &file,bool removing,std::unique_ptr<LearningLock> &held,LearningError &error) override {
        auto locks=std::make_unique<FileLocks>();
        if (!hold(learningPath(directory_,".learning.lock"),false,*locks,error) || !hold(file+".lock",removing,*locks,error,1)) return false;
        held=std::move(locks);return true;
    }
    bool lock(const std::string &file, bool readonly, bool reset, std::unique_ptr<LearningLock> &held, LearningError &error) override {
        auto locks = std::make_unique<FileLocks>();
        if (!hold(learningPath(directory_,".learning.lock"),reset,*locks,error) || (!file.empty() && !hold(file+".lock",!readonly,*locks,error,0,true))) return false;
        held = std::move(locks); return true;
    }
    bool revision(const std::string &file,std::string &value,LearningError &error) override {
        auto path=wide(file);if (path.empty()) return failed(error,9,"snapshot path");
        HANDLE handle=CreateFileW(path.c_str(),FILE_READ_ATTRIBUTES,FILE_SHARE_READ|FILE_SHARE_WRITE|FILE_SHARE_DELETE,nullptr,OPEN_EXISTING,FILE_FLAG_OPEN_REPARSE_POINT,nullptr);
        if (handle==INVALID_HANDLE_VALUE) {if (GetLastError()==ERROR_FILE_NOT_FOUND) {value="missing";return true;}return failed(error,5,"snapshot revision");}
        BY_HANDLE_FILE_INFORMATION info{};
        bool ok=GetFileInformationByHandle(handle,&info) && !(info.dwFileAttributes&(FILE_ATTRIBUTE_REPARSE_POINT|FILE_ATTRIBUTE_DIRECTORY));CloseHandle(handle);
        if (!ok) return failed(error,5,"snapshot revision");
        value=std::to_string(info.dwVolumeSerialNumber)+":"+std::to_string(info.nFileIndexHigh)+":"+std::to_string(info.nFileIndexLow)+":"+
            std::to_string(info.ftLastWriteTime.dwHighDateTime)+":"+std::to_string(info.ftLastWriteTime.dwLowDateTime)+":"+std::to_string(info.nFileSizeHigh)+":"+std::to_string(info.nFileSizeLow);
        return true;
    }
    bool read(const std::string &file, LearningBytes &bytes, bool &missing, LearningError &error) override {
        missing = false; auto path = wide(file); if (path.empty()) return failed(error,9,"snapshot path");
        HANDLE handle = CreateFileW(path.c_str(),GENERIC_READ,FILE_SHARE_READ|FILE_SHARE_DELETE,nullptr,OPEN_EXISTING,FILE_FLAG_OPEN_REPARSE_POINT,nullptr);
        if (handle == INVALID_HANDLE_VALUE) { if (GetLastError()==ERROR_FILE_NOT_FOUND) { missing=true; bytes.clear(); return true; } return failed(error,5,"snapshot open"); }
        BY_HANDLE_FILE_INFORMATION info{}; LARGE_INTEGER size{};
        bool ok = GetFileInformationByHandle(handle,&info) && !(info.dwFileAttributes & (FILE_ATTRIBUTE_REPARSE_POINT|FILE_ATTRIBUTE_DIRECTORY)) && GetFileSizeEx(handle,&size) && size.QuadPart>=0 && static_cast<uint64_t>(size.QuadPart)<=maximumLearningSnapshot+4096;
        LearningBytes next; if (ok) next.resize(static_cast<size_t>(size.QuadPart));
        size_t offset=0;
        while (ok && offset<next.size()) { DWORD count=0; ok=ReadFile(handle,next.data()+offset,static_cast<DWORD>(next.size()-offset),&count,nullptr)&&count; offset+=count; }
        if (!CloseHandle(handle)) ok=false;
        if (!ok) return failed(error,5,"snapshot read");
        bytes = std::move(next); return true;
    }
    bool writeAtomic(const std::string &file, const LearningBytes &bytes, LearningError &error) override {
        if (bytes.size()>maximumLearningSnapshot+4096) return failed(error,3,"snapshot size");
        auto target=wide(file); if (target.empty()) return failed(error,9,"snapshot path");
        GUID id{}; if (FAILED(CoCreateGuid(&id))) return failed(error,6,"temporary identity");
        wchar_t name[40]{}; StringFromGUID2(id,name,40);
        std::wstring temporary=std::filesystem::u8path(file).parent_path().wstring()+L"\\.encrypted-"+name;
        HANDLE handle=CreateFileW(temporary.c_str(),GENERIC_WRITE,0,&attributes_,CREATE_NEW,FILE_ATTRIBUTE_NORMAL|FILE_FLAG_OPEN_REPARSE_POINT,nullptr);
        if (handle==INVALID_HANDLE_VALUE) return failed(error,6,"temporary create");
        bool ok=true; size_t offset=0;
        while (offset<bytes.size()) { DWORD count=0; if (!WriteFile(handle,bytes.data()+offset,static_cast<DWORD>(bytes.size()-offset),&count,nullptr)||!count) {ok=false;break;} offset+=count; }
        if (ok && !FlushFileBuffers(handle)) ok=false;
        if (!CloseHandle(handle)) ok=false;
        if (ok && !MoveFileExW(temporary.c_str(),target.c_str(),MOVEFILE_REPLACE_EXISTING|MOVEFILE_WRITE_THROUGH)) ok=false;
        if (!ok) { DeleteFileW(temporary.c_str()); return failed(error,6,"snapshot replace"); }
        return true;
    }
    bool remove(const std::string &file, LearningError &error) override {
        auto path=wide(file); if (path.empty()) return failed(error,9,"snapshot path");
        if (DeleteFileW(path.c_str()) || GetLastError()==ERROR_FILE_NOT_FOUND) return true;
        return failed(error,5,"snapshot remove");
    }
    bool files(std::vector<std::string> &names, LearningError &error) override {
        std::error_code ec; std::vector<std::string> next;
        std::filesystem::directory_iterator iterator(std::filesystem::u8path(directory_),ec), end;
        if (ec) return failed(error,5,"vault enumeration");
        for (;iterator!=end;iterator.increment(ec)) { if (ec) return failed(error,5,"vault enumeration"); next.push_back(iterator->path().filename().u8string()); }
        if (ec) return failed(error,5,"vault enumeration");
        names=std::move(next); return true;
    }
    bool seal(const std::string &name, const LearningBytes &plain, LearningBytes &sealed, LearningError &error) override {
        if (plain.size()>maximumLearningSnapshot) return failed(error,3,"snapshot size");
        std::string label="si.astrafoundry.arion.keyboard.userdb.v1/"+name;
        DATA_BLOB input{static_cast<DWORD>(plain.size()),const_cast<BYTE*>(plain.data())}, entropy{static_cast<DWORD>(label.size()),reinterpret_cast<BYTE*>(label.data())}, output{};
        if (!CryptProtectData(&input,L"Arion keyboard learning",&entropy,nullptr,nullptr,CRYPTPROTECT_UI_FORBIDDEN,&output)) return failed(error,2,"DPAPI seal");
        LearningBytes next{'A','R','I','O','N','W','D','1'}; next.insert(next.end(),output.pbData,output.pbData+output.cbData); LocalFree(output.pbData); sealed=std::move(next); return true;
    }
    bool unseal(const std::string &name, const LearningBytes &sealed, LearningBytes &plain, LearningError &error) override {
        if (sealed.size()<=8 || sealed.size()>maximumLearningSnapshot+4096 || std::memcmp(sealed.data(),"ARIONWD1",8)) return failed(error,4,"snapshot envelope");
        std::string label="si.astrafoundry.arion.keyboard.userdb.v1/"+name;
        DATA_BLOB input{static_cast<DWORD>(sealed.size()-8),const_cast<BYTE*>(sealed.data()+8)}, entropy{static_cast<DWORD>(label.size()),reinterpret_cast<BYTE*>(label.data())}, output{};
        if (!CryptUnprotectData(&input,nullptr,&entropy,nullptr,nullptr,CRYPTPROTECT_UI_FORBIDDEN,&output)) return failed(error,2,"DPAPI authentication");
        bool valid=output.cbData<=maximumLearningSnapshot;
        if (valid) plain.assign(output.pbData,output.pbData+output.cbData);
        SecureZeroMemory(output.pbData,output.cbData); LocalFree(output.pbData);
        return valid || failed(error,3,"snapshot size");
    }
};
}
std::shared_ptr<LearningProvider> makeWindowsLearningProvider(const std::string &directory, LearningError &error) {
    auto provider=std::make_shared<WindowsProvider>(directory);
    if (!provider->initialize(error)) return nullptr;
    return provider;
}
} // namespace arion::ime
#endif
