#import "learning.h"
#include <rime/dict/db.h>
#import <Security/Security.h>
#include <cstdio>
#include <cstdlib>
#include <memory>

static void require(bool success, const char *message) {
    if (!success) { fprintf(stderr, "Encrypted learning test failed: %s\n", message); exit(1); }
}
static std::unique_ptr<rime::Db> database(NSString *file, NSData *key, NSString *name = @"test") {
    return std::unique_ptr<rime::Db>(ArionLearningTestDatabase(file, name, key));
}
static NSData *key(unsigned char byte) {
    unsigned char bytes[32]; memset(bytes, byte, sizeof(bytes));
    return [NSData dataWithBytes:bytes length:sizeof(bytes)];
}

int main() {
    @autoreleasepool {
        require(ArionLearningTestMayCreateKey(@[@".learning.lock", @"test.arion-userdb.lock"]), "locks alone permit first-time provisioning");
        for (NSString *vaultFile in @[@"test.arion-userdb", @".encrypted-crash", @"export.userdb.txt", @"arbitrary-encrypted-backup"]) {
            require(!ArionLearningTestMayCreateKey(@[vaultFile]), "any remaining vault data forbids replacement key creation");
        }
        NSDictionary *devicePolicy = @{(__bridge id)kSecAttrAccessible: (__bridge id)kSecAttrAccessibleAfterFirstUnlockThisDeviceOnly,
            (__bridge id)kSecAttrSynchronizable: @NO};
        require(ArionLearningTestVerifiedKeyPolicy(devicePolicy), "accept exact non-synchronizable device-only key policy");
        require(!ArionLearningTestVerifiedKeyPolicy(@{}), "missing classic-keychain policy attributes are not accepted");
        require(!ArionLearningTestVerifiedKeyPolicy(@{(__bridge id)kSecAttrAccessible: (__bridge id)kSecAttrAccessibleAfterFirstUnlockThisDeviceOnly}),
            "missing synchronization metadata is not equivalent to false");
        require(!ArionLearningTestVerifiedKeyPolicy(@{(__bridge id)kSecAttrAccessible: (__bridge id)kSecAttrAccessibleAfterFirstUnlock,
            (__bridge id)kSecAttrSynchronizable: @NO}), "migratable accessibility policy is refused");
        require(!ArionLearningTestVerifiedKeyPolicy(@{(__bridge id)kSecAttrAccessible: (__bridge id)kSecAttrAccessibleAfterFirstUnlockThisDeviceOnly,
            (__bridge id)kSecAttrSynchronizable: @YES}), "synchronizable key policy is refused");
        NSString *directory = [NSTemporaryDirectory() stringByAppendingPathComponent:NSUUID.UUID.UUIDString];
        require([NSFileManager.defaultManager createDirectoryAtPath:directory withIntermediateDirectories:YES
            attributes:@{NSFilePosixPermissions: @0700} error:nil], "create isolated directory");
        NSString *file = [directory stringByAppendingPathComponent:@"test.arion-userdb"];
        NSString *backup = [directory stringByAppendingPathComponent:@"export.userdb.txt"];
        NSData *testKey = key(17);
        auto db = database(file, testKey);
        require(db->Open(), "open new encrypted database");
        require(db->Update("ni hao \t synthetic-secret-phrase", "c=5 d=1 t=7"), "persist selected phrase");
        std::string value;
        require(db->Fetch("ni hao \t synthetic-secret-phrase", &value) && value == "c=5 d=1 t=7", "fetch committed phrase");
        NSDictionary *attributes = [NSFileManager.defaultManager attributesOfItemAtPath:file error:nil];
        require([attributes[NSFilePosixPermissions] unsignedIntValue] == 0600, "ciphertext file private permissions");
        NSData *first = [NSData dataWithContentsOfFile:file];
        require(first.length > 36, "encrypted envelope written");
        NSData *canary = [@"synthetic-secret-phrase" dataUsingEncoding:NSUTF8StringEncoding];
        require([first rangeOfData:canary options:0 range:NSMakeRange(0, first.length)].location == NSNotFound,
                "no plaintext user term in snapshot");
        require(db->Update("ni hao \t synthetic-secret-phrase", value), "rewrite same values");
        require(![first isEqual:[NSData dataWithContentsOfFile:file]], "fresh random nonce per write");

        auto transaction = dynamic_cast<rime::Transactional *>(db.get());
        require(transaction && transaction->BeginTransaction(), "begin selection transaction");
        require(db->Update("abort-me", "c=1"), "stage new selection");
        require(!db->Fetch("abort-me", &value), "pending changes invisible like LevelDb write batch");
        NSData *beforeAbort = [NSData dataWithContentsOfFile:file];
        require(transaction->AbortTransaction(), "abort recent selection");
        require([[NSData dataWithContentsOfFile:file] isEqual:beforeAbort], "abort leaves disk unchanged");
        require(!db->Fetch("abort-me", &value), "aborted phrase absent");
        require(transaction->BeginTransaction(), "begin persistent transaction");
        require(db->Update("second", "c=3") && transaction->CommitTransaction(), "commit transaction");
        require(db->Fetch("second", &value) && value == "c=3", "transaction visible after commit");

        auto prefix = db->Query("ni ");
        std::string found;
        require(prefix && prefix->GetNextRecord(&found, &value) && found == "ni hao \t synthetic-secret-phrase",
                "prefix cursor returns original phrase");
        require(prefix->exhausted(), "prefix cursor bounded");
        auto metadata = db->QueryMetadata();
        require(metadata && metadata->GetNextRecord(&found, &value) && found[0] == '/', "metadata prefix stripped");
        require(db->Backup(rime::path(backup.fileSystemRepresentation)), "encrypted backup even with text extension");
        NSData *exported = [NSData dataWithContentsOfFile:backup];
        require([exported rangeOfData:canary options:0 range:NSMakeRange(0, exported.length)].location == NSNotFound,
                "backup never uses plaintext uniform export");
        require(db->Erase("second") && !db->Fetch("second", &value), "delete learned phrase");
        require(db->Restore(rime::path(backup.fileSystemRepresentation)) && db->Fetch("second", &value), "encrypted restore");
        require(db->Close(), "close persisted database");
        require(db->Open() && db->Fetch("second", &value) && value == "c=3", "learning survives reopen");
        auto competing = database(file, testKey);
        require(!competing->Open(), "second writer refused by file lock");
        require(db->Close(), "close before authentication tests");

        NSData *beforeKeyFailure = [NSData dataWithContentsOfFile:file];
        auto missing = database(file, nil);
        require(!missing->Open(), "missing key fails closed");
        auto wrong = database(file, key(18));
        require(!wrong->Open(), "wrong key fails authentication");
        auto renamed = database(file, testKey, @"another-dictionary");
        require(!renamed->Open(), "associated data binds dictionary identity");
        require([[NSData dataWithContentsOfFile:file] isEqual:beforeKeyFailure], "unreadable data not replaced");
        NSMutableData *tampered = [[NSData dataWithContentsOfFile:file] mutableCopy];
        static_cast<unsigned char *>(tampered.mutableBytes)[tampered.length - 1] ^= 1;
        require([tampered writeToFile:file atomically:YES], "write isolated tamper fixture");
        auto corrupt = database(file, testKey);
        require(!corrupt->Open(), "tampered snapshot fails authentication");
        require([[NSData dataWithContentsOfFile:file] isEqual:tampered], "tampered data not reset or overwritten");
        require([exported writeToFile:file atomically:YES], "restore fixture ciphertext");

        require(db->Open(), "open for reset");
        require(transaction->BeginTransaction(), "reset transaction");
        auto all = db->QueryAll();
        while (all->GetNextRecord(&found, &value)) {
            if (!found.empty() && found[0] != '\x01') require(db->Erase(found), "erase phrase during reset");
        }
        require(transaction->CommitTransaction(), "persist reset");
        require(!db->Fetch("second", &value) && !db->Fetch("ni hao \t synthetic-secret-phrase", &value), "reset removes learned phrases");
        require(db->Close() && db->Remove(), "delete closed encrypted database");
        require(![NSFileManager.defaultManager fileExistsAtPath:file], "deleted database absent");
        NSString *failureFile = [directory stringByAppendingPathComponent:@"failure.arion-userdb"];
        auto failing = database(failureFile, testKey);
        require(failing->Open(), "open persistence failure fixture");
        require([NSFileManager.defaultManager removeItemAtPath:failureFile error:nil], "remove fixture ciphertext");
        require([NSFileManager.defaultManager createDirectoryAtPath:failureFile withIntermediateDirectories:NO attributes:nil error:nil],
                "create explicit write failure fixture");
        require(!failing->Update("must-not-persist", "c=1"), "write failure reported");
        require(failing->disabled(), "write failure disables learning instead of silently losing persistence");
        require(!failing->Fetch("must-not-persist", &value), "failed update not accepted into memory");
        require(!failing->Update("next", "c=1"), "later writes blocked until explicit recovery");
        require(failing->Close(), "close failed writer");
        require([NSFileManager.defaultManager removeItemAtPath:directory error:nil], "remove synthetic files");
        NSString *resetDirectory = [NSTemporaryDirectory() stringByAppendingPathComponent:NSUUID.UUID.UUIDString];
        require([NSFileManager.defaultManager createDirectoryAtPath:resetDirectory withIntermediateDirectories:YES attributes:nil error:nil], "create reset fixture");
        NSString *resetFile = [resetDirectory stringByAppendingPathComponent:@"test.arion-userdb"];
        auto resetDb = database(resetFile, testKey);
        require(resetDb->Open() && resetDb->Update("learned", "c=1"), "populate reset fixture");
        NSError *resetError = nil;
        require(!ArionResetEncryptedLearning(resetDirectory, &resetError) && resetError, "reset refuses live database");
        require(resetDb->Close(), "close before explicit reset");
        require(ArionResetEncryptedLearning(resetDirectory, &resetError), "explicit reset clears closed learning files");
        require(![NSFileManager.defaultManager fileExistsAtPath:resetFile], "reset ciphertext absent");
        require(!ArionEncryptedLearningError(), "explicit recovery clears sticky failure");
        require(resetDb->Open() && !resetDb->Fetch("learned", &value), "reset reuses existing key for empty learning");
        require(resetDb->Close(), "close reset fixture");
        require([NSFileManager.defaultManager removeItemAtPath:resetDirectory error:nil], "remove reset fixture");
        puts("Encrypted learning source tests passed.");
    }
    return 0;
}
