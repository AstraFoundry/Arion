// Shared public-TIS observations for the host maintenance entry and desktop.
// Call only from the main thread, or a serialized non-UI caller.
#import <Carbon/Carbon.h>
#import <Foundation/Foundation.h>

static NSString *const ArionSourceParent = @"si.astrafoundry.inputmethod.arion";
static NSString *const ArionSourceMode = @"si.astrafoundry.inputmethod.arion.pinyin";

static BOOL ArionOwnSource(TISInputSourceRef source) {
    CFStringRef value = TISGetInputSourceProperty(source, kTISPropertyInputSourceID);
    return value && (CFEqual(value, (__bridge CFStringRef)ArionSourceParent) ||
                     CFEqual(value, (__bridge CFStringRef)ArionSourceMode));
}

static BOOL ArionSourceFlag(TISInputSourceRef source, CFStringRef key) {
    CFTypeRef value = TISGetInputSourceProperty(source, key);
    return value && CFGetTypeID(value) == CFBooleanGetTypeID() && CFBooleanGetValue(value);
}

// A null list is unknown, never evidence that a source was removed.
static NSDictionary *ArionSourceObservation(void) {
    CFArrayRef all = TISCreateInputSourceList(NULL, true);
    CFArrayRef enabled = TISCreateInputSourceList(NULL, false);
    TISInputSourceRef current = TISCopyCurrentKeyboardInputSource();
    BOOL complete = all && enabled && current;
    BOOL available = NO, selected = current && ArionOwnSource(current);
    NSUInteger ownCount = 0;
    BOOL parentFound = NO, parentEnabled = NO;
    if (all) {
        for (CFIndex i = 0; i < CFArrayGetCount(all); i++) {
            TISInputSourceRef source = (TISInputSourceRef)CFArrayGetValueAtIndex(all, i);
            if (ArionOwnSource(source)) {
                ownCount++;
                CFStringRef identity = TISGetInputSourceProperty(source, kTISPropertyInputSourceID);
                if (CFEqual(identity, (__bridge CFStringRef)ArionSourceParent)) {
                    parentFound = YES;
                    CFTypeRef flag = TISGetInputSourceProperty(source, kTISPropertyInputSourceIsEnabled);
                    if (!flag || CFGetTypeID(flag) != CFBooleanGetTypeID()) complete = NO;
                    parentEnabled = ArionSourceFlag(source, kTISPropertyInputSourceIsEnabled);
                }
                selected |= ArionSourceFlag(source, kTISPropertyInputSourceIsSelected);
            }
        }
    }
    if (enabled) {
        for (CFIndex i = 0; i < CFArrayGetCount(enabled); i++) {
            TISInputSourceRef source = (TISInputSourceRef)CFArrayGetValueAtIndex(enabled, i);
            if (ArionOwnSource(source) && ArionSourceFlag(source, kTISPropertyInputSourceIsSelectCapable)) {
                CFStringRef identity = TISGetInputSourceProperty(source, kTISPropertyInputSourceID);
                if (CFEqual(identity, (__bridge CFStringRef)ArionSourceMode)) {
                    if (!parentFound) complete = NO;
                    // An enabled input-mode flag can survive a disabled parent.
                    // The public contract requires both for actual availability.
                    if (parentEnabled) available = YES;
                } else { available = YES; }
            }
        }
    }
    if (ownCount > 2) complete = NO;
    if (all) CFRelease(all);
    if (enabled) CFRelease(enabled);
    if (current) CFRelease(current);
    return @{@"complete": @(complete), @"available": @(available), @"selected": @(selected), @"ownCount": @(ownCount), @"parentFound": @(parentFound), @"parentEnabled": @(parentEnabled)};
}

static NSArray *ArionDisableOwnSources(void) {
    NSDictionary *before = ArionSourceObservation();
    if (![before[@"complete"] boolValue] || [before[@"selected"] boolValue]) return @[@(paramErr)];
    NSMutableArray *statuses = [NSMutableArray array];
    // Resolve from a successful unfiltered snapshot, child before parent.
    CFArrayRef all = TISCreateInputSourceList(NULL, true);
    if (!all) return @[@(paramErr)];
    for (NSString *identifier in @[ArionSourceMode, ArionSourceParent]) {
        for (CFIndex i = 0; i < CFArrayGetCount(all); i++) {
            TISInputSourceRef source = (TISInputSourceRef)CFArrayGetValueAtIndex(all, i);
            CFStringRef value = TISGetInputSourceProperty(source, kTISPropertyInputSourceID);
            if (!value || !CFEqual(value, (__bridge CFStringRef)identifier)) continue;
            OSStatus status = noErr;
            if (ArionSourceFlag(source, kTISPropertyInputSourceIsEnabled)) {
                status = ArionSourceFlag(source, kTISPropertyInputSourceIsEnableCapable) ? TISDisableInputSource(source) : paramErr;
            }
            [statuses addObject:@(status)];
        }
    }
    CFRelease(all);
    return statuses;
}
