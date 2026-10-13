//go:build darwin

#import <Cocoa/Cocoa.h>
#import <CoreGraphics/CoreGraphics.h>
#include "guide.h"

static NSString *const settings_bundle = @"com.apple.systempreferences";

// arion_find_settings_window reads System Settings' largest on-screen
// document window from the window server, which needs no Accessibility
// trust: the guide runs exactly while that trust is missing. Bounds are
// global points with a top-left origin on the primary display.
void arion_find_settings_window(arion_settings_window *out) {
    memset(out, 0, sizeof(*out));
    @autoreleasepool {
        NSMutableSet<NSNumber *> *pids = [NSMutableSet set];
        for (NSRunningApplication *app in [NSRunningApplication runningApplicationsWithBundleIdentifier:settings_bundle]) {
            [pids addObject:@(app.processIdentifier)];
        }
        if (pids.count == 0) return;
        out->running = 1;
        NSRunningApplication *front = [[NSWorkspace sharedWorkspace] frontmostApplication];
        out->frontmost = [front.bundleIdentifier isEqualToString:settings_bundle] ? 1 : 0;
        CFArrayRef list = CGWindowListCopyWindowInfo(kCGWindowListOptionOnScreenOnly | kCGWindowListExcludeDesktopElements, kCGNullWindowID);
        if (!list) return;
        double best = 0;
        for (NSDictionary *info in (__bridge NSArray *)list) {
            if (![pids containsObject:info[(__bridge NSString *)kCGWindowOwnerPID]]) continue;
            if ([info[(__bridge NSString *)kCGWindowLayer] intValue] != 0) continue;
            NSNumber *alpha = info[(__bridge NSString *)kCGWindowAlpha];
            if (alpha && alpha.doubleValue <= 0) continue;
            CGRect bounds;
            if (!CGRectMakeWithDictionaryRepresentation((__bridge CFDictionaryRef)info[(__bridge NSString *)kCGWindowBounds], &bounds)) continue;
            // Sheets and transient panels are smaller than the settings window.
            if (bounds.size.width < 320 || bounds.size.height < 240) continue;
            double area = bounds.size.width * bounds.size.height;
            if (area <= best) continue;
            best = area;
            out->visible = 1;
            out->x = bounds.origin.x;
            out->y = bounds.origin.y;
            out->width = bounds.size.width;
            out->height = bounds.size.height;
        }
        CFRelease(list);
    }
}

// arion_app_bundle writes the path of the running .app; it returns 0 when
// the process is not inside an application bundle (go run, test binaries).
int arion_app_bundle(char *out, size_t size) {
    out[0] = 0;
    @autoreleasepool {
        NSString *path = [NSBundle mainBundle].bundlePath;
        if (![path.pathExtension.lowercaseString isEqualToString:@"app"]) return 0;
        return strlcpy(out, path.fileSystemRepresentation, size) < size ? 1 : 0;
    }
}
