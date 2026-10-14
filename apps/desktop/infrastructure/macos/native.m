//go:build darwin

#import <AVFoundation/AVFoundation.h>
#import <ApplicationServices/ApplicationServices.h>
#import <Carbon/Carbon.h>
#import <Cocoa/Cocoa.h>
#include "native.h"

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

static int is_role(AXUIElementRef element, CFStringRef wanted) {
    CFTypeRef role = NULL;
    int match = AXUIElementCopyAttributeValue(element, kAXRoleAttribute, &role) == kAXErrorSuccess && role &&
                CFGetTypeID(role) == CFStringGetTypeID() && CFStringCompare((CFStringRef)role, wanted, 0) == kCFCompareEqualTo;
    if (role) CFRelease(role);
    return match;
}

// copy_site writes the host of the top-level web page containing element:
// the outermost AXWebArea ancestor's AXURL. Only http(s) hosts are kept;
// the path, query and page title are never read.
static void copy_site(AXUIElementRef element, char *out, size_t size) {
    out[0] = 0;
    AXUIElementRef current = (AXUIElementRef)CFRetain(element);
    for (int depth = 0; current && depth < 64; depth++) {
        if (is_role(current, CFSTR("AXWebArea"))) {
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
        AXError error = AXUIElementCopyAttributeValue(current, kAXParentAttribute, &parent);
        CFRelease(current);
        current = error == kAXErrorSuccess ? (AXUIElementRef)parent : NULL;
    }
    if (current) CFRelease(current);
}

int arion_capture_target(arion_target *out, int read_site) {
    memset(out, 0, sizeof(*out));
    if (!AXIsProcessTrusted()) return ARION_TARGET_UNTRUSTED;
    @autoreleasepool {
        NSRunningApplication *app = [[NSWorkspace sharedWorkspace] frontmostApplication];
        if (!app) return ARION_TARGET_NONE;
        out->pid = app.processIdentifier;
        if (app.bundleIdentifier) strlcpy(out->bundle, app.bundleIdentifier.UTF8String, sizeof(out->bundle));
        if (app.bundleURL) strlcpy(out->bundle_path, app.bundleURL.fileSystemRepresentation, sizeof(out->bundle_path));
        if (app.localizedName) strlcpy(out->app_name, app.localizedName.UTF8String, sizeof(out->app_name));
        out->secure_input = IsSecureEventInputEnabled() ? 1 : 0;
        int error = 0;
        AXUIElementRef element = focused_element(&error);
        if (!element) return ARION_TARGET_OK; // app-level target; safety unknown
        pid_t owner = 0;
        AXUIElementGetPid(element, &owner);
        if (owner != app.processIdentifier) {
            // Focus belongs to another process (e.g. a panel); keep it.
            out->pid = owner;
        }
        CFTypeRef role = NULL, subrole = NULL;
        AXUIElementCopyAttributeValue(element, kAXRoleAttribute, &role);
        AXUIElementCopyAttributeValue(element, kAXSubroleAttribute, &subrole);
        copy_string(role, out->role, sizeof(out->role));
        copy_string(subrole, out->subrole, sizeof(out->subrole));
        if (role) CFRelease(role);
        if (subrole) CFRelease(subrole);
        Boolean settable = false;
        if (AXUIElementIsAttributeSettable(element, kAXSelectedTextAttribute, &settable) == kAXErrorSuccess) {
            out->selected_text_settable = settable ? 1 : 0;
        }
        CFTypeRef value = NULL;
        if (AXUIElementCopyAttributeValue(element, kAXValueAttribute, &value) == kAXErrorSuccess && value) {
            out->value_readable = CFGetTypeID(value) == CFStringGetTypeID() ? 1 : 0;
            CFRelease(value);
        }
        if (read_site) copy_site(element, out->site, sizeof(out->site));
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
        if (read_site && strcmp(out->subrole, "AXSecureTextField") != 0) copy_site((AXUIElementRef)element, out->site, sizeof(out->site));
        CFRelease(element);
        return ARION_TARGET_OK;
    }
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
        int same = CFEqual(current, (CFTypeRef)element) ? 1 : 0;
        CFRelease(current);
        return same;
    }
}

int arion_secure_input(void) { return IsSecureEventInputEnabled() ? 1 : 0; }

long arion_value_length(void *element) {
    CFTypeRef value = NULL;
    if (!element || AXUIElementCopyAttributeValue((AXUIElementRef)element, kAXValueAttribute, &value) != kAXErrorSuccess || !value) return -1;
    long length = -1;
    if (CFGetTypeID(value) == CFStringGetTypeID()) length = (long)CFStringGetLength((CFStringRef)value);
    CFRelease(value);
    return length;
}

int arion_ax_insert(void *element, const char *utf8) {
    @autoreleasepool {
        NSString *text = [NSString stringWithUTF8String:utf8];
        if (!text) return -1;
        return (int)AXUIElementSetAttributeValue((AXUIElementRef)element, kAXSelectedTextAttribute, (__bridge CFStringRef)text);
    }
}

