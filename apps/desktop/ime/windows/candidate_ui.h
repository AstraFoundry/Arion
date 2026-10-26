#pragma once
#include <msctf.h>
#include <atomic>
#include <string>
#include <vector>
#include "lifetime.h"

// MinGW exposes ITfUIElementMgr but omits this documented derived interface.
#ifndef __ITfCandidateListUIElement_INTERFACE_DEFINED__
#define __ITfCandidateListUIElement_INTERFACE_DEFINED__
DEFINE_GUID(IID_ITfCandidateListUIElement,0xea1ea138,0x19df,0x11d7,0xa6,0xd2,0x00,0x06,0x5b,0x84,0x43,0x5c);
struct ITfCandidateListUIElement : ITfUIElement {
    virtual HRESULT STDMETHODCALLTYPE GetUpdatedFlags(DWORD*)=0;
    virtual HRESULT STDMETHODCALLTYPE GetDocumentMgr(ITfDocumentMgr**)=0;
    virtual HRESULT STDMETHODCALLTYPE GetCount(UINT*)=0;
    virtual HRESULT STDMETHODCALLTYPE GetSelection(UINT*)=0;
    virtual HRESULT STDMETHODCALLTYPE GetString(UINT,BSTR*)=0;
    virtual HRESULT STDMETHODCALLTYPE GetPageIndex(UINT*,UINT,UINT*)=0;
    virtual HRESULT STDMETHODCALLTYPE SetPageIndex(UINT*,UINT)=0;
    virtual HRESULT STDMETHODCALLTYPE GetCurrentPage(UINT*)=0;
};
#endif

// A snapshot owned on the editor's apartment. Assistive clients see the exact
// displayed ordering, and UI-less TSF hosts may suppress the native surface.
class CandidateUI final : public ITfCandidateListUIElement {
    CoreObjectLease lifetime;
    std::atomic<ULONG> references{1};
    ITfUIElementMgr *manager=nullptr;
    ITfDocumentMgr *document=nullptr;
    DWORD id=static_cast<DWORD>(-1);
    HWND window=nullptr;
    bool shown=true;
    std::vector<std::wstring> texts;
    UINT selection=0,pageSize=9,currentPage=0;
public:
    ~CandidateUI() { hide(); if (manager) manager->Release(); }
    void attach(ITfThreadMgr *value) {
        hide(); if (manager) { manager->Release(); manager=nullptr; }
        if (value) value->QueryInterface(IID_ITfUIElementMgr,reinterpret_cast<void**>(&manager));
    }
    void hide() {
        if (manager && id!=static_cast<DWORD>(-1)) { DWORD previous=id; id=static_cast<DWORD>(-1); manager->EndUIElement(previous); }
        if (document) { document->Release(); document=nullptr; }
        texts.clear(); window=nullptr; shown=true;
    }
    bool update(ITfContext *context,const std::vector<std::wstring> &values,size_t selected,size_t perPage,size_t page) {
        texts=values; selection=static_cast<UINT>(selected); pageSize=static_cast<UINT>(perPage); currentPage=static_cast<UINT>(page);
        if (document) { document->Release(); document=nullptr; }
        context->GetDocumentMgr(&document);
        if (manager) {
            if (id==static_cast<DWORD>(-1)) { BOOL native=TRUE; DWORD next=0; if (SUCCEEDED(manager->BeginUIElement(this,&native,&next))) { id=next; shown=native!=FALSE; } }
            else manager->UpdateUIElement(id);
        }
        return shown;
    }
    void bind(HWND value) { window=value; }
    bool nativeAllowed() const { return shown; }
    HRESULT STDMETHODCALLTYPE QueryInterface(REFIID iid,void **out) override { if (!out) return E_POINTER; *out=nullptr; if (iid!=IID_IUnknown && iid!=IID_ITfUIElement && iid!=IID_ITfCandidateListUIElement) return E_NOINTERFACE; *out=static_cast<ITfCandidateListUIElement*>(this); AddRef(); return S_OK; }
    ULONG STDMETHODCALLTYPE AddRef() override { return ++references; }
    ULONG STDMETHODCALLTYPE Release() override { auto count=--references; if (!count) delete this; return count; }
    HRESULT STDMETHODCALLTYPE GetDescription(BSTR *value) override { if (!value) return E_POINTER; *value=SysAllocString(L"Arion"); return *value?S_OK:E_OUTOFMEMORY; }
    HRESULT STDMETHODCALLTYPE GetGUID(GUID *value) override { if (!value) return E_POINTER; *value={0xb4919358,0x96f2,0x42c7,{0xb9,0x6b,0x68,0xf6,0x08,0xc2,0xc8,0x14}}; return S_OK; }
    HRESULT STDMETHODCALLTYPE Show(BOOL value) override { shown=value!=FALSE; if (window && IsWindow(window)) ShowWindow(window,shown?SW_SHOWNOACTIVATE:SW_HIDE); return S_OK; }
    HRESULT STDMETHODCALLTYPE IsShown(BOOL *value) override { if (!value) return E_POINTER; *value=shown; return S_OK; }
    HRESULT STDMETHODCALLTYPE GetUpdatedFlags(DWORD *value) override { if (!value) return E_POINTER; *value=0x3f; return S_OK; }
    HRESULT STDMETHODCALLTYPE GetDocumentMgr(ITfDocumentMgr **value) override { if (!value) return E_POINTER; *value=document; if (document) document->AddRef(); return document?S_OK:E_FAIL; }
    HRESULT STDMETHODCALLTYPE GetCount(UINT *value) override { if (!value) return E_POINTER; *value=static_cast<UINT>(texts.size()); return S_OK; }
    HRESULT STDMETHODCALLTYPE GetSelection(UINT *value) override { if (!value) return E_POINTER; *value=selection; return S_OK; }
    HRESULT STDMETHODCALLTYPE GetString(UINT index,BSTR *value) override { if (!value) return E_POINTER; *value=nullptr; if (index>=texts.size()) return E_INVALIDARG; *value=SysAllocStringLen(texts[index].data(),static_cast<UINT>(texts[index].size())); return *value?S_OK:E_OUTOFMEMORY; }
    HRESULT STDMETHODCALLTYPE GetPageIndex(UINT *values,UINT capacity,UINT *count) override {
        if (!count) return E_POINTER;
        *count=pageSize?static_cast<UINT>((texts.size()+pageSize-1)/pageSize):0;
        if (capacity<*count) return HRESULT_FROM_WIN32(ERROR_INSUFFICIENT_BUFFER);
        if (*count && !values) return E_POINTER;
        for (UINT i=0;i<*count;++i) values[i]=i*pageSize;
        return S_OK;
    }
    HRESULT STDMETHODCALLTYPE SetPageIndex(UINT *values,UINT count) override {
        if (!count || !values || values[0]) return E_INVALIDARG;
        for (UINT i=0;i<count;++i) if (values[i]!=i*pageSize || values[i]>=texts.size()) return E_INVALIDARG;
        return S_OK;
    }
    HRESULT STDMETHODCALLTYPE GetCurrentPage(UINT *value) override { if (!value) return E_POINTER; *value=currentPage; return S_OK; }
};
