//go:build darwin

package macos

/*
#cgo CFLAGS: -x objective-c -fobjc-arc
#cgo LDFLAGS: -framework Vision -framework ImageIO -framework CoreGraphics -framework ScreenCaptureKit
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

// ScreenText is text recognized in a window and where its time went.
type ScreenText struct {
	Text      string
	CaptureMs float64
	OCRMs     float64
}

// ScreenRecording reports Screen Recording access, which window text
// recognition needs. macOS does not tell undecided from denied.
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
// reading order. It is the same recognizer RecognizeWindow uses.
func RecognizeImage(data []byte, level OCRLevel) (ScreenText, error) {
	if len(data) == 0 {
		return ScreenText{}, domain.Fail(domain.CodeContractUnsupported, "empty image")
	}
	var out *C.char
	var ms C.double
	status := C.arion_ocr_image(unsafe.Pointer(&data[0]), C.long(len(data)), C.int(level), &out, &ms)
	return screenText(status, out, 0, ms)
}

// RecognizeWindow captures the frontmost window of pid and recognizes its
// text. The image is released inside the call and never stored.
func RecognizeWindow(pid int, level OCRLevel, timeout time.Duration) (ScreenText, error) {
	var out *C.char
	var captureMs, ocrMs C.double
	status := C.arion_ocr_window(C.int(pid), C.int(level), C.double(timeout.Seconds()), &out, &captureMs, &ocrMs)
	return screenText(status, out, captureMs, ocrMs)
}

func screenText(status C.int, out *C.char, captureMs, ocrMs C.double) (ScreenText, error) {
	text := ""
	if out != nil {
		text = C.GoString(out)
		C.free(unsafe.Pointer(out))
	}
	result := ScreenText{Text: text, CaptureMs: float64(captureMs), OCRMs: float64(ocrMs)}
	switch status {
	case C.ARION_CONTEXT_OK:
		return result, nil
	case C.ARION_CONTEXT_NO_PERMISSION:
		return result, domain.Fail(domain.CodePermissionDenied, "screen recording")
	case C.ARION_CONTEXT_UNSUPPORTED:
		return result, domain.Fail(domain.CodeTargetUnsupported, "window capture needs macOS 14")
	case C.ARION_CONTEXT_NO_WINDOW:
		return result, domain.Fail(domain.CodeTargetUnsupported, "no capturable window")
	case C.ARION_CONTEXT_TIMEOUT:
		return result, domain.Fail(domain.CodeTargetUnsupported, "window capture timed out")
	}
	return result, domain.Fail(domain.CodeContractUnsupported, "text recognition failed")
}

// cursorContext reads text around the selection of an AX element.
func cursorContext(element unsafe.Pointer, max int) (before, after string, ok bool) {
	var b, a *C.char
	if C.arion_cursor_context(element, C.long(max), &b, &a) != C.ARION_CONTEXT_OK {
		return "", "", false
	}
	before, after = C.GoString(b), C.GoString(a)
	C.free(unsafe.Pointer(b))
	C.free(unsafe.Pointer(a))
	return before, after, true
}
