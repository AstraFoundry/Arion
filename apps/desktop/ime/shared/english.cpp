#include "english.h"
#include "english_lexicon.h"
#include "english_history.h"
#include "learning_rime.h"
#include "menu.h"
#include <cstdio>
#include <stdexcept>
#include <rime/algo/syllabifier.h>
#include <rime/candidate.h>
#include <rime/context.h>
#include <rime/engine.h>
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
                         const std::string &input,const rime::Segment &segment,bool mayLead)
        : chinese_(std::move(chinese)),english_(std::move(english)),normal_(normal),
          early_exact_(!normal || english_.familiar_exact),
          english_leads_(mayLead && !normal && input.size()>=arion::EnglishLexicon::kMixedMinimumPrefix && english_.preferred_exact),
          prefixes_(std::move(prefixes)),
          input_(input),start_(segment.start),end_(segment.end) { Next(); }
    rime::an<rime::Candidate> Peek() override { return current_; }
    bool Next() override {
        current_.reset();
        if (phase_==Phase::ChineseHead) {
            // Complete Pinyin protects full-range sentences as well as dictionary words.
            const size_t protectedHead=normal_?3:1;
            if (prefixes_.empty() && !english_leads_ && head_<protectedHead && Chinese() &&
                (!head_ || (Chinese()->start()==start_ && Chinese()->end()==end_))) {
                ++head_; return TakeChinese();
            }
            phase_=Phase::Foreground;
        }
        if (phase_==Phase::Foreground) {
            phase_=Phase::ChinesePage;
            // One promoted display form, whether exact or completed. Case alternatives
            // retain their original cursor and remain selectable in the deferred tail.
            if (early_exact_ && TakeEnglish(english_.exact)) return true;
            if (TakeEnglish(english_.suggestions)) return true;
        }
        if (phase_==Phase::ChinesePage) {
            if (prefixes_.empty() && emitted_<ARION_MENU_MAX_PAGE_SIZE && Chinese()) return TakeChinese();
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
        // After the largest supported first page, keep both lazy tails reachable.
        // Chinese leads each tail group; unrelated dictionary scores never compete.
        if (Chinese() && tail_chinese_<2) { ++tail_chinese_; return TakeChinese(); }
        tail_chinese_=0;
        if (TakeEnglish(english_.exact) || TakeEnglish(english_.suggestions) || TakeEnglish(english_.lookup)) return true;
        if (Chinese()) return TakeChinese();
        set_exhausted(true);
        return false;
    }
 private:
    rime::an<rime::Candidate> Chinese() const { return chinese_ && !chinese_->exhausted()?chinese_->Peek():nullptr; }
    bool TakeChinese() { current_=chinese_->Peek(); chinese_->Next(); ++emitted_; return true; }
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
        ++emitted_;
        current_=rime::New<rime::SimpleCandidate>("arion_english",start_,end,text,"",raw);
        return true;
    }
    rime::an<rime::Translation> chinese_;
    arion::EnglishLexicon::Matches english_;
    bool normal_,early_exact_,english_leads_;
    std::vector<arion::EnglishLexicon::WordPrefix> prefixes_;
    size_t prefix_=0,form_=0,last_end_=0;
    std::string last_display_;
    std::string input_;
    size_t start_,end_;
    size_t head_=0,emitted_=0,tail_chinese_=0;
    enum class Phase { ChineseHead, Foreground, ChinesePage, Prefixes, Lookup };
    Phase phase_=Phase::ChineseHead;
    rime::an<rime::Candidate> current_;
};
class BilingualTranslator final : public rime::Translator {
 public:
    BilingualTranslator(const rime::Ticket &ticket,std::shared_ptr<const arion::EnglishLexicon> lexicon,
                        std::shared_ptr<PinyinSpelling> spelling,std::shared_ptr<arion::ime::EnglishHistory> history)
        : Translator(ticket),lexicon_(std::move(lexicon)),spelling_(std::move(spelling)),history_(std::move(history)) {
        chinese_.reset(rime::Translator::Require("arion_pinyin")->Create(rime::Ticket(ticket.engine,"translator")));
        auto *context=engine_->context();
        commit_=context->commit_notifier().connect([this](rime::Context *context) { Remember(context); });
        update_=context->update_notifier().connect([this](rime::Context *context) {
            if (context->input().empty()) selected_.reset();
        });
    }
    ~BilingualTranslator() override { commit_.disconnect(); update_.disconnect(); }
    rime::an<rime::Translation> Query(const std::string &input,const rime::Segment &segment) override {
        if (!selected_) {
            arion::ime::LearningError error;
            if (!history_->snapshot(selected_,error)) {
                arion::ime::reportEncryptedLearningError(error);
                return nullptr;
            }
        }
        auto chinese=chinese_->Query(input,segment);
        if (!segment.HasTag("abc")) return chinese;
        const bool normal=spelling_->Normal(input);
        const auto folded=Fold(input);
        auto english=lexicon_->Query(folded,normal?arion::EnglishLexicon::Completion::Lookup:arion::EnglishLexicon::Completion::Suggest,selected_);
        std::vector<arion::EnglishLexicon::WordPrefix> prefixes;
        if (english.empty() && !normal) prefixes=lexicon_->WordPrefixes(folded);
        if (english.empty() && prefixes.empty()) return chinese;
        return rime::New<BilingualTranslation>(std::move(chinese),std::move(english),normal,std::move(prefixes),input,segment,
            segment.end==engine_->context()->input().size() &&
            engine_->context()->caret_pos()==engine_->context()->input().size());
    }
 private:
    void Remember(rime::Context *context) {
        if (context->get_option("dumb")) return;
        std::set<std::string> keys;
        for (const auto &segment:context->composition()) {
            if (segment.status<rime::Segment::kSelected) continue;
            const auto candidate=rime::Candidate::GetGenuineCandidate(segment.GetSelectedCandidate());
            if (candidate && candidate->type()=="arion_english") keys.insert(Fold(candidate->text()));
        }
        if (keys.empty()) return;
        arion::ime::LearningError error;
        if (!history_->confirm(keys,error)) arion::ime::reportEncryptedLearningError(error);
    }
    std::unique_ptr<rime::Translator> chinese_;
    std::shared_ptr<const arion::EnglishLexicon> lexicon_;
    std::shared_ptr<PinyinSpelling> spelling_;
    std::shared_ptr<arion::ime::EnglishHistory> history_;
    arion::ime::EnglishHistory::Snapshot selected_;
    rime::connection commit_,update_;
};
class BilingualComponent final : public rime::Translator::Component {
 public:
    BilingualComponent(std::shared_ptr<const arion::EnglishLexicon> lexicon,std::shared_ptr<PinyinSpelling> spelling,
                       std::shared_ptr<arion::ime::EnglishHistory> history)
        : lexicon_(std::move(lexicon)),spelling_(std::move(spelling)),history_(std::move(history)) {}
    rime::Translator *Create(const rime::Ticket &ticket) override { return new BilingualTranslator(ticket,lexicon_,spelling_,history_); }
 private:
    std::shared_ptr<const arion::EnglishLexicon> lexicon_;
    std::shared_ptr<PinyinSpelling> spelling_;
    std::shared_ptr<arion::ime::EnglishHistory> history_;
};
}
extern "C" int arion_english_register(const char *data_directory,char *error,size_t capacity) {
    try {
        if (!data_directory || !rime::Translator::Require("arion_pinyin"))
            throw std::runtime_error("English translator dependencies unavailable");
        const auto data=std::filesystem::u8path(data_directory);
        auto lexicon=arion::EnglishLexicon::Load(data/"arion_english.lexicon");
        auto spelling=std::make_shared<PinyinSpelling>(data);
        auto history=std::make_shared<arion::ime::EnglishHistory>(arion::ime::registeredLearningProvider(),lexicon);
        arion::ime::LearningError issue;
        if (!history->open(issue)) {
            arion::ime::reportEncryptedLearningError(issue);
            throw std::runtime_error("encrypted English selection history unavailable");
        }
        rime::Registry::instance().Register("arion_bilingual",new BilingualComponent(std::move(lexicon),std::move(spelling),std::move(history)));
        if (error && capacity) error[0]=0;
        return 1;
    } catch (const std::exception &failure) {
        if (error && capacity) std::snprintf(error,capacity,"%s",failure.what());
        return 0;
    }
}
