#include "ranking.h"
#include "ranking_bridge.h"
#include "ranking_unicode.h"
#include <algorithm>
#include <cmath>
#include <cstdlib>
#include <cstring>
#include <limits>
#include <numeric>
#include <sstream>
#include <charconv>
namespace arion::ime {
namespace {
constexpr double weight=1.6, halfLife=2000;
constexpr size_t maximumPairs=20000, evictBatch=2000, maximumWordLength=12;
std::pair<uint64_t,uint64_t> valueOf(const std::string &value) {
    uint64_t count=0,tick=0;std::istringstream input(value);if (!(input>>count>>tick)) return {};
    return {count,tick};
}
std::string pairKey(const std::string &context,const std::string &word) {return "c\t"+context+"\t"+word;}
size_t length(const std::string &text) {std::vector<size_t> boundaries;return rankingGraphemes(text,boundaries)?boundaries.size()-1:0;}
}
bool reconcileContextLearning(const LearningRecords &base,const LearningChanges &changes,const LearningRecords &latest,LearningRecords &merged,LearningError &error) {
    LearningClock clock;if (!reconcileLearningClock(base,changes,latest,"@tick",clock,error)) return false;
    LearningChanges plain;
    for (const auto &[key,value]:changes) if (key!="@tick" && key.rfind("c\t",0)!=0) plain[key]=value;
    if (!reconcileLearningChanges(base,plain,latest,merged,error)) return false;
    if (changes.count("@tick")) merged["@tick"]=std::to_string(clock.globalEnd);
    auto parse=[](const std::string &text,uint64_t &count,uint64_t &tick) {
        size_t space=text.find(' ');if (space==std::string::npos) return false;
        auto a=std::from_chars(text.data(),text.data()+space,count),b=std::from_chars(text.data()+space+1,text.data()+text.size(),tick);
        return a.ec==std::errc{} && a.ptr==text.data()+space && b.ec==std::errc{} && b.ptr==text.data()+text.size();
    };
    for (const auto &[key,value]:changes) {
        if (key.rfind("c\t",0)!=0) continue;
        auto old=base.find(key),current=latest.find(key);
        if (!value) {
            // A local ranker's stale eviction cannot erase a peer's newer use.
            if (old!=base.end() && current!=latest.end() && old->second==current->second) merged.erase(key);
            continue;
        }
        uint64_t before=0,unused=0,after=0,position=0,now=0;
        if ((old!=base.end() && !parse(old->second,before,unused)) || !parse(*value,after,position) ||
            (current!=latest.end() && !parse(current->second,now,unused)) || after<before || after-before>std::numeric_limits<uint64_t>::max()-now || position>clock.localEnd) {error={10,"invalid concurrent context change"};return false;}
        uint64_t tick=clock.globalBase+(position>clock.localBase?position-clock.localBase:0);
        merged[key]=std::to_string(now+(after-before))+" "+std::to_string(tick);
    }
    if (merged.size()>maximumPairs+1) {
        std::vector<std::string> pairs;for (const auto &[key,value]:merged) if (key.rfind("c\t",0)==0) pairs.push_back(key);
        std::stable_sort(pairs.begin(),pairs.end(),[&](const auto &a,const auto &b){return valueOf(merged.at(a)).second<valueOf(merged.at(b)).second;});
        for (size_t i=0;i<std::min(evictBatch,pairs.size());++i) merged.erase(pairs[i]);
    }
    return true;
}
ContextRanker::ContextRanker(LearningRecords records):records_(std::move(records)) {
    auto tick=records_.find("@tick");if (tick!=records_.end()) {std::istringstream input(tick->second);input>>tick_;}
}
std::string ContextRanker::contextOf(const std::string &commit) {
    std::string text=rankingTrimmed(commit);std::vector<size_t> boundaries;
    if (text.empty() || !rankingGraphemes(text,boundaries)) return {};
    return text.substr(boundaries[boundaries.size()>2?boundaries.size()-3:0]);
}
double ContextRanker::boost(const std::string &word,const std::string &context) const {
    auto found=records_.find(pairKey(context,word));if (found==records_.end()) return 0;
    auto [count,tick]=valueOf(found->second);if (!count) return 0;
    double age=tick_>tick?static_cast<double>(tick_-tick):0;
    return weight*std::log1p(static_cast<double>(count))*std::pow(0.5,age/halfLife);
}
std::vector<size_t> ContextRanker::order(const std::vector<std::string> &candidates,const std::string &previous) const {
    std::vector<size_t> result(candidates.size());std::iota(result.begin(),result.end(),0);
    std::string context=contextOf(previous);if (context.empty() || candidates.size()<2) return result;
    const size_t full=length(candidates[0]);if (!full) return result;
    std::vector<double> scores;bool boosted=false;
    for (size_t i=0;i<candidates.size();++i) {double value=length(candidates[i])==full?boost(candidates[i],context):0;boosted=boosted||value>0;scores.push_back(value-std::log1p(static_cast<double>(i)));}
    if (boosted) std::stable_sort(result.begin(),result.end(),[&](size_t a,size_t b){return scores[a]>scores[b];});
    return result;
}
LearningChanges ContextRanker::learn(const std::string &commit,const std::string &previous) {
    std::string word=rankingTrimmed(commit),context=contextOf(previous);size_t size=length(word);
    if (context.empty() || !size || size>maximumWordLength || tick_==std::numeric_limits<uint64_t>::max()) return {};
    std::string key=pairKey(context,word);uint64_t count=valueOf(records_[key]).first;
    if (count==std::numeric_limits<uint64_t>::max()) return {};
    ++tick_;LearningChanges changes{{key,std::to_string(count+1)+" "+std::to_string(tick_)},{"@tick",std::to_string(tick_)}};
    for (const auto &[k,value]:changes) records_[k]=*value;
    if (records_.size()>maximumPairs+1) {
        std::vector<std::string> pairs;for (const auto &[k,value]:records_) if (k.rfind("c\t",0)==0) pairs.push_back(k);
        std::stable_sort(pairs.begin(),pairs.end(),[&](const auto &a,const auto &b){return valueOf(records_.at(a)).second<valueOf(records_.at(b)).second;});
        for (size_t i=0;i<std::min(evictBatch,pairs.size());++i) {records_.erase(pairs[i]);changes[pairs[i]]=std::nullopt;}
    }
    return changes;
}
}
struct arion_ranker {arion::ime::ContextRanker value;explicit arion_ranker(arion::ime::LearningRecords records):value(std::move(records)) {}};
extern "C" arion_ranker *arion_ranker_create(const char *const *keys,const char *const *values,size_t count) {
    arion::ime::LearningRecords records;for (size_t i=0;i<count;++i) if (keys[i] && values[i]) records[keys[i]]=values[i];return new arion_ranker(std::move(records));
}
extern "C" void arion_ranker_destroy(arion_ranker *ranker) {delete ranker;}
extern "C" size_t arion_ranker_count(const arion_ranker *ranker) {return ranker?ranker->value.count():0;}
extern "C" int arion_ranker_contains_han(const char *text,size_t length) {
    return text?arion::ime::rankingContainsHan(std::string(text,length)):-1;
}
extern "C" char *arion_ranker_context(const char *commit) {
    auto text=arion::ime::ContextRanker::contextOf(commit?commit:"");if (text.empty()) return nullptr;
    auto result=static_cast<char*>(std::malloc(text.size()+1));if (result) std::memcpy(result,text.c_str(),text.size()+1);return result;
}
extern "C" void arion_ranker_order(const arion_ranker *ranker,const char *const *candidates,size_t count,const char *previous,size_t *indices) {
    if (!indices) return;
    if (!ranker) {std::iota(indices,indices+count,0);return;}
    std::vector<std::string> values;for (size_t i=0;i<count;++i) values.emplace_back(candidates[i]?candidates[i]:"");
    auto order=ranker->value.order(values,previous?previous:"");std::copy(order.begin(),order.end(),indices);
}
extern "C" void arion_ranker_learn(arion_ranker *ranker,const char *commit,const char *previous,arion_ranker_change callback,void *context) {
    if (!ranker) return;
    auto changes=ranker->value.learn(commit?commit:"",previous?previous:"");
    if (callback) for (const auto &[key,value]:changes) callback(key.c_str(),value?value->c_str():nullptr,context);
}
