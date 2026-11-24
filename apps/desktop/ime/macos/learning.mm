#import "learning.h"
#import <Security/Security.h>
#include "../shared/learning_rime.h"
#include "../shared/learning_posix.h"
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
NSString *pathOf(const std::string &path) {return [[NSString alloc] initWithBytes:path.data() length:path.size() encoding:NSUTF8StringEncoding];}
// CryptoKit AES-256-GCM with the Data Protection Keychain key; the shared
// POSIX provider owns files, leases and the ARIONUD1 envelope.
class CryptoKitCipher final : public LearningCipher {
    NSData *key_;
    bool crypt(const std::string &label,const LearningBytes &input,LearningBytes &output,bool encrypt) {
        if (key_.length!=32) return false;
        void *bytes=nullptr;size_t count=0;
        auto operation=encrypt?arion_learning_seal:arion_learning_open;
        if (!operation(input.data(),input.size(),key_.bytes,key_.length,label.data(),label.size(),&bytes,&count)) return false;
        const auto *first=static_cast<const uint8_t*>(bytes);output.assign(first,first+count);free(bytes);return true;
    }
public:
    explicit CryptoKitCipher(NSData *key):key_(key) {}
    bool seal(const std::string &label,const LearningBytes &plain,LearningBytes &sealed) override {return crypt(label,plain,sealed,true);}
    bool open(const std::string &label,const LearningBytes &sealed,LearningBytes &plain) override {return crypt(label,sealed,plain,false);}
};
std::shared_ptr<LearningProvider> macProvider(NSString *directory,NSData *key) {
    return makePosixLearningProvider(directory.UTF8String,key?std::make_shared<CryptoKitCipher>(key):nullptr,"si.astrafoundry.arion.keyboard.userdb.v1/");
}
// The vault directory is reserved for learning data. Any non-lock entry can
// contain a snapshot, encrypted export or interrupted atomic write; none may
// silently authorize creating a replacement for a lost Keychain key.
bool hasVaultData(NSArray<NSString *> *files) {
    for (NSString *file in files) if (learningVaultEntryIsData(file.UTF8String)) return true;
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
    if (!registerEncryptedLearning(macProvider(directory,key),failure)) {fail(failure.code);if (error) *error=ArionEncryptedLearningError();return NO;}
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
    if (!resetLearningVault(macProvider(directory,nil),failure)) {fail(failure.code);if (error) *error=ArionEncryptedLearningError();return NO;}
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
    if (!registerEncryptedLearning(macProvider(directory,key),failure)) {fail(failure.code);if (error) *error=ArionEncryptedLearningError();return NO;}
    {std::lock_guard<std::mutex> lock(errorMutex);lastError=nil;}
    return YES;
}
rime::Db *ArionLearningTestDatabase(NSString *file,NSString *name,NSData *key) {
    return makeEncryptedRimeDatabase(macProvider(file.stringByDeletingLastPathComponent,key),file.UTF8String,name.UTF8String);
}
#endif
