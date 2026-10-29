#pragma once
#include "learning.h"
namespace arion::ime {
bool reconcileContextLearning(const LearningRecords &base,const LearningChanges &,const LearningRecords &latest,LearningRecords &merged,LearningError &);
class ContextRanker final {
public:
    explicit ContextRanker(LearningRecords records);
    static std::string contextOf(const std::string &commit);
    std::vector<size_t> order(const std::vector<std::string> &candidates,const unsigned char *eligible,const std::string &previous) const;
    LearningChanges learn(const std::string &commit,const std::string &previous);
    size_t count() const { return records_.size(); }
private:
    double boost(const std::string &word,const std::string &context) const;
    LearningRecords records_;
    uint64_t tick_=0;
};
}