// arion_type_unicode posts the text as keyboard Unicode events in short
// chunks. Callers remove line breaks first: Return is never synthesized.
int arion_type_unicode(const char *utf8) {
    @autoreleasepool {
        NSString *text = [NSString stringWithUTF8String:utf8];
        if (!text) return -1;
        CGEventSourceRef source = CGEventSourceCreate(kCGEventSourceStateHIDSystemState);
        NSUInteger length = text.length, offset = 0;
        while (offset < length) {
            NSRange range = [text rangeOfComposedCharacterSequencesForRange:NSMakeRange(offset, MIN((NSUInteger)16, length - offset))];
            unichar buffer[32];
            NSUInteger count = MIN(range.length, (NSUInteger)32);
            [text getCharacters:buffer range:NSMakeRange(range.location, count)];
            CGEventRef down = CGEventCreateKeyboardEvent(source, 0, true);
            CGEventRef up = CGEventCreateKeyboardEvent(source, 0, false);
            if (!down || !up) {
                if (down) CFRelease(down);
                if (up) CFRelease(up);
                if (source) CFRelease(source);
                return -2;
            }
            CGEventSetFlags(down, 0);
            CGEventSetFlags(up, 0);
            CGEventKeyboardSetUnicodeString(down, (UniCharCount)count, buffer);
            CGEventKeyboardSetUnicodeString(up, (UniCharCount)count, buffer);
            CGEventPost(kCGAnnotatedSessionEventTap, down);
            CGEventPost(kCGAnnotatedSessionEventTap, up);
            CFRelease(down);
            CFRelease(up);
            offset = range.location + count;
            usleep(4000);
        }
        if (source) CFRelease(source);
        return 0;
    }
}

static NSArray<NSPasteboardItem *> *saved_items = nil;
static NSInteger written_change = -1;

// arion_paste saves the clipboard, writes the text and posts Command-V. It
// returns the change count Arion owns for the later restore check.
long arion_paste(const char *utf8) {
    @autoreleasepool {
        NSPasteboard *board = [NSPasteboard generalPasteboard];
        NSMutableArray<NSPasteboardItem *> *copies = [NSMutableArray array];
        for (NSPasteboardItem *item in board.pasteboardItems) {
            NSPasteboardItem *copy = [[NSPasteboardItem alloc] init];
            for (NSPasteboardType type in item.types) {
                NSData *data = [item dataForType:type];
                if (data) [copy setData:data forType:type];
            }
            [copies addObject:copy];
        }
        NSString *text = [NSString stringWithUTF8String:utf8];
        if (!text) return -1;
        saved_items = copies;
        [board clearContents];
        [board setString:text forType:NSPasteboardTypeString];
        written_change = board.changeCount;
        CGEventSourceRef source = CGEventSourceCreate(kCGEventSourceStateHIDSystemState);
        CGEventRef down = CGEventCreateKeyboardEvent(source, (CGKeyCode)kVK_ANSI_V, true);
        CGEventRef up = CGEventCreateKeyboardEvent(source, (CGKeyCode)kVK_ANSI_V, false);
        CGEventSetFlags(down, kCGEventFlagMaskCommand);
        CGEventSetFlags(up, kCGEventFlagMaskCommand);
        CGEventPost(kCGAnnotatedSessionEventTap, down);
        CGEventPost(kCGAnnotatedSessionEventTap, up);
        CFRelease(down);
        CFRelease(up);
        if (source) CFRelease(source);
        return (long)written_change;
    }
}

// arion_restore_clipboard restores the saved items only when nobody wrote
// the clipboard since Arion did (DSK-A05). Returns 1 when restored.
int arion_restore_clipboard(long owned_change) {
    @autoreleasepool {
        NSPasteboard *board = [NSPasteboard generalPasteboard];
        int restored = 0;
        if (board.changeCount == owned_change && saved_items) {
            [board clearContents];
            if (saved_items.count > 0) [board writeObjects:saved_items];
            restored = 1;
        }
        saved_items = nil;
        return restored;
    }
}

long arion_clipboard_change_count(void) { return (long)[NSPasteboard generalPasteboard].changeCount; }

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

int arion_ime_register(const char *bundle_path) {
    @autoreleasepool {
        NSURL *url = [NSURL fileURLWithPath:[NSString stringWithUTF8String:bundle_path]];
        return (int)TISRegisterInputSource((__bridge CFURLRef)url);
    }
}

int arion_ime_enable(const char *source_id) {
    TISInputSourceRef source = arion_find_source(source_id);
    if (!source) return -1;
    OSStatus status = TISEnableInputSource(source);
    CFRelease(source);
    return (int)status;
}

int arion_ime_disable(const char *source_id) {
    TISInputSourceRef source = arion_find_source(source_id);
    if (!source) return 0;
    OSStatus status = TISDisableInputSource(source);
    CFRelease(source);
    return (int)status;
}

// arion_ime_state: 0 absent, 1 registered, 2 enabled, 3 selected.
int arion_ime_state(const char *source_id) {
    TISInputSourceRef source = arion_find_source(source_id);
    if (!source) return 0;
    int state = 1;
    CFBooleanRef enabled = TISGetInputSourceProperty(source, kTISPropertyInputSourceIsEnabled);
    CFBooleanRef selected = TISGetInputSourceProperty(source, kTISPropertyInputSourceIsSelected);
    if (enabled && CFBooleanGetValue(enabled)) state = 2;
    if (selected && CFBooleanGetValue(selected)) state = 3;
    CFRelease(source);
    return state;
}
