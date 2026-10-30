//go:build darwin

#import <ApplicationServices/ApplicationServices.h>
#import <Foundation/Foundation.h>
#include <time.h>
#include <math.h>
#include "context.h"
#include "native.h"

static double ax_now(void) {
    struct timespec t;
    clock_gettime(CLOCK_MONOTONIC, &t);
    return t.tv_sec + t.tv_nsec / 1e9;
}

static id ax_attribute(AXUIElementRef element, CFStringRef key) {
    CFTypeRef value = NULL;
    AXUIElementCopyAttributeValue(element, key, &value);
    return value ? CFBridgingRelease(value) : nil;
}

static BOOL ax_frame(AXUIElementRef element, CGRect *frame) {
    id position = ax_attribute(element, kAXPositionAttribute);
    id size = ax_attribute(element, kAXSizeAttribute);
    CGPoint p;
    CGSize s;
    if (!position || !size || CFGetTypeID((__bridge CFTypeRef)position) != AXValueGetTypeID() ||
        CFGetTypeID((__bridge CFTypeRef)size) != AXValueGetTypeID() ||
        !AXValueGetValue((__bridge AXValueRef)position, kAXValueCGPointType, &p) ||
        !AXValueGetValue((__bridge AXValueRef)size, kAXValueCGSizeType, &s)) return NO;
    *frame = CGRectMake(p.x, p.y, s.width, s.height);
    return !CGRectIsEmpty(*frame);
}

// Collect only visible controls. Text areas require the provider's visible
// character range; reading their entire AXValue could scrape an open document.
static int ax_collect(AXUIElementRef element, CGRect clip, int depth, int *count,
                      double deadline, NSMutableArray<NSString *> *lines, NSUInteger *length) {
    if (ax_now() >= deadline) return ARION_CONTEXT_TIMEOUT;
    if (depth > 32 || ++*count > 512) return ARION_CONTEXT_FAILED;
    AXUIElementSetMessagingTimeout(element, (float)MIN(0.05, deadline - ax_now()));
    id subrole = ax_attribute(element, kAXSubroleAttribute);
    if ([subrole isEqual:(__bridge NSString *)kAXSecureTextFieldSubrole]) return ARION_CONTEXT_OK;
    id hidden = ax_attribute(element, CFSTR("AXHidden"));
    if ([hidden isKindOfClass:[NSNumber class]] && [hidden boolValue]) return ARION_CONTEXT_OK;
    CGRect frame;
    if (!ax_frame(element, &frame)) return ARION_CONTEXT_OK;
    CGRect visible = CGRectIntersection(clip, frame);
    if (CGRectIsEmpty(visible) || CGRectIsNull(visible)) return ARION_CONTEXT_OK;
    id role = ax_attribute(element, kAXRoleAttribute);
    NSString *text = nil;
    if ([role isEqual:(__bridge NSString *)kAXTextAreaRole] ||
        [role isEqual:(__bridge NSString *)kAXTextFieldRole] ||
        [role isEqual:(__bridge NSString *)kAXComboBoxRole]) {
        id range = ax_attribute(element, kAXVisibleCharacterRangeAttribute);
        if (range && CFGetTypeID((__bridge CFTypeRef)range) == AXValueGetTypeID()) {
            CFRange visibleRange;
            if (!AXValueGetValue((__bridge AXValueRef)range, kAXValueCFRangeType, &visibleRange) ||
                visibleRange.location < 0 || visibleRange.length < 0) return ARION_CONTEXT_FAILED;
            if ((NSUInteger)visibleRange.length > 32768 - *length) return ARION_CONTEXT_FAILED;
            CFTypeRef value = NULL;
            AXUIElementCopyParameterizedAttributeValue(element, kAXStringForRangeParameterizedAttribute,
                                                       (__bridge CFTypeRef)range, &value);
            if (value) {
                id result = CFBridgingRelease(value);
                if ([result isKindOfClass:[NSString class]]) text = result;
            }
        }
    } else if ([role isEqual:(__bridge NSString *)kAXStaticTextRole]) {
        id value = ax_attribute(element, kAXValueAttribute);
        if ([value isKindOfClass:[NSString class]]) text = value;
    }
    if (text.length) {
        if (*length + text.length > 32768) return ARION_CONTEXT_FAILED;
        [lines addObject:text];
        *length += text.length;
    }
    // Do not duplicate a text provider's contents through its child runs.
    if (text.length) return ax_now() < deadline ? ARION_CONTEXT_OK : ARION_CONTEXT_TIMEOUT;
    id children = ax_attribute(element, kAXChildrenAttribute);
    if (![children isKindOfClass:[NSArray class]]) return ARION_CONTEXT_OK;
    for (id child in children) {
        if (CFGetTypeID((__bridge CFTypeRef)child) != AXUIElementGetTypeID()) continue;
        int status = ax_collect((__bridge AXUIElementRef)child, visible, depth + 1, count, deadline, lines, length);
        if (status != ARION_CONTEXT_OK) return status;
    }
    return ax_now() < deadline ? ARION_CONTEXT_OK : ARION_CONTEXT_TIMEOUT;
}

