#define WIN32_LEAN_AND_MEAN
#include <windows.h>
#include <msctf.h>
#include <textstor.h>
#include <inputscope.h>
#include <olectl.h>
#include <cstring>
#include <algorithm>
#include <atomic>
#include <iostream>
#include <string>
#include <thread>
#include "identity.h"
#include "wire.h"
#include "candidate_ui.h"
#include "language_bar.h"
#include "keyboard_options.h"

// An owned editor, backed by its actual UTF-16 document through Microsoft's
// ITextStoreACP contract. The production DLL acquires TSF locks and edits this
// store; stdout reports the resulting document without substituting results.
static constexpr UINT Command = WM_APP + 1, GrantLock = WM_APP + 2;
template<class T> static void release(T *&value) { if (value) { value->Release(); value = nullptr; } }

class Scope final : public ITfInputScope {
    std::atomic<ULONG> references{1}; InputScope kind;
public:
    explicit Scope(InputScope value):kind(value) {}
    HRESULT STDMETHODCALLTYPE QueryInterface(REFIID id, void **out) override { if (!out) return E_POINTER; *out = nullptr; if (id != IID_IUnknown && id != IID_ITfInputScope) return E_NOINTERFACE; *out = static_cast<ITfInputScope*>(this); AddRef(); return S_OK; }
    ULONG STDMETHODCALLTYPE AddRef() override { return ++references; }
    ULONG STDMETHODCALLTYPE Release() override { ULONG count = --references; if (!count) delete this; return count; }
    HRESULT STDMETHODCALLTYPE GetInputScopes(InputScope **values, UINT *count) override {
        if (!values || !count) return E_POINTER;
        *count = 0; *values = static_cast<InputScope*>(CoTaskMemAlloc(sizeof(InputScope))); if (!*values) return E_OUTOFMEMORY;
        **values = kind; *count = 1; return S_OK;
    }
    HRESULT STDMETHODCALLTYPE GetPhrase(BSTR **values, UINT *count) override { if (!values || !count) return E_POINTER; *values = nullptr; *count = 0; return S_FALSE; }
    HRESULT STDMETHODCALLTYPE GetRegularExpression(BSTR *value) override { if (!value) return E_POINTER; *value = nullptr; return S_FALSE; }
    HRESULT STDMETHODCALLTYPE GetSRGS(BSTR *value) override { if (!value) return E_POINTER; *value = nullptr; return S_FALSE; }
    HRESULT STDMETHODCALLTYPE GetXML(BSTR *value) override { if (!value) return E_POINTER; *value = nullptr; return S_FALSE; }
};

