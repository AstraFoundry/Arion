//go:build darwin

#import <ApplicationServices/ApplicationServices.h>
#import <Cocoa/Cocoa.h>
#import <ImageIO/ImageIO.h>
#import <ScreenCaptureKit/ScreenCaptureKit.h>
#import <Vision/Vision.h>
#include <stdatomic.h>
#include "context.h"

static double millis_since(CFAbsoluteTime start) { return (CFAbsoluteTimeGetCurrent() - start) * 1000.0; }

static char *copy_utf8(NSString *s) {
    const char *u = s.UTF8String ?: "";
    size_t n = strlen(u) + 1;
    char *out = malloc(n);
    memcpy(out, u, n);
    return out;
}

int arion_screen_permission(void) { return CGPreflightScreenCaptureAccess() ? 1 : 0; }

void arion_screen_request(void) { CGRequestScreenCaptureAccess(); }

// recognize returns the image's text in reading order: observations are
// grouped into lines by vertical overlap, lines top to bottom, words left
// to right.
static int recognize(CGImageRef image, int level, char **out, double *ms) {
    CFAbsoluteTime start = CFAbsoluteTimeGetCurrent();
    VNRecognizeTextRequest *request = [[VNRecognizeTextRequest alloc] init];
    request.recognitionLevel = level == ARION_OCR_FAST ? VNRequestTextRecognitionLevelFast : VNRequestTextRecognitionLevelAccurate;
    request.usesLanguageCorrection = level != ARION_OCR_FAST;
    NSArray<NSString *> *supported = [request supportedRecognitionLanguagesAndReturnError:nil] ?: @[];
    NSMutableArray<NSString *> *languages = [NSMutableArray array];
    for (NSString *wanted in @[@"zh-Hans", @"en-US"]) {
        if ([supported containsObject:wanted]) [languages addObject:wanted];
    }
    if (languages.count) request.recognitionLanguages = languages;
    VNImageRequestHandler *handler = [[VNImageRequestHandler alloc] initWithCGImage:image options:@{}];
    NSError *error = nil;
    if (![handler performRequests:@[request] error:&error]) {
        *ms = millis_since(start);
        return ARION_CONTEXT_FAILED;
    }
    NSArray<VNRecognizedTextObservation *> *observations = [request.results sortedArrayUsingComparator:^NSComparisonResult(VNRecognizedTextObservation *a, VNRecognizedTextObservation *b) {
        CGFloat ay = CGRectGetMidY(a.boundingBox), by = CGRectGetMidY(b.boundingBox);
        if (ay != by) return ay > by ? NSOrderedAscending : NSOrderedDescending;
        return CGRectGetMinX(a.boundingBox) < CGRectGetMinX(b.boundingBox) ? NSOrderedAscending : NSOrderedDescending;
    }];
    NSMutableArray<NSString *> *lines = [NSMutableArray array];
    NSMutableArray<VNRecognizedTextObservation *> *line = [NSMutableArray array];
    CGFloat lineMid = 0, lineHeight = 0;
    void (^flush)(void) = ^{
        if (!line.count) return;
        [line sortUsingComparator:^NSComparisonResult(VNRecognizedTextObservation *a, VNRecognizedTextObservation *b) {
            return CGRectGetMinX(a.boundingBox) < CGRectGetMinX(b.boundingBox) ? NSOrderedAscending : NSOrderedDescending;
        }];
        NSMutableArray<NSString *> *words = [NSMutableArray array];
        for (VNRecognizedTextObservation *o in line) {
            NSString *text = [o topCandidates:1].firstObject.string;
            if (text.length) [words addObject:text];
        }
        if (words.count) [lines addObject:[words componentsJoinedByString:@" "]];
        [line removeAllObjects];
    };
    for (VNRecognizedTextObservation *o in observations) {
        CGFloat mid = CGRectGetMidY(o.boundingBox), height = CGRectGetHeight(o.boundingBox);
        if (line.count && fabs(mid - lineMid) > MAX(height, lineHeight) / 2) flush();
        if (!line.count) {
            lineMid = mid;
            lineHeight = height;
        }
        [line addObject:o];
    }
    flush();
    *out = copy_utf8([lines componentsJoinedByString:@"\n"]);
    *ms = millis_since(start);
    return ARION_CONTEXT_OK;
}

