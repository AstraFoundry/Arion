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
        // English candidates accept letters only; delimiter input stays with the original translator.
        rime::SyllableGraph graph;
        rime::Syllabifier parser("",false,true);
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
                         const std::string &input,const rime::Segment &segment)
        : chinese_(std::move(chinese)),english_(std::move(english)),normal_(normal),
          early_exact_(!normal || english_.familiar_exact),
          foreground_lookup_(!normal && (english_.exact.Peek() || english_.suggestions.Peek())),
          input_(input),start_(segment.start),end_(segment.end) { Next(); }
    rime::an<rime::Candidate> Peek() override { return current_; }
    bool Next() override {
        current_.reset();
        if (phase_==Phase::ChineseHead) {
            if (normal_ && head_<3 && WholeWord(Chinese(),start_,end_)) { ++head_; return TakeChinese(); }
            phase_=Phase::Exact;
        }
        if (phase_==Phase::Exact) {
            if (early_exact_ && english_.exact.Peek()) return TakeEnglish(english_.exact);
            phase_=Phase::Foreground;
        }
        if (phase_==Phase::Foreground) {
            if (english_.suggestions.Peek()) return TakeEnglish(english_.suggestions);
            phase_=Phase::Lookup;
        }
        if (foreground_lookup_) {
            const bool preferEnglish=(turn_++%3)==0;
            if (english_.lookup.Peek() && (preferEnglish || !Chinese())) return TakeEnglish(english_.lookup);
        }
        if (Chinese()) return TakeChinese();
        if (!early_exact_ && english_.exact.Peek()) return TakeEnglish(english_.exact);
        if (english_.lookup.Peek()) return TakeEnglish(english_.lookup);
        set_exhausted(true);
        return false;
    }
 private:
    rime::an<rime::Candidate> Chinese() const { return chinese_ && !chinese_->exhausted()?chinese_->Peek():nullptr; }
    bool TakeChinese() { current_=chinese_->Peek(); chinese_->Next(); return true; }
    bool TakeEnglish(arion::EnglishLexicon::Cursor &cursor) {
        current_=rime::New<rime::SimpleCandidate>("arion_english",start_,end_,cursor.Peek()->display,"",input_);
        cursor.Next();
        return true;
    }
    rime::an<rime::Translation> chinese_;
    arion::EnglishLexicon::Matches english_;
    bool normal_,early_exact_,foreground_lookup_;
    std::string input_;
    size_t start_,end_;
    size_t head_=0,turn_=0;
    enum class Phase { ChineseHead, Exact, Foreground, Lookup };
    Phase phase_=Phase::ChineseHead;
    rime::an<rime::Candidate> current_;
};
class BilingualTranslator final : public rime::Translator {
 public:
    BilingualTranslator(const rime::Ticket &ticket,std::shared_ptr<const arion::EnglishLexicon> lexicon,
                        std::shared_ptr<PinyinSpelling> spelling)
        : Translator(ticket),lexicon_(std::move(lexicon)),spelling_(std::move(spelling)) {
        chinese_.reset(rime::Translator::Require("script_translator")->Create(rime::Ticket(ticket.engine,"translator")));
    }
    rime::an<rime::Translation> Query(const std::string &input,const rime::Segment &segment) override {
        auto chinese=chinese_->Query(input,segment);
        if (!segment.HasTag("abc")) return chinese;
        auto head=chinese && !chinese->exhausted()?chinese->Peek():nullptr;
        const bool normal=WholeWord(head,segment.start,segment.end) && spelling_->Normal(input);
        auto english=lexicon_->Query(input,normal?arion::EnglishLexicon::Completion::Lookup:arion::EnglishLexicon::Completion::Suggest);
        if (english.empty()) return chinese;
        return rime::New<BilingualTranslation>(std::move(chinese),std::move(english),normal,input,segment);
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
        if (!data_directory || !rime::Translator::Require("script_translator"))
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
