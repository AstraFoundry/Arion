#ifndef ARION_ENGLISH_LEXICON_H
#define ARION_ENGLISH_LEXICON_H
#include <array>
#include <cstdint>
#include <filesystem>
#include <memory>
#include <map>
#include <queue>
#include <set>
#include <string>
#include <string_view>
#include <vector>

namespace arion {
class EnglishLexicon final : public std::enable_shared_from_this<EnglishLexicon> {
    struct Index { std::vector<uint32_t> words, winners; size_t base = 1; };
 public:
    static constexpr size_t kMixedMinimumPrefix = 5;
    static constexpr uint32_t kCommonMinimumCount = 1000;
    struct Entry { std::string key, display; uint32_t rank; };
    struct WordPrefix { size_t bytes; std::vector<Entry> forms; };
    enum class Completion { Suggest, Lookup };
    class SelectedIndex {
     private:
        friend class EnglishLexicon;
        explicit SelectedIndex(std::shared_ptr<const EnglishLexicon> lexicon) : lexicon_(std::move(lexicon)) {}
        std::shared_ptr<const EnglishLexicon> lexicon_;
        std::vector<uint32_t> words_;
        std::array<Index,65> indexes_;
        std::shared_ptr<const EnglishLexicon> personal_;
    };
    class Cursor {
     public:
        const Entry *Peek() const;
        bool Next();
#ifdef ARION_ENGLISH_TESTING
        size_t NodeVisits() const { return node_visits_; }
#endif
     private:
        friend class EnglishLexicon;
        struct Node { const Index *index; uint32_t position; };
        struct Later {
            const EnglishLexicon *lexicon;
            bool operator()(Node a, Node b) const;
        };
        Cursor(std::shared_ptr<const EnglishLexicon> lexicon,std::shared_ptr<const SelectedIndex> selected);
        void Add(const Index &index,size_t begin,size_t end);
        std::shared_ptr<const EnglishLexicon> lexicon_;
        std::shared_ptr<const SelectedIndex> selected_;
        std::priority_queue<Node,std::vector<Node>,Later> queue_;
        uint32_t current_ = UINT32_MAX, exact_ = UINT32_MAX;
        size_t variant_ = 0;
        std::vector<Cursor> personal_;
#ifdef ARION_ENGLISH_TESTING
        size_t node_visits_ = 0;
#endif
    };
    struct Matches {
        Cursor exact, suggestions, lookup;
        // Native lowercase common words or personal selection, not language probabilities.
        bool familiar_exact = false;
        bool preferred_exact = false;
        bool empty() const { return !exact.Peek() && !suggestions.Peek() && !lookup.Peek(); }
    };
    static std::shared_ptr<const EnglishLexicon> Load(const std::filesystem::path &path);
    std::shared_ptr<const SelectedIndex> IndexSelected(const std::set<std::string> &keys,
        const std::map<std::string,std::string> &spellings = {}) const;
    Matches Query(std::string_view input,Completion completion = Completion::Suggest,
                  std::shared_ptr<const SelectedIndex> selected = {}) const;
    std::vector<WordPrefix> WordPrefixes(std::string_view input) const;
    bool Contains(std::string_view input) const;
    bool IsCommon(std::string_view input) const;
 private:
    struct Word { std::vector<Entry> displays; uint32_t count; bool lowercase; };
    const Word *ExactWord(std::string_view input) const;
    void BuildIndexes(std::array<Index,65> &indexes) const;
    std::vector<Word> words_;
    // Only common words enter the immutable prefix indexes. Exact membership is complete.
    std::array<Index,65> indexes_;
    uint32_t Better(uint32_t a,uint32_t b) const;
};
}
#endif
