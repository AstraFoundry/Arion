//go:build darwin

#import <ApplicationServices/ApplicationServices.h>
#import <Cocoa/Cocoa.h>
#import <ImageIO/ImageIO.h>
#import <Vision/Vision.h>
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
