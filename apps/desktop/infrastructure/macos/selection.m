//go:build darwin

#import <ApplicationServices/ApplicationServices.h>
#import <Carbon/Carbon.h>
#import <Foundation/Foundation.h>
#include <math.h>
#include <pthread.h>
#include <stdlib.h>
#include <string.h>
#include "selection.h"
#include "_cgo_export.h"

static CFMachPortRef selection_tap;
static CFRunLoopRef selection_loop;
static dispatch_semaphore_t selection_ready;
static pthread_t selection_worker;

// Observe only. The callback returns every event unchanged and does not
// read field text; a slow callback is re-enabled by macOS.
static CGEventRef selection_callback(CGEventTapProxy proxy, CGEventType type, CGEventRef event, void *info) {
    (void)proxy;
    (void)info;
    if (type == kCGEventTapDisabledByTimeout || type == kCGEventTapDisabledByUserInput) {
        if (selection_tap && CGPreflightListenEventAccess()) CGEventTapEnable(selection_tap, true);
        return event;
    }
    if (type == kCGEventLeftMouseUp) arionSelectionMouseUp();
    return event;
}

static void *selection_thread(void *arg) {
    CFRunLoopSourceRef source = (CFRunLoopSourceRef)arg;
    selection_loop = CFRunLoopGetCurrent();
    CFRetain(selection_loop);
    CFRunLoopAddSource(selection_loop, source, kCFRunLoopCommonModes);
    CFRelease(source);
    CFRunLoopPerformBlock(selection_loop, kCFRunLoopCommonModes, ^{ dispatch_semaphore_signal(selection_ready); });
    CFRunLoopRun();
    return NULL;
}

int arion_selection_watch_start(void) {
    if (!CGPreflightListenEventAccess()) return ARION_SELECTION_DENIED;
    if (selection_tap) return CGEventTapIsEnabled(selection_tap) ? ARION_SELECTION_OK : ARION_SELECTION_UNAVAILABLE;
    CGEventMask mask = CGEventMaskBit(kCGEventLeftMouseUp);
    selection_tap = CGEventTapCreate(kCGSessionEventTap, kCGHeadInsertEventTap, kCGEventTapOptionListenOnly, mask, selection_callback, NULL);
    if (!selection_tap) return CGPreflightListenEventAccess() ? ARION_SELECTION_UNAVAILABLE : ARION_SELECTION_DENIED;
    if (!CGEventTapIsEnabled(selection_tap)) {
        CFMachPortInvalidate(selection_tap);
        CFRelease(selection_tap);
        selection_tap = NULL;
        return CGPreflightListenEventAccess() ? ARION_SELECTION_UNAVAILABLE : ARION_SELECTION_DENIED;
    }
    if (!selection_ready) selection_ready = dispatch_semaphore_create(0);
    CFRunLoopSourceRef source = CFMachPortCreateRunLoopSource(kCFAllocatorDefault, selection_tap, 0);
    if (!source) {
        CFMachPortInvalidate(selection_tap);
        CFRelease(selection_tap);
        selection_tap = NULL;
        return ARION_SELECTION_UNAVAILABLE;
    }
    if (pthread_create(&selection_worker, NULL, selection_thread, (void *)source) != 0) {
        CFRelease(source);
        CFMachPortInvalidate(selection_tap);
        CFRelease(selection_tap);
        selection_tap = NULL;
        return ARION_SELECTION_UNAVAILABLE;
    }
    dispatch_semaphore_wait(selection_ready, DISPATCH_TIME_FOREVER);
    return ARION_SELECTION_OK;
}

// web_area returns the innermost AXWebArea at or above element, retained.
static AXUIElementRef web_area(AXUIElementRef element) {
    AXUIElementRef current = (AXUIElementRef)CFRetain(element);
    for (int depth = 0; current && depth < 64; depth++) {
        AXUIElementSetMessagingTimeout(current, 0.2f);
        CFTypeRef role = NULL;
        AXUIElementCopyAttributeValue(current, kAXRoleAttribute, &role);
        int web = role && CFGetTypeID(role) == CFStringGetTypeID() && CFEqual(role, CFSTR("AXWebArea"));
        if (role) CFRelease(role);
        if (web) return current;
        CFTypeRef parent = NULL;
        AXError error = AXUIElementCopyAttributeValue(current, kAXParentAttribute, &parent);
        CFRelease(current);
        current = NULL;
        if (error == kAXErrorSuccess && parent && CFGetTypeID(parent) == AXUIElementGetTypeID()) {
            current = (AXUIElementRef)parent;
        } else if (parent) {
            CFRelease(parent);
        }
    }
    if (current) CFRelease(current);
    return NULL;
}

