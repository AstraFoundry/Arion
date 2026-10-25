#pragma once
#include <windows.h>
#include <windowsx.h>
#include <algorithm>
#include <cstdint>
#include <functional>
#include <string>
#include <vector>

struct NativeCandidate {
    std::wstring text;
    std::wstring comment;
    size_t engineIndex=0;
};
struct CandidateAction {
    enum Kind { Select,Previous,Next } kind=Select;
    size_t index=0;
    uint64_t revision=0;
};
struct CandidatePalette {
    COLORREF surface=GetSysColor(COLOR_WINDOW);
    COLORREF text=GetSysColor(COLOR_WINDOWTEXT);
    COLORREF muted=GetSysColor(COLOR_GRAYTEXT);
    COLORREF border=GetSysColor(COLOR_3DSHADOW);
    COLORREF selected=RGB(255,204,0);
    COLORREF selectedText=RGB(20,17,15);
};

// Apartment-affine native surface. It never activates or moves keyboard focus;
// click callbacks carry the exact displayed revision and engine candidate ID.
class CandidateWindow {
    HWND window=nullptr;
    HINSTANCE module;
    HFONT font=nullptr;
    UINT dpi=96;
    std::vector<NativeCandidate> items;
    std::vector<RECT> rectangles;
    RECT previous{},next{};
    uint64_t revision=0;
    size_t highlighted=0;
    bool hasPrevious=false,hasNext=false;
    bool numbered=true;
    size_t columns=9;
    int fontSize=16;
    CandidatePalette palette;
    std::function<void(const CandidateAction&)> action;

