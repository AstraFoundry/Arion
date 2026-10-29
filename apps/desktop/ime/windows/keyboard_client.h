#pragma once
#include "keyboard_payload.h"
#include "keyboard_runtime.h"
#include "keyboard_options.h"

// No Rime imports enter the COM service. The optional module and its unique
// engine dependency load only after immutable manifest validation succeeds.
class KeyboardClient {
    HMODULE module=nullptr;
    RimeApi *runtime=nullptr;
    RimeApi *(*start)(const char*,const char*,char*,size_t)=nullptr;
    void (*enter)()=nullptr;
    void (*leave)()=nullptr;
    void (*stop)()=nullptr;
    int (*error)(char*,size_t)=nullptr;
    int (*reset)(char*,size_t)=nullptr;
    uint64_t (*currentGeneration)()=nullptr;
    int (*orderCandidates)(const char *const*,size_t,const unsigned char*,const char*,size_t*,char*,size_t)=nullptr;
    int (*learnCommit)(const char*,const char*,char*,size_t)=nullptr;
    int (*commitContext)(const char*,char*,size_t)=nullptr;
    int (*candidateMetadata)(uintptr_t,size_t,ArionCandidateMetadata*)=nullptr;
    int (*refreshEmoji)(uintptr_t)=nullptr;
    int (*restorePersonal)(uintptr_t,uint64_t*)=nullptr;
    int (*publishPersonal)(uintptr_t,uint64_t,const ArionPersonalTerm*,size_t)=nullptr;
    template<typename T> bool symbol(T &target,const char *name) {
        FARPROC address=GetProcAddress(module,name); static_assert(sizeof(target)==sizeof(address)); std::memcpy(&target,&address,sizeof(target)); return target!=nullptr;
    }
    static bool sameModule(const wchar_t *name,const std::wstring &expected) {
        HMODULE existing=GetModuleHandleW(name); if (!existing) return true;
        wchar_t path[32768]{}; DWORD length=GetModuleFileNameW(existing,path,32768);
        return length && length<32768 && _wcsicmp(path,expected.c_str())==0;
    }
public:
    std::string failure;
    ~KeyboardClient() { close(); }
    KeyboardClient()=default;
    KeyboardClient(const KeyboardClient&)=delete;
    KeyboardClient &operator=(const KeyboardClient&)=delete;
    bool open(HINSTANCE service,const std::wstring &userDirectory) {
        if (runtime) return true;
        wchar_t path[32768]{}; DWORD length=GetModuleFileNameW(service,path,32768);
        if (!length || length==32768) { failure="keyboard service path"; return false; }
        std::wstring directory=std::wstring(path).substr(0,std::wstring(path).find_last_of(L'\\'))+L"\\keyboard";
        std::vector<KeyboardPayloadFile> files;
        if (FAILED(verifyKeyboardPayload(directory,wide(ARION_KEYBOARD_MANIFEST_SHA256),files))) { failure="keyboard payload unverified"; return false; }
        auto library=directory+L"\\lib\\ArionKeyboard.dll",engine=directory+L"\\lib\\ArionRime.dll";
        if (!sameModule(L"ArionKeyboard.dll",library) || !sameModule(L"ArionRime.dll",engine)) { failure="different keyboard runtime already loaded"; return false; }
        module=LoadLibraryExW(library.c_str(),nullptr,LOAD_LIBRARY_SEARCH_DLL_LOAD_DIR|LOAD_LIBRARY_SEARCH_SYSTEM32);
        if (!module || !symbol(start,"arion_keyboard_start") || !symbol(enter,"arion_keyboard_enter") || !symbol(leave,"arion_keyboard_leave") || !symbol(stop,"arion_keyboard_stop") || !symbol(error,"arion_keyboard_error") || !symbol(reset,"arion_keyboard_reset_learning") || !symbol(currentGeneration,"arion_keyboard_generation")) { failure="keyboard runtime entry points"; close(); return false; }
        if (!symbol(orderCandidates,"arion_keyboard_order") || !symbol(learnCommit,"arion_keyboard_learn") || !symbol(commitContext,"arion_keyboard_context")) { failure="keyboard ranking entry points"; close(); return false; }
        if (!symbol(candidateMetadata,"arion_keyboard_candidate_metadata") || !symbol(refreshEmoji,"arion_keyboard_refresh_emoji")) { failure="keyboard Emoji entry points"; close(); return false; }
        if (!symbol(restorePersonal,"arion_keyboard_restore_personal") || !symbol(publishPersonal,"arion_keyboard_publish_personal")) { failure="keyboard personal vocabulary entry point"; close(); return false; }
        if (!prepareKeyboardDirectory(userDirectory)) { failure="keyboard profile unverified"; close(); return false; }
        char message[256]{}; auto data=utf8(directory+L"\\data"),user=utf8(userDirectory);
        runtime=start(data.c_str(),user.c_str(),message,sizeof(message));
        if (!runtime) { failure=message[0]?message:"keyboard runtime unavailable"; close(); return false; }
        failure.clear(); return true;
    }
    void close() {
        if (runtime && stop) stop();
        runtime=nullptr;
        if (module) FreeLibrary(module);
        module=nullptr; start=nullptr; enter=nullptr; leave=nullptr; stop=nullptr; error=nullptr; reset=nullptr; currentGeneration=nullptr; orderCandidates=nullptr; learnCommit=nullptr; commitContext=nullptr;
        candidateMetadata=nullptr; refreshEmoji=nullptr;
        restorePersonal=nullptr; publishPersonal=nullptr;
    }
    RimeApi *api() const { return runtime; }
    bool candidate(uintptr_t session,size_t index,ArionCandidateMetadata &metadata) { return candidateMetadata && candidateMetadata(session,index,&metadata); }
    bool refreshEmojiCandidates(uintptr_t session) { return refreshEmoji && refreshEmoji(session); }
    bool restorePersonalVocabulary(uintptr_t session,uint64_t &revision) { return restorePersonal && restorePersonal(session,&revision); }
    bool publishPersonalVocabulary(uintptr_t session,uint64_t revision,const ArionPersonalTerm *terms,size_t count) { return publishPersonal && publishPersonal(session,revision,terms,count); }
    uint64_t generation() const { return currentGeneration?currentGeneration():0; }
    bool order(const char *const *candidates,size_t count,const unsigned char *eligible,const std::string &previous,size_t *indices,std::string &issue) {
        char text[256]{}; bool ok=orderCandidates && orderCandidates(candidates,count,eligible,previous.c_str(),indices,text,sizeof(text)); issue=text; return ok;
    }
    bool learn(const std::string &commit,const std::string &previous,std::string &issue) {
        char text[256]{}; bool ok=learnCommit && learnCommit(commit.c_str(),previous.c_str(),text,sizeof(text)); issue=text; return ok;
    }
    std::string contextOf(const std::string &commit) {
        char text[4096]{}; if (!commitContext || !commitContext(commit.c_str(),text,sizeof(text))) return {}; return text;
    }
    bool ready() {
        if (!runtime) return false;
        char message[256]{}; if (error(message,sizeof(message))) { failure=message; return false; } return true;
    }
    void lock() { if (enter) enter(); }
    void unlock() { if (leave) leave(); }
    bool resetLearning() { if (!runtime) return false; char message[256]{}; bool ok=reset(message,sizeof(message))!=0; if (!ok) failure=message; return ok; }
};
class KeyboardLease {
    KeyboardClient &client;
public:
    explicit KeyboardLease(KeyboardClient &value):client(value) { client.lock(); }
    ~KeyboardLease() { client.unlock(); }
};
