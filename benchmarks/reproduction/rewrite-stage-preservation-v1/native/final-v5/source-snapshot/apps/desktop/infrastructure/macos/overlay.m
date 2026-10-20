//go:build darwin

#import <Cocoa/Cocoa.h>
#include "overlay.h"

void arion_configure_overlay(uintptr_t handle) {
    NSWindow *window = (__bridge NSWindow *)(void *)handle;
    // MyGo's DisableFullScreen sets FullScreenNone. It is mutually exclusive
    // with FullScreenAuxiliary; joining ordinary Spaces alone is insufficient.
    NSWindowCollectionBehavior behavior = window.collectionBehavior;
    behavior &= ~(NSWindowCollectionBehaviorFullScreenPrimary |
                  NSWindowCollectionBehaviorFullScreenNone |
                  NSWindowCollectionBehaviorMoveToActiveSpace |
                  NSWindowCollectionBehaviorManaged |
                  NSWindowCollectionBehaviorStationary |
                  NSWindowCollectionBehaviorParticipatesInCycle);
    behavior |= NSWindowCollectionBehaviorCanJoinAllSpaces |
                NSWindowCollectionBehaviorFullScreenAuxiliary |
                NSWindowCollectionBehaviorTransient |
                NSWindowCollectionBehaviorIgnoresCycle;
    if (@available(macOS 13.0, *)) {
        behavior &= ~(NSWindowCollectionBehaviorPrimary | NSWindowCollectionBehaviorAuxiliary);
        behavior |= NSWindowCollectionBehaviorCanJoinAllApplications;
    }
    window.collectionBehavior = behavior;
}
