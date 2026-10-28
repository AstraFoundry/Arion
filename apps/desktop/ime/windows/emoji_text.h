#pragma once
#include <windows.h>
#include <dwrite_3.h>
#include <d2d1.h>
#include <cmath>
#include <string>
#include <unordered_map>

// Shape and validate the installed Windows Emoji font. No font or artwork is
// redistributed. Unsupported/newer sequences never become selectable rows.
class WindowsEmojiFont {
    static constexpr const wchar_t *family=L"Segoe UI Emoji";
    IDWriteFactory4 *factory=nullptr;
    std::unordered_map<std::wstring,bool> support;

    class GlyphProbe final : public IDWriteTextRenderer {
        IDWriteFactory4 *factory;
    public:
        UINT32 glyphs=0;
        bool valid=true,color=false;
        explicit GlyphProbe(IDWriteFactory4 *value):factory(value) {}
        HRESULT STDMETHODCALLTYPE QueryInterface(REFIID id,void **out) override {
            if (!out) return E_POINTER;
            *out=nullptr;
            if (id==IID_IUnknown || id==__uuidof(IDWriteTextRenderer) || id==__uuidof(IDWritePixelSnapping)) {
                *out=static_cast<IDWriteTextRenderer*>(this); AddRef(); return S_OK;
            }
            return E_NOINTERFACE;
        }
        ULONG STDMETHODCALLTYPE AddRef() override { return 1; }
        ULONG STDMETHODCALLTYPE Release() override { return 1; }
        HRESULT STDMETHODCALLTYPE IsPixelSnappingDisabled(void*,BOOL *out) override { if (!out) return E_POINTER; *out=TRUE; return S_OK; }
        HRESULT STDMETHODCALLTYPE GetCurrentTransform(void*,DWRITE_MATRIX *out) override { if (!out) return E_POINTER; *out={1,0,0,1,0,0}; return S_OK; }
        HRESULT STDMETHODCALLTYPE GetPixelsPerDip(void*,FLOAT *out) override { if (!out) return E_POINTER; *out=1; return S_OK; }
        HRESULT STDMETHODCALLTYPE DrawGlyphRun(void*,FLOAT x,FLOAT y,DWRITE_MEASURING_MODE mode,const DWRITE_GLYPH_RUN *run,const DWRITE_GLYPH_RUN_DESCRIPTION *description,IUnknown*) override {
            if (!run) return E_INVALIDARG;
            glyphs+=run->glyphCount;
            for (UINT32 i=0;i<run->glyphCount;++i) if (!run->glyphIndices[i]) valid=false;
            IDWriteColorGlyphRunEnumerator1 *layers=nullptr;
            const auto formats=static_cast<DWRITE_GLYPH_IMAGE_FORMATS>(DWRITE_GLYPH_IMAGE_FORMATS_COLR|DWRITE_GLYPH_IMAGE_FORMATS_SVG|DWRITE_GLYPH_IMAGE_FORMATS_PNG|DWRITE_GLYPH_IMAGE_FORMATS_JPEG|DWRITE_GLYPH_IMAGE_FORMATS_TIFF|DWRITE_GLYPH_IMAGE_FORMATS_PREMULTIPLIED_B8G8R8A8);
            HRESULT hr=factory->TranslateColorGlyphRun(D2D1_POINT_2F{x,y},run,description,formats,mode,nullptr,0,&layers);
            if (hr==DWRITE_E_NOCOLOR) return S_OK;
            if (FAILED(hr)) return hr;
            BOOL present=FALSE;
            hr=layers->MoveNext(&present);
            color=color || (SUCCEEDED(hr) && present);
            layers->Release(); return hr;
        }
        HRESULT STDMETHODCALLTYPE DrawUnderline(void*,FLOAT,FLOAT,const DWRITE_UNDERLINE*,IUnknown*) override { return E_NOTIMPL; }
        HRESULT STDMETHODCALLTYPE DrawStrikethrough(void*,FLOAT,FLOAT,const DWRITE_STRIKETHROUGH*,IUnknown*) override { return E_NOTIMPL; }
        HRESULT STDMETHODCALLTYPE DrawInlineObject(void*,FLOAT,FLOAT,IDWriteInlineObject*,BOOL,BOOL,IUnknown*) override { return E_NOTIMPL; }
    };
public:
    ~WindowsEmojiFont() { if (factory) factory->Release(); }
    WindowsEmojiFont()=default;
    WindowsEmojiFont(const WindowsEmojiFont&)=delete;
    WindowsEmojiFont &operator=(const WindowsEmojiFont&)=delete;
    HRESULT initialize() {
        if (factory) return S_OK;
        IDWriteFactory4 *created=nullptr;
        HRESULT hr=DWriteCreateFactory(DWRITE_FACTORY_TYPE_SHARED,__uuidof(IDWriteFactory4),reinterpret_cast<IUnknown**>(&created));
        if (FAILED(hr)) return hr;
        IDWriteFontCollection *collection=nullptr;
        hr=static_cast<IDWriteFactory*>(created)->GetSystemFontCollection(&collection,FALSE);
        UINT32 index=0; BOOL found=FALSE;
        if (SUCCEEDED(hr)) hr=collection->FindFamilyName(family,&index,&found);
        if (collection) collection->Release();
        if (SUCCEEDED(hr) && !found) hr=DWRITE_E_NOFONT;
        if (FAILED(hr)) { created->Release(); return hr; }
        factory=created; return S_OK;
    }
    HRESULT layout(const std::wstring &text,FLOAT size,IDWriteTextLayout **out) {
        if (!out) return E_POINTER;
        *out=nullptr;
        if (text.empty() || text.size()>128 || !std::isfinite(size) || size<=0) return E_INVALIDARG;
        HRESULT hr=initialize();
        if (FAILED(hr)) return hr;
        IDWriteTextFormat *format=nullptr;
        hr=factory->CreateTextFormat(family,nullptr,DWRITE_FONT_WEIGHT_NORMAL,DWRITE_FONT_STYLE_NORMAL,DWRITE_FONT_STRETCH_NORMAL,size,L"",&format);
        if (SUCCEEDED(hr)) hr=format->SetWordWrapping(DWRITE_WORD_WRAPPING_NO_WRAP);
        if (SUCCEEDED(hr)) hr=factory->CreateTextLayout(text.data(),static_cast<UINT32>(text.size()),format,4096,4096,out);
        if (format) format->Release();
        return hr;
    }
    HRESULT supports(const std::wstring &text,bool &out) {
        out=false;
        auto known=support.find(text);
        if (known!=support.end()) { out=known->second; return S_OK; }
        IDWriteTextLayout *shaped=nullptr;
        HRESULT hr=layout(text,16,&shaped);
        if (FAILED(hr)) return hr;
        UINT32 count=0;
        DWRITE_CLUSTER_METRICS cluster{};
        hr=shaped->GetClusterMetrics(&cluster,1,&count);
        if (hr==HRESULT_FROM_WIN32(ERROR_INSUFFICIENT_BUFFER)) hr=S_OK;
        if (SUCCEEDED(hr) && count==1 && cluster.length==text.size()) {
            GlyphProbe probe(factory);
            hr=shaped->Draw(nullptr,&probe,0,0);
            // A single base glyph, one complete cluster and native color layers
            // exclude tofu, regional-letter flags and decomposed ZWJ sequences.
            out=SUCCEEDED(hr) && probe.valid && probe.color && probe.glyphs==1;
        }
        shaped->Release();
        if (SUCCEEDED(hr)) support.emplace(text,out);
        return hr;
    }
    HRESULT supportsUTF8(const char *text,bool &out) {
        out=false;
        if (!text) return E_INVALIDARG;
        int count=MultiByteToWideChar(CP_UTF8,MB_ERR_INVALID_CHARS,text,-1,nullptr,0);
        if (count<2 || count>129) return E_INVALIDARG;
        std::wstring value(static_cast<size_t>(count),L'\0');
        if (!MultiByteToWideChar(CP_UTF8,MB_ERR_INVALID_CHARS,text,-1,value.data(),count)) return HRESULT_FROM_WIN32(GetLastError());
        value.pop_back(); return supports(value,out);
    }
};
