// Synthetic filesystem checks only; never registers or removes a real TIP.
#define wmain arion_control_main
#include "control.cpp"
#undef wmain
#include <fstream>
#include <filesystem>
#include <stdexcept>

static void require(bool result,const char *message) { if (!result) throw std::runtime_error(message); }
static void write(const std::wstring &path,const std::string &bytes) {
    std::ofstream stream(std::filesystem::path(path),std::ios::binary);
    stream<<bytes; stream.close(); require(bool(stream),"write fixture");
}
static std::wstring fixture(const std::wstring &parent,const wchar_t *name,bool keyboard=true) {
    const auto version=parent+L"\\"+name;
    require(CreateDirectoryW(version.c_str(),nullptr),"create version");
    write(version+L"\\ArionTSF.dll","synthetic host");
    if (!keyboard) return version+L"\\ArionTSF.dll";
    const auto root=version+L"\\keyboard";
    for (const auto &directory:{root,root+L"\\lib",root+L"\\data",root+L"\\data\\build"}) require(CreateDirectoryW(directory.c_str(),nullptr),"create fixture directory");
    const wchar_t *names[]={L"lib/ArionKeyboard.dll",L"lib/ArionRime.dll",L"data/build/arion_pinyin.schema.yaml"};
    std::string manifest="{\"files\":{";
    for (size_t i=0;i<3;++i) {
        auto path=keyboardNativePath(root,names[i]); write(path,"fixture");
        if (i) manifest+=",";
        manifest+=quoted(names[i])+":{\"sha256\":"+quoted(fileHash(path))+",\"sizeBytes\":7}";
    }
    write(root+L"\\runtime.json",manifest+"}}");
    return version+L"\\ArionTSF.dll";
}
int wmain(int count,wchar_t **args) {
    try {
        if (count==2) {
            HMODULE mapped=LoadLibraryExW(args[1],nullptr,LOAD_LIBRARY_SEARCH_DLL_LOAD_DIR|LOAD_LIBRARY_SEARCH_SYSTEM32);
            require(mapped!=nullptr,"map actual owned loader DLL");
            { PayloadRemovalGuard guard; require(FAILED(guard.reserve(args[1])),"mapped loader refuses reservation before any unregister callback"); }
            require(FreeLibrary(mapped),"release owned loader mapping");
            { PayloadRemovalGuard guard;
                require(SUCCEEDED(guard.reserve(args[1])),"unmapped loader permits reservation");
                mapped=LoadLibraryExW(args[1],nullptr,LOAD_LIBRARY_SEARCH_DLL_LOAD_DIR|LOAD_LIBRARY_SEARCH_SYSTEM32);
                if (mapped) FreeLibrary(mapped);
                require(!mapped,"reservation prevents a new image mapping before unregister");
            }
            mapped=LoadLibraryExW(args[1],nullptr,LOAD_LIBRARY_SEARCH_DLL_LOAD_DIR|LOAD_LIBRARY_SEARCH_SYSTEM32);
            require(mapped!=nullptr,"failed preflight did not mutate loader bytes"); FreeLibrary(mapped);
        } else require(count==1,"optional owned loader DLL argument");
        wchar_t temporary[MAX_PATH]{},unique[MAX_PATH]{};
        require(GetTempPathW(MAX_PATH,temporary)>0,"temporary root");
        require(GetTempFileNameW(temporary,L"ari",0,unique)!=0,"unique fixture path");
        require(DeleteFileW(unique)&&CreateDirectoryW(unique,nullptr),"create owned fixture");
        const std::wstring root=unique;
        auto installed=fixture(root,L"locked");
        const auto keyboard=root+L"\\locked\\keyboard";
        HANDLE locked=CreateFileW((keyboard+L"\\lib\\ArionRime.dll").c_str(),GENERIC_READ,FILE_SHARE_READ,nullptr,OPEN_EXISTING,0,nullptr);
        require(locked!=INVALID_HANDLE_VALUE,"hold simulated loaded module");
        { PayloadRemovalGuard guard; require(FAILED(guard.prepare(installed)),"locked module must refuse before unregister"); }
        require(GetFileAttributesW(installed.c_str())!=INVALID_FILE_ATTRIBUTES,"host preserved after refusal");
        CloseHandle(locked);
        write(keyboard+L"\\unexpected.txt","preserve");
        { PayloadRemovalGuard guard; require(FAILED(guard.prepare(installed)),"unlisted file must refuse before unregister"); }
        require(DeleteFileW((keyboard+L"\\unexpected.txt").c_str()),"remove only owned extra fixture");
        write(keyboard+L"\\lib\\ArionRime.dll","changed");
        { PayloadRemovalGuard guard; require(FAILED(guard.prepare(installed)),"tampered module must refuse"); }
        write(keyboard+L"\\lib\\ArionRime.dll","fixture");
        {
            PayloadRemovalGuard guard; require(SUCCEEDED(guard.prepare(installed)),"reserve valid closure");
            require(SUCCEEDED(removeKeyboardPayload(installed,guard.files,guard.handles)),"remove verified closure with reserved handles");
        }
        require(GetFileAttributesW((root+L"\\locked").c_str())==INVALID_FILE_ATTRIBUTES,"version removed");
        installed=fixture(root,L"legacy",false);
        { PayloadRemovalGuard guard; require(SUCCEEDED(guard.prepare(installed)),"reserve legacy host"); require(SUCCEEDED(removeKeyboardPayload(installed,guard.files,guard.handles)),"remove legacy host"); }
        require(RemoveDirectoryW(root.c_str()),"remove empty owned test root");
        std::cout<<"PASS: locked/tampered closures preserve files; verified and legacy closures remove exactly owned files.\n";
        return 0;
    } catch (const std::exception &error) { std::cerr<<error.what()<<"\n"; return 1; }
}