int arion_ocr_image(const void *bytes, long length, int level, char **out, double *ms) {
    *out = NULL;
    *ms = 0;
    @autoreleasepool {
        CFDataRef data = CFDataCreate(NULL, bytes, length);
        CGImageSourceRef source = CGImageSourceCreateWithData(data, NULL);
        CFRelease(data);
        if (!source) return ARION_CONTEXT_FAILED;
        CGImageRef image = CGImageSourceCreateImageAtIndex(source, 0, NULL);
        CFRelease(source);
        if (!image) return ARION_CONTEXT_FAILED;
        int status = recognize(image, level, out, ms);
        CGImageRelease(image);
        return status;
    }
}

// frontmost_window_id is the first on-screen, normal-layer window of pid in
// the window server's front-to-back order.
static CGWindowID frontmost_window_id(int pid) {
    CGWindowID found = kCGNullWindowID;
    CFArrayRef windows = CGWindowListCopyWindowInfo(kCGWindowListOptionOnScreenOnly | kCGWindowListExcludeDesktopElements, kCGNullWindowID);
    if (!windows) return found;
    for (NSDictionary *w in (__bridge NSArray *)windows) {
        if ([w[(id)kCGWindowOwnerPID] intValue] == pid && [w[(id)kCGWindowLayer] intValue] == 0) {
            found = [w[(id)kCGWindowNumber] unsignedIntValue];
            break;
        }
    }
    CFRelease(windows);
    return found;
}

// A capture outlives its caller after a timeout; the completion then
// releases the image itself.
@interface ArionCapture : NSObject
@property(nonatomic) dispatch_semaphore_t done;
@property(nonatomic) CGImageRef image;
@property(nonatomic) int status;
@property(atomic) BOOL abandoned;
@end
@implementation ArionCapture
@end

int arion_ocr_window(int pid, int level, double timeout_seconds, char **out, double *capture_ms, double *ocr_ms) {
    *out = NULL;
    *capture_ms = *ocr_ms = 0;
    if (@available(macOS 14.0, *)) {
        if (!NSClassFromString(@"SCScreenshotManager")) return ARION_CONTEXT_UNSUPPORTED;
        if (!CGPreflightScreenCaptureAccess()) return ARION_CONTEXT_NO_PERMISSION;
        CGWindowID windowID = frontmost_window_id(pid);
        if (windowID == kCGNullWindowID) return ARION_CONTEXT_NO_WINDOW;
        CFAbsoluteTime start = CFAbsoluteTimeGetCurrent();
        ArionCapture *capture = [ArionCapture new];
        capture.done = dispatch_semaphore_create(0);
        capture.status = ARION_CONTEXT_FAILED;
        [SCShareableContent getShareableContentExcludingDesktopWindows:YES onScreenWindowsOnly:YES completionHandler:^(SCShareableContent *content, NSError *error) {
            SCWindow *target = nil;
            for (SCWindow *w in content.windows) {
                if (w.windowID == windowID) target = w;
            }
            if (!target) {
                capture.status = ARION_CONTEXT_NO_WINDOW;
                dispatch_semaphore_signal(capture.done);
                return;
            }
            SCContentFilter *filter = [[SCContentFilter alloc] initWithDesktopIndependentWindow:target];
            SCStreamConfiguration *config = [[SCStreamConfiguration alloc] init];
            config.width = (size_t)(target.frame.size.width * filter.pointPixelScale);
            config.height = (size_t)(target.frame.size.height * filter.pointPixelScale);
            config.showsCursor = NO;
            // Looked up at run time: a class reference would keep the app
            // from launching on macOS 13, where the class does not exist.
            id manager = NSClassFromString(@"SCScreenshotManager");
            [manager captureImageWithFilter:filter configuration:config completionHandler:^(CGImageRef image, NSError *captureError) {
                if (image && !capture.abandoned) {
                    capture.image = CGImageRetain(image);
                    capture.status = ARION_CONTEXT_OK;
                }
                dispatch_semaphore_signal(capture.done);
            }];
        }];
        if (dispatch_semaphore_wait(capture.done, dispatch_time(DISPATCH_TIME_NOW, (int64_t)(timeout_seconds * NSEC_PER_SEC))) != 0) {
            capture.abandoned = YES;
            return ARION_CONTEXT_TIMEOUT;
        }
        *capture_ms = millis_since(start);
        if (capture.status != ARION_CONTEXT_OK) return capture.status;
        int status;
        @autoreleasepool {
            status = recognize(capture.image, level, out, ocr_ms);
        }
        CGImageRelease(capture.image);
        capture.image = NULL;
        return status;
    }
    return ARION_CONTEXT_UNSUPPORTED;
}

