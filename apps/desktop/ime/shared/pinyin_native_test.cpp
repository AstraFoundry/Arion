// Real pinned Rime conversion with the production encrypted learning provider.
// The caller supplies an isolated, newly created profile directory.
#include "pinyin.h"
#include "menu.h"
#include "input.h"
#include "learning_rime.h"
#include "emoji.h"
#include "english.h"
#include "english_admission_cases.h"
#include <algorithm>
#include "spelling.h"
#include "spelling_options.h"
#include <filesystem>
#include <iostream>
#include <stdexcept>
#include <string>

static void require(bool condition, const char *message) {
    if (!condition) throw std::runtime_error(message);
}
struct Snapshot {
    ArionPinyinSnapshot value{};
    Snapshot(RimeApi *api, uintptr_t session) {
        require(arion_pinyin_snapshot(api,session,&value)==0,"snapshot failed");
    }
    ~Snapshot() { arion_pinyin_snapshot_dispose(&value); }
    Snapshot(const Snapshot &) = delete;
};
static void type(RimeApi *api, uintptr_t session, const std::string &keys) {
    for (unsigned char key:keys) require(arion_pinyin_process_key(api,session,key,0),"Pinyin key not handled");
}
static void printCase(const char *name) { std::cout<<"PASS "<<name<<"\n"; }
static size_t candidateIndex(RimeApi *api,uintptr_t session,const std::string &expected) {
    char **texts=nullptr; size_t count=0, found=SIZE_MAX;
    require(arion_pinyin_candidates(api,session,0,189,&texts,&count),"candidate enumeration failed");
    for (size_t i=0;i<count;i++) if (expected==texts[i]) { found=i; break; }
    arion_pinyin_candidates_dispose(texts,count);
    require(found!=SIZE_MAX,"required real candidate absent");
    return found;
}
static void selectPhrase(RimeApi *api,uintptr_t session,const char *keys,const char *expected,const char *name) {
    arion_pinyin_clear(api,session); type(api,session,keys);
    require(arion_pinyin_select(api,session,candidateIndex(api,session,expected),0),"phrase selection failed");
    Snapshot result(api,session);
    require(std::string(result.value.commit)==expected,"phrase selection did not commit expected engine result");
    printCase(name);
}
static void convert(RimeApi *api, uintptr_t session,const char *keys,const char *expected,const char *name) {
    arion_pinyin_clear(api,session);
    type(api,session,keys);
    {
        Snapshot before(api,session);
        require(before.value.count>0 && before.value.preedit[0],"real candidate/preedit missing");
        require(std::string(before.value.commit).empty(),"composition committed before selection");
        require(std::string(before.value.candidates[0])==expected,"unexpected pinned first candidate");
    }
    require(arion_pinyin_process_key(api,session,' ',0),"Space selection not handled");
    Snapshot after(api,session);
    require(std::string(after.value.commit)==expected,"selected text not committed by engine");
    require(std::string(after.value.preedit).empty(),"preedit survived commit");
    Snapshot repeated(api,session);
    require(std::string(repeated.value.commit).empty(),"commit consumed more than once");
    printCase(name);
}
int main(int argc,char **argv) {
    if (argc!=6) return 2;
    RimeApi *api=rime_get_api(); bool initialized=false;
    try {
        std::filesystem::path profile=std::filesystem::u8path(argv[2]);
        require(std::filesystem::is_directory(profile) && std::filesystem::is_empty(profile),"profile must be a new empty directory");
        std::string data=argv[1], user=argv[2], build=(std::filesystem::u8path(data)/"build").u8string();
        RIME_STRUCT(RimeTraits,traits);
        traits.shared_data_dir=data.c_str(); traits.prebuilt_data_dir=build.c_str(); traits.staging_dir=build.c_str();
        traits.user_data_dir=user.c_str(); traits.app_name="rime.arion"; traits.min_log_level=3; traits.log_dir="";
        api->setup(&traits);
        char error[512]{};
        std::string learning=(profile/"learning").u8string();
        require(arion_learning_windows_register(learning.c_str(),error,sizeof(error)),error);
        arion_emoji_register();
        api->initialize(&traits); initialized=true;
        arion_spelling_register();
        require(arion_english_register(data.c_str(),error,sizeof(error)),error);
        require(!arion_learning_error(error,sizeof(error)),error);
        auto first=arion_pinyin_create_session(api),second=arion_pinyin_create_session(api);
        require(first && second && first!=second,"independent sessions unavailable");
        arion_pinyin_set_options(api,first,0,0,0); arion_pinyin_set_options(api,second,0,0,0);
        convert(api,first,"nihao","\xe4\xbd\xa0\xe5\xa5\xbd","full-pinyin-nihao");
        convert(api,first,"zhongguo","\xe4\xb8\xad\xe5\x9b\xbd","full-pinyin-zhongguo");
        for (const char *word : {"artificial", "hello", "computer", "software", "can", "an", "you", "hi", "bailing", "jingling"}) {
            arion_pinyin_clear(api,first); type(api,first,word);
            auto index=candidateIndex(api,first,word);
            require(index<5,"English exact match missing from smallest candidate page");
            if (std::string(word)=="can" || std::string(word)=="an" || std::string(word)=="you") {
                Snapshot chinese(api,first);
                require(std::string(chinese.value.candidates[0])!=word,"ambiguous English displaced first Chinese word");
            }
            require(arion_pinyin_select(api,first,index,0),"English engine selection failed");
            Snapshot english(api,first);
            require(std::string(english.value.commit)==word && !english.value.ascii,"English commit changed mode or text");
        }
        convert(api,first,"artificial","artificial","english-space-engine-commit");
        selectPhrase(api,first,"artificial","Artificial","english-sentence-case-engine-selection");
        printCase("english-candidates-in-chinese-mode");
        for (auto sample : {std::pair<const char *,const char *>{"artifi","artificial"},
                            {"compu","computer"},{"softw","software"},{"hel","hello"}}) {
            arion_pinyin_clear(api,first); type(api,first,sample.first);
            auto index=candidateIndex(api,first,sample.second);
            require(index<60,"English prefix completion missing from paginated candidates");
            require(arion_pinyin_select(api,first,index,0),"English completion engine selection failed");
            Snapshot result(api,first);
            require(std::string(result.value.commit)==sample.second && !result.value.ascii,
                    "English completion changed word or language mode");
        }
        printCase("english-prefix-completion-in-chinese-mode");
        for (auto sample : {std::pair<const char *,const char *>{"github","GitHub"},
                            {"javascript","JavaScript"},{"api","API"},{"frontend","frontend"},{"backend","backend"}})
            selectPhrase(api,first,sample.first,sample.second,"expanded-english-canonical-case");
        arion_pinyin_clear(api,first); type(api,first,"artifi");
        char **expanded=nullptr; size_t expandedCount=0;
        require(arion_pinyin_candidates(api,first,0,90,&expanded,&expandedCount),"expanded English menu unavailable");
        size_t distinctEnglish=0;
        for (size_t i=0;i<expandedCount;++i) if (std::string(expanded[i]).rfind("artifi",0)==0) ++distinctEnglish;
        arion_pinyin_candidates_dispose(expanded,expandedCount);
        require(distinctEnglish>3,"lower-ranked English words were truncated");
        arion_pinyin_clear(api,first);
        for(int fixture=3;fixture<argc;++fixture) for(size_t pageSize:{size_t(5),size_t(9)}) {
            for(const auto &sample:EnglishAdmissionCases(std::filesystem::u8path(argv[fixture]))) {
                arion_pinyin_clear(api,first); type(api,first,sample.input.c_str());
                ArionPinyinMenu admitted{};
                require(arion_menu_build(&admitted,api,first,pageSize,nullptr,nullptr),"admission native menu");
                const size_t visible=arion_menu_visible_count(&admitted);
                size_t englishCount=0,selected=SIZE_MAX;
                std::cout<<"ADMISSION\t"<<sample.id<<'\t'<<pageSize;
                for(size_t i=0;i<visible;++i) {
                    englishCount+=EnglishAdmissionWord(admitted.texts[i]);
                    if(admitted.texts[i]==sample.expected) selected=i;
                    std::cout<<'\t'<<admitted.texts[i];
                }
                std::cout<<std::endl;
                require(englishCount<=sample.maximumEnglish,(sample.id+" unexpected English suggestions").c_str());
                require(!sample.chineseFirst || (visible && static_cast<unsigned char>(admitted.texts[0][0])>=128),"admission Chinese priority");
                size_t engineIndex=0;
                if(sample.expected!="-") require(selected!=SIZE_MAX &&
                    arion_menu_engine_index(&admitted,selected,0,0,&engineIndex),(sample.id+" expected English on first page").c_str());
                arion_menu_dispose(&admitted);
                if(sample.expected!="-") {
                    require(arion_pinyin_select(api,first,engineIndex,0),"admission actual candidate selection");
                    Snapshot committed(api,first);
                    require(sample.expected==committed.value.commit,"admission actual commit");
                    Snapshot once(api,first);
                    require(!once.value.commit[0],"admission duplicate commit");
                }
                arion_pinyin_clear(api,first);
            }
        }
        printCase("english-admission-development-and-validation-five-nine-pages");
        selectPhrase(api,first,"ni'hao","\xe4\xbd\xa0\xe5\xa5\xbd","apostrophe-syllable-boundary");
        selectPhrase(api,first,"nh","\xe4\xbd\xa0\xe5\xa5\xbd","abbreviated-pinyin");
        selectPhrase(api,first,"lvse","\xe7\xbb\xbf\xe8\x89\xb2","umlaut-v-spelling");
        arion_pinyin_clear(api,first);
        const int literalKeys[]={'l',0x00fc,'s','e'};
        for (int key : literalKeys)
            require(arion_pinyin_process_key(api,first,arion_key_normalize(key,0,0),0),"literal umlaut key not handled");
        require(arion_pinyin_select(api,first,candidateIndex(api,first,u8"绿色"),0),"literal umlaut candidate not selected");
        { Snapshot literal(api,first); require(std::string(literal.value.commit)==u8"绿色","literal umlaut did not commit real conversion"); }
        printCase("literal-umlaut-shared-normalization");
        type(api,first,"nihao");
        require(arion_pinyin_select(api,first,candidateIndex(api,first,"\xe4\xbd\xa0"),0),"prefix selection failed");
        { Snapshot prefix(api,first); require(prefix.value.preedit[0] && !prefix.value.commit[0],"partial selection committed prematurely"); }
        require(arion_pinyin_process_key(api,first,' ',0),"suffix selection failed");
        { Snapshot suffix(api,first); require(std::string(suffix.value.commit)=="\xe4\xbd\xa0\xe5\xa5\xbd","partial selection lost suffix"); }
        printCase("partial-selection-suffix-continuation");
        type(api,first,"nihao");
        { Snapshot isolated(api,second); require(!isolated.value.preedit[0] && !isolated.value.count,"session state leaked"); }
        printCase("independent-sessions");
        require(arion_pinyin_process_key(api,first,0xff1b,0),"Escape not handled");
        { Snapshot cancelled(api,first); require(!cancelled.value.preedit[0] && !cancelled.value.commit[0],"cancel committed text"); }
        printCase("escape-cancels-without-commit");
        type(api,first,"nihaoo");
        require(arion_pinyin_process_key(api,first,0xff08,0),"Backspace not handled");
        { Snapshot edited(api,first); require(edited.value.count && std::string(edited.value.candidates[0])=="\xe4\xbd\xa0\xe5\xa5\xbd","Backspace did not restore conversion"); }
        printCase("backspace-reconverts");
        arion_pinyin_clear(api,first);
        type(api,first,"shi");
        { Snapshot before(api,first); require(!before.value.last_page,"paging fixture lacks candidates"); }
        require(arion_pinyin_process_key(api,first,0xff56,0),"PageDown not handled");
        { Snapshot page(api,first); require(page.value.page==1 && page.value.count,"next page absent"); }
        printCase("candidate-page-down");
        arion_pinyin_clear(api,first);
        type(api,first,"shi");
        ArionPinyinMenu menu{};
        require(arion_menu_build(&menu,api,first,5,nullptr,nullptr),"shared menu build failed");
        require(arion_menu_prepare(&menu,api,first) && arion_menu_visible_count(&menu)==5,"configured candidate page size ignored");
        require(arion_menu_grid(&menu,api,first,1) && arion_menu_turn(&menu,api,first,1),"shared expanded menu navigation failed");
        require(arion_menu_prepare(&menu,api,first) && arion_menu_row_count(&menu)>1,"expanded rows unavailable");
        size_t engine_index=0;
        require(arion_menu_engine_index(&menu,1,0,0,&engine_index) && engine_index==6,"display-to-engine mapping differs");
        std::string chosen=menu.texts[6];
        require(arion_pinyin_select(api,first,engine_index,0),"shared menu candidate selection failed");
        { Snapshot selected(api,first); require(std::string(selected.value.commit)==chosen,"selected engine index committed another candidate"); }
        arion_menu_dispose(&menu);
        printCase("shared-configurable-menu-expansion-and-engine-index-selection");
        arion_pinyin_clear(api,first);
        arion_pinyin_set_options(api,first,1,1,0);
        require(!arion_pinyin_process_key(api,first,'a',0),"English-mode Latin was consumed");
        { Snapshot english(api,first); require(english.value.ascii && !english.value.commit[0] && !english.value.preedit[0],"English mode produced Chinese/full-width text"); }
        printCase("english-pass-through-with-full-width-preference");
        arion_pinyin_set_options(api,first,0,0,0);
        require(arion_pinyin_process_key(api,first,',',0),"Chinese punctuation not handled");
        { Snapshot punctuation(api,first); require(std::string(punctuation.value.commit)=="\xef\xbc\x8c","Chinese comma differs"); }
        printCase("direct-chinese-punctuation");
        api->set_option(first,"arion_emoji",True);
        type(api,first,"yinggelan");
        const std::string tagFlag=u8"\U0001f3f4\U000e0067\U000e0062\U000e0065\U000e006e\U000e0067\U000e007f";
        size_t tagIndex=candidateIndex(api,first,tagFlag);
        ArionCandidateMetadata tagMetadata{};
        require(arion_candidate_metadata(first,tagIndex,&tagMetadata) && tagMetadata.emoji,
                "subdivision flag metadata unavailable");
        require(std::string(tagMetadata.sequence_id)=="1F3F4-E0067-E0062-E0065-E006E-E0067-E007F",
                "subdivision flag sequence ID truncated");
        require(arion_pinyin_select(api,first,tagIndex,0),"subdivision flag selection failed");
        { Snapshot flag(api,first); require(std::string(flag.value.commit)==tagFlag,"subdivision flag commit changed Unicode"); }
        printCase("emoji-subdivision-flag-metadata-and-exact-commit");
        arion_spelling_apply(api,second,1,0);
        for (const char *keys : {"zhngguo", "zhoongguo", "zhognguo", "zhomgguo"}) {
            arion_pinyin_clear(api,second); type(api,second,keys);
            const auto index=candidateIndex(api,second,u8"中国");
            ArionCandidateMetadata coverage{};
            require(arion_candidate_metadata(second,index,&coverage) && coverage.start==0 &&
                    coverage.end==std::string(keys).size(),"corrected candidate must cover the entire raw input");
            require(arion_pinyin_select(api,second,index,0),"corrected candidate selection failed");
            { Snapshot corrected(api,second); require(std::string(corrected.value.commit)==u8"中国" &&
                !corrected.value.preedit[0],"corrected full-input commit differs"); }
            { Snapshot consumed(api,second); require(!consumed.value.commit[0],"corrected commit repeated"); }
        }
        printCase("four-typo-operations-full-raw-coverage-and-single-commit");
        arion_spelling_apply(api,second,0,1u);
        selectPhrase(api,second,"zongguo",u8"中国","independent-z-zh-fuzzy-with-typo-disabled");
        arion_spelling_apply(api,second,0,0);
        for (const char *word : {"hello", "Hello", "HELLO", "hElLo"})
            selectPhrase(api,second,word,word,"english-candidate-preserves-explicit-case");
        for (bool englishFirst : {false,true}) {
            arion_pinyin_clear(api,second);
            type(api,second,englishFirst?"HELLOnihao":"nihaoHELLO");
            require(arion_pinyin_select(api,second,candidateIndex(api,second,englishFirst?"HELLO":u8"你好"),0),
                    "mixed prefix selection failed");
            { Snapshot prefix(api,second); require(prefix.value.preedit[0] && !prefix.value.commit[0],
                    "mixed prefix committed before suffix confirmation"); }
            require(arion_pinyin_select(api,second,candidateIndex(api,second,englishFirst?u8"你好":"HELLO"),0),
                    "mixed suffix selection failed");
            { Snapshot final(api,second); require(std::string(final.value.commit)==(englishFirst?u8"HELLO你好":u8"你好HELLO") &&
                    !final.value.preedit[0],"mixed final text or termination differs"); }
            { Snapshot consumed(api,second); require(!consumed.value.commit[0],"mixed commit repeated"); }
        }
        printCase("bidirectional-mixed-partial-conversion-commits-once");
        arion_pinyin_destroy_session(api,first); arion_pinyin_destroy_session(api,second);
        api->finalize(); initialized=false;
        require(!arion_learning_error(error,sizeof(error)),error);
        std::cout<<"PASS production-encrypted-learning-lifecycle\n";
        return 0;
    } catch (const std::exception &error) {
        std::cerr<<"FAIL "<<error.what()<<"\n";
        if (initialized) api->finalize();
        return 1;
    }
}
