//go:build darwin

#import <ApplicationServices/ApplicationServices.h>
#import <Foundation/Foundation.h>
#include <pthread.h>
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
