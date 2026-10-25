// Explicitly load owned fixture DLLs; no registry, profile or user app changes.
#define WIN32_LEAN_AND_MEAN
#include <windows.h>
#include <iostream>
#include <cstdlib>
#include "loader.h"
static void require(bool value,const char *message) { if (!value) { std::cerr<<"Loader regression: "<<message<<'\n'; std::exit(1); } }
template<typename T> static T symbol(HMODULE module,const char *name) { auto address=GetProcAddress(module,name); T value=nullptr; static_assert(sizeof(value)==sizeof(address)); memcpy(&value,&address,sizeof(value)); return value; }
using GetFactory=HRESULT(STDAPICALLTYPE*)(REFCLSID,REFIID,void**);
using CanUnload=HRESULT(STDAPICALLTYPE*)();
using Configure=void(STDAPICALLTYPE*)(const wchar_t*);
using Version=int(STDAPICALLTYPE*)();
using Cleanup=void(STDAPICALLTYPE*)(BOOL);
static int version(const std::wstring &path) {
    auto module=GetModuleHandleW(path.c_str());
    require(module!=nullptr,"expected core is mapped");
    auto get=symbol<Version>(module,"ArionLoaderFixtureVersion"); require(get!=nullptr,"fixture version export"); return get();
}
int wmain(int count,wchar_t **args) {
    require(count==5,"test loader, two cores and production loader paths required");
    std::wstring root=L"C:\\Program Files\\Arion\\InputMethod",hash;
    const std::wstring valid=root+L"\\"+std::wstring(64,L'a')+L"\\ArionTSF.dll";
    require(arionLoader::corePath(root,valid,hash) && hash==std::wstring(64,L'a'),"exact immutable core layout");
    for (const auto &bad:{valid+L".evil",root+L"\\..\\"+std::wstring(64,L'a')+L"\\ArionTSF.dll",root+L"X\\"+std::wstring(64,L'a')+L"\\ArionTSF.dll",root+L"\\"+std::wstring(64,L'A')+L"\\ArionTSF.dll",std::wstring(args[2])}) require(!arionLoader::corePath(root,bad,hash),"unsafe core layout refused");
    auto module=LoadLibraryExW(args[1],nullptr,LOAD_LIBRARY_SEARCH_DLL_LOAD_DIR|LOAD_LIBRARY_SEARCH_SYSTEM32);
    require(module!=nullptr,"load test-only stable loader");
    HANDLE removal=CreateFileW(args[1],GENERIC_WRITE|DELETE|FILE_READ_ATTRIBUTES,FILE_SHARE_READ|FILE_SHARE_DELETE,nullptr,OPEN_EXISTING,FILE_FLAG_OPEN_REPARSE_POINT,nullptr);
    if (removal!=INVALID_HANDLE_VALUE) CloseHandle(removal);
    require(removal==INVALID_HANDLE_VALUE,"loaded stable loader refuses delete reservation before unregister");
    auto get=symbol<GetFactory>(module,"DllGetClassObject"); auto unload=symbol<CanUnload>(module,"DllCanUnloadNow"); auto configure=symbol<Configure>(module,"ArionLoaderTestSetCorePath");
    require(get && unload && configure,"test-only configuration export");
    IClassFactory *factory=nullptr;
    require(get(ArionService,IID_IClassFactory,reinterpret_cast<void**>(&factory))==S_OK && factory,"cached stable factory");
    ITfTextInputProcessorEx *processor=nullptr;
    require(factory->CreateInstance(nullptr,IID_ITfTextInputProcessorEx,reinterpret_cast<void**>(&processor))==S_OK,"cached stable facade");
    void *unsupported=nullptr;
    require(processor->QueryInterface(IID_ITfThreadMgr,&unsupported)==E_NOINTERFACE && !unsupported,"facade exposes no core internal sinks");
    configure(args[2]); require(processor->ActivateEx(nullptr,1,0)==S_OK && version(args[2])==1,"first core activation");
    auto cleanup=symbol<Cleanup>(GetModuleHandleW(args[2]),"ArionLoaderFixtureCleanup"); require(cleanup!=nullptr,"cleanup hold export");
    cleanup(TRUE);
    configure(args[3]);
    require(unload()==S_FALSE && version(args[2])==1 && !GetModuleHandleW(args[3]),"live old delegate never unloaded");
    require(processor->Deactivate()==S_OK && processor->ActivateEx(nullptr,1,0)==S_OK && version(args[2])==1,"async cleanup still pins old core across activation");
    require(processor->Deactivate()==S_OK,"release second old delegate"); cleanup(FALSE);
    require(processor->ActivateEx(nullptr,1,0)==S_OK && version(args[3])==2 && !GetModuleHandleW(args[2]),"same cached facade and factory switch after idle");
    require(processor->Deactivate()==S_OK,"release current delegate");
    require(factory->LockServer(TRUE)==S_OK,"stable server lock");
    configure(args[2]); require(processor->ActivateEx(nullptr,1,0)==S_OK && version(args[3])==2,"server lock prevents switching an idle core");
    require(processor->Deactivate()==S_OK && factory->LockServer(FALSE)==S_OK,"unlock current core");
    require(processor->ActivateEx(nullptr,1,0)==S_OK && version(args[2])==1 && !GetModuleHandleW(args[3]),"unlock permits next activation to switch");
    require(processor->Deactivate()==S_OK,"final deactivate");
    configure(L"Z:\\arion-owned-missing-fixture\\ArionTSF.dll");
    require(FAILED(processor->ActivateEx(nullptr,1,0)),"missing new implementation fails explicitly");
    require(unload()==S_FALSE,"cached facade/factory retain loader while idle core is freed");
    require(!GetModuleHandleW(args[2]),"idle core released independently of cached facade");
    processor->Release(); factory->Release(); require(unload()==S_OK,"all owned objects released"); FreeLibrary(module);
    module=LoadLibraryExW(args[4],nullptr,LOAD_LIBRARY_SEARCH_DLL_LOAD_DIR|LOAD_LIBRARY_SEARCH_SYSTEM32);
    require(module && !GetProcAddress(module,"ArionLoaderTestSetCorePath"),"production loader has no unsafe configuration export");
    get=symbol<GetFactory>(module,"DllGetClassObject"); unload=symbol<CanUnload>(module,"DllCanUnloadNow");
    factory=nullptr; processor=nullptr;
    require(get(ArionService,IID_IClassFactory,reinterpret_cast<void**>(&factory))==S_OK && factory->CreateInstance(nullptr,IID_ITfTextInputProcessorEx,reinterpret_cast<void**>(&processor))==S_OK,"production facade construction does not load untrusted code");
    require(processor->ActivateEx(nullptr,1,0)==E_ACCESSDENIED,"production loader refuses to run from user workspace");
    processor->Release(); factory->Release(); require(unload()==S_OK,"production failure releases facade"); FreeLibrary(module);
    std::cout<<"PASS: cached factory/facade reload, live-object and async cleanup pinning, server locks, explicit load failure, trusted path policy, production test-hook absence. No registration changed.\n";
    return 0;
}
