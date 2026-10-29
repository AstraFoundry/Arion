#pragma once
#include <cstddef>
#include <cstdint>
#include "personal_vocabulary_bridge.h"
#include <set>
#include <string>
#include <vector>
namespace rime { class Prism; struct SyllableGraph; }
namespace arion::ime {
struct PersonalTerm { std::string id, text, reading; };
struct PersonalMatch { size_t term, end; };
bool decodePersonalTerms(const ArionPersonalTerm*,size_t,std::vector<PersonalTerm>&);
// Internal adapter access; caller must own the serialized session thread.
rime::Prism* personalVocabularySession(uintptr_t session,uint64_t& revision);

// Immutable after construction; this projection never writes ordinary learning.
class PersonalVocabulary {
 public:
    // A rejected update leaves the previous complete projection untouched.
    bool replace(const std::vector<PersonalTerm>& terms, rime::Prism& prism);
    std::vector<PersonalMatch> lookup(const rime::SyllableGraph& graph) const;
    const PersonalTerm& term(size_t index) const { return terms_.at(index); }
    size_t size() const { return terms_.size(); }
 private:
    std::vector<PersonalTerm> terms_;
    std::vector<std::vector<std::set<int>>> codes_;
};
}
