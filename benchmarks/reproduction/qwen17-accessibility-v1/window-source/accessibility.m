//go:build darwin

#import <ApplicationServices/ApplicationServices.h>
#import <Foundation/Foundation.h>
#include <time.h>
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
    if ([role isEqual:(__bridge NSString *)kAXTextAreaRole]) {
        id range = ax_attribute(element, kAXVisibleCharacterRangeAttribute);
        if (range && CFGetTypeID((__bridge CFTypeRef)range) == AXValueGetTypeID()) {
            CFTypeRef value = NULL;
            AXUIElementCopyParameterizedAttributeValue(element, kAXStringForRangeParameterizedAttribute,
                                                       (__bridge CFTypeRef)range, &value);
            if (value) {
                id result = CFBridgingRelease(value);
                if ([result isKindOfClass:[NSString class]]) text = result;
            }
        }
    } else if ([role isEqual:(__bridge NSString *)kAXStaticTextRole] ||
               [role isEqual:(__bridge NSString *)kAXTextFieldRole]) {
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
