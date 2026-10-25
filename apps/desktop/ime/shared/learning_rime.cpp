#include "learning_rime.h"
#include "ranking.h"
#include <rime/registry.h>
#include <rime/dict/db.h>
#include <rime/dict/user_db.h>
#include <cstring>
#include <algorithm>
#include <rime/algo/dynamics.h>
#include <climits>
#include <cmath>
#include <sstream>
#include <charconv>

namespace arion::ime {
namespace {
std::mutex globalMutex;
LearningError globalError;
std::shared_ptr<LearningProvider> registeredProvider;
struct RimeValue { int64_t count=0;double dee=0;uint64_t tick=0; };
bool rimeValue(const std::string &text,RimeValue &value) {
    std::istringstream input(text);std::string c,d,t,extra;
    if (!(input>>c>>d>>t) || input>>extra || c.rfind("c=",0) || d.rfind("d=",0) || t.rfind("t=",0)) return false;
    auto count=std::from_chars(c.data()+2,c.data()+c.size(),value.count),tick=std::from_chars(t.data()+2,t.data()+t.size(),value.tick);
    std::istringstream floating(d.substr(2));
    return count.ec==std::errc{} && count.ptr==c.data()+c.size() && tick.ec==std::errc{} && tick.ptr==t.data()+t.size() &&
        value.count>=-INT_MAX && value.count<=INT_MAX && (floating>>value.dee) && floating.eof() && std::isfinite(value.dee) && value.dee>=0;
}
bool reconcileRime(const LearningRecords &base,const LearningChanges &changes,const LearningRecords &latest,LearningRecords &merged,LearningError &error) {
    // Pinned Rime 1.17: dict/user_dictionary.cc UpdateEntry/UpdateTickCount
    // and algo/dynamics.h. Replay this writer's confirmation, not an absolute
    // cached counter, after reloading the latest authenticated snapshot.
    const std::string clockKey="\x01/tick";LearningClock clock;
    if (!reconcileLearningClock(base,changes,latest,clockKey,clock,error)) return false;
    LearningChanges plain;struct Entry {std::string key;RimeValue before,after,current;};std::vector<Entry> entries;
    for (const auto &[key,value]:changes) {
        if (key==clockKey) continue;
        auto old=base.find(key),current=latest.find(key);Entry entry{key,{},{},{}};
        bool known=value && key.find('\t')!=std::string::npos && rimeValue(*value,entry.after) &&
            (old==base.end() || rimeValue(old->second,entry.before)) && (current==latest.end() || rimeValue(current->second,entry.current));
        if (known) entries.push_back(std::move(entry));else plain[key]=value;
    }
    if (!reconcileLearningChanges(base,plain,latest,merged,error)) return false;
    if (changes.count(clockKey)) merged[clockKey]=std::to_string(clock.globalEnd);
    for (const auto &entry:entries) {
        const auto &before=entry.before,&after=entry.after,&current=entry.current;
        auto reject=[&]() {error={10,"unsupported concurrent Rime record change"};return false;};
        if (after.tick>clock.localEnd) return reject();
        uint64_t position=after.tick>clock.localBase?after.tick-clock.localBase:0;
        uint64_t tick=std::max(current.tick,clock.globalBase+position);
        int64_t count=current.count;double increment=0;
        if (after.count<0) count=-std::max<int64_t>(1,std::abs(current.count));
        else {
            int64_t delta=after.count-std::abs(before.count);
            if (delta<0 || delta>INT_MAX-std::abs(current.count)) return reject();
            count=std::abs(current.count)+delta;
            if (delta>0) increment=static_cast<double>(delta);
            else if (after.dee!=before.dee || after.tick!=before.tick) increment=0.1;
        }
        double dee=rime::algo::formula_d(increment,static_cast<double>(tick),std::min(10000.0,current.dee),static_cast<double>(current.tick));
        std::ostringstream value;value<<"c="<<count<<" d="<<dee<<" t="<<tick;merged[entry.key]=value.str();
    }
    return true;
}
void report(const std::string &name, const LearningError &error) { if (name=="arion_context" || !error.code) return; std::lock_guard<std::mutex> lock(globalMutex); globalError=error; }
class Cursor final : public rime::DbAccessor {
    LearningRecords records_;
    LearningRecords::const_iterator current_;
    bool metadata_;
public:
    Cursor(LearningRecords records,const std::string &prefix,bool metadata):DbAccessor(prefix),records_(std::move(records)),metadata_(metadata) { Reset(); }
    bool Reset() override { current_=records_.lower_bound(prefix_); return true; }
    bool Jump(const std::string &key) override { current_=records_.lower_bound(key); return true; }
    bool exhausted() override { return current_==records_.end() || current_->first.compare(0,prefix_.size(),prefix_)!=0; }
    bool GetNextRecord(std::string *key,std::string *value) override { if (!key || !value || exhausted()) return false; *key=current_->first; if (metadata_) key->erase(0,1); *value=current_->second; ++current_; return true; }
};
class Database final : public rime::Db, public rime::Transactional {
    LearningDatabase db_;
    bool result(bool ok) { loaded_=db_.loaded(); readonly_=db_.readonly(); disabled_=disabled_||db_.disabled(); in_transaction_=db_.inTransaction(); if (!ok) report(name_,db_.error()); return ok; }
    rime::an<rime::DbAccessor> query(const std::string &prefix,bool metadata) { if (!loaded_) return nullptr;auto records=db_.records(false);if (db_.error().code) {report(name_,db_.error());return nullptr;}return std::make_shared<Cursor>(std::move(records),prefix,metadata); }
public:
    Database(std::shared_ptr<LearningProvider> provider,const std::string &file,const std::string &name):Db(rime::path(file),name),db_(std::move(provider),file,name,reconcileRime) {}
    ~Database() override { Close(); }
    bool Open() override { if (!result(db_.open())) return false; if (db_.wasMissing() && !CreateMetadata()) { Close(); return false; } return true; }
    bool OpenReadOnly() override { return result(db_.open(true)); }
    bool Close() override { return result(db_.close()); }
    bool Remove() override { return result(db_.remove()); }
    bool Backup(const rime::path &destination) override { return result(db_.backup(destination.string())); }
    bool Restore(const rime::path &source) override { return result(db_.restore(source.string())); }
    bool CreateMetadata() override { return MetaUpdate("/db_name",name_) && MetaUpdate("/db_type","userdb") && MetaUpdate("/rime_version",rime_get_api()->get_version()); }
    bool MetaFetch(const std::string &key,std::string *value) override { return Fetch(std::string(1,'\x01')+key,value); }
    bool MetaUpdate(const std::string &key,const std::string &value) override { return Update(std::string(1,'\x01')+key,value); }
    rime::an<rime::DbAccessor> QueryMetadata() override { return query(std::string(1,'\x01'),true); }
    rime::an<rime::DbAccessor> QueryAll() override { return query("",false); }
    rime::an<rime::DbAccessor> Query(const std::string &prefix) override { return query(prefix,false); }
    bool Fetch(const std::string &key,std::string *value) override {bool found=db_.fetch(key,value);if (db_.error().code) report(name_,db_.error());return found;}
    bool Update(const std::string &key,const std::string &value) override { return !disabled_ && result(db_.update(key,value)); }
    bool Erase(const std::string &key) override { return !disabled_ && result(db_.erase(key)); }
    bool BeginTransaction() override { return !disabled_ && result(db_.begin()); }
    bool AbortTransaction() override { return result(db_.abort()); }
    bool CommitTransaction() override { return !disabled_ && result(db_.commit()); }
};
class Component final : public rime::UserDb::Component {
    std::shared_ptr<LearningProvider> provider_;
public:
    explicit Component(std::shared_ptr<LearningProvider> provider):provider_(std::move(provider)) {}
    std::string extension() const override { return ".arion-userdb"; }
    rime::Db *Create(const std::string &name) override {
        if (!validLearningName(name)) { report(name,{9,"database name"}); return nullptr; }
        return new Database(provider_,learningPath(provider_->directory(),name+extension()),name);
    }
};
}
bool registerEncryptedLearning(std::shared_ptr<LearningProvider> provider,LearningError &error) {
    if (!validateLearningVault(provider,error)) { report("",error); return false; }
    rime::Registry::instance().Register("arion_encrypted_userdb",new Component(provider));
    std::lock_guard<std::mutex> lock(globalMutex); registeredProvider=std::move(provider); globalError={}; return true;
}
LearningError encryptedLearningError() { std::lock_guard<std::mutex> lock(globalMutex); return globalError; }
void clearEncryptedLearningError() { std::lock_guard<std::mutex> lock(globalMutex); globalError={}; }
std::shared_ptr<LearningProvider> registeredLearningProvider() { std::lock_guard<std::mutex> lock(globalMutex); return registeredProvider; }
rime::Db *makeEncryptedRimeDatabase(std::shared_ptr<LearningProvider> provider,const std::string &file,const std::string &name) { return new Database(std::move(provider),file,name); }
std::unique_ptr<LearningDatabase> openContextLearning(LearningError &error) {
    auto provider=registeredLearningProvider(); if (!provider) { error={1,"provider missing"}; return nullptr; }
    auto db=std::make_unique<LearningDatabase>(provider,learningPath(provider->directory(),"arion_context.arion-userdb"),"arion_context",reconcileContextLearning);
    if (!db->open()) { error=db->error(); return nullptr; }
    if (db->wasMissing() && (!db->begin() || !db->update("\x01/db_name","arion_context") || !db->update("\x01/db_type","userdb") ||
        !db->update("\x01/rime_version",rime_get_api()->get_version()) || !db->commit())) { error=db->error(); return nullptr; }
    return db;
}
}

namespace {
int outputError(const arion::ime::LearningError &error,char *output,size_t capacity) {
    if (output && capacity) { std::string message=error.code?"encrypted learning: "+error.operation:std::string{}; size_t count=std::min(message.size(),capacity-1); std::memcpy(output,message.data(),count); output[count]='\0'; }
    return error.code;
}
}
extern "C" int arion_learning_error(char *error,size_t capacity) { return outputError(arion::ime::encryptedLearningError(),error,capacity); }
#ifdef _WIN32
extern "C" int arion_learning_windows_register(const char *directory,char *error,size_t capacity) {
    arion::ime::LearningError failure;
    auto provider=arion::ime::makeWindowsLearningProvider(directory?directory:"",failure);
    bool ok=provider && arion::ime::registerEncryptedLearning(provider,failure); outputError(failure,error,capacity); return ok;
}
extern "C" int arion_learning_windows_reset(const char *directory,char *error,size_t capacity) {
    arion::ime::LearningError failure;
    auto provider=arion::ime::makeWindowsLearningProvider(directory?directory:"",failure);
    bool ok=provider && arion::ime::resetLearningVault(provider,failure); if (ok) arion::ime::clearEncryptedLearningError(); outputError(failure,error,capacity); return ok;
}
#endif
