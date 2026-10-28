#ifndef ARION_ENGLISH_LEXICON_H
#define ARION_ENGLISH_LEXICON_H
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
    struct Entry { std::string key, display; uint32_t rank; };
    class Cursor {
     public:
        const Entry *Peek() const;
        bool Next();
#ifdef ARION_ENGLISH_TESTING
        size_t NodeVisits() const { return node_visits_; }
#endif
     private:
        friend class EnglishLexicon;
        struct Later {
            const EnglishLexicon *lexicon;
            bool operator()(uint32_t a, uint32_t b) const;
        };
        using Queue = std::priority_queue<uint32_t,std::vector<uint32_t>,Later>;
        explicit Cursor(std::shared_ptr<const EnglishLexicon> lexicon);
        void Add(Queue &queue, size_t begin, size_t end);
        std::shared_ptr<const EnglishLexicon> lexicon_;
        Queue exact_, completion_;
        std::string completion_key_;
        uint32_t current_ = UINT32_MAX;
#ifdef ARION_ENGLISH_TESTING
        size_t node_visits_ = 0;
#endif
    };
    static std::shared_ptr<const EnglishLexicon> Load(const std::filesystem::path &path);
    Cursor Query(std::string_view input) const;
 private:
    std::vector<Entry> entries_;
    std::vector<uint32_t> winner_;
    size_t base_ = 1;
    uint32_t Better(uint32_t a, uint32_t b) const;
};
}
#endif
