#pragma once
#include "keyboard_client.h"
#include "candidate_window.h"
#include "native_keys.h"
#include "../shared/menu.h"
#include "keyboard_settings.h"
#include "candidate_ui.h"
#include "keyboard_options.h"
#include "native_copy.h"
#include <climits>
#include <cctype>

struct TypingCommand {
    enum Kind { Key,Choose,Previous,Next,Inspect,ToggleASCII,ShiftASCII,TogglePunctuation,ToggleWidth } kind=Key;
    NativeKey key;
    size_t engineIndex=0;
    uint64_t revision=0;
};

// All methods run on the editor's COM apartment. Engine mutations require the
// caller's granted write edit session; test-key callbacks only read state.
class Typing {
    KeyboardClient &runtime;
    ITfCompositionSink *sink;
    ITfContext *context=nullptr;
    ITfComposition *composition=nullptr;
    ITfRange *candidateAnchor=nullptr;
    void clearAnchor() { if (candidateAnchor) { candidateAnchor->Release(); candidateAnchor=nullptr; } }
    uintptr_t session=0;
    uint64_t generation=0;
    std::wstring original,lastText;
    size_t caret=0;
    ArionPinyinMenu menu{};
    CandidateWindow &panel;
    CandidateUI *accessible=new CandidateUI;
    bool ascii=false,fullWidth=false,englishPunctuation=false;
    bool healthy=true;
    size_t pageSize=9;
    int textSize=16;
    CandidatePalette palette;
    bool settingsChanged=false,spellingChanged=false;
    bool emojiSuggestions=true,emojiInComposition=false,typoCorrection=false,neuralRanking=false;
    uint64_t rankingRevision=0;
    std::vector<size_t> rankingSlots;
    std::vector<std::string> rankingIDs;
    uint32_t fuzzyPinyin=0;
    std::wstring optionsPath;
    std::function<bool(TfEditCookie,ITfRange*)> safe;
    std::function<bool(TfEditCookie,ITfRange*,std::wstring&)> read;
    std::function<void(ITfComposition*,const std::wstring&,const std::wstring&)> revoke;
    std::function<void(const std::string&,const std::string&,uint64_t)> learn;
    std::function<void(TfEditCookie,NativeCopy)> modeChanged;
    std::string previousCommit;
    ULONGLONG previousTime=0;
    std::string previousContext() { return GetTickCount64()-previousTime<120000?runtime.contextOf(previousCommit):std::string{}; }
    static int rank(void *target,const char *const *texts,size_t count,const unsigned char *eligible,size_t *indices) {
        auto self=static_cast<Typing*>(target);
        if (self->runtime.order(texts,count,eligible,self->previousContext(),indices,self->rankingFailure)) return 1;
        // Context learning is optional. Keep the real engine order and surface
        // its separate error; a ranking failure cannot disable normal Pinyin.
        for (size_t i=0;i<count;++i) indices[i]=i;
        return 1;
    }
    static int candidateKind(void *target,uintptr_t current,size_t index,ArionCandidateMetadata *metadata) {
        return static_cast<Typing*>(target)->runtime.candidate(current,index,*metadata);
    }
    bool buildMenu() {
        return arion_menu_build_typed(&menu,runtime.api(),session,pageSize,rank,this,candidateKind,this)!=0 && runtime.ready();
    }
    bool select(size_t index) {
        if (index>=menu.count || !arion_pinyin_select(runtime.api(),session,menu.indices[index],0)) return false;
        emojiInComposition=emojiInComposition || menu.metadata[index].emoji; return true;
    }
    void applyCandidatePreferences() {
        arion_spelling_apply(runtime.api(),session,typoCorrection,fuzzyPinyin);
        if (session && (runtime.api()->get_option(session,"arion_emoji")!=0)!=emojiSuggestions)
            runtime.api()->set_option(session,"arion_emoji",emojiSuggestions);
    }
    void loadOptions() {
        KeyboardOptions options;
        if (!readKeyboardOptions(optionsPath,options)) { modeFailure=nativeCopy(NativeCopy::OptionsFailed); return; }
        if (modeFailure==nativeCopy(NativeCopy::OptionsFailed)) modeFailure.clear();
        if (fullWidth!=options.fullWidth || englishPunctuation!=options.englishPunctuation) {
            fullWidth=options.fullWidth; englishPunctuation=options.englishPunctuation;
            if (session && runtime.api() && generation==runtime.generation()) arion_pinyin_set_options(runtime.api(),session,ascii,fullWidth,englishPunctuation);
        }
    }

