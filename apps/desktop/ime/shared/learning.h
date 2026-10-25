#pragma once
#include <cstdint>
#include <map>
#include <memory>
#include <mutex>
#include <optional>
#include <string>
#include <vector>

namespace arion::ime {
using LearningBytes = std::vector<uint8_t>;
using LearningRecords = std::map<std::string, std::string>;
using LearningChanges = std::map<std::string, std::optional<std::string>>;
struct LearningError { int code = 0; std::string operation; };
using LearningReconciler = bool (*)(const LearningRecords &base, const LearningChanges &changes,
                                    const LearningRecords &latest, LearningRecords &merged, LearningError &);
constexpr size_t maximumLearningSnapshot = 32 * 1024 * 1024;
constexpr size_t maximumLearningEntries = 100000;
constexpr size_t maximumLearningField = 1024 * 1024;

class LearningLock { public: virtual ~LearningLock() = default; };
// Providers own OS cryptography, secret material and file primitives only.
// Neither a provider nor this core may substitute plaintext after failure.
class LearningProvider {
public:
    virtual ~LearningProvider() = default;
    virtual const std::string &directory() const = 0;
    virtual bool cooperative() const { return false; }
    virtual bool lease(const std::string &file, bool removing, std::unique_ptr<LearningLock> &held, LearningError &error) {
        return lock(file,!removing,false,held,error);
    }
    virtual bool revision(const std::string &, std::string &value, LearningError &) { value.clear(); return true; }
    virtual bool lock(const std::string &file, bool readonly, bool reset,
                      std::unique_ptr<LearningLock> &held, LearningError &error) = 0;
    virtual bool read(const std::string &file, LearningBytes &bytes, bool &missing, LearningError &error) = 0;
    virtual bool writeAtomic(const std::string &file, const LearningBytes &bytes, LearningError &error) = 0;
    virtual bool remove(const std::string &file, LearningError &error) = 0;
    virtual bool files(std::vector<std::string> &names, LearningError &error) = 0;
    virtual bool seal(const std::string &name, const LearningBytes &plain, LearningBytes &sealed, LearningError &error) = 0;
    virtual bool unseal(const std::string &name, const LearningBytes &sealed, LearningBytes &plain, LearningError &error) = 0;
};

// The binary-plist subset is shared across OSes and reads the original macOS
// NSData-pair array format, preserving existing encrypted snapshot contents.
bool encodeLearningRecords(const LearningRecords &, LearningBytes &, LearningError &);
bool decodeLearningRecords(const LearningBytes &, LearningRecords &, LearningError &);
bool validLearningName(const std::string &);
std::string learningPath(const std::string &directory, const std::string &name);
bool validateLearningVault(const std::shared_ptr<LearningProvider> &, LearningError &);
bool resetLearningVault(const std::shared_ptr<LearningProvider> &, LearningError &);
bool reconcileLearningChanges(const LearningRecords &base,const LearningChanges &,const LearningRecords &latest,LearningRecords &merged,LearningError &);
struct LearningClock { uint64_t localBase=0,localEnd=0,globalBase=0,globalEnd=0; };
bool reconcileLearningClock(const LearningRecords &base,const LearningChanges &,const LearningRecords &latest,
                            const std::string &key,LearningClock &,LearningError &);

class LearningDatabase final {
public:
    LearningDatabase(std::shared_ptr<LearningProvider>, std::string file, std::string name, LearningReconciler = nullptr);
    ~LearningDatabase();
    bool open(bool readonly = false);
    bool close();
    bool remove();
    bool backup(const std::string &destination);
    bool restore(const std::string &source);
    bool fetch(const std::string &key, std::string *value);
    bool update(const std::string &key, const std::string &value);
    bool erase(const std::string &key);
    bool begin();
    bool abort();
    bool commit();
    LearningRecords records(bool observe = true);
    const LearningError &error() const { return error_; }
    bool loaded() const { return loaded_; }
    bool readonly() const { return readonly_; }
    bool disabled() const { return disabled_; }
    bool inTransaction() const { return transaction_; }
    bool wasMissing() const { return missing_; }
private:
    bool fail(int code, const char *operation);
    bool persist(const LearningRecords &);
    bool decode(const LearningBytes &, LearningRecords &);
    bool refresh();
    bool refreshLocked();
    bool apply(const LearningChanges &, const LearningRecords &base);
    std::shared_ptr<LearningProvider> provider_;
    std::string file_, name_;
    mutable std::recursive_mutex mutex_;
    std::unique_ptr<LearningLock> held_;
    LearningRecords records_;
    LearningRecords observed_, pendingBase_;
    LearningChanges pending_;
    LearningReconciler reconcile_ = nullptr;
    std::string revision_;
    LearningError error_;
    bool loaded_ = false, readonly_ = false, disabled_ = false, transaction_ = false, missing_ = false;
};

#ifdef _WIN32
std::shared_ptr<LearningProvider> makeWindowsLearningProvider(const std::string &directory, LearningError &error);
#endif
} // namespace arion::ime
