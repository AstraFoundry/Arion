// Explicit developer reset helper. No secret bytes are read or exported.
#import <Cocoa/Cocoa.h>
#import <Security/Security.h>
#import <LocalAuthentication/LocalAuthentication.h>
#include "native.h"

static BOOL accepted(OSStatus status) {
    if (status == errSecSuccess || status == errSecItemNotFound) return YES;
    fprintf(stderr, "Arion Keychain reset failed: %d\n", (int)status);
    return NO;
}

static BOOL exactItem(NSString *service, NSString *account, BOOL protectedStore, BOOL execute) {
    NSMutableDictionary *query = [@{(__bridge id)kSecClass: (__bridge id)kSecClassGenericPassword,
        (__bridge id)kSecAttrService: service, (__bridge id)kSecAttrAccount: account,
        (__bridge id)kSecUseDataProtectionKeychain: @(protectedStore)} mutableCopy];
    if (protectedStore) {
        query[(__bridge id)kSecAttrSynchronizable] = @NO;
        LAContext *context = [LAContext new];
        context.interactionNotAllowed = YES;
        query[(__bridge id)kSecUseAuthenticationContext] = context;
    }
    if (execute) return accepted(SecItemDelete((__bridge CFDictionaryRef)query));
    query[(__bridge id)kSecReturnAttributes] = @YES;
    query[(__bridge id)kSecMatchLimit] = (__bridge id)kSecMatchLimitOne;
    CFTypeRef attributes = NULL;
    OSStatus status = SecItemCopyMatching((__bridge CFDictionaryRef)query, &attributes);
    if (attributes) CFRelease(attributes);
    return accepted(status);
}

static BOOL accountItems(BOOL execute) {
    NSDictionary *query = @{(__bridge id)kSecClass: (__bridge id)kSecClassGenericPassword,
        (__bridge id)kSecAttrService: @"Arion", (__bridge id)kSecUseDataProtectionKeychain: @NO,
        (__bridge id)kSecReturnAttributes: @YES, (__bridge id)kSecMatchLimit: (__bridge id)kSecMatchLimitAll};
    CFTypeRef result = NULL;
    OSStatus status = SecItemCopyMatching((__bridge CFDictionaryRef)query, &result);
    if (status == errSecItemNotFound) return YES;
    if (!accepted(status)) return NO;
    NSArray *items = CFBridgingRelease(result);
    for (NSDictionary *item in items) {
        NSString *account = item[(__bridge id)kSecAttrAccount];
        if ([account isEqual:@"account-refresh-token"] || [account hasPrefix:@"account-refresh-token/"]) {
            if (!exactItem(@"Arion", account, NO, execute)) return NO;
        }
    }
    return YES;
}

int main(int argc, const char *argv[]) {
    @autoreleasepool {
        NSArray<NSString *> *args = NSProcessInfo.processInfo.arguments;
        if (argc < 2 || (! [args[1] isEqual:@"check"] && ![args[1] isEqual:@"execute"])) return 2;
        BOOL execute = [args[1] isEqual:@"execute"];
        if ([NSRunningApplication runningApplicationsWithBundleIdentifier:@"si.astrafoundry.arion"].count > 0) {
            fprintf(stderr, "Fully quit Arion before resetting.\n");
            return 1;
        }
        // Never request Keychain dialogs from this command-line helper.
        // Legacy login-Keychain items still require the legacy interaction API.
#pragma clang diagnostic push
#pragma clang diagnostic ignored "-Wdeprecated-declarations"
        OSStatus interaction = SecKeychainSetUserInteractionAllowed(false);
#pragma clang diagnostic pop
        if (!accepted(interaction)) return 1;
        if (execute && [args containsObject:@"inputMethod"]) {
            int status = arion_ime_disable("si.astrafoundry.inputmethod.arion.pinyin");
            if (status != 0) { fprintf(stderr, "Arion input source disable failed: %d\n", status); return 1; }
            arion_ime_terminate("si.astrafoundry.inputmethod.arion");
            NSDate *deadline = [NSDate dateWithTimeIntervalSinceNow:5];
            while ([NSRunningApplication runningApplicationsWithBundleIdentifier:@"si.astrafoundry.inputmethod.arion"].count > 0) {
                if (deadline.timeIntervalSinceNow <= 0) {
                    fprintf(stderr, "ArionInput did not terminate; profile and keys were retained.\n");
                    return 1;
                }
                [NSRunLoop.currentRunLoop runUntilDate:[NSDate dateWithTimeIntervalSinceNow:0.05]];
            }
        }
        if ([args containsObject:@"credentials"] && !accountItems(execute)) return 1;
        if ([args containsObject:@"vocabulary"] && !exactItem(@"Arion", @"vocabulary-key", NO, execute)) return 1;
        if ([args containsObject:@"keyboard"]) {
            for (NSNumber *store in @[@YES, @NO]) {
                if (!exactItem(@"si.astrafoundry.arion.keyboard.learning", @"aes256-gcm-v1", store.boolValue, execute)) return 1;
            }
        }
        (void)argv;
        return 0;
    }
}
