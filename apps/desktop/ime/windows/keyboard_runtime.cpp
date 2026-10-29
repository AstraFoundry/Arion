#define WIN32_LEAN_AND_MEAN
#define ARION_KEYBOARD_BUILD
#include <windows.h>
#include <filesystem>
#include <cstring>
#include <mutex>
#include <string>
#include "keyboard_runtime.h"
#include "../shared/personal_vocabulary_bridge.h"
#include "emoji_text.h"
#include "../shared/pinyin.h"
#include "../shared/emoji.h"
#include "../shared/spelling.h"
#include "../shared/english.h"
#include "../shared/learning_rime.h"
#include "../shared/ranking.h"
#include "../shared/ranking_unicode.h"

namespace {
std::recursive_mutex mutex;
RimeApi *runtime=nullptr;
size_t clients=0;
uint64_t generation=1;
std::string dataDirectory,userDirectory,prebuiltDirectory,learningDirectory;
std::string failure;
std::unique_ptr<arion::ime::LearningDatabase> contextDatabase;
WindowsEmojiFont emojiFont;
int emojiSupported(void*,const char *text) {
    bool supported=false;
    if (FAILED(emojiFont.supportsUTF8(text,supported))) failure="Windows Emoji font shaping failed";
    return supported?1:0;
}
void output(const std::string &message,char *error,size_t capacity) {
    if (!error || !capacity) return;
    size_t count=std::min(message.size(),capacity-1); std::memcpy(error,message.data(),count); error[count]='\0';
}
bool directory(const std::string &path,bool create) {
    std::error_code error;
    auto native=std::filesystem::u8path(path);
    if (create) std::filesystem::create_directories(native,error);
    if (error || !std::filesystem::is_directory(native,error)) return false;
    DWORD flags=GetFileAttributesW(native.wstring().c_str());
    return flags!=INVALID_FILE_ATTRIBUTES && !(flags&FILE_ATTRIBUTE_REPARSE_POINT);
}
RimeTraits traits() {
    RimeTraits value{}; value.data_size=sizeof(value)-sizeof(value.data_size);
    value.shared_data_dir=dataDirectory.c_str(); value.prebuilt_data_dir=prebuiltDirectory.c_str(); value.staging_dir=prebuiltDirectory.c_str(); value.user_data_dir=userDirectory.c_str();
    value.app_name="rime.arion"; value.distribution_name="Arion"; value.distribution_code_name="arion"; value.distribution_version="1"; value.min_log_level=3; value.log_dir="";
    return value;
}
bool contextReady(char *error,size_t capacity) {
    if (!runtime || !failure.empty()) { output(failure.empty()?"keyboard runtime unavailable":failure,error,capacity); return false; }
    if (!contextDatabase) { arion::ime::LearningError issue; contextDatabase=arion::ime::openContextLearning(issue); }
    if (!contextDatabase || contextDatabase->disabled()) { output("encrypted context learning unavailable",error,capacity); return false; }
    return true;
}
arion::ime::LearningRecords rankingRecords() {
    auto records=contextDatabase->records();
    for (auto it=records.begin();it!=records.end();) if (!it->first.empty() && it->first[0]=='\x01') it=records.erase(it); else ++it;
    return records;
}
}

