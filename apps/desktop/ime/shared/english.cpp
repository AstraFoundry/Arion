#include "english.h"
#include "english_lexicon.h"
#include <cstdio>
#include <stdexcept>
#include <rime/candidate.h>
#include <rime/registry.h>
#include <rime/segmentation.h>
#include <rime/translation.h>
#include <rime/translator.h>

namespace {
class BilingualTranslation final : public rime::Translation {
 public:
    BilingualTranslation(rime::an<rime::Translation> chinese,
                         arion::EnglishLexicon::Cursor english,
                         const std::string &input,const rime::Segment &segment)
        : chinese_(std::move(chinese)),english_(std::move(english)),
          input_(input),start_(segment.start),end_(segment.end) { Next(); }
    rime::an<rime::Candidate> Peek() override { return current_; }
    bool Next() override {
        current_.reset();
        if (phase_==Phase::ChineseHead) {
            auto candidate=Chinese();
            if (head_<3 && candidate && candidate->start()==start_ && candidate->end()==end_ &&
                (candidate->type()=="phrase" || candidate->type()=="user_phrase")) {
                ++head_; return TakeChinese();
            }
            phase_=Phase::EnglishPreview;
        }
        if (phase_==Phase::EnglishPreview) {
            if (preview_<4 && english_.Peek()) { ++preview_; return TakeEnglish(); }
            phase_=Phase::Tail;
        }
        // Both tails remain pageable; a broad English prefix cannot bury Chinese alternatives.
        const bool preferChinese=(tail_++%3)!=2;
        if (Chinese() && (preferChinese || !english_.Peek())) return TakeChinese();
        if (english_.Peek()) return TakeEnglish();
        set_exhausted(true);
        return false;
    }
 private:
    rime::an<rime::Candidate> Chinese() const { return chinese_ && !chinese_->exhausted()?chinese_->Peek():nullptr; }
    bool TakeChinese() { current_=chinese_->Peek(); chinese_->Next(); return true; }
    bool TakeEnglish() {
        current_=rime::New<rime::SimpleCandidate>("arion_english",start_,end_,english_.Peek()->display,"",input_);
        current_->set_quality(10);
        english_.Next();
        return true;
    }
    rime::an<rime::Translation> chinese_;
    arion::EnglishLexicon::Cursor english_;
    std::string input_;
    size_t start_,end_,head_=0,preview_=0,tail_=0;
    enum class Phase { ChineseHead, EnglishPreview, Tail };
    Phase phase_=Phase::ChineseHead;
    rime::an<rime::Candidate> current_;
};
class BilingualTranslator final : public rime::Translator {
 public:
    BilingualTranslator(const rime::Ticket &ticket,std::shared_ptr<const arion::EnglishLexicon> lexicon)
        : Translator(ticket),lexicon_(std::move(lexicon)) {
        chinese_.reset(rime::Translator::Require("script_translator")->Create(rime::Ticket(ticket.engine,"translator")));
    }
    rime::an<rime::Translation> Query(const std::string &input,const rime::Segment &segment) override {
        auto chinese=chinese_->Query(input,segment);
        if (!segment.HasTag("abc")) return chinese;
        auto english=lexicon_->Query(input);
        if (!english.Peek()) return chinese;
        return rime::New<BilingualTranslation>(std::move(chinese),std::move(english),input,segment);
    }
 private:
    std::unique_ptr<rime::Translator> chinese_;
    std::shared_ptr<const arion::EnglishLexicon> lexicon_;
};
class BilingualComponent final : public rime::Translator::Component {
 public:
    explicit BilingualComponent(std::shared_ptr<const arion::EnglishLexicon> lexicon) : lexicon_(std::move(lexicon)) {}
    rime::Translator *Create(const rime::Ticket &ticket) override { return new BilingualTranslator(ticket,lexicon_); }
 private:
    std::shared_ptr<const arion::EnglishLexicon> lexicon_;
};
}
extern "C" int arion_english_register(const char *data_directory,char *error,size_t capacity) {
    try {
        if (!data_directory || !rime::Translator::Require("script_translator"))
            throw std::runtime_error("English translator dependencies unavailable");
        auto lexicon=arion::EnglishLexicon::Load(std::filesystem::u8path(data_directory)/"arion_english.lexicon");
        rime::Registry::instance().Register("arion_bilingual",new BilingualComponent(std::move(lexicon)));
        if (error && capacity) error[0]=0;
        return 1;
    } catch (const std::exception &failure) {
        if (error && capacity) std::snprintf(error,capacity,"%s",failure.what());
        return 0;
    }
}
