//go:build darwin

#import <AVFoundation/AVFoundation.h>
#import <ApplicationServices/ApplicationServices.h>
#import <Carbon/Carbon.h>
#import <Cocoa/Cocoa.h>
#import <Security/Security.h>
#include "native.h"
#import "../../ime/macos/source_maintenance.h"
#include <fcntl.h>
#include <stdio.h>
#include <errno.h>

int arion_mic_status(void) {
    switch ([AVCaptureDevice authorizationStatusForMediaType:AVMediaTypeAudio]) {
    case AVAuthorizationStatusAuthorized: return ARION_AUTH_GRANTED;
    case AVAuthorizationStatusDenied: return ARION_AUTH_DENIED;
    case AVAuthorizationStatusRestricted: return ARION_AUTH_RESTRICTED;
    default: return ARION_AUTH_UNDETERMINED;
    }
}

void arion_mic_request(void) {
    [AVCaptureDevice requestAccessForMediaType:AVMediaTypeAudio completionHandler:^(BOOL granted) { (void)granted; }];
}

int arion_ax_trusted(int prompt) {
    if (!prompt) return AXIsProcessTrusted() ? 1 : 0;
    NSDictionary *options = @{(__bridge NSString *)kAXTrustedCheckOptionPrompt: @YES};
    return AXIsProcessTrustedWithOptions((__bridge CFDictionaryRef)options) ? 1 : 0;
}

void arion_open_url(const char *url) {
    @autoreleasepool {
        NSURL *target = [NSURL URLWithString:[NSString stringWithUTF8String:url]];
        if (target) [[NSWorkspace sharedWorkspace] openURL:target];
    }
}

static void copy_string(CFTypeRef value, char *out, size_t size) {
    out[0] = 0;
    if (value && CFGetTypeID(value) == CFStringGetTypeID()) {
        CFStringGetCString((CFStringRef)value, out, (CFIndex)size, kCFStringEncodingUTF8);
    }
}

static AXUIElementRef focused_element(int *error) {
    AXUIElementRef system = AXUIElementCreateSystemWide();
    CFTypeRef element = NULL;
    AXError result = AXUIElementCopyAttributeValue(system, kAXFocusedUIElementAttribute, &element);
    CFRelease(system);
    *error = (int)result;
    if (result != kAXErrorSuccess || !element) return NULL;
    return (AXUIElementRef)element;
}

// copy_site reads only the authorized outermost web area's http(s) host.
static void copy_site(AXUIElementRef element, char *out, size_t size) {
    out[0] = 0;
    AXUIElementRef current = (AXUIElementRef)CFRetain(element);
    for (int depth = 0; current && depth < 64; depth++) {
        CFTypeRef role = NULL;
        AXError error = AXUIElementCopyAttributeValue(current, kAXRoleAttribute, &role);
        int web = error == kAXErrorSuccess && role && CFEqual(role, CFSTR("AXWebArea"));
        if (role) CFRelease(role);
        if (web) {
            CFTypeRef url = NULL;
            if (AXUIElementCopyAttributeValue(current, kAXURLAttribute, &url) == kAXErrorSuccess && url) {
                if (CFGetTypeID(url) == CFURLGetTypeID()) {
                    NSURL *page = (__bridge NSURL *)url;
                    NSString *scheme = page.scheme.lowercaseString;
                    if (([scheme isEqualToString:@"https"] || [scheme isEqualToString:@"http"]) && page.host.length) {
                        strlcpy(out, page.host.lowercaseString.UTF8String, size);
                    }
                }
                CFRelease(url);
            }
        }
        CFTypeRef parent = NULL;
        error = AXUIElementCopyAttributeValue(current, kAXParentAttribute, &parent);
        CFRelease(current);
        current = error == kAXErrorSuccess ? (AXUIElementRef)parent : NULL;
    }
    if (current) CFRelease(current);
}