static NSString *string_for_range(AXUIElementRef element, CFRange range, NSString *value) {
    if (range.length <= 0) return @"";
    AXValueRef param = AXValueCreate(kAXValueCFRangeType, &range);
    CFTypeRef result = NULL;
    AXError error = AXUIElementCopyParameterizedAttributeValue(element, kAXStringForRangeParameterizedAttribute, param, &result);
    CFRelease(param);
    if (error == kAXErrorSuccess && result && CFGetTypeID(result) == CFStringGetTypeID()) {
        return CFBridgingRelease(result);
    }
    if (result) CFRelease(result);
    if (value && range.location + range.length <= (CFIndex)value.length) {
        return [value substringWithRange:NSMakeRange(range.location, range.length)];
    }
    return @"";
}

int arion_cursor_context(void *element, long max, char **before, char **after) {
    *before = *after = NULL;
    if (!element) return ARION_CONTEXT_FAILED;
    @autoreleasepool {
        AXUIElementRef el = (AXUIElementRef)element;
        AXUIElementSetMessagingTimeout(el, 0.3f);
        CFTypeRef rangeValue = NULL;
        CFRange selection = {0, 0};
        if (AXUIElementCopyAttributeValue(el, kAXSelectedTextRangeAttribute, &rangeValue) != kAXErrorSuccess || !rangeValue) {
            return ARION_CONTEXT_FAILED;
        }
        BOOL ok = CFGetTypeID(rangeValue) == AXValueGetTypeID() && AXValueGetValue(rangeValue, kAXValueCFRangeType, &selection);
        CFRelease(rangeValue);
        if (!ok) return ARION_CONTEXT_FAILED;
        NSString *value = nil;
        CFTypeRef raw = NULL;
        if (AXUIElementCopyAttributeValue(el, kAXValueAttribute, &raw) == kAXErrorSuccess && raw) {
            if (CFGetTypeID(raw) == CFStringGetTypeID()) value = CFBridgingRelease(raw);
            else CFRelease(raw);
        }
        CFIndex total = value ? (CFIndex)value.length : 0;
        CFTypeRef count = NULL;
        if (AXUIElementCopyAttributeValue(el, kAXNumberOfCharactersAttribute, &count) == kAXErrorSuccess && count) {
            if (CFGetTypeID(count) == CFNumberGetTypeID()) CFNumberGetValue(count, kCFNumberCFIndexType, &total);
            CFRelease(count);
        }
        // AX ranges count UTF-16 units; twice max keeps max characters of
        // any script after the Go side trims by character.
        CFIndex span = (CFIndex)max * 2;
        CFIndex from = MAX(0, selection.location - span);
        CFIndex end = selection.location + selection.length;
        *before = copy_utf8(string_for_range(el, CFRangeMake(from, selection.location - from), value));
        *after = copy_utf8(string_for_range(el, CFRangeMake(end, MAX(0, MIN(span, total - end))), value));
        return ARION_CONTEXT_OK;
    }
}
