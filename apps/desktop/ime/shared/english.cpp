#include "english.h"
#include "english_lexicon.h"
#include <cstdio>
#include <stdexcept>
#include <rime/algo/syllabifier.h>
#include <rime/candidate.h>
#include <rime/dict/prism.h>
#include <rime/registry.h>
#include <rime/segmentation.h>
#include <rime/translation.h>
#include <rime/translator.h>

namespace {
char Lower(char c) { return c>='A' && c<='Z'?c+('a'-'A'):c; }
char Upper(char c) { return c>='a' && c<='z'?c-('a'-'A'):c; }
std::string Fold(std::string text) { for (auto &c:text) c=Lower(c); return text; }
std::string Display(std::string text,const std::string &raw) {
    bool capitals=false,lowercase=false;
    for (char c:raw) { capitals|=c>='A' && c<='Z'; lowercase|=c>='a' && c<='z'; }
    if (!capitals) return text;
    if (!lowercase) { for (auto &c:text) c=Upper(c); return text; }
    if (raw[0]>='A' && raw[0]<='Z' && raw.substr(1)==Fold(raw.substr(1))) text[0]=Upper(text[0]);
    else if (Fold(text).compare(0,raw.size(),Fold(raw))==0) text.replace(0,raw.size(),raw);
    return text;
}
bool WholeWord(const rime::an<rime::Candidate> &candidate,size_t start,size_t end) {
    return candidate && candidate->start()==start && candidate->end()==end &&
        (candidate->type()=="phrase" || candidate->type()=="user_phrase");
}
class PinyinSpelling final {
 public:
    explicit PinyinSpelling(const std::filesystem::path &data) : prism_(data/"build/pinyin_simp.prism.bin") {
        if (!prism_.Load()) throw std::runtime_error("Pinyin spelling prism unavailable");
    }
    bool Normal(const std::string &input) {
        // Keep ordinary Pinyin, including explicit syllable boundaries, in its original lanes.
        rime::SyllableGraph graph;
        rime::Syllabifier parser(" '",false,true);
        parser.BuildSyllableGraph(input,prism_,&graph);
        auto end=graph.vertices.find(input.size());
        return end!=graph.vertices.end() && end->second==rime::kNormalSpelling;
    }
 private:
    rime::Prism prism_;
};
class BilingualTranslation final : public rime::Translation {
 public:
    BilingualTranslation(rime::an<rime::Translation> chinese,
                         arion::EnglishLexicon::Matches english,bool normal,
                         std::vector<arion::EnglishLexicon::WordPrefix> prefixes,
                         const std::string &input,const rime::Segment &segment)
        : chinese_(std::move(chinese)),english_(std::move(english)),normal_(normal),
          early_exact_(!normal || english_.familiar_exact),
          foreground_lookup_(!normal && (english_.exact.Peek() || english_.suggestions.Peek())),
          prefixes_(std::move(prefixes)),
          input_(input),start_(segment.start),end_(segment.end) { Next(); }
    rime::an<rime::Candidate> Peek() override { return current_; }
    bool Next() override {
        current_.reset();
        if (phase_==Phase::ChineseHead) {
            if (normal_ && head_<3 && WholeWord(Chinese(),start_,end_)) { ++head_; return TakeChinese(); }
            phase_=Phase::Exact;
        }
        if (phase_==Phase::Exact) {
            if (early_exact_ && TakeEnglish(english_.exact)) return true;
            phase_=Phase::Foreground;
        }
        if (phase_==Phase::Foreground) {
            if (TakeEnglish(english_.suggestions)) return true;
            phase_=Phase::Prefixes;
        }
        if (phase_==Phase::Prefixes) {
            // Keep full Chinese proposals before an explicit partial English choice.
            if (!prefixes_.empty() && Chinese() && Chinese()->start()==start_ && Chinese()->end()==end_) return TakeChinese();
            while (prefix_<prefixes_.size()) {
                const auto &prefix=prefixes_[prefix_];
                while (form_<prefix.forms.size()) {
                    const auto &entry=prefix.forms[form_++];
                    if (TakeEntry(entry,start_+prefix.bytes)) return true;
                }
                ++prefix_; form_=0;
            }
            phase_=Phase::Lookup;
        }
        if (foreground_lookup_) {
            const bool preferEnglish=(turn_++%3)==0;
            if ((preferEnglish || !Chinese()) && TakeEnglish(english_.lookup)) return true;
        }
        if (Chinese()) return TakeChinese();
        if (!early_exact_ && TakeEnglish(english_.exact)) return true;
        if (TakeEnglish(english_.lookup)) return true;
        set_exhausted(true);
        return false;
    }
 private:
    rime::an<rime::Candidate> Chinese() const { return chinese_ && !chinese_->exhausted()?chinese_->Peek():nullptr; }
    bool TakeChinese() { current_=chinese_->Peek(); chinese_->Next(); return true; }
    bool TakeEnglish(arion::EnglishLexicon::Cursor &cursor) {
        while (const auto *entry=cursor.Peek()) {
            const auto value=*entry;
            cursor.Next();
            if (TakeEntry(value,end_)) return true;
        }
        return false;
    }
    bool TakeEntry(const arion::EnglishLexicon::Entry &entry,size_t end) {
        const auto raw=input_.substr(0,end-start_);
        auto text=Display(entry.display,raw);
        if (end==last_end_ && text==last_display_) return false;
        last_end_=end; last_display_=text;
        current_=rime::New<rime::SimpleCandidate>("arion_english",start_,end,text,"",raw);
        return true;
    }
    rime::an<rime::Translation> chinese_;
    arion::EnglishLexicon::Matches english_;
    bool normal_,early_exact_,foreground_lookup_;
    std::vector<arion::EnglishLexicon::WordPrefix> prefixes_;
    size_t prefix_=0,form_=0,last_end_=0;
    std::string last_display_;
    std::string input_;
    size_t start_,end_;
    size_t head_=0,turn_=0;
    enum class Phase { ChineseHead, Exact, Foreground, Prefixes, Lookup };
    Phase phase_=Phase::ChineseHead;
    rime::an<rime::Candidate> current_;
};
class BilingualTranslator final : public rime::Translator {
 public:
    BilingualTranslator(const rime::Ticket &ticket,std::shared_ptr<const arion::EnglishLexicon> lexicon,
                        std::shared_ptr<PinyinSpelling> spelling)
        : Translator(ticket),lexicon_(std::move(lexicon)),spelling_(std::move(spelling)) {
        chinese_.reset(rime::Translator::Require("arion_pinyin")->Create(rime::Ticket(ticket.engine,"translator")));
    }
    rime::an<rime::Translation> Query(const std::string &input,const rime::Segment &segment) override {
        auto chinese=chinese_->Query(input,segment);
        if (!segment.HasTag("abc")) return chinese;
        auto head=chinese && !chinese->exhausted()?chinese->Peek():nullptr;
        const bool normal=WholeWord(head,segment.start,segment.end) && spelling_->Normal(input);
        const auto folded=Fold(input);
        auto english=lexicon_->Query(folded,normal?arion::EnglishLexicon::Completion::Lookup:arion::EnglishLexicon::Completion::Suggest);
        std::vector<arion::EnglishLexicon::WordPrefix> prefixes;
        if (english.empty() && !spelling_->Normal(input)) prefixes=lexicon_->WordPrefixes(folded);
        if (english.empty() && prefixes.empty()) return chinese;
        return rime::New<BilingualTranslation>(std::move(chinese),std::move(english),normal,std::move(prefixes),input,segment);
    }
 private:
    std::unique_ptr<rime::Translator> chinese_;
    std::shared_ptr<const arion::EnglishLexicon> lexicon_;
    std::shared_ptr<PinyinSpelling> spelling_;
};
class BilingualComponent final : public rime::Translator::Component {
 public:
    BilingualComponent(std::shared_ptr<const arion::EnglishLexicon> lexicon,std::shared_ptr<PinyinSpelling> spelling)
        : lexicon_(std::move(lexicon)),spelling_(std::move(spelling)) {}
    rime::Translator *Create(const rime::Ticket &ticket) override { return new BilingualTranslator(ticket,lexicon_,spelling_); }
 private:
    std::shared_ptr<const arion::EnglishLexicon> lexicon_;
    std::shared_ptr<PinyinSpelling> spelling_;
};
}
extern "C" int arion_english_register(const char *data_directory,char *error,size_t capacity) {
    try {
        if (!data_directory || !rime::Translator::Require("arion_pinyin"))
            throw std::runtime_error("English translator dependencies unavailable");
        const auto data=std::filesystem::u8path(data_directory);
        auto lexicon=arion::EnglishLexicon::Load(data/"arion_english.lexicon");
        auto spelling=std::make_shared<PinyinSpelling>(data);
        rime::Registry::instance().Register("arion_bilingual",new BilingualComponent(std::move(lexicon),std::move(spelling)));
        if (error && capacity) error[0]=0;
        return 1;
    } catch (const std::exception &failure) {
        if (error && capacity) std::snprintf(error,capacity,"%s",failure.what());
        return 0;
    }
}
