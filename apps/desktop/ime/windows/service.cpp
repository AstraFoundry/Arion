#define WIN32_LEAN_AND_MEAN
#include <windows.h>
#include <msctf.h>
#include <inputscope.h>
#include <sddl.h>
#include <wincrypt.h>
#include <vector>
#include <string>
#include <atomic>
#include <thread>
#include <mutex>
#include <memory>
#include <shlobj.h>
#include <knownfolders.h>
#include <shellapi.h>
#ifdef ARION_TSF_TESTING
#include <cstdio>
#endif
#include "identity.h"
#include "wire.h"
#include "transport.h"
#include "typing.h"
#include "text_layout.h"
#include "lifetime.h"
#include "language_bar.h"
#include "mode_badge.h"

static HINSTANCE module;
static constexpr UINT Incoming = WM_APP + 41;
static constexpr UINT Disconnected = WM_APP + 42;
static constexpr UINT Learn = WM_APP + 43;
static constexpr UINT Layout = WM_APP + 44;
// Protocol 2 requires PRIVATE/password/PIN rejection before composition.
static constexpr unsigned WireProtocol=2;
struct LearnCommit { std::string commit,previous; uint64_t generation=0; };

template<class T> void drop(T *&value) { if (value) { value->Release(); value = nullptr; } }
static std::wstring newID() {
    GUID id; if (FAILED(CoCreateGuid(&id))) return {};
    wchar_t text[40]; StringFromGUID2(id, text, 40); return text;
}
static std::wstring buildIdentity() {
    wchar_t path[32768]; DWORD length = GetModuleFileNameW(module, path, 32768);
    if (!length || length == 32768) return {};
    HANDLE file = CreateFileW(path, GENERIC_READ, FILE_SHARE_READ | FILE_SHARE_DELETE, nullptr, OPEN_EXISTING, 0, nullptr);
    if (file == INVALID_HANDLE_VALUE) return {};
    HCRYPTPROV provider = 0; HCRYPTHASH hash = 0;
    bool ok = CryptAcquireContextW(&provider, nullptr, nullptr, PROV_RSA_AES, CRYPT_VERIFYCONTEXT) && CryptCreateHash(provider, CALG_SHA_256, 0, 0, &hash);
    BYTE buffer[16384]; DWORD count;
    while (ok) { if (!ReadFile(file, buffer, sizeof(buffer), &count, nullptr)) { ok = false; break; } if (!count) break; ok = CryptHashData(hash, buffer, count, 0) != FALSE; }
    BYTE digest[32]; DWORD size = 32;
    if (ok) ok = CryptGetHashParam(hash, HP_HASHVAL, digest, &size, 0) != FALSE;
    if (hash) CryptDestroyHash(hash);
    if (provider) CryptReleaseContext(provider, 0);
    CloseHandle(file);
    if (!ok) return {};
    std::wstring result; for (BYTE byte : digest) { const wchar_t *hex = L"0123456789abcdef"; result += hex[byte >> 4]; result += hex[byte & 15]; } return result;
}

class Service;
class LayoutEdit final : public ITfEditSession {
    CoreObjectLease lifetime;
    std::atomic<ULONG> references{1}; Service *owner; ITfContext *context;
    std::wstring activation; uint64_t revision,ticket;
public:
    LayoutEdit(Service*,ITfContext*,const std::wstring&,uint64_t,uint64_t);
    ~LayoutEdit();
    HRESULT STDMETHODCALLTYPE QueryInterface(REFIID id,void **out) override { if (!out) return E_POINTER; *out=nullptr; if (id!=IID_IUnknown && id!=IID_ITfEditSession) return E_NOINTERFACE; *out=static_cast<ITfEditSession*>(this); AddRef(); return S_OK; }
    ULONG STDMETHODCALLTYPE AddRef() override { return ++references; }
    ULONG STDMETHODCALLTYPE Release() override { auto count=--references; if (!count) delete this; return count; }
    HRESULT STDMETHODCALLTYPE DoEditSession(TfEditCookie) override;
};
class KeyEdit final : public ITfEditSession {
    CoreObjectLease lifetime;
    std::atomic<ULONG> references{1}; Service *owner; ITfContext *context;
    std::wstring activation; TypingCommand command;
public:
    bool handled=false;
    KeyEdit(Service*,ITfContext*,const std::wstring&,const TypingCommand&);
    ~KeyEdit();
    HRESULT STDMETHODCALLTYPE QueryInterface(REFIID id,void **out) override { if (!out) return E_POINTER; *out=nullptr; if (id!=IID_IUnknown && id!=IID_ITfEditSession) return E_NOINTERFACE; *out=static_cast<ITfEditSession*>(this); AddRef(); return S_OK; }
    ULONG STDMETHODCALLTYPE AddRef() override { return ++references; }
    ULONG STDMETHODCALLTYPE Release() override { auto count=--references; if (!count) delete this; return count; }
    HRESULT STDMETHODCALLTYPE DoEditSession(TfEditCookie) override;
};
class Edit final : public ITfEditSession {
    CoreObjectLease lifetime;
    std::atomic<ULONG> references{1}; Service *owner; ITfContext *context; WireMessage message;
    ITfComposition *cleanup = nullptr;
    std::wstring expected, original;
public:
    Edit(Service *service, ITfContext *target, const WireMessage &value, ITfComposition *owned = nullptr, const std::wstring &text = L"", const std::wstring &saved = L"");
    ~Edit();
    HRESULT STDMETHODCALLTYPE QueryInterface(REFIID id, void **out) override { if (!out) return E_POINTER; *out = nullptr; if (id == IID_IUnknown || id == IID_ITfEditSession) { *out = static_cast<ITfEditSession*>(this); AddRef(); return S_OK; } return E_NOINTERFACE; }
    ULONG STDMETHODCALLTYPE AddRef() override { return ++references; }
    ULONG STDMETHODCALLTYPE Release() override { auto count = --references; if (!count) delete this; return count; }
    HRESULT STDMETHODCALLTYPE DoEditSession(TfEditCookie cookie) override;
};

