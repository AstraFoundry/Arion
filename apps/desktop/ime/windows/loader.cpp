#define WIN32_LEAN_AND_MEAN
#include <windows.h>
#include <objbase.h>
#include <atomic>
#include <mutex>
#include <new>
#include "loader.h"

namespace {
using GetFactory=HRESULT(STDAPICALLTYPE*)(REFCLSID,REFIID,void**);
using CanUnload=HRESULT(STDAPICALLTYPE*)();
std::recursive_mutex mutex;
std::atomic<ULONG> factories{0};
HMODULE self=nullptr,core=nullptr;
HANDLE pinned=INVALID_HANDLE_VALUE;
GetFactory getFactory=nullptr;
CanUnload canUnload=nullptr;
std::wstring loaded;
unsigned calls=0,locks=0;
#ifdef ARION_TSF_LOADER_TEST
std::wstring testPath;
#endif
template<typename T> T symbol(HMODULE module,const char *name) {
    auto address=GetProcAddress(module,name); T result=nullptr;
    static_assert(sizeof(result)==sizeof(address)); memcpy(&result,&address,sizeof(result)); return result;
}
HRESULT configured(std::wstring &path) {
#ifdef ARION_TSF_LOADER_TEST
    if (!testPath.empty()) { path=testPath; return S_OK; }
#endif
    std::wstring root,current; HRESULT hr=arionLoader::machineRoot(root);
    wchar_t own[32768]{};
    if (FAILED(hr)) return hr;
    DWORD size=GetModuleFileNameW(self,own,32768);
    if (!size || size>=32768 || !arionLoader::loaderPath(root,own)) return E_ACCESSDENIED;
    hr=arionLoader::protectedPath(root,own);
    if (SUCCEEDED(hr)) hr=arionLoader::readValue(nullptr,current);
    if (hr!=S_OK) return FAILED(hr)?hr:REGDB_E_CLASSNOTREG;
    if (!arionLoader::loaderPath(root,current)) return E_ACCESSDENIED;
    hr=arionLoader::readValue(L"CorePath",path);
    if (hr!=S_OK) return FAILED(hr)?hr:REGDB_E_CLASSNOTREG;
    std::wstring hash; return arionLoader::validateCore(root,path,hash);
}
bool idle() { return !calls && !locks && (!core || canUnload()==S_OK); }
void unload() {
    if (core) FreeLibrary(core);
    if (pinned!=INVALID_HANDLE_VALUE) CloseHandle(pinned);
    core=nullptr; pinned=INVALID_HANDLE_VALUE; getFactory=nullptr; canUnload=nullptr; loaded.clear();
}
HRESULT ensure() {
    std::wstring path; HRESULT hr=configured(path);
    if (FAILED(hr)) return hr;
    if (core && _wcsicmp(path.c_str(),loaded.c_str())==0) return S_OK;
    // A composition or server lock retains its exact implementation. The next
    // activation after it becomes idle observes the newly published CorePath.
    if (core && !idle()) return S_OK;
    HANDLE file=CreateFileW(path.c_str(),GENERIC_READ,FILE_SHARE_READ,nullptr,OPEN_EXISTING,FILE_FLAG_OPEN_REPARSE_POINT,nullptr);
    if (file==INVALID_HANDLE_VALUE) return HRESULT_FROM_WIN32(GetLastError());
    BY_HANDLE_FILE_INFORMATION info{};
    if (!GetFileInformationByHandle(file,&info) || (info.dwFileAttributes&(FILE_ATTRIBUTE_DIRECTORY|FILE_ATTRIBUTE_REPARSE_POINT)) || info.nNumberOfLinks!=1) { CloseHandle(file); return E_ACCESSDENIED; }
#ifndef ARION_TSF_LOADER_TEST
    std::wstring root,hash;
    hr=arionLoader::machineRoot(root);
    if (SUCCEEDED(hr)) hr=arionLoader::validateCore(root,path,hash);
    if (FAILED(hr)) { CloseHandle(file); return hr; }
#endif
    // Keep the file pinned across validation, loading and its module lifetime.
    // Never search the current directory or a user's PATH for dependencies.
    // Release the idle closure before loading another version, so Windows does
    // not resolve equal dependency basenames against the retained old closure.
    unload();
    HMODULE next=LoadLibraryExW(path.c_str(),nullptr,LOAD_LIBRARY_SEARCH_DLL_LOAD_DIR|LOAD_LIBRARY_SEARCH_SYSTEM32);
    if (!next) { DWORD error=GetLastError(); CloseHandle(file); return HRESULT_FROM_WIN32(error); }
    auto factory=symbol<GetFactory>(next,"DllGetClassObject"); auto check=symbol<CanUnload>(next,"DllCanUnloadNow");
    if (!factory || !check) { FreeLibrary(next); CloseHandle(file); return HRESULT_FROM_WIN32(ERROR_PROC_NOT_FOUND); }
    core=next; pinned=file; getFactory=factory; canUnload=check; loaded=path; return S_OK;
}
// Reserve the current module before invoking arbitrary COM methods. Calls may
// reenter the loader; no global mutex is held while the core executes them.
struct Invocation {
    GetFactory factory=nullptr;
    HRESULT result=S_OK;
    explicit Invocation(bool resolve=false) {
        std::lock_guard<std::recursive_mutex> guard(mutex);
        if (resolve) result=ensure();
        if (SUCCEEDED(result)) { ++calls; factory=getFactory; }
    }
    ~Invocation() { if (SUCCEEDED(result)) { std::lock_guard<std::recursive_mutex> guard(mutex); --calls; } }
};
class Processor final:public ITfTextInputProcessorEx {
    std::atomic<ULONG> references{1};
    ITfTextInputProcessorEx *delegate=nullptr;
public:
    Processor() { ++factories; }
    ~Processor() { if (delegate) { Invocation call; delegate->Release(); } --factories; }
    HRESULT STDMETHODCALLTYPE QueryInterface(REFIID id,void **out) override {
        if (!out) return E_POINTER;
        *out=nullptr;
        if (id!=IID_IUnknown && id!=IID_ITfTextInputProcessor && id!=IID_ITfTextInputProcessorEx) return E_NOINTERFACE;
        *out=static_cast<ITfTextInputProcessorEx*>(this); AddRef(); return S_OK;
    }
    ULONG STDMETHODCALLTYPE AddRef() override { return ++references; }
    ULONG STDMETHODCALLTYPE Release() override { auto value=--references; if (!value) delete this; return value; }
    HRESULT STDMETHODCALLTYPE Activate(ITfThreadMgr *manager,TfClientId client) override { return ActivateEx(manager,client,0); }
    HRESULT STDMETHODCALLTYPE ActivateEx(ITfThreadMgr *manager,TfClientId client,DWORD flags) override {
        if (delegate) return E_UNEXPECTED;
        Invocation call(true); if (FAILED(call.result)) return call.result;
        IClassFactory *factory=nullptr;
        HRESULT hr=call.factory(ArionService,IID_IClassFactory,reinterpret_cast<void**>(&factory));
        ITfTextInputProcessorEx *next=nullptr;
        if (hr==S_OK && factory) { hr=factory->CreateInstance(nullptr,IID_ITfTextInputProcessorEx,reinterpret_cast<void**>(&next)); factory->Release(); }
        else if (SUCCEEDED(hr)) hr=E_UNEXPECTED;
        if (hr!=S_OK || !next) { if (next) next->Release(); return FAILED(hr)?hr:E_UNEXPECTED; }
        delegate=next;
        next->AddRef();
        hr=next->ActivateEx(manager,client,flags);
        if (hr!=S_OK && delegate==next) { delegate=nullptr; next->Release(); }
        next->Release(); return hr;
    }
    HRESULT STDMETHODCALLTYPE Deactivate() override {
        if (!delegate) return S_OK;
        Invocation call;
        auto previous=delegate; delegate=nullptr;
        HRESULT hr=previous->Deactivate(); previous->Release(); return hr;
    }
};
class Factory final:public IClassFactory {
    std::atomic<ULONG> references{1};
public:
    Factory() { ++factories; }
    ~Factory() { --factories; }
    HRESULT STDMETHODCALLTYPE QueryInterface(REFIID id,void **out) override { if (!out) return E_POINTER; *out=nullptr; if (id!=IID_IUnknown && id!=IID_IClassFactory) return E_NOINTERFACE; *out=static_cast<IClassFactory*>(this); AddRef(); return S_OK; }
    ULONG STDMETHODCALLTYPE AddRef() override { return ++references; }
    ULONG STDMETHODCALLTYPE Release() override { auto value=--references; if (!value) delete this; return value; }
    HRESULT STDMETHODCALLTYPE CreateInstance(IUnknown *outer,REFIID id,void **out) override {
        if (!out) return E_POINTER;
        *out=nullptr;
        if (outer) return CLASS_E_NOAGGREGATION;
        auto processor=new(std::nothrow) Processor;
        if (!processor) return E_OUTOFMEMORY;
        HRESULT hr=processor->QueryInterface(id,out); processor->Release(); return hr;
    }
    HRESULT STDMETHODCALLTYPE LockServer(BOOL lock) override {
        std::lock_guard<std::recursive_mutex> guard(mutex);
        if (!lock && !locks) return E_UNEXPECTED;
        if (lock) ++locks; else --locks;
        return S_OK;
    }
};
}
extern "C" __declspec(dllexport) HRESULT STDAPICALLTYPE DllGetClassObject(REFCLSID clsid,REFIID id,void **out) {
    if (!out) return E_POINTER;
    *out=nullptr;
    if (clsid!=ArionService) return CLASS_E_CLASSNOTAVAILABLE;
    auto factory=new(std::nothrow) Factory;
    if (!factory) return E_OUTOFMEMORY;
    HRESULT hr=factory->QueryInterface(id,out); factory->Release(); return hr;
}
extern "C" __declspec(dllexport) HRESULT STDAPICALLTYPE DllCanUnloadNow() {
    std::lock_guard<std::recursive_mutex> guard(mutex);
    if (!idle()) return S_FALSE;
    unload(); return factories.load()==0?S_OK:S_FALSE;
}
#ifdef ARION_TSF_LOADER_TEST
extern "C" __declspec(dllexport) void STDAPICALLTYPE ArionLoaderTestSetCorePath(const wchar_t *path) {
    std::lock_guard<std::recursive_mutex> guard(mutex); testPath=path?path:L"";
}
#endif
BOOL WINAPI DllMain(HINSTANCE instance,DWORD reason,LPVOID) {
    if (reason==DLL_PROCESS_ATTACH) { self=instance; DisableThreadLibraryCalls(instance); }
    return TRUE;
}
