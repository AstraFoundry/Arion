#pragma once
#include "learning.h"
#include "personal_vocabulary.h"
namespace arion::ime {
// One whole projection occupies one encrypted database record. Concurrent
// updates cannot merge disjoint words and accidentally resurrect deletions.
class PersonalVocabularyStore {
 public:
    explicit PersonalVocabularyStore(std::shared_ptr<LearningProvider> provider);
    bool open();
    bool load(rime::Prism&,uint64_t& revision,std::vector<PersonalTerm>& terms);
    bool save(rime::Prism&,uint64_t revision,const std::vector<PersonalTerm>& terms);
    const LearningError& error() const { return error_; }
 private:
    bool fail(const char* operation);
    LearningDatabase database_;
    LearningError error_;
};
}
