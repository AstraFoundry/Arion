#import <ApplicationServices/ApplicationServices.h>
#import <Foundation/Foundation.h>

static AXError status;
static int present;
static AXError test_attribute(AXUIElementRef scope, CFStringRef attribute, CFTypeRef *out) {
    if (!CFEqual(attribute, kAXFocusedUIElementAttribute)) abort();
    *out = present ? CFRetain(CFSTR("owned focus fixture")) : NULL;
    return status;
}
#define AXUIElementCopyAttributeValue test_attribute
#include "../focus.h"
#undef AXUIElementCopyAttributeValue
#define CHECK(x) do { if (!(x)) { fprintf(stderr, "failed at line %d: %s\n", __LINE__, #x); return 1; } } while (0)

int main(void) {
    int absent, error;
    status = kAXErrorNoValue;
    CHECK(arion_copy_focus(NULL, 0, &absent, &error) == NULL);
    CHECK(absent == 1 && error == kAXErrorNoValue);
    CHECK(arion_copy_focus(NULL, 1, &absent, NULL) == NULL && absent == 0);
    AXError uncertain[] = { kAXErrorFailure, kAXErrorCannotComplete, kAXErrorAttributeUnsupported, kAXErrorAPIDisabled, kAXErrorSuccess };
    for (int i = 0; i < sizeof(uncertain)/sizeof(uncertain[0]); i++) {
        status = uncertain[i];
        CHECK(arion_copy_focus(NULL, 0, &absent, NULL) == NULL && absent == 0);
    }
    status = kAXErrorSuccess; present = 1;
    AXUIElementRef focus = arion_copy_focus(NULL, 0, &absent, NULL);
    CHECK(focus != NULL && absent == 0);
    CFRelease(focus);
    status = kAXErrorCannotComplete;
    CHECK(arion_copy_focus(NULL, 0, &absent, NULL) == NULL && absent == 0);
    return 0;
}
