// Exercises the real production Keychain boundary in separate owned processes.
// Never requests kSecReturnData outside the production encryption adapter.
#import <Cocoa/Cocoa.h>
#import <Security/Security.h>
#import <LocalAuthentication/LocalAuthentication.h>
#include <sys/stat.h>
#import "../keyboard.h"
#import "../learning.h"

extern int32_t arion_keyboard_sha256(const void *, size_t, unsigned char *);
static NSString *const service = @"si.astrafoundry.arion.keyboard.learning";
static NSString *const account = @"aes256-gcm-v1";

static void require(BOOL condition, NSString *message) {
    if (!condition) @throw [NSException exceptionWithName:@"KeychainLearningTestFailure" reason:message userInfo:nil];
}
static NSString *digest(NSData *data) {
    unsigned char bytes[32];
    require(arion_keyboard_sha256(data.bytes, data.length, bytes), @"hash operation failed");
    NSMutableString *result = [NSMutableString stringWithCapacity:64];
    for (NSUInteger i = 0; i < sizeof(bytes); i++) [result appendFormat:@"%02x", bytes[i]];
    return result;
}
static void save(id value, NSString *file) {
    NSData *data = [NSJSONSerialization dataWithJSONObject:value options:NSJSONWritingPrettyPrinted error:nil];
    require(data && [data writeToFile:file options:NSDataWritingAtomic error:nil], @"cannot persist owned test metadata");
    require(chmod(file.fileSystemRepresentation, 0600) == 0, @"cannot protect owned test metadata");
}
static NSMutableDictionary *state(NSString *file) {
    NSData *bytes = [NSData dataWithContentsOfFile:file];
    id value = bytes ? [NSJSONSerialization JSONObjectWithData:bytes options:NSJSONReadingMutableContainers error:nil] : nil;
    require([value isKindOfClass:NSDictionary.class] && [value[@"baselineAbsent"] isEqual:@YES], @"owned baseline-absence receipt is required");
    return value;
}
static OSStatus attributes(NSNumber *dataProtection, NSDictionary **result) {
    LAContext *authentication = [LAContext new];
    authentication.interactionNotAllowed = YES;
    NSMutableDictionary *query = [@{(__bridge id)kSecClass: (__bridge id)kSecClassGenericPassword,
        (__bridge id)kSecAttrService: service, (__bridge id)kSecAttrAccount: account,
        (__bridge id)kSecMatchLimit: (__bridge id)kSecMatchLimitOne,
        (__bridge id)kSecReturnAttributes: @YES, (__bridge id)kSecReturnPersistentRef: @YES,
        (__bridge id)kSecUseAuthenticationContext: authentication} mutableCopy];
    if (dataProtection) query[(__bridge id)kSecUseDataProtectionKeychain] = dataProtection;
    if (dataProtection.boolValue) query[(__bridge id)kSecAttrSynchronizable] = (__bridge id)kSecAttrSynchronizableAny;
    CFTypeRef value = NULL;
    OSStatus status = SecItemCopyMatching((__bridge CFDictionaryRef)query, &value);
    if (value) {
        id object = CFBridgingRelease(value);
        require([object isKindOfClass:NSDictionary.class], @"attributes query returned an invalid result type");
        if (result) *result = object;
    }
    return status;
}
static NSData *persistentReference(NSDictionary *attributes) {
    id reference = attributes[(__bridge id)kSecValuePersistentRef];
    require([reference isKindOfClass:NSData.class] && [reference length] > 0, @"owned item has no persistent reference");
    return reference;
}
static NSDictionary *publicAttributes(NSDictionary *attributes) {
    id accessible = attributes[(__bridge id)kSecAttrAccessible];
    id sync = attributes[(__bridge id)kSecAttrSynchronizable];
    id created = attributes[(__bridge id)kSecAttrCreationDate];
    return @{@"accessiblePresent": @(accessible != nil), @"accessible": accessible ?: NSNull.null,
        @"matchesAfterFirstUnlockThisDeviceOnly": @([accessible isEqual:(__bridge id)kSecAttrAccessibleAfterFirstUnlockThisDeviceOnly]),
        @"synchronizablePresent": @(sync != nil), @"synchronizable": sync ?: NSNull.null,
        @"creationDate": [created isKindOfClass:NSDate.class] ? @([(NSDate *)created timeIntervalSince1970]) : NSNull.null};
}
static NSDictionary *backingEvidence(NSData *ownedReference) {
    NSMutableDictionary *result = [NSMutableDictionary dictionary];
    for (NSNumber *selector in @[@NO, @YES]) {
        NSDictionary *item = nil;
        OSStatus status = attributes(selector, &item);
        NSString *name = selector.boolValue ? @"dataProtection" : @"legacy";
        result[name] = @{@"status": @(status),
            @"sameOwnedReference": @(status == errSecSuccess && [persistentReference(item) isEqual:ownedReference]),
            @"attributes": status == errSecSuccess ? publicAttributes(item) : NSNull.null};
    }
    return result;
}
static NSDictionary *inventory(NSString *directory) {
    NSMutableDictionary *result = [NSMutableDictionary dictionary];
    NSDirectoryEnumerator *enumerator = [NSFileManager.defaultManager enumeratorAtPath:directory];
    require(enumerator != nil, @"owned vault directory is missing");
    for (NSString *member in enumerator) {
        NSString *file = [directory stringByAppendingPathComponent:member];
        NSDictionary *attributes = [NSFileManager.defaultManager attributesOfItemAtPath:file error:nil];
        if ([attributes[NSFileType] isEqual:NSFileTypeDirectory]) continue;
        require([attributes[NSFileType] isEqual:NSFileTypeRegular], @"owned vault contains a nonregular file");
        NSData *bytes = [NSData dataWithContentsOfFile:file];
        require(bytes != nil, @"cannot read owned vault bytes");
        result[member] = @{@"sha256": digest(bytes), @"sizeBytes": @(bytes.length)};
    }
    return result;
}
static ArionKeyboardSnapshot *type(ArionKeyboardEngine *engine, uintptr_t session, NSString *input) {
    ArionKeyboardSnapshot *snapshot = nil;
    for (NSUInteger i = 0; i < input.length; i++) {
        require([engine processKey:[input characterAtIndex:i] modifiers:0 session:session], @"real engine did not handle fixture key");
        snapshot = [engine snapshot:session error:nil];
        require(snapshot && !snapshot.failure && snapshot.commit.length == 0, @"fixture composition failed or committed unexpectedly");
    }
    return snapshot;
}
static void requireOwnedItem(NSMutableDictionary *receipt, NSDictionary **item) {
    NSData *reference = [[NSData alloc] initWithBase64EncodedString:receipt[@"persistentReference"] options:0];
    NSNumber *backend = receipt[@"dataProtection"];
    require(reference.length > 0 && [backend isKindOfClass:NSNumber.class], @"owned item receipt is missing its exact reference or backend");
    CFTypeRef value = NULL;
    OSStatus status = SecItemCopyMatching((__bridge CFDictionaryRef)@{
        (__bridge id)kSecValuePersistentRef: reference, (__bridge id)kSecUseDataProtectionKeychain: backend,
        (__bridge id)kSecReturnAttributes: @YES, (__bridge id)kSecReturnPersistentRef: @YES}, &value);
    id found = value ? CFBridgingRelease(value) : nil;
    require(status == errSecSuccess && [found isKindOfClass:NSDictionary.class] &&
        [persistentReference(found) isEqual:reference], @"owned exact-reference key item is unavailable");
    if (item) *item = found;
}

