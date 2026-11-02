#include "english_history.h"

namespace arion::ime {
namespace {
bool validKey(const std::string &key) {
    if (key.size()<2 || key.size()>128) return false;
    for (const auto c:key) if (c<'a' || c>'z') return false;
    return true;
}
std::string fold(std::string word) {
    for (auto &c:word) if (c>='A' && c<='Z') c+='a'-'A';
    return word;
}
}
EnglishHistory::EnglishHistory(std::shared_ptr<LearningProvider> provider,
                             std::shared_ptr<const EnglishLexicon> lexicon)
    : database_(provider,provider?learningPath(provider->directory(),"arion_english_selected.arion-userdb"):"",
                "arion_english_selected"),lexicon_(std::move(lexicon)) {}

bool EnglishHistory::open(LearningError &error) {
    if (!database_.open()) { error=database_.error(); return false; }
    Snapshot initial;
    return snapshot(initial,error);
}
bool EnglishHistory::read(std::map<std::string,std::string> &keys,LearningError &error) {
    const auto records=database_.records();
    if (database_.error().code) { error=database_.error(); return false; }
    for (const auto &[key,value]:records) {
        if (!validKey(key) || (value!="1" && fold(value)!=key)) { error={10,"English selection record"}; return false; }
        keys.emplace(key,value);
    }
    return true;
}
bool EnglishHistory::snapshot(Snapshot &output,LearningError &error) {
    std::lock_guard<std::mutex> lock(mutex_);
    std::map<std::string,std::string> keys;
    if (!read(keys,error)) return false;
    if (!snapshot_ || keys!=indexed_) {
        std::set<std::string> selected;
        std::map<std::string,std::string> spellings;
        for (const auto &[key,value]:keys) {
            selected.insert(key);
            if (value!="1") spellings.emplace(key,value);
        }
        auto next=lexicon_->IndexSelected(selected,spellings);
        indexed_=std::move(keys);
        snapshot_=std::move(next);
    }
    output=snapshot_;
    error={};
    return true;
}
bool EnglishHistory::confirm(const std::set<std::string> &keys,LearningError &error) {
    std::lock_guard<std::mutex> lock(mutex_);
    std::map<std::string,std::string> additions;
    for (const auto &key:keys) {
        if (!validKey(key)) { error={10,"English selection key"}; return false; }
        if (!lexicon_->IsCommon(key)) additions.emplace(key,"1");
    }
    return store(additions,error);
}
bool EnglishHistory::confirmRaw(const std::string &word,LearningError &error) {
    std::lock_guard<std::mutex> lock(mutex_);
    const auto key=fold(word);
    // A language switch confirms only a bounded single English spelling.
    if (!validKey(key) || lexicon_->IsCommon(key)) { error={}; return true; }
    return store({{key,lexicon_->Contains(key)?"1":word}},error);
}
bool EnglishHistory::store(const std::map<std::string,std::string> &keys,LearningError &error) {
    auto additions=keys;
    if (additions.empty()) { error={}; return true; }
    std::map<std::string,std::string> stored;
    if (!read(stored,error)) return false;
    for (const auto &[key,value]:stored) additions.erase(key);
    if (additions.empty()) { error={}; return true; }
    if (!database_.begin()) { error=database_.error(); return false; }
    for (const auto &[key,value]:additions) {
        if (database_.update(key,value)) continue;
        error=database_.error();
        database_.abort();
        return false;
    }
    if (!database_.commit()) {
        error=database_.error();
        database_.abort();
        return false;
    }
    error={};
    return true;
}
}
