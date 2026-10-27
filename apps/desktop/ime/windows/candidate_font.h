#pragma once
#include <windows.h>
#include <dwrite_3.h>
#include <string>
#include "candidate_font_resource.h"

inline constexpr wchar_t CandidateFontFamily[]=L"Source Han Sans SC";

// Load only the packaged font bytes; installed fonts cannot replace this face.
class CandidateFont {
    IDWriteFactory5 *factory=nullptr;
    IDWriteInMemoryFontFileLoader *loader=nullptr;
    IDWriteFontCollection1 *collection=nullptr;
    bool registered=false;
public:
    ~CandidateFont() {
        if (collection) collection->Release();
        if (registered) factory->UnregisterFontFileLoader(loader);
        if (loader) loader->Release();
        if (factory) factory->Release();
    }
    CandidateFont()=default;
    CandidateFont(const CandidateFont&)=delete;
    CandidateFont &operator=(const CandidateFont&)=delete;
    HRESULT initialize(HINSTANCE module) {
        if (collection) return S_OK;
        HRSRC resource=FindResourceW(module,MAKEINTRESOURCEW(ARION_SOURCE_HAN_REGULAR),MAKEINTRESOURCEW(10));
        if (!resource) return HRESULT_FROM_WIN32(ERROR_RESOURCE_DATA_NOT_FOUND);
        HGLOBAL loaded=LoadResource(module,resource);
        auto bytes=loaded?LockResource(loaded):nullptr;
        DWORD size=SizeofResource(module,resource);
        if (!bytes || !size) return HRESULT_FROM_WIN32(ERROR_RESOURCE_DATA_NOT_FOUND);
        HRESULT hr=S_OK;
        if (!factory) hr=DWriteCreateFactory(DWRITE_FACTORY_TYPE_SHARED,__uuidof(IDWriteFactory5),reinterpret_cast<IUnknown**>(&factory));
        if (SUCCEEDED(hr) && !loader) hr=factory->CreateInMemoryFontFileLoader(&loader);
        if (SUCCEEDED(hr) && !registered) { hr=factory->RegisterFontFileLoader(loader); registered=SUCCEEDED(hr); }
        IDWriteFontFile *file=nullptr; IDWriteFontSetBuilder1 *builder=nullptr; IDWriteFontSet *set=nullptr;
        if (SUCCEEDED(hr)) hr=loader->CreateInMemoryFontFileReference(factory,bytes,size,nullptr,&file);
        if (SUCCEEDED(hr)) hr=factory->CreateFontSetBuilder(&builder);
        if (SUCCEEDED(hr)) hr=builder->AddFontFile(file);
        if (SUCCEEDED(hr)) hr=builder->CreateFontSet(&set);
        if (SUCCEEDED(hr)) hr=factory->CreateFontCollectionFromFontSet(set,&collection);
        UINT32 index=0; BOOL exists=FALSE;
        if (SUCCEEDED(hr)) hr=collection->FindFamilyName(CandidateFontFamily,&index,&exists);
        if (SUCCEEDED(hr) && !exists) hr=DWRITE_E_NOFONT;
        if (set) set->Release();
        if (builder) builder->Release();
        if (file) file->Release();
        if (FAILED(hr) && collection) { collection->Release(); collection=nullptr; }
        return hr;
    }
    HRESULT createLayout(HINSTANCE module,const std::wstring &value,FLOAT size,FLOAT width,IDWriteTextLayout **out) {
        HRESULT hr=initialize(module);
        IDWriteTextFormat *format=nullptr;
        if (SUCCEEDED(hr)) hr=factory->CreateTextFormat(CandidateFontFamily,collection,DWRITE_FONT_WEIGHT_NORMAL,DWRITE_FONT_STYLE_NORMAL,DWRITE_FONT_STRETCH_NORMAL,size,L"",&format);
        if (SUCCEEDED(hr)) hr=format->SetWordWrapping(DWRITE_WORD_WRAPPING_NO_WRAP);
        if (SUCCEEDED(hr)) hr=factory->CreateTextLayout(value.data(),static_cast<UINT32>(value.size()),format,width,4096,out);
        if (format) format->Release();
        return hr;
    }
    HRESULT ellipsis(IDWriteTextFormat *format,IDWriteInlineObject **out) { return factory->CreateEllipsisTrimmingSign(format,out); }
};
