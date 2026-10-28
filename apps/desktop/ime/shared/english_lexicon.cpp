#include "english_lexicon.h"
#include <algorithm>
#include <charconv>
#include <fstream>
#include <stdexcept>
#include <tuple>

namespace arion {
namespace {
uint32_t number(std::string_view text) {
    uint32_t value=0;
    auto parsed=std::from_chars(text.data(),text.data()+text.size(),value);
    if (text.empty() || parsed.ec!=std::errc() || parsed.ptr!=text.data()+text.size())
        throw std::runtime_error("invalid English lexicon number");
    return value;
}
bool letters(std::string_view text, bool lowercase) {
    if (text.size()<2 || text.size()>128) return false;
    for (char c:text) if (!(c>='a' && c<='z') && (lowercase || !(c>='A' && c<='Z'))) return false;
    return true;
}
}
std::shared_ptr<const EnglishLexicon> EnglishLexicon::Load(const std::filesystem::path &path) {
    if (std::filesystem::file_size(path)>64*1024*1024) throw std::runtime_error("English lexicon exceeds size limit");
    std::ifstream file(path);
    std::string line;
    const std::string header="ARION_ENGLISH\t1\t";
    if (!std::getline(file,line) || line.compare(0,header.size(),header)!=0)
        throw std::runtime_error("invalid English lexicon header");
    const uint32_t count=number(std::string_view(line).substr(header.size()));
    if (!count || count>1000000) throw std::runtime_error("invalid English lexicon count");
    auto lexicon=std::shared_ptr<EnglishLexicon>(new EnglishLexicon);
    lexicon->entries_.reserve(count);
    std::vector<bool> seen(count,false);
    while (std::getline(file,line)) {
        const auto first=line.find('\t'), second=line.find('\t',first==std::string::npos?0:first+1);
        if (first==std::string::npos || second==std::string::npos || lexicon->entries_.size()>=count)
            throw std::runtime_error("invalid English lexicon row");
        Entry entry{line.substr(0,first),line.substr(first+1,second-first-1),number(std::string_view(line).substr(second+1))};
        if (!letters(entry.key,true) || !letters(entry.display,false) || entry.key.size()!=entry.display.size() ||
            entry.rank>=count || seen[entry.rank]) throw std::runtime_error("invalid English lexicon entry");
        for (size_t i=0;i<entry.key.size();++i) {
            const char display=entry.display[i];
            if (entry.key[i]!=(display>='A' && display<='Z'?display+'a'-'A':display))
                throw std::runtime_error("English lexicon spelling mismatch");
        }
        if (!lexicon->entries_.empty()) {
            const auto &previous=lexicon->entries_.back();
            if (std::tie(previous.key,previous.display)>=std::tie(entry.key,entry.display))
                throw std::runtime_error("English lexicon order or duplicate");
        }
        seen[entry.rank]=true;
        lexicon->entries_.push_back(std::move(entry));
    }
    if (!file.eof() || lexicon->entries_.size()!=count) throw std::runtime_error("incomplete English lexicon");
    // Case families are consecutive in rank; a prefix presents their preferred spelling once.
    for (size_t begin=0;begin<count;) {
        size_t end=begin+1;
        uint32_t low=lexicon->entries_[begin].rank,high=low;
        while (end<count && lexicon->entries_[end].key==lexicon->entries_[begin].key) {
            low=std::min(low,lexicon->entries_[end].rank); high=std::max(high,lexicon->entries_[end].rank); ++end;
        }
        if (end-begin>32 || high-low+1!=end-begin) throw std::runtime_error("invalid English case family");
        begin=end;
    }
    while (lexicon->base_<count) lexicon->base_*=2;
    lexicon->winner_.assign(lexicon->base_*2,UINT32_MAX);
    for (uint32_t i=0;i<count;++i) lexicon->winner_[lexicon->base_+i]=i;
    for (size_t i=lexicon->base_-1;i>0;--i)
        lexicon->winner_[i]=lexicon->Better(lexicon->winner_[i*2],lexicon->winner_[i*2+1]);
    return lexicon;
}
uint32_t EnglishLexicon::Better(uint32_t a, uint32_t b) const {
    if (a==UINT32_MAX) return b;
    if (b==UINT32_MAX) return a;
    return entries_[a].rank<entries_[b].rank?a:b;
}
bool EnglishLexicon::Cursor::Later::operator()(uint32_t a, uint32_t b) const {
    return lexicon->entries_[lexicon->winner_[a]].rank>lexicon->entries_[lexicon->winner_[b]].rank;
}
EnglishLexicon::Cursor::Cursor(std::shared_ptr<const EnglishLexicon> lexicon)
    : lexicon_(std::move(lexicon)),exact_(Later{lexicon_.get()}),completion_(Later{lexicon_.get()}) {}
void EnglishLexicon::Cursor::Add(Queue &queue,size_t begin,size_t end) {
    for (begin+=lexicon_->base_,end+=lexicon_->base_;begin<end;begin/=2,end/=2) {
        if (begin%2) queue.push(static_cast<uint32_t>(begin++));
        if (end%2) queue.push(static_cast<uint32_t>(--end));
    }
}
const EnglishLexicon::Entry *EnglishLexicon::Cursor::Peek() const {
    return current_==UINT32_MAX?nullptr:&lexicon_->entries_[current_];
}
bool EnglishLexicon::Cursor::Next() {
    current_=UINT32_MAX;
    const bool completion=exact_.empty();
    Queue &queue=completion?completion_:exact_;
    while (!queue.empty()) {
        const uint32_t node=queue.top(); queue.pop();
#ifdef ARION_ENGLISH_TESTING
        ++node_visits_;
#endif
        if (node>=lexicon_->base_) {
            const auto index=lexicon_->winner_[node];
            const auto &key=lexicon_->entries_[index].key;
            if (completion && key==completion_key_) continue;
            if (completion) completion_key_=key;
            current_=index; return true;
        }
        for (uint32_t child:{node*2,node*2+1})
            if (lexicon_->winner_[child]!=UINT32_MAX) queue.push(child);
    }
    return false;
}
EnglishLexicon::Cursor EnglishLexicon::Query(std::string_view input) const {
    Cursor cursor(shared_from_this());
    if (!letters(input,true)) return cursor;
    auto lower=[this](std::string_view key) {
        return static_cast<size_t>(std::lower_bound(entries_.begin(),entries_.end(),key,
            [](const Entry &entry,std::string_view value){ return entry.key<value; })-entries_.begin());
    };
    const size_t begin=lower(input);
    const size_t exactEnd=static_cast<size_t>(std::upper_bound(entries_.begin(),entries_.end(),input,
        [](std::string_view value,const Entry &entry){ return value<entry.key; })-entries_.begin());
    cursor.Add(cursor.exact_,begin,exactEnd);
    if (input.size()>=3) cursor.Add(cursor.completion_,exactEnd,lower(std::string(input)+'{'));
    cursor.Next();
    return cursor;
}
}
