#import "learning.h"
#import <Security/Security.h>
#include "../shared/learning_rime.h"
#include <cerrno>
#include <cstring>
#include <fcntl.h>
#include <sys/file.h>
#include <sys/stat.h>
#include <unistd.h>

extern "C" int32_t arion_learning_seal(const void *, size_t, const void *, size_t,
                                       const void *, size_t, void **, size_t *);
extern "C" int32_t arion_learning_open(const void *, size_t, const void *, size_t,
                                       const void *, size_t, void **, size_t *);
using namespace arion::ime;
namespace {
std::mutex errorMutex;
NSError *lastError;
bool fail(NSInteger code) {
    std::lock_guard<std::mutex> lock(errorMutex);
    lastError=[NSError errorWithDomain:@"si.astrafoundry.arion.keyboard.learning" code:code userInfo:nil];
    return false;
}
bool failed(LearningError &error,int code,const char *operation) {error={code,operation};return false;}
NSString *pathOf(const std::string &path) {return [[NSString alloc] initWithBytes:path.data() length:path.size() encoding:NSUTF8StringEncoding];}
class FileLocks final : public LearningLock {
public:
    std::vector<int> files;
    ~FileLocks() override {for (int fd:files) {flock(fd,LOCK_UN);close(fd);}}
};
class MacProvider final : public LearningProvider {
    std::string directory_;
    NSData *key_;
    bool hold(const std::string &file,bool exclusive,FileLocks &held,LearningError &error) {
        int fd=open(pathOf(file).fileSystemRepresentation,O_RDWR|O_CREAT|O_NOFOLLOW,0600);
        struct stat info{};
        if (fd<0) return failed(error,8,"lock open");
        if (fstat(fd,&info) || !S_ISREG(info.st_mode) || flock(fd,(exclusive?LOCK_EX:LOCK_SH)|LOCK_NB)) {close(fd);return failed(error,8,"lock acquisition");}
        held.files.push_back(fd);return true;
    }
    bool crypt(const std::string &name,const LearningBytes &input,LearningBytes &output,bool encrypt,LearningError &error) {
        if (key_.length!=32) return failed(error,1,"key unavailable");
        std::string label="si.astrafoundry.arion.keyboard.userdb.v1/"+name;
        void *bytes=nullptr;size_t count=0;
        auto operation=encrypt?arion_learning_seal:arion_learning_open;
        if (!operation(input.data(),input.size(),key_.bytes,key_.length,label.data(),label.size(),&bytes,&count)) return failed(error,2,"AES-GCM authentication");
        const auto *first=static_cast<const uint8_t*>(bytes);output.assign(first,first+count);free(bytes);return true;
    }
public:
    MacProvider(NSString *directory,NSData *key):directory_(directory.UTF8String),key_(key) {}
    const std::string &directory() const override {return directory_;}
    bool lock(const std::string &file,bool readonly,bool reset,std::unique_ptr<LearningLock> &held,LearningError &error) override {
        auto locks=std::make_unique<FileLocks>();
        if (!hold(learningPath(directory_,".learning.lock"),reset,*locks,error) || (!file.empty() && !hold(file+".lock",!readonly,*locks,error))) return false;
        held=std::move(locks);return true;
    }
    bool read(const std::string &file,LearningBytes &bytes,bool &missing,LearningError &error) override {
        missing=false;int fd=open(pathOf(file).fileSystemRepresentation,O_RDONLY|O_NOFOLLOW);
        if (fd<0) {if (errno==ENOENT) {missing=true;bytes.clear();return true;}return failed(error,5,"snapshot open");}
        struct stat info{};
        if (fstat(fd,&info) || !S_ISREG(info.st_mode) || info.st_size<0 || static_cast<uint64_t>(info.st_size)>maximumLearningSnapshot+64) {close(fd);return failed(error,5,"snapshot size");}
        LearningBytes next(static_cast<size_t>(info.st_size));size_t offset=0;
        while (offset<next.size()) {ssize_t count=::read(fd,next.data()+offset,next.size()-offset);if (count<0 && errno==EINTR) continue;if (count<=0) {close(fd);return failed(error,5,"snapshot read");}offset+=count;}
        if (close(fd)) return failed(error,5,"snapshot close");
        bytes=std::move(next);return true;
    }
    bool writeAtomic(const std::string &file,const LearningBytes &bytes,LearningError &error) override {
        NSString *destination=pathOf(file),*directory=destination.stringByDeletingLastPathComponent;
        NSString *temporary=[directory stringByAppendingPathComponent:[@".encrypted-" stringByAppendingString:NSUUID.UUID.UUIDString]];
        int fd=open(temporary.fileSystemRepresentation,O_WRONLY|O_CREAT|O_EXCL|O_NOFOLLOW,0600);
        if (fd<0) return failed(error,6,"temporary create");
        size_t offset=0;bool ok=true;
        while (offset<bytes.size()) {ssize_t count=::write(fd,bytes.data()+offset,bytes.size()-offset);if (count<0 && errno==EINTR) continue;if (count<=0) {ok=false;break;}offset+=count;}
        if (ok && fsync(fd)) ok=false;
        if (close(fd)) ok=false;
        if (ok && rename(temporary.fileSystemRepresentation,destination.fileSystemRepresentation)) ok=false;
        if (!ok) {unlink(temporary.fileSystemRepresentation);return failed(error,6,"snapshot replace");}
        int directoryFD=open(directory.fileSystemRepresentation,O_RDONLY|O_DIRECTORY);
        if (directoryFD<0) return failed(error,6,"vault open");
        ok=fsync(directoryFD)==0;if (close(directoryFD)) ok=false;
        return ok || failed(error,6,"vault flush");
    }
    bool remove(const std::string &file,LearningError &error) override {
        NSString *path=pathOf(file);
        if (unlink(path.fileSystemRepresentation) && errno!=ENOENT) return failed(error,5,"snapshot remove");
        NSString *directory=path.stringByDeletingLastPathComponent;int fd=open(directory.fileSystemRepresentation,O_RDONLY|O_DIRECTORY);
        if (fd<0) return failed(error,5,"vault open");
        bool ok=fsync(fd)==0;if (close(fd)) ok=false;return ok || failed(error,5,"vault flush");
    }
    bool files(std::vector<std::string> &names,LearningError &error) override {
        NSArray<NSString *> *files=[NSFileManager.defaultManager contentsOfDirectoryAtPath:pathOf(directory_) error:nil];
        if (!files) return failed(error,5,"vault enumeration");
        names.clear();for (NSString *file in files) {const char *value=file.UTF8String;if (!value) return failed(error,9,"snapshot path");names.emplace_back(value);}return true;
    }
    bool seal(const std::string &name,const LearningBytes &plain,LearningBytes &sealed,LearningError &error) override {
        LearningBytes encrypted;if (!crypt(name,plain,encrypted,true,error)) return false;
        sealed={'A','R','I','O','N','U','D','1'};sealed.insert(sealed.end(),encrypted.begin(),encrypted.end());return true;
    }
    bool unseal(const std::string &name,const LearningBytes &sealed,LearningBytes &plain,LearningError &error) override {
        if (sealed.size()<36 || sealed.size()>maximumLearningSnapshot+36 || std::memcmp(sealed.data(),"ARIONUD1",8)) return failed(error,4,"snapshot envelope");
        return crypt(name,LearningBytes(sealed.begin()+8,sealed.end()),plain,false,error);
    }
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

bool keychainFailure(NSInteger code, OSStatus status, NSString *reason) {
    std::lock_guard<std::mutex> lock(errorMutex);
    lastError = [NSError errorWithDomain:@"si.astrafoundry.arion.keyboard.learning" code:code
        userInfo:@{NSLocalizedFailureReasonErrorKey: reason,
            NSUnderlyingErrorKey: [NSError errorWithDomain:NSOSStatusErrorDomain code:status userInfo:nil]}];
    return false;
}

bool verifiedKeyPolicy(NSDictionary *attributes) {
    return [attributes[(__bridge id)kSecAttrAccessible] isEqual:(__bridge id)kSecAttrAccessibleAfterFirstUnlockThisDeviceOnly] &&
        [attributes[(__bridge id)kSecAttrSynchronizable] isEqual:@NO];
}

NSData *keychainKey(bool mayCreate) {
    NSDictionary *legacyQuery = @{(__bridge id)kSecClass: (__bridge id)kSecClassGenericPassword,
        (__bridge id)kSecAttrService: @"si.astrafoundry.arion.keyboard.learning",
        (__bridge id)kSecAttrAccount: @"aes256-gcm-v1", (__bridge id)kSecUseDataProtectionKeychain: @NO,
        (__bridge id)kSecReturnAttributes: @YES, (__bridge id)kSecMatchLimit: (__bridge id)kSecMatchLimitOne};
    OSStatus legacyStatus = SecItemCopyMatching((__bridge CFDictionaryRef)legacyQuery, nullptr);
    if (legacyStatus == errSecSuccess) {
        keychainFailure(11, legacyStatus, @"Legacy keyboard learning key requires explicit migration; no key or vault was changed.");
        return nil;
    }
    if (legacyStatus != errSecItemNotFound) {
        keychainFailure(14, legacyStatus, @"Cannot establish absence of a legacy keyboard learning key."); return nil;
    }
    NSMutableDictionary *query = [@{(__bridge id)kSecClass: (__bridge id)kSecClassGenericPassword,
        (__bridge id)kSecAttrService: @"si.astrafoundry.arion.keyboard.learning",
        (__bridge id)kSecAttrAccount: @"aes256-gcm-v1", (__bridge id)kSecUseDataProtectionKeychain: @YES,
        (__bridge id)kSecAttrSynchronizable: @NO, (__bridge id)kSecReturnData: @YES,
        (__bridge id)kSecReturnAttributes: @YES,
        (__bridge id)kSecMatchLimit: (__bridge id)kSecMatchLimitOne} mutableCopy];
    CFTypeRef result = nullptr;
    OSStatus status = SecItemCopyMatching((__bridge CFDictionaryRef)query, &result);
    if (status == errSecSuccess) {
        id object = CFBridgingRelease(result);
        if (![object isKindOfClass:NSDictionary.class] || !verifiedKeyPolicy(object)) {
            keychainFailure(13, errSecParam, @"Keyboard learning key accessibility or synchronization policy does not match the required device-only policy.");
            return nil;
        }
        NSData *key = object[(__bridge id)kSecValueData];
        if ([key isKindOfClass:NSData.class] && key.length == 32) return key;
        fail(1); return nil;
    }
    if (status != errSecItemNotFound || !mayCreate) {
        keychainFailure(status == errSecMissingEntitlement ? 12 : 14, status,
            status == errSecMissingEntitlement ? @"Keyboard learning requires a provisioning-profile-authorized Data Protection Keychain entitlement." :
                @"Keyboard learning Data Protection key is unavailable; no replacement key was created.");
        return nil;
    }
    NSMutableData *key = [NSMutableData dataWithLength:32];
    if (SecRandomCopyBytes(kSecRandomDefault, key.length, key.mutableBytes) != errSecSuccess) { fail(1); return nil; }
    [query removeObjectForKey:(__bridge id)kSecReturnData];
    [query removeObjectForKey:(__bridge id)kSecReturnAttributes];
    [query removeObjectForKey:(__bridge id)kSecMatchLimit];
    query[(__bridge id)kSecValueData] = key;
    query[(__bridge id)kSecAttrAccessible] = (__bridge id)kSecAttrAccessibleAfterFirstUnlockThisDeviceOnly;
    status = SecItemAdd((__bridge CFDictionaryRef)query, nullptr);
    if (status != errSecSuccess && status != errSecDuplicateItem) {
        keychainFailure(status == errSecMissingEntitlement ? 12 : 14, status,
            status == errSecMissingEntitlement ? @"Keyboard learning requires a provisioning-profile-authorized Data Protection Keychain entitlement." :
                @"Cannot create the required device-only keyboard learning key.");
        return nil;
    }
    // Re-read the OS-kept item, including its policy, before accepting the key.
    return keychainKey(false);
}
} // namespace

NSError *ArionEncryptedLearningError(void) {
    {std::lock_guard<std::mutex> lock(errorMutex);if (lastError) return lastError;}
    auto error=encryptedLearningError();
    return error.code?[NSError errorWithDomain:@"si.astrafoundry.arion.keyboard.learning" code:error.code userInfo:nil]:nil;
}
BOOL ArionRegisterEncryptedLearning(NSString *directory,NSError **error) {
    NSError *directoryError=nil;
    if (![NSFileManager.defaultManager createDirectoryAtPath:directory withIntermediateDirectories:YES attributes:@{NSFilePosixPermissions:@0700} error:&directoryError]) {fail(5);if (error) *error=ArionEncryptedLearningError();return NO;}
    NSArray<NSString *> *files=[NSFileManager.defaultManager contentsOfDirectoryAtPath:directory error:&directoryError];
    if (!files) {fail(5);if (error) *error=ArionEncryptedLearningError();return NO;}
    NSData *key=keychainKey(!hasVaultData(files));
    if (!key) {if (error) *error=ArionEncryptedLearningError();return NO;}
    LearningError failure;
    if (!registerEncryptedLearning(std::make_shared<MacProvider>(directory,key),failure)) {fail(failure.code);if (error) *error=ArionEncryptedLearningError();return NO;}
    {std::lock_guard<std::mutex> lock(errorMutex);lastError=nil;}
    return YES;
}
struct ArionContextStore {std::unique_ptr<LearningDatabase> db;};
ArionContextStore *ArionContextStoreOpen(void) {
    LearningError failure;auto store=std::make_unique<ArionContextStore>();store->db=openContextLearning(failure);
    return store->db?store.release():nullptr;
}
NSDictionary<NSString *,NSString *> *ArionContextStoreRecords(ArionContextStore *store) {
    if (!store) return nil;
    NSMutableDictionary *records=[NSMutableDictionary dictionary];
    for (const auto &[key,value]:store->db->records()) {
        if (!key.empty() && key[0]=='\x01') continue;
        NSString *k=pathOf(key),*v=pathOf(value);if (!k || !v) return nil;records[k]=v;
    }
    return records;
}
BOOL ArionContextStoreApply(ArionContextStore *store,NSDictionary<NSString *,id> *changes) {
    if (!store || !store->db->begin()) return NO;
    for (NSString *key in changes) {
        id value=changes[key];bool ok=false;
        if ([value isKindOfClass:NSString.class]) ok=store->db->update(key.UTF8String,[value UTF8String]);
        else if (value==NSNull.null) ok=store->db->erase(key.UTF8String);
        if (!ok) {store->db->abort();return NO;}
    }
    return store->db->commit();
}
void ArionContextStoreClose(ArionContextStore *store) {delete store;}
BOOL ArionResetEncryptedLearning(NSString *directory,NSError **error) {
    LearningError failure;
    if (!resetLearningVault(std::make_shared<MacProvider>(directory,nil),failure)) {fail(failure.code);if (error) *error=ArionEncryptedLearningError();return NO;}
    {std::lock_guard<std::mutex> lock(errorMutex);lastError=nil;}
    clearEncryptedLearningError();return YES;
}
#ifdef ARION_LEARNING_TESTING
BOOL ArionLearningTestVerifiedKeyPolicy(NSDictionary *attributes) {return verifiedKeyPolicy(attributes);}
BOOL ArionLearningTestMayCreateKey(NSArray<NSString *> *files) {return !hasVaultData(files);}
BOOL ArionRegisterTestEncryptedLearning(NSString *directory,NSData *key,NSError **error) {
    if (key.length!=32) {fail(1);if (error) *error=ArionEncryptedLearningError();return NO;}
    if (![NSFileManager.defaultManager createDirectoryAtPath:directory withIntermediateDirectories:YES attributes:@{NSFilePosixPermissions:@0700} error:error]) return NO;
    LearningError failure;
    if (!registerEncryptedLearning(std::make_shared<MacProvider>(directory,key),failure)) {fail(failure.code);if (error) *error=ArionEncryptedLearningError();return NO;}
    {std::lock_guard<std::mutex> lock(errorMutex);lastError=nil;}
    return YES;
}
rime::Db *ArionLearningTestDatabase(NSString *file,NSString *name,NSData *key) {
    return makeEncryptedRimeDatabase(std::make_shared<MacProvider>(file.stringByDeletingLastPathComponent,key),file.UTF8String,name.UTF8String);
}
#endif
