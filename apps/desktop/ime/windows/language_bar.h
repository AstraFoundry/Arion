#pragma once
#include <ctfutb.h>
#include <olectl.h>
#include <string>
#include <functional>
#include <vector>
#include "identity.h"
#include "lifetime.h"
#include "native_copy.h"

// ABI declarations absent from MinGW's ctfutb.h; the Microsoft SDK defines
// the same ITfLangBarItem-derived button and five additional methods.
#ifndef __ITfLangBarItemButton_INTERFACE_DEFINED__
#define __ITfLangBarItemButton_INTERFACE_DEFINED__
DEFINE_GUID(IID_ITfLangBarItemButton,0x28c7f1d0,0xde25,0x11d2,0xaf,0xdd,0x00,0x10,0x5a,0x27,0x99,0xb5);
struct ITfMenu;
enum TfLBIClick { TF_LBI_CLK_RIGHT=1,TF_LBI_CLK_LEFT=2 };
struct ITfLangBarItemButton : ITfLangBarItem {
    virtual HRESULT STDMETHODCALLTYPE OnClick(TfLBIClick,POINT,const RECT*)=0;
    virtual HRESULT STDMETHODCALLTYPE InitMenu(ITfMenu*)=0;
    virtual HRESULT STDMETHODCALLTYPE OnMenuSelect(UINT)=0;
    virtual HRESULT STDMETHODCALLTYPE GetIcon(HICON*)=0;
    virtual HRESULT STDMETHODCALLTYPE GetText(BSTR*)=0;
};
#endif
#ifndef __ITfMenu_INTERFACE_DEFINED__
#define __ITfMenu_INTERFACE_DEFINED__
DEFINE_GUID(IID_ITfMenu,0x6f8a98e4,0xaaa0,0x4f15,0x8c,0x5b,0x07,0xe0,0xdf,0x0a,0x3d,0xd8);
#define TF_LBMENUF_SEPARATOR 0x4
#define TF_LBMENUF_GRAYED 0x10
struct ITfMenu : IUnknown {
    virtual HRESULT STDMETHODCALLTYPE AddMenuItem(UINT,DWORD,HBITMAP,HBITMAP,const WCHAR*,ULONG,ITfMenu**)=0;
};
#endif
#ifndef TF_LBI_ICON
#define TF_LBI_ICON 0x1
#define TF_LBI_TEXT 0x2
#define TF_LBI_TOOLTIP 0x4
#define TF_LBI_STATUS 0x10000
#define TF_LBI_STYLE_HIDDENSTATUSCONTROL 0x1
#define TF_LBI_STYLE_BTN_BUTTON 0x10000
#define TF_LBI_STYLE_BTN_MENU 0x20000
#define TF_LBI_STATUS_HIDDEN 0x1
#endif
// GUID_LBI_INPUTMODE from the Windows SDK; MinGW omits its declaration.
inline constexpr GUID ArionInputModeItem={0x2c77a81e,0x41cc,0x4178,{0xa3,0xa7,0x5f,0x8a,0x98,0x75,0x68,0xe6}};
enum class KeyboardMenuAction { Voice=1,Language,Width,Punctuation,CopyPending,Reset,OpenApp };
struct KeyboardMenuState {
    bool available=false,ascii=false,fullWidth=false,englishPunctuation=false,voice=false;
    std::wstring failure,dictationKey;
    bool pending=false;
};
class LanguageBar final : public ITfLangBarItemButton,public ITfSource {
    CoreObjectLease lifetime;
    std::atomic<ULONG> references{1};
    ITfLangBarItemMgr *manager=nullptr;
    ITfLangBarItemSink *sink=nullptr;
    bool hidden=false;
    std::function<KeyboardMenuState()> state;
    std::function<void(KeyboardMenuAction)> command;
    static HRESULT string(const wchar_t *text,BSTR *output) { if (!output) return E_POINTER; *output=SysAllocString(text); return *output?S_OK:E_OUTOFMEMORY; }
    struct MenuItem { UINT id; DWORD flags; std::wstring text; };
    static std::vector<MenuItem> menuItems(const KeyboardMenuState &current) {
        std::vector<MenuItem> items;
        auto item=[&](KeyboardMenuAction action,std::wstring text,bool enabled=true) {
            items.push_back({static_cast<UINT>(action),enabled?0u:TF_LBMENUF_GRAYED,std::move(text)});
        };
        auto separator=[&]() { items.push_back({0,TF_LBMENUF_SEPARATOR,{}}); };
        item(KeyboardMenuAction::Voice,nativeVoiceTitle(current.dictationKey),current.voice);
        separator();
        if (!current.failure.empty()) { items.push_back({0,TF_LBMENUF_GRAYED,current.failure}); separator(); }
        item(KeyboardMenuAction::Language,nativeCopy(current.available?(current.ascii?NativeCopy::SwitchChinese:NativeCopy::SwitchEnglish):NativeCopy::Unavailable),current.available);
        item(KeyboardMenuAction::Width,nativeCopy(current.fullWidth?NativeCopy::HalfWidth:NativeCopy::FullWidth),current.available);
        item(KeyboardMenuAction::Punctuation,nativeCopy(current.englishPunctuation?NativeCopy::ChinesePunctuation:NativeCopy::EnglishPunctuation),current.available);
        separator();
        if (current.pending) item(KeyboardMenuAction::CopyPending,nativeCopy(NativeCopy::CopyPending));
        item(KeyboardMenuAction::Reset,nativeCopy(NativeCopy::Reset));
        item(KeyboardMenuAction::OpenApp,nativeCopy(NativeCopy::OpenApp));
        return items;
    }
    static bool enabled(KeyboardMenuAction action,const KeyboardMenuState &current) {
        switch (action) {
        case KeyboardMenuAction::Voice: return current.voice;
        case KeyboardMenuAction::Language: case KeyboardMenuAction::Width: case KeyboardMenuAction::Punctuation: return current.available;
        case KeyboardMenuAction::CopyPending: return current.pending;
        case KeyboardMenuAction::Reset: case KeyboardMenuAction::OpenApp: return true;
        }
        return false;
    }
    void notify(DWORD flags) {
        // A sink may unadvise itself or release its owner during OnUpdate.
        AddRef();
        auto observer=sink;
        if (observer) { observer->AddRef(); observer->OnUpdate(flags); observer->Release(); }
        Release();
    }
public:
    LanguageBar(std::function<KeyboardMenuState()> value,std::function<void(KeyboardMenuAction)> action):state(std::move(value)),command(std::move(action)) {}
    ~LanguageBar() { detach(); if (sink) sink->Release(); }
    HRESULT attach(ITfThreadMgr *thread) { if (!thread) return E_POINTER; if (manager) return E_UNEXPECTED; HRESULT hr=thread->QueryInterface(IID_ITfLangBarItemMgr,reinterpret_cast<void**>(&manager)); if (SUCCEEDED(hr)) { hr=manager->AddItem(this); if (FAILED(hr)) { manager->Release(); manager=nullptr; } } return hr; }
    void detach() { if (manager) { manager->RemoveItem(this); manager->Release(); manager=nullptr; } state={}; command={}; }
    void changed() { notify(TF_LBI_ICON|TF_LBI_TEXT|TF_LBI_TOOLTIP|TF_LBI_STATUS); }
    HRESULT STDMETHODCALLTYPE QueryInterface(REFIID iid,void **out) override {
        if (!out) return E_POINTER;
        *out=nullptr;
        if (iid==IID_IUnknown || iid==IID_ITfLangBarItem || iid==IID_ITfLangBarItemButton) *out=static_cast<ITfLangBarItemButton*>(this);
        else if (iid==IID_ITfSource) *out=static_cast<ITfSource*>(this);
        else return E_NOINTERFACE;
        AddRef(); return S_OK;
    }
    ULONG STDMETHODCALLTYPE AddRef() override { return ++references; }
    ULONG STDMETHODCALLTYPE Release() override { auto count=--references; if (!count) delete this; return count; }
    HRESULT STDMETHODCALLTYPE AdviseSink(REFIID iid,IUnknown *value,DWORD *cookie) override {
        if (!cookie || !value) return E_POINTER;
        *cookie=TF_INVALID_COOKIE;
        if (iid!=IID_ITfLangBarItemSink) return CONNECT_E_CANNOTCONNECT;
        if (sink) return CONNECT_E_ADVISELIMIT;
        HRESULT hr=value->QueryInterface(iid,reinterpret_cast<void**>(&sink)); if (SUCCEEDED(hr)) *cookie=1; return hr;
    }
    HRESULT STDMETHODCALLTYPE UnadviseSink(DWORD cookie) override { if (cookie!=1 || !sink) return CONNECT_E_NOCONNECTION; auto observer=sink; sink=nullptr; observer->Release(); return S_OK; }
    HRESULT STDMETHODCALLTYPE GetInfo(TF_LANGBARITEMINFO *info) override {
        if (!info) return E_POINTER;
        *info={}; info->clsidService=ArionService; info->guidItem=ArionInputModeItem;
        info->dwStyle=TF_LBI_STYLE_BTN_BUTTON|TF_LBI_STYLE_BTN_MENU|TF_LBI_STYLE_HIDDENSTATUSCONTROL; wcscpy_s(info->szDescription,ArionName); return S_OK;
    }
    HRESULT STDMETHODCALLTYPE GetStatus(DWORD *value) override { if (!value) return E_POINTER; *value=hidden?TF_LBI_STATUS_HIDDEN:0; return S_OK; }
    HRESULT STDMETHODCALLTYPE Show(BOOL value) override { bool next=value==FALSE; if (hidden!=next) { hidden=next; notify(TF_LBI_STATUS); } return S_OK; }
    HRESULT STDMETHODCALLTYPE GetTooltipString(BSTR *value) override {
        auto current=state?state():KeyboardMenuState{};
        return string(!current.failure.empty()?current.failure.c_str():!current.available?nativeCopy(NativeCopy::Unavailable):current.ascii?nativeCopy(NativeCopy::SwitchChinese):nativeCopy(NativeCopy::SwitchEnglish),value);
    }
    HRESULT STDMETHODCALLTYPE GetText(BSTR *value) override {
        auto current=state?state():KeyboardMenuState{};
        return string(!current.available || !current.failure.empty()?L"!":nativeCopy(current.ascii?NativeCopy::EnglishBadge:NativeCopy::ChineseBadge),value);
    }
    HRESULT STDMETHODCALLTYPE GetIcon(HICON *value) override {
        if (!value) return E_POINTER;
        *value=nullptr;
        const int width=GetSystemMetrics(SM_CXSMICON),height=GetSystemMetrics(SM_CYSMICON);
        HDC screen=GetDC(nullptr);
        if (!screen) return E_FAIL;
        HDC dc=CreateCompatibleDC(screen),maskDC=CreateCompatibleDC(screen);
        HBITMAP color=CreateCompatibleBitmap(screen,width,height),mask=CreateBitmap(width,height,1,1,nullptr);
        HFONT font=CreateFontW(-height*4/5,0,0,0,FW_MEDIUM,FALSE,FALSE,FALSE,DEFAULT_CHARSET,0,0,ANTIALIASED_QUALITY,0,L"Segoe UI");
        HRESULT hr=E_OUTOFMEMORY;
        if (dc && maskDC && color && mask && font) {
            HGDIOBJ previous=SelectObject(dc,color),oldFont=SelectObject(dc,font),oldMask=SelectObject(maskDC,mask);
            if (previous && previous!=HGDI_ERROR && oldFont && oldFont!=HGDI_ERROR && oldMask && oldMask!=HGDI_ERROR) {
                RECT area{0,0,width,height};
                FillRect(dc,&area,GetSysColorBrush(COLOR_WINDOW)); SetBkMode(dc,TRANSPARENT); SetTextColor(dc,GetSysColor(COLOR_WINDOWTEXT));
                BSTR text=nullptr; hr=GetText(&text);
                if (SUCCEEDED(hr)) {
                    DrawTextW(dc,text,-1,&area,DT_SINGLELINE|DT_CENTER|DT_VCENTER); SysFreeString(text);
                    PatBlt(maskDC,0,0,width,height,BLACKNESS);
                    ICONINFO info{TRUE,0,0,mask,color}; *value=CreateIconIndirect(&info); hr=*value?S_OK:E_FAIL;
                }
            } else hr=E_FAIL;
            if (oldMask && oldMask!=HGDI_ERROR) SelectObject(maskDC,oldMask);
            if (oldFont && oldFont!=HGDI_ERROR) SelectObject(dc,oldFont);
            if (previous && previous!=HGDI_ERROR) SelectObject(dc,previous);
        }
        if (font) DeleteObject(font);
        if (maskDC) DeleteDC(maskDC);
        if (dc) DeleteDC(dc);
        if (mask) DeleteObject(mask);
        if (color) DeleteObject(color);
        ReleaseDC(nullptr,screen);
        return hr;
    }
    HRESULT STDMETHODCALLTYPE OnClick(TfLBIClick click,POINT point,const RECT*) override {
        if (!state || !command) return E_FAIL;
        if (click==TF_LBI_CLK_LEFT) return OnMenuSelect(static_cast<UINT>(KeyboardMenuAction::Language));
        if (click!=TF_LBI_CLK_RIGHT) return E_INVALIDARG;
        auto current=state();
        HMENU menu=CreatePopupMenu();
        if (!menu) return E_FAIL;
        for (const auto &item:menuItems(current)) {
            UINT flags=item.flags&TF_LBMENUF_SEPARATOR?MF_SEPARATOR:MF_STRING|((item.flags&TF_LBMENUF_GRAYED)?MF_GRAYED:MF_ENABLED);
            if (!AppendMenuW(menu,flags,item.id,item.text.c_str())) { DestroyMenu(menu); return E_FAIL; }
        }
        HWND owner=GetFocus(); if (owner) owner=GetAncestor(owner,GA_ROOT);
        if (!owner) { DestroyMenu(menu); return E_FAIL; }
        // Popup tracking pumps messages, including service deactivation.
        AddRef();
        UINT selected=TrackPopupMenuEx(menu,TPM_RETURNCMD|TPM_NONOTIFY|TPM_RIGHTBUTTON,point.x,point.y,owner,nullptr);
        DestroyMenu(menu);
        HRESULT result=selected?OnMenuSelect(selected):S_OK;
        Release(); return result;
    }
    HRESULT STDMETHODCALLTYPE InitMenu(ITfMenu *menu) override {
        if (!menu) return E_POINTER;
        if (!state) return E_FAIL;
        for (const auto &item:menuItems(state())) {
            HRESULT hr=menu->AddMenuItem(item.id,item.flags,nullptr,nullptr,item.text.empty()?nullptr:item.text.c_str(),static_cast<ULONG>(item.text.size()),nullptr);
            if (FAILED(hr)) return hr;
        }
        return S_OK;
    }
    HRESULT STDMETHODCALLTYPE OnMenuSelect(UINT id) override {
        if (id<static_cast<UINT>(KeyboardMenuAction::Voice) || id>static_cast<UINT>(KeyboardMenuAction::OpenApp)) return E_INVALIDARG;
        if (!command || !state) return E_FAIL;
        auto action=static_cast<KeyboardMenuAction>(id);
        if (!enabled(action,state())) return S_FALSE;
        // Copy the callback: dispatch may detach the language bar.
        auto dispatch=command;
        AddRef(); dispatch(action); Release(); return S_OK;
    }
};
