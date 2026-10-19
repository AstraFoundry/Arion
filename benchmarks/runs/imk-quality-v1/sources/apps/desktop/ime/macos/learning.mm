#import "learning.h"
#import <Security/Security.h>
#include <rime/registry.h>
#include <rime/dict/db.h>
#include <rime/dict/user_db.h>
#include <algorithm>
#include <cerrno>
#include <cstring>
#include <fcntl.h>
#include <mutex>
#include <optional>
#include <sys/file.h>
#include <sys/stat.h>
#include <unistd.h>

extern "C" int32_t arion_learning_seal(const void *, size_t, const void *, size_t,
                                       const void *, size_t, void **, size_t *);
extern "C" int32_t arion_learning_open(const void *, size_t, const void *, size_t,
                                       const void *, size_t, void **, size_t *);

namespace {
constexpr size_t maximumSnapshot = 32 * 1024 * 1024;
constexpr size_t maximumEntries = 100000;
constexpr char magic[] = "ARIONUD1";
constexpr char metadataPrefix = '\x01';
using Records = std::map<std::string, std::string>;
std::mutex errorMutex;
NSError *lastError;

bool fail(NSInteger code) {
    std::lock_guard<std::mutex> lock(errorMutex);
    lastError = [NSError errorWithDomain:@"si.astrafoundry.arion.keyboard.learning"
                                   code:code userInfo:nil];
    return false;
}

NSData *associatedData(const std::string &name) {
    std::string label = "si.astrafoundry.arion.keyboard.userdb.v1/" + name;
    return [NSData dataWithBytes:label.data() length:label.size()];
}
NSData *crypt(NSData *data, NSData *key, const std::string &name, bool encrypt) {
    if (key.length != 32) { fail(1); return nil; }
    NSData *aad = associatedData(name);
    void *bytes = nullptr;
    size_t count = 0;
    auto operation = encrypt ? arion_learning_seal : arion_learning_open;
    if (!operation(data.bytes, data.length, key.bytes, key.length,
                   aad.bytes, aad.length, &bytes, &count)) { fail(2); return nil; }
    return [NSData dataWithBytesNoCopy:bytes length:count freeWhenDone:YES];
}

NSData *encode(const Records &records, NSData *key, const std::string &name) {
    if (records.size() > maximumEntries) { fail(3); return nil; }
    NSMutableArray *rows = [NSMutableArray arrayWithCapacity:records.size()];
    for (const auto &[word, value] : records) {
        [rows addObject:@[[NSData dataWithBytes:word.data() length:word.size()],
                         [NSData dataWithBytes:value.data() length:value.size()]]];
    }
    NSData *plain = [NSPropertyListSerialization dataWithPropertyList:rows
        format:NSPropertyListBinaryFormat_v1_0 options:0 error:nil];
    if (!plain || plain.length > maximumSnapshot) { fail(3); return nil; }
    NSData *sealed = crypt(plain, key, name, true);
    if (!sealed) return nil;
    NSMutableData *envelope = [NSMutableData dataWithBytes:magic length:sizeof(magic) - 1];
    [envelope appendData:sealed];
    return envelope;
}

bool decode(NSData *envelope, NSData *key, const std::string &name, Records *records) {
    const size_t header = sizeof(magic) - 1;
    if (envelope.length < header + 28 || envelope.length > maximumSnapshot + header + 28 ||
        memcmp(envelope.bytes, magic, header)) return fail(4);
    NSData *sealed = [envelope subdataWithRange:NSMakeRange(header, envelope.length - header)];
    NSData *plain = crypt(sealed, key, name, false);
    if (!plain) return false;
    id rows = [NSPropertyListSerialization propertyListWithData:plain options:NSPropertyListImmutable
                                                       format:nil error:nil];
    if (![rows isKindOfClass:NSArray.class] || [rows count] > maximumEntries) return fail(4);
    Records parsed;
    for (id row in rows) {
        if (![row isKindOfClass:NSArray.class] || [row count] != 2 ||
            ![row[0] isKindOfClass:NSData.class] || ![row[1] isKindOfClass:NSData.class]) return fail(4);
        NSData *word = row[0], *value = row[1];
        if (word.length > 1024 * 1024 || value.length > 1024 * 1024) return fail(4);
        std::string wordString(static_cast<const char *>(word.bytes), word.length);
        std::string valueString(static_cast<const char *>(value.bytes), value.length);
        if (!parsed.emplace(std::move(wordString), std::move(valueString)).second) return fail(4);
    }
    *records = std::move(parsed);
    return true;
}

bool readFile(NSString *file, NSData **result, bool *missing) {
    *missing = false;
    int fd = open(file.fileSystemRepresentation, O_RDONLY | O_NOFOLLOW);
    if (fd < 0) { if (errno == ENOENT) { *missing = true; return true; } return fail(5); }
    struct stat info;
    if (fstat(fd, &info) || !S_ISREG(info.st_mode) || info.st_size < 0 ||
        static_cast<uint64_t>(info.st_size) > maximumSnapshot + 64) { close(fd); return fail(5); }
    NSMutableData *data = [NSMutableData dataWithLength:static_cast<NSUInteger>(info.st_size)];
    size_t offset = 0;
    while (offset < data.length) {
        ssize_t count = read(fd, static_cast<char *>(data.mutableBytes) + offset, data.length - offset);
        if (count < 0 && errno == EINTR) continue;
        if (count <= 0) { close(fd); return fail(5); }
        offset += count;
    }
    if (close(fd)) return fail(5);
    *result = data;
    return true;
}

bool writeFile(NSString *file, NSData *data) {
    NSString *directory = file.stringByDeletingLastPathComponent;
    NSString *temporary = [directory stringByAppendingPathComponent:
        [@".encrypted-" stringByAppendingString:NSUUID.UUID.UUIDString]];
    int fd = open(temporary.fileSystemRepresentation, O_WRONLY | O_CREAT | O_EXCL | O_NOFOLLOW, 0600);
    if (fd < 0) return fail(6);
    size_t offset = 0;
    bool ok = true;
    while (offset < data.length) {
        ssize_t count = write(fd, static_cast<const char *>(data.bytes) + offset, data.length - offset);
        if (count < 0 && errno == EINTR) continue;
        if (count <= 0) { ok = false; break; }
        offset += count;
    }
    if (ok && fsync(fd)) ok = false;
    if (close(fd)) ok = false;
    if (ok && rename(temporary.fileSystemRepresentation, file.fileSystemRepresentation)) ok = false;
    if (!ok) { unlink(temporary.fileSystemRepresentation); return fail(6); }
    int directoryFD = open(directory.fileSystemRepresentation, O_RDONLY | O_DIRECTORY);
    if (directoryFD < 0) return fail(6);
    ok = fsync(directoryFD) == 0;
    if (close(directoryFD)) ok = false;
    return ok || fail(6);
}

class Cursor final : public rime::DbAccessor {
public:
    Cursor(const Records &records, const std::string &prefix, bool metadata)
        : DbAccessor(prefix), records_(records), metadata_(metadata) { Reset(); }
    bool Reset() override { current_ = records_.lower_bound(prefix_); return true; }
    bool Jump(const std::string &key) override { current_ = records_.lower_bound(key); return true; }
    bool exhausted() override {
        return current_ == records_.end() || current_->first.compare(0, prefix_.size(), prefix_) != 0;
    }
    bool GetNextRecord(std::string *key, std::string *value) override {
        if (!key || !value || exhausted()) return false;
        *key = current_->first;
        if (metadata_) key->erase(0, 1);
        *value = current_->second;
        ++current_;
        return true;
    }
private:
    Records records_;
    Records::const_iterator current_;
    bool metadata_;
};

class EncryptedDb final : public rime::Db, public rime::Transactional {
public:
    EncryptedDb(const rime::path &file, const std::string &name, NSData *key)
        : Db(file, name), key_(key), file_([NSString stringWithUTF8String:file.string().c_str()]) {}
    ~EncryptedDb() override { Close(); }
    bool Open() override { return openDatabase(false); }
    bool OpenReadOnly() override { return openDatabase(true); }
    bool Close() override {
        std::lock_guard<std::recursive_mutex> lock(mutex_);
        if (!loaded_) return false;
        // Rime's UserDictionary destructor normally commits first. Preserve a
        // pending successful selection when a standalone caller closes the DB.
        bool ok = !in_transaction_ || CommitTransaction();
        loaded_ = false; readonly_ = false; pending_.clear(); in_transaction_ = false;
        records_.clear(); unlockFile();
        return ok;
    }
    bool Remove() override {
        std::lock_guard<std::recursive_mutex> lock(mutex_);
        if (loaded_) return fail(7);
        NSString *lockPath = [file_ stringByAppendingString:@".lock"];
        int fd = open(lockPath.fileSystemRepresentation, O_RDWR | O_CREAT | O_NOFOLLOW, 0600);
        if (fd < 0) return fail(8);
        if (flock(fd, LOCK_EX | LOCK_NB)) { close(fd); return fail(8); }
        bool ok = unlink(file_.fileSystemRepresentation) == 0 || errno == ENOENT;
        flock(fd, LOCK_UN); close(fd);
        return ok || fail(5);
    }
    bool Backup(const rime::path &destination) override {
        std::lock_guard<std::recursive_mutex> lock(mutex_);
        if (!loaded_ || disabled_) return fail(7);
        NSString *target = [NSString stringWithUTF8String:destination.string().c_str()];
        if ([target hasSuffix:@".lock"]) return fail(9);
        NSData *encrypted = encode(records_, key_, name_);
        return encrypted && writeFile([NSString stringWithUTF8String:destination.string().c_str()], encrypted);
    }
    bool Restore(const rime::path &source) override {
        std::lock_guard<std::recursive_mutex> lock(mutex_);
        if (!loaded_ || readonly_ || disabled_ || in_transaction_) return fail(7);
        NSData *encrypted = nil; bool missing;
        if (!readFile([NSString stringWithUTF8String:source.string().c_str()], &encrypted, &missing) || missing) return fail(5);
        Records restored;
        return decode(encrypted, key_, name_, &restored) && persist(restored);
    }
    bool CreateMetadata() override {
        return MetaUpdate("/db_name", name_) && MetaUpdate("/db_type", "userdb") &&
               MetaUpdate("/rime_version", rime_get_api()->get_version());
    }
    bool MetaFetch(const std::string &key, std::string *value) override { return Fetch(std::string(1, metadataPrefix) + key, value); }
    bool MetaUpdate(const std::string &key, const std::string &value) override { return Update(std::string(1, metadataPrefix) + key, value); }
    rime::an<rime::DbAccessor> QueryMetadata() override { return query(std::string(1, metadataPrefix), true); }
    rime::an<rime::DbAccessor> QueryAll() override { return query("", false); }
    rime::an<rime::DbAccessor> Query(const std::string &prefix) override { return query(prefix, false); }
    bool Fetch(const std::string &key, std::string *value) override {
        std::lock_guard<std::recursive_mutex> lock(mutex_);
        if (!loaded_ || !value) return false;
        auto found = records_.find(key);
        if (found == records_.end()) return false;
        *value = found->second; return true;
    }
    bool Update(const std::string &key, const std::string &value) override {
        std::lock_guard<std::recursive_mutex> lock(mutex_);
        if (!loaded_ || readonly_ || disabled_ || key.size() > 1024 * 1024 || value.size() > 1024 * 1024) return fail(7);
        if (in_transaction_) { pending_[key] = value; return true; }
        Records next = records_; next[key] = value; return persist(next);
    }
    bool Erase(const std::string &key) override {
        std::lock_guard<std::recursive_mutex> lock(mutex_);
        if (!loaded_ || readonly_ || disabled_) return fail(7);
        if (in_transaction_) { pending_[key] = std::nullopt; return true; }
        Records next = records_; next.erase(key); return persist(next);
    }
    bool BeginTransaction() override {
        std::lock_guard<std::recursive_mutex> lock(mutex_);
        if (!loaded_ || readonly_ || disabled_ || in_transaction_) return fail(7);
        pending_.clear(); in_transaction_ = true; return true;
    }
    bool AbortTransaction() override {
        std::lock_guard<std::recursive_mutex> lock(mutex_);
        if (!loaded_ || !in_transaction_) return false;
        pending_.clear(); in_transaction_ = false; return true;
    }
    bool CommitTransaction() override {
        std::lock_guard<std::recursive_mutex> lock(mutex_);
        if (!loaded_ || readonly_ || disabled_ || !in_transaction_) return false;
        Records next = records_;
        for (const auto &[key, value] : pending_) { if (value) next[key] = *value; else next.erase(key); }
        if (!persist(next)) return false;
        pending_.clear(); in_transaction_ = false; return true;
    }
private:
    bool openDatabase(bool readonly) {
        std::lock_guard<std::recursive_mutex> lock(mutex_);
        if (loaded_ || key_.length != 32) return fail(1);
        NSString *directoryLock = [file_.stringByDeletingLastPathComponent stringByAppendingPathComponent:@".learning.lock"];
        directoryLockFD_ = open(directoryLock.fileSystemRepresentation, O_RDWR | O_CREAT | O_NOFOLLOW, 0600);
        if (directoryLockFD_ < 0 || flock(directoryLockFD_, LOCK_SH | LOCK_NB)) { unlockFile(); return fail(8); }
        NSString *lockPath = [file_ stringByAppendingString:@".lock"];
        lockFD_ = open(lockPath.fileSystemRepresentation, O_RDWR | O_CREAT | O_NOFOLLOW, 0600);
        if (lockFD_ < 0 || flock(lockFD_, (readonly ? LOCK_SH : LOCK_EX) | LOCK_NB)) { unlockFile(); return fail(8); }
        NSData *encrypted = nil; bool missing;
        if (!readFile(file_, &encrypted, &missing) || (missing && readonly)) { unlockFile(); return fail(5); }
        Records restored;
        if (!missing && !decode(encrypted, key_, name_, &restored)) { unlockFile(); return false; }
        records_ = std::move(restored); loaded_ = true; readonly_ = readonly;
        if (missing && !CreateMetadata()) { loaded_ = false; records_.clear(); unlockFile(); return false; }
        return true;
    }
    bool persist(const Records &records) {
        NSData *encrypted = encode(records, key_, name_);
        if (!encrypted || !writeFile(file_, encrypted)) { disabled_ = true; return false; }
        records_ = records; return true;
    }
    rime::an<rime::DbAccessor> query(const std::string &prefix, bool metadata) {
        std::lock_guard<std::recursive_mutex> lock(mutex_);
        if (!loaded_) return nullptr;
        return std::make_shared<Cursor>(records_, prefix, metadata);
    }
    void unlockFile() {
        if (lockFD_ >= 0) { flock(lockFD_, LOCK_UN); close(lockFD_); lockFD_ = -1; }
        if (directoryLockFD_ >= 0) { flock(directoryLockFD_, LOCK_UN); close(directoryLockFD_); directoryLockFD_ = -1; }
    }
    NSData *key_;
    NSString *file_;
    Records records_;
    std::map<std::string, std::optional<std::string>> pending_;
    std::recursive_mutex mutex_;
    int lockFD_ = -1;
    int directoryLockFD_ = -1;
};

class EncryptedComponent final : public rime::UserDb::Component {
public:
    EncryptedComponent(NSString *directory, NSData *key) : directory_(directory), key_(key) {}
    std::string extension() const override { return ".arion-userdb"; }
    rime::Db *Create(const std::string &name) override {
        if (name.empty() || name == "." || name == ".." || name.find_first_not_of(
            "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789_.-") != std::string::npos) {
            fail(9); return nullptr;
        }
        NSString *filename = [[NSString stringWithUTF8String:name.c_str()] stringByAppendingString:@".arion-userdb"];
        NSString *file = [directory_ stringByAppendingPathComponent:filename];
        return new EncryptedDb(rime::path(file.fileSystemRepresentation), name, key_);
    }
private:
    NSString *directory_;
    NSData *key_;
};

// The vault directory is reserved for learning data. Any non-lock entry can
// contain a snapshot, encrypted export or interrupted atomic write; none may
// silently authorize creating a replacement for a lost Keychain key.
bool hasVaultData(NSArray<NSString *> *files) {
    for (NSString *file in files) {
        if (![file isEqual:@".learning.lock"] && ![file hasSuffix:@".arion-userdb.lock"] &&
            ![file isEqual:@".DS_Store"]) return true;
    }
    return false;
}

NSData *keychainKey(bool mayCreate) {
    NSMutableDictionary *query = [@{(__bridge id)kSecClass: (__bridge id)kSecClassGenericPassword,
        (__bridge id)kSecAttrService: @"si.astrafoundry.arion.keyboard.learning",
        (__bridge id)kSecAttrAccount: @"aes256-gcm-v1", (__bridge id)kSecReturnData: @YES,
        (__bridge id)kSecMatchLimit: (__bridge id)kSecMatchLimitOne} mutableCopy];
    CFTypeRef result = nullptr;
    OSStatus status = SecItemCopyMatching((__bridge CFDictionaryRef)query, &result);
    if (status == errSecSuccess) {
        NSData *key = CFBridgingRelease(result);
        if (key.length == 32) return key;
        fail(1); return nil;
    }
    if (status != errSecItemNotFound || !mayCreate) { fail(1); return nil; }
    NSMutableData *key = [NSMutableData dataWithLength:32];
    if (SecRandomCopyBytes(kSecRandomDefault, key.length, key.mutableBytes) != errSecSuccess) { fail(1); return nil; }
    [query removeObjectForKey:(__bridge id)kSecReturnData];
    [query removeObjectForKey:(__bridge id)kSecMatchLimit];
    query[(__bridge id)kSecValueData] = key;
    query[(__bridge id)kSecAttrAccessible] = (__bridge id)kSecAttrAccessibleAfterFirstUnlockThisDeviceOnly;
    status = SecItemAdd((__bridge CFDictionaryRef)query, nullptr);
    if (status == errSecDuplicateItem) return keychainKey(false);
    if (status != errSecSuccess) { fail(1); return nil; }
    return key;
}
}  // namespace

