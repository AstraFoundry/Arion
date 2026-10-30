#ifndef ARION_FOCUS_H
#define ARION_FOCUS_H
#import <ApplicationServices/ApplicationServices.h>

// NoValue is a supported focus attribute with no current value. Unsupported,
// denied and unresponsive providers do not establish absence of input.
static AXUIElementRef arion_copy_focus(AXUIElementRef scope, int secure, int *no_input, int *error) {
    CFTypeRef element = NULL;
    AXError result = AXUIElementCopyAttributeValue(scope, kAXFocusedUIElementAttribute, &element);
    if (error) *error = result;
    if (no_input) *no_input = !secure && !element && result == kAXErrorNoValue;
    if (result != kAXErrorSuccess) {
        if (element) CFRelease(element);
        return NULL;
    }
    return (AXUIElementRef)element;
}
#endif
