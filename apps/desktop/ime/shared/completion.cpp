#include "completion.h"
#include "ranking_unicode.h"
#include <cstring>
#include <rime/candidate.h>
#include <rime/context.h>
#include <rime/menu.h>
#include <rime/service.h>
#include <rime/translation.h>

namespace {
// Own the previous menu, never its Segment. Lazy paging keeps every original
// candidate without introducing a segment/menu/translation ownership cycle.
class CompletionTranslation final : public rime::Translation {
 public:
  CompletionTranslation(rime::an<rime::Menu> original, rime::an<rime::Candidate> candidate)
      : original_(original), candidate_(candidate) {}
  rime::an<rime::Candidate> Peek() override { return candidate_; }
  bool Next() override {
    do { candidate_=original_->GetCandidateAt(index_++); }
    while (candidate_ && candidate_->text()==text_);
    set_exhausted(!candidate_);
    return !exhausted();
  }
  void SetText(const std::string& text) { text_=text; }
 private:
  rime::an<rime::Menu> original_;
  rime::an<rime::Candidate> candidate_;
  std::string text_;
  size_t index_=0;
};
}
extern "C" int arion_completion_insert(uintptr_t session,const char *input,const char *text) {
 if (!input || !text || !*input || !*text || strnlen(input,257)>256 || strnlen(text,129)>128 ||
     arion::ime::rankingContainsHan(text)!=1) return 0;
 auto current=rime::Service::instance().GetSession(session);
 if (!current || !current->context()) return 0;
 auto* context=current->context();
 if (context->get_option("ascii_mode") || context->input()!=input ||
     context->caret_pos()!=context->input().size() || context->composition().size()!=1) return 0;
 auto& segment=context->composition().back();
 if (!segment.menu || segment.start!=0 || segment.end!=context->input().size() ||
     segment.status>=rime::Segment::kSelected || !segment.HasTag("abc")) return 0;
 auto first=segment.menu->GetCandidateAt(0);
 if (!first || first->type()=="arion_completion") return 0;
 auto candidate=rime::New<rime::SimpleCandidate>("arion_completion",0,segment.end,text);
 auto translation=rime::New<CompletionTranslation>(segment.menu,candidate);
 translation->SetText(text);
 auto menu=rime::New<rime::Menu>();
 menu->AddTranslation(translation);
 if (menu->Prepare(1)!=1) return 0;
 segment.menu=menu;
 segment.selected_index=0;
 return 1;
}
