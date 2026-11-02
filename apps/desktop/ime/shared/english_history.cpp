#include "english_history.h"

namespace arion::ime {
namespace {
bool validKey(const std::string &key) {
    if (key.size()<2 || key.size()>128) return false;
    for (const auto c:key) if (c<'a' || c>'z') return false;
    return true;
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
bool EnglishHistory::read(std::set<std::string> &keys,LearningError &error) {
    const auto records=database_.records();
    if (database_.error().code) { error=database_.error(); return false; }
    for (const auto &[key,value]:records) {
        if (!validKey(key) || value!="1") { error={10,"English selection record"}; return false; }
        keys.insert(key);
    }
    return true;
}
bool EnglishHistory::snapshot(Snapshot &output,LearningError &error) {
    std::lock_guard<std::mutex> lock(mutex_);
    std::set<std::string> keys;
    if (!read(keys,error)) return false;
    if (!snapshot_ || keys!=indexed_) {
        auto next=lexicon_->IndexSelected(keys);
        indexed_=std::move(keys);
        snapshot_=std::move(next);
    }
    output=snapshot_;
    error={};
    return true;
}
bool EnglishHistory::confirm(const std::set<std::string> &keys,LearningError &error) {
    std::lock_guard<std::mutex> lock(mutex_);
    std::set<std::string> additions,stored;
    for (const auto &key:keys) {
        if (!validKey(key) || !lexicon_->Contains(key)) { error={10,"English selection key"}; return false; }
        if (!lexicon_->IsCommon(key)) additions.insert(key);
    }
    if (additions.empty()) { error={}; return true; }
    if (!read(stored,error)) return false;
    for (const auto &key:stored) additions.erase(key);
    if (additions.empty()) { error={}; return true; }
    if (!database_.begin()) { error=database_.error(); return false; }
    for (const auto &key:additions) {
        if (database_.update(key,"1")) continue;
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