static CFTypeRef selected_range(AXUIElementRef element) {
    CFTypeRef range = NULL;
    if (!element || AXUIElementCopyAttributeValue(element, kAXSelectedTextRangeAttribute, &range) != kAXErrorSuccess) return NULL;
    if (range && CFGetTypeID(range) == AXValueGetTypeID() && AXValueGetType((AXValueRef)range) == kAXValueCFRangeType) return range;
    if (range) CFRelease(range);
    return NULL;
}

int arion_selection_matches(void *element, void *selection) {
    if (!selection) return 1;
    CFTypeRef current = selected_range((AXUIElementRef)element);
    int same = current && CFEqual(current, (CFTypeRef)selection);
    if (current) CFRelease(current);
    return same;
}

void *arion_copy_selection(void *element) {
    return (void *)selected_range((AXUIElementRef)element);
}

int arion_capture_target(arion_target *out, int read_site) {
    memset(out, 0, sizeof(*out));
    if (!AXIsProcessTrusted()) return ARION_TARGET_UNTRUSTED;
    @autoreleasepool {
        NSRunningApplication *app = [[NSWorkspace sharedWorkspace] frontmostApplication];
        if (!app) return ARION_TARGET_NONE;
        out->pid = app.processIdentifier;
        if (app.bundleIdentifier) strlcpy(out->bundle, app.bundleIdentifier.UTF8String, sizeof(out->bundle));
        if (app.localizedName) strlcpy(out->app_name, app.localizedName.UTF8String, sizeof(out->app_name));
        out->secure_input = IsSecureEventInputEnabled() ? 1 : 0;
        int error = 0;
        AXUIElementRef element = focused_element(&error);
        if (!element) return ARION_TARGET_OK; // app-level target; safety unknown
        CFTypeRef role = NULL, subrole = NULL;
        AXUIElementCopyAttributeValue(element, kAXRoleAttribute, &role);
        AXUIElementCopyAttributeValue(element, kAXSubroleAttribute, &subrole);
        copy_string(role, out->role, sizeof(out->role));
        copy_string(subrole, out->subrole, sizeof(out->subrole));
        if (role) CFRelease(role);
        if (subrole) CFRelease(subrole);
        CFTypeRef value = NULL;
        if (AXUIElementCopyAttributeValue(element, kAXValueAttribute, &value) == kAXErrorSuccess && value) {
            out->value_readable = CFGetTypeID(value) == CFStringGetTypeID() ? 1 : 0;
            CFRelease(value);
        }
        int secure = out->secure_input || strcmp(out->subrole, "AXSecureTextField") == 0;
        if (read_site && !secure) copy_site(element, out->site, sizeof(out->site));
        if (!secure) out->selection = (void *)selected_range(element);
        out->element = (void *)element; // retained; released by arion_release
        return ARION_TARGET_OK;
    }
}

int arion_frontmost_pid(void) {
    @autoreleasepool {
        NSRunningApplication *app = [[NSWorkspace sharedWorkspace] frontmostApplication];
        return app ? app.processIdentifier : 0;
    }
}

// arion_peek_target reads the focused field of the app with pid, which need
// not be frontmost any more, without keeping an element. It never reads a
// field's value and skips the site of a secure field.
int arion_peek_target(int pid, arion_target *out, int read_site) {
    memset(out, 0, sizeof(*out));
    if (!AXIsProcessTrusted()) return ARION_TARGET_UNTRUSTED;
    if (pid <= 0 || pid == getpid()) return ARION_TARGET_NONE;
    @autoreleasepool {
        NSRunningApplication *app = [NSRunningApplication runningApplicationWithProcessIdentifier:pid];
        if (!app) return ARION_TARGET_NONE;
        out->pid = pid;
        if (app.bundleIdentifier) strlcpy(out->bundle, app.bundleIdentifier.UTF8String, sizeof(out->bundle));
        if (app.localizedName) strlcpy(out->app_name, app.localizedName.UTF8String, sizeof(out->app_name));
        out->secure_input = IsSecureEventInputEnabled() ? 1 : 0;
        AXUIElementRef application = AXUIElementCreateApplication(pid);
        AXUIElementSetMessagingTimeout(application, 0.5f);
        CFTypeRef element = NULL;
        AXError error = AXUIElementCopyAttributeValue(application, kAXFocusedUIElementAttribute, &element);
        CFRelease(application);
        if (error != kAXErrorSuccess || !element) return ARION_TARGET_OK;
        AXUIElementSetMessagingTimeout((AXUIElementRef)element, 0.5f);
        CFTypeRef role = NULL, subrole = NULL;
        AXUIElementCopyAttributeValue((AXUIElementRef)element, kAXRoleAttribute, &role);
        AXUIElementCopyAttributeValue((AXUIElementRef)element, kAXSubroleAttribute, &subrole);
        copy_string(role, out->role, sizeof(out->role));
        copy_string(subrole, out->subrole, sizeof(out->subrole));
        if (role) CFRelease(role);
        if (subrole) CFRelease(subrole);
        if (read_site && !out->secure_input && strcmp(out->subrole, "AXSecureTextField") != 0) copy_site((AXUIElementRef)element, out->site, sizeof(out->site));
        CFRelease(element);
        return ARION_TARGET_OK;
    }
}

