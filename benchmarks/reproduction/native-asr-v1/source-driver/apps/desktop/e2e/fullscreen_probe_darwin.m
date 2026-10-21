//go:build arion_e2e && darwin

#import <Cocoa/Cocoa.h>
#import <CoreGraphics/CoreGraphics.h>
#include "fullscreen_probe_darwin.h"

// Read compositor membership and ordering, not just NSWindow.isVisible,
// which can remain true while a window is in a different Space.
void arion_observe_overlay(uintptr_t handle, int fixture_pid, arion_overlay_observation *out) {
    memset(out, 0, sizeof(*out));
    NSWindow *window = (__bridge NSWindow *)(void *)handle;
    out->on_active_space = window.isOnActiveSpace;
    out->behavior = window.collectionBehavior;
    out->frontmost_pid = NSWorkspace.sharedWorkspace.frontmostApplication.processIdentifier;
    CFArrayRef list = CGWindowListCopyWindowInfo(kCGWindowListOptionOnScreenOnly | kCGWindowListExcludeDesktopElements, kCGNullWindowID);
    if (!list) return;
    NSInteger overlayIndex = -1, fixtureIndex = -1;
    CGRect overlayBounds = CGRectZero, fixtureBounds = CGRectZero;
    NSInteger index = 0;
    for (NSDictionary *info in (__bridge NSArray *)list) {
        if ([info[(__bridge NSString *)kCGWindowNumber] integerValue] == window.windowNumber) {
            overlayIndex = index;
            out->on_screen = 1;
            CGRectMakeWithDictionaryRepresentation((__bridge CFDictionaryRef)info[(__bridge NSString *)kCGWindowBounds], &overlayBounds);
        }
        if ([info[(__bridge NSString *)kCGWindowOwnerPID] intValue] == fixture_pid &&
            [info[(__bridge NSString *)kCGWindowLayer] intValue] == 0) {
            fixtureIndex = index;
            CGRectMakeWithDictionaryRepresentation((__bridge CFDictionaryRef)info[(__bridge NSString *)kCGWindowBounds], &fixtureBounds);
        }
        index++;
    }
    out->above_fixture = overlayIndex >= 0 && fixtureIndex >= 0 && overlayIndex < fixtureIndex;
    out->overlaps_fixture = overlayIndex >= 0 && fixtureIndex >= 0 && CGRectIntersectsRect(overlayBounds, fixtureBounds);
    CFRelease(list);
}
