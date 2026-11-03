#import <ApplicationServices/ApplicationServices.h>
#import <Carbon/Carbon.h>
#import <Cocoa/Cocoa.h>

static int trusted = 1, secure = 0, owner = 42;
static CFStringRef focused = CFSTR("owned field A");
static CFStringRef window = CFSTR("owned window A");
static AXError focusStatus = kAXErrorSuccess, windowStatus = kAXErrorSuccess;
static int fieldSecure;
static int posts;

static Boolean test_trusted(void) { return trusted; }
static AXUIElementRef test_app(pid_t pid) { return (AXUIElementRef)CFRetain(CFSTR("owned app")); }
static AXError test_timeout(AXUIElementRef element, float value) { return kAXErrorSuccess; }
static AXError test_attribute(AXUIElementRef element, CFStringRef key, CFTypeRef *out) {
    *out = NULL;
    if (CFEqual(key, kAXFocusedWindowAttribute)) {
        if (window) *out = CFRetain(window);
        return windowStatus;
    }
    if (CFEqual(key, kAXFocusedUIElementAttribute)) {
        if (focused) *out = CFRetain(focused);
        return focusStatus;
    }
    if (CFEqual(key, kAXSubroleAttribute)) {
        if (fieldSecure) *out = CFRetain(kAXSecureTextFieldSubrole);
        return kAXErrorSuccess;
    }
    abort();
}
int arion_secure_input(void) { return secure; }
int arion_frontmost_pid(void) { return owner; }
int arion_focus_matches(void *element, int pid) { abort(); }
int arion_selection_matches(void *element, void *selection) { abort(); }
void arion_enable_web_accessibility(int pid) {}

// Only owned metadata is supplied. No AX client queries an external app,
// and event delivery is intercepted before reaching the system.
static void test_post(CGEventTapLocation location, CGEventRef event) {
    if (CGEventGetIntegerValueField(event, kCGKeyboardEventKeycode) != kVK_ANSI_V ||
        CGEventGetFlags(event) != kCGEventFlagMaskCommand ||
        CGEventGetType(event) != (posts % 2 ? kCGEventKeyUp : kCGEventKeyDown)) abort();
    posts++;
}
#define AXIsProcessTrusted test_trusted
#define AXUIElementCreateApplication test_app
#define AXUIElementSetMessagingTimeout test_timeout
#define AXUIElementCopyAttributeValue test_attribute
#define AXUIElementGetTypeID CFStringGetTypeID
#define CGEventPost test_post
#include "../paste.m"
#undef CGEventPost
#define CHECK(x) do { if (!(x)) { fprintf(stderr, "failed at line %d: %s\n", __LINE__, #x); return 1; } } while (0)

