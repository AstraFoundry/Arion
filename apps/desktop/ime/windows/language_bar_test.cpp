#include "language_bar.h"
#include <cstdio>
#include <cstdlib>

static void require(bool value,const char *message) {
    if (!value) { std::fprintf(stderr,"FAIL: %s\n",message); std::exit(1); }
}
struct Menu final : ITfMenu {
    struct Entry { UINT id; DWORD flags; std::wstring text; };
    ULONG references=1;
    std::vector<Entry> entries;
    size_t failAt=static_cast<size_t>(-1);
    HRESULT STDMETHODCALLTYPE QueryInterface(REFIID iid,void **out) override {
        if (!out) return E_POINTER;
        *out=nullptr;
        if (iid!=IID_IUnknown && iid!=IID_ITfMenu) return E_NOINTERFACE;
        *out=static_cast<ITfMenu*>(this); AddRef(); return S_OK;
    }
    ULONG STDMETHODCALLTYPE AddRef() override { return ++references; }
    ULONG STDMETHODCALLTYPE Release() override { return --references; }
    HRESULT STDMETHODCALLTYPE AddMenuItem(UINT id,DWORD flags,HBITMAP bitmap,HBITMAP mask,const WCHAR *text,ULONG length,ITfMenu **submenu) override {
        require(!bitmap && !mask && !submenu,"flat menu has no bitmap ownership or submenu");
        if (entries.size()==failAt) return E_OUTOFMEMORY;
        require(text || length==0,"menu text pointer matches length");
        entries.push_back({id,flags,text?std::wstring(text,length):std::wstring{}});
        return S_OK;
    }
};
struct Sink final : ITfLangBarItemSink {
    ULONG references=1;
    std::vector<DWORD> notifications;
    LanguageBar *bar=nullptr;
    bool removeDuringNotify=false;
    bool releaseOwnerDuringNotify=false;
    HRESULT STDMETHODCALLTYPE QueryInterface(REFIID iid,void **out) override {
        if (!out) return E_POINTER;
        *out=nullptr;
        if (iid!=IID_IUnknown && iid!=IID_ITfLangBarItemSink) return E_NOINTERFACE;
        *out=static_cast<ITfLangBarItemSink*>(this); AddRef(); return S_OK;
    }
    ULONG STDMETHODCALLTYPE AddRef() override { return ++references; }
    ULONG STDMETHODCALLTYPE Release() override { return --references; }
    HRESULT STDMETHODCALLTYPE OnUpdate(DWORD flags) override {
        notifications.push_back(flags);
        if (removeDuringNotify) require(bar->UnadviseSink(1)==S_OK,"sink may unadvise during notification");
        if (releaseOwnerDuringNotify) bar->Release();
        return S_OK;
    }
};
static const Menu::Entry &entry(const Menu &menu,KeyboardMenuAction action) {
    for (const auto &item:menu.entries) if (item.id==static_cast<UINT>(action)) return item;
    require(false,"menu action exists"); std::abort();
}
int main() {
    const long before=objects;
    KeyboardMenuState state;
    state.available=true; state.voice=true; state.pending=true; state.dictationKey=L"right_ctrl";
    std::vector<KeyboardMenuAction> actions;
    auto bar=new LanguageBar([&] { return state; },[&](KeyboardMenuAction action) { actions.push_back(action); });
    require(objects==before+1,"COM object lifetime recorded");
    TF_LANGBARITEMINFO info{};
    require(bar->GetInfo(nullptr)==E_POINTER && bar->GetInfo(&info)==S_OK,"GetInfo validates output");
    const GUID inputMode={0x2c77a81e,0x41cc,0x4178,{0xa3,0xa7,0x5f,0x8a,0x98,0x75,0x68,0xe6}};
    require(info.clsidService==ArionService && info.guidItem==inputMode,"system input-mode GUID and stable service identity");
    require(info.dwStyle==(TF_LBI_STYLE_BTN_BUTTON|TF_LBI_STYLE_BTN_MENU|TF_LBI_STYLE_HIDDENSTATUSCONTROL),"supported button, menu and visibility styles only");
    require(std::wstring(info.szDescription)==ArionName,"centralized product name");
    ITfSource *source=nullptr; IUnknown *identity=nullptr,*sourceIdentity=nullptr;
    require(bar->QueryInterface(IID_ITfSource,reinterpret_cast<void**>(&source))==S_OK,"source interface available");
    require(bar->QueryInterface(IID_IUnknown,reinterpret_cast<void**>(&identity))==S_OK,"button IUnknown");
    require(source->QueryInterface(IID_IUnknown,reinterpret_cast<void**>(&sourceIdentity))==S_OK && identity==sourceIdentity,"COM canonical IUnknown identity");
    sourceIdentity->Release(); identity->Release(); source->Release();
    Menu menu;
    require(bar->InitMenu(nullptr)==E_POINTER && bar->InitMenu(&menu)==S_OK,"native TSF menu populated");
    const UINT expected[]={1,0,2,3,4,0,5,6,7};
    require(menu.entries.size()==9,"full menu count");
    for (size_t i=0;i<9;++i) {
        require(menu.entries[i].id==expected[i],"native menu order");
        require(menu.entries[i].flags==(expected[i]?0u:TF_LBMENUF_SEPARATOR),"enabled items and separators");
    }
    require(entry(menu,KeyboardMenuAction::Voice).text==nativeVoiceTitle(state.dictationKey),"localized voice accelerator");
    require(entry(menu,KeyboardMenuAction::Language).text==nativeCopy(NativeCopy::SwitchEnglish),"Chinese mode action");
    state.ascii=true; state.fullWidth=true; state.englishPunctuation=true;
    Menu modes; require(bar->InitMenu(&modes)==S_OK,"alternate mode menu");
    require(entry(modes,KeyboardMenuAction::Language).text==nativeCopy(NativeCopy::SwitchChinese),"English mode action");
    require(entry(modes,KeyboardMenuAction::Width).text==nativeCopy(NativeCopy::HalfWidth),"full-width mode action");
    require(entry(modes,KeyboardMenuAction::Punctuation).text==nativeCopy(NativeCopy::ChinesePunctuation),"punctuation mode action");
    for (UINT id=1;id<=7;++id) require(bar->OnMenuSelect(id)==S_OK,"enabled native action dispatched");
    require(actions.size()==7,"all seven actions forwarded once");
    require(bar->OnMenuSelect(0)==E_INVALIDARG && bar->OnMenuSelect(8)==E_INVALIDARG,"unknown command rejected");
    state.available=false; state.voice=false; state.pending=false; state.failure=nativeCopy(NativeCopy::OptionsFailed);
    Menu unavailable; require(bar->InitMenu(&unavailable)==S_OK,"unavailable menu populated");
    require(unavailable.entries.size()==10,"failure shown and pending command omitted");
    require(unavailable.entries[2].id==0 && unavailable.entries[2].flags==TF_LBMENUF_GRAYED && unavailable.entries[2].text==state.failure,"failure is disabled descriptive row");
    for (UINT id=1;id<=5;++id) require(bar->OnMenuSelect(id)==S_FALSE,"stale or disabled command never dispatched");
    for (auto action:{KeyboardMenuAction::Voice,KeyboardMenuAction::Language,KeyboardMenuAction::Width,KeyboardMenuAction::Punctuation})
        require(entry(unavailable,action).flags==TF_LBMENUF_GRAYED,"disabled action conveyed to TSF");
    require(actions.size()==7,"stale menu dispatch did not run callbacks");
    require(bar->OnMenuSelect(6)==S_OK && bar->OnMenuSelect(7)==S_OK,"recovery and app entry remain accessible");
    require(bar->OnClick(TF_LBI_CLK_LEFT,{},nullptr)==S_FALSE,"unavailable left click does not toggle");
    state.available=true;
    require(bar->OnClick(TF_LBI_CLK_LEFT,{},nullptr)==S_OK && actions.back()==KeyboardMenuAction::Language,"left click toggles through same guarded dispatcher");
    Menu failed; failed.failAt=2;
    require(bar->InitMenu(&failed)==E_OUTOFMEMORY && failed.entries.size()==2,"native menu insertion failure propagated");
    BSTR text=nullptr;
    require(bar->GetText(&text)==S_OK && std::wstring(text)==L"!","failure state badge"); SysFreeString(text);
    require(bar->GetTooltipString(&text)==S_OK && std::wstring(text)==state.failure,"failure tooltip"); SysFreeString(text);
    state.failure.clear();
    require(bar->GetText(&text)==S_OK && std::wstring(text)==nativeCopy(NativeCopy::EnglishBadge),"current mode badge"); SysFreeString(text);
    Sink sink; sink.bar=bar; DWORD cookie=0;
    require(bar->AdviseSink(IID_IUnknown,&sink,&cookie)==CONNECT_E_CANNOTCONNECT && cookie==TF_INVALID_COOKIE,"unsupported sink rejected");
    require(bar->AdviseSink(IID_ITfLangBarItemSink,&sink,&cookie)==S_OK && cookie==1 && sink.references==2,"sink owned once");
    require(bar->AdviseSink(IID_ITfLangBarItemSink,&sink,&cookie)==CONNECT_E_ADVISELIMIT,"second sink rejected");
    DWORD status=99;
    require(bar->Show(FALSE)==S_OK && bar->GetStatus(&status)==S_OK && status==TF_LBI_STATUS_HIDDEN,"Show hides supported status");
    require(sink.notifications.back()==TF_LBI_STATUS,"Show notifies exact status change");
    const size_t count=sink.notifications.size(); bar->Show(FALSE);
    require(sink.notifications.size()==count,"unchanged Show does not notify");
    require(bar->Show(TRUE)==S_OK && bar->GetStatus(&status)==S_OK && status==0,"Show restores visible status");
    bar->changed(); require(sink.notifications.back()==(TF_LBI_ICON|TF_LBI_TEXT|TF_LBI_TOOLTIP|TF_LBI_STATUS),"state invalidates all button properties");
    sink.removeDuringNotify=true; bar->changed();
    require(sink.references==1 && bar->UnadviseSink(1)==CONNECT_E_NOCONNECTION,"reentrant unadvise releases all sink references");
    require(bar->GetIcon(nullptr)==E_POINTER,"icon validates output");
    DWORD gdiBefore=0;
    for (int i=0;i<16;++i) {
        HICON icon=nullptr; require(bar->GetIcon(&icon)==S_OK && icon,"caller receives owned icon");
        ICONINFO bits{}; require(GetIconInfo(icon,&bits)!=FALSE,"icon is valid native GDI object");
        BITMAP bitmap{}; require(GetObjectW(bits.hbmColor,sizeof(bitmap),&bitmap)!=0,"icon color bitmap");
        require(bitmap.bmWidth==GetSystemMetrics(SM_CXSMICON) && bitmap.bmHeight==GetSystemMetrics(SM_CYSMICON),"small-icon system dimensions");
        DeleteObject(bits.hbmColor); DeleteObject(bits.hbmMask); require(DestroyIcon(icon)!=FALSE,"caller destroys icon");
        // The first draw initializes process-local system brush/font caches.
        if (i==0) gdiBefore=GetGuiResources(GetCurrentProcess(),GR_GDIOBJECTS);
    }
    require(GetGuiResources(GetCurrentProcess(),GR_GDIOBJECTS)==gdiBefore,"repeated icons leave no GDI handles");
    bar->detach(); require(bar->OnMenuSelect(7)==E_FAIL && bar->InitMenu(&menu)==E_FAIL,"detached bar cannot dispatch");
    bar->Release(); require(objects==before,"COM object lifetime balanced");
    auto notifying=new LanguageBar([] { return KeyboardMenuState{}; },[](KeyboardMenuAction) {});
    Sink releasing; releasing.bar=notifying; releasing.removeDuringNotify=true; releasing.releaseOwnerDuringNotify=true;
    require(notifying->AdviseSink(IID_ITfLangBarItemSink,&releasing,&cookie)==S_OK,"reentrant owner fixture advised");
    notifying->changed();
    require(objects==before && releasing.references==1,"notification survives final owner release");
    LanguageBar *dispatching=nullptr;
    dispatching=new LanguageBar([] { KeyboardMenuState current; current.available=true; return current; },[&](KeyboardMenuAction) { dispatching->detach(); dispatching->Release(); });
    require(dispatching->OnMenuSelect(2)==S_OK && objects==before,"dispatch survives callback detachment and final owner release");
    std::puts("PASS: language bar COM identity, Windows input-mode metadata, native menu order/disabled/state, stale dispatch, notifications, reentrant sink, owned icons");
    return 0;
}