class Service final : public ITfTextInputProcessorEx, public ITfThreadMgrEventSink, public ITfThreadFocusSink, public ITfCompositionSink, public ITfTextEditSink, public ITfTextLayoutSink, public ITfKeyEventSink {
    CoreObjectLease lifetime;
    std::atomic<ULONG> references{1};
    ITfThreadMgr *manager = nullptr;
    ITfContext *context = nullptr;
    ITfComposition *composition = nullptr;
    TfClientId client = TF_CLIENTID_NULL;
    DWORD managerCookie = TF_INVALID_COOKIE, focusCookie = TF_INVALID_COOKIE, editCookie = TF_INVALID_COOKIE;
    DWORD layoutCookie=TF_INVALID_COOKIE;
    uint64_t layoutEpoch=0,layoutSerial=0,layoutPending=0;
    bool layoutQueued=false,layoutDirty=false;
    HWND window = nullptr;
    bool focused = false, mutating = false;
    std::wstring activation, session, build, original, lastText;
    uint64_t revision = 0;
    PipeTransport transport;
    ITfKeystrokeMgr *keys=nullptr;
    KeyboardClient keyboard;
    CandidateWindow candidates{module};
    ModeBadge badge{module};
    LanguageBar *languageBar=nullptr;
    std::unique_ptr<Typing> typing;
    ArionShiftTap shiftTap{};
    std::wstring settingsPath;
    FILETIME settingsWrite{};
    KeyboardSettings settings;