int main(void) {
    @autoreleasepool {
        void *capturedWindow = arion_history_window(42, NULL);
        int capturedSecure, noInput;
        void *field = arion_history_focus(42, &capturedSecure, &noInput, NULL);
        CHECK(field && capturedWindow && !capturedSecure && !noInput);
        // No geometry API is provided: an exact field/window still validates.
        CHECK(arion_history_check(42, field, capturedWindow));
        CHECK(arion_history_status(42, field, capturedWindow) == ARION_HISTORY_READY);
        int queryError;
        NSPasteboard *board = [NSPasteboard pasteboardWithUniqueName];
        [board clearContents];
        CHECK(post_history(42, field, capturedWindow, board.changeCount, board) == ARION_PASTE_OK);
        CHECK(posts == 2);
        CHECK(post_history(42, field, capturedWindow, board.changeCount + 1, board) == ARION_PASTE_CHANGED);
        CHECK(posts == 2);
        focused = CFSTR("owned field B");
        CHECK(arion_history_status(42, field, capturedWindow) == ARION_HISTORY_FOCUS_CHANGED);
        CHECK(!arion_history_check(42, field, capturedWindow));
        CHECK(post_history(42, field, capturedWindow, board.changeCount, board) == ARION_PASTE_TARGET_CHANGED);
        CHECK(posts == 2);
        focused = CFSTR("owned field A");
        window = CFSTR("owned window B");
        CHECK(arion_history_status(42, field, capturedWindow) == ARION_HISTORY_WINDOW_CHANGED);
        CHECK(!arion_history_check(42, field, capturedWindow));
        window = CFSTR("owned window A");
        owner = 43;
        CHECK(arion_history_status(42, field, capturedWindow) == ARION_HISTORY_OWNER_CHANGED);
        CHECK(!arion_history_check(42, field, capturedWindow));
        owner = 42;
        trusted = 0;
        CHECK(arion_history_status(42, field, capturedWindow) == ARION_HISTORY_PERMISSION_DENIED);
        CHECK(!arion_history_check(42, field, capturedWindow));
        trusted = 1;
        secure = 1;
        CHECK(arion_history_status(42, field, capturedWindow) == ARION_HISTORY_SECURE_INPUT);
        CHECK(!arion_history_check(42, field, capturedWindow));
        secure = 0;
        fieldSecure = 1;
        CHECK(!arion_history_check(42, field, capturedWindow));
        fieldSecure = 0;

        // Missing AX focus may use the exact captured window, never just a PID.
        focused = NULL;
        AXError uncertain[] = {kAXErrorAttributeUnsupported, kAXErrorCannotComplete,
            kAXErrorAPIDisabled, kAXErrorFailure, kAXErrorSuccess, kAXErrorNoValue};
        for (int index = 0; index < sizeof(uncertain) / sizeof(uncertain[0]); index++) {
            focusStatus = uncertain[index];
            CHECK(!arion_history_check(42, field, capturedWindow));
            int allowed = uncertain[index] == kAXErrorNoValue || uncertain[index] == kAXErrorAttributeUnsupported;
            CHECK(arion_history_window_fallback(uncertain[index]) == allowed);
            CHECK(arion_history_check(42, NULL, capturedWindow) == allowed);
            CHECK(!arion_history_check(42, NULL, NULL));
        }
        CHECK(arion_history_status(42, NULL, NULL) == ARION_HISTORY_MISSING_FOCUS);
        CHECK(arion_history_status(42, field, capturedWindow) == ARION_HISTORY_FOCUS_UNAVAILABLE);
        focusStatus = kAXErrorAttributeUnsupported;
        void *missing = arion_history_focus(42, &capturedSecure, NULL, &queryError);
        CHECK(!missing && queryError == kAXErrorAttributeUnsupported);
        CHECK(post_history(42, NULL, capturedWindow, board.changeCount, board) == ARION_PASTE_OK);
        CHECK(posts == 4);
        focusStatus = kAXErrorNoValue;
        CHECK(arion_history_check(42, NULL, capturedWindow));
        CHECK(post_history(42, NULL, capturedWindow, board.changeCount, board) == ARION_PASTE_OK);
        CHECK(posts == 6);
        CHECK(post_history(42, NULL, capturedWindow, board.changeCount + 1, board) == ARION_PASTE_CHANGED);
        window = CFSTR("owned window B");
        CHECK(post_history(42, NULL, capturedWindow, board.changeCount, board) == ARION_PASTE_TARGET_CHANGED);
        window = CFSTR("owned window A");
        owner = 43;
        CHECK(!arion_history_check(42, NULL, capturedWindow));
        owner = 42;
        secure = 1;
        CHECK(!arion_history_check(42, NULL, capturedWindow));
        secure = 0;
        trusted = 0;
        CHECK(!arion_history_check(42, NULL, capturedWindow));
        trusted = 1;
        focused = CFSTR("new field exposed in original window"); focusStatus = kAXErrorSuccess;
        fieldSecure = 1;
        CHECK(!arion_history_check(42, NULL, capturedWindow));
        fieldSecure = 0;
        CHECK(arion_history_check(42, NULL, capturedWindow));
        CHECK(posts == 6);
        focused = CFSTR("owned field A"); focusStatus = kAXErrorSuccess;
        windowStatus = kAXErrorCannotComplete;
        CHECK(arion_history_status(42, field, capturedWindow) == ARION_HISTORY_WINDOW_UNAVAILABLE);
        CHECK(!arion_history_check(42, field, capturedWindow));
        windowStatus = kAXErrorSuccess;
        CHECK(arion_history_check(42, field, capturedWindow));
        CFRelease(field);
        [board releaseGlobally];
        CFRelease(capturedWindow);
        puts("clipboard history target identity checks passed");
    }
    return 0;
}
