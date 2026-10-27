#include "emoji.h"

#include <rime/candidate.h>
#include <rime/context.h>
#include <rime/engine.h>
#include <rime/filter.h>
#include <rime/menu.h>
#include <rime/registry.h>
#include <rime/service.h>
#include <rime/translation.h>

#include <algorithm>
#include <cstring>
#include <iterator>
#include <string>
#include <unordered_set>

namespace {
ArionEmojiSupported supported = nullptr;
void *support_context = nullptr;
struct Mapping {
    const char *trigger;
    const char *text;
    const char *sequence_id;
};

// Values are Unicode text rendered by the host OS; no Emoji artwork is bundled.
#include "emoji_data.h"

class EmojiCandidate final : public rime::SimpleCandidate {
 public:
  EmojiCandidate(const rime::Candidate& source, const Mapping& mapping)
      : SimpleCandidate("arion_emoji", source.start(), source.end(), mapping.text,
                        "", source.preedit()),
        source_(source.text()), sequence_id_(mapping.sequence_id) {}

  const std::string& source() const { return source_; }
  const std::string& sequence_id() const { return sequence_id_; }

 private:
  std::string source_;
  std::string sequence_id_;
};

class EmojiTranslation final : public rime::PrefetchTranslation {
 public:
  EmojiTranslation(rime::an<rime::Translation> translation, size_t start, size_t end)
      : PrefetchTranslation(translation), start_(start), end_(end) {}

 protected:
  bool Replenish() override {
    if (translation_->exhausted()) return false;
    auto source = translation_->Peek();
    if (!source) return false;
    translation_->Next();
    cache_.push_back(source);
    if (++text_seen_ > 27) return true;
    if (source->start() != start_ || source->end() != end_ ||
        (source->type() != "phrase" && source->type() != "user_phrase")) return true;
    const auto match = std::lower_bound(
        std::begin(mappings), std::end(mappings), source->text(),
        [](const Mapping& mapping, const std::string& trigger) {
          return std::strcmp(mapping.trigger, trigger.c_str()) < 0;
        });
    if (match != std::end(mappings) && source->text() == match->trigger &&
        (!supported || supported(support_context, match->text)) &&
        emitted_.insert(match->text).second) {
      cache_.push_back(rime::New<EmojiCandidate>(*source, *match));
    }
    return true;
  }

 private:
  size_t start_;
  size_t end_;
  std::unordered_set<std::string> emitted_;
  size_t text_seen_ = 0;
};

class EmojiFilter final : public rime::Filter {
 public:
  explicit EmojiFilter(const rime::Ticket& ticket) : Filter(ticket) {}

  rime::an<rime::Translation> Apply(rime::an<rime::Translation> translation,
                                    rime::CandidateList*) override {
    auto* context = engine_->context();
    if (!context || !context->get_option("arion_emoji") ||
        context->get_option("ascii_mode") ||
        context->caret_pos() != context->input().size() ||
        context->composition().empty()) return translation;
    const auto& segment = context->composition().back();
    if (segment.end != context->input().size() ||
        segment.status >= rime::Segment::kSelected ||
        !segment.HasTag("abc")) return translation;
    return rime::New<EmojiTranslation>(translation, segment.start, segment.end);
  }
};

template <size_t N>
bool copy(const std::string& value, char (&out)[N]) {
  if (value.size() >= N) return false;
  std::memcpy(out, value.data(), value.size());
  out[value.size()] = '\0';
  return true;
}
}

extern "C" void arion_emoji_register(void) {
  rime::Registry::instance().Register("arion_emoji", new rime::Component<EmojiFilter>());
}

extern "C" void arion_emoji_set_support(ArionEmojiSupported check, void *context) {
  supported = check;
  support_context = context;
}

extern "C" int arion_emoji_candidate_metadata(uintptr_t session, size_t index,
                                                 ArionEmojiCandidateMetadata *out) {
  if (!out) return 0;
  std::memset(out, 0, sizeof(*out));
  auto current = rime::Service::instance().GetSession(session);
  if (!current || !current->context() || current->context()->composition().empty()) return 0;
  auto& segment = current->context()->composition().back();
  if (!segment.menu) return 0;
  auto candidate = segment.menu->GetCandidateAt(index);
  if (!candidate) return 0;
  out->start = candidate->start();
  out->end = candidate->end();
  if (auto* emoji = dynamic_cast<EmojiCandidate*>(candidate.get())) {
    out->emoji = 1;
    return copy(emoji->source(), out->source) &&
           copy(emoji->sequence_id(), out->sequence_id);
  }
  return 1;
}

extern "C" int arion_emoji_refresh_after_caret_move(uintptr_t session) {
  auto current = rime::Service::instance().GetSession(session);
  if (!current || !current->context()) return 0;
  if (current->context()->get_option("arion_emoji") && current->context()->IsComposing())
    current->context()->RefreshNonConfirmedComposition();
  return 1;
}
