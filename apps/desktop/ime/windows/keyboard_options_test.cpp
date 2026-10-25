#define WIN32_LEAN_AND_MEAN
#include <windows.h>
#include <winioctl.h>
#include "keyboard_options.h"
#include <fstream>
#include <iostream>
#include <stdexcept>
static void require(bool ok,const char *message) {if (!ok) throw std::runtime_error(message);}
static std::string bytes(const std::wstring &path) {std::ifstream file(std::filesystem::path(path),std::ios::binary);return std::string(std::istreambuf_iterator<char>(file),{});}
static void write(const std::wstring &path,const std::string &value) {std::ofstream file(std::filesystem::path(path),std::ios::binary);file<<value;require(file.good(),"write owned fixture");}
class Processes {
    HANDLE job_=nullptr;
    std::vector<PROCESS_INFORMATION> children_;
public:
    Processes() {job_=CreateJobObjectW(nullptr,nullptr);JOBOBJECT_EXTENDED_LIMIT_INFORMATION policy{};policy.BasicLimitInformation.LimitFlags=JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE;require(job_ && SetInformationJobObject(job_,JobObjectExtendedLimitInformation,&policy,sizeof(policy)),"owned child process lifetime");}
    ~Processes() {for (const auto &child:children_) {CloseHandle(child.hThread);CloseHandle(child.hProcess);}if (job_) CloseHandle(job_);}
    void add(const std::wstring &path,const wchar_t *field) {
        wchar_t executable[32768]{};require(GetModuleFileNameW(nullptr,executable,32768)!=0,"owned executable path");
        std::wstring command=L"\""+std::wstring(executable)+L"\" --worker \""+path+L"\" "+field;
        STARTUPINFOW startup{};startup.cb=sizeof(startup);PROCESS_INFORMATION child{};
        require(CreateProcessW(executable,command.data(),nullptr,nullptr,FALSE,CREATE_NO_WINDOW|CREATE_SUSPENDED,nullptr,nullptr,&startup,&child)!=FALSE,"launch owned options child");
        if (!AssignProcessToJobObject(job_,child.hProcess)) {TerminateProcess(child.hProcess,99);CloseHandle(child.hThread);CloseHandle(child.hProcess);throw std::runtime_error("own options worker lifetime");}
        children_.push_back(child);
    }
    void run() {
        for (const auto &child:children_) require(ResumeThread(child.hThread)!=static_cast<DWORD>(-1),"start options worker");
        for (const auto &child:children_) {require(WaitForSingleObject(child.hProcess,15000)==WAIT_OBJECT_0,"options worker deadline");DWORD status=1;require(GetExitCodeProcess(child.hProcess,&status) && status==0,"cross-process toggle failed");}
    }
};
static void junction(const std::wstring &path,const std::wstring &input) {
    std::wstring target=std::filesystem::path(input).make_preferred().wstring();
    require(CreateDirectoryW(path.c_str(),nullptr)!=FALSE,"create owned junction directory");
    keyboardOptionsDetail::Handle directory(CreateFileW(path.c_str(),GENERIC_WRITE,0,nullptr,OPEN_EXISTING,FILE_FLAG_OPEN_REPARSE_POINT|FILE_FLAG_BACKUP_SEMANTICS,nullptr));
    require(directory.value!=INVALID_HANDLE_VALUE,"open owned junction");
    std::wstring substitute=L"\\??\\"+target;size_t substituteBytes=substitute.size()*sizeof(wchar_t),printBytes=target.size()*sizeof(wchar_t);
    struct Header {DWORD tag;WORD length,reserved,substituteOffset,substituteLength,printOffset,printLength;};
    std::vector<unsigned char> buffer(sizeof(Header)+substituteBytes+sizeof(wchar_t)+printBytes+sizeof(wchar_t));
    auto header=reinterpret_cast<Header*>(buffer.data());header->tag=IO_REPARSE_TAG_MOUNT_POINT;header->length=static_cast<WORD>(buffer.size()-8);header->substituteLength=static_cast<WORD>(substituteBytes);header->printOffset=static_cast<WORD>(substituteBytes+sizeof(wchar_t));header->printLength=static_cast<WORD>(printBytes);
    std::memcpy(buffer.data()+sizeof(Header),substitute.c_str(),substituteBytes+sizeof(wchar_t));std::memcpy(buffer.data()+sizeof(Header)+header->printOffset,target.c_str(),printBytes+sizeof(wchar_t));
    DWORD count=0;require(DeviceIoControl(directory.value,FSCTL_SET_REPARSE_POINT,buffer.data(),static_cast<DWORD>(buffer.size()),nullptr,0,&count,nullptr)!=FALSE,"set actual owned directory reparse point");
}
int wmain(int argc,wchar_t **argv) {
    try {
        if (argc==4 && std::wstring(argv[1])==L"--worker") {KeyboardOptions result;KeyboardOption field=std::wstring(argv[3])==L"width"?KeyboardOption::FullWidth:KeyboardOption::EnglishPunctuation;for (int i=0;i<31;++i) require(toggleKeyboardOption(argv[2],field,result),"atomic worker toggle");return 0;}
        require(argc==2,"fresh owned options directory required");std::filesystem::path root(argv[1]);
        wchar_t *owner=nullptr;require(ConvertSidToStringSidW(keyboardOptionsDetail::security().user(),&owner)!=FALSE,"private fixture owner");
        std::wstring directoryPolicy=L"D:P(A;OICI;FA;;;SY)(A;OICI;FA;;;"+std::wstring(owner)+L")";LocalFree(owner);PSECURITY_DESCRIPTOR directoryDescriptor=nullptr;
        require(ConvertStringSecurityDescriptorToSecurityDescriptorW(directoryPolicy.c_str(),SDDL_REVISION_1,&directoryDescriptor,nullptr)!=FALSE,"private fixture policy");
        SECURITY_ATTRIBUTES directoryAttributes{sizeof(directoryAttributes),directoryDescriptor,FALSE};bool created=CreateDirectoryW(root.c_str(),&directoryAttributes)!=FALSE;LocalFree(directoryDescriptor);require(created,"fresh owned private fixture directory");auto path=(root/L"options.json").wstring();
        KeyboardOptions options{true,true};require(readKeyboardOptions(path,options) && !options.fullWidth && !options.englishPunctuation,"absent file defaults");
        write(path,"{\"fullWidth\":\"unknown\",\"englishPunctuation\":null}");require(readKeyboardOptions(path,options) && !options.fullWidth && !options.englishPunctuation,"unknown values default");
        write(path,"malformed");require(readKeyboardOptions(path,options) && !options.fullWidth && !options.englishPunctuation,"malformed JSON follows startup defaults");
        require(toggleKeyboardOption(path,KeyboardOption::FullWidth,options) && options.fullWidth && !options.englishPunctuation,"persist first option");
        require(toggleKeyboardOption(path,KeyboardOption::EnglishPunctuation,options) && options.fullWidth && options.englishPunctuation,"preserve independent option");
        auto old=bytes(path);require(SetFileAttributesW(path.c_str(),FILE_ATTRIBUTE_READONLY)!=FALSE,"set failure fixture readonly");KeyboardOptions unchanged{false,true};
        require(!toggleKeyboardOption(path,KeyboardOption::FullWidth,unchanged) && !unchanged.fullWidth && unchanged.englishPunctuation && bytes(path)==old,"failed replace preserves file and caller mode");
        require(SetFileAttributesW(path.c_str(),FILE_ATTRIBUTE_NORMAL)!=FALSE,"restore owned file attributes");
        {
            keyboardOptionsDetail::Handle held(CreateFileW(path.c_str(),GENERIC_READ,FILE_SHARE_READ,nullptr,OPEN_EXISTING,0,nullptr));require(held.value!=INVALID_HANDLE_VALUE,"hold nonreplaceable owned target");
            require(!toggleKeyboardOption(path,KeyboardOption::FullWidth,unchanged) && bytes(path)==old && !unchanged.fullWidth && unchanged.englishPunctuation,"replacement sharing failure preserves mode");
        }
        require(toggleKeyboardOption(path,KeyboardOption::FullWidth,options) && !options.fullWidth && options.englishPunctuation,"recovery only after actual successful write");
        std::filesystem::remove(std::filesystem::path(path));
        {Processes processes;processes.add(path,L"width");processes.add(path,L"punctuation");processes.run();}
        require(readKeyboardOptions(path,options) && options.fullWidth && options.englishPunctuation,"parallel process updates preserve both fields and odd toggle counts");
        const auto linked=(root/L"linked.json").wstring();require(CreateHardLinkW(linked.c_str(),path.c_str(),nullptr)!=FALSE,"create owned hardlink fixture");
        require(!toggleKeyboardOption(linked,KeyboardOption::FullWidth,unchanged),"multiply-linked target refused");std::filesystem::remove(std::filesystem::path(linked));
        const auto unsafe=(root/L"unsafe.json").wstring();write(unsafe,"{}");wchar_t *sid=nullptr;require(ConvertSidToStringSidW(keyboardOptionsDetail::security().user(),&sid)!=FALSE,"owned ACL user");
        std::wstring policy=L"D:P(A;;FA;;;"+std::wstring(sid)+L")(A;;GW;;;WD)";LocalFree(sid);PSECURITY_DESCRIPTOR descriptor=nullptr;
        require(ConvertStringSecurityDescriptorToSecurityDescriptorW(policy.c_str(),SDDL_REVISION_1,&descriptor,nullptr)!=FALSE,"unsafe fixture policy");
        BOOL present=FALSE,defaulted=FALSE;PACL acl=nullptr;require(GetSecurityDescriptorDacl(descriptor,&present,&acl,&defaulted)!=FALSE && present,"unsafe fixture DACL");
        require(SetNamedSecurityInfoW(const_cast<wchar_t*>(unsafe.c_str()),SE_FILE_OBJECT,DACL_SECURITY_INFORMATION|PROTECTED_DACL_SECURITY_INFORMATION,nullptr,nullptr,acl,nullptr)==ERROR_SUCCESS,"set owned unsafe fixture ACL");LocalFree(descriptor);
        require(!readKeyboardOptions(unsafe,unchanged) && !toggleKeyboardOption(unsafe,KeyboardOption::FullWidth,unchanged) && bytes(unsafe)=="{}","foreign-write ACL refused without overwrite");
        const auto redirect=(root/L"redirect").wstring();junction(redirect,root.wstring());
        require(!readKeyboardOptions(redirect+L"\\options.json",unchanged) && !toggleKeyboardOption(redirect+L"\\options.json",KeyboardOption::FullWidth,unchanged),"actual reparse parent refused");
        require(RemoveDirectoryW(redirect.c_str())!=FALSE,"remove owned junction only");
        for (const auto &file:std::filesystem::directory_iterator(root)) require(file.path().filename().wstring().rfind(L".options-",0)!=0,"failed write temporary files cleaned");
        std::cout<<"PASS actual options atomic toggles, failure preservation, private ACL/reparse checks and independent process updates.\n";return 0;
    } catch (const std::exception &error) {std::cerr<<"FAIL "<<error.what()<<'\n';return 1;}
}
