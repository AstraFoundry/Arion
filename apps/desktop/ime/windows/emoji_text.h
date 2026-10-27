#pragma once
#include <windows.h>
#include <dwrite_3.h>
#include <d2d1.h>
#include <cmath>
#include <string>
#include <unordered_map>

// Shape and draw the installed Windows Emoji font. No font or artwork is
// redistributed. Unsupported/newer sequences never become selectable rows.
class WindowsEmojiFont {
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
        if (SUCCEEDED(hr)) hr=collection->FindFamilyName(L"Segoe UI Emoji",&index,&found);
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
        hr=factory->CreateTextFormat(L"Segoe UI Emoji",nullptr,DWRITE_FONT_WEIGHT_NORMAL,DWRITE_FONT_STYLE_NORMAL,DWRITE_FONT_STRETCH_NORMAL,size,L"",&format);
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

// Direct2D's DC target is the supported interop boundary for the existing GDI
// candidate surface. Only typed Emoji rows use color-font drawing.
class WindowsEmojiPainter {
    ID2D1Factory *factory=nullptr;
    ID2D1DCRenderTarget *target=nullptr;
    ID2D1SolidColorBrush *brush=nullptr;
public:
    ~WindowsEmojiPainter() { if (brush) brush->Release(); if (target) target->Release(); if (factory) factory->Release(); }
    WindowsEmojiPainter()=default;
    WindowsEmojiPainter(const WindowsEmojiPainter&)=delete;
    WindowsEmojiPainter &operator=(const WindowsEmojiPainter&)=delete;
    HRESULT initialize() {
        HRESULT hr=S_OK;
        if (!factory) hr=D2D1CreateFactory(D2D1_FACTORY_TYPE_SINGLE_THREADED,&factory);
        if (FAILED(hr)) return hr;
        if (!target) {
            auto properties=D2D1::RenderTargetProperties(D2D1_RENDER_TARGET_TYPE_SOFTWARE,D2D1::PixelFormat(DXGI_FORMAT_B8G8R8A8_UNORM,D2D1_ALPHA_MODE_IGNORE),96,96);
            hr=factory->CreateDCRenderTarget(&properties,&target);
        }
        if (SUCCEEDED(hr) && !brush) hr=target->CreateSolidColorBrush(D2D1::ColorF(D2D1::ColorF::Black),&brush);
        return hr;
    }
    HRESULT begin(HDC dc,const RECT &area) {
        HRESULT hr=initialize();
        if (SUCCEEDED(hr)) hr=target->BindDC(dc,&area);
        if (SUCCEEDED(hr)) target->BeginDraw();
        return hr;
    }
    void draw(IDWriteTextLayout *layout,FLOAT x,FLOAT y,COLORREF foreground) {
        brush->SetColor(D2D1::ColorF(GetRValue(foreground)/255.0f,GetGValue(foreground)/255.0f,GetBValue(foreground)/255.0f));
        target->DrawTextLayout(D2D1::Point2F(x,y),layout,brush,static_cast<D2D1_DRAW_TEXT_OPTIONS>(D2D1_DRAW_TEXT_OPTIONS_CLIP|D2D1_DRAW_TEXT_OPTIONS_ENABLE_COLOR_FONT));
    }
    HRESULT end() {
        HRESULT hr=target->EndDraw();
        if (hr==D2DERR_RECREATE_TARGET) { brush->Release(); brush=nullptr; target->Release(); target=nullptr; }
        return hr;
    }
};
