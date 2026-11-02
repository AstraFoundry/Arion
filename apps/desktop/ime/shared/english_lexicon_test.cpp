#include "english_lexicon.h"
#include <algorithm>
#include <fstream>
#include <iostream>
#include <stdexcept>
#include <set>

static void require(bool condition,const char *message) { if (!condition) throw std::runtime_error(message); }
static void write(const std::filesystem::path &file,const std::string &text) { std::ofstream out(file); out<<text; }
static std::vector<std::string> collect(arion::EnglishLexicon::Cursor cursor) {
    std::vector<std::string> result;
    while (const auto *entry=cursor.Peek()) { result.push_back(entry->display); cursor.Next(); }
    return result;
}
int main(int argc,char **argv) {
    if (argc!=2) return 2;
    try {
        using Lexicon=arion::EnglishLexicon;
        const auto file=std::filesystem::u8path(argv[1]);
        std::vector<std::string> words{"pre","GitHub"};
        std::set<std::string> selectedKeys;
        for (size_t i=0;i<1024;++i) {
            std::string word="prefix";
            for (size_t digit:{size_t(676),size_t(26),size_t(1)}) word+=static_cast<char>('a'+(i/digit)%26);
            words.push_back(word); selectedKeys.insert(word);
        }
        std::vector<std::string> rows;
        for (size_t i=0;i<words.size();++i) {
            auto key=words[i]; for (auto &c:key) if(c>='A' && c<='Z') c+='a'-'A';
            rows.push_back(key+'\t'+std::to_string(words.size()-i-1)+"\t0\t"+words[i]);
        }
        std::sort(rows.begin(),rows.end());
        std::string text="ARION_ENGLISH\t2\t"+std::to_string(rows.size())+'\n';
        for (const auto &row:rows) text+=row+'\n';
        write(file,text);
        auto lexicon=Lexicon::Load(file);
        auto cold=lexicon->Query("prefix");
        require(cold.empty() && cold.lookup.NodeVisits()==0 && cold.suggestions.NodeVisits()==0,
                "unselected uncommon prefixes leaked or scanned through deferred lookup");
        auto sparse=lexicon->IndexSelected({words.back()});
        auto sparseMatch=lexicon->Query("prefix",Lexicon::Completion::Lookup,sparse).lookup;
        require(sparseMatch.Peek() && sparseMatch.Peek()->display==words.back() && sparseMatch.NodeVisits()<3 && !sparseMatch.Next(),
                "sparse personal selection scanned or admitted unrelated uncommon words");
        auto selected=lexicon->IndexSelected(selectedKeys);
        auto first=lexicon->Query("prefix",Lexicon::Completion::Lookup,selected).lookup;
        auto second=lexicon->Query("prefix",Lexicon::Completion::Lookup,selected).lookup;
        require(first.Peek() && second.Peek() && first.NodeVisits()<40,"selected dense prefix unavailable or scanned");
        const std::string leading=second.Peek()->display;
        std::set<std::string> seen;
        uint32_t previous=0;
        while (const auto *entry=first.Peek()) {
            require(seen.insert(entry->display).second,"duplicate personal completion");
            require(seen.size()==1 || entry->rank>previous,"personal completion rank differs across pages");
            previous=entry->rank; first.Next();
        }
        require(seen.size()==1024 && second.Peek()->display==leading,"personal pagination truncated or another cursor changed");
        auto retained=[&]() {
            auto temporary=Lexicon::Load(file);
            return temporary->Query("prefix",Lexicon::Completion::Lookup,temporary->IndexSelected(selectedKeys)).lookup;
        }();
        require(collect(retained).size()==1024,"cursor did not retain its lexicon and selected index");
        require(lexicon->Query("prefix").empty(),"personal index mutated cold membership");
        auto exact=lexicon->Query("pre");
        require(collect(exact.exact)==std::vector<std::string>{"pre"} && !exact.preferred_exact && !exact.familiar_exact,
                "uncommon complete spelling was lost or gained autonomous leadership");
        require(lexicon->Query("pr",Lexicon::Completion::Lookup,selected).empty(),"two-letter prefix expanded");
        require(collect(lexicon->Query("github").exact)==std::vector<std::string>{"GitHub"},"source case was lost");
        require(lexicon->Query("Git").empty() && lexicon->Query("pre'").empty(),"unsupported spelling was normalized silently");
        std::string observed="ARION_ENGLISH\t2\t"+std::to_string(rows.size())+'\n';
        for(auto row:rows) {
            const auto field=row.rfind("\t0\t");
            require(field!=std::string::npos,"synthetic observation field missing");
            row.replace(field,3,"\t2000\t"); observed+=row+'\n';
        }
        write(file,observed);
        auto dense=Lexicon::Load(file);
        auto blocked=dense->Query("pre"),admitted=dense->Query("prefix");
        require(!blocked.suggestions.Peek() && blocked.suggestions.NodeVisits()==0,"overlong completion scanned rejected words");
        require(admitted.suggestions.NodeVisits()<40 && !admitted.lookup.Peek(),"dense common admission scanned or duplicated lookup");
        require(collect(admitted.suggestions).size()==1024,"common dense pagination was truncated");
        bool wrongOwner=false;
        try { dense->Query("prefix",Lexicon::Completion::Suggest,selected); } catch (const std::invalid_argument &) { wrongOwner=true; }
        require(wrongOwner,"selected index was reused with another lexicon's word IDs");

        write(file,"ARION_ENGLISH\t2\t3\ngit\t2\t2000\tgit,Git\ngithub\t0\t0\tGitHub,Github\ngitlab\t1\t0\tGitLab,gitlab\n");
        auto cases=Lexicon::Load(file);
        require(!cases->Query("git").lookup.Peek(),"technical-source case bypassed uncommon eligibility");
        auto caseSelection=cases->IndexSelected({"git","github","gitlab","retiredword"});
        auto prefix=cases->Query("git",Lexicon::Completion::Lookup,caseSelection).lookup;
        require(collect(prefix)==std::vector<std::string>{"GitHub","GitLab"},"selected canonical rank or one-form-per-prefix changed");
        require(collect(cases->Query("github").exact)==std::vector<std::string>{"GitHub","Github"},
                "complete uncommon spelling lost its exact case family");
        auto selectedExact=cases->Query("github",Lexicon::Completion::Suggest,caseSelection);
        require(selectedExact.preferred_exact && selectedExact.familiar_exact,"selected proper-case word lost familiarity");
        auto segments=cases->WordPrefixes("githubnihao");
        require(segments.size()==2 && segments[0].bytes==6 && segments[0].forms[0].display=="GitHub" &&
                segments[0].forms.size()==2 && segments[1].bytes==3,"complete mixed word lost ranges, order or case");
        require(cases->WordPrefixes("github").size()==1 && cases->WordPrefixes("git").empty(),
                "word-prefix query included the whole input or invented a completion");
        require(cases->WordPrefixes("github"+std::string(256,'n'))[0].bytes==6,"long mixed input hid a complete English prefix");
        require(cases->WordPrefixes("GitHubnihao").empty(),"prefix lookup silently folded an unsupported key");
        require(cases->Query("git").familiar_exact && !cases->Query("github").familiar_exact,"native lowercase commonness changed");
        for (const char *invalidKey:{"GitHub","bad-key","a", ""}) {
            bool rejected=false;
            try { cases->IndexSelected({invalidKey}); } catch (const std::invalid_argument &) { rejected=true; }
            require(rejected,"malformed selected key entered the index");
        }

        write(file,"ARION_ENGLISH\t2\t5\nartifice\t2\t0\tartifice,Artifice\nartificial\t0\t4523\tartificial,Artificial\nartificially\t1\t2200\tartificially\nxing\t3\t697\tXING\nxinput\t4\t0\txinput\n");
        auto admission=Lexicon::Load(file);
        auto eligible=admission->Query("artifi");
        require(collect(eligible.suggestions)==std::vector<std::string>{"artificial","artificially"},"common automatic ranking or membership changed");
        require(!eligible.lookup.Peek(),"unobserved prefix escaped through lookup");
        auto learned=admission->IndexSelected({"artifice","artificial","xing"});
        auto learnedMatches=admission->Query("artifi",Lexicon::Completion::Suggest,learned);
        require(collect(learnedMatches.suggestions)==std::vector<std::string>{"artificial","artificially","artifice"} && !learnedMatches.lookup.Peek(),
                "common and personal indexes did not merge by original rank without duplicates");
        auto shortPrefix=admission->Query("art");
        require(!shortPrefix.suggestions.Peek() && shortPrefix.suggestions.NodeVisits()==0,"coverage gate scanned rejected words");
        require(admission->Query("xin").empty(),"low-frequency prefix escaped through a later lane");
        require(collect(admission->Query("xin",Lexicon::Completion::Lookup,learned).lookup)==std::vector<std::string>{"XING"},
                "selection unlocked another word sharing its prefix");
        require(collect(admission->Query("xinput").exact)==std::vector<std::string>{"xinput"},"complete unknown spelling became unavailable");
        auto pinyin=admission->Query("artifi",Lexicon::Completion::Lookup);
        require(!pinyin.suggestions.Peek() && collect(pinyin.lookup)==std::vector<std::string>{"artificial","artificially"},
                "competing Pinyin lost eligible common coverage or leaked an uncommon word");

        write(file,"ARION_ENGLISH\t2\t4\nhabitat\t3\t1000\tHabitat\nhabitual\t0\t1000\thabitual,Habitual\nhabitually\t1\t999\thabitually\nhabitus\t2\t0\thabitus\n");
        auto boundary=Lexicon::Load(file);
        require(boundary->IsCommon("habitual") && boundary->IsCommon("habitat") && !boundary->IsCommon("habitually") &&
                !boundary->IsCommon("habitus") && boundary->Contains("habitus") && !boundary->Contains("absentword"),
                "existing source-frequency commonness boundary changed");
        require(collect(boundary->Query("hab").lookup)==std::vector<std::string>{"habitual","Habitat"},
                "999/unobserved or capitalization changed incomplete-word eligibility");
        require(boundary->Query("habitat").preferred_exact && !boundary->Query("habitat").familiar_exact,
                "source-native case was treated as lowercase evidence");
        require(!boundary->Query("habi").suggestions.Peek() && boundary->Query("habit").suggestions.Peek(),
                "foreground minimum length changed independently of commonness");
        require(collect(boundary->Query("habitually").exact)==std::vector<std::string>{"habitually"} &&
                collect(boundary->Query("habitus").exact)==std::vector<std::string>{"habitus"},"uncommon complete words were deleted");
        auto boundarySelected=boundary->IndexSelected({"habitually"});
        require(collect(boundary->Query("hab",Lexicon::Completion::Lookup,boundarySelected).lookup)==
                std::vector<std::string>{"habitual","habitually","Habitat"},"selected uncommon eligibility changed common rank or leaked its neighbor");

        for (const char *invalid:{"ARION_ENGLISH\t1\t1\nab\t0\t0\tab\n", "ARION_ENGLISH\t2\t2\nab\t0\t0\tab\n",
                "ARION_ENGLISH\t2\t2\nab\t0\t0\tab\nac\t0\t0\tac\n", "ARION_ENGLISH\t2\t1\nab\t0\t0\tAC\n",
                "ARION_ENGLISH\t2\t1\nab\t2\t0\tab\n", "ARION_ENGLISH\t2\t2\nac\t0\t0\tac\nab\t1\t0\tab\n",
                "ARION_ENGLISH\t2\t1\nab\t0\t4294967296\tab\n", "ARION_ENGLISH\t2\t1\nab\t0\t0\tab,ab\n"}) {
            write(file,invalid); bool rejected=false;
            try { Lexicon::Load(file); } catch (const std::exception &) { rejected=true; }
            require(rejected,"malformed lexicon was accepted");
        }
        std::cout<<"PASS source-frequency exact-only admission, selected-word indexes, ranked pagination, ownership and malformed asset rejection\n";
        return 0;
    } catch (const std::exception &error) { std::cerr<<error.what()<<'\n'; return 1; }
}