    bool sessionReady() {
        if (!healthy || FAILED(panel.error()) || !runtime.ready()) return false;
        if (!session) {
            session=arion_pinyin_create_session(runtime.api());
            uint64_t restoredRevision=0;
            if (session && !runtime.restorePersonalVocabulary(session,restoredRevision)) {
                runtime.api()->destroy_session(session);session=0;
            }
            generation=runtime.generation();
            if (session) { arion_pinyin_set_options(runtime.api(),session,ascii,fullWidth,englishPunctuation); applyCandidatePreferences(); }
        }
        return session!=0;
    }
    bool position(TfEditCookie cookie,ITfRange *range,size_t offset,ITfRange **output) {
        if (!output || offset>static_cast<size_t>(LONG_MAX)) return false;
        *output=nullptr; ITfRange *next=nullptr;
        if (FAILED(range->Clone(&next))) return false;
        LONG moved=0; HRESULT hr=next->ShiftStart(cookie,static_cast<LONG>(offset),&moved,nullptr);
        if (SUCCEEDED(hr) && moved==static_cast<LONG>(offset)) hr=next->Collapse(cookie,TF_ANCHOR_START); else hr=E_FAIL;
        if (FAILED(hr)) { next->Release(); return false; }
        *output=next; return true;
    }
    HRESULT selection(TfEditCookie cookie,ITfRange *range,size_t offset) {
        ITfRange *point=nullptr;
        if (!position(cookie,range,offset,&point)) return E_FAIL;
        TF_SELECTION selected{point,{TF_AE_NONE,FALSE}};
        HRESULT hr=context->SetSelection(cookie,1,&selected); point->Release(); return hr;
    }
    HRESULT acquire(TfEditCookie cookie) {
        if (composition) {
            ITfRange *range=nullptr;
            bool allowed=SUCCEEDED(composition->GetRange(&range)) && safe(cookie,range) && same(cookie);
            if (range) range->Release();
            return allowed?S_OK:E_ACCESSDENIED;
        }
        TF_SELECTION selected{}; ULONG count=0;
        HRESULT hr=context->GetSelection(cookie,TF_DEFAULT_SELECTION,1,&selected,&count);
        if (FAILED(hr) || count!=1) return E_FAIL;
        if (!safe(cookie,selected.range) || !read(cookie,selected.range,original)) hr=E_ACCESSDENIED;
        ITfContextComposition *owner=nullptr;
        if (SUCCEEDED(hr)) hr=context->QueryInterface(IID_ITfContextComposition,reinterpret_cast<void**>(&owner));
        if (SUCCEEDED(hr)) { hr=owner->StartComposition(cookie,selected.range,sink,&composition); owner->Release(); if (SUCCEEDED(hr) && !composition) hr=E_ACCESSDENIED; }
        if (SUCCEEDED(hr) && !candidateAnchor) {
            hr=selected.range->Clone(&candidateAnchor);
            if (SUCCEEDED(hr)) hr=candidateAnchor->Collapse(cookie,TF_ANCHOR_START);
            if (SUCCEEDED(hr)) hr=candidateAnchor->SetGravity(cookie,TF_GRAVITY_BACKWARD,TF_GRAVITY_BACKWARD);
        }
        selected.range->Release();
        if (SUCCEEDED(hr)) { lastText=original; caret=original.size(); }
        return hr;
    }
    bool allowed(TfEditCookie cookie) {
        TF_SELECTION selected{}; ULONG count=0;
        HRESULT hr=context->GetSelection(cookie,TF_DEFAULT_SELECTION,1,&selected,&count);
        bool result=SUCCEEDED(hr) && count==1 && safe(cookie,selected.range);
        if (count) selected.range->Release();
        return result;
    }
    // 0 passes to the engine, 1 changes only the shared menu, 2 selects text.
    int menuKey(const NativeKey &key) {
        if (!menu.count || ascii) return 0;
        if (key.symbol>='1' && key.symbol<='9') {
            size_t index=menu.page*menu.page_size+static_cast<size_t>(key.symbol-'1');
            if (index<menu.count && static_cast<size_t>(key.symbol-'1')<menu.page_size) return select(index)?2:-1;
            return 1;
        }
        if (key.symbol==' ') return select(menu.highlight)?2:-1;
        bool before=key.symbol=='-' || key.symbol=='[' || key.symbol==0xff55 || key.symbol==0xff52;
        bool after=key.symbol=='=' || key.symbol==']' || key.symbol==0xff56 || key.symbol==0xff54;
        if (before) { if (menu.expanded && menu.page==0) arion_menu_grid(&menu,runtime.api(),session,0); else arion_menu_turn(&menu,runtime.api(),session,-1); return 1; }
        if (after) {
            if (menu.expanded) arion_menu_turn(&menu,runtime.api(),session,1);
            else if (arion_menu_load(&menu,runtime.api(),session,(menu.page+1)*menu.page_size) && (menu.page+1)*menu.page_size<menu.count) {
                menu.expanded=1; menu.top_row=menu.page;
                if (!arion_menu_turn(&menu,runtime.api(),session,1)) menu.expanded=0;
            }
            return 1;
        }
        if (menu.expanded && (key.symbol==0xff51 || key.symbol==0xff53)) { arion_menu_move(&menu,key.symbol==0xff51?-1:1); return 1; }
        return 0;
    }
    void show() {
        if (!composition || !menu.count || !arion_menu_prepare(&menu,runtime.api(),session)) { panel.hide(); accessible->hide(); return; }
        std::vector<std::wstring> strings;
        strings.reserve(menu.count);
        for (size_t i=0;i<menu.count;++i) strings.push_back(wide(menu.texts[i]));
        accessible->update(context,strings,menu.highlight,menu.page_size,menu.page);
        // Service queues one read edit after this write session completes.
        // Do not hide the panel by querying layout invalidated by SetText.
    }
public:
    // Geometry can lag SetText in browser text stores. This path runs under a
    // read lock after layout notification and never calls the engine or writes
    // the document. The current revision must still own its exact text/caret.
    HRESULT refreshLayout(TfEditCookie cookie,uint64_t expected) {
        if (!context || !composition || !menu.count || expected!=revision) return S_FALSE;
        if (session && generation!=runtime.generation()) { invalidate(); return S_FALSE; }
        ITfRange *range=nullptr,*point=nullptr; ITfContextView *view=nullptr; RECT area{}; BOOL clipped=FALSE; HWND owner=nullptr;
        HRESULT hr=composition->GetRange(&range);
        if (SUCCEEDED(hr) && (!safe(cookie,range) || !same(cookie))) {
            range->Release(); invalidate(); return E_ACCESSDENIED;
        }
        if (SUCCEEDED(hr)) hr=candidateAnchor?candidateAnchor->Clone(&point):E_FAIL;
        if (SUCCEEDED(hr)) hr=context->GetActiveView(&view);
        if (SUCCEEDED(hr)) hr=view->GetTextExt(cookie,point,&area,&clipped);
        if (SUCCEEDED(hr)) hr=view->GetWnd(&owner);
        if (view) view->Release();
        if (point) point->Release();
        if (range) range->Release();
#ifdef ARION_TSF_TESTING
        std::fprintf(stderr,"TSF candidate extent revision=%llu hr=%08lx clipped=%d rect=%ld,%ld,%ld,%ld\n",static_cast<unsigned long long>(revision),static_cast<unsigned long>(hr),clipped?1:0,area.left,area.top,area.right,area.bottom);
#endif
        if (FAILED(hr) || clipped || area.bottom<=area.top || area.right<area.left) { panel.hide(); accessible->bind(nullptr); return FAILED(hr)?hr:S_FALSE; }
        if (area.right==area.left) ++area.right;
        std::vector<NativeCandidate> candidates;
        bool native=accessible->nativeAllowed();
        size_t first=(menu.expanded?menu.top_row:menu.page)*menu.page_size;
        size_t count=menu.expanded?std::min(menu.count-first,arion_menu_row_count(&menu)*menu.page_size):arion_menu_visible_count(&menu);
        for (size_t i=0;i<count;++i) candidates.push_back({wide(menu.texts[first+i]),L"",menu.indices[first+i],menu.metadata[first+i].emoji!=0});
        if (native) {
            if (!panel.show(owner,area,candidates,menu.highlight-first,menu.page>0,!menu.exhausted || first+count<menu.count,revision,palette,menu.page_size,menu.expanded!=0,textSize)) {
                accessible->bind(nullptr); runtime.failure="keyboard candidate rendering unavailable"; return E_FAIL;
            }
            accessible->bind(panel.handle());
        }
        else panel.hide();
        return S_OK;
    }
private:
    HRESULT publish(TfEditCookie cookie) {
        ArionPinyinSnapshot snapshot{};
        if (arion_pinyin_snapshot(runtime.api(),session,&snapshot)!=0) return E_FAIL;
        ascii=snapshot.ascii!=0; ++revision;
        std::wstring committed=wide(snapshot.commit?snapshot.commit:""),preedit=wide(snapshot.preedit?snapshot.preedit:"");
        bool delivered=false;
        size_t nextCaret=snapshot.cursor_utf16;
        arion_pinyin_snapshot_dispose(&snapshot);
        ITfRange *range=nullptr; HRESULT hr=composition?composition->GetRange(&range):E_FAIL;
        if (SUCCEEDED(hr) && !committed.empty()) {
            hr=range->SetText(cookie,0,committed.data(),static_cast<LONG>(committed.size()));
            if (SUCCEEDED(hr)) {
                lastText=committed; hr=selection(cookie,range,committed.size());
                if (SUCCEEDED(hr)) hr=composition->EndComposition(cookie);
                if (SUCCEEDED(hr)) {
                    delivered=true;
                    if (!emojiInComposition) learn(utf8(committed),previousContext(),runtime.generation());
                    previousCommit=emojiInComposition?std::string{}:utf8(committed); previousTime=GetTickCount64();
                    emojiInComposition=false;
                    composition->Release(); composition=nullptr; original.clear(); lastText.clear();
                }
            }
        }
        if (range) { range->Release(); range=nullptr; }
        if (SUCCEEDED(hr) && !preedit.empty()) {
            if (!composition) hr=acquire(cookie);
            if (SUCCEEDED(hr)) hr=composition->GetRange(&range);
            if (SUCCEEDED(hr)) hr=range->SetText(cookie,0,preedit.data(),static_cast<LONG>(preedit.size()));
            if (SUCCEEDED(hr)) { lastText=preedit; caret=std::min(nextCaret,preedit.size()); hr=selection(cookie,range,caret); }
            if (range) range->Release();
            if (SUCCEEDED(hr) && !buildMenu()) hr=E_FAIL;
            if (SUCCEEDED(hr)) show();
        } else if (SUCCEEDED(hr)) cancel(cookie);
        if (FAILED(hr)) recoveryText=(delivered?std::wstring{}:committed)+preedit;
        return hr;
    }
public:
    std::function<void(const std::string&,uint64_t)> rankingChanged;
    void freezeRanking() {
        rankingSlots.clear(); rankingIDs.clear(); ++rankingRevision;
        if (neuralRanking && rankingChanged) rankingChanged({},rankingRevision);
    }
    void requestRanking() {
        auto before=previousContext();
        if (!neuralRanking || !rankingChanged || !composition || ascii || menu.page || menu.expanded ||
            menu.highlight || menu.count<2 || before.size()>2048) return;
        const char *input=runtime.api()->get_input(session);
        if (!input || !*input || std::strlen(input)>256 || runtime.api()->get_caret_pos(session)!=std::strlen(input) || !runtime.han(menu.texts[0])) return;
        std::string candidates;
        for (size_t i=0;i<std::min(menu.count,size_t{27});++i) {
            const auto &item=menu.metadata[i];
            if (item.emoji || item.start!=menu.metadata[0].start || item.end!=menu.metadata[0].end || item.end<=item.start || !runtime.han(menu.texts[i])) continue;
            auto id="c"+std::to_string(menu.indices[i]);
            if (!candidates.empty()) candidates+=",";
            candidates+="{\"id\":"+::quoted(wide(id))+",\"text\":"+::quoted(wide(menu.texts[i]))+",\"source\":"+::quoted(wide(item.type))+",\"start\":"+std::to_string(item.start)+",\"end\":"+std::to_string(item.end)+"}";
            rankingSlots.push_back(i); rankingIDs.push_back(id);
        }
        if (rankingIDs.size()<2) { rankingSlots.clear();rankingIDs.clear();return; }
        rankingChanged("\"input\":"+::quoted(wide(input))+",\"before\":"+::quoted(wide(before))+",\"candidates\":["+candidates+"]",rankingRevision);
    }
    HRESULT applyRanking(TfEditCookie cookie,const WireMessage &message) {
        KeyboardLease lease(runtime);
        if (!session || generation!=runtime.generation() || !neuralRanking || rankingIDs.empty() || message.revision!=rankingRevision || !same(cookie)) return S_FALSE;
        if (!message.rankingOK) return S_FALSE;
        if (message.rankedIDs.size()!=rankingIDs.size()) return S_FALSE;
        auto selected=std::find(rankingIDs.begin(),rankingIDs.end(),message.rankedIDs[0]);
        if (selected==rankingIDs.end()) return S_FALSE;
        size_t chosen=static_cast<size_t>(selected-rankingIDs.begin());
        auto expected=rankingIDs; auto id=expected[chosen]; expected.erase(expected.begin()+chosen); expected.insert(expected.begin(),id);
        if (expected!=message.rankedIDs || !buildMenu() || !arion_menu_promote_slots(&menu,rankingSlots.data(),rankingSlots.size(),chosen)) return S_FALSE;
        ++revision;
        show(); return S_OK;
    }
    uint64_t revision=0;
    std::string rankingFailure;
    std::wstring modeFailure,recoveryText;
    Typing(KeyboardClient &client,ITfCompositionSink *owner,CandidateWindow &candidates,
        std::function<bool(TfEditCookie,ITfRange*)> allowed,
        std::function<bool(TfEditCookie,ITfRange*,std::wstring&)> text,
        std::function<void(ITfComposition*,const std::wstring&,const std::wstring&)> cleanup,
        std::function<void(const std::string&,const std::string&,uint64_t)> learning,
        std::function<void(TfEditCookie,NativeCopy)> mode):runtime(client),sink(owner),panel(candidates),safe(std::move(allowed)),read(std::move(text)),revoke(std::move(cleanup)),learn(std::move(learning)),modeChanged(std::move(mode)) {}
    ~Typing() { invalidate(); if (context) context->Release(); accessible->Release(); }
    void manager(ITfThreadMgr *value) { accessible->attach(value); }
    bool composing() const { return composition!=nullptr; }
    bool available() { if (FAILED(panel.error())) { runtime.failure="keyboard Emoji rendering unavailable"; return false; } return healthy && runtime.ready(); }
    // Idle vocabulary updates do not read or mutate target text and therefore
    // need no TSF edit cookie; the caller remains on the owning COM apartment.
    const char *vocabulary(const WireMessage &message,uint64_t &version) {
        if (composing()) return "busy";
        KeyboardLease lease(runtime);
        if (session && generation!=runtime.generation()) session=0;
        if (!sessionReady()) return "rejected";
        if (message.op==L"vocabulary_query")
            return runtime.restorePersonalVocabulary(session,version)?"ok":"rejected";
        if (message.op!=L"vocabulary_publish") return "rejected";
        std::vector<ArionPersonalTerm> terms;
        terms.reserve(message.terms.size());
        for (const auto &term:message.terms) terms.push_back({term.id.c_str(),term.text.c_str(),term.reading.c_str()});
        if (!runtime.publishPersonalVocabulary(session,message.vocabularyRevision,terms.data(),terms.size())) return "rejected";
        version=message.vocabularyRevision; return "ok";
    }
    bool isASCII() const { return ascii; }
    bool isFullWidth() const { return fullWidth; }
    bool isEnglishPunctuation() const { return englishPunctuation; }
    void options(const std::wstring &path) { optionsPath=path; KeyboardOptions value; if (readKeyboardOptions(path,value)) { fullWidth=value.fullWidth; englishPunctuation=value.englishPunctuation; } else modeFailure=nativeCopy(NativeCopy::OptionsFailed); }
    void configure(const KeyboardSettings &settings) {
        if (neuralRanking!=settings.neuralRanking) freezeRanking();
        neuralRanking=settings.neuralRanking;
        spellingChanged=spellingChanged || typoCorrection!=settings.typoCorrection || fuzzyPinyin!=settings.fuzzyPinyin;
        settingsChanged=settingsChanged || pageSize!=settings.pageSize || textSize!=settings.textSize || emojiSuggestions!=settings.emojiSuggestions || typoCorrection!=settings.typoCorrection || fuzzyPinyin!=settings.fuzzyPinyin || memcmp(&palette,&settings.palette,sizeof(palette))!=0;
        pageSize=settings.pageSize; textSize=settings.textSize; palette=settings.palette;
        emojiSuggestions=settings.emojiSuggestions;
        typoCorrection=settings.typoCorrection; fuzzyPinyin=settings.fuzzyPinyin;
    }
    void attach(ITfContext *value) { if (value==context) return; invalidate(); healthy=true; previousCommit.clear(); if (context) context->Release(); context=value; if (context) context->AddRef(); }
    bool same(TfEditCookie cookie) {
        if (!composition || !context) return false;
        ITfRange *range=nullptr,*expected=nullptr; TF_SELECTION selected{}; ULONG count=0; bool valid=false;
        if (SUCCEEDED(composition->GetRange(&range)) && position(cookie,range,caret,&expected) && SUCCEEDED(context->GetSelection(cookie,TF_DEFAULT_SELECTION,1,&selected,&count)) && count==1) {
            LONG start=1,end=1; std::wstring current;
            valid=safe(cookie,range) && SUCCEEDED(selected.range->CompareStart(cookie,expected,TF_ANCHOR_START,&start)) && SUCCEEDED(selected.range->CompareEnd(cookie,expected,TF_ANCHOR_START,&end)) && !start && !end && read(cookie,range,current) && current==lastText;
        }
        if (count) selected.range->Release();
        if (expected) expected->Release();
        if (range) range->Release();
        return valid;
    }
    void cancel(TfEditCookie cookie) {
        freezeRanking();
        KeyboardLease lease(runtime);
        if (composition) {
            ITfRange *range=nullptr;
            if (SUCCEEDED(composition->GetRange(&range))) { std::wstring current; if (read(cookie,range,current) && current==lastText) range->SetText(cookie,0,original.data(),static_cast<LONG>(original.size())); range->Release(); }
            composition->EndComposition(cookie); composition->Release(); composition=nullptr;
        }
        if (session && runtime.api() && generation==runtime.generation()) arion_pinyin_clear(runtime.api(),session);
        arion_menu_dispose(&menu); panel.hide(); accessible->hide(); clearAnchor(); original.clear(); lastText.clear(); caret=0; ++revision;
        emojiInComposition=false;
    }
    void invalidate() {
        freezeRanking();
        KeyboardLease lease(runtime);
        if (composition) { revoke(composition,lastText,original); composition->Release(); composition=nullptr; }
        if (session && runtime.api() && generation==runtime.generation()) arion_pinyin_destroy_session(runtime.api(),session);
        session=0; arion_menu_dispose(&menu); panel.hide(); accessible->hide(); clearAnchor(); original.clear(); lastText.clear(); ++revision;
        emojiInComposition=false;
    }
    bool terminated(TfEditCookie cookie,ITfComposition *value) {
        if (value!=composition) return false;
        KeyboardLease lease(runtime);
        ITfRange *range=nullptr;
        if (SUCCEEDED(composition->GetRange(&range))) {
            std::wstring current;
            if (read(cookie,range,current) && current==lastText) {
                HRESULT restored=range->SetText(cookie,0,original.data(),static_cast<LONG>(original.size()));
#ifdef ARION_TSF_TESTING
                std::fprintf(stderr,"TSF keyboard termination restore=%08lx\n",static_cast<unsigned long>(restored));
#else
                (void)restored;
#endif
            }
            range->Release();
        }
        composition->Release(); composition=nullptr;
        if (session && runtime.api() && generation==runtime.generation()) arion_pinyin_clear(runtime.api(),session);
        arion_menu_dispose(&menu); panel.hide(); accessible->hide(); clearAnchor(); original.clear(); lastText.clear(); ++revision;
        emojiInComposition=false;
        return true;
    }
    HRESULT process(TfEditCookie cookie,const TypingCommand &command,bool &handled) {
        handled=false;
        if (!context) return E_FAIL;
        KeyboardLease lease(runtime);
        // Another COM service instance may have reset the shared engine. Restore
        // only our unchanged marked bytes before discarding the expired handle.
        if (session && generation!=runtime.generation()) { cancel(cookie); session=0; }
        loadOptions();
        if (spellingChanged) {
            if (composition) cancel(cookie);
            applyCandidatePreferences();
            spellingChanged=false;
        }
        if (command.kind==TypingCommand::Inspect) {
            if (composition && !same(cookie)) invalidate();
            else if (settingsChanged) {
                freezeRanking();
                applyCandidatePreferences();
                if (composition) { ++revision; if (buildMenu()) show(); else { healthy=false; cancel(cookie); return E_FAIL; } }
            }
            settingsChanged=false; return S_OK;
        }
        freezeRanking();
        if (command.kind!=TypingCommand::Key && command.kind!=TypingCommand::ToggleASCII && command.kind!=TypingCommand::ShiftASCII && command.kind!=TypingCommand::TogglePunctuation && command.kind!=TypingCommand::ToggleWidth && command.revision!=revision) return S_FALSE;
        if (composition && !same(cookie)) { invalidate(); return S_FALSE; }
        if (settingsChanged) {
            applyCandidatePreferences(); settingsChanged=false;
            if (composition) {
                ++revision;
                if (!buildMenu()) { healthy=false; cancel(cookie); return E_FAIL; }
                show();
                if (command.kind==TypingCommand::Choose || command.kind==TypingCommand::Previous || command.kind==TypingCommand::Next) return S_FALSE;
            }
        }
        ArionKeyAction action=arion_key_action(command.key.symbol,command.key.modifiers,ascii);
        if (command.kind==TypingCommand::Key && action==ARION_KEY_PASS) { cancel(cookie); return S_OK; }
        if (command.kind==TypingCommand::ToggleASCII || command.kind==TypingCommand::ShiftASCII || action==ARION_KEY_TOGGLE_ASCII) {
            if (!available() || !allowed(cookie)) return E_ACCESSDENIED;
            if (command.kind==TypingCommand::ShiftASCII && !ascii && composition) {
                if (!arion_pinyin_commit_raw(runtime.api(),session)) return E_FAIL;
                HRESULT committed=publish(cookie);
                if (FAILED(committed)) { healthy=false; cancel(cookie); return committed; }
            }
            cancel(cookie); previousCommit.clear(); ascii=!ascii;
            if (session) arion_pinyin_set_options(runtime.api(),session,ascii,fullWidth,englishPunctuation);
            handled=true; modeChanged(cookie,ascii?NativeCopy::EnglishBadge:NativeCopy::ChineseBadge); return S_OK;
        }
        if (command.kind==TypingCommand::TogglePunctuation || action==ARION_KEY_TOGGLE_PUNCTUATION || command.kind==TypingCommand::ToggleWidth) {
            if (!available() || !allowed(cookie)) return E_ACCESSDENIED;
            KeyboardOptions next;
            if (!toggleKeyboardOption(optionsPath,command.kind==TypingCommand::ToggleWidth?KeyboardOption::FullWidth:KeyboardOption::EnglishPunctuation,next)) { modeFailure=nativeCopy(NativeCopy::OptionsFailed); handled=true; return S_OK; }
            modeFailure.clear(); cancel(cookie); previousCommit.clear(); fullWidth=next.fullWidth; englishPunctuation=next.englishPunctuation;
            if (session) arion_pinyin_set_options(runtime.api(),session,ascii,fullWidth,englishPunctuation);
            handled=true;
            modeChanged(cookie,command.kind==TypingCommand::ToggleWidth?(fullWidth?NativeCopy::FullWidthBadge:NativeCopy::HalfWidthBadge):(englishPunctuation?NativeCopy::EnglishPunctuationBadge:NativeCopy::ChinesePunctuationBadge));
            return S_OK;
        }
        HRESULT hr=acquire(cookie); if (FAILED(hr)) return hr;
        if (!sessionReady()) { recoveryText=lastText; healthy=false; cancel(cookie); return E_FAIL; }
        if (command.kind==TypingCommand::Choose) {
            size_t index=0;
            while (index<menu.count && menu.indices[index]!=command.engineIndex) ++index;
            if (index==menu.count) return S_FALSE;
            handled=select(index);
        }
        else if (command.kind==TypingCommand::Previous || command.kind==TypingCommand::Next) { handled=arion_menu_turn(&menu,runtime.api(),session,command.kind==TypingCommand::Previous?-1:1)!=0; if (handled) { ++revision; show(); } return S_OK; }
        else {
            int menuAction=menuKey(command.key);
            if (menuAction<0) { recoveryText=lastText; healthy=false; cancel(cookie); runtime.failure="keyboard candidate selection failed"; return E_FAIL; }
            if (menuAction) {
                handled=true;
                if (menuAction==1) { ++revision; show(); return S_OK; }
            } else {
                int key=arion_key_normalize(command.key.symbol,command.key.modifiers,ascii);
                if (composition && menu.highlight<menu.count && menu.metadata[menu.highlight].emoji && key>=0 && key<128 && std::ispunct(static_cast<unsigned char>(key))) emojiInComposition=true;
                handled=arion_pinyin_process_key(runtime.api(),session,key,command.key.modifiers)!=0;
                if (emojiSuggestions && (key==0xff50 || key==0xff57 || key==0xff51 || key==0xff53) && !runtime.refreshEmojiCandidates(session)) {
                    recoveryText=lastText; healthy=false; cancel(cookie); runtime.failure="keyboard Emoji caret refresh failed"; return E_FAIL;
                }
            }
        }
        hr=publish(cookie);
        if (SUCCEEDED(hr) && command.kind==TypingCommand::Key && ((command.key.symbol>='a' && command.key.symbol<='z') || (command.key.symbol>='A' && command.key.symbol<='Z') || command.key.symbol==0xff08 || command.key.symbol==0xffff)) requestRanking();
        if (FAILED(hr)) { if (recoveryText.empty()) recoveryText=lastText; healthy=false; cancel(cookie); runtime.failure="keyboard composition unavailable"; }
        return hr;
    }
};
