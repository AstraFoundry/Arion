#include "spelling.h"
#include "spelling_options.h"
#include "personal_vocabulary.h"
#include "personal_vocabulary_bridge.h"

#include <rime/algo/syllabifier.h>
#include <rime/candidate.h>
#include <rime/context.h>
#include <rime/dict/corrector.h>
#include <rime/engine.h>
#include <rime/dict/dictionary.h>
#include <rime/gear/script_translator.h>
#include <rime/gear/poet.h>
#include <rime/registry.h>
#include <rime/segmentation.h>
#include <rime/service.h>
#include <rime/ticket.h>
#include <rime/translation.h>
#include <rime/translator.h>
#include <algorithm>
#include <cmath>
#include <iterator>
#include <set>
#include <string_view>
#include <vector>
#include <cstring>
#include <mutex>
#include <tuple>

namespace {
thread_local const rime::Engine* originalQueryEngine=nullptr;

class OriginalQueryScope {
 public:
    explicit OriginalQueryScope(const rime::Engine* engine) : previous_(originalQueryEngine) { originalQueryEngine=engine; }
    ~OriginalQueryScope() { originalQueryEngine=previous_; }
 private:
    const rime::Engine* previous_;
};

bool sameCandidate(const rime::an<rime::Candidate>& a,const rime::an<rime::Candidate>& b) {
    return a && b && a->start()==b->start() && a->end()==b->end() && a->text()==b->text();
}

// Keep the original full-input choice selectable even when the corrected
// graph replaces its sentence. All objects remain genuine engine candidates.
class OriginalRetainingTranslation final : public rime::Translation {
 public:
    OriginalRetainingTranslation(rime::an<rime::Translation> source,rime::an<rime::Candidate> original,bool originalFirst)
        : OriginalRetainingTranslation(std::move(source),std::vector<rime::an<rime::Candidate>>{original},originalFirst) {}
    OriginalRetainingTranslation(rime::an<rime::Translation> source,std::vector<rime::an<rime::Candidate>> originals,bool originalFirst)
        : source_(std::move(source)), originals_(std::move(originals)), show_original_(originalFirst) { prepare(); }
    rime::an<rime::Candidate> Peek() override {
        return exhausted()?nullptr:show_original_?originals_[next_]:source_->Peek();
    }
    bool Next() override {
        if (exhausted()) return false;
        auto current=Peek();
        seen_.insert({current->text(),current->start(),current->end()});
        if (show_original_) ++next_;
        else { source_->Next(); show_original_=true; }
        prepare();
        return !exhausted();
    }
 private:
    using Key=std::tuple<std::string,size_t,size_t>;
    bool seen(const rime::an<rime::Candidate>& candidate) const {
        return candidate && seen_.count({candidate->text(),candidate->start(),candidate->end()});
    }
    void prepare() {
        while (next_<originals_.size() && seen(originals_[next_])) ++next_;
        if (next_==originals_.size()) show_original_=false;
        while (source_ && !source_->exhausted() && seen(source_->Peek())) source_->Next();
        if (!source_ || source_->exhausted()) show_original_=next_<originals_.size();
        set_exhausted(!show_original_ && (!source_ || source_->exhausted()));
    }
    rime::an<rime::Translation> source_;
    std::vector<rime::an<rime::Candidate>> originals_;
    std::set<Key> seen_;
    size_t next_=0;
    bool show_original_=false;
};

// Supplement original-path sentences after full-input correction choices,
// before shorter prefixes. Never discard a candidate to enforce a page cap.
class SentenceSupplementTranslation final : public rime::Translation {
 public:
    SentenceSupplementTranslation(rime::an<rime::Translation> source,
        std::vector<rime::an<rime::Candidate>> sentences,size_t start,size_t end)
        : source_(std::move(source)), sentences_(std::move(sentences)), start_(start), end_(end) { prepare(); }
    rime::an<rime::Candidate> Peek() override {
        if (exhausted()) return nullptr;
        return supplement_?sentences_[next_]:source_->Peek();
    }
    bool Next() override {
        if (exhausted()) return false;
        const auto candidate=Peek();
        if (candidate && candidate->start()==start_ && candidate->end()==end_) seen_.insert(candidate->text());
        if (supplement_) ++next_; else source_->Next();
        prepare();
        return !exhausted();
    }
 private:
    void prepare() {
        while (next_<sentences_.size() && seen_.count(sentences_[next_]->text())) ++next_;
        const auto candidate=source_->exhausted()?nullptr:source_->Peek();
        supplement_=next_<sentences_.size() && (!candidate || candidate->start()!=start_ || candidate->end()!=end_);
        set_exhausted(source_->exhausted() && next_==sentences_.size());
    }
    rime::an<rime::Translation> source_;
    std::vector<rime::an<rime::Candidate>> sentences_;
    std::set<std::string> seen_;
    size_t start_,end_,next_=0;
    bool supplement_=false;
};

class CorrectionSearchScope {
 public:
    CorrectionSearchScope(int& sentences,int& homophones)
        : sentences_(sentences), homophones_(homophones), old_sentences_(sentences), old_homophones_(homophones) {
        sentences_=1; homophones_=1;
    }
    ~CorrectionSearchScope() { sentences_=old_sentences_; homophones_=old_homophones_; }
 private:
    int& sentences_;
    int& homophones_;
    int old_sentences_,old_homophones_;
};

// One native translator, dictionary and learning owner. Query-time bounds
// prevent generated corrections exhausting Rime's shared correction quota.
class SpellingTranslator;
std::map<rime::Context*,SpellingTranslator*> personalTranslators;
std::mutex personalMutex;
class SpellingTranslator final : public rime::ScriptTranslator {
 public:
    explicit SpellingTranslator(const rime::Ticket& ticket) : ScriptTranslator(ticket) { std::lock_guard<std::mutex> lock(personalMutex); personalTranslators[engine_->context()]=this; }
    ~SpellingTranslator() override { std::lock_guard<std::mutex> lock(personalMutex); personalTranslators.erase(engine_->context()); }
    rime::Prism* personalSession(uint64_t& revision) {
        if (!engine_->context()->input().empty() || !dict_ || !dict_->loaded()) return nullptr;
        revision=personalRevision_;return dict_->prism().get();
    }
    bool replacePersonal(uint64_t revision,const std::vector<arion::ime::PersonalTerm>& terms) {
        if (!revision || revision<=personalRevision_ || !engine_->context()->input().empty() || !dict_ || !dict_->loaded()) return false;
        if (!personal_.replace(terms,*dict_->prism())) return false;
        personalRevision_=revision;
        return true;
    }
    rime::an<rime::Translation> Query(const std::string& input,const rime::Segment& segment) override {
        auto result=queryPinyin(input,segment);
        if (!personal_.size() || !dict_ || !dict_->loaded() || !segment.HasTag("abc")) return result;
        rime::Syllabifier parser(" '",false,false);
        rime::SyllableGraph graph;
        parser.BuildSyllableGraph(input,*dict_->prism(),&graph);
        auto matches=personal_.lookup(graph);
        // Keep the ordinary first choice and insert explicit personal matches
        // immediately after it. The engine still owns partial selection/commit.
        std::vector<rime::an<rime::Candidate>> candidates;
        for (const auto& match:matches) {
            const auto& term=personal_.term(match.term);
            candidates.push_back(rime::New<rime::SimpleCandidate>("arion_personal",segment.start,segment.start+match.end,term.text,"",input.substr(0,match.end)));
        }
        if (!candidates.empty()) result=rime::New<OriginalRetainingTranslation>(result,std::move(candidates),false);
        return result;
    }
 private:
    arion::ime::PersonalVocabulary personal_;
    uint64_t personalRevision_=0;
    rime::an<rime::Translation> queryPinyin(const std::string& input,const rime::Segment& segment) {
        auto* context=engine_->context();
        bool alternatives=context->get_option("arion_typo");
        for (const auto* option:arion_fuzzy_options) alternatives|=context->get_option(option);
        if (!alternatives || !dict_ || !dict_->loaded())
            return ScriptTranslator::Query(input,segment);
        rime::an<rime::Candidate> original;
        rime::an<rime::Translation> baseline;
        std::vector<rime::an<rime::Candidate>> sentences;
        {
            OriginalQueryScope scope(engine_);
            baseline=ScriptTranslator::Query(input,segment);
            if (baseline && !baseline->exhausted()) original=baseline->Peek();
            if (baseline) while (!baseline->exhausted()) {
                const auto candidate=baseline->Peek();
                if (!candidate || candidate->type()!="sentence" || candidate->start()!=segment.start || candidate->end()!=segment.start+input.size()) break;
                sentences.push_back(candidate);
                baseline->Next();
            }
        }
        rime::an<rime::Translation> corrected;
        {
            CorrectionSearchScope scope(max_sentences_,max_homophones_);
            corrected=ScriptTranslator::Query(input,segment);
        }
        // An empty alternate search must not remove existing engine matches.
        if (!corrected || corrected->exhausted()) {
            if (!sentences.empty()) return rime::New<SentenceSupplementTranslation>(baseline,std::move(sentences),segment.start,segment.start+input.size());
            return baseline;
        }
        if (original && original->start()==segment.start && original->end()==segment.start+input.size() &&
            !sameCandidate(corrected->Peek(),original)) corrected=rime::New<OriginalRetainingTranslation>(corrected,original,input.size()<=2);
        if (!sentences.empty()) corrected=rime::New<SentenceSupplementTranslation>(corrected,std::move(sentences),segment.start,segment.start+input.size());
        return corrected;
    }
};

struct Pair { const char* a; const char* b; bool initial; };
constexpr Pair pairs[] = {
    {"z", "zh", true},
    {"c", "ch", true},
    {"s", "sh", true},
    {"n", "l", true},
    {"f", "h", true},
    {"r", "l", true},
    {"an", "ang", false},
    {"en", "eng", false},
    {"in", "ing", false},
    {"ian", "iang", false},
    {"uan", "uang", false},
};

// A single keyboard substitution must be physically adjacent on QWERTY.
bool adjacent(char a, char b) {
    constexpr std::string_view rows[] = {"qwertyuiop", "asdfghjkl", "zxcvbnm"};
    constexpr double offsets[] = {0, 0.25, 0.75};
    for (size_t x=0; x<3; ++x) for (size_t y=0; y<3; ++y) {
        auto i=rows[x].find(a), j=rows[y].find(b);
        if (i!=std::string_view::npos && j!=std::string_view::npos)
            return a!=b && std::abs(static_cast<int>(x)-static_cast<int>(y))<=1 &&
                   std::abs((i+offsets[x])-(j+offsets[y]))<=1;
    }
    return false;
}

size_t initialLength(const std::string& text) {
    if (text.size()>=2 && text[1]=='h' && (text[0]=='z' || text[0]=='c' || text[0]=='s')) return 2;
    if (!text.empty() && std::string_view("bpmfdtnlgkhjqxrzcsyw").find(text[0])!=std::string_view::npos) return 1;
    return 0;
}

class SpellingCorrector final : public rime::Corrector {
 public:
    explicit SpellingCorrector(const rime::Ticket& ticket) : engine_(ticket.engine) {}
    void ToleranceSearch(const rime::Prism& prism, const std::string& input,
                         rime::corrector::Corrections* results, size_t tolerance) override {
        if (!engine_ || tolerance==0) return;
        auto* context=engine_->context();
        if (!context || context->get_option("ascii_mode") || originalQueryEngine==engine_) return;
        uint32_t fuzzy=0;
        for (size_t i=0; i<std::size(pairs); ++i)
            if (context->get_option(arion_fuzzy_options[i])) fuzzy|=1u<<i;
        arion::ime::spellingCorrections(prism,input,context->get_option("arion_typo"),fuzzy,results);
    }
 private:
    rime::Engine* engine_;
};
}

