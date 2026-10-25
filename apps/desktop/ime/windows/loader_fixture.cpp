// Owned native core used only by loader_test.cpp. Never registered as a TIP.
#define WIN32_LEAN_AND_MEAN
#include <windows.h>
#include <atomic>
#include "identity.h"
#ifndef ARION_LOADER_FIXTURE_VERSION
#error A fixture version is required
#endif
static std::atomic<ULONG> objects{0};
class Processor final:public ITfTextInputProcessorEx {
    std::atomic<ULONG> refs{1};
public:
    Processor() { ++objects; } ~Processor() { --objects; }
    HRESULT STDMETHODCALLTYPE QueryInterface(REFIID id,void **out) override { if (!out) return E_POINTER; *out=nullptr; if (id!=IID_IUnknown && id!=IID_ITfTextInputProcessor && id!=IID_ITfTextInputProcessorEx) return E_NOINTERFACE; *out=static_cast<ITfTextInputProcessorEx*>(this); AddRef(); return S_OK; }
    ULONG STDMETHODCALLTYPE AddRef() override { return ++refs; }
    ULONG STDMETHODCALLTYPE Release() override { auto count=--refs; if (!count) delete this; return count; }
    HRESULT STDMETHODCALLTYPE Activate(ITfThreadMgr*,TfClientId) override { return S_OK; }
    HRESULT STDMETHODCALLTYPE ActivateEx(ITfThreadMgr*,TfClientId,DWORD) override { return S_OK; }
    HRESULT STDMETHODCALLTYPE Deactivate() override { return S_OK; }
};
class Factory final:public IClassFactory {
    std::atomic<ULONG> refs{1};
public:
    Factory() { ++objects; } ~Factory() { --objects; }
    HRESULT STDMETHODCALLTYPE QueryInterface(REFIID id,void **out) override { if (!out) return E_POINTER; *out=nullptr; if (id!=IID_IUnknown && id!=IID_IClassFactory) return E_NOINTERFACE; *out=static_cast<IClassFactory*>(this); AddRef(); return S_OK; }
    ULONG STDMETHODCALLTYPE AddRef() override { return ++refs; }
    ULONG STDMETHODCALLTYPE Release() override { auto count=--refs; if (!count) delete this; return count; }
    HRESULT STDMETHODCALLTYPE CreateInstance(IUnknown *outer,REFIID id,void **out) override { if (outer) return CLASS_E_NOAGGREGATION; auto value=new Processor; HRESULT hr=value->QueryInterface(id,out); value->Release(); return hr; }
    HRESULT STDMETHODCALLTYPE LockServer(BOOL lock) override { if (lock) ++objects; else --objects; return S_OK; }
};
extern "C" __declspec(dllexport) HRESULT STDAPICALLTYPE DllGetClassObject(REFCLSID clsid,REFIID id,void **out) { if (clsid!=ArionService) return CLASS_E_CLASSNOTAVAILABLE; auto factory=new Factory; HRESULT hr=factory->QueryInterface(id,out); factory->Release(); return hr; }
extern "C" __declspec(dllexport) HRESULT STDAPICALLTYPE DllCanUnloadNow() { return objects.load()?S_FALSE:S_OK; }
extern "C" __declspec(dllexport) int STDAPICALLTYPE ArionLoaderFixtureVersion() { return ARION_LOADER_FIXTURE_VERSION; }
extern "C" __declspec(dllexport) void STDAPICALLTYPE ArionLoaderFixtureCleanup(BOOL held) { if (held) ++objects; else --objects; }