class Store final : public ITextStoreACP, public ITfContextOwnerCompositionSink {
    std::atomic<ULONG> references{1};
    ITextStoreACPSink *sink = nullptr; IUnknown *sinkIdentity = nullptr;
    DWORD sinkMask = 0, lock = 0, pendingLock = 0;
    bool requestedScope = false;
    bool valid(LONG start, LONG end) const { return start >= 0 && end >= start && static_cast<size_t>(end) <= value.size(); }
    bool readable() const { return (lock & TS_LF_READ) != 0; }
    bool writable() const { return (lock & TS_LF_READWRITE) == TS_LF_READWRITE; }
    void selectionChanged() { if (sink && (sinkMask & TS_AS_SEL_CHANGE)) sink->OnSelectionChange(); }
    void textChanged(const TS_TEXTCHANGE &change) { if (sink && (sinkMask & TS_AS_TEXT_CHANGE)) sink->OnTextChange(0, &change); }
    void repaint() { if (window) InvalidateRect(window, nullptr, TRUE); }
public:
    HWND window = nullptr;
    bool password = false, readonly = false, refuseComposition = false, deferLocks = false,zeroCaret=false,noCaret=false,transitory=false,scopeAvailable=true;
    bool layoutMissing=false,layoutClipped=false,delayLayout=false,readLocksOnly=false;
    LONG layoutOffset=0;
    unsigned extentFailures=0,readExtents=0,writeExtents=0;
    InputScope inputScope=IS_DEFAULT;
    std::wstring value = L"A\U0001f642\u4e2dZ";
    LONG start = 3, end = 3;
    unsigned writes = 0, starts = 0, ends = 0, locks = 0, refused = 0, deferred = 0;
    ITfDocumentMgr *document = nullptr; ITfContext *context = nullptr;
    ~Store() { release(sink); release(sinkIdentity); release(context); release(document); }
    HRESULT initialize(ITfThreadMgr *manager, TfClientId client) {
        HRESULT hr = manager->CreateDocumentMgr(&document); TfEditCookie cookie = 0;
        if (SUCCEEDED(hr)) hr = document->CreateContext(client, 0, static_cast<ITextStoreACP*>(this), &context, &cookie);
        if (SUCCEEDED(hr)) hr = document->Push(context);
        return hr;
    }
    void shutdown() { if (document) document->Pop(TF_POPF_ALL); release(context); release(document); }
    void grantPending() {
        if (!sink || lock || deferLocks || !pendingLock) return;
        DWORD requested = pendingLock; pendingLock = 0; HRESULT session = S_OK; RequestLock(requested, &session);
    }
    void userSelection(LONG first, LONG last) { if (lock || !valid(first,last)) return; start = first; end = last; selectionChanged(); repaint(); }
    void userReplace(const std::wstring &text, bool reset = false) {
        if (lock || readonly) return;
        LONG first = reset ? 0 : start, last = reset ? static_cast<LONG>(value.size()) : end;
        TS_TEXTCHANGE change{first,last,first + static_cast<LONG>(text.size())};
        value.replace(first,last-first,text); start = end = change.acpNewEnd;
        textChanged(change); selectionChanged(); repaint();
    }
    void setReadonly(bool next) { readonly = next; if (sink && (sinkMask & TS_AS_STATUS_CHANGE)) sink->OnStatusChange(next ? TS_SD_READONLY : 0); }
    void setScopeAvailable(bool value) { scopeAvailable=value; if (sink) { TS_ATTRID attribute=GUID_PROP_INPUTSCOPE; sink->OnAttrsChange(0,static_cast<LONG>(this->value.size()),1,&attribute); } }
    void layoutReady() { layoutMissing=false; if (sink && (sinkMask & TS_AS_LAYOUT_CHANGE)) sink->OnLayoutChange(TS_LC_CHANGE,1); }
    HRESULT STDMETHODCALLTYPE QueryInterface(REFIID id, void **out) override {
        if (!out) return E_POINTER;
        *out = nullptr;
        if (id == IID_IUnknown || id == IID_ITextStoreACP) *out = static_cast<ITextStoreACP*>(this);
        else if (id == IID_ITfContextOwnerCompositionSink) *out = static_cast<ITfContextOwnerCompositionSink*>(this);
        else return E_NOINTERFACE;
        AddRef(); return S_OK;
    }
    ULONG STDMETHODCALLTYPE AddRef() override { return ++references; }
    ULONG STDMETHODCALLTYPE Release() override { ULONG count = --references; if (!count) delete this; return count; }
    HRESULT STDMETHODCALLTYPE AdviseSink(REFIID id, IUnknown *object, DWORD mask) override {
        if (id != IID_ITextStoreACPSink || !object) return E_INVALIDARG;
        IUnknown *identity = nullptr; HRESULT hr = object->QueryInterface(IID_IUnknown,reinterpret_cast<void**>(&identity));
        if (FAILED(hr)) return hr;
        if (sinkIdentity && identity != sinkIdentity) { identity->Release(); return CONNECT_E_ADVISELIMIT; }
        if (!sink) { hr = object->QueryInterface(IID_ITextStoreACPSink,reinterpret_cast<void**>(&sink)); if (FAILED(hr)) { identity->Release(); return hr; } sinkIdentity = identity; }
        else identity->Release();
        sinkMask = mask; return S_OK;
    }
    HRESULT STDMETHODCALLTYPE UnadviseSink(IUnknown *object) override {
        IUnknown *identity = nullptr; if (!object || FAILED(object->QueryInterface(IID_IUnknown,reinterpret_cast<void**>(&identity)))) return E_INVALIDARG;
        bool same = identity == sinkIdentity; identity->Release(); if (!same) return CONNECT_E_NOCONNECTION;
        release(sink); release(sinkIdentity); sinkMask = 0; return S_OK;
    }
    HRESULT STDMETHODCALLTYPE RequestLock(DWORD flags, HRESULT *session) override {
        if (!session) return E_POINTER;
        if (!sink) return E_UNEXPECTED;
        if (readLocksOnly && (flags&TS_LF_READWRITE)==TS_LF_READWRITE) { *session=TS_E_READONLY; return S_OK; }
        if (lock || deferLocks) {
            if (flags & TS_LF_SYNC) { *session = TS_E_SYNCHRONOUS; return S_OK; }
            ++deferred;
            pendingLock |= flags & ~TS_LF_SYNC; *session = TS_S_ASYNC; return S_OK;
        }
        lock = flags; ++locks; *session = sink->OnLockGranted(flags); lock = 0;
        if (pendingLock) PostMessageW(window, GrantLock, 0, 0);
        repaint(); return S_OK;
    }
    HRESULT STDMETHODCALLTYPE GetStatus(TS_STATUS *status) override { if (!status) return E_POINTER; status->dwDynamicFlags = readonly ? TS_SD_READONLY : 0; status->dwStaticFlags = TS_SS_NOHIDDENTEXT|(transitory?TS_SS_TRANSITORY:0); return S_OK; }
    HRESULT STDMETHODCALLTYPE QueryInsert(LONG first, LONG last, ULONG, LONG *outStart, LONG *outEnd) override { if (!outStart || !outEnd) return E_POINTER; if (!valid(first,last)) return TS_E_INVALIDPOS; *outStart = first; *outEnd = last; return S_OK; }
    HRESULT STDMETHODCALLTYPE GetSelection(ULONG index, ULONG count, TS_SELECTION_ACP *selection, ULONG *fetched) override {
        if (!selection || !fetched) return E_POINTER;
        *fetched = 0;
        if (!readable()) return TS_E_NOLOCK;
        if ((index != static_cast<ULONG>(TS_DEFAULT_SELECTION) && index != 0) || !count) return E_INVALIDARG;
        selection[0] = {start,end,{TS_AE_END,FALSE}}; *fetched = 1; return S_OK;
    }
    HRESULT STDMETHODCALLTYPE SetSelection(ULONG count, const TS_SELECTION_ACP *selection) override {
        if (!writable()) return TS_E_NOLOCK;
        if (!selection || count != 1 || !valid(selection->acpStart,selection->acpEnd)) return E_INVALIDARG;
        start = selection->acpStart; end = selection->acpEnd; repaint(); return S_OK;
    }
    HRESULT STDMETHODCALLTYPE GetText(LONG first, LONG last, WCHAR *text, ULONG capacity, ULONG *copied, TS_RUNINFO *runs, ULONG runCapacity, ULONG *runCount, LONG *next) override {
        if (!readable()) return TS_E_NOLOCK;
        if (!copied || !runCount || !next) return E_POINTER;
        *copied = *runCount = 0; if (last == -1) last = static_cast<LONG>(value.size());
        if (!valid(first,last) || (capacity && !text) || (runCapacity && !runs)) return TS_E_INVALIDPOS;
        ULONG length = static_cast<ULONG>(last-first), fetched = capacity ? std::min(capacity,length) : runCapacity ? length : 0;
        if (capacity) { std::copy_n(value.data()+first,fetched,text); *copied = fetched; }
        if (runCapacity && fetched) { runs[0] = {fetched,TS_RT_PLAIN}; *runCount = 1; }
        *next = first + static_cast<LONG>(fetched); return S_OK;
    }
    HRESULT STDMETHODCALLTYPE SetText(DWORD, LONG first, LONG last, const WCHAR *text, ULONG length, TS_TEXTCHANGE *change) override {
        if (!writable()) return TS_E_NOLOCK;
        if (readonly) return TS_E_READONLY;
        if (!change || (length && !text)) return E_POINTER;
        if (!valid(first,last)) return TS_E_INVALIDPOS;
        *change = {first,last,first+static_cast<LONG>(length)};
        value.replace(first,last-first,text ? std::wstring(text,length) : L""); start = end = change->acpNewEnd; ++writes; if (delayLayout) layoutMissing=true; repaint(); return S_OK;
    }
    HRESULT STDMETHODCALLTYPE GetFormattedText(LONG, LONG, IDataObject **object) override { if (object) *object = nullptr; return E_NOTIMPL; }
    HRESULT STDMETHODCALLTYPE GetEmbedded(LONG, REFGUID, REFIID, IUnknown **object) override { if (object) *object = nullptr; return E_NOTIMPL; }
    HRESULT STDMETHODCALLTYPE QueryInsertEmbedded(const GUID*, const FORMATETC*, BOOL *allowed) override { if (!allowed) return E_POINTER; *allowed = FALSE; return S_OK; }
    HRESULT STDMETHODCALLTYPE InsertEmbedded(DWORD, LONG, LONG, IDataObject*, TS_TEXTCHANGE*) override { return E_NOTIMPL; }
    HRESULT STDMETHODCALLTYPE InsertTextAtSelection(DWORD flags, const WCHAR *text, ULONG length, LONG *first, LONG *last, TS_TEXTCHANGE *change) override {
        if (!readable()) return TS_E_NOLOCK;
        LONG savedStart = start, savedEnd = end;
        if (flags & TS_IAS_QUERYONLY) { if (first) *first = savedStart; if (last) *last = savedEnd; return S_OK; }
        HRESULT hr = SetText(0,savedStart,savedEnd,text,length,change);
        if (SUCCEEDED(hr)) { if (first) *first = savedStart; if (last) *last = savedStart+static_cast<LONG>(length); }
        return hr;
    }
    HRESULT STDMETHODCALLTYPE InsertEmbeddedAtSelection(DWORD, IDataObject*, LONG*, LONG*, TS_TEXTCHANGE*) override { return E_NOTIMPL; }
    HRESULT STDMETHODCALLTYPE RequestSupportedAttrs(DWORD, ULONG count, const TS_ATTRID *ids) override { requestedScope = false; for (ULONG i=0;i<count;++i) if (ids[i] == GUID_PROP_INPUTSCOPE) requestedScope = true; return S_OK; }
    HRESULT STDMETHODCALLTYPE RequestAttrsAtPosition(LONG, ULONG count, const TS_ATTRID *ids, DWORD flags) override { return RequestSupportedAttrs(flags,count,ids); }
    HRESULT STDMETHODCALLTYPE RequestAttrsTransitioningAtPosition(LONG, ULONG count, const TS_ATTRID *ids, DWORD flags) override { return RequestSupportedAttrs(flags,count,ids); }
    HRESULT STDMETHODCALLTYPE FindNextAttrTransition(LONG, LONG halt, ULONG, const TS_ATTRID*, DWORD, LONG *next, BOOL *found, LONG *offset) override { if (!next || !found || !offset) return E_POINTER; *next = halt; *found = FALSE; *offset = 0; return S_OK; }
    HRESULT STDMETHODCALLTYPE RetrieveRequestedAttrs(ULONG capacity, TS_ATTRVAL *attrs, ULONG *count) override {
        if (!count) return E_POINTER;
        *count = 0;
        if (requestedScope && capacity && attrs && scopeAvailable) { attrs[0].idAttr = GUID_PROP_INPUTSCOPE; attrs[0].dwOverlapId = 0; VariantInit(&attrs[0].varValue); attrs[0].varValue.vt = VT_UNKNOWN; attrs[0].varValue.punkVal = new Scope(password?IS_PASSWORD:inputScope); *count = 1; requestedScope = false; }
        return S_OK;
    }
    HRESULT STDMETHODCALLTYPE GetEndACP(LONG *position) override { if (!readable()) return TS_E_NOLOCK; if (!position) return E_POINTER; *position = static_cast<LONG>(value.size()); return S_OK; }
    HRESULT STDMETHODCALLTYPE GetActiveView(TsViewCookie *view) override { if (!view) return E_POINTER; *view = 1; return S_OK; }
    HRESULT STDMETHODCALLTYPE GetACPFromPoint(TsViewCookie, const POINT*, DWORD, LONG *position) override { if (!position) return E_POINTER; *position = start; return S_OK; }
    HRESULT STDMETHODCALLTYPE GetTextExt(TsViewCookie, LONG first, LONG last, RECT *area, BOOL *clipped) override {
        if (writable()) ++writeExtents; else if (readable()) ++readExtents;
        if (layoutMissing) { ++extentFailures; return TS_E_NOLAYOUT; }
        if (noCaret) return E_FAIL;
        if (!area || !clipped) return E_POINTER;
        if (!valid(first,last)) return TS_E_INVALIDPOS;
        POINT point{12+first*10+layoutOffset,12}; ClientToScreen(window,&point); *area = {point.x,point.y,point.x+(zeroCaret?0:std::max<LONG>(1,(last-first)*10)),point.y+24}; *clipped = layoutClipped; return S_OK;
    }
    HRESULT STDMETHODCALLTYPE GetScreenExt(TsViewCookie, RECT *area) override { if (!area) return E_POINTER; GetWindowRect(window,area); return S_OK; }
    HRESULT STDMETHODCALLTYPE GetWnd(TsViewCookie, HWND *result) override { if (!result) return E_POINTER; *result = window; return S_OK; }
    HRESULT STDMETHODCALLTYPE OnStartComposition(ITfCompositionView*, BOOL *allowed) override { if (!allowed) return E_POINTER; *allowed = !refuseComposition && !readonly && !password; if (*allowed) ++starts; else ++refused; return S_OK; }
    HRESULT STDMETHODCALLTYPE OnUpdateComposition(ITfCompositionView*, ITfRange*) override { repaint(); return S_OK; }
    HRESULT STDMETHODCALLTYPE OnEndComposition(ITfCompositionView*) override { ++ends; repaint(); return S_OK; }
};

