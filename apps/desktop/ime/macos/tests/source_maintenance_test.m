#import <Foundation/Foundation.h>
#import <Carbon/Carbon.h>
#include <assert.h>

static NSArray *installed;
static NSArray *enabled;
static NSDictionary *current;
static int disables;
static CFArrayRef testList(CFDictionaryRef filter, Boolean all) {
    (void)filter;
    NSArray *value = all ? installed : enabled;
    return value ? (__bridge_retained CFArrayRef)value : NULL;
}
static TISInputSourceRef testCurrent(void) { return current ? (TISInputSourceRef)CFBridgingRetain(current) : NULL; }
static void *testProperty(TISInputSourceRef source, CFStringRef key) { return (__bridge void *)((__bridge NSDictionary *)source)[(__bridge NSString *)key]; }
static OSStatus testDisable(TISInputSourceRef source) { (void)source; disables++; return noErr; }
#define TISCreateInputSourceList testList
#define TISCopyCurrentKeyboardInputSource testCurrent
#define TISGetInputSourceProperty testProperty
#define TISDisableInputSource testDisable
#import "../source_maintenance.h"

static NSDictionary *source(NSString *identity, BOOL active, BOOL selectable) {
    return @{(__bridge NSString *)kTISPropertyInputSourceID: identity,
             (__bridge NSString *)kTISPropertyInputSourceIsEnabled: @(active),
             (__bridge NSString *)kTISPropertyInputSourceIsSelectCapable: @(selectable),
             (__bridge NSString *)kTISPropertyInputSourceIsEnableCapable: @YES};
}
int main(void) { @autoreleasepool {
    NSDictionary *parent = source(ArionSourceParent, YES, NO);
    NSDictionary *disabledParent = source(ArionSourceParent, NO, NO);
    NSDictionary *mode = source(ArionSourceMode, YES, YES);
    current = source(@"com.apple.keylayout.ABC", YES, YES);
    installed = @[parent, mode]; enabled = installed;
    assert([ArionSourceObservation()[@"available"] boolValue]);
    installed = @[disabledParent, mode]; enabled = @[mode];
    NSDictionary *state = ArionSourceObservation();
    assert([state[@"complete"] boolValue] && ![state[@"available"] boolValue]);
    installed = @[mode];
    assert(![ArionSourceObservation()[@"complete"] boolValue]);
    installed = nil;
    assert(![ArionSourceObservation()[@"complete"] boolValue]);
    installed = @[parent, mode]; enabled = nil;
    assert(![ArionSourceObservation()[@"complete"] boolValue]);
    enabled = installed; current = mode;
    assert([ArionSourceObservation()[@"selected"] boolValue]);
    assert([ArionDisableOwnSources().firstObject intValue] != 0 && disables == 0);
    current = source(@"com.apple.keylayout.ABC", YES, YES);
    assert(ArionDisableOwnSources().count == 2 && disables == 2);
    installed = @[parent, mode, mode]; enabled = installed;
    assert(![ArionSourceObservation()[@"complete"] boolValue]);
    puts("{\"cases\":9,\"passed\":true,\"boundary\":\"production observation with controlled TIS snapshots\"}");
} return 0; }