namespace arion::ime {
// Only normal-edge reachability is needed to protect exact input. Keep the
// pinned prism's spellings and delimiter semantics, but do not materialize or
// transpose a second complete graph at each correction-search suffix.
bool normalSpelling(const rime::Prism& prism,const std::string& input) {
    if (input.empty()) return false;
    std::vector<unsigned char> reachable(input.size()+1,0);
    reachable[0]=1;
    auto& dictionary=const_cast<rime::Prism&>(prism);
    const auto delimiter=[](char key) { return key==' ' || key=='\''; };
    for (size_t position=0; position<input.size(); ++position) {
        if (!reachable[position]) continue;
        size_t begin=position;
        while (begin<input.size() && delimiter(input[begin])) ++begin;
        std::vector<rime::Prism::Match> matches;
        dictionary.CommonPrefixSearch(input.substr(begin),&matches);
        for (const auto& match:matches) {
            if (!match.length) continue;
            bool normal=false;
            for (auto spelling=dictionary.QuerySpelling(match.value); !spelling.exhausted(); spelling.Next()) {
                const auto properties=spelling.properties();
                if (properties.type==rime::kNormalSpelling && !properties.is_correction) { normal=true; break; }
            }
            if (!normal) continue;
            size_t end=begin+match.length;
            while (end<input.size() && delimiter(input[end])) ++end;
            if (end==input.size()) return true;
            reachable[end]=1;
        }
    }
    return false;
}

void spellingCorrections(const rime::Prism& prism, const std::string& input,
                         bool typos, uint32_t fuzzy, rime::corrector::Corrections* results) {
    if (!results || (!typos && !fuzzy)) return;
    if (typos) {
        // Preserve complete normal input before adding alternative edges.
        if (normalSpelling(prism,input)) typos=false;
        if (!typos && !fuzzy) return;
    }
    // Mandarin full-Pinyin syllables have at most six ASCII letters. One
    // extra keystroke requires examining seven raw bytes, never the full buffer.
    const size_t limit=std::min<size_t>(7,input.find_first_not_of("abcdefghijklmnopqrstuvwxyz"));
    const size_t available=std::min(limit,input.size());
    std::set<int> exact;
    for (size_t length=1; length<=available; ++length) {
        int spelling;
        if (prism.GetValue(input.substr(0,length),&spelling)) exact.insert(spelling);
    }
    auto offer=[&](const std::string& value,size_t length,bool explicitFuzzy=false) {
        if (value.size()<2 || value.size()>6) return;
        int spelling;
        if (!prism.GetValue(value,&spelling) || (!explicitFuzzy && exact.count(spelling))) return;
        // Typo edges must not consume suffix keys for an existing exact ID.
        // The patched syllabifier separately tracks each match's raw length.
        // An explicitly enabled fuzzy final may consume more input than its
        // exact prefix (bang -> ban). Keep both raw lengths for that choice.
        bool normal=false;
        for (auto a=const_cast<rime::Prism&>(prism).QuerySpelling(spelling); !a.exhausted(); a.Next()) {
            const auto properties=a.properties();
            if (properties.type==rime::kNormalSpelling && !properties.is_correction) { normal=true; break; }
        }
        if (!normal) return;
        auto found=results->find(spelling);
        // The correction map deduplicates by spelling; keep the longest raw
        // coverage at equal cost without allocating a set of edit strings.
        if (found==results->end() || found->second.distance>1 ||
            (found->second.distance==1 && found->second.length<length))
            (*results)[spelling]={1,spelling,length};
    };
    for (size_t length=2; length<=available; ++length) {
        const auto raw=input.substr(0,length);
        if (typos) {
            for (size_t i=0; i<raw.size(); ++i) {
                auto deleted=raw; deleted.erase(i,1); offer(deleted,length);
                if (i+1<raw.size() && raw[i]!=raw[i+1]) {
                    auto swapped=raw; std::swap(swapped[i],swapped[i+1]); offer(swapped,length);
                }
                auto replaced=raw;
                for (char c='a'; c<='z'; ++c) if (adjacent(raw[i],c)) {
                    replaced[i]=c; offer(replaced,length);
                }
            }
            if (raw.size()<6) for (size_t i=0; i<=raw.size(); ++i) for (char c='a'; c<='z'; ++c) {
                if (i && raw[i-1]==c) continue; // Same insertion as the preceding position.
                auto inserted=raw; inserted.insert(i,1,c); offer(inserted,length);
            }
        }
        // Fuzzy alternatives operate on the syllable's complete initial/final,
        // not substrings: enabling an/ang must not silently enable ian/iang.
        std::set<std::string> fuzzyVariants{raw};
        for (size_t i=0; i<std::size(pairs); ++i) if (fuzzy & (1u<<i)) {
            const auto prior=fuzzyVariants;
            for (const auto& value : prior) {
                const auto& pair=pairs[i];
                size_t split=initialLength(value);
                auto part=pair.initial?value.substr(0,split):value.substr(split);
                const char* replacement=part==pair.a?pair.b:part==pair.b?pair.a:nullptr;
                if (replacement) fuzzyVariants.insert(pair.initial?
                    std::string(replacement)+value.substr(split):value.substr(0,split)+replacement);
            }
        }
        for (const auto& value : fuzzyVariants) if (value!=raw) offer(value,length,true);
    }
}
}