// web_selection returns the web area and its selected text marker range,
// both retained, or 0 when there is no web area or no selection.
static int web_selection(void *element, AXUIElementRef *area, CFTypeRef *range) {
    *area = NULL;
    *range = NULL;
    if (!element) return 0;
    AXUIElementRef web = web_area((AXUIElementRef)element);
    if (!web) return 0;
    AXUIElementSetMessagingTimeout(web, 0.3f);
    CFTypeRef markers = NULL;
    if (AXUIElementCopyAttributeValue(web, CFSTR("AXSelectedTextMarkerRange"), &markers) != kAXErrorSuccess || !markers) {
        if (markers) CFRelease(markers);
        CFRelease(web);
        return 0;
    }
    *area = web;
    *range = markers;
    return 1;
}

int arion_web_selection(void *element, long max, char **text) {
    *text = NULL;
    @autoreleasepool {
        AXUIElementRef area = NULL;
        CFTypeRef range = NULL;
        if (!web_selection(element, &area, &range)) return ARION_WEB_NONE;
        CFTypeRef value = NULL;
        AXError error = AXUIElementCopyParameterizedAttributeValue(area, CFSTR("AXStringForTextMarkerRange"), range, &value);
        CFRelease(range);
        CFRelease(area);
        if (error != kAXErrorSuccess || !value || CFGetTypeID(value) != CFStringGetTypeID()) {
            if (value) CFRelease(value);
            return ARION_WEB_FAILED;
        }
        NSString *selected = CFBridgingRelease(value);
        if (max < 0 || selected.length > (NSUInteger)max * 2) return ARION_WEB_FAILED;
        if (!selected.length) return ARION_WEB_NONE;
        *text = strdup(selected.UTF8String ?: "");
        return *text ? ARION_WEB_OK : ARION_WEB_FAILED;
    }
}

int arion_web_selection_bounds(void *element, double *x, double *y, double *width, double *height) {
    if (!AXIsProcessTrusted() || IsSecureEventInputEnabled()) return 0;
    @autoreleasepool {
        AXUIElementRef area = NULL;
        CFTypeRef range = NULL;
        if (!web_selection(element, &area, &range)) return 0;
        CFTypeRef bounds = NULL;
        AXError error = AXUIElementCopyParameterizedAttributeValue(area, CFSTR("AXBoundsForTextMarkerRange"), range, &bounds);
        CFRelease(range);
        CFRelease(area);
        CGRect frame = CGRectZero;
        int ok = error == kAXErrorSuccess && bounds && CFGetTypeID(bounds) == AXValueGetTypeID() &&
            AXValueGetValue((AXValueRef)bounds, kAXValueCGRectType, &frame);
        if (bounds) CFRelease(bounds);
        if (!ok || !isfinite(frame.origin.x) || !isfinite(frame.origin.y) || !isfinite(frame.size.width) ||
            !isfinite(frame.size.height) || frame.size.width < 0 || frame.size.height <= 0) return 0;
        *x = frame.origin.x; *y = frame.origin.y; *width = frame.size.width; *height = frame.size.height;
        return 1;
    }
}

void arion_selection_watch_stop(void) {
    if (!selection_tap) return;
    CFRunLoopPerformBlock(selection_loop, kCFRunLoopCommonModes, ^{
        CGEventTapEnable(selection_tap, false);
        CFMachPortInvalidate(selection_tap);
        CFRunLoopStop(selection_loop);
    });
    CFRunLoopWakeUp(selection_loop);
    pthread_join(selection_worker, NULL);
    CFRelease(selection_tap);
    selection_tap = NULL;
    CFRelease(selection_loop);
    selection_loop = NULL;
}