NSError *ArionEncryptedLearningError(void) {
    std::lock_guard<std::mutex> lock(errorMutex);
    return lastError;
}

BOOL ArionRegisterEncryptedLearning(NSString *directory, NSError **error) {
    NSError *directoryError = nil;
    if (![NSFileManager.defaultManager createDirectoryAtPath:directory withIntermediateDirectories:YES
        attributes:@{NSFilePosixPermissions: @0700} error:&directoryError]) {
        fail(5); if (error) *error = ArionEncryptedLearningError(); return NO;
    }
    NSArray<NSString *> *files = [NSFileManager.defaultManager contentsOfDirectoryAtPath:directory error:&directoryError];
    if (!files) { fail(5); if (error) *error = ArionEncryptedLearningError(); return NO; }
    NSPredicate *encryptedFiles = [NSPredicate predicateWithBlock:^BOOL(NSString *name, NSDictionary *bindings) {
        return [name hasSuffix:@".arion-userdb"];
    }];
    NSArray<NSString *> *snapshots = [files filteredArrayUsingPredicate:encryptedFiles];
    NSData *key = keychainKey(!hasVaultData(files));
    if (!key) { if (error) *error = ArionEncryptedLearningError(); return NO; }
    for (NSString *filename in snapshots) {
        NSString *file = [directory stringByAppendingPathComponent:filename];
        NSData *encrypted = nil; bool missing; Records records;
        std::string name(filename.stringByDeletingPathExtension.UTF8String);
        if (!readFile(file, &encrypted, &missing) || missing || !decode(encrypted, key, name, &records)) {
            if (error) *error = ArionEncryptedLearningError(); return NO;
        }
    }
    rime::Registry::instance().Register("arion_encrypted_userdb", new EncryptedComponent(directory, key));
    { std::lock_guard<std::mutex> lock(errorMutex); lastError = nil; }
    return YES;
}