    static LRESULT CALLBACK dispatch(HWND window, UINT message, WPARAM wp, LPARAM lp) {
        Service *self = reinterpret_cast<Service*>(GetWindowLongPtrW(window, GWLP_USERDATA));
        if (message == WM_NCCREATE) { self = static_cast<Service*>(reinterpret_cast<CREATESTRUCTW*>(lp)->lpCreateParams); SetWindowLongPtrW(window, GWLP_USERDATA, reinterpret_cast<LONG_PTR>(self)); }
        if (!self) return DefWindowProcW(window, message, wp, lp);
        if (message==Layout) { if (static_cast<uint64_t>(wp)==self->layoutEpoch) { self->layoutQueued=false; self->requestLayout(); } return 0; }
        if (message == Incoming) {
            auto value=reinterpret_cast<WireMessage*>(lp);
            if (static_cast<uint64_t>(wp)==self->transport.generation()) self->receive(*value);
            delete value; return 0;
        }
        if (message == Disconnected) {
            if (static_cast<uint64_t>(wp)==self->transport.generation()) { self->disconnect(); self->invalidate(); }
            return 0;
        }
        if (message == Learn) { std::unique_ptr<LearnCommit> value(reinterpret_cast<LearnCommit*>(lp)); if (value->generation==self->keyboard.generation()) self->keyboard.learn(value->commit,value->previous,self->typing->rankingFailure); return 0; }
        if (message == WM_TIMER && wp==2) { self->badge.hide(); KillTimer(window,2); return 0; }
        if (message == WM_TIMER) { self->refreshSettings(); self->connect(); if (!self->session.empty()) { WireMessage check; check.op=L"inspect"; check.activation=self->activation; check.session=self->session; self->request(check); } if (self->typing->composing()) { TypingCommand check; check.kind=TypingCommand::Inspect; self->keyRequest(check); } if (self->languageBar) self->languageBar->changed(); return 0; }
        return DefWindowProcW(window, message, wp, lp);
    }
    void send(const wchar_t *op) {
        std::string text = "{\"op\":" + ::quoted(op) + ",\"activation\":" + ::quoted(activation) + ",\"session\":" + ::quoted(session) + ",\"build\":" + ::quoted(build) + ",\"client\":\"Windows editor\",\"text\":\"\"}\n";
        if (std::wcscmp(op,L"hello")==0) text.insert(1,"\"protocol\":"+std::to_string(WireProtocol)+",");
        transport.send(std::move(text));
    }
    void disconnect() {
        transport.disconnect();
    }
    void refreshSettings(bool force=false) {
        if (settingsPath.empty()) return;
        WIN32_FILE_ATTRIBUTE_DATA information{};
        bool exists=GetFileAttributesExW(settingsPath.c_str(),GetFileExInfoStandard,&information)!=FALSE;
        FILETIME stamp=exists?information.ftLastWriteTime:FILETIME{};
        if (force || CompareFileTime(&stamp,&settingsWrite)) { settings=readKeyboardSettings(settingsPath); typing->configure(settings); settingsWrite=stamp; }
    }
    void showMode(TfEditCookie cookie,NativeCopy value) {
        if (!context) return;
        TF_SELECTION selection{}; ULONG count=0; ITfContextView *view=nullptr; RECT area{}; BOOL clipped=FALSE; HWND owner=nullptr;
        HRESULT hr=context->GetSelection(cookie,TF_DEFAULT_SELECTION,1,&selection,&count);
        if (SUCCEEDED(hr) && count==1 && safe(cookie,selection.range)) {
            selection.range->Collapse(cookie,TF_ANCHOR_END);
            hr=context->GetActiveView(&view);
            if (SUCCEEDED(hr)) hr=view->GetTextExt(cookie,selection.range,&area,&clipped);
            if (SUCCEEDED(hr)) hr=view->GetWnd(&owner);
        } else hr=E_ACCESSDENIED;
        if (view) view->Release();
        if (count) selection.range->Release();
        if (FAILED(hr) || clipped || area.bottom<=area.top || area.right<area.left) { badge.hide(); return; }
        NativeCopy previous=value==NativeCopy::ChineseBadge?NativeCopy::EnglishBadge:value==NativeCopy::EnglishBadge?NativeCopy::ChineseBadge:value==NativeCopy::FullWidthBadge?NativeCopy::HalfWidthBadge:value==NativeCopy::HalfWidthBadge?NativeCopy::FullWidthBadge:value==NativeCopy::ChinesePunctuationBadge?NativeCopy::EnglishPunctuationBadge:NativeCopy::ChinesePunctuationBadge;
        badge.show(owner,area,nativeCopy(previous),nativeCopy(value),settings.palette);
    }
    KeyboardMenuState menuState() {
        KeyboardMenuState value; value.available=typing->available(); value.ascii=typing->isASCII(); value.fullWidth=typing->isFullWidth(); value.englishPunctuation=typing->isEnglishPunctuation();
        value.voice=focused && transport.connected(); value.dictationKey=settings.dictationKey; value.pending=!typing->recoveryText.empty();
        if (!typing->modeFailure.empty()) value.failure=typing->modeFailure;
        else if (!value.available) value.failure=nativeCopy(NativeCopy::Unavailable);
        else if (!typing->rankingFailure.empty()) value.failure=nativeCopy(NativeCopy::RankingUnavailable);
        return value;
    }
    void menuCommand(KeyboardMenuAction action) {
        arion_shift_reset(&shiftTap);
        if (action==KeyboardMenuAction::Voice) { connect(); if (focused && transport.connected()) send(L"dictate"); }
        else if (action==KeyboardMenuAction::Language || action==KeyboardMenuAction::Width || action==KeyboardMenuAction::Punctuation) {
            TypingCommand command; command.kind=action==KeyboardMenuAction::Language?TypingCommand::ToggleASCII:action==KeyboardMenuAction::Width?TypingCommand::ToggleWidth:TypingCommand::TogglePunctuation;
            keyRequest(command);
        } else if (action==KeyboardMenuAction::Reset) {
            typing->invalidate();
            if (!keyboard.resetLearning()) typing->modeFailure=nativeCopy(NativeCopy::ResetFailed); else typing->modeFailure.clear();
        } else if (action==KeyboardMenuAction::OpenApp) {
            if (reinterpret_cast<INT_PTR>(ShellExecuteW(nullptr,L"open",L"arion://open",nullptr,nullptr,SW_SHOWNORMAL))<=32) typing->modeFailure=nativeCopy(NativeCopy::OpenAppFailed);
        }
        else if (action==KeyboardMenuAction::CopyPending && !typing->recoveryText.empty() && OpenClipboard(window)) {
            const auto &text=typing->recoveryText; size_t size=(text.size()+1)*sizeof(wchar_t); HGLOBAL memory=GlobalAlloc(GMEM_MOVEABLE,size);
            if (memory) { void *bytes=GlobalLock(memory); if (bytes) { memcpy(bytes,text.c_str(),size); GlobalUnlock(memory); if (EmptyClipboard() && SetClipboardData(CF_UNICODETEXT,memory)) memory=nullptr; } if (memory) GlobalFree(memory); }
            CloseClipboard();
        }
        if (languageBar) languageBar->changed();
    }
    void connect() {
        if (!focused || !context || activation.empty() || transport.connected()) return;
        auto path = userPipe(); if (path.empty()) return;
#ifdef ARION_TSF_TESTING
        wchar_t isolated[256]{}; DWORD count=GetEnvironmentVariableW(L"ARION_TSF_TEST_PIPE",isolated,256);
        if (count && count<256 && std::wstring(isolated).rfind(L"\\\\.\\pipe\\Arion.TSF.Test.",0)==0) path=isolated;
#endif
        if (!transport.connect(path, window, Incoming, Disconnected)) return;
        send(L"hello"); send(L"activate");
    }
    void request(const WireMessage &message) {
        if (!context) return;
        auto edit = new Edit(this, context, message); HRESULT result = E_FAIL;
        HRESULT hr = context->RequestEditSession(client, edit, TF_ES_ASYNCDONTCARE | (message.op==L"inspect" ? TF_ES_READ : TF_ES_READWRITE), &result);
        edit->Release();
        if (FAILED(hr) || (FAILED(result) && result != TF_S_ASYNC)) invalidate();
    }
    void invalidate() {
        if (session.empty() && !composition) return;
        send(L"escape");
        session.clear(); revision = 0;
        if (composition && context) {
            WireMessage cancel; cancel.op = L"revoke"; cancel.activation = activation;
            auto cleanup = new Edit(this, context, cancel, composition, lastText, original);
            drop(composition);
            HRESULT result = E_FAIL;
            context->RequestEditSession(client, cleanup, TF_ES_ASYNC | TF_ES_READWRITE, &result);
            cleanup->Release();
        }
    }
    void cleanupTyping(ITfComposition *owned,const std::wstring &expected,const std::wstring &saved) {
        if (!context) return;
        WireMessage cancel; cancel.op=L"revoke";
        auto cleanup=new Edit(this,context,cancel,owned,expected,saved);
        HRESULT result=E_FAIL;
        HRESULT requested=context->RequestEditSession(client,cleanup,TF_ES_ASYNC|TF_ES_READWRITE,&result);
#ifdef ARION_TSF_TESTING
        std::fprintf(stderr,"TSF keyboard cleanup request=%08lx result=%08lx\n",static_cast<unsigned long>(requested),static_cast<unsigned long>(result));
#else
        (void)requested;
#endif
        cleanup->Release();
    }
    void cancelVoice(TfEditCookie cookie) {
        if (session.empty() && !composition) return;
        send(L"escape"); session.clear(); revision=0;
        if (!composition) return;
        ITfRange *range=nullptr;
        if (SUCCEEDED(composition->GetRange(&range))) {
            std::wstring current;
            if (text(cookie,range,current) && current==lastText) range->SetText(cookie,0,original.data(),static_cast<LONG>(original.size()));
            range->Release();
        }
        composition->EndComposition(cookie); drop(composition);
    }
    bool keyRequest(const TypingCommand &command) {
        if (!context || !focused) return false;
        auto edit=new KeyEdit(this,context,activation,command); HRESULT result=E_FAIL;
        HRESULT hr=context->RequestEditSession(client,edit,TF_ES_SYNC|TF_ES_READWRITE,&result);
        bool handled=SUCCEEDED(hr) && SUCCEEDED(result) && edit->handled;
        edit->Release(); return handled;
    }
    void clearLayout() { ++layoutEpoch; layoutPending=0; layoutQueued=false; layoutDirty=false; }
    void queueLayout() {
        if (!window || !focused || !context || !typing->composing()) return;
        layoutDirty=true;
        if (!layoutQueued && !layoutPending) layoutQueued=PostMessageW(window,Layout,static_cast<WPARAM>(layoutEpoch),0)!=FALSE;
    }
    void requestLayout() {
        if (!layoutDirty || layoutPending || !focused || !context || !typing->composing()) return;
        layoutDirty=false; const auto ticket=++layoutSerial; layoutPending=ticket;
        auto edit=new LayoutEdit(this,context,activation,typing->revision,ticket); HRESULT result=E_FAIL;
        const HRESULT hr=context->RequestEditSession(client,edit,TF_ES_ASYNC|TF_ES_READ,&result);
        edit->Release();
        if ((FAILED(hr) || FAILED(result)) && layoutPending==ticket) layoutPending=0;
    }
    NativeKey nativeKey(WPARAM key,LPARAM details) {
        BYTE state[256]{}; GetKeyboardState(state);
        auto native=windowsKey(key,details,state,GetKeyboardLayout(0));
        if ((key==0 || key==255) && ((state[VK_SHIFT]&0x80)!=0)!=static_cast<bool>(shiftTap.held)) { native.shift=true; native.shiftSide=0; }
        return native;
    }
    bool wantsKey(const NativeKey &key) {
        if (!session.empty() || typing->composing() || shiftTap.held) return true;
        return typing->available() && (key.shift || arion_key_action(key.symbol,key.modifiers,typing->isASCII())!=ARION_KEY_PASS);
    }
    void setContext(ITfDocumentMgr *document) {
        ITfContext *next = nullptr;
        if (document) document->GetTop(&next);
        if (next == context) { drop(next); return; }
        clearLayout();
        badge.hide();
        typing->attach(nullptr); arion_shift_reset(&shiftTap);
        invalidate(); send(L"deactivate"); disconnect();
        if (context) { ITfSource *source = nullptr; if (SUCCEEDED(context->QueryInterface(IID_ITfSource, reinterpret_cast<void**>(&source)))) { if (editCookie!=TF_INVALID_COOKIE) source->UnadviseSink(editCookie); if (layoutCookie!=TF_INVALID_COOKIE) source->UnadviseSink(layoutCookie); source->Release(); } editCookie=layoutCookie=TF_INVALID_COOKIE; }
        drop(composition); drop(context); context = next;
        typing->attach(context);
        refreshSettings(true);
        activation = context ? newID() : L"";
        if (context) { ITfSource *source = nullptr; if (SUCCEEDED(context->QueryInterface(IID_ITfSource, reinterpret_cast<void**>(&source)))) { source->AdviseSink(IID_ITfTextEditSink, static_cast<ITfTextEditSink*>(this), &editCookie); source->AdviseSink(IID_ITfTextLayoutSink,static_cast<ITfTextLayoutSink*>(this),&layoutCookie); source->Release(); } }
        connect();
    }
    void receive(const WireMessage &message) {
        if (!focused || !context || message.activation != activation || activation.empty()) return;
        if (message.op == L"begin") {
            if (!session.empty() || message.session.empty()) return;
            session = message.session; revision = 0; request(message);
        } else if (!session.empty() && message.session == session) {
            if (message.op == L"mark") { if (message.revision <= revision) return; revision = message.revision; request(message); }
            else if (message.op == L"commit") request(message);
            else if (message.op == L"cancel") invalidate();
        }
    }
    bool safe(TfEditCookie cookie, ITfRange *range) {
        TF_STATUS status{};
        HRESULT statusResult=context->GetStatus(&status);
        if (FAILED(statusResult) || (status.dwDynamicFlags & TF_SD_READONLY)) return false;
        const bool transitory=(status.dwStaticFlags & TF_SS_TRANSITORY)!=0;
        HWND focus = GetFocus(); wchar_t name[64]{}; if (focus) GetClassNameW(focus, name, 64);
        bool nativeRichEdit = _wcsicmp(name,L"RICHEDIT50W")==0;
        if ((_wcsicmp(name, L"Edit") == 0 || nativeRichEdit) && (GetWindowLongPtrW(focus, GWL_STYLE) & (ES_PASSWORD | ES_READONLY))) return false;
        // The stock RichEdit password/read-only styles are an independent,
        // verified safety adapter. Custom text stores must expose input scope.
        if (nativeRichEdit) return true;
        ITfReadOnlyProperty *property = nullptr;
        if (FAILED(context->GetAppProperty(GUID_PROP_INPUTSCOPE, &property))) return false;
        VARIANT value; VariantInit(&value);
        HRESULT hr = property->GetValue(cookie, range, &value); property->Release();
        // Chromium marks ordinary editable stores transitory too. That flag
        // describes document lifetime, not editability; require an explicit
        // input scope rather than treating it as a password/read-only signal.
        bool allowed = SUCCEEDED(hr) && (!transitory || (value.vt == VT_UNKNOWN && value.punkVal));
        if (allowed && value.vt == VT_UNKNOWN && value.punkVal) {
            ITfInputScope *scope = nullptr;
            if (SUCCEEDED(value.punkVal->QueryInterface(IID_ITfInputScope, reinterpret_cast<void**>(&scope)))) {
                InputScope *scopes = nullptr; UINT count = 0;
                if (FAILED(scope->GetInputScopes(&scopes, &count)) || (count && !scopes) || (transitory && !count)) allowed = false;
                // Chromium maps password fields to IS_PRIVATE. PIN/password
                // variants are equally unsuitable for composition/learning.
                for (UINT i = 0; scopes && i < count; ++i) {
                    InputScope field=scopes[i];
                    if (field==IS_PASSWORD || field==IS_PRIVATE || field==IS_NUMERIC_PASSWORD || field==IS_NUMERIC_PIN || field==IS_ALPHANUMERIC_PIN || field==IS_ALPHANUMERIC_PIN_SET) allowed=false;
                }
                CoTaskMemFree(scopes); scope->Release();
            } else allowed = false;
        }
        VariantClear(&value); return allowed;
    }
    bool sameOwned(TfEditCookie cookie) {
        if (!composition || !context) return false;
        ITfRange *range=nullptr; TF_SELECTION selection{}; ULONG count=0;
        if (SUCCEEDED(composition->GetRange(&range)) && SUCCEEDED(context->GetSelection(cookie,TF_DEFAULT_SELECTION,1,&selection,&count)) && count==1) {
            std::wstring current; LONG start=1,end=1;
            selection.range->CompareStart(cookie,range,TF_ANCHOR_END,&start);
            selection.range->CompareEnd(cookie,range,TF_ANCHOR_END,&end);
            bool unchanged=text(cookie,range,current) && current==lastText && start==0 && end==0;
            selection.range->Release(); range->Release(); return unchanged;
        }
        drop(range); if (count) drop(selection.range); return false;
    }
public:
    Service() {
        typing=std::make_unique<Typing>(keyboard,this,candidates,
            [this](TfEditCookie cookie,ITfRange *range) { return safe(cookie,range); },
            [](TfEditCookie cookie,ITfRange *range,std::wstring &value) { return text(cookie,range,value); },
            [this](ITfComposition *owned,const std::wstring &expected,const std::wstring &saved) { cleanupTyping(owned,expected,saved); },
            [this](const std::string &commit,const std::string &previous,uint64_t generation) {
                auto value=new LearnCommit{commit,previous,generation};
                if (!PostMessageW(window,Learn,0,reinterpret_cast<LPARAM>(value))) delete value;
            },[this](TfEditCookie cookie,NativeCopy mode) { showMode(cookie,mode); });
        candidates.onAction([this](const CandidateAction &action) {
            TypingCommand command; command.revision=action.revision; command.engineIndex=action.index;
            command.kind=action.kind==CandidateAction::Select?TypingCommand::Choose:action.kind==CandidateAction::Previous?TypingCommand::Previous:TypingCommand::Next;
            keyRequest(command);
        });
    }
    ~Service() { Deactivate(); typing.reset(); }
    static bool text(TfEditCookie cookie, ITfRange *range, std::wstring &result) {
        ITfRange *copy = nullptr;
        if (FAILED(range->Clone(&copy))) return false;
        wchar_t buffer[4096]; ULONG count = 0; HRESULT hr = S_OK;
        do { hr = copy->GetText(cookie, TF_TF_MOVESTART, buffer, 4096, &count); result.append(buffer, count); } while (SUCCEEDED(hr) && count && result.size() < (1u << 20));
        copy->Release(); return SUCCEEDED(hr) && result.size() < (1u << 20);
    }
    HRESULT edit(ITfContext *target, TfEditCookie cookie, const WireMessage &message) {
        if (target != context || !focused || message.activation != activation || session.empty() || message.session != session) return S_FALSE;
        if (message.op == L"mark" && message.revision != revision) return S_FALSE;
        if (message.op==L"inspect") { if (composition && !sameOwned(cookie)) invalidate(); return S_OK; }
        if (message.op!=L"begin" && composition && !sameOwned(cookie)) { invalidate(); return S_FALSE; }
        mutating = true;
        if (message.op==L"begin") { badge.hide(); typing->cancel(cookie); }
        ITfRange *range = nullptr;
        HRESULT hr = S_OK;
        if (!composition) {
            TF_SELECTION selection{}; ULONG count = 0;
            hr = context->GetSelection(cookie, TF_DEFAULT_SELECTION, 1, &selection, &count);
            if (SUCCEEDED(hr) && count == 1) {
                if (!safe(cookie, selection.range)) hr = E_ACCESSDENIED;
                original.clear();
                if (SUCCEEDED(hr) && !text(cookie, selection.range, original)) hr = E_FAIL;
                lastText = original;
                ITfContextComposition *compositions = nullptr;
                if (SUCCEEDED(hr)) hr = context->QueryInterface(IID_ITfContextComposition, reinterpret_cast<void**>(&compositions));
                if (SUCCEEDED(hr)) { hr = compositions->StartComposition(cookie, selection.range, this, &composition); compositions->Release(); if (SUCCEEDED(hr) && !composition) hr=E_ACCESSDENIED; }
                selection.range->Release();
            } else hr = E_FAIL;
        }
        if (SUCCEEDED(hr) && composition) hr = composition->GetRange(&range);
        if (SUCCEEDED(hr) && range && message.op != L"begin") {
            const auto &replacement = message.op == L"cancel" ? original : message.text;
            hr = range->SetText(cookie, 0, replacement.data(), static_cast<LONG>(replacement.size()));
            if (SUCCEEDED(hr)) lastText = replacement;
        }
        if (SUCCEEDED(hr) && range) {
            ITfRange *caret = nullptr;
            if (SUCCEEDED(range->Clone(&caret))) { caret->Collapse(cookie, TF_ANCHOR_END); TF_SELECTION selection{caret, {TF_AE_NONE, FALSE}}; context->SetSelection(cookie, 1, &selection); caret->Release(); }
        }
        drop(range);
        if (SUCCEEDED(hr) && message.op == L"begin") send(L"begun");
        if (SUCCEEDED(hr) && (message.op == L"commit" || message.op == L"cancel")) {
            hr = composition->EndComposition(cookie); drop(composition);
            if (SUCCEEDED(hr) && message.op == L"commit") send(L"committed");
            session.clear(); revision = 0;
        }
        mutating = false;
        if (FAILED(hr)) invalidate();
        return hr;
    }
    HRESULT keyEdit(ITfContext *target,const std::wstring &epoch,TfEditCookie cookie,const TypingCommand &command,bool &handled) {
        if (target!=context || epoch!=activation || !focused) return S_FALSE;
        mutating=true;
        if (command.kind==TypingCommand::Key) badge.hide();
        if (command.kind!=TypingCommand::Inspect) cancelVoice(cookie);
        const auto previousRevision=typing->revision;
        HRESULT hr=typing->process(cookie,command,handled);
        if (languageBar) languageBar->changed();
        mutating=false; if (typing->revision!=previousRevision) queueLayout(); return hr;
    }
    HRESULT layoutEdit(ITfContext *target,const std::wstring &epoch,uint64_t expected,uint64_t ticket,TfEditCookie cookie) {
        HRESULT hr=S_FALSE;
        if (layoutPending!=ticket) return hr;
        if (target==context && epoch==activation && focused) hr=typing->refreshLayout(cookie,expected);
        // A key/layout event arriving while the read session waited requests
        // exactly one more refresh of the latest revision, not an old range.
        if (layoutPending==ticket) { layoutPending=0; if (layoutDirty) queueLayout(); }
        return hr;
    }
    HRESULT STDMETHODCALLTYPE QueryInterface(REFIID id, void **out) override {
        if (!out) return E_POINTER;
        *out = nullptr;
#ifdef ARION_TSF_MANUAL_KEYS
        if (id==IID_ITfLangBarItemButton && languageBar) return languageBar->QueryInterface(id,out);
#endif
        if (id == IID_IUnknown || id == IID_ITfTextInputProcessor || id == IID_ITfTextInputProcessorEx) *out = static_cast<ITfTextInputProcessorEx*>(this);
        else if (id == IID_ITfThreadMgrEventSink) *out = static_cast<ITfThreadMgrEventSink*>(this);
        else if (id == IID_ITfThreadFocusSink) *out = static_cast<ITfThreadFocusSink*>(this);
        else if (id == IID_ITfCompositionSink) *out = static_cast<ITfCompositionSink*>(this);
        else if (id == IID_ITfTextEditSink) *out = static_cast<ITfTextEditSink*>(this);
        else if (id == IID_ITfTextLayoutSink) *out = static_cast<ITfTextLayoutSink*>(this);
        else if (id == IID_ITfKeyEventSink) *out = static_cast<ITfKeyEventSink*>(this);
        else return E_NOINTERFACE;
        AddRef(); return S_OK;
    }
    ULONG STDMETHODCALLTYPE AddRef() override { return ++references; }
    ULONG STDMETHODCALLTYPE Release() override { auto count = --references; if (!count) delete this; return count; }
    HRESULT STDMETHODCALLTYPE Activate(ITfThreadMgr *value, TfClientId id) override { return ActivateEx(value, id, 0); }
    HRESULT STDMETHODCALLTYPE ActivateEx(ITfThreadMgr *value, TfClientId id, DWORD flags) override {
        if (manager || !value || flags & TF_TMAE_SECUREMODE || !ordinaryHost()) return E_ACCESSDENIED;
        manager = value; manager->AddRef(); client = id; build = buildIdentity();
        // Embedded editors can own keyboard focus on a different thread or
        // process from their foreground top-level HWND. TSF owns this signal.
        // Observe before advising sinks so a synchronous focus notification
        // during registration remains the most recent state.
        BOOL threadFocused=FALSE;
        focused=SUCCEEDED(manager->IsThreadFocus(&threadFocused)) && threadFocused;
        typing->manager(manager);
        if (build.empty()) { Deactivate(); return E_FAIL; }
        WNDCLASSW definition{}; definition.lpfnWndProc = dispatch; definition.hInstance = module; definition.lpszClassName = ArionClass;
        if (!RegisterClassW(&definition) && GetLastError() != ERROR_CLASS_ALREADY_EXISTS) { Deactivate(); return E_FAIL; }
        window = CreateWindowExW(0, ArionClass, L"", 0, 0,0,0,0, HWND_MESSAGE, nullptr, module, this);
        if (!window) { Deactivate(); return E_FAIL; }
        ITfSource *source = nullptr; HRESULT hr = manager->QueryInterface(IID_ITfSource, reinterpret_cast<void**>(&source));
        if (SUCCEEDED(hr)) { hr = source->AdviseSink(IID_ITfThreadMgrEventSink, static_cast<ITfThreadMgrEventSink*>(this), &managerCookie); if (SUCCEEDED(hr)) hr = source->AdviseSink(IID_ITfThreadFocusSink, static_cast<ITfThreadFocusSink*>(this), &focusCookie); source->Release(); }
        if (FAILED(hr)) { Deactivate(); return hr; }
        hr=manager->QueryInterface(IID_ITfKeystrokeMgr,reinterpret_cast<void**>(&keys));
        if (SUCCEEDED(hr)) hr=keys->AdviseKeyEventSink(client,static_cast<ITfKeyEventSink*>(this),TRUE);
#ifdef ARION_TSF_TESTING
        if (FAILED(hr)) std::fprintf(stderr,"TSF key sink registration client=%lu HRESULT=%08lx\n",static_cast<unsigned long>(client),static_cast<unsigned long>(hr));
#endif
#ifdef ARION_TSF_MANUAL_KEYS
#ifndef ARION_TSF_TESTING
#error Manual key dispatch is restricted to the isolated test build
#endif
        // A manually instantiated editor client is not a system-activated TIP.
        // This fixture explicitly tests its key sink + actual TSF document
        // locks, not Windows foreground-sink installation or OS key dispatch.
        if (FAILED(hr)) { drop(keys); hr=S_OK; }
#endif
        if (FAILED(hr)) { Deactivate(); return hr; }
        PWSTR roaming=nullptr;
        if (SUCCEEDED(SHGetKnownFolderPath(FOLDERID_RoamingAppData,0,nullptr,&roaming))) {
            std::wstring user=std::wstring(roaming)+L"\\Arion\\keyboard"; CoTaskMemFree(roaming);
            settingsPath=user.substr(0,user.find_last_of(L'\\'))+L"\\keyboard.json";
#ifdef ARION_TSF_TESTING
            wchar_t isolated[32768]{}; DWORD count=GetEnvironmentVariableW(L"ARION_TSF_TEST_USER_DIRECTORY",isolated,32768);
            if (count && count<32768) { user=isolated; settingsPath=user+L"\\keyboard.json"; }
#endif
            keyboard.open(module,user);
            typing->options(user+L"\\options.json");
#ifdef ARION_TSF_TESTING
            if (!keyboard.failure.empty()) std::fprintf(stderr,"TSF keyboard startup: %s\n",keyboard.failure.c_str());
#endif
            refreshSettings(true);
        }
        languageBar=new LanguageBar([this] { return menuState(); },[this](KeyboardMenuAction action) { menuCommand(action); });
        hr=languageBar->attach(manager);
#ifdef ARION_TSF_TESTING
        std::fprintf(stderr,"TSF language bar registration=%08lx\n",static_cast<unsigned long>(hr));
#endif
#ifndef ARION_TSF_MANUAL_KEYS
        if (FAILED(hr)) { Deactivate(); return hr; }
#endif
#ifdef ARION_TSF_TESTING
        std::fprintf(stderr,"TSF fixture focus=%d\n",focused?1:0);
#endif
#ifdef ARION_TSF_MANUAL_KEYS
        focused=GetFocus()!=nullptr;
#endif
        ITfDocumentMgr *document = nullptr; if (SUCCEEDED(manager->GetFocus(&document))) { setContext(document); drop(document); }
        SetTimer(window, 1, 250, nullptr); return S_OK;
    }
    HRESULT STDMETHODCALLTYPE Deactivate() override {
        if (!manager) return S_OK;
        focused = false; setContext(nullptr); disconnect();
        badge.hide();
        if (languageBar) { languageBar->detach(); languageBar->Release(); languageBar=nullptr; }
        typing->manager(nullptr);
        ITfSource *source = nullptr; if (SUCCEEDED(manager->QueryInterface(IID_ITfSource, reinterpret_cast<void**>(&source)))) { if (managerCookie != TF_INVALID_COOKIE) source->UnadviseSink(managerCookie); if (focusCookie != TF_INVALID_COOKIE) source->UnadviseSink(focusCookie); source->Release(); }
        managerCookie = focusCookie = TF_INVALID_COOKIE;
        if (keys) { keys->UnadviseKeyEventSink(client); drop(keys); }
        if (window) { KillTimer(window, 1); KillTimer(window,2); MSG pending; while (PeekMessageW(&pending,window,Learn,Learn,PM_REMOVE)) delete reinterpret_cast<LearnCommit*>(pending.lParam); DestroyWindow(window); window = nullptr; UnregisterClassW(ArionClass,module); }
        keyboard.close(); arion_shift_reset(&shiftTap);
        drop(manager); client = TF_CLIENTID_NULL; return S_OK;
    }
    HRESULT STDMETHODCALLTYPE OnInitDocumentMgr(ITfDocumentMgr*) override { return S_OK; }
    HRESULT STDMETHODCALLTYPE OnUninitDocumentMgr(ITfDocumentMgr *document) override { ITfContext *top = nullptr; if (document) document->GetTop(&top); if (top == context) setContext(nullptr); drop(top); return S_OK; }
    HRESULT STDMETHODCALLTYPE OnSetFocus(ITfDocumentMgr *document, ITfDocumentMgr*) override {
#ifdef ARION_TSF_TESTING
        std::fprintf(stderr,"TSF document focus event=%p\n",static_cast<void*>(document));
#endif
        setContext(document); return S_OK;
    }
    HRESULT STDMETHODCALLTYPE OnPushContext(ITfContext *value) override { if (value != context) { ITfDocumentMgr *document = nullptr; if (SUCCEEDED(value->GetDocumentMgr(&document))) { setContext(document); drop(document); } } return S_OK; }
    HRESULT STDMETHODCALLTYPE OnPopContext(ITfContext *value) override { if (value == context) { ITfDocumentMgr *document = nullptr; if (SUCCEEDED(manager->GetFocus(&document))) { setContext(document); drop(document); } } return S_OK; }
    HRESULT STDMETHODCALLTYPE OnSetThreadFocus() override {
        focused=true;
        ITfDocumentMgr *document=nullptr;
        if (manager && SUCCEEDED(manager->GetFocus(&document))) { setContext(document); drop(document); }
        refreshSettings(true); connect(); return S_OK;
    }
    HRESULT STDMETHODCALLTYPE OnKillThreadFocus() override {
        if (!focused) { arion_shift_reset(&shiftTap); return S_OK; }
        focused=false;
        clearLayout();
        badge.hide(); typing->invalidate(); arion_shift_reset(&shiftTap);
        invalidate(); send(L"deactivate"); disconnect(); activation=context?newID():L""; return S_OK;
    }
    HRESULT STDMETHODCALLTYPE OnSetFocus(BOOL foreground) override {
        if (foreground) return OnSetThreadFocus();
        return OnKillThreadFocus();
    }
    HRESULT STDMETHODCALLTYPE OnTestKeyDown(ITfContext *target,WPARAM key,LPARAM details,BOOL *eaten) override { if (!eaten) return E_POINTER; *eaten=target==context && focused && wantsKey(nativeKey(key,details)); return S_OK; }
    HRESULT STDMETHODCALLTYPE OnTestKeyUp(ITfContext *target,WPARAM key,LPARAM details,BOOL *eaten) override { if (!eaten) return E_POINTER; auto native=nativeKey(key,details); *eaten=target==context && focused && typing->available() && native.shift && shiftTap.held; return S_OK; }
    HRESULT STDMETHODCALLTYPE OnKeyDown(ITfContext *target,WPARAM key,LPARAM details,BOOL *eaten) override {
        if (!eaten) return E_POINTER;
        *eaten=FALSE; if (target!=context || !focused) return S_OK;
        badge.hide();
        auto native=nativeKey(key,details);
        if (native.shift) { if (typing->available()) arion_shift_transition(&shiftTap,1,1,native.shiftSide,native.modifiers&~1,shiftTap.held && (!native.shiftSide || native.shiftSide==shiftTap.side),GetTickCount64()/1000.0); return S_OK; }
        arion_shift_interrupt(&shiftTap);
        TypingCommand command; command.key=native; *eaten=keyRequest(command); return S_OK;
    }
    HRESULT STDMETHODCALLTYPE OnKeyUp(ITfContext *target,WPARAM key,LPARAM details,BOOL *eaten) override {
        if (!eaten) return E_POINTER;
        *eaten=FALSE; if (target!=context || !focused) return S_OK;
        auto native=nativeKey(key,details);
        if (native.shift && arion_shift_transition(&shiftTap,1,0,native.shiftSide,native.modifiers&~1,0,GetTickCount64()/1000.0)) { TypingCommand command; command.kind=TypingCommand::ToggleASCII; *eaten=keyRequest(command); }
        return S_OK;
    }
    HRESULT STDMETHODCALLTYPE OnPreservedKey(ITfContext*,REFGUID,BOOL *eaten) override { if (!eaten) return E_POINTER; *eaten=FALSE; return S_OK; }
    HRESULT STDMETHODCALLTYPE OnCompositionTerminated(TfEditCookie cookie, ITfComposition *value) override {
        if (!mutating) { mutating=true; bool handled=typing->terminated(cookie,value); mutating=false; if (handled) return S_OK; }
        if (value==composition && !mutating) {
            mutating=true;
            ITfRange *range=nullptr;
            if (SUCCEEDED(value->GetRange(&range))) {
                std::wstring current;
                if (text(cookie,range,current) && current==lastText) range->SetText(cookie,0,original.data(),static_cast<LONG>(original.size()));
                range->Release();
            }
            drop(composition); mutating=false; invalidate();
        }
        return S_OK;
    }
    HRESULT STDMETHODCALLTYPE OnEndEdit(ITfContext*, TfEditCookie cookie, ITfEditRecord *record) override {
        if (mutating) return S_OK;
        if (typing->composing()) { if (typing->same(cookie)) return S_OK; typing->invalidate(); }
        // TSF may coalesce or omit the owner's own notification. Compare actual
        // owned text and caret rather than skipping the next notification.
        if (composition && !session.empty() && sameOwned(cookie)) return S_OK;
        if (!mutating) { BOOL selection = FALSE; record->GetSelectionStatus(&selection); IEnumTfRanges *ranges = nullptr; if (SUCCEEDED(record->GetTextAndPropertyUpdates(TF_GTP_INCL_TEXT, nullptr, 0, &ranges))) { ITfRange *range = nullptr; ULONG count = 0; ranges->Next(1, &range, &count); if (selection || count) { invalidate(); send(L"deactivate"); activation = newID(); send(L"activate"); } drop(range); ranges->Release(); } else invalidate(); }
        return S_OK;
    }
    HRESULT STDMETHODCALLTYPE OnLayoutChange(ITfContext *target,TfLayoutCode code,ITfContextView*) override {
        if (target==context && code==TF_LC_CHANGE) queueLayout();
        return S_OK;
    }
};
LayoutEdit::LayoutEdit(Service *service,ITfContext *target,const std::wstring &epoch,uint64_t expected,uint64_t serial):owner(service),context(target),activation(epoch),revision(expected),ticket(serial) { owner->AddRef(); context->AddRef(); }
LayoutEdit::~LayoutEdit() { drop(context); owner->Release(); }
HRESULT LayoutEdit::DoEditSession(TfEditCookie cookie) { return owner->layoutEdit(context,activation,revision,ticket,cookie); }
KeyEdit::KeyEdit(Service *service,ITfContext *target,const std::wstring &epoch,const TypingCommand &value):owner(service),context(target),activation(epoch),command(value) { owner->AddRef(); context->AddRef(); }
KeyEdit::~KeyEdit() { drop(context); owner->Release(); }
HRESULT KeyEdit::DoEditSession(TfEditCookie cookie) { return owner->keyEdit(context,activation,cookie,command,handled); }
Edit::Edit(Service *service, ITfContext *target, const WireMessage &value, ITfComposition *owned, const std::wstring &text, const std::wstring &saved):owner(service), context(target), message(value), cleanup(owned), expected(text), original(saved) { owner->AddRef(); context->AddRef(); if (cleanup) cleanup->AddRef(); }
Edit::~Edit() { drop(cleanup); drop(context); owner->Release(); }
HRESULT Edit::DoEditSession(TfEditCookie cookie) {
    if (!cleanup) return owner->edit(context, cookie, message);
    ITfRange *range = nullptr;
    if (SUCCEEDED(cleanup->GetRange(&range))) {
        std::wstring current;
        // A user edit wins over deferred cancellation. Never restore through a
        // range whose contents have changed since our last owned write.
        if (Service::text(cookie, range, current) && current == expected) range->SetText(cookie, 0, original.data(), static_cast<LONG>(original.size()));
        range->Release();
    }
    return cleanup->EndComposition(cookie);
}

