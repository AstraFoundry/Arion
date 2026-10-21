// Exercise production tap ownership with a real CFRunLoop and a controlled
// Core Graphics permission/tap boundary. No system permission is requested.
#import <Cocoa/Cocoa.h>
#import <ApplicationServices/ApplicationServices.h>
#include <assert.h>
static bool consent, enabled, disableOnCreate, failCreate;
static int creates, callbacks;
static bool testAccess(void) { return consent; }
static void portCallback(CFMachPortRef port, void *message, CFIndex size, void *info) {
    (void)port; (void)message; (void)size; (void)info;
}
static CFMachPortRef testCreate(CGEventTapLocation location, CGEventTapPlacement placement,
    CGEventTapOptions options, CGEventMask mask, CGEventTapCallBack callback, void *info) {
    (void)callback; (void)info;
    assert(location == kCGSessionEventTap && placement == kCGHeadInsertEventTap && options == kCGEventTapOptionListenOnly);
    assert(mask == (CGEventMaskBit(kCGEventKeyDown) | CGEventMaskBit(kCGEventFlagsChanged)));
    creates++;
    enabled = !disableOnCreate;
    return failCreate ? NULL : CFMachPortCreate(NULL, portCallback, NULL, NULL);
}
static bool testEnabled(CFMachPortRef port) { (void)port; return enabled; }
static void testEnable(CFMachPortRef port, bool value) { (void)port; enabled = value; }
void arionHoldRaw(int type, long long keycode, unsigned long long flags, int repeat) {
    assert(type == kCGEventFlagsChanged && keycode == 61 && flags == 0x80040 && !repeat);
    callbacks++;
}
#define CGPreflightListenEventAccess testAccess
#define CGEventTapCreate testCreate
#define CGEventTapIsEnabled testEnabled
#define CGEventTapEnable testEnable
#include "../hold.m"
int main(void) { @autoreleasepool {
    assert(arion_hold_start() == ARION_HOLD_PERMISSION_DENIED && creates == 0 && !hold_tap);
    consent = true; failCreate = true;
    assert(arion_hold_start() == ARION_HOLD_UNAVAILABLE && !hold_tap);
    failCreate = false; disableOnCreate = true;
    assert(arion_hold_start() == ARION_HOLD_UNAVAILABLE && !hold_tap);
    disableOnCreate = false;
    for (int i = 0; i < 100; i++) {
        assert(arion_hold_start() == 0 && hold_tap && hold_loop && enabled);
        int count = creates;
        assert(arion_hold_start() == 0 && creates == count);
        arion_hold_stop();
        assert(!hold_tap && !hold_loop);
        arion_hold_stop();
    }
    assert(arion_hold_start() == 0);
    dispatch_semaphore_t delivered = dispatch_semaphore_create(0);
    CFRunLoopPerformBlock(hold_loop, kCFRunLoopCommonModes, ^{
        CGEventRef event = CGEventCreateKeyboardEvent(NULL, 61, true);
        CGEventSetFlags(event, 0x80040);
        hold_callback(NULL, kCGEventFlagsChanged, event, NULL);
        CFRelease(event);
        dispatch_semaphore_signal(delivered);
    });
    CFRunLoopWakeUp(hold_loop);
    assert(dispatch_semaphore_wait(delivered, dispatch_time(DISPATCH_TIME_NOW, NSEC_PER_SEC)) == 0 && callbacks == 1);
    enabled = false;
    hold_callback(NULL, kCGEventTapDisabledByTimeout, NULL, NULL);
    assert(enabled);
    consent = false; enabled = false;
    hold_callback(NULL, kCGEventTapDisabledByUserInput, NULL, NULL);
    assert(!enabled && arion_hold_start() != 0);
    arion_hold_stop();
    consent = true;
    assert(arion_hold_start() == 0);
    arion_hold_stop();
    puts("Native hold listener: denied/disabled/create failure, 100 restarts, callback delivery, timeout and consent recovery passed");
} }