void *arion_retain(void *element) {
    return element ? (void *)CFRetain((CFTypeRef)element) : NULL;
}

void arion_release(void *element) {
    if (element) CFRelease((CFTypeRef)element);
}

int arion_focus_matches(void *element, int pid) {
    @autoreleasepool {
        NSRunningApplication *app = [[NSWorkspace sharedWorkspace] frontmostApplication];
        int error = 0;
        AXUIElementRef current = focused_element(&error);
        if (!element) {
            if (current) CFRelease(current);
            return app && app.processIdentifier == pid;
        }
        if (!current) return 0;
        int same = app && app.processIdentifier == pid && CFEqual(current, (CFTypeRef)element);
        CFRelease(current);
        return same;
    }
}

int arion_secure_input(void) { return IsSecureEventInputEnabled() ? 1 : 0; }

// Input source registration for Arion's IMK bundle (IME-01): explicit,
// user-initiated install and removal.
static TISInputSourceRef arion_find_source(const char *source_id) {
    NSDictionary *filter = @{(__bridge NSString *)kTISPropertyInputSourceID: [NSString stringWithUTF8String:source_id]};
    CFArrayRef list = TISCreateInputSourceList((__bridge CFDictionaryRef)filter, true);
    if (!list) return NULL;
    TISInputSourceRef source = NULL;
    if (CFArrayGetCount(list) > 0) {
        source = (TISInputSourceRef)CFRetain(CFArrayGetValueAtIndex(list, 0));
    }
    CFRelease(list);
    return source;
}

int arion_ime_bundle_matches(const char *bundle_path, const char *source_id) {
    @autoreleasepool {
        NSString *path = [NSString stringWithUTF8String:bundle_path];
        NSData *data = [NSData dataWithContentsOfFile:[path stringByAppendingPathComponent:@"Contents/Info.plist"]];
        if (!data) return 0;
        id plist = [NSPropertyListSerialization propertyListWithData:data options:0 format:NULL error:NULL];
        if (![plist isKindOfClass:NSDictionary.class]) return 0;
        NSString *identifier = [NSString stringWithUTF8String:source_id];
        id component = plist[@"ComponentInputModeDict"];
        if (![component isKindOfClass:NSDictionary.class]) return 0;
        id modes = component[@"tsInputModeListKey"];
        if (![modes isKindOfClass:NSDictionary.class]) return 0;
        id mode = modes[identifier];
        return [mode isKindOfClass:NSDictionary.class] && [mode[@"TISInputSourceID"] isEqual:identifier];
    }
}