extern "C" RimeApi *arion_keyboard_start(const char *data,const char *user,char *error,size_t capacity) {
    std::lock_guard<std::recursive_mutex> lock(mutex);
    if (!data || !*data || !user || !*user) { output("keyboard runtime directories missing",error,capacity); return nullptr; }
    if (runtime) {
        if (!failure.empty()) { output(failure,error,capacity); return nullptr; }
        if (dataDirectory!=data || userDirectory!=user) { output("keyboard runtime directory mismatch",error,capacity); return nullptr; }
        if (arion_learning_error(error,capacity)) return nullptr;
        ++clients; output("",error,capacity); return runtime;
    }
    try {
        dataDirectory=data; userDirectory=user;
        prebuiltDirectory=(std::filesystem::u8path(dataDirectory)/"build").u8string();
        learningDirectory=(std::filesystem::u8path(userDirectory)/"learning").u8string();
        if (!directory(dataDirectory,false) || !directory(prebuiltDirectory,false) || !directory(userDirectory,true)) { failure="keyboard runtime directory validation"; output(failure,error,capacity); return nullptr; }
        RimeApi *api=rime_get_api();
        if (!arion_pinyin_api_supported(api)) { failure="keyboard runtime API unsupported"; output(failure,error,capacity); return nullptr; }
        RimeTraits configuration=traits();
        api->setup(&configuration);
        if (FAILED(emojiFont.initialize())) { failure="Windows Emoji font unavailable"; output(failure,error,capacity); return nullptr; }
        arion_emoji_set_support(emojiSupported,nullptr);
        arion_emoji_register();
        if (!arion_learning_windows_register(learningDirectory.c_str(),error,capacity)) { failure="encrypted keyboard learning unavailable"; return nullptr; }
        api->initialize(&configuration);
        arion_spelling_register();
        if (!arion_english_register(dataDirectory.c_str(),error,capacity)) {
            api->finalize(); failure="English lexicon initialization failed"; return nullptr;
        }
        if (arion_learning_error(error,capacity)) { api->finalize(); failure="encrypted keyboard learning unavailable"; return nullptr; }
        runtime=api; clients=1; failure.clear(); output("",error,capacity); return runtime;
    } catch (const std::exception &) { failure="keyboard runtime initialization failed"; output(failure,error,capacity); return nullptr; }
}
extern "C" void arion_keyboard_enter(void) { mutex.lock(); }
extern "C" void arion_keyboard_leave(void) { mutex.unlock(); }
extern "C" int arion_keyboard_candidate_metadata(uintptr_t session,size_t index,ArionCandidateMetadata *metadata) {
    std::lock_guard<std::recursive_mutex> lock(mutex);
    return runtime && failure.empty() && arion_candidate_metadata(session,index,metadata);
}
extern "C" int arion_keyboard_refresh_emoji(uintptr_t session) {
    std::lock_guard<std::recursive_mutex> lock(mutex);
    return runtime && failure.empty() && arion_emoji_refresh_after_caret_move(session);
}
extern "C" int arion_keyboard_error(char *error,size_t capacity) {
    std::lock_guard<std::recursive_mutex> lock(mutex);
    if (!runtime || !failure.empty()) { output(failure.empty()?"keyboard runtime unavailable":failure,error,capacity); return 1; }
    return arion_learning_error(error,capacity);
}
extern "C" int arion_keyboard_reset_learning(char *error,size_t capacity) {
    std::lock_guard<std::recursive_mutex> lock(mutex);
    if (!runtime) { output("keyboard runtime unavailable",error,capacity); return 0; }
    runtime->cleanup_all_sessions(); runtime->finalize(); contextDatabase.reset();
    ++generation;
    auto provider=arion::ime::registeredLearningProvider();
    arion::ime::LearningError issue;
    const bool reset=arion::ime::resetLearningVault(provider,issue);
    if (!reset) {
        // Code 8 is refusal of the initial vault lock, before any deletion.
        // Another editor may still own a lease. Reopen only an authenticated,
        // intact vault; partial deletion or prior persistence errors stay fatal.
        arion::ime::LearningError validation;
        if (issue.code!=8 || arion::ime::encryptedLearningError().code ||
            !arion::ime::validateLearningVault(provider,validation)) {
            failure="encrypted keyboard learning reset failed";
            output(failure,error,capacity); return 0;
        }
    } else arion::ime::clearEncryptedLearningError();
    if (!arion_learning_windows_register(learningDirectory.c_str(),error,capacity)) {
        failure="encrypted keyboard learning unavailable"; return 0;
    }
    arion_emoji_register();
    RimeTraits configuration=traits(); runtime->initialize(&configuration);
    arion_spelling_register();
    if (!arion_english_register(dataDirectory.c_str(),error,capacity)) {
        failure="English lexicon initialization failed"; return 0;
    }
    if (arion_learning_error(error,capacity)) { failure="encrypted keyboard learning unavailable"; return 0; }
    failure.clear();
    output(reset?"":"encrypted keyboard learning reset refused: vault in use",error,capacity);
    return reset?1:0;
}
extern "C" uint64_t arion_keyboard_generation(void) {
    std::lock_guard<std::recursive_mutex> lock(mutex);
    return generation;
}
extern "C" int arion_keyboard_order(const char *const *candidates,size_t count,const unsigned char *eligible,const char *previous,size_t *indices,char *error,size_t capacity) {
    std::lock_guard<std::recursive_mutex> lock(mutex);
    if (!candidates || !indices || count>8192 || !contextReady(error,capacity)) return 0;
    try {
        std::vector<std::string> values; values.reserve(count);
        for (size_t i=0;i<count;++i) { if (!candidates[i]) return 0; values.emplace_back(candidates[i]); }
        arion::ime::ContextRanker ranker(rankingRecords());
        if (contextDatabase->disabled()) { output("encrypted context learning unavailable",error,capacity); return 0; }
        auto order=ranker.order(values,eligible,previous?previous:"");
        if (order.size()!=count) return 0;
        std::copy(order.begin(),order.end(),indices); output("",error,capacity); return 1;
    } catch (const std::exception &) { output("context candidate ranking failed",error,capacity); return 0; }
}
extern "C" int arion_keyboard_learn(const char *commit,const char *previous,char *error,size_t capacity) {
    std::lock_guard<std::recursive_mutex> lock(mutex);
    if (!commit || !contextReady(error,capacity)) return 0;
    try {
        const int han=arion::ime::rankingContainsHan(commit);
        if (han<0) { output("context learning invalid UTF-8",error,capacity); return 0; }
        if (!han) { output("",error,capacity); return 1; }
        arion::ime::ContextRanker ranker(rankingRecords());
        if (contextDatabase->disabled()) { output("encrypted context learning unavailable",error,capacity); return 0; }
        auto changes=ranker.learn(commit,previous?previous:"");
        if (changes.empty()) { output("",error,capacity); return 1; }
        bool ok=contextDatabase->begin();
        for (const auto &[key,value]:changes) if (ok) ok=value?contextDatabase->update(key,*value):contextDatabase->erase(key);
        if (ok) ok=contextDatabase->commit(); else contextDatabase->abort();
        if (!ok) { output("encrypted context learning write failed",error,capacity); return 0; }
        output("",error,capacity); return 1;
    } catch (const std::exception &) { output("context learning failed",error,capacity); return 0; }
}
extern "C" int arion_keyboard_context(const char *commit,char *context,size_t capacity) {
    if (!commit || !context || !capacity) return 0;
    try { auto value=arion::ime::ContextRanker::contextOf(commit); if (value.size()>=capacity) return 0; output(value,context,capacity); return 1; }
    catch (const std::exception &) { return 0; }
}
extern "C" void arion_keyboard_stop(void) {
    std::lock_guard<std::recursive_mutex> lock(mutex);
    if (!clients || --clients) return;
    runtime->cleanup_all_sessions(); ++generation; runtime->finalize(); contextDatabase.reset(); runtime=nullptr;
}

extern "C" int arion_keyboard_restore_personal(uintptr_t session,uint64_t *revision) {
    std::lock_guard<std::recursive_mutex> lock(mutex);
    if (!runtime || !failure.empty()) return 0;
    return arion_personal_vocabulary_restore(session,revision);
}

extern "C" int arion_keyboard_publish_personal(uintptr_t session,uint64_t revision,const ArionPersonalTerm *terms,size_t count) {
    std::lock_guard<std::recursive_mutex> lock(mutex);
    if (!runtime || !failure.empty()) return 0;
    return arion_personal_vocabulary_publish(session,revision,terms,count);
}

extern "C" int arion_keyboard_contains_han(const char *text,size_t length) { return text ? arion::ime::rankingContainsHan(std::string(text,length)) : -1; }
