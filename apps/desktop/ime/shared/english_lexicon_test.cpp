#include "english_lexicon.h"
#include <algorithm>
#include <fstream>
#include <iostream>
#include <stdexcept>
#include <set>

static void require(bool condition,const char *message) { if (!condition) throw std::runtime_error(message); }
static void write(const std::filesystem::path &file,const std::string &text) { std::ofstream out(file); out<<text; }
int main(int argc,char **argv) {
    if (argc!=2) return 2;
    try {
        const auto file=std::filesystem::u8path(argv[1]);
        std::vector<std::string> words{"pre","GitHub"};
        for (size_t i=0;i<1024;++i) {
            std::string word="prefix";
            for (size_t digit:{size_t(676),size_t(26),size_t(1)}) word+=static_cast<char>('a'+(i/digit)%26);
            words.push_back(word);
        }
        std::vector<std::string> rows;
        for (size_t i=0;i<words.size();++i) {
            auto key=words[i]; for (auto &c:key) if(c>='A' && c<='Z') c+= 'a'-'A';
            rows.push_back(key+'\t'+std::to_string(words.size()-i-1)+"\t0\t"+words[i]);
        }
        std::sort(rows.begin(),rows.end());
        std::string text="ARION_ENGLISH\t2\t"+std::to_string(rows.size())+'\n';
        for (const auto &row:rows) text+=row+'\n';
        write(file,text);
        auto lexicon=arion::EnglishLexicon::Load(file);
        auto first=lexicon->Query("prefix").lookup,second=lexicon->Query("prefix").lookup;
        require(first.Peek() && second.Peek(),"dense prefix unavailable");
        require(first.NodeVisits()<40,"first candidate scanned the dense prefix");
        auto retained=arion::EnglishLexicon::Load(file)->Query("prefix").lookup;
        require(retained.Peek(),"cursor did not retain its immutable lexicon");
        const std::string leading=second.Peek()->display;
        std::set<std::string> seen;
        uint32_t previous=0;
        while (auto entry=first.Peek()) {
            require(seen.insert(entry->display).second,"duplicate completion");
            require(seen.size()==1 || entry->rank>previous,"completion rank differs across pages");
            previous=entry->rank; first.Next();
        }
        require(seen.size()==1024,"later prefix matches were truncated");
        require(second.Peek()->display==leading,"independent query cursor changed");
        auto exact=lexicon->Query("pre").exact;
        require(exact.Peek() && exact.Peek()->display=="pre","exact match must lead despite low frequency");
        require(!exact.Next() && lexicon->Query("pre").lookup.Peek()->display==leading,"exact and deferred completion are not disjoint");
        require(lexicon->Query("pr").empty(),"two-letter prefix expanded");
        auto github=lexicon->Query("github").exact;
        require(github.Peek() && github.Peek()->display=="GitHub","source case was lost");
        require(lexicon->Query("Git").empty() && lexicon->Query("pre'").empty(),"unsupported spelling was normalized silently");
        require(!lexicon->Query("prefix").suggestions.Peek(),"unobserved words acquired automatic slots");
        std::string observed="ARION_ENGLISH\t2\t"+std::to_string(rows.size())+'\n';
        for(auto row:rows) {
            const auto field=row.rfind("\t0\t");
            require(field!=std::string::npos,"synthetic observation field missing");
            row.replace(field,3,"\t2000\t"); observed+=row+'\n';
        }
        write(file,observed);
        auto dense=arion::EnglishLexicon::Load(file);
        auto blocked=dense->Query("pre"),admitted=dense->Query("prefix");
        require(!blocked.suggestions.Peek() && blocked.suggestions.NodeVisits()==0,"dense overlong prefix scanned rejected words");
        require(admitted.suggestions.NodeVisits()<40 && !admitted.lookup.Peek(),"dense admission index scanned or duplicated lookup");
        size_t admittedCount=0;
        while(admitted.suggestions.Peek()) { ++admittedCount; admitted.suggestions.Next(); }
        require(admittedCount==1024,"admitted dense prefix was truncated");
        write(file,"ARION_ENGLISH\t2\t3\ngit\t2\t2000\tgit,Git\ngithub\t0\t0\tGitHub,Github\ngitlab\t1\t0\tGitLab,gitlab\n");
        auto cases=arion::EnglishLexicon::Load(file);
        auto prefix=cases->Query("git").lookup;
        require(prefix.Peek()->display=="GitHub","preferred completion case changed");
        require(prefix.Next() && prefix.Peek()->display=="GitLab","case variant occupied another completion slot");
        require(!prefix.Next(),"completion repeated a case family");
        auto variants=cases->Query("github").exact;
        require(variants.Peek()->display=="GitHub" && variants.Next() && variants.Peek()->display=="Github" && !variants.Next(),
                "exact input did not expose its complete case family");
        auto segments=cases->WordPrefixes("githubnihao");
        require(segments.size()==2 && segments[0].bytes==6 && segments[0].forms[0].display=="GitHub" &&
                segments[0].forms.size()==2 && segments[1].bytes==3,"exact word prefixes lost consumed range, order or source case");
        require(cases->WordPrefixes("github").size()==1 && cases->WordPrefixes("git").empty(),
                "word-prefix query included the whole input or invented a completion");
        require(cases->WordPrefixes("github"+std::string(256,'n'))[0].bytes==6,
                "long mixed input hid a valid bounded English prefix");
        require(cases->WordPrefixes("GitHubnihao").empty(),"prefix lookup silently folded an unsupported key");
        require(cases->Query("git").familiar_exact && !cases->Query("github").familiar_exact,"native lowercase evidence was lost");
        write(file,"ARION_ENGLISH\t2\t5\nartifice\t2\t0\tartifice,Artifice\nartificial\t0\t4523\tartificial,Artificial\nartificially\t1\t2200\tartificially\nxing\t3\t697\tXING\nxinput\t4\t0\txinput\n");
        auto admission=arion::EnglishLexicon::Load(file);
        auto eligible=admission->Query("artifi");
        require(eligible.suggestions.Peek()->display=="artificial","observed useful completion missing");
        require(eligible.suggestions.Next() && eligible.suggestions.Peek()->display=="artificially" && !eligible.suggestions.Next(),"automatic ranking or membership differs");
        require(eligible.lookup.Peek()->display=="artifice" && !eligible.lookup.Next(),"deferred words were lost or duplicated");
        auto shortPrefix=admission->Query("art");
        require(!shortPrefix.suggestions.Peek() && shortPrefix.suggestions.NodeVisits()==0,"overlong completion required rejected-word scanning");
        auto weak=admission->Query("xin");
        require(!weak.suggestions.Peek() && weak.lookup.Peek(),"rare or unknown completions were promoted or deleted");
        auto pinyin=admission->Query("artifi",arion::EnglishLexicon::Completion::Lookup);
        size_t complete=0;
        while(pinyin.lookup.Peek()) { ++complete; pinyin.lookup.Next(); }
        require(!pinyin.suggestions.Peek() && complete==3,"competing Pinyin lost full lookup coverage");
        for (const char *invalid:{"ARION_ENGLISH\t1\t1\nab\t0\t0\tab\n", "ARION_ENGLISH\t2\t2\nab\t0\t0\tab\n",
                "ARION_ENGLISH\t2\t2\nab\t0\t0\tab\nac\t0\t0\tac\n", "ARION_ENGLISH\t2\t1\nab\t0\t0\tAC\n",
                "ARION_ENGLISH\t2\t1\nab\t2\t0\tab\n", "ARION_ENGLISH\t2\t2\nac\t0\t0\tac\nab\t1\t0\tab\n",
                "ARION_ENGLISH\t2\t1\nab\t0\t4294967296\tab\n", "ARION_ENGLISH\t2\t1\nab\t0\t0\tab,ab\n"}) {
            write(file,invalid); bool rejected=false;
            try { arion::EnglishLexicon::Load(file); } catch (const std::exception &) { rejected=true; }
            require(rejected,"malformed lexicon was accepted");
        }
        std::cout<<"PASS complete ranked English pagination, exact/case, independent cursors and malformed asset rejection\n";
        return 0;
    } catch (const std::exception &error) { std::cerr<<error.what()<<'\n'; return 1; }
}