int arion_accessibility_window(void *element, int pid, double timeout_seconds, char **out, double *ms) {
    *out = NULL;
    *ms = 0;
    double start = ax_now();
    if (!AXIsProcessTrusted()) return ARION_CONTEXT_NO_PERMISSION;
    if (!element || timeout_seconds <= 0 || !arion_focus_matches(element, pid) || arion_secure_input()) return ARION_CONTEXT_NO_WINDOW;
    @autoreleasepool {
        AXUIElementRef target = (AXUIElementRef)element;
        AXUIElementSetMessagingTimeout(target, 0.05f);
        id window = ax_attribute(target, kAXWindowAttribute);
        if (!window || CFGetTypeID((__bridge CFTypeRef)window) != AXUIElementGetTypeID()) return ARION_CONTEXT_NO_WINDOW;
        AXUIElementSetMessagingTimeout((__bridge AXUIElementRef)window, 0.05f);
        CGRect frame;
        if (!ax_frame((__bridge AXUIElementRef)window, &frame)) return ARION_CONTEXT_FAILED;
        int count = 0;
        NSUInteger length = 0;
        NSMutableArray<NSString *> *lines = [NSMutableArray array];
        int status = ax_collect((__bridge AXUIElementRef)window, frame, 0, &count, start + timeout_seconds, lines, &length);
        *ms = (ax_now() - start) * 1000;
        if (!arion_focus_matches(element, pid) || arion_secure_input()) return ARION_CONTEXT_NO_WINDOW;
        if (status != ARION_CONTEXT_OK) return status;
        if (!lines.count) return ARION_CONTEXT_UNSUPPORTED;
        *out = strdup([lines componentsJoinedByString:@"\n"].UTF8String);
        return ARION_CONTEXT_OK;
    }
}

// Position the history picker using the selection geometry when supported,
// otherwise the editable field's own frame. Neither path reads text content.
int arion_history_anchor(void *element, double *x, double *y, double *width, double *height) {
    if (!AXIsProcessTrusted() || !element || arion_secure_input()) return 0;
    @autoreleasepool {
        AXUIElementRef field = (AXUIElementRef)element;
        AXUIElementSetMessagingTimeout(field, 0.2f);
        id role = ax_attribute(field, kAXRoleAttribute);
        id subrole = ax_attribute(field, kAXSubroleAttribute);
        if ([subrole isEqual:(__bridge NSString *)kAXSecureTextFieldSubrole]) return 0;
        id selection = ax_attribute(field, kAXSelectedTextRangeAttribute);
        BOOL editable = [role isEqual:(__bridge NSString *)kAXTextFieldRole] ||
            [role isEqual:(__bridge NSString *)kAXTextAreaRole] ||
            [role isEqual:(__bridge NSString *)kAXComboBoxRole];
        if (!editable) return 0;
        CGRect frame = CGRectZero;
        if (selection && CFGetTypeID((__bridge CFTypeRef)selection) == AXValueGetTypeID()) {
            CFTypeRef bounds = NULL;
            if (AXUIElementCopyParameterizedAttributeValue(field, kAXBoundsForRangeParameterizedAttribute,
                    (__bridge CFTypeRef)selection, &bounds) == kAXErrorSuccess && bounds) {
                if (CFGetTypeID(bounds) == AXValueGetTypeID()) AXValueGetValue((AXValueRef)bounds, kAXValueCGRectType, &frame);
                CFRelease(bounds);
            }
        }
        if (frame.size.height <= 0 && (!editable || !ax_frame(field, &frame))) return 0;
        if (!isfinite(frame.origin.x) || !isfinite(frame.origin.y) || !isfinite(frame.size.width) ||
            !isfinite(frame.size.height) || frame.size.width < 0 || frame.size.height <= 0) return 0;
        *x = frame.origin.x; *y = frame.origin.y; *width = frame.size.width; *height = frame.size.height;
        return 1;
    }
}

// A positive no-input observation may choose copy-only dictation before an
// IME session starts. An unavailable provider never authorizes that route.
int arion_no_input(void) {
    if (!AXIsProcessTrusted() || arion_secure_input()) return 0;
    int secure = 0, absent = 0;
    void *focused = arion_history_focus(arion_frontmost_pid(), &secure, &absent);
    if (!focused) return absent;
    @autoreleasepool {
        AXUIElementRef field = (AXUIElementRef)focused;
        id role = ax_attribute(field, kAXRoleAttribute);
        id selection = ax_attribute(field, kAXSelectedTextRangeAttribute);
        BOOL input = [role isEqual:(__bridge NSString *)kAXTextFieldRole] ||
            [role isEqual:(__bridge NSString *)kAXTextAreaRole] ||
            [role isEqual:(__bridge NSString *)kAXComboBoxRole] || selection != nil;
        int noInput = !secure && [role isKindOfClass:NSString.class] && [(NSString *)role length] > 0 && !input;
        CFRelease(focused);
        return noInput;
    }
}