// This validates Developer ID and entitlement consistency, not a fixed
// publisher. Release packaging owns publisher/profile authorization checks.
int arion_ime_validate_payload(const char *bundle_path, const char *bundle_id) {
    @autoreleasepool {
        NSURL *url = [NSURL fileURLWithPath:[NSString stringWithUTF8String:bundle_path]];
        NSString *identifier = [NSString stringWithUTF8String:bundle_id];
        NSCharacterSet *invalidIdentifier = [[NSCharacterSet characterSetWithCharactersInString:@"abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789.-"] invertedSet];
        if (identifier.length == 0 || [identifier rangeOfCharacterFromSet:invalidIdentifier].location != NSNotFound) return errSecParam;
        SecStaticCodeRef code = NULL;
        OSStatus status = SecStaticCodeCreateWithPath((__bridge CFURLRef)url, kSecCSDefaultFlags, &code);
        if (status != errSecSuccess) return (int)status;
        SecRequirementRef requirement = NULL;
        @try {
            NSString *rule = [NSString stringWithFormat:@"identifier \"%@\" and anchor apple generic and certificate 1[field.1.2.840.113635.100.6.2.6] exists and certificate leaf[field.1.2.840.113635.100.6.1.13] exists", identifier];
            status = SecRequirementCreateWithString((__bridge CFStringRef)rule, kSecCSDefaultFlags, &requirement);
            if (status != errSecSuccess) return (int)status;
            status = SecStaticCodeCheckValidity(code, kSecCSStrictValidate | kSecCSCheckAllArchitectures | kSecCSCheckNestedCode, requirement);
            if (status != errSecSuccess) return (int)status;
            CFDictionaryRef info = NULL;
            status = SecCodeCopySigningInformation(code, kSecCSSigningInformation, &info);
            if (status != errSecSuccess) return (int)status;
            NSDictionary *signing = CFBridgingRelease(info);
            NSString *team = signing[(__bridge id)kSecCodeInfoTeamIdentifier];
            NSDictionary *entitlements = signing[(__bridge id)kSecCodeInfoEntitlementsDict];
            if (![team isKindOfClass:NSString.class] || team.length == 0 ||
                ![signing[(__bridge id)kSecCodeInfoIdentifier] isEqual:identifier] ||
                ![entitlements isKindOfClass:NSDictionary.class] ||
                ![entitlements[@"com.apple.developer.team-identifier"] isEqual:team] ||
                ![entitlements[@"com.apple.application-identifier"] isEqual:[NSString stringWithFormat:@"%@.%@", team, identifier]]) return errSecCSReqFailed;
            // An explicit earlier group would change the default DP key group.
            id groups = entitlements[@"keychain-access-groups"];
            if (groups && ![groups isEqual:@[[NSString stringWithFormat:@"%@.%@", team, identifier]]]) return errSecCSReqFailed;
            NSString *metadata = [[url URLByAppendingPathComponent:@"Contents/Info.plist"] path];
            NSDictionary *plist = [NSDictionary dictionaryWithContentsOfFile:metadata];
            if (![plist[@"CFBundleIdentifier"] isEqual:identifier]) return errSecCSReqFailed;
            NSURL *profile = [url URLByAppendingPathComponent:@"Contents/embedded.provisionprofile"];
            NSNumber *regular = nil;
            if (![profile getResourceValue:&regular forKey:NSURLIsRegularFileKey error:nil] || !regular.boolValue ||
                [[NSFileManager.defaultManager attributesOfItemAtPath:profile.path error:nil][NSFileSize] unsignedLongLongValue] == 0) return errSecCSReqFailed;
            return errSecSuccess;
        } @finally {
            if (requirement) CFRelease(requirement);
            CFRelease(code);
        }
    }
}

// TextInputSources.h requires UI applications to call TIS on the main
// thread. Installation copies files on a worker; marshal only native work.
// Non-UI command-line clients may use TIS from serialized worker threads.
static int arion_ime_on_main(int (^operation)(void)) {
    if (!NSApp) {
        static NSRecursiveLock *lock;
        static dispatch_once_t once;
        dispatch_once(&once, ^{ lock = [NSRecursiveLock new]; });
        [lock lock];
        @try { return operation(); }
        @finally { [lock unlock]; }
    }
    if ([NSThread isMainThread]) return operation();
    __block int result;
    dispatch_sync(dispatch_get_main_queue(), ^{ result = operation(); });
    return result;
}

