#include "english_lexicon.h"
#include <algorithm>
#include <charconv>
#include <fstream>
#include <set>
#include <stdexcept>

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
    const std::string header="ARION_ENGLISH\t2\t";
    if (!std::getline(file,line) || line.compare(0,header.size(),header)!=0)
        throw std::runtime_error("invalid English lexicon header");
    const uint32_t count=number(std::string_view(line).substr(header.size()));
    if (!count || count>1000000) throw std::runtime_error("invalid English lexicon count");
    auto lexicon=std::shared_ptr<EnglishLexicon>(new EnglishLexicon);
    lexicon->words_.reserve(count);
    std::vector<bool> seen(count,false);
    while (std::getline(file,line)) {
        const auto first=line.find('\t'), second=line.find('\t',first==std::string::npos?0:first+1);
        const auto third=line.find('\t',second==std::string::npos?0:second+1);
        if (first==std::string::npos || second==std::string::npos || third==std::string::npos || lexicon->words_.size()>=count)
            throw std::runtime_error("invalid English lexicon row");
        const std::string key=line.substr(0,first);
        const uint32_t rank=number(std::string_view(line).substr(first+1,second-first-1));
        Word word{{},number(std::string_view(line).substr(second+1,third-second-1)),false};
        if (!letters(key,true) || rank>=count || seen[rank] ||
            (!lexicon->words_.empty() && lexicon->words_.back().displays[0].key>=key))
            throw std::runtime_error("invalid English lexicon key, order or rank");
        std::set<std::string> displays;
        size_t begin=third+1;
        for (;;) {
            const size_t comma=line.find(',',begin);
            const std::string display=line.substr(begin,comma==std::string::npos?comma:comma-begin);
            if (!letters(display,false) || display.size()!=key.size() || !displays.insert(display).second || word.displays.size()>=32)
                throw std::runtime_error("invalid English lexicon case family");
            for (size_t i=0;i<key.size();++i) {
                const char c=display[i];
                if (key[i]!=(c>='A' && c<='Z'?c+'a'-'A':c)) throw std::runtime_error("English lexicon spelling mismatch");
            }
            word.lowercase=word.lowercase || display==key;
            word.displays.push_back({key,display,rank});
            if (comma==std::string::npos) break;
            begin=comma+1;
        }
        seen[rank]=true;
        const uint32_t id=static_cast<uint32_t>(lexicon->words_.size());
        const size_t minimum=word.count>=kSuggestionMinimumCount?std::max(size_t(3),(key.size()+1)/2):0;
        lexicon->indexes_[minimum].words.push_back(id);
        lexicon->words_.push_back(std::move(word));
    }
    if (!file.eof() || lexicon->words_.size()!=count) throw std::runtime_error("incomplete English lexicon");
    for (auto &index:lexicon->indexes_) {
        if (index.words.empty()) continue;
        while (index.base<index.words.size()) index.base*=2;
        index.winners.assign(index.base*2,UINT32_MAX);
        for (size_t i=0;i<index.words.size();++i) index.winners[index.base+i]=index.words[i];
        for (size_t i=index.base-1;i>0;--i) index.winners[i]=lexicon->Better(index.winners[i*2],index.winners[i*2+1]);
    }
    return lexicon;
}
uint32_t EnglishLexicon::Better(uint32_t a, uint32_t b) const {
    if (a==UINT32_MAX) return b;
    if (b==UINT32_MAX) return a;
    return words_[a].displays[0].rank<words_[b].displays[0].rank?a:b;
}
bool EnglishLexicon::Cursor::Later::operator()(Node a, Node b) const {
    const auto word=[this](Node n) { return lexicon->indexes_[n.bucket].winners[n.position]; };
    return lexicon->words_[word(a)].displays[0].rank>lexicon->words_[word(b)].displays[0].rank;
}
EnglishLexicon::Cursor::Cursor(std::shared_ptr<const EnglishLexicon> lexicon)
    : lexicon_(std::move(lexicon)),queue_(Later{lexicon_.get()}) {}
void EnglishLexicon::Cursor::Add(uint32_t bucket,size_t begin,size_t end) {
    const auto &index=lexicon_->indexes_[bucket];
    for (begin+=index.base,end+=index.base;begin<end;begin/=2,end/=2) {
        if (begin%2) queue_.push({bucket,static_cast<uint32_t>(begin++)});
        if (end%2) queue_.push({bucket,static_cast<uint32_t>(--end)});
    }
}
const EnglishLexicon::Entry *EnglishLexicon::Cursor::Peek() const {
    return current_==UINT32_MAX?nullptr:&lexicon_->words_[current_].displays[variant_];
}
bool EnglishLexicon::Cursor::Next() {
    if (exact_!=UINT32_MAX) {
        if (++variant_<lexicon_->words_[exact_].displays.size()) return true;
        exact_=UINT32_MAX;
    }
    current_=UINT32_MAX; variant_=0;
    while (!queue_.empty()) {
        const auto node=queue_.top(); queue_.pop();
#ifdef ARION_ENGLISH_TESTING
        ++node_visits_;
#endif
        const auto &index=lexicon_->indexes_[node.bucket];
        if (node.position>=index.base) { current_=index.winners[node.position]; return true; }
        for (uint32_t child:{node.position*2,node.position*2+1})
            if (index.winners[child]!=UINT32_MAX) queue_.push({node.bucket,child});
    }
    return false;
}
EnglishLexicon::Matches EnglishLexicon::Query(std::string_view input,Completion completion) const {
    auto owned=shared_from_this();
    Matches result{Cursor(owned),Cursor(owned),Cursor(owned),false};
    if (!letters(input,true)) return result;
    auto exact=std::lower_bound(words_.begin(),words_.end(),input,
        [](const Word &word,std::string_view value){return word.displays[0].key<value;});
    if (exact!=words_.end() && exact->displays[0].key==input) {
        result.exact.current_=result.exact.exact_=static_cast<uint32_t>(exact-words_.begin());
        result.familiar_exact=exact->lowercase && exact->count>=kSuggestionMinimumCount;
    }
    if (input.size()<3) return result;
    const std::string upper=std::string(input)+'{';
    for (uint32_t bucket=0;bucket<indexes_.size();++bucket) {
        const auto &index=indexes_[bucket];
        if (index.words.empty()) continue;
        const auto first=std::upper_bound(index.words.begin(),index.words.end(),input,
            [this](std::string_view key,uint32_t id){return key<words_[id].displays[0].key;});
        const auto last=std::lower_bound(first,index.words.end(),upper,
            [this](uint32_t id,std::string_view key){return words_[id].displays[0].key<key;});
        auto &cursor=completion==Completion::Suggest && bucket && bucket<=input.size()?result.suggestions:result.lookup;
        cursor.Add(bucket,first-index.words.begin(),last-index.words.begin());
    }
    result.suggestions.Next(); result.lookup.Next();
    return result;
}
}