static Store *editor = nullptr, *password = nullptr;
static ITfThreadMgr *manager = nullptr;
static ITfKeyEventSink *keySink = nullptr;
static ITfThreadMgrEventSink *manualEvents = nullptr;
static IUnknown *manualService=nullptr;
static HWND oldCandidates=nullptr;
static HANDLE optionsBlock=INVALID_HANDLE_VALUE;
static HWND candidateWindow() {
    HWND found=nullptr;
    while ((found=FindWindowExW(nullptr,found,L"Arion.NativeCandidates",nullptr))) { DWORD process=0; GetWindowThreadProcessId(found,&process); if (process==GetCurrentProcessId() && IsWindowVisible(found) && GetPropW(found,L"Arion.CandidateSurface")==reinterpret_cast<HANDLE>(1)) return found; }
    return nullptr;
}
static ITfLangBarItemButton *button() {
    ITfLangBarItemMgr *items=nullptr; IEnumTfLangBarItems *enumerator=nullptr; ITfLangBarItemButton *result=nullptr;
    if (SUCCEEDED(manager->QueryInterface(IID_ITfLangBarItemMgr,reinterpret_cast<void**>(&items)))) {
        if (SUCCEEDED(items->EnumItems(&enumerator))) {
            ITfLangBarItem *item=nullptr; ULONG count=0;
            while (enumerator->Next(1,&item,&count)==S_OK && count) { TF_LANGBARITEMINFO info{}; item->GetInfo(&info); if (info.clsidService==ArionService) item->QueryInterface(IID_ITfLangBarItemButton,reinterpret_cast<void**>(&result)); item->Release(); if (result) break; }
            enumerator->Release();
        }
        items->Release();
    }
#ifdef ARION_TSF_MANUAL_KEYS
    if (!result && manualService) manualService->QueryInterface(IID_ITfLangBarItemButton,reinterpret_cast<void**>(&result));
#endif
    return result;
}
static void focus(Store *store) {
    SetFocus(store->window); manager->SetFocus(store->document);
#ifdef ARION_TSF_MANUAL_KEYS
    if (manualEvents) manualEvents->OnSetFocus(store->document,nullptr);
#endif
}
static LRESULT CALLBACK dispatch(HWND window, UINT message, WPARAM wp, LPARAM lp) {
    Store *store = reinterpret_cast<Store*>(GetWindowLongPtrW(window,GWLP_USERDATA));
    if (message == WM_NCCREATE) { store = static_cast<Store*>(reinterpret_cast<CREATESTRUCTW*>(lp)->lpCreateParams); SetWindowLongPtrW(window,GWLP_USERDATA,reinterpret_cast<LONG_PTR>(store)); }
    if (message == GrantLock && store) { store->grantPending(); return 0; }
    if (message == WM_SETFOCUS && store && manager && store->document) { manager->SetFocus(store->document); return 0; }
    if (message == WM_PAINT && store) {
        PAINTSTRUCT paint{}; HDC dc = BeginPaint(window,&paint); RECT area{}; GetClientRect(window,&area); area.left += 12; area.top += 12;
        auto shown = store->password ? std::wstring(store->value.size(),L'\x2022') : store->value;
        DrawTextW(dc,shown.data(),static_cast<int>(shown.size()),&area,DT_LEFT|DT_WORDBREAK); EndPaint(window,&paint); return 0;
    }
    if (message == Command) {
        auto command = reinterpret_cast<std::string*>(lp);
        if (command->rfind("key:",0)==0) {
            size_t split=command->find(':',4); unsigned long key=std::stoul(command->substr(4,split-4));
            unsigned modifiers=split==std::string::npos?0:static_cast<unsigned>(std::stoul(command->substr(split+1)));
            bool up=command->find(":up")!=std::string::npos;
            BYTE saved[256]{},state[256]{}; GetKeyboardState(saved);
            state[VK_SHIFT]=(modifiers&1)?0x80:0; state[VK_CONTROL]=(modifiers&4)?0x80:0; state[VK_MENU]=(modifiers&8)?0x80:0; state[VK_LWIN]=(modifiers&(1<<26))?0x80:0;
            SetKeyboardState(state);
            ITfContext *target=GetFocus()==password->window?password->context:editor->context;
            BOOL tested=FALSE,eaten=FALSE;
            HRESULT test=up?keySink->OnTestKeyUp(target,key,1,&tested):keySink->OnTestKeyDown(target,key,1,&tested),actual=S_FALSE;
            if (tested) actual=up?keySink->OnKeyUp(target,key,1,&eaten):keySink->OnKeyDown(target,key,1,&eaten);
            SetKeyboardState(saved);
            std::cout << "{\"tested\":" << (tested?"true":"false") << ",\"eaten\":" << (eaten?"true":"false") << ",\"testHR\":" << static_cast<long>(test) << ",\"actualHR\":" << static_cast<long>(actual) << "}" << std::endl;
        }
        else if (command->rfind("action:",0)==0) { auto item=button(); HRESULT hr=item?item->OnMenuSelect(static_cast<UINT>(std::stoul(command->substr(7)))):E_NOINTERFACE; if (item) item->Release(); std::cout << "{\"hr\":" << static_cast<long>(hr) << "}" << std::endl; }
        else if (*command=="menu") {
            ITfUIElementMgr *elements=nullptr; IEnumTfUIElements *enumerator=nullptr; bool first=true;
            std::cout << "{\"strings\":["; UINT selection=0,page=0,count=0;
            if (SUCCEEDED(manager->QueryInterface(IID_ITfUIElementMgr,reinterpret_cast<void**>(&elements)))) {
                if (SUCCEEDED(elements->EnumUIElements(&enumerator))) {
                    ITfUIElement *element=nullptr; ULONG fetched=0;
                    while (enumerator->Next(1,&element,&fetched)==S_OK && fetched) {
                        ITfCandidateListUIElement *list=nullptr;
                        if (SUCCEEDED(element->QueryInterface(IID_ITfCandidateListUIElement,reinterpret_cast<void**>(&list)))) {
                            list->GetCount(&count); list->GetSelection(&selection); list->GetCurrentPage(&page);
                            for (UINT i=0;i<std::min<UINT>(count,256);++i) { BSTR value=nullptr; if (SUCCEEDED(list->GetString(i,&value))) { if (!first) std::cout << ','; std::cout << ::quoted(std::wstring(value,SysStringLen(value))); first=false; SysFreeString(value); } }
                            list->Release();
                        }
                        element->Release();
                    }
                    enumerator->Release();
                }
                elements->Release();
            }
            HWND candidates=candidateWindow(); RECT area{}; if (candidates) GetWindowRect(candidates,&area);
            std::cout << "],\"selection\":" << selection << ",\"page\":" << page << ",\"visible\":" << (candidates?"true":"false") << ",\"x\":" << area.left << ",\"y\":" << area.top << ",\"width\":" << area.right-area.left << ",\"height\":" << area.bottom-area.top << "}" << std::endl;
        }
        else if (*command=="mode") { auto item=button(); BSTR value=nullptr; if (item) item->GetText(&value); std::cout << "{\"mode\":" << ::quoted(value?std::wstring(value,SysStringLen(value)):L"") << "}" << std::endl; if (value) SysFreeString(value); if (item) item->Release(); }
        else if (*command=="badge") {
            HWND found=nullptr,badge=nullptr;
            while ((found=FindWindowExW(nullptr,found,L"Arion.ModeBadge",nullptr))) { DWORD process=0; GetWindowThreadProcessId(found,&process); if (process==GetCurrentProcessId() && IsWindowVisible(found)) { badge=found; break; } }
            RECT area{}; if (badge) GetWindowRect(badge,&area);
            std::cout << "{\"visible\":" << (badge?"true":"false") << ",\"width\":" << area.right-area.left << ",\"height\":" << area.bottom-area.top << ",\"dpi\":" << GetDpiForWindow(editor->window) << ",\"hit\":" << (badge?SendMessageW(badge,WM_NCHITTEST,0,MAKELPARAM(area.left+1,area.top+1)):0) << ",\"focusRetained\":" << (GetFocus()==editor->window?"true":"false") << "}" << std::endl;
        }
        else if (*command=="lockoptions") { wchar_t profile[32768]{}; GetEnvironmentVariableW(L"ARION_TSF_TEST_USER_DIRECTORY",profile,32768); optionsBlock=CreateFileW((std::wstring(profile)+L"\\options.json").c_str(),GENERIC_READ|GENERIC_WRITE,0,nullptr,OPEN_EXISTING,FILE_ATTRIBUTE_NORMAL,nullptr); std::cout << "{\"locked\":" << (optionsBlock!=INVALID_HANDLE_VALUE?"true":"false") << "}" << std::endl; }
        else if (*command=="unlockoptions") { if (optionsBlock!=INVALID_HANDLE_VALUE) CloseHandle(optionsBlock); optionsBlock=INVALID_HANDLE_VALUE; std::cout << "{\"done\":true}" << std::endl; }
        else if (*command=="cacheclick") { oldCandidates=candidateWindow(); std::cout << "{\"done\":true}" << std::endl; }
        else if (command->rfind("click:",0)==0 || *command=="staleclick") { HWND target=*command=="staleclick"?oldCandidates:candidateWindow(); int row=*command=="staleclick"?0:std::stoi(command->substr(6)); if (target) SendMessageW(target,WM_LBUTTONUP,0,MAKELPARAM(12,row*32+16)); std::cout << "{\"done\":true}" << std::endl; }
        else if (*command == "text") std::cout << "{\"text\":" << ::quoted(editor->value) << "}" << std::endl;
        else if (*command == "layoutstate") std::cout << "{\"missing\":" << (editor->layoutMissing?"true":"false") << ",\"failures\":" << editor->extentFailures << ",\"reads\":" << editor->readExtents << ",\"writeExtents\":" << editor->writeExtents << ",\"writes\":" << editor->writes << "}" << std::endl;
        else if (*command == "state") std::cout << "{\"text\":" << ::quoted(editor->value) << ",\"start\":" << editor->start << ",\"end\":" << editor->end << ",\"writes\":" << editor->writes << ",\"starts\":" << editor->starts << ",\"ends\":" << editor->ends << ",\"locks\":" << editor->locks << ",\"refused\":" << editor->refused << ",\"deferred\":" << editor->deferred << "}" << std::endl;
        else if (*command == "quit") DestroyWindow(window);
        else {
            if (*command == "password") focus(password);
            else if (*command == "edit") focus(editor);
            else if (*command == "select") editor->userSelection(1,1);
            else if (*command == "reset") { focus(editor); editor->setReadonly(false); editor->userReplace(L"A\U0001f642\u4e2dZ",true); editor->userSelection(3,4); }
            else if (*command == "readonly") editor->setReadonly(true);
            else if (*command == "writable") editor->setReadonly(false);
            else if (*command == "user") editor->userReplace(L"user");
            else if (*command == "replace") { editor->start = 3; editor->end = static_cast<LONG>(editor->value.size())-1; editor->userReplace(L"user"); }
            else if (*command == "block") editor->deferLocks = true;
            else if (*command == "unblock") { editor->deferLocks = false; editor->grantPending(); }
            else if (*command == "refuse") editor->refuseComposition = true;
            else if (*command == "accept") editor->refuseComposition = false;
            else if (*command == "zerocaret") editor->zeroCaret = true;
            else if (*command == "nocaret") editor->noCaret = true;
            else if (*command == "delaylayout") editor->delayLayout=true;
            else if (*command == "instantlayout") { editor->delayLayout=false; editor->layoutReady(); }
            else if (*command == "layout") editor->layoutReady();
            else if (*command == "nolayout") editor->layoutMissing=true;
            else if (*command == "clipped") { editor->layoutClipped=true; editor->layoutReady(); }
            else if (*command == "unclipped") { editor->layoutClipped=false; editor->layoutReady(); }
            else if (*command == "scroll") { editor->layoutOffset+=40; editor->layoutReady(); }
            else if (*command == "readlocks") editor->readLocksOnly=true;
            else if (*command == "writelocks") editor->readLocksOnly=false;
            else if (*command == "caret") { editor->noCaret=false; editor->zeroCaret=false; }
            else if (*command == "chromium") { editor->transitory=true; password->transitory=true; }
            else if (*command == "regularstore") { editor->transitory=false; password->transitory=false; }
            else if (*command == "noscope") editor->setScopeAvailable(false);
            else if (*command == "scope") editor->setScopeAvailable(true);
            else if (command->rfind("inputscope:",0)==0) { editor->inputScope=static_cast<InputScope>(std::stoi(command->substr(11))); editor->setScopeAvailable(editor->scopeAvailable); }
            else if (*command == "threadblur") {
                ITfThreadFocusSink *sink=nullptr;
                if (SUCCEEDED(keySink->QueryInterface(IID_ITfThreadFocusSink,reinterpret_cast<void**>(&sink)))) { sink->OnKillThreadFocus(); sink->Release(); }
            }
            else if (*command == "keyboardfocus") keySink->OnSetFocus(TRUE);
            else if (*command == "keyboardblur") keySink->OnSetFocus(FALSE);
            else { delete command; return 0; }
            std::cout << "{\"done\":true}" << std::endl;
        }
        delete command; return 0;
    }
    if (message == WM_DESTROY && !store) { PostQuitMessage(0); return 0; }
    return DefWindowProcW(window,message,wp,lp);
}

