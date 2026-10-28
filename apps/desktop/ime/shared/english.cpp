#include "english.h"
#include <rime/candidate.h>
#include <rime/context.h>
#include <rime/engine.h>
#include <rime/filter.h>
#include <rime/registry.h>
#include <rime/translation.h>

namespace {
bool english(const rime::an<rime::Candidate>& candidate) {
    // English is the shared schema's only table translator.
    return candidate && candidate->type() == "table";
}

class EnglishTranslation final : public rime::PrefetchTranslation {
 public:
    explicit EnglishTranslation(rime::an<rime::Translation> source)
        : PrefetchTranslation(source) {}

 protected:
    bool Replenish() override {
        if (translation_->exhausted()) return false;
        if (initial_) {
            initial_ = false;
            rime::CandidateList words;
            // The generated prefix index yields at most three words and one case variant.
            while (!translation_->exhausted() && words.size() < 4 && english(translation_->Peek())) {
                words.push_back(translation_->Peek());
                translation_->Next();
            }
            if (!words.empty()) {
                // Keep up to three actual whole-input Chinese words first.
                // Generated sentences and shorter Pinyin prefixes stay behind English.
                for (size_t i = 0; i < 3 && !translation_->exhausted(); ++i) {
                    auto candidate = translation_->Peek();
                    if (!candidate || candidate->start() != words.front()->start() ||
                        candidate->end() != words.front()->end() ||
                        (candidate->type() != "phrase" && candidate->type() != "user_phrase")) break;
                    cache_.push_back(candidate);
                    translation_->Next();
                }
                for (const auto& word : words) cache_.push_back(word);
                return true;
            }
        }
        auto candidate = translation_->Peek();
        if (!candidate) return false;
        cache_.push_back(candidate);
        translation_->Next();
        return true;
    }

 private:
    bool initial_ = true;
};

class EnglishFilter final : public rime::Filter {
 public:
    explicit EnglishFilter(const rime::Ticket& ticket) : Filter(ticket) {}
    rime::an<rime::Translation> Apply(rime::an<rime::Translation> source,
                                     rime::CandidateList*) override {
        if (engine_->context()->get_option("ascii_mode")) return source;
        return rime::New<EnglishTranslation>(source);
    }
};
}

extern "C" void arion_english_register(void) {
    rime::Registry::instance().Register("arion_english", new rime::Component<EnglishFilter>());
}
