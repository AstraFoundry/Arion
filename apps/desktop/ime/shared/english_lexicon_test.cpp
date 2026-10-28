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
            rows.push_back(key+'\t'+words[i]+'\t'+std::to_string(words.size()-i-1));
        }
        std::sort(rows.begin(),rows.end());
        std::string text="ARION_ENGLISH\t1\t"+std::to_string(rows.size())+'\n';
        for (const auto &row:rows) text+=row+'\n';
        write(file,text);
        auto lexicon=arion::EnglishLexicon::Load(file);
        auto first=lexicon->Query("prefix"),second=lexicon->Query("prefix");
        require(first.Peek() && second.Peek(),"dense prefix unavailable");
        require(first.NodeVisits()<40,"first candidate scanned the dense prefix");
        auto retained=arion::EnglishLexicon::Load(file)->Query("prefix");
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
        auto exact=lexicon->Query("pre");
        require(exact.Peek() && exact.Peek()->display=="pre","exact match must lead despite low frequency");
        require(exact.Next() && exact.Peek()->display==leading,"completion missing after exact match");
        require(!lexicon->Query("pr").Peek(),"two-letter prefix expanded");
        auto github=lexicon->Query("github");
        require(github.Peek() && github.Peek()->display=="GitHub","source case was lost");
        require(!lexicon->Query("Git").Peek() && !lexicon->Query("pre'").Peek(),"unsupported spelling was normalized silently");
        write(file,"ARION_ENGLISH\t1\t5\ngit\tgit\t4\ngithub\tGitHub\t0\ngithub\tGithub\t1\ngitlab\tGitLab\t2\ngitlab\tgitlab\t3\n");
        auto cases=arion::EnglishLexicon::Load(file);
        auto prefix=cases->Query("git");
        require(prefix.Peek()->display=="git","exact word lost priority over preferred completions");
        require(prefix.Next() && prefix.Peek()->display=="GitHub","preferred completion case changed");
        require(prefix.Next() && prefix.Peek()->display=="GitLab","case variant occupied another completion slot");
        require(!prefix.Next(),"completion repeated a case family");
        auto variants=cases->Query("github");
        require(variants.Peek()->display=="GitHub" && variants.Next() && variants.Peek()->display=="Github" && !variants.Next(),
                "exact input did not expose its complete case family");
        for (const char *invalid:{"ARION_ENGLISH\t2\t1\nab\tab\t0\n", "ARION_ENGLISH\t1\t2\nab\tab\t0\n",
                "ARION_ENGLISH\t1\t2\nab\tab\t0\nac\tac\t0\n", "ARION_ENGLISH\t1\t1\nab\tAC\t0\n",
                "ARION_ENGLISH\t1\t1\nab\tab\t2\n", "ARION_ENGLISH\t1\t2\nac\tac\t0\nab\tab\t1\n"}) {
            write(file,invalid); bool rejected=false;
            try { arion::EnglishLexicon::Load(file); } catch (const std::exception &) { rejected=true; }
            require(rejected,"malformed lexicon was accepted");
        }
        std::cout<<"PASS complete ranked English pagination, exact/case, independent cursors and malformed asset rejection\n";
        return 0;
    } catch (const std::exception &error) { std::cerr<<error.what()<<'\n'; return 1; }
}