class Factory final : public IClassFactory {
    CoreObjectLease lifetime;
    std::atomic<ULONG> references{1};
public:
    HRESULT STDMETHODCALLTYPE QueryInterface(REFIID id, void **out) override { if (!out) return E_POINTER; *out = nullptr; if (id != IID_IUnknown && id != IID_IClassFactory) return E_NOINTERFACE; *out = static_cast<IClassFactory*>(this); AddRef(); return S_OK; }
    ULONG STDMETHODCALLTYPE AddRef() override { return ++references; }
    ULONG STDMETHODCALLTYPE Release() override { auto count = --references; if (!count) delete this; return count; }
    HRESULT STDMETHODCALLTYPE CreateInstance(IUnknown *outer, REFIID id, void **out) override { if (outer) return CLASS_E_NOAGGREGATION; auto service = new Service; HRESULT hr = service->QueryInterface(id, out); service->Release(); return hr; }
    HRESULT STDMETHODCALLTYPE LockServer(BOOL lock) override { if (lock) ++objects; else --objects; return S_OK; }
};
extern "C" __declspec(dllexport) HRESULT __stdcall DllGetClassObject(REFCLSID clsid, REFIID id, void **out) { if (clsid != ArionService) return CLASS_E_CLASSNOTAVAILABLE; auto factory = new Factory; HRESULT hr = factory->QueryInterface(id, out); factory->Release(); return hr; }
extern "C" __declspec(dllexport) HRESULT __stdcall DllCanUnloadNow() { return objects.load() == 0 ? S_OK : S_FALSE; }
BOOL WINAPI DllMain(HINSTANCE instance, DWORD reason, LPVOID) { if (reason == DLL_PROCESS_ATTACH) { module = instance; DisableThreadLibraryCalls(instance); } return TRUE; }
