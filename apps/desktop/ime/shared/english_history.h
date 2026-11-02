#pragma once
#include "english_lexicon.h"
#include "learning.h"
#include <set>

namespace arion::ime {
// Confirmed English words grant completion eligibility independently of Chinese learning.
class EnglishHistory final {
public:
    using Snapshot = std::shared_ptr<const EnglishLexicon::SelectedIndex>;
    EnglishHistory(std::shared_ptr<LearningProvider>, std::shared_ptr<const EnglishLexicon>);
    bool open(LearningError &);
    bool snapshot(Snapshot &, LearningError &);
    bool confirm(const std::set<std::string> &keys, LearningError &);
    bool confirmRaw(const std::string &word, LearningError &);
private:
    bool read(std::map<std::string,std::string> &, LearningError &);
    bool store(const std::map<std::string,std::string> &, LearningError &);
    LearningDatabase database_;
    std::shared_ptr<const EnglishLexicon> lexicon_;
    std::map<std::string,std::string> indexed_;
    Snapshot snapshot_;
    std::mutex mutex_;
};
}
