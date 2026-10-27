#pragma once
#include "emoji_text.h"

// One premultiplied surface keeps rounded edges smooth against any host color.
class CandidatePainter {
    ID2D1Factory *factory=nullptr;
    IDWriteFactory *textFactory=nullptr;
    ID2D1DCRenderTarget *target=nullptr;
    ID2D1SolidColorBrush *brush=nullptr;
    HDC dc=nullptr;
    HBITMAP bitmap=nullptr;
    HGDIOBJ previous=nullptr;
    SIZE size{};
    void releaseSurface() {
        if (dc && previous) SelectObject(dc,previous);
        if (bitmap) DeleteObject(bitmap);
        if (dc) DeleteDC(dc);
        dc=nullptr; bitmap=nullptr; previous=nullptr;
    }
    static D2D1_COLOR_F color(COLORREF value) {
        return D2D1::ColorF(GetRValue(value)/255.0f,GetGValue(value)/255.0f,GetBValue(value)/255.0f);
    }
    HRESULT createLayout(const std::wstring &value,const wchar_t *family,FLOAT fontSize,FLOAT width,IDWriteTextLayout **out) {
        HRESULT hr=S_OK;
        if (!textFactory) hr=DWriteCreateFactory(DWRITE_FACTORY_TYPE_SHARED,__uuidof(IDWriteFactory),reinterpret_cast<IUnknown**>(&textFactory));
        IDWriteTextFormat *format=nullptr;
        if (SUCCEEDED(hr)) hr=textFactory->CreateTextFormat(family,nullptr,DWRITE_FONT_WEIGHT_NORMAL,DWRITE_FONT_STYLE_NORMAL,DWRITE_FONT_STRETCH_NORMAL,fontSize,L"",&format);
        if (SUCCEEDED(hr)) hr=format->SetWordWrapping(DWRITE_WORD_WRAPPING_NO_WRAP);
        if (SUCCEEDED(hr)) hr=textFactory->CreateTextLayout(value.data(),static_cast<UINT32>(value.size()),format,width,4096,out);
        if (format) format->Release();
        return hr;
    }

public:
    ~CandidatePainter() {
        releaseSurface();
        if (brush) brush->Release();
        if (target) target->Release();
        if (textFactory) textFactory->Release();
        if (factory) factory->Release();
    }
    CandidatePainter()=default;
    CandidatePainter(const CandidatePainter&)=delete;
    CandidatePainter &operator=(const CandidatePainter&)=delete;
    HRESULT measure(const std::wstring &value,const wchar_t *family,FLOAT fontSize,int &width) {
        IDWriteTextLayout *shaped=nullptr;
        HRESULT hr=createLayout(value,family,fontSize,4096,&shaped);
        DWRITE_TEXT_METRICS metrics{};
        if (SUCCEEDED(hr)) hr=shaped->GetMetrics(&metrics);
        if (SUCCEEDED(hr)) width=static_cast<int>(std::ceil(metrics.widthIncludingTrailingWhitespace));
        if (shaped) shaped->Release();
        return hr;
    }
    HRESULT begin(const RECT &area) {
        HRESULT hr=S_OK;
        if (!factory) hr=D2D1CreateFactory(D2D1_FACTORY_TYPE_SINGLE_THREADED,&factory);
        if (SUCCEEDED(hr) && !textFactory) hr=DWriteCreateFactory(DWRITE_FACTORY_TYPE_SHARED,__uuidof(IDWriteFactory),reinterpret_cast<IUnknown**>(&textFactory));
        if (SUCCEEDED(hr) && !target) {
            auto properties=D2D1::RenderTargetProperties(D2D1_RENDER_TARGET_TYPE_SOFTWARE,D2D1::PixelFormat(DXGI_FORMAT_B8G8R8A8_UNORM,D2D1_ALPHA_MODE_PREMULTIPLIED),96,96);
            hr=factory->CreateDCRenderTarget(&properties,&target);
        }
        if (SUCCEEDED(hr) && !brush) hr=target->CreateSolidColorBrush(D2D1::ColorF(0,0,0),&brush);
        if (FAILED(hr)) return hr;
        if (!dc || size.cx!=area.right || size.cy!=area.bottom) {
            releaseSurface(); size={area.right,area.bottom};
            BITMAPINFO info{}; info.bmiHeader.biSize=sizeof(BITMAPINFOHEADER);
            info.bmiHeader.biWidth=size.cx; info.bmiHeader.biHeight=-size.cy;
            info.bmiHeader.biPlanes=1; info.bmiHeader.biBitCount=32; info.bmiHeader.biCompression=BI_RGB;
            dc=CreateCompatibleDC(nullptr);
            if (!dc) return E_OUTOFMEMORY;
            void *pixels=nullptr;
            bitmap=CreateDIBSection(dc,&info,DIB_RGB_COLORS,&pixels,nullptr,0);
            if (!bitmap) { releaseSurface(); return E_OUTOFMEMORY; }
            previous=SelectObject(dc,bitmap);
        }
        hr=target->BindDC(dc,&area);
        if (FAILED(hr)) return hr;
        target->BeginDraw();
        target->SetAntialiasMode(D2D1_ANTIALIAS_MODE_PER_PRIMITIVE);
        target->SetTextAntialiasMode(D2D1_TEXT_ANTIALIAS_MODE_GRAYSCALE);
        target->Clear(D2D1::ColorF(0,0,0,0));
        return S_OK;
    }
    void rounded(const RECT &area,FLOAT radius,COLORREF fill,COLORREF stroke,FLOAT thickness=0) {
        FLOAT inset=thickness/2;
        auto rect=D2D1::RoundedRect(D2D1::RectF(area.left+inset,area.top+inset,area.right-inset,area.bottom-inset),radius,radius);
        brush->SetColor(color(fill)); target->FillRoundedRectangle(rect,brush);
        if (thickness>0) { brush->SetColor(color(stroke)); target->DrawRoundedRectangle(rect,brush,thickness); }
    }
    HRESULT layout(IDWriteTextLayout *value,const RECT &cell,COLORREF foreground,DWRITE_TEXT_ALIGNMENT alignment=DWRITE_TEXT_ALIGNMENT_LEADING) {
        DWRITE_TEXT_METRICS metrics{}; DWRITE_OVERHANG_METRICS ink{};
        HRESULT hr=value->GetMetrics(&metrics);
        if (SUCCEEDED(hr)) hr=value->SetMaxHeight(metrics.height);
        if (SUCCEEDED(hr)) hr=value->GetOverhangMetrics(&ink);
        if (FAILED(hr)) return hr;
        // Center actual visible glyphs, independently of each font's baseline
        // and em-box whitespace. Smaller labels must not sink below Chinese.
        FLOAT x=static_cast<FLOAT>(cell.left);
        if (alignment==DWRITE_TEXT_ALIGNMENT_TRAILING) x=cell.right-metrics.widthIncludingTrailingWhitespace;
        if (alignment==DWRITE_TEXT_ALIGNMENT_CENTER) x=(cell.left+cell.right-metrics.widthIncludingTrailingWhitespace)/2.0f;
        FLOAT y=(cell.top+cell.bottom-(metrics.height+ink.bottom-ink.top))/2.0f;
        brush->SetColor(color(foreground));
        target->PushAxisAlignedClip(D2D1::RectF(static_cast<FLOAT>(cell.left),static_cast<FLOAT>(cell.top),static_cast<FLOAT>(cell.right),static_cast<FLOAT>(cell.bottom)),D2D1_ANTIALIAS_MODE_ALIASED);
        target->DrawTextLayout(D2D1::Point2F(x,y),value,brush,static_cast<D2D1_DRAW_TEXT_OPTIONS>(D2D1_DRAW_TEXT_OPTIONS_NO_SNAP|D2D1_DRAW_TEXT_OPTIONS_ENABLE_COLOR_FONT));
        target->PopAxisAlignedClip();
        return S_OK;
    }
    HRESULT text(const std::wstring &value,const wchar_t *family,FLOAT fontSize,const RECT &cell,COLORREF foreground,DWRITE_TEXT_ALIGNMENT alignment=DWRITE_TEXT_ALIGNMENT_LEADING) {
        IDWriteTextLayout *shaped=nullptr;
        HRESULT hr=createLayout(value,family,fontSize,static_cast<FLOAT>(cell.right-cell.left),&shaped);
        IDWriteInlineObject *ellipsis=nullptr;
        if (SUCCEEDED(hr)) hr=textFactory->CreateEllipsisTrimmingSign(shaped,&ellipsis);
        DWRITE_TRIMMING trimming{DWRITE_TRIMMING_GRANULARITY_CHARACTER,0,0};
        if (SUCCEEDED(hr)) hr=shaped->SetTrimming(&trimming,ellipsis);
        if (SUCCEEDED(hr)) hr=layout(shaped,cell,foreground,alignment);
        if (ellipsis) ellipsis->Release();
        if (shaped) shaped->Release();
        return hr;
    }
    HRESULT finish(HWND window,HDC copy=nullptr) {
        HRESULT hr=target->EndDraw();
        if (SUCCEEDED(hr) && copy && !BitBlt(copy,0,0,size.cx,size.cy,dc,0,0,SRCCOPY)) hr=HRESULT_FROM_WIN32(GetLastError());
        if (SUCCEEDED(hr) && !copy) {
            RECT bounds{}; GetWindowRect(window,&bounds);
            POINT destination{bounds.left,bounds.top},source{};
            BLENDFUNCTION blend{AC_SRC_OVER,0,255,AC_SRC_ALPHA};
            if (!UpdateLayeredWindow(window,nullptr,&destination,&size,dc,&source,0,&blend,ULW_ALPHA)) hr=HRESULT_FROM_WIN32(GetLastError());
        }
        return hr;
    }
};
