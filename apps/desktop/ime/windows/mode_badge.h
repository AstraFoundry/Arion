#pragma once
#include <windows.h>
#include <cmath>
#include <string>
#include "candidate_window.h"

// Opaque, click-through caret badge; only the label flips. Window geometry and
// visibility never animate, and accessibility animation preferences win.
class ModeBadge {
    HINSTANCE module;
    HWND window=nullptr;
    UINT dpi=96;
    ULONGLONG started=0;
    bool animate=false;
    std::wstring before,after;
    CandidatePalette palette;
    int scale(int value) const { return MulDiv(value,static_cast<int>(dpi),96); }
    static LRESULT CALLBACK dispatch(HWND handle,UINT message,WPARAM wp,LPARAM lp) {
        auto self=reinterpret_cast<ModeBadge*>(GetWindowLongPtrW(handle,GWLP_USERDATA));
        if (message==WM_NCCREATE) { self=static_cast<ModeBadge*>(reinterpret_cast<CREATESTRUCTW*>(lp)->lpCreateParams); self->window=handle; SetWindowLongPtrW(handle,GWLP_USERDATA,reinterpret_cast<LONG_PTR>(self)); }
        if (!self) return DefWindowProcW(handle,message,wp,lp);
        if (message==WM_NCHITTEST) return HTTRANSPARENT;
        if (message==WM_MOUSEACTIVATE) return MA_NOACTIVATE;
        if (message==WM_ERASEBKGND) return 1;
        if (message==WM_PAINT) { self->paint(); return 0; }
        if (message==WM_TIMER) {
            if (wp==2) self->hide();
            else { if (GetTickCount64()-self->started>=240) KillTimer(handle,1); InvalidateRect(handle,nullptr,FALSE); }
            return 0;
        }
        return DefWindowProcW(handle,message,wp,lp);
    }
    void paint() {
        PAINTSTRUCT paint{}; HDC destination=BeginPaint(window,&paint); RECT area{}; GetClientRect(window,&area);
        HDC dc=CreateCompatibleDC(destination),label=CreateCompatibleDC(destination);
        HBITMAP bitmap=CreateCompatibleBitmap(destination,area.right,area.bottom),letters=CreateCompatibleBitmap(destination,area.right,area.bottom);
        HGDIOBJ oldBitmap=SelectObject(dc,bitmap),oldLetters=SelectObject(label,letters);
        HBRUSH surface=CreateSolidBrush(palette.surface); FillRect(dc,&area,surface); FillRect(label,&area,surface);
        SetBkMode(label,TRANSPARENT); SetTextColor(label,palette.text);
        HFONT font=CreateFontW(-scale(13),0,0,0,FW_MEDIUM,FALSE,FALSE,FALSE,DEFAULT_CHARSET,0,0,ANTIALIASED_QUALITY,0,L"Segoe UI"); HGDIOBJ oldFont=SelectObject(label,font);
        ULONGLONG elapsed=GetTickCount64()-started; bool old=animate && elapsed<120;
        double factor=1.0;
        if (animate && elapsed<240) factor=old?std::cos(static_cast<double>(elapsed)/120*1.5707963267948966):std::sin(static_cast<double>(elapsed-120)/120*1.5707963267948966);
        const auto &text=old?before:after;
        DrawTextW(label,text.data(),static_cast<int>(text.size()),&area,DT_SINGLELINE|DT_CENTER|DT_VCENTER|DT_NOPREFIX);
        int height=std::max(0,static_cast<int>(std::lround(area.bottom*factor)));
        if (height) { SetStretchBltMode(dc,HALFTONE); StretchBlt(dc,0,(area.bottom-height)/2,area.right,height,label,0,0,area.right,area.bottom,SRCCOPY); }
        BitBlt(destination,0,0,area.right,area.bottom,dc,0,0,SRCCOPY);
        SelectObject(label,oldFont); DeleteObject(font); DeleteObject(surface); SelectObject(label,oldLetters); SelectObject(dc,oldBitmap); DeleteObject(letters); DeleteObject(bitmap); DeleteDC(label); DeleteDC(dc); EndPaint(window,&paint);
    }
public:
    explicit ModeBadge(HINSTANCE value):module(value) {}
    ~ModeBadge() { if (window) DestroyWindow(window); UnregisterClassW(L"Arion.ModeBadge",module); }
    void hide() { if (window) { KillTimer(window,1); KillTimer(window,2); ShowWindow(window,SW_HIDE); } }
    bool show(HWND owner,const RECT &caret,const std::wstring &previous,const std::wstring &next,const CandidatePalette &colors) {
        if (!owner || !IsWindow(owner) || caret.bottom<=caret.top || caret.right<caret.left) { hide(); return false; }
        WNDCLASSW definition{}; definition.lpfnWndProc=dispatch; definition.hInstance=module; definition.lpszClassName=L"Arion.ModeBadge";
        if (!RegisterClassW(&definition) && GetLastError()!=ERROR_CLASS_ALREADY_EXISTS) return false;
        if (!window) window=CreateWindowExW(WS_EX_NOACTIVATE|WS_EX_TOOLWINDOW|WS_EX_TRANSPARENT,definition.lpszClassName,L"Arion",WS_POPUP,0,0,0,0,owner,nullptr,module,this);
        if (!window) return false;
        SetWindowLongPtrW(window,GWLP_HWNDPARENT,reinterpret_cast<LONG_PTR>(owner));
        dpi=GetDpiForWindow(owner); if (!dpi) dpi=96;
        int side=scale(23); SetWindowRgn(window,CreateRoundRectRgn(0,0,side+1,side+1,scale(8),scale(8)),FALSE);
        MONITORINFO monitor{}; monitor.cbSize=sizeof(monitor); if (!GetMonitorInfoW(MonitorFromRect(&caret,MONITOR_DEFAULTTONEAREST),&monitor)) return false;
        int x=std::max(static_cast<int>(monitor.rcWork.left),std::min(static_cast<int>(caret.left),static_cast<int>(monitor.rcWork.right)-side));
        int y=caret.bottom+scale(4); if (y+side>monitor.rcWork.bottom) y=caret.top-side-scale(4); y=std::max(static_cast<int>(monitor.rcWork.top),y);
        before=previous; after=next; palette=colors; started=GetTickCount64();
        BOOL animations=FALSE; animate=SystemParametersInfoW(SPI_GETCLIENTAREAANIMATION,0,&animations,0) && animations && before!=after;
        SetWindowPos(window,HWND_TOPMOST,x,y,side,side,SWP_NOACTIVATE|SWP_SHOWWINDOW); InvalidateRect(window,nullptr,FALSE); UpdateWindow(window);
        if (animate) SetTimer(window,1,15,nullptr); else KillTimer(window,1);
        SetTimer(window,2,1000,nullptr); return true;
    }
};