extern "C" void arion_spelling_register(void) {
    rime::Registry::instance().Register("corrector",new rime::Component<SpellingCorrector>());
    rime::Registry::instance().Register("arion_pinyin",new rime::Component<SpellingTranslator>());
}

extern "C" int arion_personal_vocabulary_replace(uintptr_t session,uint64_t revision,const ArionPersonalTerm* terms,size_t count) {
    if (!revision || count>2000 || (count && !terms)) return 0;
    auto current=rime::Service::instance().GetSession(session);
    if (!current || !current->context()) return 0;
    std::vector<arion::ime::PersonalTerm> entries;
    if (!arion::ime::decodePersonalTerms(terms,count,entries)) return 0;
    std::lock_guard<std::mutex> lock(personalMutex);
    auto translator=personalTranslators.find(current->context());
    return translator!=personalTranslators.end() && translator->second->replacePersonal(revision,entries);
}

namespace arion::ime {
rime::Prism* personalVocabularySession(uintptr_t session,uint64_t& revision) {
    auto current=rime::Service::instance().GetSession(session);
    if (!current || !current->context()) return nullptr;
    std::lock_guard<std::mutex> lock(personalMutex);
    auto translator=personalTranslators.find(current->context());
    return translator==personalTranslators.end()?nullptr:translator->second->personalSession(revision);
}
}
