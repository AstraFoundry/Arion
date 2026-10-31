#ifndef ARION_ENGLISH_LEXICON_H
#define ARION_ENGLISH_LEXICON_H
#include <array>
#include <cstdint>
#include <filesystem>
#include <memory>
#include <queue>
#include <string>
#include <string_view>
#include <vector>

namespace arion {
class EnglishLexicon final : public std::enable_shared_from_this<EnglishLexicon> {
 public:
    static constexpr size_t kMixedMinimumPrefix = 5;
    struct Entry { std::string key, display; uint32_t rank; };
    struct WordPrefix { size_t bytes; std::vector<Entry> forms; };
    enum class Completion { Suggest, Lookup };
    class Cursor {
     public:
        const Entry *Peek() const;
        bool Next();
#ifdef ARION_ENGLISH_TESTING
        size_t NodeVisits() const { return node_visits_; }
#endif
     private:
        friend class EnglishLexicon;
        struct Node { uint32_t bucket, position; };
        struct Later {
            const EnglishLexicon *lexicon;
            bool operator()(Node a, Node b) const;
        };
        explicit Cursor(std::shared_ptr<const EnglishLexicon> lexicon);
        void Add(uint32_t bucket, size_t begin, size_t end);
        std::shared_ptr<const EnglishLexicon> lexicon_;
        std::priority_queue<Node,std::vector<Node>,Later> queue_;
        uint32_t current_ = UINT32_MAX, exact_ = UINT32_MAX;
        size_t variant_ = 0;
#ifdef ARION_ENGLISH_TESTING
        size_t node_visits_ = 0;
#endif
    };
    struct Matches {
        Cursor exact, suggestions, lookup;
        // Native lowercase and enough observed use; not a language probability.
        bool familiar_exact = false;
        bool empty() const { return !exact.Peek() && !suggestions.Peek() && !lookup.Peek(); }
    };
    static std::shared_ptr<const EnglishLexicon> Load(const std::filesystem::path &path);
    Matches Query(std::string_view input, Completion completion = Completion::Suggest) const;
    std::vector<WordPrefix> WordPrefixes(std::string_view input) const;
 private:
    struct Word { std::vector<Entry> displays; uint32_t count; bool lowercase; };
    const Word *ExactWord(std::string_view input) const;
    struct Index { std::vector<uint32_t> words, winners; size_t base = 1; };
    static constexpr uint32_t kSuggestionMinimumCount = 1000;
    std::vector<Word> words_;
    // Each key occurs once: bucket zero is lookup-only, others name minimum prefix length.
    std::array<Index,65> indexes_;
    uint32_t Better(uint32_t a, uint32_t b) const;
};
}
#endif