int main(int argc, const char *argv[]) {
    @autoreleasepool {
        if (argc != 7) { fprintf(stderr, "usage: keychain-learning PHASE DATA USER RECEIPT REPORT RUN_ID\n"); return 2; }
        NSString *phase = @(argv[1]), *data = @(argv[2]), *user = @(argv[3]);
        NSString *receiptFile = @(argv[4]), *reportFile = @(argv[5]), *runID = @(argv[6]);
        NSString *vault = [user stringByAppendingPathComponent:@"learning"];
        NSMutableDictionary *report = [@{@"phase": phase, @"runID": runID,
            @"boundary": @"production encryption/engine with real OS-kept key; owned profile; no IMK source dispatch",
            @"secretDataRequestedByHarness": @NO} mutableCopy];
        Boolean priorInteraction = false;
#pragma clang diagnostic push
#pragma clang diagnostic ignored "-Wdeprecated-declarations"
        OSStatus interactionStatus = SecKeychainGetUserInteractionAllowed(&priorInteraction);
        if (interactionStatus == errSecSuccess) interactionStatus = SecKeychainSetUserInteractionAllowed(false);
#pragma clang diagnostic pop
        report[@"interactionPolicyStatus"] = @(interactionStatus);
        BOOL passed = NO;
        @try {
            require(interactionStatus == errSecSuccess, @"cannot establish noninteractive legacy Keychain boundary");
            if ([phase isEqual:@"guard"]) {
                require(![NSFileManager.defaultManager fileExistsAtPath:receiptFile], @"baseline receipt already exists");
                OSStatus status = attributes(nil, NULL);
                report[@"baselineStatus"] = @(status);
                require(status == errSecItemNotFound && attributes(@NO, NULL) == errSecItemNotFound &&
                    attributes(@YES, NULL) == errSecItemNotFound, @"ABORT: production key already exists or absence cannot be proven");
                save(@{@"baselineAbsent": @YES, @"baselineStatus": @(status), @"runID": runID,
                    @"started": @(NSDate.date.timeIntervalSince1970)}, receiptFile);
            } else {
                NSMutableDictionary *receipt = state(receiptFile);
                require([receipt[@"runID"] isEqual:runID], @"owned receipt run identifier differs");
                if ([phase isEqual:@"legacy-rejection"]) {
                    require(attributes(@NO, NULL) == errSecItemNotFound && attributes(@YES, NULL) == errSecItemNotFound,
                        @"ABORT: namespace changed before legacy rejection test");
                    NSMutableData *secret = [NSMutableData dataWithLength:32];
                    require(SecRandomCopyBytes(kSecRandomDefault, secret.length, secret.mutableBytes) == errSecSuccess,
                        @"legacy fixture random generation failed");
                    CFTypeRef created = NULL;
                    OSStatus status = SecItemAdd((__bridge CFDictionaryRef)@{(__bridge id)kSecClass: (__bridge id)kSecClassGenericPassword,
                        (__bridge id)kSecAttrService: service, (__bridge id)kSecAttrAccount: account,
                        (__bridge id)kSecUseDataProtectionKeychain: @NO, (__bridge id)kSecValueData: secret,
                        (__bridge id)kSecReturnPersistentRef: @YES}, &created);
                    report[@"addStatus"] = @(status); require(status == errSecSuccess, @"owned legacy fixture creation failed");
                    NSData *reference = CFBridgingRelease(created);
                    require([reference isKindOfClass:NSData.class], @"legacy fixture lacks ownership reference");
                    receipt[@"persistentReference"] = [reference base64EncodedStringWithOptions:0]; receipt[@"dataProtection"] = @NO; save(receipt, receiptFile);
                    NSError *error = nil;
                    BOOL registered = ArionRegisterEncryptedLearning(vault, &error);
                    report[@"registered"] = @(registered); report[@"failureCode"] = @(error.code);
                    report[@"failureReason"] = error.userInfo[NSLocalizedFailureReasonErrorKey] ?: @"";
                    require(!registered && error.code == 11, @"legacy key was not explicitly rejected before DP access");
                    requireOwnedItem(receipt, NULL);
                    report[@"dataProtectionStatus"] = @(attributes(@YES, NULL));
                    require([report[@"dataProtectionStatus"] intValue] == errSecItemNotFound, @"legacy rejection created a second backend key");
                    report[@"vaultInventory"] = inventory(vault);
                    require([report[@"vaultInventory"] count] == 0, @"legacy rejection wrote vault data");
                } else if ([phase isEqual:@"capability"]) {
                    require(attributes(@NO, NULL) == errSecItemNotFound && attributes(@YES, NULL) == errSecItemNotFound,
                        @"ABORT: namespace changed before Data Protection capability test");
                    NSMutableData *secret = [NSMutableData dataWithLength:32];
                    require(SecRandomCopyBytes(kSecRandomDefault, secret.length, secret.mutableBytes) == errSecSuccess,
                        @"capability random generation failed");
                    NSDictionary *add = @{(__bridge id)kSecClass: (__bridge id)kSecClassGenericPassword,
                        (__bridge id)kSecAttrService: service, (__bridge id)kSecAttrAccount: account,
                        (__bridge id)kSecUseDataProtectionKeychain: @YES, (__bridge id)kSecAttrSynchronizable: @NO,
                        (__bridge id)kSecAttrAccessible: (__bridge id)kSecAttrAccessibleAfterFirstUnlockThisDeviceOnly,
                        (__bridge id)kSecValueData: secret, (__bridge id)kSecReturnPersistentRef: @YES};
                    CFTypeRef created = NULL;
                    OSStatus status = SecItemAdd((__bridge CFDictionaryRef)add, &created);
                    report[@"addStatus"] = @(status);
                    require(status == errSecSuccess, @"Data Protection capability add failed; inspect exact addStatus");
                    NSData *reference = CFBridgingRelease(created);
                    require([reference isKindOfClass:NSData.class], @"capability add returned no ownership reference");
                    receipt[@"persistentReference"] = [reference base64EncodedStringWithOptions:0];
                    receipt[@"dataProtection"] = @YES; save(receipt, receiptFile);
                    NSDictionary *item = nil;
                    status = attributes(@YES, &item); report[@"readAttributesStatus"] = @(status);
                    require(status == errSecSuccess && [persistentReference(item) isEqual:reference], @"DP item ownership verification failed");
                    report[@"keyAttributes"] = publicAttributes(item);
                    report[@"backingEvidence"] = backingEvidence(reference);
                    require([item[(__bridge id)kSecAttrAccessible] isEqual:(__bridge id)kSecAttrAccessibleAfterFirstUnlockThisDeviceOnly] &&
                        [item[(__bridge id)kSecAttrSynchronizable] isEqual:@NO], @"DP accessibility or synchronization policy mismatch");
                    status = SecItemDelete((__bridge CFDictionaryRef)@{(__bridge id)kSecValuePersistentRef: reference,
                        (__bridge id)kSecUseDataProtectionKeychain: @YES});
                    report[@"deleteStatus"] = @(status);
                    require(status == errSecSuccess && attributes(@YES, NULL) == errSecItemNotFound &&
                        attributes(@NO, NULL) == errSecItemNotFound, @"capability test did not restore absent namespace");
                } else if ([phase isEqual:@"writer"]) {
                    require(attributes(@NO, NULL) == errSecItemNotFound && attributes(@YES, NULL) == errSecItemNotFound, @"ABORT: key appeared after baseline; do not read or delete it");
                    NSError *error = nil;
                    BOOL registered = ArionRegisterEncryptedLearning(vault, &error);
                    NSDictionary *item = nil;
                    require(attributes(@YES, &item) == errSecSuccess, @"created Data Protection key attributes unavailable");
                    NSData *reference = persistentReference(item);
                    receipt[@"persistentReference"] = [reference base64EncodedStringWithOptions:0];
                    receipt[@"referenceDigest"] = digest(reference);
                    receipt[@"dataProtection"] = @YES;
                    save(receipt, receiptFile); // Cleanup receipt is durable before checking registration or engine work.
                    require(registered, @"production OS-key provisioning failed");
                    report[@"keyAttributes"] = publicAttributes(item);
                    report[@"backingEvidence"] = backingEvidence(reference);
                    ArionKeyboardEngine *engine = ArionKeyboardEngine.shared;
                    require([engine startWithData:data userDirectory:user error:&error], @"production keyboard startup failed");
                    uintptr_t session = [engine createSession];
                    require(session != 0, @"real engine session unavailable");
                    ArionKeyboardSnapshot *before = type(engine, session, @"shishi");
                    NSUInteger selection = NSNotFound;
                    for (NSUInteger i = 1; i < before.candidates.count; i++) if (before.candidates[i].length == 2) { selection = i; break; }
                    require(selection != NSNotFound, @"real fixture lacks a nonfirst complete candidate");
                    NSString *chosen = before.candidates[selection];
                    require([engine select:selection session:session], @"actual selection failed");
                    ArionKeyboardSnapshot *committed = [engine snapshot:session error:&error];
                    require(committed && !committed.failure && [committed.commit isEqual:chosen], @"engine did not confirm the selected fixture");
                    [engine destroySession:session]; [engine stop];
                    require(!ArionEncryptedLearningError(), @"learning persistence failed");
                    receipt[@"chosen"] = chosen;
                    receipt[@"vaultInventory"] = inventory(vault);
                    save(receipt, receiptFile);
                    report[@"beforeCandidates"] = before.candidates;
                    report[@"committed"] = chosen;
                    report[@"vaultInventory"] = receipt[@"vaultInventory"];
                } else if ([phase isEqual:@"reader"]) {
                    NSDictionary *item = nil; requireOwnedItem(receipt, &item);
                    report[@"keyAttributes"] = publicAttributes(item);
                    ArionKeyboardEngine *engine = ArionKeyboardEngine.shared;
                    require([engine startWithData:data userDirectory:user error:nil], @"fresh-process production key read failed");
                    uintptr_t session = [engine createSession]; require(session != 0, @"reader session unavailable");
                    ArionKeyboardSnapshot *snapshot = type(engine, session, @"shishi");
                    require([snapshot.candidates.firstObject isEqual:receipt[@"chosen"]], @"actual learned ranking did not survive fresh-process restart");
                    report[@"afterRestartCandidates"] = snapshot.candidates;
                    [engine clear:session]; [engine destroySession:session]; [engine stop];
                    require(!ArionEncryptedLearningError(), @"fresh-process read or cancellation failed");
                    receipt[@"vaultInventory"] = inventory(vault); save(receipt, receiptFile);
                } else if ([phase isEqual:@"delete"] || [phase isEqual:@"cleanup"]) {
                    OSStatus legacy = attributes(@NO, NULL), modern = attributes(@YES, NULL);
                    OSStatus status = errSecItemNotFound;
                    if (legacy == errSecSuccess || modern == errSecSuccess) {
                        NSDictionary *item = nil; requireOwnedItem(receipt, &item);
                        NSData *reference = persistentReference(item);
                        status = SecItemDelete((__bridge CFDictionaryRef)@{(__bridge id)kSecValuePersistentRef: reference,
                            (__bridge id)kSecUseDataProtectionKeychain: receipt[@"dataProtection"]});
                        require(status == errSecSuccess, @"owned exact-reference/backend deletion failed");
                    } else require(legacy == errSecItemNotFound && modern == errSecItemNotFound, @"owned key cleanup status is unknown");
                    report[@"deleteStatus"] = @(status);
                    report[@"finalLegacyStatus"] = @(attributes(@NO, NULL));
                    report[@"finalDataProtectionStatus"] = @(attributes(@YES, NULL));
                    report[@"finalKeyStatus"] = @(attributes(nil, NULL));
                    require([report[@"finalLegacyStatus"] intValue] == errSecItemNotFound &&
                        [report[@"finalDataProtectionStatus"] intValue] == errSecItemNotFound &&
                        [report[@"finalKeyStatus"] intValue] == errSecItemNotFound, @"production key namespace not restored absent");
                    receipt[@"keyDeleted"] = @YES; save(receipt, receiptFile);
                } else if ([phase isEqual:@"missing"]) {
                    require(attributes(@NO, NULL) == errSecItemNotFound && attributes(@YES, NULL) == errSecItemNotFound, @"missing-key phase requires owned key deletion");
                    NSDictionary *before = inventory(vault);
                    require([before isEqual:receipt[@"vaultInventory"]], @"vault changed before missing-key phase");
                    NSError *error = nil;
                    BOOL started = [ArionKeyboardEngine.shared startWithData:data userDirectory:user error:&error];
                    report[@"startupSucceeded"] = @(started);
                    report[@"failureDomain"] = error.domain ?: @"";
                    report[@"failureCode"] = @(error.code);
                    require(!started && error && ArionEncryptedLearningError(), @"missing real OS key did not fail closed");
                    require(attributes(@NO, NULL) == errSecItemNotFound && attributes(@YES, NULL) == errSecItemNotFound, @"missing-key startup silently provisioned replacement key");
                    NSDictionary *after = inventory(vault);
                    require([before isEqual:after], @"missing-key startup changed encrypted vault bytes");
                    report[@"vaultUnchanged"] = @YES;
                    report[@"vaultInventory"] = after;
                } else require(NO, @"unknown test phase");
            }
            passed = YES;
        } @catch (NSException *exception) {
            report[@"failure"] = exception.reason ?: exception.name;
        } @finally {
            [ArionKeyboardEngine.shared stop];
#pragma clang diagnostic push
#pragma clang diagnostic ignored "-Wdeprecated-declarations"
            OSStatus restore = interactionStatus == errSecSuccess ? SecKeychainSetUserInteractionAllowed(priorInteraction) : interactionStatus;
#pragma clang diagnostic pop
            report[@"interactionPolicyRestored"] = @(restore == errSecSuccess);
            if (restore != errSecSuccess) passed = NO;
            report[@"passed"] = @(passed);
            save(report, reportFile);
        }
        return passed ? 0 : 1;
    }
}