    int scale(int value) const { return MulDiv(value,static_cast<int>(dpi),96); }
    static LRESULT CALLBACK dispatch(HWND handle,UINT message,WPARAM wp,LPARAM lp) {
        auto self=reinterpret_cast<CandidateWindow*>(GetWindowLongPtrW(handle,GWLP_USERDATA));
        if (message==WM_NCCREATE) { self=static_cast<CandidateWindow*>(reinterpret_cast<CREATESTRUCTW*>(lp)->lpCreateParams); self->window=handle; SetWindowLongPtrW(handle,GWLP_USERDATA,reinterpret_cast<LONG_PTR>(self)); }
        if (!self) return DefWindowProcW(handle,message,wp,lp);
        if (message==WM_MOUSEACTIVATE) return MA_NOACTIVATE;
        if (message==WM_ERASEBKGND) return 1;
        if (message==WM_PAINT) { self->paint(); return 0; }
        if (message==WM_LBUTTONUP) {
            if (!self->numbered) return 0;
            POINT point{GET_X_LPARAM(lp),GET_Y_LPARAM(lp)};
            CandidateAction selected; selected.revision=self->revision;
            bool found=false;
            for (size_t i=0;i<self->rectangles.size();++i) if (PtInRect(&self->rectangles[i],point)) { selected.index=self->items[i].engineIndex; found=true; break; }
            if (!found && self->hasPrevious && PtInRect(&self->previous,point)) { selected.kind=CandidateAction::Previous; found=true; }
            if (!found && self->hasNext && PtInRect(&self->next,point)) { selected.kind=CandidateAction::Next; found=true; }
            if (found && self->action) self->action(selected);
            return 0;
        }
        return DefWindowProcW(handle,message,wp,lp);
    }
    void paint() {
        PAINTSTRUCT paint{}; HDC destination=BeginPaint(window,&paint);
        RECT area{}; GetClientRect(window,&area);
        HDC dc=CreateCompatibleDC(destination); HBITMAP bitmap=CreateCompatibleBitmap(destination,std::max(1L,area.right),std::max(1L,area.bottom));
        auto priorBitmap=SelectObject(dc,bitmap); auto priorFont=SelectObject(dc,font);
        SetBkMode(dc,TRANSPARENT);
        HBRUSH surface=CreateSolidBrush(palette.surface),selection=CreateSolidBrush(palette.selected),border=CreateSolidBrush(palette.border);
        FillRect(dc,&area,surface); FrameRect(dc,&area,border);
        for (size_t i=0;i<items.size();++i) {
            RECT cell=rectangles[i];
            if (i==highlighted) {
                HGDIOBJ oldBrush=SelectObject(dc,selection),oldPen=SelectObject(dc,GetStockObject(NULL_PEN));
                RoundRect(dc,cell.left,cell.top,cell.right,cell.bottom,scale(12),scale(12)); SelectObject(dc,oldPen); SelectObject(dc,oldBrush);
            }
            SetTextColor(dc,i==highlighted ? palette.selectedText : palette.muted);
            RECT number=cell; number.left+=scale(5); number.right=number.left+(numbered?scale(16):0);
            std::wstring label=std::to_wstring(i%columns+1); if (numbered) DrawTextW(dc,label.c_str(),static_cast<int>(label.size()),&number,DT_SINGLELINE|DT_VCENTER|DT_NOPREFIX);
            SetTextColor(dc,i==highlighted ? palette.selectedText : palette.text);
            RECT text=cell; text.left=number.right+scale(3); text.right-=scale(6);
            DrawTextW(dc,items[i].text.c_str(),static_cast<int>(items[i].text.size()),&text,DT_SINGLELINE|DT_VCENTER|DT_END_ELLIPSIS|DT_NOPREFIX);
        }
        SetTextColor(dc,palette.muted);
        if (hasPrevious) DrawTextW(dc,L"\u2039",1,&previous,DT_SINGLELINE|DT_CENTER|DT_VCENTER);
        if (hasNext) DrawTextW(dc,L"\u203a",1,&next,DT_SINGLELINE|DT_CENTER|DT_VCENTER);
        BitBlt(destination,0,0,area.right,area.bottom,dc,0,0,SRCCOPY);
        DeleteObject(border); DeleteObject(selection); DeleteObject(surface);
        SelectObject(dc,priorFont); SelectObject(dc,priorBitmap); DeleteObject(bitmap); DeleteDC(dc); EndPaint(window,&paint);
    }
public:
    explicit CandidateWindow(HINSTANCE instance):module(instance) {}
    ~CandidateWindow() { if (window) DestroyWindow(window); if (font) DeleteObject(font); UnregisterClassW(L"Arion.NativeCandidates",module); }
    CandidateWindow(const CandidateWindow&)=delete;
    CandidateWindow &operator=(const CandidateWindow&)=delete;
    void onAction(std::function<void(const CandidateAction&)> callback) { action=std::move(callback); }
    HWND handle() const { return window; }
    bool visible() const { return window && IsWindowVisible(window); }
    void hide() { if (window) ShowWindow(window,SW_HIDE); items.clear(); rectangles.clear(); revision=0; }
    bool show(HWND owner,const RECT &caret,const std::vector<NativeCandidate> &candidates,size_t selected,bool before,bool after,uint64_t generation,const CandidatePalette &colors,size_t pageSize=9,bool grid=false,int textSize=16,bool numbers=true) {
        if (!owner || !IsWindow(owner) || candidates.empty() || !pageSize || pageSize>9 || candidates.size()>(grid?pageSize*6:pageSize) || caret.bottom<=caret.top || caret.right<caret.left) { hide(); return false; }
        WNDCLASSW definition{}; definition.lpfnWndProc=dispatch; definition.hInstance=module; definition.lpszClassName=L"Arion.NativeCandidates"; definition.hCursor=LoadCursorW(nullptr,MAKEINTRESOURCEW(32512));
        if (!RegisterClassW(&definition) && GetLastError()!=ERROR_CLASS_ALREADY_EXISTS) return false;
        if (!window) window=CreateWindowExW(WS_EX_NOACTIVATE|WS_EX_TOOLWINDOW,definition.lpszClassName,L"Arion",WS_POPUP,0,0,0,0,owner,nullptr,module,this);
        if (!window) return false;
        SetWindowLongPtrW(window,GWLP_HWNDPARENT,reinterpret_cast<LONG_PTR>(owner));
        UINT updated=GetDpiForWindow(owner); if (!updated) updated=96;
        if (!font || updated!=dpi || textSize!=fontSize) {
            fontSize=textSize;
            dpi=updated; if (font) DeleteObject(font);
            NONCLIENTMETRICSW metrics{}; metrics.cbSize=sizeof(metrics); SystemParametersInfoW(SPI_GETNONCLIENTMETRICS,sizeof(metrics),&metrics,0);
            metrics.lfMessageFont.lfHeight=-scale(fontSize); font=CreateFontIndirectW(&metrics.lfMessageFont);
        }
        items=candidates; highlighted=std::min(selected,items.size()-1); hasPrevious=before; hasNext=after; revision=generation; palette=colors; rectangles.clear();
        numbered=numbers;
        SetPropW(window,L"Arion.CandidateSurface",reinterpret_cast<HANDLE>(numbers?1:2));
        columns=grid?pageSize:items.size();
        MONITORINFO monitor{}; monitor.cbSize=sizeof(monitor); if (!GetMonitorInfoW(MonitorFromRect(&caret,MONITOR_DEFAULTTONEAREST),&monitor)) return false;
        HDC dc=GetDC(window); HGDIOBJ old=SelectObject(dc,font);
        int rowHeight=scale(fontSize+16),left=scale(4),arrows=scale((before?20:0)+(after?20:0));
        size_t rows=(items.size()+columns-1)/columns;
        int height=rowHeight*static_cast<int>(rows)+(grid&&arrows?scale(24):0);
        int maximum=std::max(1,static_cast<int>((monitor.rcWork.right-monitor.rcWork.left-scale(8)-scale(5)*static_cast<int>(columns)-(grid?0:arrows))/static_cast<LONG>(columns)));
        std::vector<int> widths(columns,scale(48)),starts(columns,0);
        for (size_t i=0;i<items.size();++i) { SIZE extent{}; GetTextExtentPoint32W(dc,items[i].text.c_str(),static_cast<int>(items[i].text.size()),&extent); widths[i%columns]=std::min(maximum,std::max(widths[i%columns],static_cast<int>(extent.cx)+scale(numbered?36:16))); }
        for (size_t i=0;i<columns;++i) { starts[i]=left; left+=widths[i]+scale(5); }
        for (size_t i=0;i<items.size();++i) { int top=static_cast<int>(i/columns)*rowHeight; rectangles.push_back({starts[i%columns],top+scale(4),starts[i%columns]+widths[i%columns],top+rowHeight-scale(4)}); }
        SelectObject(dc,old); ReleaseDC(window,dc);
        int arrowTop=grid?rowHeight*static_cast<int>(rows):0,arrowBottom=grid?height:rowHeight;
        int arrowLeft=grid?std::max(scale(4),left-arrows):left;
        previous={arrowLeft,arrowTop,arrowLeft+(before?scale(20):0),arrowBottom}; if (before) arrowLeft+=scale(20);
        next={arrowLeft,arrowTop,arrowLeft+(after?scale(20):0),arrowBottom}; if (after) arrowLeft+=scale(20);
        if (!grid) left=arrowLeft;
        int width=left+scale(4),x=std::max(static_cast<int>(monitor.rcWork.left),std::min(static_cast<int>(caret.left),static_cast<int>(monitor.rcWork.right)-width)),y=caret.bottom+scale(4);
        if (y+height>monitor.rcWork.bottom) y=caret.top-height-scale(4);
        y=std::max(static_cast<int>(monitor.rcWork.top),y);
        SetWindowPos(window,HWND_TOPMOST,x,y,width,height,SWP_NOACTIVATE|SWP_SHOWWINDOW); InvalidateRect(window,nullptr,FALSE);
        return true;
    }
};