int wmain(int count, wchar_t **args) {
    HWND previousForeground=GetForegroundWindow();
#ifdef ARION_TSF_MANUAL_KEYS
    wchar_t profile[32768]{}; DWORD profileLength=GetEnvironmentVariableW(L"ARION_TSF_TEST_USER_DIRECTORY",profile,32768);
    if (!profileLength || profileLength>=32768 || !keyboardOptionsDetail::security().ready) return 12;
    if (!CreateDirectoryW(profile,&keyboardOptionsDetail::security().attributes) && GetLastError()!=ERROR_ALREADY_EXISTS) return 12;
    keyboardOptionsDetail::Handle profileHandle(CreateFileW(profile,FILE_READ_ATTRIBUTES|READ_CONTROL,FILE_SHARE_READ|FILE_SHARE_WRITE,nullptr,OPEN_EXISTING,FILE_FLAG_BACKUP_SEMANTICS|FILE_FLAG_OPEN_REPARSE_POINT,nullptr));
    if (profileHandle.value==INVALID_HANDLE_VALUE || !keyboardOptionsDetail::security().valid(profileHandle.value,true)) return 12;
#endif
    if (count != 2 || FAILED(CoInitializeEx(nullptr,COINIT_APARTMENTTHREADED))) return 2;
    HMODULE dll = LoadLibraryW(args[1]); if (!dll) return 3;
    HRESULT(__stdcall *create)(REFCLSID,REFIID,void**) = nullptr;
    FARPROC address = GetProcAddress(dll,"DllGetClassObject");
    static_assert(sizeof(create) == sizeof(address)); std::memcpy(&create,&address,sizeof(create));
    IClassFactory *factory = nullptr; ITfTextInputProcessor *service = nullptr;
    if (!create || FAILED(create(ArionService,IID_IClassFactory,reinterpret_cast<void**>(&factory)))) return 4;
    HRESULT hr = factory->CreateInstance(nullptr,IID_ITfTextInputProcessor,reinterpret_cast<void**>(&service)); factory->Release(); if (FAILED(hr)) return 5;
    hr = CoCreateInstance(CLSID_TF_ThreadMgr,nullptr,CLSCTX_INPROC_SERVER,IID_ITfThreadMgr,reinterpret_cast<void**>(&manager));
    TfClientId app = 0, client = 0;
    if (FAILED(hr)) return 6;
#ifdef ARION_TSF_MANUAL_KEYS
    ITfThreadMgrEx *isolated=nullptr;
    hr=manager->QueryInterface(IID_ITfThreadMgrEx,reinterpret_cast<void**>(&isolated));
    if (SUCCEEDED(hr)) { hr=isolated->ActivateEx(&app,TF_TMAE_NOACTIVATETIP); isolated->Release(); }
#else
    hr=manager->Activate(&app);
#endif
    if (FAILED(hr)) return 6;
    // The manually instantiated service is an owned test client. Key sinks
    // require an activated client ID; GetClientId alone does not activate one.
    client=app;
    HINSTANCE instance = GetModuleHandleW(nullptr); WNDCLASSW definition{}; definition.lpfnWndProc = dispatch; definition.hInstance = instance; definition.lpszClassName = L"ArionOwnedACPFixture"; definition.hbrBackground = reinterpret_cast<HBRUSH>(COLOR_WINDOW+1); RegisterClassW(&definition);
    HWND window = CreateWindowExW(0,definition.lpszClassName,L"Arion owned TSF text store",WS_OVERLAPPEDWINDOW|WS_VISIBLE,150,150,600,250,nullptr,nullptr,instance,nullptr);
    editor = new Store; password = new Store; password->password = true; password->value.clear(); password->start = password->end = 0;
    editor->window = CreateWindowExW(WS_EX_CLIENTEDGE,definition.lpszClassName,L"",WS_CHILD|WS_VISIBLE|WS_TABSTOP,20,20,540,80,window,nullptr,instance,editor);
    password->window = CreateWindowExW(WS_EX_CLIENTEDGE,definition.lpszClassName,L"",WS_CHILD|WS_VISIBLE|WS_TABSTOP,20,120,300,40,window,nullptr,instance,password);
    if (FAILED(editor->initialize(manager,app)) || FAILED(password->initialize(manager,app))) return 9;
    SetForegroundWindow(window); focus(editor);
    hr = service->Activate(manager,client); if (FAILED(hr)) { std::cerr << "TSF activation failed 0x" << std::hex << static_cast<unsigned long>(hr) << std::endl; return 7; }
    if (FAILED(service->QueryInterface(IID_ITfKeyEventSink,reinterpret_cast<void**>(&keySink)))) return 10;
#ifdef ARION_TSF_MANUAL_KEYS
    if (FAILED(service->QueryInterface(IID_ITfThreadMgrEventSink,reinterpret_cast<void**>(&manualEvents)))) return 11;
    manualService=service;
#endif
    std::cout << "{\"ready\":true}" << std::endl;
    std::thread input([window] { std::string line; while (std::getline(std::cin,line)) { auto command = new std::string(line); if (!PostMessageW(window,Command,0,reinterpret_cast<LPARAM>(command))) delete command; if (line == "quit") return; } PostMessageW(window,WM_CLOSE,0,0); });
    MSG message; while (GetMessageW(&message,nullptr,0,0)>0) { TranslateMessage(&message); DispatchMessageW(&message); }
    if (manualEvents) { manualEvents->Release(); manualEvents=nullptr; }
    manualService=nullptr;
    if (optionsBlock!=INVALID_HANDLE_VALUE) CloseHandle(optionsBlock);
    keySink->Release(); keySink=nullptr; service->Deactivate(); service->Release(); manager->SetFocus(nullptr); editor->shutdown(); password->shutdown(); editor->Release(); password->Release(); manager->Deactivate(); manager->Release(); manager = nullptr;
    if (input.joinable()) input.join();
    FreeLibrary(dll); CoUninitialize();
    if (previousForeground && IsWindow(previousForeground)) SetForegroundWindow(previousForeground);
    return 0;
}
