#include "learning.h"
#include <algorithm>
#include <cstring>
#include <filesystem>
#include <limits>
#include <charconv>

namespace arion::ime {
namespace {
bool failure(LearningError &error, int code, const char *operation) { error = {code,operation}; return false; }
void integer(LearningBytes &out, uint64_t value, size_t size) { for (size_t i=size;i;--i) out.push_back(static_cast<uint8_t>(value >> ((i-1)*8))); }
size_t width(uint64_t value) { return value <= 0xff ? 1 : value <= 0xffff ? 2 : value <= 0xffffffff ? 4 : 8; }
void marker(LearningBytes &out, uint8_t type, size_t length) {
    if (length < 15) out.push_back(type | static_cast<uint8_t>(length));
    else { out.push_back(type | 15); size_t bytes = width(length); out.push_back(static_cast<uint8_t>(0x10 | (bytes == 1 ? 0 : bytes == 2 ? 1 : bytes == 4 ? 2 : 3))); integer(out,length,bytes); }
}
class PlistReader {
    const LearningBytes &bytes;
    uint64_t objectCount = 0, table = 0, root = 0;
    size_t offsetSize = 0, referenceSize = 0;
    bool number(size_t offset, size_t size, uint64_t &value) const {
        if (!size || size > 8 || offset > bytes.size() || size > bytes.size()-offset) return false;
        value = 0; for (size_t i=0;i<size;++i) value = (value << 8) | bytes[offset+i]; return true;
    }
    bool object(uint64_t id, uint8_t kind, size_t &position, size_t &length) const {
        uint64_t offset = 0;
        if (id >= objectCount || !number(static_cast<size_t>(table+id*offsetSize),offsetSize,offset) || offset < 8 || offset >= table) return false;
        position = static_cast<size_t>(offset); uint8_t tag = bytes[position++]; if ((tag & 0xf0) != kind) return false;
        uint64_t count = tag & 15;
        if (count == 15) {
            if (position >= table || (bytes[position] & 0xf0) != 0x10 || (bytes[position] & 15) > 3) return false;
            size_t size = size_t{1} << (bytes[position++] & 15);
            if (!number(position,size,count)) return false;
            position += size;
        }
        if (count > maximumLearningSnapshot || position > table) return false;
        length = static_cast<size_t>(count); return true;
    }
    bool array(uint64_t id, std::vector<uint64_t> &values, size_t maximum) const {
        size_t offset = 0, length = 0;
        if (!object(id,0xa0,offset,length) || length > maximum || length > (table-offset)/referenceSize) return false;
        values.resize(length);
        for (size_t i=0;i<length;++i) if (!number(offset+i*referenceSize,referenceSize,values[i]) || values[i] >= objectCount) return false;
        return true;
    }
    bool data(uint64_t id, std::string &value) const {
        size_t offset = 0, length = 0;
        if (!object(id,0x40,offset,length) || length > maximumLearningField || length > table-offset) return false;
        value.assign(reinterpret_cast<const char*>(bytes.data()+offset),length); return true;
    }
public:
    explicit PlistReader(const LearningBytes &source):bytes(source) {}
    bool read(LearningRecords &result) {
        if (bytes.size() < 40 || bytes.size() > maximumLearningSnapshot || std::memcmp(bytes.data(),"bplist00",8)) return false;
        size_t trailer = bytes.size()-32; offsetSize = bytes[trailer+6]; referenceSize = bytes[trailer+7];
        if (!offsetSize || offsetSize > 8 || !referenceSize || referenceSize > 8 || !number(trailer+8,8,objectCount) ||
            !number(trailer+16,8,root) || !number(trailer+24,8,table) || !objectCount || objectCount > 1+maximumLearningEntries*3 ||
            root >= objectCount || table < 8 || table > trailer || objectCount > (trailer-table)/offsetSize) return false;
        std::vector<uint64_t> rows;
        if (!array(root,rows,maximumLearningEntries)) return false;
        LearningRecords parsed;
        for (uint64_t row : rows) {
            std::vector<uint64_t> pair; std::string key,value;
            if (!array(row,pair,2) || pair.size()!=2 || !data(pair[0],key) || !data(pair[1],value) || !parsed.emplace(std::move(key),std::move(value)).second) return false;
        }
        result = std::move(parsed); return true;
    }
};
}

bool encodeLearningRecords(const LearningRecords &records, LearningBytes &result, LearningError &error) {
    if (records.size() > maximumLearningEntries) return failure(error,3,"snapshot entries");
    uint64_t count = 1+records.size()*3; size_t refs = width(count-1);
    LearningBytes bytes{'b','p','l','i','s','t','0','0'}; std::vector<uint64_t> offsets;
    offsets.reserve(static_cast<size_t>(count)); offsets.push_back(bytes.size()); marker(bytes,0xa0,records.size());
    for (size_t i=0;i<records.size();++i) integer(bytes,1+i*3,refs);
    for (const auto &[key,value] : records) {
        if (key.size() > maximumLearningField || value.size() > maximumLearningField || bytes.size() > maximumLearningSnapshot-key.size() || bytes.size()+key.size() > maximumLearningSnapshot-value.size()) return failure(error,3,"snapshot fields");
        uint64_t index = offsets.size(); offsets.push_back(bytes.size()); marker(bytes,0xa0,2); integer(bytes,index+1,refs); integer(bytes,index+2,refs);
        offsets.push_back(bytes.size()); marker(bytes,0x40,key.size()); bytes.insert(bytes.end(),key.begin(),key.end());
        offsets.push_back(bytes.size()); marker(bytes,0x40,value.size()); bytes.insert(bytes.end(),value.begin(),value.end());
    }
    uint64_t table = bytes.size(); size_t offsetsWidth = width(table);
    if (table > maximumLearningSnapshot || count > (maximumLearningSnapshot-table)/offsetsWidth || table+count*offsetsWidth > maximumLearningSnapshot-32) return failure(error,3,"snapshot size");
    for (uint64_t offset : offsets) integer(bytes,offset,offsetsWidth);
    bytes.insert(bytes.end(),6,0); bytes.push_back(static_cast<uint8_t>(offsetsWidth)); bytes.push_back(static_cast<uint8_t>(refs));
    integer(bytes,count,8); integer(bytes,0,8); integer(bytes,table,8); result = std::move(bytes); error = {}; return true;
}
bool decodeLearningRecords(const LearningBytes &bytes, LearningRecords &records, LearningError &error) {
    if (!PlistReader(bytes).read(records)) return failure(error,4,"snapshot format");
    error = {}; return true;
}
bool validLearningName(const std::string &name) { return !name.empty() && name!="." && name!=".." && name.find_first_not_of("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789_.-")==std::string::npos; }
std::string learningPath(const std::string &directory, const std::string &name) { return (std::filesystem::u8path(directory)/std::filesystem::u8path(name)).u8string(); }
bool validateLearningVault(const std::shared_ptr<LearningProvider> &provider, LearningError &error) {
    if (!provider) return failure(error,1,"provider missing");
    std::unique_ptr<LearningLock> held; if (!provider->lock("",true,false,held,error)) return false;
    std::vector<std::string> files; if (!provider->files(files,error)) return false;
    for (const auto &file : files) {
        constexpr const char *suffix = ".arion-userdb";
        if (file.size() < std::strlen(suffix) || file.compare(file.size()-std::strlen(suffix),std::strlen(suffix),suffix)) continue;
        std::string name = file.substr(0,file.size()-std::strlen(suffix)); if (!validLearningName(name)) return failure(error,9,"database name");
        std::string path=learningPath(provider->directory(),file);
        std::unique_ptr<LearningLock> snapshotLock;
        if (!provider->lock(path,true,false,snapshotLock,error)) return false;
        LearningBytes sealed,plain; LearningRecords records; bool missing = false;
        if (!provider->read(path,sealed,missing,error)) return false;
        if (missing) return failure(error,5,"snapshot missing");
        if (!provider->unseal(name,sealed,plain,error) || !decodeLearningRecords(plain,records,error)) return false;
    }
    error = {}; return true;
}
bool resetLearningVault(const std::shared_ptr<LearningProvider> &provider, LearningError &error) {
    if (!provider) return failure(error,1,"provider missing");
    std::unique_ptr<LearningLock> held; if (!provider->lock("",false,true,held,error)) return false;
    std::vector<std::string> files; if (!provider->files(files,error)) return false;
    for (const auto &file : files) if ((file.size() >= 13 && file.compare(file.size()-13,13,".arion-userdb")==0) || file.rfind(".encrypted-",0)==0) {
        if (!provider->remove(learningPath(provider->directory(),file),error)) return false;
    }
    error = {}; return true;
}
bool reconcileLearningChanges(const LearningRecords &base,const LearningChanges &changes,const LearningRecords &latest,LearningRecords &merged,LearningError &error) {
    merged=latest;
    for (const auto &[key,value]:changes) {
        auto previous=base.find(key),found=latest.find(key);
        std::optional<std::string> old=previous==base.end()?std::nullopt:std::make_optional(previous->second);
        std::optional<std::string> current=found==latest.end()?std::nullopt:std::make_optional(found->second);
        if (old!=current && value!=current) return failure(error,10,"concurrent record conflict");
        if (value) merged[key]=*value;else merged.erase(key);
    }
    return true;
}
bool reconcileLearningClock(const LearningRecords &base,const LearningChanges &changes,const LearningRecords &latest,const std::string &key,LearningClock &clock,LearningError &error) {
    auto number=[](const std::string &text,uint64_t &value) {auto parsed=std::from_chars(text.data(),text.data()+text.size(),value);return parsed.ec==std::errc{} && parsed.ptr==text.data()+text.size();};
    auto old=base.find(key),current=latest.find(key);auto next=changes.find(key);
    if ((old!=base.end() && !number(old->second,clock.localBase)) || (current!=latest.end() && !number(current->second,clock.globalBase))) return failure(error,10,"invalid learning clock");
    clock.localEnd=clock.localBase;
    if (next!=changes.end() && (!next->second || !number(*next->second,clock.localEnd))) return failure(error,10,"invalid learning clock change");
    if (clock.localEnd<clock.localBase || clock.localEnd-clock.localBase>std::numeric_limits<uint64_t>::max()-clock.globalBase) return failure(error,10,"learning clock overflow");
    clock.globalEnd=clock.globalBase+(clock.localEnd-clock.localBase);return true;
}

LearningDatabase::LearningDatabase(std::shared_ptr<LearningProvider> provider, std::string file, std::string name, LearningReconciler reconcile):provider_(std::move(provider)),file_(std::move(file)),name_(std::move(name)),reconcile_(reconcile) {}
LearningDatabase::~LearningDatabase() { close(); }
bool LearningDatabase::fail(int code, const char *operation) { return failure(error_,code,operation); }
bool LearningDatabase::decode(const LearningBytes &sealed, LearningRecords &records) { LearningBytes plain; return provider_->unseal(name_,sealed,plain,error_) && decodeLearningRecords(plain,records,error_); }
bool LearningDatabase::open(bool readonly) {
    std::lock_guard<std::recursive_mutex> lock(mutex_);
    if (loaded_ || !provider_) return fail(1,"database state");
    std::unique_ptr<LearningLock> snapshot;
    if (provider_->cooperative()) {
        if (!provider_->lease(file_,false,held_,error_) || !provider_->lock(file_,true,false,snapshot,error_)) {held_.reset();return false;}
    } else if (!provider_->lock(file_,readonly,false,held_,error_)) return false;
    LearningBytes bytes; bool missing = false; LearningRecords restored;
    if (!provider_->read(file_,bytes,missing,error_) || (missing && readonly) || (!missing && !decode(bytes,restored))) { held_.reset(); if (!error_.code) fail(5,"snapshot missing"); return false; }
    if (provider_->cooperative() && !provider_->revision(file_,revision_,error_)) {held_.reset();return false;}
    records_ = std::move(restored); observed_=records_; loaded_ = true; readonly_ = readonly; missing_ = missing; error_ = {}; return true;
}
bool LearningDatabase::close() {
    std::lock_guard<std::recursive_mutex> lock(mutex_); if (!loaded_) return false;
    bool ok = !transaction_ || commit(); loaded_ = readonly_ = transaction_ = false; records_.clear(); observed_.clear(); pendingBase_.clear(); pending_.clear(); revision_.clear(); held_.reset(); return ok;
}
bool LearningDatabase::persist(const LearningRecords &records) {
    LearningBytes plain,sealed;
    if (!encodeLearningRecords(records,plain,error_) || !provider_->seal(name_,plain,sealed,error_) || !provider_->writeAtomic(file_,sealed,error_)) { disabled_ = true; return false; }
    records_ = records;
    if (provider_->cooperative() && !provider_->revision(file_,revision_,error_)) {disabled_=true;return false;}
    error_ = {}; return true;
}
bool LearningDatabase::refreshLocked() {
    std::string revision;
    if (!provider_->revision(file_,revision,error_)) {disabled_=true;return false;}
    if (!revision.empty() && revision==revision_) return true;
    LearningBytes bytes;bool missing=false;LearningRecords latest;
    if (!provider_->read(file_,bytes,missing,error_) || (!missing && !decode(bytes,latest))) {disabled_=true;return false;}
    if (missing && revision_!="missing") {disabled_=true;return fail(5,"snapshot disappeared");}
    records_=std::move(latest);revision_=std::move(revision);return true;
}
bool LearningDatabase::refresh() {
    if (!provider_->cooperative()) return true;
    std::unique_ptr<LearningLock> held;
    return provider_->lock(file_,true,false,held,error_) && refreshLocked();
}
bool LearningDatabase::apply(const LearningChanges &changes,const LearningRecords &base) {
    LearningRecords next=records_;std::unique_ptr<LearningLock> held;
    if (provider_->cooperative()) {
        if (!provider_->lock(file_,false,false,held,error_) || !refreshLocked()) return false;
        next=records_;
        if (reconcile_) {if (!reconcile_(base,changes,records_,next,error_)) return false;}
        else if (!reconcileLearningChanges(base,changes,records_,next,error_)) return false;
    } else for (const auto &[key,value]:changes) {if (value) next[key]=*value;else next.erase(key);}
    if (!persist(next)) return false;
    for (const auto &[key,value]:changes) {if (value) observed_[key]=*value;else observed_.erase(key);}
    return true;
}
bool LearningDatabase::remove() {
    std::lock_guard<std::recursive_mutex> lock(mutex_);if (loaded_ || !provider_) return fail(7,"database state");
    std::unique_ptr<LearningLock> lease,held;
    if (provider_->cooperative() && !provider_->lease(file_,true,lease,error_)) return false;
    return provider_->lock(file_,false,false,held,error_) && provider_->remove(file_,error_);
}
bool LearningDatabase::backup(const std::string &destination) {
    std::lock_guard<std::recursive_mutex> lock(mutex_); if (!loaded_ || disabled_) return fail(7,"database state");
    if (destination.size() >= 5 && destination.compare(destination.size()-5,5,".lock")==0) return fail(9,"backup path");
    if (!refresh()) return false;
    LearningBytes plain,sealed; return encodeLearningRecords(records_,plain,error_) && provider_->seal(name_,plain,sealed,error_) && provider_->writeAtomic(destination,sealed,error_);
}
bool LearningDatabase::restore(const std::string &source) {
    std::lock_guard<std::recursive_mutex> lock(mutex_); if (!loaded_ || readonly_ || disabled_ || transaction_) return fail(7,"database state");
    LearningBytes sealed; bool missing = false; LearningRecords restored;
    std::unique_ptr<LearningLock> held;
    if (provider_->cooperative() && !provider_->lock(file_,false,false,held,error_)) return false;
    if (!provider_->read(source,sealed,missing,error_)) return false;
    if (missing) return fail(5,"snapshot missing");
    if (!decode(sealed,restored) || !persist(restored)) return false;
    observed_=records_;return true;
}
bool LearningDatabase::fetch(const std::string &key, std::string *value) {
    std::lock_guard<std::recursive_mutex> lock(mutex_);if (!loaded_ || !value || !refresh()) return false;
    auto found=records_.find(key);
    if (!transaction_ || !pending_.count(key)) {
        if (found==records_.end()) observed_.erase(key);else observed_[key]=found->second;
        if (transaction_) {if (found==records_.end()) pendingBase_.erase(key);else pendingBase_[key]=found->second;}
    }
    if (found==records_.end()) return false;
    *value=found->second;return true;
}
bool LearningDatabase::update(const std::string &key, const std::string &value) {
    std::lock_guard<std::recursive_mutex> lock(mutex_); if (!loaded_ || readonly_ || disabled_ || key.size()>maximumLearningField || value.size()>maximumLearningField) return fail(7,"database state");
    if (transaction_) { pending_[key] = value; return true; } return apply({{key,value}},observed_);
}
bool LearningDatabase::erase(const std::string &key) { std::lock_guard<std::recursive_mutex> lock(mutex_); if (!loaded_ || readonly_ || disabled_) return fail(7,"database state"); if (transaction_) { pending_[key] = std::nullopt; return true; } return apply({{key,std::nullopt}},observed_); }
bool LearningDatabase::begin() { std::lock_guard<std::recursive_mutex> lock(mutex_); if (!loaded_ || readonly_ || disabled_ || transaction_) return fail(7,"database state");if (!refresh()) return false;pending_.clear();pendingBase_=observed_; transaction_ = true; return true; }
bool LearningDatabase::abort() { std::lock_guard<std::recursive_mutex> lock(mutex_); if (!loaded_ || !transaction_) return false; pending_.clear();pendingBase_.clear(); transaction_ = false; return true; }
bool LearningDatabase::commit() {
    std::lock_guard<std::recursive_mutex> lock(mutex_); if (!loaded_ || readonly_ || disabled_ || !transaction_) return false;
    if (!apply(pending_,pendingBase_)) return false;
    pending_.clear();pendingBase_.clear(); transaction_ = false; return true;
}
LearningRecords LearningDatabase::records(bool observe) {std::lock_guard<std::recursive_mutex> lock(mutex_);if (!loaded_ || !refresh()) return {};if (observe) observed_=records_;return records_;}
} // namespace arion::ime
