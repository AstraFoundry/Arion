#include "ranking.h"
#include "ranking_unicode.h"
#include "ranking_bridge.h"
#include <algorithm>
#include <cstdlib>
#include <fstream>
#include <iostream>
#include <sstream>
using namespace arion::ime;
static void require(bool ok,const char *message) {if (!ok) {std::cerr<<"Ranking test failed: "<<message<<'\n';std::exit(1);}}
static void append(std::string &text,uint32_t cp) {
    if (cp<0x80) text.push_back(static_cast<char>(cp));
    else if (cp<0x800) {text.push_back(static_cast<char>(0xc0|(cp>>6)));text.push_back(static_cast<char>(0x80|(cp&63)));}
    else if (cp<0x10000) {text.push_back(static_cast<char>(0xe0|(cp>>12)));text.push_back(static_cast<char>(0x80|((cp>>6)&63)));text.push_back(static_cast<char>(0x80|(cp&63)));}
    else {text.push_back(static_cast<char>(0xf0|(cp>>18)));text.push_back(static_cast<char>(0x80|((cp>>12)&63)));text.push_back(static_cast<char>(0x80|((cp>>6)&63)));text.push_back(static_cast<char>(0x80|(cp&63)));}
}
int main(int argc,char **argv) {
    require(argc==3,"Unicode grapheme and Scripts corpus paths required");std::ifstream file(argv[1]);require(file.good(),"open pinned conformance data");
    std::string line;size_t cases=0;
    while (std::getline(file,line)) {
        line=line.substr(0,line.find('#'));std::istringstream input(line);std::string token,text;std::vector<size_t> expected;
        while (input>>token) {if (token=="÷") expected.push_back(text.size());else if (token!="×") append(text,static_cast<uint32_t>(std::stoul(token,nullptr,16)));}
        if (expected.empty()) continue;
        std::vector<size_t> actual;require(rankingGraphemes(text,actual),"valid corpus UTF-8");
        if (actual!=expected) {std::cerr<<"Grapheme mismatch case "<<cases<<": "<<line<<'\n';return 1;}++cases;
    }
    require(cases>700,"complete conformance corpus");
    // Independently parse the official Script property, including every
    // unlisted scalar's default Unknown value, rather than duplicating ranges.
    std::ifstream scripts(argv[2]);require(scripts.good(),"open pinned Script property data");
    std::vector<bool> han(0x110000,false);
    size_t hanCount=0,scalarCount=0;
    while (std::getline(scripts,line)) {
        line=line.substr(0,line.find('#'));
        auto delimiter=line.find(';');if (delimiter==std::string::npos) continue;
        std::istringstream property(line.substr(delimiter+1));std::string name;property>>name;
        if (name!="Han") continue;
        auto span=line.substr(0,delimiter);auto split=span.find("..");
        uint32_t first=static_cast<uint32_t>(std::stoul(span,nullptr,16));
        uint32_t last=split==std::string::npos?first:static_cast<uint32_t>(std::stoul(span.substr(split+2),nullptr,16));
        require(first<=last && last<han.size(),"valid source Script range");
        for (uint32_t cp=first;cp<=last;++cp) {require(!han[cp],"source Han ranges do not overlap");han[cp]=true;++hanCount;}
    }
    require(hanCount>100000,"complete Unicode 17 Han source");
    for (uint32_t cp=0;cp<han.size();++cp) {
        if (cp>=0xd800 && cp<=0xdfff) continue;
        std::string text;append(text,cp);
        require(rankingContainsHan(text)==static_cast<int>(han[cp]),"all scalar Han classifications match official Scripts.txt");
        require(arion_ranker_contains_han(text.data(),text.size())==static_cast<int>(han[cp]),"C bridge preserves every scalar including NUL");
        ++scalarCount;
    }
    require(rankingContainsHan("")==0 && rankingContainsHan("Latin 123!")==0 && rankingContainsHan(u8"é🙂かな한글")==0,"empty, Latin and non-Han scripts excluded");
    require(rankingContainsHan(u8"Latin你🙂")==1 && rankingContainsHan(u8"\U00020000")==1 && rankingContainsHan(u8"\U00031350")==1,"mixed and supplementary Han included");
    const std::string embedded=std::string("A\0",2)+u8"你";
    require(arion_ranker_contains_han(embedded.data(),embedded.size())==1 && arion_ranker_contains_han(nullptr,0)==-1,"length-aware C bridge preserves NUL and rejects null input");
    std::vector<size_t> out;for (const auto &bad:{std::string("\xc0\xaf"),std::string("\xed\xa0\x80"),std::string("\xf4\x90\x80\x80"),std::string("\xe2\x82")}) require(!rankingGraphemes(bad,out),"invalid UTF-8 refused");
    for (const auto &bad:{std::string("\xc0\xaf"),std::string("\xed\xa0\x80"),std::string("\xf4\x90\x80\x80"),std::string("\xe2\x82"),std::string("\x80"),std::string("\xe2\x28\xa1")}) {
        for (const auto &text:{bad,std::string(u8"你")+bad,bad+u8"你"}) {
            require(rankingContainsHan(text)==-1,"full UTF-8 rejected before or after Han");
            require(arion_ranker_contains_han(text.data(),text.size())==-1,"C bridge keeps invalid UTF-8 distinct from non-Han");
        }
    }
    require(ContextRanker::contextOf(u8"今天天气很好。 ")==u8"很好","existing trailing punctuation/context policy");
    require(ContextRanker::contextOf(u8"。！？ ").empty(),"punctuation-only no context");
    require(ContextRanker::contextOf(u8"AB👨‍👩‍👧‍👦")==u8"B👨‍👩‍👧‍👦","ZWJ emoji is one context character");
    require(ContextRanker::contextOf(u8"ABC🙂")==u8"C🙂","original supplementary-symbol context keys retained");
    ContextRanker ranker({});std::vector<std::string> candidates{u8"你好",u8"拟好",u8"你",u8"妮好"};
    ContextRanker unrestricted({});require(unrestricted.learn("Latin","ctx").size()==2 && ContextRanker::contextOf("Latin")=="in","general ranker learning/context semantics remain unrestricted");
    require(ranker.order(candidates,u8"很好")==std::vector<size_t>({0,1,2,3}),"cold producer order");
    auto changes=ranker.learn(u8"妮好。",u8"很好");require(changes.size()==2,"pair and tick persisted together");
    require(ranker.order(candidates,u8"很好")==std::vector<size_t>({0,3,1,2}),"one learning confirmation order parity");
    ranker.learn(u8"妮好",u8"很好");ranker.learn(u8"妮好",u8"很好");
    require(ranker.order(candidates,u8"很好")[0]==3,"repeat confirmed full phrase promoted");
    for (int i=0;i<20;++i) ranker.learn(u8"你",u8"很好");
    require(ranker.order(candidates,u8"很好")[0]!=2,"prefix candidate never receives context boost");
    ContextRanker mixed({});
    std::vector<std::string> bilingual{u8"惨",u8"残",u8"餐","can","Can",u8"参",u8"灿",u8"蚕"};
    for (int i=0;i<20;++i) for (const auto &word:{u8"参",u8"灿",u8"蚕"}) mixed.learn(word,u8"很好");
    auto bilingualOrder=mixed.order(bilingual,u8"很好");
    require(bilingualOrder[0]>=5,"Chinese context learning still promotes confirmed words");
    require(bilingualOrder[3]==3 && bilingualOrder[4]==4,"English slots survive multiple Chinese context promotions");
    require(ranker.learn("abcdefghijklmn",u8"很好").empty(),"long sentences excluded");
    require(ranker.learn(u8"妮好","").empty(),"no previous context excluded");
    ContextRanker decayed({{"@tick","100000"},{u8"c\t很好\t妮好","1 1"}});
    require(decayed.order(candidates,u8"很好")==std::vector<size_t>({0,1,2,3}),"old counts decay");
    LearningRecords full{{"@tick","20000"}};for (int i=0;i<20000;++i) full["c\tctx\t"+std::to_string(i)]="1 "+std::to_string(i+1);
    ContextRanker bounded(std::move(full));auto eviction=bounded.learn("new","ctx");
    require(bounded.count()==18002,"oldest 2000 pairs evicted at bound");
    require(eviction.at("c\tctx\t0")==std::nullopt && eviction.at("c\tctx\t1999")==std::nullopt,"eviction persisted atomically with learning");
    auto bridge=arion_ranker_create(nullptr,nullptr,0);const char *values[]={u8"你好",u8"拟好"};size_t indices[2]{};
    arion_ranker_order(bridge,values,2,nullptr,indices);require(indices[0]==0 && indices[1]==1,"C bridge identity");arion_ranker_destroy(bridge);
    std::cout<<"Shared ranking, "<<cases<<" Unicode 17 grapheme cases and "<<scalarCount<<" Script scalar cases ("<<hanCount<<" Han) passed.\n";
}