BOOL ArionResetEncryptedLearning(NSString *directory, NSError **error) {
    NSString *directoryLock = [directory stringByAppendingPathComponent:@".learning.lock"];
    int directoryFD = open(directoryLock.fileSystemRepresentation, O_RDWR | O_CREAT | O_NOFOLLOW, 0600);
    if (directoryFD < 0) { fail(8); if (error) *error = ArionEncryptedLearningError(); return NO; }
    struct DirectoryLock {
        int fd;
        ~DirectoryLock() { flock(fd, LOCK_UN); close(fd); }
    } held{directoryFD};
    if (flock(directoryFD, LOCK_EX | LOCK_NB)) { fail(8); if (error) *error = ArionEncryptedLearningError(); return NO; }
    NSArray<NSString *> *files = [NSFileManager.defaultManager contentsOfDirectoryAtPath:directory error:error];
    if (!files) return NO;
    NSMutableArray<NSString *> *snapshots = [NSMutableArray array];
    for (NSString *file in files) {
        if ([file hasSuffix:@".arion-userdb"]) [snapshots addObject:file];
    }
    [snapshots sortUsingSelector:@selector(compare:)];
    std::vector<int> locks;
    bool ok = true;
    for (NSString *snapshot in snapshots) {
        NSString *lockPath = [[directory stringByAppendingPathComponent:snapshot] stringByAppendingString:@".lock"];
        int fd = open(lockPath.fileSystemRepresentation, O_RDWR | O_CREAT | O_NOFOLLOW, 0600);
        if (fd < 0) { ok = false; break; }
        if (flock(fd, LOCK_EX | LOCK_NB)) { close(fd); ok = false; break; }
        locks.push_back(fd);
    }
    if (ok) {
        for (NSString *snapshot in snapshots) {
            NSString *file = [directory stringByAppendingPathComponent:snapshot];
            if (unlink(file.fileSystemRepresentation) && errno != ENOENT) { ok = false; break; }
        }
    }
    if (ok) {
        for (NSString *file in files) {
            if ([file hasPrefix:@".encrypted-"]) {
                NSString *temporary = [directory stringByAppendingPathComponent:file];
                if (unlink(temporary.fileSystemRepresentation) && errno != ENOENT) { ok = false; break; }
            }
        }
    }
    if (ok) {
        int fd = open(directory.fileSystemRepresentation, O_RDONLY | O_DIRECTORY);
        if (fd < 0) ok = false;
        else { ok = fsync(fd) == 0; if (close(fd)) ok = false; }
    }
    for (int fd : locks) { flock(fd, LOCK_UN); close(fd); }
    if (!ok) { fail(8); if (error) *error = ArionEncryptedLearningError(); return NO; }
    { std::lock_guard<std::mutex> lock(errorMutex); lastError = nil; }
    return YES;
}

#ifdef ARION_LEARNING_TESTING
BOOL ArionLearningTestMayCreateKey(NSArray<NSString *> *files) { return !hasVaultData(files); }
BOOL ArionRegisterTestEncryptedLearning(NSString *directory, NSData *key, NSError **error) {
    if (key.length != 32) { fail(1); if (error) *error = ArionEncryptedLearningError(); return NO; }
    if (![NSFileManager.defaultManager createDirectoryAtPath:directory withIntermediateDirectories:YES
        attributes:@{NSFilePosixPermissions: @0700} error:error]) return NO;
    rime::Registry::instance().Register("arion_encrypted_userdb", new EncryptedComponent(directory, key));
    { std::lock_guard<std::mutex> lock(errorMutex); lastError = nil; }
    return YES;
}
rime::Db *ArionLearningTestDatabase(NSString *file, NSString *name, NSData *key) {
    return new EncryptedDb(rime::path(file.fileSystemRepresentation), name.UTF8String, key);
}
#endif
