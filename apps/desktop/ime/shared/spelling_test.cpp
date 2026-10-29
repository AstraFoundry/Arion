#include "spelling.h"
#include "personal_vocabulary.h"
#include <rime/algo/syllabifier.h>
#include <rime/dict/corrector.h>
#include <cstdlib>
#include <iostream>
#include <random>

static void require(bool condition, const char* message) {
    if (!condition) { std::cerr<<message<<'\n'; std::exit(1); }
}
int main(int argc, char** argv) {
    require(argc==3,"isolated output path and real prism required");
    rime::Prism prism(rime::path(std::string(argv[1])+"/spelling.bin"));
    require(prism.Build({"zhong","zong","lan","lang","lian","liang","nan","nang","hao","ni"}),"build synthetic prism");
    auto contains=[&](const std::string& input,const std::string& target,bool typo,uint32_t fuzzy,size_t end) {
        rime::corrector::Corrections result;
        arion::ime::spellingCorrections(prism,input,typo,fuzzy,&result);
        int id; require(prism.GetValue(target,&id),"fixture spelling exists");
        auto found=result.find(id);
        return found!=result.end() && found->second.length==end;
    };
    for (const auto& input : {"zhng","zhoong","zhogn","zhomg"}) {
        require(contains(input,"zhong",true,0,std::string(input).size()),"all four typo operations retain full raw coverage");
        require(!contains(input,"zhong",false,0,std::string(input).size()),"disabled typos do not expand");
    }
    require(contains("zhognhao","zhong",true,0,5),"corrected syllable before suffix retains raw boundary");
    require(contains("zhogn'hao","zhong",true,0,5),"apostrophe bounds spelling edits");
    require(!contains("nihao","ni",true,0,3),"an exact syllable cannot gain a longer unpenalized corrected edge");
    require(!contains("zong","zhong",true,0,4),"valid syllables are not automatically corrected into other valid syllables");
    require(!contains("nihao","nan",true,0,2),"a complete normal-spelling path disables typo expansion");
    require(!contains("z'hogn","zhong",true,0,6),"edits cannot cross apostrophe");
    require(!contains("zong","zhong",false,0,4),"fuzzy disabled");
    require(contains("zong","zhong",false,arion::ime::Z_ZH,4),"explicit initial fuzzy pair");
    require(contains("zhong","zong",false,arion::ime::Z_ZH,5),"fuzzy pair is bidirectional");
    require(contains("lan","lang",false,arion::ime::AN_ANG,3),"explicit final fuzzy pair");
    require(!contains("lian","liang",false,arion::ime::AN_ANG,4),"an pair does not implicitly enable ian pair");
    require(contains("lian","liang",false,arion::ime::IAN_IANG,4),"independent ian pair");
    require(contains("nan","lang",false,arion::ime::N_L|arion::ime::AN_ANG,3),"selected initial and final pairs compose");
    rime::Prism real{rime::path(argv[2])};
    require(real.Load(),"load production prism");
    const struct { const char* first; const char* second; uint32_t flag; } fuzzyPairs[] = {
        {"zong","zhong",arion::ime::Z_ZH}, {"cong","chong",arion::ime::C_CH},
        {"san","shan",arion::ime::S_SH}, {"na","la",arion::ime::N_L},
        {"fan","han",arion::ime::F_H}, {"ran","lan",arion::ime::R_L},
        {"ban","bang",arion::ime::AN_ANG}, {"ben","beng",arion::ime::EN_ENG},
        {"bin","bing",arion::ime::IN_ING}, {"lian","liang",arion::ime::IAN_IANG},
        {"guan","guang",arion::ime::UAN_UANG},
    };
    for (const auto& pair:fuzzyPairs) for (bool reverse:{false,true}) {
        const std::string input=reverse?pair.second:pair.first;
        const std::string target=reverse?pair.first:pair.second;
        int id; require(real.GetValue(target,&id),"fuzzy target exists in production prism");
        for (uint32_t selected=0;selected<=11;++selected) {
            const uint32_t flag=selected?1u<<(selected-1):0;
            rime::corrector::Corrections result;
            arion::ime::spellingCorrections(real,input,false,flag,&result);
            const auto found=result.find(id);
            const bool matches=found!=result.end() && found->second.length==input.size();
            if (matches!=(flag==pair.flag)) {
                std::cerr<<"Fuzzy isolation differs: "<<input<<" -> "<<target<<" option "<<flag<<'\n';
                std::exit(1);
            }
        }
    }
    auto normalParity=[&](const std::string& input) {
        rime::Syllabifier parser(" '",false,false);
        rime::SyllableGraph graph;
        parser.BuildSyllableGraph(input,real,&graph);
        const auto end=graph.vertices.find(input.size());
        const bool expected=end!=graph.vertices.end() && end->second==rime::kNormalSpelling;
        if (arion::ime::normalSpelling(real,input)!=expected) {
            std::cerr<<"Normal-spelling parity differs for: "<<input<<'\n';
            std::exit(1);
        }
    };
    for (const std::string input:{"","'"," ","nihao","ni'hao","'nihao'","ni  hao", "n'h", "nh", "xian", "xi'an", "lv", "lve", "nue", "NiHao", "zhognhao"}) normalParity(input);
    std::mt19937 random(1701);
    const std::string syllables[]={"ni","hao","zhong","guo","xian","xi","an","lv","lve","n","h","xing","cha","gong","fang"};
    for (size_t trial=0; trial<1000; ++trial) {
        std::string input;
        const size_t count=1+random()%8;
        for (size_t i=0; i<count; ++i) {
            if (random()%3==0) input+=random()%2?'\'':' ';
            input+=syllables[random()%std::size(syllables)];
        }
        normalParity(input);
        input.insert(random()%(input.size()+1),1,static_cast<char>('a'+random()%26));
        normalParity(input);
        input.erase(random()%input.size(),1);
        normalParity(input);
        normalParity(input.substr(random()%(input.size()+1)));
    }
    std::cout<<"Normal-spelling protection matches actual Rime graphs for 4,016 deterministic cases.\n";
    arion::ime::PersonalVocabulary vocabulary;
    require(vocabulary.replace({{"personal-1","星槎工坊","xing cha gong fang"}},real),"resolve real personal pronunciation");
    auto lookup=[&](const std::string& input) {
        rime::Syllabifier parser(" '",false,false);
        rime::SyllableGraph graph;
        parser.BuildSyllableGraph(input,real,&graph);
        return vocabulary.lookup(graph);
    };
    for (const std::string input:{"xingchagongfang","xing'cha'gong'fang","xcgf"}) {
        auto matches=lookup(input);
        require(!matches.empty() && matches[0].end==input.size(),"personal exact/apostrophe/abbreviation coverage");
    }
    auto prefix=lookup("xingchagongfangnihao");
    require(!prefix.empty() && prefix[0].end==std::string("xingchagongfang").size(),"personal prefix preserves suffix byte boundary");
    require(lookup("xingchenggongfang").empty(),"different pronunciation cannot invent personal candidate");
    require(!vocabulary.replace({{"bad","星槎","zzzzzz"}},real),"unknown dictionary syllable rejected");
    require(vocabulary.size()==1 && !lookup("xcgf").empty(),"invalid update preserves complete old projection");
    require(!vocabulary.replace({{"bad","星槎","x c"}},real),"abbreviation cannot authorize stored pronunciation");
    require(!vocabulary.replace({{"same","星槎","xing cha"},{"same","星辰","xing chen"}},real),"duplicate IDs rejected atomically");
    require(vocabulary.replace({},real) && lookup("xcgf").empty(),"removal eliminates projection matches");
    std::cout<<"Bounded spelling: four typo operations, raw boundaries, disabled policy and independent fuzzy pairs passed.\n";
    std::cout<<"Personal vocabulary: actual prism resolution, abbreviation, prefix ranges, atomic rejection and deletion passed.\n";
}
