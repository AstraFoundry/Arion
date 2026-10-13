//go:build darwin

#import <ApplicationServices/ApplicationServices.h>
#import <Foundation/Foundation.h>
#include <pthread.h>
#include "hold.h"
#include "_cgo_export.h"

static CFMachPortRef hold_tap;
static CFRunLoopRef hold_loop;
static dispatch_semaphore_t hold_ready;

int arion_listen_access(void) { return CGPreflightListenEventAccess() ? 1 : 0; }
int arion_listen_request(void) { return CGRequestListenEventAccess() ? 1 : 0; }

// The tap only observes: it forwards raw fields and returns every event
// unchanged. macOS disables a tap whose callback is slow; it is re-enabled.
static CGEventRef hold_callback(CGEventTapProxy proxy, CGEventType type, CGEventRef event, void *info) {
    (void)proxy;
    (void)info;
    if (type == kCGEventTapDisabledByTimeout || type == kCGEventTapDisabledByUserInput) {
        if (hold_tap) CGEventTapEnable(hold_tap, true);
        return event;
    }
    if (type == kCGEventKeyDown || type == kCGEventFlagsChanged) {
        arionHoldRaw((int)type, (long long)CGEventGetIntegerValueField(event, kCGKeyboardEventKeycode),
                     (unsigned long long)CGEventGetFlags(event),
                     (int)CGEventGetIntegerValueField(event, kCGKeyboardEventAutorepeat));
    }
    return event;
}

static void *hold_thread(void *arg) {
    CFRunLoopSourceRef source = (CFRunLoopSourceRef)arg;
    hold_loop = CFRunLoopGetCurrent();
    CFRetain(hold_loop);
    CFRunLoopAddSource(hold_loop, source, kCFRunLoopCommonModes);
    CFRelease(source);
    dispatch_semaphore_signal(hold_ready);
    CFRunLoopRun();
    return NULL;
}

int arion_hold_start(void) {
    if (hold_tap) return 0;
    CGEventMask mask = CGEventMaskBit(kCGEventKeyDown) | CGEventMaskBit(kCGEventFlagsChanged);
    hold_tap = CGEventTapCreate(kCGSessionEventTap, kCGHeadInsertEventTap, kCGEventTapOptionListenOnly, mask, hold_callback, NULL);
    if (!hold_tap) return -1;
    if (!hold_ready) hold_ready = dispatch_semaphore_create(0);
    CFRunLoopSourceRef source = CFMachPortCreateRunLoopSource(kCFAllocatorDefault, hold_tap, 0);
    pthread_t thread;
    if (pthread_create(&thread, NULL, hold_thread, (void *)source) != 0) {
        CFRelease(source);
        CFMachPortInvalidate(hold_tap);
        CFRelease(hold_tap);
        hold_tap = NULL;
        return -1;
    }
    pthread_detach(thread);
    // Stop needs the thread's run loop; wait until it exists.
    dispatch_semaphore_wait(hold_ready, DISPATCH_TIME_FOREVER);
    return 0;
}

void arion_hold_stop(void) {
    if (!hold_tap) return;
    CGEventTapEnable(hold_tap, false);
    CFMachPortInvalidate(hold_tap);
    CFRelease(hold_tap);
    hold_tap = NULL;
    CFRunLoopStop(hold_loop);
    CFRelease(hold_loop);
    hold_loop = NULL;
}
