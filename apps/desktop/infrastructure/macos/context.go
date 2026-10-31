//go:build darwin

package macos

/*
#cgo CFLAGS: -x objective-c -fobjc-arc
#cgo LDFLAGS: -framework Vision -framework ImageIO -framework CoreGraphics
#include <stdlib.h>
#include "context.h"
*/
import "C"

import (
	"time"
	"unsafe"

	"arion.local/core/domain"
)

// OCRLevel selects Vision's recognition level.
type OCRLevel int

const (
	OCRAccurate OCRLevel = C.ARION_OCR_ACCURATE
	OCRFast     OCRLevel = C.ARION_OCR_FAST
)

// ScreenText carries collected window text and acquisition/recognition timings.
type ScreenText struct {
	Text      string
	CaptureMs float64
	OCRMs     float64
}

// ScreenRecording reports the historical OCR benchmark permission. Product
// window context uses Accessibility. macOS does not tell undecided from denied.
func (Permissions) ScreenRecording() AuthStatus {
	if C.arion_screen_permission() == 1 {
		return AuthGranted
	}
	return AuthDenied
}

// RequestScreenRecording shows the system prompt the first time; later
// requests open the privacy pane, where the user adds Arion by hand.
func (p Permissions) RequestScreenRecording() {
	C.arion_screen_request()
	if p.ScreenRecording() != AuthGranted {
		p.Open(PaneScreenRecording)
	}
}

// PaneScreenRecording is the Screen & System Audio Recording privacy pane.
const PaneScreenRecording Pane = "x-apple.systempreferences:com.apple.preference.security?Privacy_ScreenCapture"

// RecognizeImage recognizes the text of an encoded image with Vision, in
// reading order. Product window context uses ReadWindowText.
func RecognizeImage(data []byte, level OCRLevel) (ScreenText, error) {
	if len(data) == 0 {
		return ScreenText{}, domain.Fail(domain.CodeContractUnsupported, "empty image")
	}
	var out *C.char
	var ms C.double
	status := C.arion_ocr_image(unsafe.Pointer(&data[0]), C.long(len(data)), C.int(level), &out, &ms)
	return screenText(status, out, ms)
}

func screenText(status C.int, out *C.char, ocrMs C.double) (ScreenText, error) {
	text := ""
	if out != nil {
		text = C.GoString(out)
		C.free(unsafe.Pointer(out))
	}
	result := ScreenText{Text: text, OCRMs: float64(ocrMs)}
	if status == C.ARION_CONTEXT_OK {
		return result, nil
	}
	return result, domain.Fail(domain.CodeContractUnsupported, "text recognition failed")
}

// cursorContext reads text around the selection of an AX element.
func cursorContext(element unsafe.Pointer, max int) (before, after string, ok bool) {
	before, after, _, ok = cursorContextSelection(element, max)
	return
}

func cursorContextSelection(element unsafe.Pointer, max int) (before, after, selected string, ok bool) {
	var b, a, s *C.char
	if C.arion_cursor_context(element, C.long(max), &b, &a, &s) != C.ARION_CONTEXT_OK {
		return "", "", "", false
	}
	defer C.free(unsafe.Pointer(b))
	defer C.free(unsafe.Pointer(a))
	defer C.free(unsafe.Pointer(s))
	return C.GoString(b), C.GoString(a), C.GoString(s), true
}

// ReadWindowText reads the captured target window's visible Accessibility text.
// Read duration uses CaptureMs; OCRMs is zero because no recognition runs.
func ReadWindowText(element unsafe.Pointer, pid int, timeout time.Duration) (ScreenText, error) {
	var out *C.char
	var ms C.double
	status := C.arion_accessibility_window(element, C.int(pid), C.double(timeout.Seconds()), &out, &ms)
	text := ""
	if out != nil {
		text = C.GoString(out)
		C.free(unsafe.Pointer(out))
	}
	result := ScreenText{Text: text, CaptureMs: float64(ms)}
	switch status {
	case C.ARION_CONTEXT_OK:
		return result, nil
	case C.ARION_CONTEXT_NO_PERMISSION:
		return result, domain.Fail(domain.CodePermissionDenied, "accessibility")
	case C.ARION_CONTEXT_NO_WINDOW:
		return result, domain.Fail(domain.CodeTargetChanged, "context target changed")
	case C.ARION_CONTEXT_TIMEOUT:
		return result, domain.Fail(domain.CodeTargetUnsupported, "accessibility read timed out")
	}
	return result, domain.Fail(domain.CodeTargetUnsupported, "visible accessibility text unavailable or exceeds bounds")
}
