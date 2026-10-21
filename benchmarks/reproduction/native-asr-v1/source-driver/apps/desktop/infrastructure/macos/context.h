#ifndef ARION_CONTEXT_H
#define ARION_CONTEXT_H

enum {
    ARION_CONTEXT_OK = 0,
    ARION_CONTEXT_UNSUPPORTED = 1, // macOS before 14 for window capture
    ARION_CONTEXT_NO_PERMISSION = 2,
    ARION_CONTEXT_NO_WINDOW = 3,
    ARION_CONTEXT_FAILED = 4,
    ARION_CONTEXT_TIMEOUT = 5,
};

// Text recognition levels of Vision.
enum {
    ARION_OCR_ACCURATE = 0,
    ARION_OCR_FAST = 1,
};

int arion_screen_permission(void);
void arion_screen_request(void);

// arion_ocr_image recognizes the text of an encoded image (PNG/JPEG) in
// reading order. *out is malloc'd UTF-8 the caller frees.
int arion_ocr_image(const void *bytes, long length, int level, char **out, double *ms);

// arion_ocr_window captures the frontmost on-screen window of pid with
// ScreenCaptureKit and recognizes its text. The image never leaves this
// function. *out is malloc'd UTF-8 the caller frees.
int arion_ocr_window(int pid, int level, double timeout_seconds, char **out, double *capture_ms, double *ocr_ms);

// Reads visible text from the captured element's window using Accessibility.
// No screenshots or OCR; returns no partial text on a limit or focus change.
int arion_accessibility_window(void *element, int pid, double timeout_seconds, char **out, double *ms);

// arion_cursor_context reads up to max characters before and after the
// focused element's selection. Strings are malloc'd UTF-8 the caller frees.
int arion_cursor_context(void *element, long max, char **before, char **after);

#endif
