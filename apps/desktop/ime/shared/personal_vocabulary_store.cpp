#include "personal_vocabulary_store.h"
#include "learning_rime.h"
#include <charconv>

namespace arion::ime {
PersonalVocabularyStore::PersonalVocabularyStore(std::shared_ptr<LearningProvider> provider)
    : database_(provider,provider?learningPath(provider->directory(),"arion_personal.arion-userdb"):"","arion_personal") {}
bool PersonalVocabularyStore::fail(const char* operation) { error_={10,operation};return false; }
bool PersonalVocabularyStore::open() {
    if (!database_.open()) {error_=database_.error();return false;}
    error_={};return true;
}

bool PersonalVocabularyStore::load(rime::Prism& prism,uint64_t& revision,std::vector<PersonalTerm>& terms) {
    if (!database_.loaded()) return fail("personal projection unopened");
    auto records=database_.records();
    if (database_.error().code) {error_=database_.error();return false;}
    if (records.empty()) {revision=0;terms.clear();error_={};return true;}
    if (records.size()!=1 || !records.count("projection") || records.at("projection").size()>maximumLearningField) return fail("personal projection envelope");
    const auto& body=records.at("projection");
    LearningRecords fields;
    if (!decodeLearningRecords(LearningBytes(body.begin(),body.end()),fields,error_)) return false;
    auto version=fields.find("version"),sequence=fields.find("revision");
    if (version==fields.end() || version->second!="1" || sequence==fields.end()) return fail("personal projection version");
    uint64_t next=0;
    auto parsed=std::from_chars(sequence->second.data(),sequence->second.data()+sequence->second.size(),next);
    if (!next || parsed.ec!=std::errc{} || parsed.ptr!=sequence->second.data()+sequence->second.size()) return fail("personal projection revision");
    std::vector<PersonalTerm> restored;
    for (const auto& [key,text]:fields) {
        if (key=="version" || key=="revision") continue;
        if (key.size()<3 || key[1]!='/' || (key[0]!='t' && key[0]!='r')) return fail("personal projection field");
        const auto id=key.substr(2);
        if (!fields.count("t/"+id) || !fields.count("r/"+id)) return fail("personal projection incomplete term");
        if (key[0]=='t') restored.push_back({id,text,fields.at("r/"+id)});
    }
    PersonalVocabulary validated;
    if (!validated.replace(restored,prism)) return fail("personal projection reading");
    revision=next;terms=std::move(restored);error_={};return true;
}
bool PersonalVocabularyStore::save(rime::Prism& prism,uint64_t revision,const std::vector<PersonalTerm>& terms) {
    PersonalVocabulary validated;
    if (!revision || !validated.replace(terms,prism)) return fail("personal projection invalid update");
    uint64_t previous=0;std::vector<PersonalTerm> old;
    if (!load(prism,previous,old)) return false;
    if (revision<=previous) return fail("personal projection stale update");
    LearningRecords fields{{"version","1"},{"revision",std::to_string(revision)}};
    for (const auto& term:terms) {fields["t/"+term.id]=term.text;fields["r/"+term.id]=term.reading;}
    LearningBytes encoded;
    if (!encodeLearningRecords(fields,encoded,error_)) return false;
    if (encoded.size()>maximumLearningField) return fail("personal projection size");
    // update uses the observed complete old record as its compare-and-swap base.
    if (!database_.update("projection",std::string(encoded.begin(),encoded.end()))) {error_=database_.error();return false;}
    error_={};return true;
}
}

extern "C" int arion_personal_vocabulary_restore(uintptr_t session,uint64_t* revision) {
    if (!revision) return 0;
    uint64_t current=0;
    auto* prism=arion::ime::personalVocabularySession(session,current);
    if (!prism) return 0;
    arion::ime::PersonalVocabularyStore store(arion::ime::registeredLearningProvider());
    uint64_t saved=0;std::vector<arion::ime::PersonalTerm> terms;
    if (!store.open() || !store.load(*prism,saved,terms) || saved<current) return 0;
    if (saved>current) {
        std::vector<ArionPersonalTerm> views;
        for (const auto& term:terms) views.push_back({term.id.c_str(),term.text.c_str(),term.reading.c_str()});
        if (!arion_personal_vocabulary_replace(session,saved,views.data(),views.size())) return 0;
    }
    *revision=saved;return 1;
}

extern "C" int arion_personal_vocabulary_publish(uintptr_t session,uint64_t revision,const ArionPersonalTerm* terms,size_t count) {
    uint64_t current=0;
    auto* prism=arion::ime::personalVocabularySession(session,current);
    std::vector<arion::ime::PersonalTerm> entries;
    if (!prism || revision<=current || !arion::ime::decodePersonalTerms(terms,count,entries)) return 0;
    arion::ime::PersonalVocabularyStore store(arion::ime::registeredLearningProvider());
    if (!store.open() || !store.save(*prism,revision,entries)) return 0;
    return arion_personal_vocabulary_replace(session,revision,terms,count);
}
