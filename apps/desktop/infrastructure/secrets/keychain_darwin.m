//go:build darwin

#import <Foundation/Foundation.h>
#import <Security/Security.h>
#include "keychain_darwin.h"

static NSDictionary *query(const char *service, const char *account) {
    return @{
        (__bridge id)kSecClass: (__bridge id)kSecClassGenericPassword,
        (__bridge id)kSecAttrService: [NSString stringWithUTF8String:service],
        (__bridge id)kSecAttrAccount: [NSString stringWithUTF8String:account],
    };
}

int arion_keychain_read(const char *service, const char *account, unsigned char *out, int capacity) {
    @autoreleasepool {
        NSMutableDictionary *q = [query(service, account) mutableCopy];
        q[(__bridge id)kSecReturnData] = @YES;
        q[(__bridge id)kSecMatchLimit] = (__bridge id)kSecMatchLimitOne;
        CFTypeRef result = NULL;
        OSStatus status = SecItemCopyMatching((__bridge CFDictionaryRef)q, &result);
        if (status != errSecSuccess) return (int)status;
        NSData *data = (__bridge_transfer NSData *)result;
        if ((int)data.length != capacity) return ARION_KEYCHAIN_SIZE;
        memcpy(out, data.bytes, data.length);
        return 0;
    }
}

int arion_keychain_write(const char *service, const char *account, const unsigned char *bytes, int length) {
    @autoreleasepool {
        NSMutableDictionary *q = [query(service, account) mutableCopy];
        q[(__bridge id)kSecValueData] = [NSData dataWithBytes:bytes length:(NSUInteger)length];
        q[(__bridge id)kSecAttrAccessible] = (__bridge id)kSecAttrAccessibleAfterFirstUnlockThisDeviceOnly;
        return (int)SecItemAdd((__bridge CFDictionaryRef)q, NULL);
    }
}