int arion_ime_register(const char *bundle_path) {
    return arion_ime_on_main(^int(void) {
        @autoreleasepool {
            NSURL *url = [NSURL fileURLWithPath:[NSString stringWithUTF8String:bundle_path]];
            // Register the completed installed bundle before TIS resolves its ID.
            // LaunchServices can otherwise retain a source-tree build in progress.
            OSStatus status = LSRegisterURL((__bridge CFURLRef)url, true);
            if (status != noErr) return (int)status;
            return (int)TISRegisterInputSource((__bridge CFURLRef)url);
        }
    });
}

// Input modes may retain an enabled flag while their parent is disabled.
// The Text Input Sources contract requires enabling the parent first.
static TISInputSourceRef arion_ime_parent(TISInputSourceRef source) {
    CFStringRef type = TISGetInputSourceProperty(source, kTISPropertyInputSourceType);
    if (!type || !CFEqual(type, kTISTypeKeyboardInputMode)) return NULL;
    CFStringRef bundle = TISGetInputSourceProperty(source, kTISPropertyBundleID);
    if (!bundle) return NULL;
    return arion_find_source([(__bridge NSString *)bundle UTF8String]);
}

int arion_ime_enable(const char *source_id) {
    return arion_ime_on_main(^int(void) {
        TISInputSourceRef source = arion_find_source(source_id);
        if (!source) return -1;
        TISInputSourceRef parent = arion_ime_parent(source);
        CFStringRef type = TISGetInputSourceProperty(source, kTISPropertyInputSourceType);
        if (type && CFEqual(type, kTISTypeKeyboardInputMode) && !parent) {
            CFRelease(source);
            return -1;
        }
        OSStatus status = parent ? TISEnableInputSource(parent) : noErr;
        if (status == noErr) status = TISEnableInputSource(source);
        if (parent) CFRelease(parent);
        CFRelease(source);
        return (int)status;
    });
}

// arion_ime_state: 0 absent, 1 registered, 2 enabled, 3 selected.
int arion_ime_state(const char *source_id) {
    return arion_ime_on_main(^int(void) {
        TISInputSourceRef source = arion_find_source(source_id);
        if (!source) return 0;
        int state = 1;
        CFBooleanRef enabled = TISGetInputSourceProperty(source, kTISPropertyInputSourceIsEnabled);
        CFBooleanRef selected = TISGetInputSourceProperty(source, kTISPropertyInputSourceIsSelected);
        TISInputSourceRef parent = arion_ime_parent(source);
        CFBooleanRef parentEnabled = parent ? TISGetInputSourceProperty(parent, kTISPropertyInputSourceIsEnabled) : NULL;
        if (enabled && CFBooleanGetValue(enabled) && (!parent || (parentEnabled && CFBooleanGetValue(parentEnabled)))) state = 2;
        if (parent) CFRelease(parent);
        if (selected && CFBooleanGetValue(selected)) state = 3;
        CFRelease(source);
        return state;
    });
}

static int arion_ime_source_flags(TISInputSourceRef source, CFStringRef expected) {
    if (!source) return 0;
    int flags = 1;
    CFStringRef identity = TISGetInputSourceProperty(source, kTISPropertyInputSourceID);
    if (identity && CFEqual(identity, expected)) flags |= 16;
    const CFStringRef keys[] = {kTISPropertyInputSourceIsEnabled, kTISPropertyInputSourceIsSelected, kTISPropertyInputSourceIsSelectCapable};
    const int bits[] = {2, 4, 8};
    for (int i = 0; i < 3; i++) {
        CFBooleanRef value = TISGetInputSourceProperty(source, keys[i]);
        if (value && CFGetTypeID(value) == CFBooleanGetTypeID() && CFBooleanGetValue(value)) flags |= bits[i];
    }
    return flags;
}

