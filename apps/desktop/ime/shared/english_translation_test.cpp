// Exercise the production lazy translator with controlled Chinese provenance.
#include "english.cpp"
#include <fstream>
#include <iostream>
#include <set>

static void require(bool condition,const char *message) {
    if (!condition) throw std::runtime_error(message);
}
static rime::an<rime::Translation> chinese(const std::string &type,size_t end,size_t count=30) {
    auto result=rime::New<rime::FifoTranslation>();
    for (size_t i=0;i<count;++i)
        result->Append(rime::New<rime::SimpleCandidate>(type,0,end,"Chinese"+std::to_string(i)));
    return result;
}
static std::vector<rime::an<rime::Candidate>> collect(rime::an<rime::Translation> source) {
    std::vector<rime::an<rime::Candidate>> result;
    while (!source->exhausted()) {
        require(result.size()<200,"lazy stream failed to terminate");
        result.push_back(source->Peek()); source->Next();
    }
    return result;
}
int main(int argc,char **argv) {
    if (argc!=2) return 2;
    try {
        const auto file=std::filesystem::u8path(argv[1]);
        std::ofstream(file)<<"ARION_ENGLISH\t2\t6\n"
            "can\t2\t3000\tcan,Can,CAN\n"
            "hel\t5\t0\tHel\n"
            "hello\t0\t4000\thello,Hello,HELLO\n"
            "helloworld\t4\t2000\thelloworld\n"
            "help\t1\t5000\thelp\n"
            "helper\t3\t2000\thelper\n";
        auto lexicon=arion::EnglishLexicon::Load(file);
        for (const auto &type:{"phrase","user_phrase","sentence","arion_personal"}) {
            rime::Segment segment(0,3);
            auto result=collect(rime::New<BilingualTranslation>(chinese(type,3),
                lexicon->Query("can",arion::EnglishLexicon::Completion::Lookup),true,
                std::vector<arion::EnglishLexicon::WordPrefix>{},"can",segment,true));
            require(result.front()->text()=="Chinese0","complete Pinyin lost its first choice by candidate type");
            for (size_t page:{size_t(5),size_t(9)}) {
                size_t english=0;
                for (size_t i=0;i<page;++i) english+=result[i]->type()=="arion_english";
                require(english==1,"exact case family consumed multiple first-page slots");
            }
            require(result[3]->text()=="can","familiar exact English lost its accessible nonleading slot");
            std::set<std::string> forms;
            for (const auto &candidate:result) {
                if (candidate->type()=="arion_english") forms.insert(candidate->text());
                require(candidate->start()==0 && candidate->end()==3,"candidate coverage changed");
            }
            require(forms==std::set<std::string>{"can","Can","CAN"},"deferred exact cases were lost");
        }
        for (const auto &input:{"hel","hell","hello","HELLO"}) {
            const auto length=std::string(input).size();
            rime::Segment segment(0,length);
            auto result=collect(rime::New<BilingualTranslation>(chinese("sentence",length),
                lexicon->Query(Fold(input)),false,std::vector<arion::EnglishLexicon::WordPrefix>{},input,segment,true));
            const bool leads=length>=arion::EnglishLexicon::kMixedMinimumPrefix;
            require(result.front()->text()==(leads?std::string(input):"Chinese0"),"short exact or incomplete English displaced Chinese");
            size_t english=0;
            for (size_t i=0;i<9;++i) english+=result[i]->type()=="arion_english";
            require(english==(std::string(input)=="hell"?0:1),"unqualified completion or case family occupied first page");
            std::set<std::string> forms;
            for (const auto &candidate:result) if(candidate->type()=="arion_english")
                require(forms.insert(candidate->text()).second,"transformed English case repeated across deferred pages");
            require(forms.count(leads?std::string(input):"hello"),"lazy tail lost a matching English word");
        }
        {
            rime::Segment segment(0,5);
            auto result=collect(rime::New<BilingualTranslation>(chinese("sentence",5),
                lexicon->Query("hello"),false,std::vector<arion::EnglishLexicon::WordPrefix>{},"hello",segment,false));
            require(result.front()->text()=="Chinese0" && result[1]->text()=="hello",
                    "caret-prefix query incorrectly authorized English leadership");
        }
        require(!lexicon->Query("hell").suggestions.Peek() && lexicon->Query("hello").suggestions.Peek(),
                "mixed completion did not enforce the five-letter admission threshold");
        std::cout<<"PASS sentence/personal Pinyin protection, exact-only leadership, bounded first pages and exhaustive case tails\n";
    } catch (const std::exception &error) { std::cerr<<error.what()<<'\n'; return 1; }
}
