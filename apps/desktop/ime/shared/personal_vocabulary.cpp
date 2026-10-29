#include "personal_vocabulary.h"
#include "ranking_unicode.h"
#include <rime/algo/syllabifier.h>
#include <rime/dict/prism.h>
#include <algorithm>
#include <cstring>

namespace arion::ime {
bool decodePersonalTerms(const ArionPersonalTerm* terms,size_t count,std::vector<PersonalTerm>& result) {
    if (count>2000 || (count && !terms)) return false;
    std::vector<PersonalTerm> next;
    for (size_t i=0;i<count;++i) {
        const auto& term=terms[i];
        if (!term.id || !term.text || !term.reading || strnlen(term.id,129)>128 ||
            strnlen(term.text,257)>256 || strnlen(term.reading,449)>448) return false;
        next.push_back({term.id,term.text,term.reading});
    }
    result=std::move(next);return true;
}
bool PersonalVocabulary::replace(const std::vector<PersonalTerm>& terms,rime::Prism& prism) {
    if (terms.size()>2000) return false;
    std::set<std::string> ids;
    std::vector<std::vector<std::set<int>>> codes;
    for (const auto& term:terms) {
        if (term.id.empty() || term.id.size()>128 || !ids.insert(term.id).second ||
            term.text.empty() || term.text.size()>256 || rankingContainsHan(term.text)<0 ||
            term.reading.empty() || term.reading.size()>448) return false;
        if (term.text.find_first_of("\r\n\t")!=std::string::npos || term.text.find('\0')!=std::string::npos) return false;
        std::vector<std::set<int>> code;
        size_t start=0;
        while (start<term.reading.size()) {
            auto end=term.reading.find(' ',start);
            if (end==std::string::npos) end=term.reading.size();
            const auto syllable=term.reading.substr(start,end-start);
            if (syllable.empty() || syllable.size()>6 ||
                !std::all_of(syllable.begin(),syllable.end(),[](char c){return c>='a' && c<='z';})) return false;
            rime::Syllabifier parser(" '",false,true);
            rime::SyllableGraph graph;
            parser.BuildSyllableGraph(syllable,prism,&graph);
            std::set<int> syllables;
            auto first=graph.edges.find(0);
            if (first!=graph.edges.end()) {
                auto last=first->second.find(syllable.size());
                if (last!=first->second.end()) for (const auto& [id,properties]:last->second)
                    if (properties.type==rime::kNormalSpelling && !properties.is_correction) syllables.insert(id);
            }
            if (syllables.empty()) return false;
            code.push_back(std::move(syllables));
            if (code.size()>64 || (end<term.reading.size() && end+1==term.reading.size())) return false;
            start=end+1;
        }
        codes.push_back(std::move(code));
    }
    terms_=terms;
    codes_=std::move(codes);
    return true;
}

std::vector<PersonalMatch> PersonalVocabulary::lookup(const rime::SyllableGraph& graph) const {
    std::vector<PersonalMatch> matches;
    for (size_t term=0;term<codes_.size();++term) {
        std::set<size_t> positions{0};
        for (const auto& syllables:codes_[term]) {
            std::set<size_t> next;
            for (auto start:positions) {
                auto from=graph.edges.find(start);
                if (from==graph.edges.end()) continue;
                for (const auto& [end,spellings]:from->second) {
                    if (end<=start || end>graph.input_length) continue;
                    for (const auto& [id,properties]:spellings) {
                        if (syllables.count(id) && properties.type<=rime::kAbbreviation) {
                            next.insert(end); break;
                        }
                    }
                }
            }
            positions=std::move(next);
            if (positions.empty()) break;
        }
        for (auto end:positions) matches.push_back({term,end});
    }
    std::stable_sort(matches.begin(),matches.end(),[](const auto& a,const auto& b){return a.end>b.end;});
    return matches;
}
}