void arion_ime_inspect(const char *source_id, const char *parent_id, arion_ime_snapshot *out) {
    arion_ime_on_main(^int(void) {
        @autoreleasepool {
            NSString *modeID = [NSString stringWithUTF8String:source_id];
            NSString *parentID = [NSString stringWithUTF8String:parent_id];
            TISInputSourceRef mode = arion_find_source(source_id);
            TISInputSourceRef parent = arion_find_source(parent_id);
            out->mode_flags = arion_ime_source_flags(mode, (__bridge CFStringRef)modeID);
            out->parent_flags = arion_ime_source_flags(parent, (__bridge CFStringRef)parentID);
            out->selected_kind = 0;
            TISInputSourceRef current = TISCopyCurrentKeyboardInputSource();
            if (current) {
                CFStringRef identifier = TISGetInputSourceProperty(current, kTISPropertyInputSourceID);
                if (identifier) out->selected_kind = CFEqual(identifier, (__bridge CFStringRef)modeID) ? 1 :
                    CFEqual(identifier, (__bridge CFStringRef)parentID) ? 2 : 3;
                CFRelease(current);
            }
            if (mode) CFRelease(mode);
            if (parent) CFRelease(parent);
            return 0;
        }
    });
}

int arion_ime_open_settings(void) {
    return arion_ime_on_main(^int(void) {
        @autoreleasepool {
            NSURL *target = [NSURL URLWithString:@"x-apple.systempreferences:com.apple.preference.keyboard?InputSources"];
            return [[NSWorkspace sharedWorkspace] openURL:target] ? 1 : 0;
        }
    });
}

int arion_ime_select(const char *source_id) {
    return arion_ime_on_main(^int(void) {
        TISInputSourceRef source = arion_find_source(source_id);
        if (!source) return paramErr;
        OSStatus result = TISSelectInputSource(source);
        CFRelease(source);
        return (int)result;
    });
}

int arion_ime_switch_away(const char *source_id) {
    return arion_ime_on_main(^int(void) {
        TISInputSourceRef source = arion_find_source(source_id);
        if (!source) return noErr;
        CFBooleanRef selected = TISGetInputSourceProperty(source, kTISPropertyInputSourceIsSelected);
        BOOL active = selected && CFBooleanGetValue(selected);
        CFRelease(source);
        if (!active) return noErr;
        TISInputSourceRef replacement = TISCopyCurrentASCIICapableKeyboardLayoutInputSource();
        if (!replacement) return paramErr;
        OSStatus result = TISSelectInputSource(replacement);
        CFRelease(replacement);
        return (int)result;
    });
}

int arion_ime_processes(const char *bundle_id, int action) {
    return arion_ime_on_main(^int(void) {
        @autoreleasepool {
            NSString *identifier = [NSString stringWithUTF8String:bundle_id];
            int count = 0;
            for (NSRunningApplication *app in [NSRunningApplication runningApplicationsWithBundleIdentifier:identifier]) {
                if (app.terminated) continue;
                count++;
                if (action == 1) [app terminate];
                if (action == 2 && ![app forceTerminate]) return -1;
            }
            return count;
        }
    });
}


// The observer belongs to one explicit setup attempt. Cancellation is checked
// on the main thread immediately before selection, including queued requests.
@interface ArionInputSourceWatch : NSObject
@property(atomic) BOOL cancelled;
@property(nonatomic, strong) dispatch_semaphore_t signal;
@end
@implementation ArionInputSourceWatch
@end

static void arion_ime_source_changed(CFNotificationCenterRef center, void *observer, CFStringRef name, const void *object, CFDictionaryRef info) {
    ArionInputSourceWatch *watch = (__bridge ArionInputSourceWatch *)observer;
    dispatch_semaphore_signal(watch.signal);
}

void *arion_ime_watch_create(void) {
    __block void *result = NULL;
    arion_ime_on_main(^int(void) {
        ArionInputSourceWatch *watch = [ArionInputSourceWatch new];
        watch.signal = dispatch_semaphore_create(0);
        result = (__bridge_retained void *)watch;
        CFNotificationCenterRef center = CFNotificationCenterGetDistributedCenter();
        CFNotificationCenterAddObserver(center, result, arion_ime_source_changed, kTISNotifyEnabledKeyboardInputSourcesChanged, NULL, CFNotificationSuspensionBehaviorDeliverImmediately);
        CFNotificationCenterAddObserver(center, result, arion_ime_source_changed, kTISNotifySelectedKeyboardInputSourceChanged, NULL, CFNotificationSuspensionBehaviorDeliverImmediately);
        return 0;
    });
    return result;
}

