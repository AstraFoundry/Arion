#pragma once
#include "english_history.h"
#include <algorithm>
#include <filesystem>
#include <fstream>
#include <stdexcept>

inline void testEnglishHistory(const std::shared_ptr<arion::ime::LearningProvider> &provider) {
    using namespace arion::ime;
    auto check=[](bool ok,const char *message) { if (!ok) throw std::runtime_error(message); };
    const auto fixture=std::filesystem::u8path(provider->directory())/"fixture.lexicon";
    std::ofstream(fixture)<<"ARION_ENGLISH\t2\t4\ncommon\t0\t1000\tcommon\nrareone\t1\t0\tRareOne\nrarethree\t3\t0\trarethree\nraretwo\t2\t999\traretwo\n";
    const auto lexicon=arion::EnglishLexicon::Load(fixture);
    std::filesystem::remove(fixture);
    LearningError error;
    const auto file=learningPath(provider->directory(),"arion_english_selected.arion-userdb");
    auto history=std::make_unique<EnglishHistory>(provider,lexicon);
    check(history->open(error),"open empty English history");
    EnglishHistory::Snapshot empty,selected;
    check(history->snapshot(empty,error) && lexicon->Query("rare",arion::EnglishLexicon::Completion::Lookup,empty).empty(),
          "missing history grants no rare completion");
    check(history->confirm({"rareone"},error) && history->snapshot(selected,error),"persist selected English key");
    check(lexicon->Query("rare",arion::EnglishLexicon::Completion::Lookup,selected).lookup.Peek()->display=="RareOne",
          "selection unlocks only the selected full English word");
    check(lexicon->Query("rare",arion::EnglishLexicon::Completion::Lookup,empty).empty(),"old composition snapshot stays immutable");
    LearningBytes sealed; bool missing=false;
    check(provider->read(file,sealed,missing,error) && !missing,"read encrypted English snapshot");
    const std::string canary="rareone";
    check(std::search(sealed.begin(),sealed.end(),canary.begin(),canary.end())==sealed.end(),"English selected key is never plaintext");
    check(history->confirm({"rareone","common"},error),"idempotent confirmation and common word selection");
    LearningBytes repeated;
    check(provider->read(file,repeated,missing,error) && repeated==sealed,"repeat and common selection do not rewrite history");
    check(history->confirmRaw("RareOne",error) && history->confirmRaw("NewWord",error) && history->snapshot(selected,error),
          "raw confirmation learns both bundled and unknown English words");
    check(lexicon->Query("newword",arion::EnglishLexicon::Completion::Lookup,selected).exact.Peek()->display=="NewWord",
          "unknown English word preserves typed case");
    check(lexicon->Query("newwo",arion::EnglishLexicon::Completion::Suggest,selected).suggestions.Peek(),
          "unknown English word supports later prefix completion");
    check(lexicon->Query("newword",arion::EnglishLexicon::Completion::Lookup,empty).empty(),
          "raw learning does not mutate an active composition snapshot");
    check(provider->read(file,sealed,missing,error),"read raw English confirmation snapshot");
    const std::string rawCanary="NewWord";
    check(std::search(sealed.begin(),sealed.end(),rawCanary.begin(),rawCanary.end())==sealed.end(),"new English display spelling is never plaintext");
    for (const char *word:{"", "a", "ni'hao", "word word", "word123", "hello-world"})
        check(history->confirmRaw(word,error),"unsupported raw spelling is ignored");
    check(history->confirmRaw("NewWord",error) && history->confirm({"newword"},error),"repeated new word selections are idempotent");
    check(provider->read(file,repeated,missing,error) && repeated==sealed,"invalid and repeated raw confirmations do not rewrite history");
    check(!resetLearningVault(provider,error),"active English history lease refuses reset");
    if (provider->cooperative()) {
        EnglishHistory peer(provider,lexicon);
        check(peer.open(error) && peer.confirm({"rareone","raretwo"},error),"peer same-key and different-key selections merge");
        check(history->snapshot(selected,error),"observe peer selection at composition boundary");
        auto matches=lexicon->Query("rare",arion::EnglishLexicon::Completion::Lookup,selected).lookup;
        check(matches.Peek() && matches.Next() && matches.Peek()->display=="raretwo","peer selection preserves both keys");
    }
    history.reset();
    history=std::make_unique<EnglishHistory>(provider,lexicon);
    check(history->open(error) && history->snapshot(selected,error),"selected English survives close and reopen");
    check(lexicon->Query("rare",arion::EnglishLexicon::Completion::Lookup,selected).lookup.Peek(),"reopened history authorizes completion");
    check(lexicon->Query("newword",arion::EnglishLexicon::Completion::Lookup,selected).exact.Peek()->display=="NewWord",
          "new English word and case survive close and reopen");

    // Force the actual replacement to fail; no failed transaction may survive close.
    const auto saved=file+".saved";
    std::filesystem::rename(std::filesystem::u8path(file),std::filesystem::u8path(saved));
    std::filesystem::create_directory(std::filesystem::u8path(file));
    check(!history->confirm({"rarethree"},error),"unwritable English history rejects confirmation");
    check(error.code!=0,"English persistence failure stays explicit");
    history.reset();
    check(std::filesystem::is_directory(std::filesystem::u8path(file)),"destruction cannot retry failed transaction");
    std::filesystem::remove(std::filesystem::u8path(file));
    std::filesystem::rename(std::filesystem::u8path(saved),std::filesystem::u8path(file));

    {
        LearningDatabase invalid(provider,file,"arion_english_selected");
        check(invalid.open() && invalid.update("INVALID","1"),"write authenticated invalid history fixture");
    }
    history=std::make_unique<EnglishHistory>(provider,lexicon);
    check(!history->open(error) && error.code,"authenticated malformed English records reject startup");
    history.reset();
    check(resetLearningVault(provider,error),"closed English history resets with existing vault");
    history=std::make_unique<EnglishHistory>(provider,lexicon);
    check(history->open(error) && history->snapshot(empty,error) && lexicon->Query("rare",arion::EnglishLexicon::Completion::Lookup,empty).empty(),
          "reset removes learned English exception");
    history.reset();
}
