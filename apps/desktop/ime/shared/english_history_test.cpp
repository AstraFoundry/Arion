// Portable history transactions with an opaque in-memory provider. OS encryption
// and real editor dispatch remain separate native acceptance checks.
#include "english_history.h"
#include <fstream>
#include <iostream>
#include <stdexcept>

using namespace arion::ime;
static void require(bool value,const char *message) {
    if (!value) throw std::runtime_error(message);
}
class MemoryProvider final : public LearningProvider {
    const std::string directory_="english-history-test";
    std::map<std::string,LearningBytes> files_;
    std::map<LearningBytes,std::pair<std::string,LearningBytes>> envelopes_;
    uint64_t next_=0;
    class Lock final : public LearningLock {};
public:
    bool refuse=false;
    size_t writes=0;
    const std::string &directory() const override { return directory_; }
    bool lock(const std::string &,bool,bool,std::unique_ptr<LearningLock> &held,LearningError &) override {
        held=std::make_unique<Lock>(); return true;
    }
    bool read(const std::string &path,LearningBytes &bytes,bool &missing,LearningError &) override {
        const auto found=files_.find(path);
        missing=found==files_.end();
        bytes=missing?LearningBytes{}:found->second;
        return true;
    }
    bool writeAtomic(const std::string &path,const LearningBytes &bytes,LearningError &error) override {
        if (refuse) { error={6,"injected write failure"}; return false; }
        files_[path]=bytes; ++writes; return true;
    }
    bool remove(const std::string &path,LearningError &) override { files_.erase(path); return true; }
    bool files(std::vector<std::string> &,LearningError &) override { return false; }
    bool seal(const std::string &name,const LearningBytes &plain,LearningBytes &sealed,LearningError &) override {
        const auto token=std::to_string(++next_);
        sealed.assign(token.begin(),token.end());
        envelopes_[sealed]={name,plain}; return true;
    }
    bool unseal(const std::string &name,const LearningBytes &sealed,LearningBytes &plain,LearningError &error) override {
        const auto found=envelopes_.find(sealed);
        if (found==envelopes_.end() || found->second.first!=name) { error={5,"unknown test envelope"}; return false; }
        plain=found->second.second; return true;
    }
};
int main(int argc,char **argv) {
    if (argc!=2) return 2;
    try {
        using Lexicon=arion::EnglishLexicon;
        const auto fixture=std::filesystem::u8path(argv[1]);
        std::ofstream(fixture)<<"ARION_ENGLISH\t2\t2\ncommon\t0\t1000\tcommon\nrareword\t1\t0\tRareWord\n";
        const auto lexicon=Lexicon::Load(fixture);
        auto provider=std::make_shared<MemoryProvider>();
        LearningError error;
        EnglishHistory::Snapshot original,learned;
        {
            EnglishHistory history(provider,lexicon);
            require(history.open(error) && history.snapshot(original,error),"open original history snapshot");
            require(history.confirmRaw("RareWord",error) && history.confirmRaw("NewWord",error),"confirm bundled and unknown raw English");
            require(history.snapshot(learned,error),"capture learned snapshot");
            require(lexicon->Query("newwo",Lexicon::Completion::Suggest,learned).suggestions.Peek()->display=="NewWord",
                    "raw word cannot complete with original case");
            require(lexicon->Query("newwo",Lexicon::Completion::Suggest,original).empty(),"active snapshot changed after confirmation");
            const auto writes=provider->writes;
            for (const char *word:{"NewWord","newword","common","","x","two words","ni'hao","word123"})
                require(history.confirmRaw(word,error),"ignore invalid or repeated raw spelling");
            require(history.confirm({"newword","rareword"},error) && writes==provider->writes,"repeated confirmation rewrites history");
        }
        {
            EnglishHistory reopened(provider,lexicon);
            require(reopened.open(error) && reopened.snapshot(learned,error),"reopen confirmed history");
            require(lexicon->Query("newword",Lexicon::Completion::Lookup,learned).exact.Peek()->display=="NewWord",
                    "new English word did not survive reopening");
            require(lexicon->Query("rarew",Lexicon::Completion::Suggest,learned).suggestions.Peek(),"bundled raw confirmation was lost");
            provider->refuse=true;
            require(!reopened.confirmRaw("FailedWord",error) && error.code,"write failure was hidden");
        }
        provider->refuse=false;
        {
            EnglishHistory reopened(provider,lexicon);
            require(reopened.open(error) && reopened.snapshot(learned,error),"reopen after refused write");
            require(lexicon->Query("failedword",Lexicon::Completion::Lookup,learned).empty(),"failed write unlocked an English word");
        }
        std::cout<<"PASS raw English history, original case, immutable snapshots, reopen, idempotence and failed-write isolation\n";
    } catch (const std::exception &error) { std::cerr<<error.what()<<'\n'; return 1; }
}