void arion_ime_watch_cancel(void *value) {
    ArionInputSourceWatch *watch = (__bridge ArionInputSourceWatch *)value;
    watch.cancelled = YES;
    dispatch_semaphore_signal(watch.signal);
}

void arion_ime_watch_wait(void *value, int milliseconds) {
    ArionInputSourceWatch *watch = (__bridge ArionInputSourceWatch *)value;
    dispatch_semaphore_wait(watch.signal, dispatch_time(DISPATCH_TIME_NOW, (int64_t)milliseconds * NSEC_PER_MSEC));
}

void arion_ime_watch_close(void *value) {
    arion_ime_on_main(^int(void) {
        CFNotificationCenterRemoveEveryObserver(CFNotificationCenterGetDistributedCenter(), value);
        ArionInputSourceWatch *watch = (__bridge_transfer ArionInputSourceWatch *)value;
        watch.cancelled = YES;
        return 0;
    });
}

int arion_ime_watch_select(void *value, const char *source_id) {
    return arion_ime_on_main(^int(void) {
        ArionInputSourceWatch *watch = (__bridge ArionInputSourceWatch *)value;
        if (watch.cancelled) return userCanceledErr;
        return arion_ime_select(source_id);
    });
}

char *arion_ime_removal_observation(void) {
    __block char *result = NULL;
    arion_ime_on_main(^int(void) {
        NSData *json = [NSJSONSerialization dataWithJSONObject:ArionSourceObservation() options:0 error:nil];
        if (json) result = strndup(json.bytes, json.length);
        return 0;
    });
    return result;
}

char *arion_ime_current_id(void) {
    __block char *result = NULL;
    arion_ime_on_main(^int(void) {
        TISInputSourceRef current = TISCopyCurrentKeyboardInputSource();
        if (current) {
            NSString *value = (__bridge NSString *)TISGetInputSourceProperty(current, kTISPropertyInputSourceID);
            if (value) result = strdup(value.UTF8String);
            CFRelease(current);
        }
        return 0;
    });
    return result;
}

int arion_ime_swap_supported(const char *path) {
    @autoreleasepool {
        NSNumber *supported = nil;
        NSURL *url = [NSURL fileURLWithPath:[NSString stringWithUTF8String:path]];
        if (![url getResourceValue:&supported forKey:NSURLVolumeSupportsSwapRenamingKey error:nil]) return 0;
        return supported.boolValue ? 1 : 0;
    }
}

int arion_ime_exchange(const char *first, const char *second, int existing) {
    @autoreleasepool {
        NSString *a = [NSString stringWithUTF8String:first];
        NSString *b = [NSString stringWithUTF8String:second];
        int aFD = open(a.stringByDeletingLastPathComponent.fileSystemRepresentation, O_RDONLY | O_DIRECTORY | O_NOFOLLOW | O_CLOEXEC);
        if (aFD < 0) return errno;
        int bFD = open(b.stringByDeletingLastPathComponent.fileSystemRepresentation, O_RDONLY | O_DIRECTORY | O_NOFOLLOW | O_CLOEXEC);
        if (bFD < 0) { int status = errno; close(aFD); return status; }
        unsigned int flags = (existing ? RENAME_SWAP : RENAME_EXCL) | RENAME_NOFOLLOW_ANY;
        int status = renameatx_np(aFD, a.lastPathComponent.fileSystemRepresentation,
                                 bFD, b.lastPathComponent.fileSystemRepresentation, flags) == 0 ? 0 : errno;
        close(aFD); close(bFD);
        return status;
    }
}

int arion_ime_maintenance_version(const char *path) {
    @autoreleasepool {
        NSString *metadata = [[NSString stringWithUTF8String:path] stringByAppendingPathComponent:@"Contents/Info.plist"];
        NSDictionary *info = [NSDictionary dictionaryWithContentsOfURL:[NSURL fileURLWithPath:metadata] error:nil];
        return [info[@"ArionMaintenanceProtocol"] intValue];
    }
}
