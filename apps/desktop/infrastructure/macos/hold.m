//go:build darwin

#import <ApplicationServices/ApplicationServices.h>
#import <Foundation/Foundation.h>
#include <pthread.h>
#include "hold.h"
#include "_cgo_export.h"

static CFMachPortRef hold_tap;
static CFRunLoopRef hold_loop;
static dispatch_semaphore_t hold_ready;
static pthread_t hold_worker;

int arion_listen_access(void) { return CGPreflightListenEventAccess() ? 1 : 0; }
int arion_listen_request(void) { return CGRequestListenEventAccess() ? 1 : 0; }

// The tap only observes: it forwards raw fields and returns every event
// unchanged. macOS disables a tap whose callback is slow; it is re-enabled.
static CGEventRef hold_callback(CGEventTapProxy proxy, CGEventType type, CGEventRef event, void *info) {
    (void)proxy;
    (void)info;
    if (type == kCGEventTapDisabledByTimeout || type == kCGEventTapDisabledByUserInput) {
        if (hold_tap && CGPreflightListenEventAccess()) CGEventTapEnable(hold_tap, true);
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
    // Publish readiness from inside the running loop, so an immediate stop
    // cannot race the first CFRunLoopRun invocation.
    CFRunLoopPerformBlock(hold_loop, kCFRunLoopCommonModes, ^{ dispatch_semaphore_signal(hold_ready); });
    CFRunLoopRun();
    return NULL;
}

int arion_hold_start(void) {
    if (!CGPreflightListenEventAccess()) return ARION_HOLD_PERMISSION_DENIED;
    if (hold_tap) return CGEventTapIsEnabled(hold_tap) ? 0 : ARION_HOLD_UNAVAILABLE;
    CGEventMask mask = CGEventMaskBit(kCGEventKeyDown) | CGEventMaskBit(kCGEventFlagsChanged);
    hold_tap = CGEventTapCreate(kCGSessionEventTap, kCGHeadInsertEventTap, kCGEventTapOptionListenOnly, mask, hold_callback, NULL);
    if (!hold_tap) return CGPreflightListenEventAccess() ? ARION_HOLD_UNAVAILABLE : ARION_HOLD_PERMISSION_DENIED;
    if (!CGEventTapIsEnabled(hold_tap)) {
        CFMachPortInvalidate(hold_tap);
        CFRelease(hold_tap);
        hold_tap = NULL;
        return CGPreflightListenEventAccess() ? ARION_HOLD_UNAVAILABLE : ARION_HOLD_PERMISSION_DENIED;
    }
    if (!hold_ready) hold_ready = dispatch_semaphore_create(0);
    CFRunLoopSourceRef source = CFMachPortCreateRunLoopSource(kCFAllocatorDefault, hold_tap, 0);
    if (!source) {
        CFMachPortInvalidate(hold_tap);
        CFRelease(hold_tap);
        hold_tap = NULL;
        return ARION_HOLD_UNAVAILABLE;
    }
    if (pthread_create(&hold_worker, NULL, hold_thread, (void *)source) != 0) {
        CFRelease(source);
        CFMachPortInvalidate(hold_tap);
        CFRelease(hold_tap);
        hold_tap = NULL;
        return ARION_HOLD_UNAVAILABLE;
    }
    // Stop needs the thread's run loop; wait until it exists.
    dispatch_semaphore_wait(hold_ready, DISPATCH_TIME_FOREVER);
    return 0;
}

void arion_hold_stop(void) {
    if (!hold_tap) return;
    // Retire the tap on its own loop; callbacks cannot race a released port.
    CFRunLoopPerformBlock(hold_loop, kCFRunLoopCommonModes, ^{
        CGEventTapEnable(hold_tap, false);
        CFMachPortInvalidate(hold_tap);
        CFRunLoopStop(hold_loop);
    });
    CFRunLoopWakeUp(hold_loop);
    pthread_join(hold_worker, NULL);
    CFRelease(hold_tap);
    hold_tap = NULL;
    CFRelease(hold_loop);
    hold_loop = NULL;
}
